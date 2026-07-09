# Architecture

How hollow-grid-go is put together, and why. The north star is the upstream
`docs/protocol.md` (the language-agnostic wire contract) and `smoke.mjs` (the
executable conformance suite). Nothing here invents protocol; it implements it.

## The shape

```
                 player (wscat / bot.mjs / smoke.mjs)
                          |  WebSocket /ws  (UTF-8 text, CRLF lines)
                          v
  cmd/world  -->  internal/transport (Server)
                    |  per connection:
                    v
                  session  --- a SINGLE goroutine select-loop ---
                    |        { player command | heartbeat tick | disconnect }
                    |
        +-----------+------------------------------+
        v                      v                   v
  internal/world         internal/store      internal/event
  (rooms, races,         (CharStore:          (@event <name> <json>
   mobs, items,           FileStore now,       framing)
   the living clock)      Grid later)
```

There is no shared mutable game state behind a lock on the session loop. Each
connection owns its own session goroutine; the world's content (rooms, the room
graph, the bestiary) is read-mostly. Shared **presence** (who is in which room,
tell/yell/emote routing) lives in `internal/transport/hub.go`, a session registry
guarded by a mutex; mob death and rescue state are world-wide on the `Server`.

## The session select-loop (the keystone)

A MUD has two clocks: the player's commands, and the world's heartbeat (combat
rounds, regen, the turning day). Handling both without locks is the central
design choice.

`internal/transport/conn.go` runs **one goroutine per connection**:

```go
cmds := make(chan string)      // fed by a small reader goroutine
heartbeat := time.NewTicker(2s)
for {
    select {
    case cmd := <-cmds:        // a player command
    case <-heartbeat.C:        // the world beat: clock, combat, regen
    case <-readerDone:         // disconnect
    }
}
```

Because every mutation of `session`/`Player` state happens inside this one loop --
commands and ticks are serialized by the `select` -- **no mutexes are needed.** A
combat fight is just the heartbeat finding `player.Target != nil` and resolving a
round; `rest` is the heartbeat finding `position == "resting"` and regenerating.
The reader goroutine only reads and forwards on a channel; a `stop` channel and
the request context tear it down cleanly on disconnect.

This is why combat, regen, and the day/night clock all "just work" on the same
beat, and why adding the next tick-driven system is cheap.

## The `@event` channel

`internal/event` is tiny on purpose: `Line(name, payload)` marshals a payload to
`@event <name> <json>`. The transport interleaves these with prose and flushes a
whole command's response (prose + events) as one WebSocket message.

The contract (protocol.md s2): the prose is for humans, the `@event` channel is
the machine-readable truth, and the two must never drift. Every player-affecting
change emits its event: `room.info`, `char.vitals`, `char.affects`,
`char.equipment`, `room.actions`, `combat.*`, `world.state`, `char.dream`,
`grid.rescued`, … The payload structs live in `internal/world` next to the model
that produces them, with exact JSON field names.

### Moral choice as data

The most important property: moral choices are first-class, labelled actions, not
prose to be parsed. `room.actions` carries each contextual action with a `kind`
(`move`/`moral`/…) and, for moral ones, a `valence` (`virtuous`/`corrupt`/
`grave`). A *hunted* race who is offered `join` at the Cinder Front recruiter sees
that action flagged `valence:"grave"` -- the gravest betrayal -- computed per-player
from race stance. An agent reads the ethics; it does not infer them.

## The world model (`internal/world`)

- **Rooms** form a graph (`Exits: direction -> room id`); an exit not listed does
  not exist (the no-silent-no-op rule). Rooms may carry contextual `Actions`
  (moral choices), live `Mobs`, and a `Captive`.
- **Races** (`races.go`) are opaque federated labels with light mechanical leans
  (hp/damage/armor/regen) and a Cinder Front **stance** (`accepted`/`tolerated`/
  `hunted`) -- the stance is the heart of the moral system: where you stand before
  you have done anything.
- **Mobs** (`mobs.go`) are template + current HP instances spawned into rooms,
  with loot tables, XP awards, and optional poison chance.
- **Items** (`items.go`) carry slots, combat leans, sell `value`, and optional
  `use` effects; the player has an inventory and equipment map. Ground piles are
  world-local on the transport `Server` (`ground.go`) and appear in `room.info.items`.
- **The living world** is a *pure function of elapsed time*: `World.State()`
  computes `tick`/`phase`/`weather` from how long the world has been up, so every
  observer agrees and there is no shared clock to race. The heartbeat just emits
  it, so the tick advances on its own. Combat (including stolen-kill displacement),
  regen, and poison ticks share that heartbeat.

## Persistence and the federation seam (`internal/store`)

`CharStore` is the single seam between standalone and federated operation:

```go
type CharStore interface {
    Load(name string) (world.CharSheet, bool, error)
    Commit(name string, sheet world.CharSheet) error
}
```

- `FileStore` (today) persists the canonical `CharSheet` as one JSON file per
  character -- dependency-free, human-inspectable, the documented offline fallback.
- The federation client (later) implements the *same interface* against the Grid
  (`loadCharacter`/`commitCharacter`). Per the trust model, only the canonical
  `CharSheet` (level/xp/gold/faction/morality/title/race/ashsworn) ever
  round-trips; inventory, HP, room, and position are world-local and never shared.

Identity is name-based (protocol.md s1): a known name resumes its sheet and skips
the race menu; a new name chooses a race once. Persistence is best-effort --
a store failure is logged but never blocks play.

## Federation (`internal/grid`)

The Grid Hub backend remains the upstream's other half (`the-hollow-grid/grid-hub/`).
This port implements the **world-side client**:

- **`LocalHub`** (default when `GRID_HUB_URL` is unset): in-process fallback so
  standalone play and smoke's single-world phases work without Cloudflare.
- **`RemoteHub`** (when `GRID_HUB_URL` + `GRID_HUB_TOKEN` are set): HTTP JSON-RPC
  to `POST /rpc` on the hub (production: `https://grid-hub.skyphusion.org/rpc`).

`RunFederation()` heartbeats presence, polls gridcast, and syncs tide on a 2s loop.
Every hub call is best-effort; failures are logged and play continues on local state.

The `CharStore` seam (`internal/store`) still holds the offline fallback; when the
hub is bound, canonical `CharSheet` fields round-trip through `loadCharacter` /
`commitCharacter` on connect and checkpoint.

## What is deliberately NOT here

- **The Grid Hub server itself.** Tide, ledger, registry, and cross-world rolls
  are authoritative in the upstream hub Worker. This repo is a world node only.
- **Full trust hardening.** Per-world keys, leased progression deltas, and
  server-side validation of commits are design targets in `the-hollow-grid/docs/federation.md`
  but not enforced yet (fine while one operator runs all worlds).

## Testing & conformance

- `internal/transport/*_test.go` drives **real WebSocket sessions** against an
  `httptest` server (login, movement, combat to a kill, the moral arc, rescue,
  the economy, the living world). Tests drain in-flight sessions before temp-dir
  cleanup so a disconnect-time persist never races.
- The real bar is the upstream `smoke.mjs` (**135 checks**): point it at a running
  server or the container and it asserts the exact `@event` truth. Set
  `DUSTFALL_URL` too for the federation phase (travel, cross-world identity).
  Prod Rust Choir baseline (2026-07-09): **158 ok / 0 fail / 1 skip** with hub +
  Dustfall live.
