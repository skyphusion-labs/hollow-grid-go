# hollow-grid-go

A Go implementation of a **world server** for [The Hollow Grid](https://github.com/SkyPhusion/the-hollow-grid)
federated MUD. The reference grid is TypeScript on Cloudflare Workers; this is a
from-scratch port of the *framework* a world needs to speak the Grid protocol and
(optionally) join the federation as a first-class node.

The wire contract is language-agnostic and specified upstream in
`docs/protocol.md`. This repo targets that contract verbatim so the existing
clients, `bot.mjs`, and the `smoke.mjs` assertion suite work against it unchanged.

## What a world provides (porting checklist, protocol.md s4)

1. A WebSocket `/ws` endpoint speaking the line protocol (login -> play).
2. The full command set + the structured `@event` channel.
3. Local persistence for rooms/mobs/items/positions.
4. A tick loop: combat, regen/poison, respawns, the living world.
5. Optionally a `GridHubApi` client to federate (a world is fully playable solo).
6. `/health` + `/health/deep` probes.

## Run

```
go run ./cmd/world --addr :8790 --world-name "The Hollow Grid (Go)"
# then: wscat -c ws://localhost:8790/ws
#       curl localhost:8790/health
```

## Status

Early scaffold. See docs/PLAN.md for the phased build. Federation is additive and
not yet wired; this runs as a standalone world.
