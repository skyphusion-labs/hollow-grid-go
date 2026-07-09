# CLAUDE.md

Guidance for Claude Code (and the crew) working in this repo.

## What this is

**A Hollow Grid world server, in Go.** The Hollow Grid is a federated text MUD whose reference
implementation is TypeScript on Cloudflare Workers ([the-hollow-grid](https://github.com/SkyPhusion/the-hollow-grid));
this is a from-scratch port of the **world half** -- a single autonomous game world that speaks the
Grid's language-agnostic wire protocol and can (eventually) join the federation as a node. Players (or
LLM agents) connect over WebSocket and play with plain-text commands.

**Status:** a fully playable STANDALONE world. The definition of done is the upstream `smoke.mjs`
conformance suite; currently **56 / 134** checks green (the remainder are the federation engine +
multiplayer, see `docs/PLAN.md`). Federation is additive and never blocks play; it is not wired yet.

## The Grid federation (the shared map)

```
        the-hollow-grid (TS / Cloudflare Workers) -- the reference implementation
                 |
                 |  the same language-agnostic wire protocol (docs/protocol.md upstream)
                 |
   +-------------+--------------------+
   |                                  |
 world node (TS world)          world node (THIS repo: a Go world)
   |                                  |
   +------------- Grid Hub ------------+   <-- the shared federation backend IS the Grid
                     |
            wss:// players + agents (per world, plain-text + the @event channel)
```

A "world" is content on a generic engine; the shared backend (the Grid Hub) is what makes it a
federation. A Go world is a first-class citizen of the same Grid as a TS world because the wire protocol
is deliberately language-agnostic. The federation client is the open seam here (`internal/store`), not
yet implemented.

## The contract (build to the upstream wire protocol)

- **Upstream spec:** `the-hollow-grid/docs/protocol.md` -- the language-agnostic wire protocol, the
  full `@event <name> <json>` vocabulary, and the federation contract. THIS is what a port builds
  against; do not invent protocol, mirror it.
- **Transport:** `/ws` WebSocket, plain UTF-8, CRLF lines; a login flow (banner -> name -> race menu ->
  play), name-based identity, resume. Plus `/health` (liveness) + `/health/deep` (per-dependency).
- **The `@event` channel is the machine-readable source of truth.** Every player-affecting state change
  is emitted as `@event <name> <json>` alongside the human prose (`room.info`, `char.vitals`,
  `combat.*`, `world.state`, `grid.rescued`, `char.dream`, `char.equipment`, and so on). RULE: any
  canonical, player-affecting state belongs in a structured event, never prose-only -- the two channels
  drifting apart is what makes a MUD un-testable. When you add state a client/bot/test would need, emit
  it here and add an assertion.

## Commands

This is a Go module (`github.com/SkyPhusion/hollow-grid-go`), NOT an npm project. Single runtime
dependency: `github.com/coder/websocket`.

```bash
# Run the world (default :8790):
go run ./cmd/world --addr :8790 --world-name "The Hollow Grid (Go)" --data ./data

# Clean before commit + test:
go vet ./...
gofmt -l .                  # must print nothing
go test ./...               # unit + transport conformance (drive real WebSocket sessions)
go build ./...

# Play / score it with the upstream client + conformance suite (from the the-hollow-grid repo):
wscat -c ws://localhost:8790/ws
MUD_URL=ws://localhost:8790/ws node smoke.mjs    # the scoreboard: watch the 56/134 count climb

# Container (multi-stage static CGO-off binary on distroless):
docker build -t hollow-grid-go .
docker run -d --name hollow-grid-go -p 8790:8790 -v hollow-grid-go-data:/data hollow-grid-go
```

**Toolchain:** `go.mod` requires **Go 1.26.x** (`go 1.26.4`) and the Dockerfile builds on `golang:1.26`.
NOTE: `.github/workflows/ci.yml` currently pins `setup-go` to `1.22`, a drift from `go.mod` to reconcile
(a 1.22 toolchain cannot build a module that declares `go 1.26.4`). When touching CI, align it to
`go.mod` (the intent stated in `release.yml`: "Go is pinned to go.mod so CI never drifts").

## Verifying changes

Two layers. (1) `go test ./...` is the local gate: unit tests plus transport conformance tests that
drive REAL WebSocket sessions (`internal/transport/conn_test.go`). Run it plus `go vet ./...` and
`gofmt -l .` (clean) before committing. (2) The upstream `smoke.mjs` is the definition-of-done
scoreboard: point it at a running server (or the container) and the green count is the real progress
metric (56/134 today). Assert on the `@event` channel, not prose -- it is the machine-readable truth.

## Architecture

```
cmd/world/          the server entrypoint (flags, HTTP server, graceful shutdown)
internal/event/     the @event channel framing
internal/world/     the world model: rooms, races, mobs, items, the living clock, @event payloads
internal/store/     the CharStore interface + a dependency-free FileStore (the federation seam)
internal/transport/ the player transport: the WebSocket server and the session select-loop
docs/               protocol notes (ARCHITECTURE.md, COMMANDS.md, PLAN.md)
Dockerfile          multi-stage build -> distroless
```

- **Game content is data, the engine is generic.** Worlds, races (the 7 canonical, each with a Cinder
  Front stance + a cooldown ability), mobs, and items live in `internal/world/` data, not logic. Add
  content by editing the data, not the engine.
- **The living world turns on its own.** A heartbeat clock advances `world.state` (phase/weather) and
  drives async tick-resolved combat (`combat.*`) and HP regen, so the world keeps moving without input.
- **The moral arc is legible as data.** The Cinder Front recruiter brands `faction:front`; a hunted
  race who joins is branded `ash-sworn` (the kapo) with the choice flagged `valence:"grave"`; rescue is
  a real, named, unfarmable virtuous act on `grid.rescued`. These are first-class `@event` payloads.
- **The federation seam is `internal/store`.** The `CharStore` interface (today a local `FileStore`) is
  exactly where the federation client will later persist to the Grid; keep the persistence boundary clean.

## Conventions

- **No em-dashes (U+2014) or en-dashes (U+2013) anywhere** in source, comments, docs, or in-game text.
  Use commas, semicolons, parentheses, or `--`.
- Handle / username is `skyphusion` across all services.
- Go, standard library first; one runtime dep (`coder/websocket`). Justify any new dependency.
- Output is plain UTF-8 with CRLF lines for clean rendering in line-based clients; an undeclared
  exit/command returns a clear message (no silent no-op -- that was the bug that motivated the project).
- **CI runners:** PUBLIC repo -> GitHub-hosted `ubuntu-latest` for `ci.yml` and `release.yml`
  (fork-safe). `release.yml` builds + pushes to GHCR (`:<sha>` + `:latest`) on push to `main`;
  the smoke conformance run is informational / non-blocking while the port is in progress (a
  partial pass never reds the build). After a green GHCR push, dispatches `fleet-chezmoi`
  `rust-choir-roll` to pull + redeploy Rust Choir on biafra (org secrets
  `FLEET_DISPATCH_TOKEN` + `GHCR_READ_PAT`; see `crew-secrets` README and
  `fleet-chezmoi/system/swarm/RUNBOOK-rust-choir-roll.md`).

## Crew + identity

- The FIRST command in any op is the member's own login shell: `sudo -u <member> bash -lc '<ops>'`
  (loads their `$HOME`, their `~/dev/hollow-grid-go` clone, their gh creds). Commits and PRs land under
  the member's `skyphusion-<member>` identity, never Conrad's.
- The TS reference world is `~/dev/the-hollow-grid`; the wire protocol + the `smoke.mjs` scoreboard live
  there, and this port builds to that spec.

## Commits & versioning

Conventional Commits (`feat(scope):`, `fix(scope):`, `docs:`); body explains the why, footer lists
files touched. SemVer-style `0.MINOR.PATCH` while pre-1.0 (PATCH for fixes, MINOR for a new system /
command / content set or a batch of newly-green conformance checks).
