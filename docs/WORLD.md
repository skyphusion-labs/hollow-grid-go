# Rust Choir: the third world on the Grid

**Rust Choir** is the Go fleet node's identity in the Hollow Grid federation. It
speaks the same wire protocol and passes the same conformance suite as the
reference world, but it is not a clone of hollow.skyphusion.org or Dustfall.

**Live:** `wss://rustchoir.skyphusion.org/ws` (biafra container + cloudflared tunnel).

## The pitch

The primary world is the **moral crucible**: the Cinder Front, the market, the
holding pit, the wastes. Dustfall is the same engine wearing a different
deployment badge, proving federation is real.

Rust Choir is the **memory node**.

Where the others ask *what will you do*, Rust Choir asks *what will the network
remember you for?* Its signature geography is the **Grid Gate** tract (reachable
east from the service tunnels): dead terminals, Ash Road, the Memorial Static
where names scroll too fast to read, and a Cinder Checkpoint where you choose
between the Front's coin and the refugees in line.

## What is different here

| Axis | Primary / Dustfall | Rust Choir |
|---|---|---|
| Identity | The Hollow Grid / Dustfall | **Rust Choir** (archivist node) |
| Signature zone | Canonical map only | **Grid Gate + Memorial Static** grafted onto the canonical graph |
| `/map.svg` | TS-generated from rooms.ts | Rust-toned SVG with the bonus tract marked |
| Default listen flavor | Generic dead-network voices | Same transmission pool; prose framing leans **archivist** |
| Fleet home | Cloudflare Workers | **Hetzner fleet** container (`:8790`, distroless GHCR image) |
| Engine | TypeScript World DO | **Go** (`hollow-grid-go`) |

Mechanically the races, Cinder Front arc, holding-pit rescue, and `@event`
vocabulary are identical. Differentiation is **place and voice**, not protocol.

## Design rules (for agents and humans)

1. **Conformance first.** Creative content grafts from rooms the smoke suite does
   not pin (the tunnels east exit is the current graft point).
2. **Moral weight stays data.** Every meaningful choice emits `room.actions`
   with `valence`; rescues emit `grid.rescued`; oaths land in the trace ledger.
3. **Federation is additive.** The world runs standalone on LocalHub; with
   `GRID_HUB_URL` it registers and syncs through `internal/grid/RemoteHub`.

## Federation join (fleet)

Production on biafra (`/opt/stacks/rust-choir`):

| Env | Example |
|---|---|
| `WORLD_NAME` | `Rust Choir` |
| `WORLD_URL` | `wss://rustchoir.skyphusion.org/ws` |
| `GRID_HUB_URL` | `https://grid-hub.skyphusion.org/rpc` |
| `GRID_HUB_TOKEN` | from `crew-secrets` (never commit) |

Deploy / roll: `fleet-chezmoi/system/swarm/RUNBOOK-rust-choir-roll.md` and
`system/stacks/biafra/rust-choir/README.md`. CI on `main` dispatches
`rust-choir-roll` after a green GHCR push.

Players and bots `travel` here from hollow or Dustfall once the hub registry
lists Rust Choir as reachable.

## Play it

```sh
# local dev
go run ./cmd/world --world-name "Rust Choir" --world-url "wss://rustchoir.skyphusion.org/ws"

# prod
wscat -c wss://rustchoir.skyphusion.org/ws

# east from the tunnels -> the Grid Gate tract
```

Score with upstream smoke:

```sh
MUD_URL=wss://rustchoir.skyphusion.org/ws \
DUSTFALL_URL=wss://dustfall.skyphusion.org/ws \
node /path/to/the-hollow-grid/smoke.mjs
```

## Load testing

Five LLM bots (Scrape, Ash, Chrome, Static on biafra + LocalScrape on laptop) soak
Rust Choir alongside six on the TS worlds. See `mud-bots` v1.0.5+ and
`fleet-chezmoi/system/stacks/biafra/mud-bots/README.md`.
