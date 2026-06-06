# Build plan

Porting the Hollow Grid world framework to Go, against the upstream
`docs/protocol.md` contract. Federation is additive and last; a standalone world
comes first (protocol.md s4: "A world is fully playable standalone").

## Phase 0 - transport foundation (THIS COMMIT)
- [x] HTTP server, graceful shutdown
- [x] `/ws` WebSocket, plain UTF-8 text, CRLF lines
- [x] login flow: banner -> name prompt -> (race menu TODO) -> play
- [x] `@event` channel framing (`@event <name> <json>`)
- [x] `room.info` + `char.vitals` emission
- [x] `/health` + `/health/deep` probes
- [x] tiny 2-room world graph + `look` + movement

## Phase 1 - the world model
- [ ] race menu for new characters (federated opaque race label)
- [ ] real room graph + content loading (worlds/*.jsonc compatible)
- [ ] mobs, items, inventory; `char.affects`, `char.equipment`, `room.actions`
- [ ] local persistence (SQLite via modernc.org/sqlite, or bolt)

## Phase 2 - the living world (tick loop)
- [ ] combat (`combat.start/round/end`), regen/poison, respawns, death
- [ ] day/night + weather (`world.state`), wandering ghost transmissions

## Phase 3 - federation (GridHubApi client; additive, non-blocking)
- [ ] HTTP client for the GridHubApi (record/recent, tide, gridcast, char load/commit, register/listWorlds)
- [ ] `/grid-event` inbound endpoint for hub fan-out (gridcast)
- [ ] graceful degradation: hub-down => local-only, reconcile on reconnect

## Conformance
The bar is `smoke.mjs` (upstream) passing against this server, and `bot.mjs`
playing it. Same `@event` names/fields, same login flow, same health paths.
