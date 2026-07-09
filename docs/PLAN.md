# Build plan & status

Porting the Hollow Grid world framework to Go, against the upstream
`docs/protocol.md`. The scoreboard is the upstream `smoke.mjs` (**135 checks**):
**build the port to pass it, phase by phase.** Prod Rust Choir (hub + Dustfall
live) baseline: **158 ok / 0 fail / 1 skip** (2026-07-09); the skip is the
holding-pit warden grace wall-clock wait on slow boxes.

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

## Next (polish / known gaps)

- [x] **Session-local `resolved` moral state** -- join/defend no longer reappear in
  `room.actions` after reconnect when faction is already set (derived from
  `CharSheet.faction`).
- [ ] **NPCs + `talk` depth** in the tavern (dust-dealer / wench prose beyond
  affordances); smoke green, content optional.
- [x] **Stolen-kill vitals sync** -- TS v0.29.9 parity when another player kills
  your mob mid-fight (`combat.end` + `inCombat: false` for the displaced fighter).

## TS parity (2026-07-09)

Full command-list parity with the reference world beyond smoke:

- [x] Ground loot piles + `get`/`take`, `drop`, `use`/`drink`/`eat`, `examine`
- [x] `flee`, `say`, `sit`, `status`/`hp`, `home`, `time`, verb aliases
- [x] Mob loot tables, XP/level-up on kill, combat poison + poison ticks
- [x] Sell prices from item `value` (ally bonus preserved)

## Deferred -- hub-side / trust (upstream)

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
