# Rust Choir: the third world on the Grid

**Rust Choir** is the Go fleet node's identity in the Hollow Grid federation. It
speaks the same wire protocol and passes the same conformance suite as the
reference world, but it is not a clone of hollow.skyphusion.org or Dustfall.

The public host is retired. Run the world locally. See the README.

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
| Where it runs | Cloudflare Workers | Go container you run (the public host is retired) |
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

## Federation join

There is no fleet host. To register with the live hub, set:

| Env | Example |
|---|---|
| `WORLD_NAME` | `Rust Choir` |
| `WORLD_URL` | the public URL of the node you are running |
| `GRID_HUB_URL` | `https://grid-hub.skyphusion.org/rpc` |
| `GRID_HUB_TOKEN` | a hub token (never commit) |

CI on `main` pushes the image to GHCR. It does not roll a host.

## Play it

```sh
go run ./cmd/world --addr :8790 --world-name "Rust Choir" --data ./data
wscat -c ws://127.0.0.1:8790/ws
```

Score a local node against the live Dustfall world:

```sh
MUD_URL=ws://127.0.0.1:8790/ws \
DUSTFALL_URL=wss://dustfall.skyphusion.org/ws \
node /path/to/the-hollow-grid/smoke.mjs
```

## Load testing

LLM bots in the `mud-bots` repo can soak a world you run. The old fleet population is retired.
