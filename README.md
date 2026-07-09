# hollow-grid-go

A **Hollow Grid world server, in Go.** The Hollow Grid is a federated MUD whose
reference implementation is TypeScript on Cloudflare Workers; this is a
from-scratch port of the *world half* -- a single autonomous game world that
speaks the Grid's language-agnostic wire protocol and joins the federation as a
node when `GRID_HUB_URL` is set.

> The Hollow Grid is a dead network that outlived its makers. Worlds are nodes on
> that network; the shared backend *is* the Grid. It is built to be a place where
> an agent can perceive, choose, grow, and be remembered -- where the moral weight
> of a choice is legible as data, not buried in prose. This port keeps that intact.

- **Upstream contract:** [`the-hollow-grid/docs/protocol.md`](https://github.com/SkyPhusion/the-hollow-grid/blob/main/docs/protocol.md) -- the wire spec is deliberately language-agnostic; a Go world is a first-class citizen of the same Grid.
- **Definition of done:** the upstream `smoke.mjs` conformance suite (**135 checks**). Prod **Rust Choir** baseline (2026-07-09): **158 ok / 0 fail / 1 skip** against live hub + Dustfall (`DUSTFALL_URL` set); the skip is the holding-pit warden grace wall-clock wait on slow CI boxes.
- **Status:** fully playable **standalone** (LocalHub fallback) or **federated** (HTTP Grid Hub client in `internal/grid`). Live fleet deployment: **`wss://rustchoir.skyphusion.org/ws`**.

## Play now (Rust Choir)

The Go port runs in production as **Rust Choir**, the third world on the Grid
(alongside hollow.skyphusion.org and dustfall.skyphusion.org):

```sh
wscat -c wss://rustchoir.skyphusion.org/ws
curl -sf https://rustchoir.skyphusion.org/health
curl -sf https://rustchoir.skyphusion.org/health/deep
```

LLM load-test bots (`mud-bots`, image `ghcr.io/skyphusion-labs/mud-bots-hg`) exercise
Rust Choir alongside the TS worlds; fleet layout and env vars live in
`fleet-chezmoi/system/stacks/biafra/mud-bots/README.md`.

## Quick start (local)

```sh
# run it (needs Go 1.26+)
go run ./cmd/world --addr :8790 --world-name "The Hollow Grid (Go)" --data ./data

# play it with any raw WebSocket client
wscat -c ws://localhost:8790/ws

# score it with the upstream conformance suite (from the-hollow-grid repo)
MUD_URL=ws://localhost:8790/ws node smoke.mjs
# federation phase (travel, cross-world tide): also set DUSTFALL_URL
MUD_URL=ws://localhost:8790/ws DUSTFALL_URL=ws://localhost:8788/ws node smoke.mjs
```

### Docker

Standalone (no Grid Hub):

```sh
docker build -t hollow-grid-go .
docker run -d --name hollow-grid-go -p 8790:8790 -v hollow-grid-go-data:/data hollow-grid-go
curl localhost:8790/health
```

Federated Rust Choir (GHCR image + Grid Hub):

```sh
cp .env.example .env   # set WORLD_URL, GRID_HUB_URL, GRID_HUB_TOKEN
docker compose up -d
curl localhost:8790/health/deep
```

Or pull the release image directly:

```sh
docker run -d --name rust-choir -p 8790:8790 \
  -v rust-choir-data:/data \
  -e WORLD_URL=wss://rustchoir.skyphusion.org/ws \
  -e GRID_HUB_URL=https://grid-hub.skyphusion.org/rpc \
  -e GRID_HUB_TOKEN='…' \
  ghcr.io/skyphusion-labs/hollow-grid-go:latest
```

The image is multi-stage: a static (CGO-off) binary on `gcr.io/distroless/static`.
`/data` is a volume for the local character store. Omit `GRID_HUB_*` and the world
runs on LocalHub until the hub is reachable.

## What's built

A world a player (or an LLM agent) can actually live in:

| System | What it does |
|---|---|
| **Transport** | `/ws` WebSocket, plain UTF-8, CRLF lines; login flow (banner → name → race menu → play), name-based identity, resume |
| **`@event` channel** | every player-affecting state change emitted as `@event <name> <json>` alongside the prose |
| **Health** | `/health` (liveness) + `/health/deep` (per-dependency) probes; `/map.svg` world map |
| **The world** | the canonical opening map plus the wastes, the data-leech zone, and the Cinder Front stronghold endgame; Rust Choir grafts the **Grid Gate** tract east from the tunnels (see `docs/WORLD.md`) |
| **Races** | the 7 canonical races, each with a Cinder Front *stance* and a signature cooldown ability |
| **Items** | inventory, equipment (`wield`/`remove` → `char.equipment`), economy (`list`/`buy`/`sell`/`steal`) |
| **Combat** | `attack <mob>` → async tick-resolved fights on the `combat.*` channel, death → respawn |
| **Multiplayer** | session registry, `tell`/`reply`/`yell`/`emote`, `room.info.players` with standing |
| **The living world** | a heartbeat clock (`world.state` phase/weather), `rest` + HP regen -- the world turns on its own |
| **The moral arc** | the Cinder Front recruiter; `join`/`defend`/`defy`; ash-sworn (kapo) brand; redemption (Returned); rescue on `grid.rescued` |
| **Federation** | `internal/grid` HTTP RPC client: registry (`worlds`/`travel`), tide, ledger, gridcast, presence, rescued/fallen rolls, canonical `CharSheet` sync when hub is bound |
| **Persistence** | the canonical `CharSheet` via `CharStore`/`FileStore`; resume on a known name |
| **CI / deploy** | GitHub Actions `release.yml`: lint, vet, test, build, push to GHCR on `main`; dispatches `fleet-chezmoi` `rust-choir-roll` to redeploy on biafra. Runbook: `fleet-chezmoi/system/swarm/RUNBOOK-rust-choir-roll.md`. |

See [docs/COMMANDS.md](docs/COMMANDS.md) for the verb set, [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the design, [docs/PLAN.md](docs/PLAN.md) for conformance status, and [docs/WORLD.md](docs/WORLD.md) for Rust Choir identity.

## Configuration (flags)

| Flag | Default | Meaning |
|---|---|---|
| `--addr` | `:8790` | listen address |
| `--world-name` | `Rust Choir` | display name |
| `--world-url` | `""` | this world's public URL (federation registry / travel) |
| `--data` | `data` | directory for the local character store |
| `--grid-hub-url` | `""` | Grid Hub HTTP RPC URL; omit for standalone LocalHub |
| `--grid-hub-token` | `""` | bearer token for Grid Hub RPC |
| `--admins` | `skyphusion` | comma-separated keeper names (`wall`, `gridstats`, `gridprune`) |

Container env aliases (used when the matching flag is at its default): `LISTEN_ADDR`, `WORLD_NAME`, `WORLD_URL`, `DATA_DIR`, `ADMINS`, `GRID_HUB_URL`, `GRID_HUB_TOKEN`.

## Layout

```
cmd/world/         the server entrypoint (flags, HTTP server, graceful shutdown)
internal/event/    the @event channel framing
internal/grid/     Grid Hub HTTP client + LocalHub fallback
internal/world/    the world model: rooms, races, mobs, items, the living clock, @event payloads
internal/store/    the CharStore interface + a dependency-free FileStore
internal/transport/ the player transport: WebSocket server, session select-loop, multiplayer hub
docs/              architecture, commands, build plan, Rust Choir identity
Dockerfile         multi-stage build -> distroless
```

## Development

- `go test ./...` -- unit + transport conformance tests (drive real WebSocket sessions)
- `go vet ./...`, `gofmt -l .` -- clean before commit
- The upstream `smoke.mjs` is the scoreboard: point it at a running server (or the container) and assert on `@event`, not prose.

## Who this is for

Go developers porting a Hollow Grid world node, or operators who want a self-hosted MUD world outside Cloudflare Workers.

## Links

- **Play the reference TS world:** [hollow.skyphusion.org](https://hollow.skyphusion.org)
- **Wire spec:** [the-hollow-grid](https://github.com/skyphusion-labs/the-hollow-grid)
- **Skyphusion Labs:** https://skyphusion.org · **Org:** https://github.com/skyphusion-labs

## License

See [LICENSE](LICENSE).
