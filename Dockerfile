# syntax=docker/dockerfile:1
#
# A Hollow Grid world server, containerised. Multi-stage: build a static binary
# against the pinned Go toolchain, then ship just that binary on a distroless
# base (no shell, no package manager, tiny attack surface). The end state is to
# run this via `docker run`, with the character store on a mounted
# /data volume.

# --- build ---
FROM golang:1.26 AS build
WORKDIR /src
# Cache the module download layer separately from the source.
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# A fully static binary (CGO off) so it runs on the scratch-like distroless base.
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /world ./cmd/world

# --- run ---
FROM gcr.io/distroless/static-debian12 AS run
COPY --from=build /world /world
# Local character persistence (CharSheets). Mount a volume to keep it across
# restarts; the world is fully playable without it (a fresh store each run).
VOLUME ["/data"]
EXPOSE 8790
ENTRYPOINT ["/world"]
# Overridable via flags or env (LISTEN_ADDR, DATA_DIR, WORLD_NAME, WORLD_URL,
# GRID_HUB_URL, GRID_HUB_TOKEN). See compose.yaml and .env.example.
CMD ["--addr", ":8790", "--data", "/data", "--world-name", "Rust Choir"]
