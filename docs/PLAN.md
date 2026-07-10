# Build plan & status

Porting the Hollow Grid world framework to Go, against the upstream
`docs/protocol.md`. The scoreboard is the upstream `smoke.mjs` (**~153 checks**
on a standalone run; grows with federation blocks when `DUSTFALL_URL` is up):
**build the port to pass it, phase by phase.**

**Status (2026-07-10, re-baseline #40):** upstream `smoke.mjs` now carries **~153
executable checks** on standalone (1 federation block SKIPs when `DUSTFALL_URL` is down).
**Local standalone** (`go run ./cmd/world`, LocalHub, no `GRID_HUB_URL`):
**149 ok / 3 fail / 1 skip** (rancid, the-hollow-grid @ main). The three fails are
hub-bound on LocalHub (`gridcast`, global tide delta, `gridstats` ledger composition);
expected until #42 federation client work binds a real hub. **Live Rust Choir** (Mackaye
reference, bots offline, `DUSTFALL_URL` set): **135 ok / 19 fail / 0 skip** -- login
burst + several hub-write features still red; see #40 comment for failure clusters.
Prior quiet-prod note (2026-07-09): **156 ok / 1 fail / 0 skip** under different
suite revision / load conditions.

## Done

**Phase 0 -- transport foundation**
- [x] HTTP server, graceful shutdown, `/health` + `/health/deep`
- [x] `/ws` WebSocket, UTF-8 text, CRLF lines
- [x] login flow: banner → name → race menu → play; name-based identity
- [x] the `@event` channel framing

**Phase 1 -- the world**
- [x] the 7 canonical races, with Cinder Front stance + signature abilities
- [x] the canonical opening map + the wastes (Ash Flats, Scorch Road, Refugee Waystation)
- [x] items / inventory / equipment, the starter shiv, `title`
- [x] mobs, `consider`, `look <mob>`
- [x] **async combat** on a tick, death → respawn
- [x] the **living-world heartbeat** -- `world.state`, `rest` + regen
- [x] **the Cinder Front moral arc** -- join, ash-sworn, defy, market refusal
- [x] the Refugee Waystation -- `talk`, `treat` (tide-gated medic)
- [x] the tinker **economy** -- `list`/`buy`
- [x] the **holding-pit rescue** -- warden, `free` → `grid.rescued`; warden grace window + antidote affordance gate
- [x] **dreams** -- `sleep` → `char.dream`
- [x] **persistence** -- `CharStore`/`FileStore`; resume on a known name
- [x] **Docker** + **CI** (GHCR push + auto-roll to biafra via `rust-choir-roll`)

**Phase 2 -- multiplayer and federation (world-side)**
- [x] **Multiplayer** -- session registry, `tell`/`reply`/`yell`/`emote`, `room.info.players`
- [x] **`listen` + `ping`** -- `grid.transmission`, `grid.echo`, `grid.federation`
- [x] **`/map.svg`** -- world map endpoint
- [x] **Rust Choir identity** -- Grid Gate tract, archivist voice (see `docs/WORLD.md`)
- [x] **the redemption arc** -- Returned / ash-marked penance
- [x] **the data-leech zone** -- sump → floodgate → Cold Storage Row, core-shard quest
- [x] **the Cinder Front stronghold (endgame)** -- checkpoint, cages, Ashmonger
- [x] **Grid Hub HTTP client** -- `internal/grid/RemoteHub`, `worlds`/`travel`, tide,
  ledger, gridcast, presence, rescued/fallen rolls (production: `GRID_HUB_URL`)

## Shipped this session (rancid, 2026-07-09)

| PR | What |
|---|---|
| [#35](https://github.com/skyphusion-labs/hollow-grid-go/pull/35) | TS parity: ground loot, get/drop/use/examine, flee, stolen-kill vitals, poison, mob loot/XP, sell `value`, moral `room.actions` from faction, missing verbs/aliases |
| [#36](https://github.com/skyphusion-labs/hollow-grid-go/pull/36) | Forgive recipient prose + `char.forgiven` use `PushReliable` (kapo smoke was dropping under heartbeat load) |

Both merged + GHCR `latest` + `rust-choir-roll` to biafra. New transport files:
`internal/transport/{combat,ground,items_cmds}.go` (+ `combat_parity_test.go`).

## Next (optional / not blockers)

- [ ] **NPCs + `talk` depth** in the tavern (dust-dealer / wench prose beyond
  affordances); smoke already green.
- [ ] **Smoke client hardening** (upstream): Node can abort the suite on a late
  WebSocket `ErrorEvent` after `RD.sock.close()` even when assertions passed;
  wrap or harden `mkClient` open/error handlers in `the-hollow-grid/smoke.mjs`
  if prod smoke runs keep dying mid-suite under tunnel jitter.
- [ ] **Warden grace flake** -- tighten or SKIP the post-respawn `free` check more
  cleanly when combat variance wins (upstream smoke concern).

## Phase 3 -- federation client (schedulable; #42)

Grid Hub HTTP ingress is **live** at `https://grid-hub.skyphusion.org/rpc`.
hollow-grid-py already federates against it (worked example; CF User-Agent gotcha
fixed in hollow-grid-py#6). Rust Choir uses the same pattern via
`internal/grid/RemoteHub` when `GRID_HUB_URL` is set.

Remaining Go-port work is **#42** (federation client hardening): tick loop survives
hub failure (`finally`-reschedule), 2s timeout cap on hub polls, structured events on
cross-fighter state changes (requirements from the-hollow-grid PR #55 / v0.29.9).
Test the blackholed-hub path, not only the happy path.

## Deferred -- hub-side trust only (upstream)

The Grid Hub **server** (authoritative D1 + GridHub DO) lives in `the-hollow-grid`.
Trust hardening (per-world keys, leased progression deltas, commit validation) is
documented in `docs/federation.md` section 10 and is not required for the current
single-operator fleet (hollow + Dustfall + Rust Choir).

## Conformance: how to read the scoreboard

```sh
# single-world (federation phase SKIPs if DUSTFALL_URL unset)
MUD_URL=ws://localhost:8790/ws node /path/to/the-hollow-grid/smoke.mjs

# full federation phase (second TS world must be live)
MUD_URL=ws://localhost:8790/ws \
DUSTFALL_URL=ws://localhost:8788/ws \
node /path/to/the-hollow-grid/smoke.mjs

# prod Rust Choir
MUD_URL=wss://rustchoir.skyphusion.org/ws \
DUSTFALL_URL=wss://dustfall.skyphusion.org/ws \
node smoke.mjs
```

Assert on `@event`, not prose. A green run is the definition of done for the
world port; remaining red on standalone-only runs is expected when federation
targets are unreachable.

## Load testing (mud-bots)

LLM agents live in the separate [`mud-bots`](https://github.com/SkyPhusion/mud-bots)
repo (`hollow-grid/bot.mjs`, GHCR `mud-bots-hg`). Fleet layout, AIG tokens, and
the 11-bot soak (3 hollow + 3 dustfall + 5 rustchoir) are documented in
`fleet-chezmoi/system/stacks/biafra/mud-bots/README.md`. Bot findings append to
`*-bugs.jsonl`; server-side regressions belong in hollow-grid-go issues.
