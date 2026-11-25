# hollow-grid-go

This file is the guide for any coding agent and any contributor. Follow it.

## What this project is

Rust Choir is a Hollow Grid world server, written in Go. It is a language port of the world half. The reference implementation is [the-hollow-grid](https://github.com/skyphusion-labs/the-hollow-grid) (TypeScript on Cloudflare Workers). Players connect over WebSocket. They play with plain-text commands.

Protocol authority is `the-hollow-grid/docs/protocol.md`. Do not invent protocol. Mirror it. Assert on `@event`. Do not assert on English prose alone.

The world is playable standalone. It is also playable federated. Definition of done is upstream `smoke.mjs`. Score it against current the-hollow-grid head. A dated score is not the current gate. Read [README.md](README.md) and [docs/PLAN.md](docs/PLAN.md) for the last recorded score. Score again before you cite it.

The public world runs in a Cloudflare container. Open https://rustchoir.skyphusion.org/ or connect to `wss://rustchoir.skyphusion.org/ws`. You can also run the published image `ghcr.io/skyphusion-labs/hollow-grid-go`.

## Federation

A world is content on a generic engine. The Grid Hub is the shared backend. A Go world joins the same Grid as a TypeScript world because the wire protocol is language-agnostic.

Set `GRID_HUB_URL` to use `internal/grid/RemoteHub`. `internal/store` stays the offline fallback. A hub failure must not block play.

## Contract

Read `the-hollow-grid/docs/protocol.md` before you add a command or an event.

Transport is `/ws`, plain UTF-8, CRLF lines. Login runs banner, then name, then the race menu, then play. Identity is the character name. A returning name resumes. `/health` is liveness. `/health/deep` checks each dependency.

Every player-affecting state change is an `@event <name> <json>` line next to the prose. Examples include `room.info`, `char.vitals`, `combat.*`, `world.state`, `grid.rescued`, `char.dream`, `char.equipment`. The full list is the upstream spec. When you add state a client needs, emit the event. Then add an assertion.

## Commands

This is a Go module, `github.com/SkyPhusion/hollow-grid-go`. It is not an npm project. The runtime dependency is `github.com/coder/websocket`. See `go.mod` for the Go version. The Dockerfile builds with that same Go line. Do not drift [.github/workflows/ci.yml](.github/workflows/ci.yml) or [.github/workflows/release.yml](.github/workflows/release.yml) off `go.mod`.

Flag defaults live in `cmd/world/main.go`: address `:8790`, world name `Rust Choir`, data dir `data`.

```bash
go run ./cmd/world --addr :8790 --world-name "Rust Choir" --data data
go vet ./...
gofmt -l .                  # must print nothing
go test ./...
go build ./...
```

Play and score from a the-hollow-grid checkout:

```bash
wscat -c ws://localhost:8790/ws
MUD_URL=ws://localhost:8790/ws node smoke.mjs
```

Set `DUSTFALL_URL` when you score the federation phase. Point it at a Dustfall you are actually running.

```bash
docker build -t hollow-grid-go .
docker run -d --name hollow-grid-go -p 8790:8790 -v hollow-grid-go-data:/data hollow-grid-go
```

## How to verify

Two layers. `go test ./...` is the local gate, with `go vet ./...` and a clean `gofmt -l .`. Transport tests in `internal/transport/conn_test.go` drive real WebSocket sessions.

CI runs `go vet ./...`, `go test -race ./...`, and `go build ./...`. See [.github/workflows/ci.yml](.github/workflows/ci.yml).

Upstream `smoke.mjs` is the definition-of-done scoreboard. Point it at a running server or the container. Assert on `@event`, not prose.

CI and the release workflow both require upstream smoke to pass against a local server or image. Verify the public Cloudflare host separately.

## Architecture

```
cmd/world/          server entry (flags, HTTP, shutdown)
internal/event/     @event framing
internal/grid/      Grid Hub client and LocalHub fallback
internal/world/     rooms, races, mobs, items, clock, event payloads
internal/store/     CharStore and FileStore (offline fallback)
internal/transport/ WebSocket server, session loop, multiplayer hub
docs/               ARCHITECTURE.md, COMMANDS.md, PLAN.md, WORLD.md
Dockerfile          multi-stage build to a distroless image
```

Game content is data. The engine stays generic. Worlds, races, mobs, and items live in `internal/world/` data. The seven canonical races each have a Cinder Front stance and a cooldown ability. Add content by editing data. Do not edit the engine for a new item.

A heartbeat clock advances `world.state` (phase and weather). It also drives tick combat (`combat.*`) and HP regen. The world keeps moving without new input.

The moral arc is data. The Cinder Front recruiter brands `faction:front`. A hunted race who joins takes the brand `ash-sworn`, with `valence:"grave"`. Rescue is a named act on `grid.rescued`. You cannot farm it. These are `@event` payloads.

`FileStore` in `internal/store` implements `CharStore` for local play. Set `GRID_HUB_URL` and `RemoteHub` loads the canonical sheet. It also commits that sheet (`LoadCharacter`, `CommitCharacter`).

## Conventions

Do not use an em dash (U+2014) or an en dash (U+2013) in source, comments, docs, or in-game text. Use a comma, parentheses, or two hyphens (`--`).

Prefer the Go standard library. Justify any new dependency. Output is plain UTF-8 with CRLF lines. An undeclared exit returns a clear message. An unknown command returns a clear message. Silence was the bug that started the project.

This is a public repo. CI uses GitHub-hosted `ubuntu-latest`. That choice stays safe for fork pull requests.

## Docs

Write Simplified Technical English. Use short sentences. Put one idea in each sentence. Use the active present tense. Use plain words. Use the same word every time.

State the limit next to the capability. Do not turn a measured result into a promise. Quote a smoke score with its date. Say that it is not the current gate.

## Commits and version

Use Conventional Commits (`feat(scope):`, `fix(scope):`, `docs:`). The body says why.

There is no manifest version. The git tag is the version of record. The CHANGELOG heading is the version of record. The tag form is `v*`.

A version moves only in a release pull request. Write the changelog in that same pull request. A feature pull request does not bump the version. A fix pull request does not bump the version. Put the tag on the merge commit of the release pull request.

While the project is pre-1.0, use `0.MINOR.PATCH`. PATCH covers a fix. MINOR covers a new system, command, content set, or a batch of newly green conformance checks.

Before a release pull request, look for a tag of the current version. A matching tag means the cycle is closed. The release pull request then bumps. No matching tag means a bump already landed outside a release pull request. Stop and read why.

## Release

The release workflow publishes tagged images to GHCR. Cloudflare hosting uses `wrangler.jsonc` separately.

For a Worker-only change on the existing deployment, use Wrangler 4.168.0 or
newer with `wrangler deploy --containers-rollout=none`. This reuses the deployed
container image metadata. Before changing the container image, plan a data
migration. A filesystem snapshot restores the image it was taken from.

A merge to `main` runs CI and builds the image. It does not push the image. Cut an annotated tag on `main` to publish.

```bash
git tag -a vX.Y.Z -m "vX.Y.Z"
git push origin vX.Y.Z
```

`release.yml` checks that the tag commit is an ancestor of `origin/main`. Then it pushes `:<sha>` and `:latest`.

## Safety

Do not put a secret in a tracked file. Do not put a secret in a log. Do not put a secret in an issue.
