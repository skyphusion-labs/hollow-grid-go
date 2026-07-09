# Build plan & status

Porting the Hollow Grid world framework to Go, against the upstream
`docs/protocol.md`. The scoreboard is the upstream `smoke.mjs` (134 checks at the
time of writing): **build the port to pass it, phase by phase.** Prod Rust Choir
(Rust Choir + Dustfall) baseline: **158 ok / 0 fail / 1 skip** (2026-07-09); the
skip is the warden grace wall-clock wait on slow boxes -- fixed on branch
`feat/warden-grace-window`.

## Done

**Phase 0 — transport foundation**
- [x] HTTP server, graceful shutdown, `/health` + `/health/deep`
- [x] `/ws` WebSocket, UTF-8 text, CRLF lines
- [x] login flow: banner → name → race menu → play; name-based identity
- [x] the `@event` channel framing

**Phase 1 — the world**
- [x] the 7 canonical races, with Cinder Front stance + signature abilities (Requisition + cooldown, the heals, Forage)
- [x] the canonical opening map (the Cracked Nexus, tavern, market, holding pit, workshop, roof, tunnels) + the wastes (Ash Flats, Scorch Road, Refugee Waystation)
- [x] items / inventory / equipment (`wield`/`remove` → `char.equipment`), the starter shiv, `title`
- [x] mobs (`room.info.mobs`), `consider`, `look <mob>`
- [x] **async combat** — `attack` → `combat.start/round/end` on a tick, death → respawn
- [x] the **living-world heartbeat** — `world.state` clock (pure function of elapsed time), `rest` + regen
- [x] **the Cinder Front moral arc** — `join` → `faction:front`; the ash-sworn (kapo) brand for hunted races; `room.actions` moral choices with `valence` (`grave` for a hunted join); `defy`; the honest market refuses collaborators
- [x] the Refugee Waystation — `talk` reacts to standing, `treat` (medic gated by the collective tide, graceful without a hub)
- [x] the tinker **economy** — `list`/`buy` gear for gold (20 starting gold)
- [x] the **holding-pit rescue** — beat the warden, `free` the captive → `grid.rescued` (named, +morality, unfarmable); post-kill **warden grace window** (v0.29.3) and antidote affordance gate (v0.29.8)
- [x] **dreams** — `sleep` → `char.dream`, a mirror of your record
- [x] **persistence** — the canonical `CharSheet` via `CharStore`/`FileStore`; resume on a known name
- [x] **Docker** (multi-stage -> distroless) + **CI** (GitHub Actions: `go vet` + build, unit tests + the upstream conformance suite; GHCR push + auto-roll to biafra on `main` via `fleet-chezmoi` `rust-choir-roll`)

## Next (world-local)

- [x] **Multiplayer** -- session registry, `tell`/`reply`/`yell`/`emote`, `room.info.players` with standing
- [x] **`listen` + `ping`** -- `grid.transmission`, `grid.echo`, `grid.federation` (local hub fallback)
- [x] **`/map.svg`** -- minimal world map endpoint
- [x] **Rust Choir identity** -- default world name, Grid Gate tract linked from tunnels (see `docs/WORLD.md`)
- [ ] **NPCs + `talk`** in the tavern (the dust-dealer / wench; `room.actions` social affordances) -- smoke green; content depth optional
- [x] **the redemption arc** — the way back from the cinders (a Front member / ash-sworn redeeming)
- [x] **the data-leech zone** — the sump → floodgate → Cold Storage Row, the data-leech mob, the core-shard quest from the stranded operator
- [x] **the Cinder Front stronghold (endgame)** — the muster yard past the checkpoint/gate, Front troopers, the cages, the **Ashmonger** boss on the dais

## Next (architecture)

- [x] **Multiplayer** -- a shared session registry on the `Server` plus a broadcast
  path, so `room.info.players` lists others (with `standing`/`ash-sworn`), and
  `tell`/`reply`/`yell`/`emote` work.
- [ ] **Grid Hub HTTP client** -- fleet nodes cannot use CF service bindings today;
  needs hub ingress or a relay Worker (Phase 3).

## Deferred — the federation engine (not this repo)

The Grid Hub backend (the `GridHubApi`: the shared ledger, the global faction
tide, cross-world chat, the world registry, the rescued/memorial rolls, presence,
travel) is the upstream's other half. This port builds the world side and the
`CharStore` seam only. When the hub exposes an HTTP ingress for external nodes, a
`GridHubApi` client lands here as Phase 3 — additive, best-effort, never blocking
play. Until then these checks (`grid.federation`, `grid.echo`, cross-world `who`,
the persistent rolls, `travel`, `gridcast`) stay red by design.

## Conformance: how to read the scoreboard

```sh
# against a running server (host) or the container
MUD_URL=ws://localhost:8790/ws node /path/to/the-hollow-grid/smoke.mjs
# DUSTFALL_URL too, or it SKIPs the second-world federation phase
```

A green run is the definition of done. The remaining red is the deferred work
above, not defects.
