# hollow-grid-go

A **Hollow Grid world server, in Go.** The Hollow Grid is a federated MUD whose
reference implementation is TypeScript on Cloudflare Workers; this is a
from-scratch port of the *world half* — a single autonomous game world that
speaks the Grid's language-agnostic wire protocol and can (eventually) join the
federation as a node.

> The Hollow Grid is a dead network that outlived its makers. Worlds are nodes on
> that network; the shared backend *is* the Grid. It is built to be a place where
> an agent can perceive, choose, grow, and be remembered — where the moral weight
> of a choice is legible as data, not buried in prose. This port keeps that intact.

- **Upstream contract:** [`the-hollow-grid/docs/protocol.md`](https://github.com/SkyPhusion/the-hollow-grid) — the wire spec is deliberately language-agnostic; a Go world is a first-class citizen of the same Grid.
- **Definition of done:** the upstream `smoke.mjs` conformance suite. Currently **107 / 134** checks green on branch `feat/rust-choir-world`; remaining red is endgame map content (data-leech zone, Cinder Front stronghold) and full Grid Hub federation (tide, travel, gridcast, rescued roll).
- **Status:** a fully playable **standalone** world. Federation is additive and never blocks play; it is not wired yet.

## Quick start

```sh
# run it (needs Go 1.26+)
go run ./cmd/world --addr :8790 --world-name "The Hollow Grid (Go)" --data ./data

# play it with any raw WebSocket client
wscat -c ws://localhost:8790/ws

# or drive the bot / smoke suite from the upstream repo against it
MUD_URL=ws://localhost:8790/ws node bot.mjs
MUD_URL=ws://localhost:8790/ws node smoke.mjs
```

### Docker

```sh
docker build -t hollow-grid-go .
docker run -d --name hollow-grid-go -p 8790:8790 -v hollow-grid-go-data:/data hollow-grid-go
curl localhost:8790/health
```

The image is multi-stage: a static (CGO-off) binary on `gcr.io/distroless/static`.
`/data` is a volume for the local character store.

## What's built

A world a player (or an LLM agent) can actually live in:

| System | What it does |
|---|---|
| **Transport** | `/ws` WebSocket, plain UTF-8, CRLF lines; login flow (banner → name → race menu → play), name-based identity, resume |
| **`@event` channel** | every player-affecting state change emitted as `@event <name> <json>` alongside the prose |
| **Health** | `/health` (liveness) + `/health/deep` (per-dependency) probes |
| **The world** | the Cracked Nexus and its opening map (tavern, market, workshop, roof, tunnels), out into the wastes (Ash Flats → Scorch Road → Refugee Waystation), plus a preserved creative bonus zone |
| **Races** | the 7 canonical races, each with a Cinder Front *stance* and a signature cooldown ability (Requisition, Vanish, Regenerate, …) |
| **Items** | inventory, equipment (`wield`/`remove` → `char.equipment`), a starter shiv |
| **Combat** | `attack <mob>` → async tick-resolved fights on the `combat.*` channel, death → respawn |
| **The living world** | a heartbeat clock (`world.state` phase/weather), `rest` + HP regen — the world turns on its own |
| **The moral arc** | the Cinder Front recruiter; `join` brands you `faction:front`; a *hunted* race who joins is branded **ash-sworn** (the kapo), with the choice flagged `valence:"grave"`; the honest market refuses collaborators; the Refugee Waystation reads your standing |
| **Rescue** | beat the holding-pit warden, `free` the captive — a real virtuous act on `grid.rescued`, named and unfarmable |
| **Economy** | the tinker's shop (`list`/`buy` gear for gold) |
| **Dreams** | `sleep` → `char.dream`, a mirror of your own record |
| **Persistence** | the canonical `CharSheet` saved locally (the seam the federation client will later implement against the Grid) |
| **CI** | GitHub Actions: `go vet` + build, unit tests + the upstream E2E conformance suite. Image build + deploy are not yet automated. |

See [docs/COMMANDS.md](docs/COMMANDS.md) for the verb set and [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the design.

## Configuration (flags)

| Flag | Default | Meaning |
|---|---|---|
| `--addr` | `:8790` | listen address |
| `--world-name` | `The Hollow Grid (Go)` | display name |
| `--world-url` | `""` | this world's public URL (for the federation registry, later) |
| `--data` | `data` | directory for the local character store |

## Layout

```
cmd/world/         the server entrypoint (flags, HTTP server, graceful shutdown)
internal/event/    the @event channel framing
internal/world/    the world model: rooms, races, mobs, items, the living clock, @event payloads
internal/store/    the CharStore interface + a dependency-free FileStore (the federation seam)
internal/transport/ the player transport: the WebSocket server and the session select-loop
docs/              protocol notes, architecture, the build plan, the command reference
Dockerfile         multi-stage build -> distroless
```

## Development

- `go test ./...` — unit + transport conformance tests (drive real WebSocket sessions)
- `go vet ./...`, `gofmt -l .` — clean before commit
- The upstream `smoke.mjs` is the scoreboard: point it at a running server (or the container) and watch the count climb.

## License

See [LICENSE](LICENSE).
