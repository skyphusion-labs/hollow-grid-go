# Rust Choir: the third world on the Grid

**Rust Choir** is the Go fleet node's identity in the Hollow Grid federation. It
speaks the same wire protocol and passes the same conformance suite as the
reference world, but it is not a clone of hollow.skyphusion.org or Dustfall.

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
| Default listen flavor | Generic dead-network voices | Same transmission pool, but the prose framing leans **archivist** ("tune the dead frequencies") |
| Fleet home | Cloudflare Workers | **Hetzner fleet** container (`:8790`, distroless image) |

Mechanically the races, Cinder Front arc, holding-pit rescue, and `@event`
vocabulary are identical. Differentiation is **place and voice**, not protocol.

## Design rules (for agents and humans)

1. **Conformance first.** Any creative content must graft from rooms the smoke
   suite does not pin (the tunnels east exit is the current graft point).
2. **Moral weight stays data.** Every meaningful choice emits `room.actions`
   with `valence`; rescues emit `grid.rescued`; oaths land in the trace ledger.
3. **Federation is additive.** The world runs standalone on `FileStore`; the
   Grid Hub client (`internal/grid`) mirrors traces when reachable.

## Federation join (fleet)

Production target: `wss://rustchoir.skyphusion.org/ws` (or similar), container on
the fleet, registering with the shared grid-hub alongside hollow and Dustfall.

**Open seam:** grid-hub today exposes RPC only via Cloudflare service bindings.
Fleet join needs either an HTTP ingress on the hub or a thin CF Worker relay.
Track in `docs/PLAN.md` Phase 3.

## Play it

```sh
go run ./cmd/world --world-name "Rust Choir" --world-url "wss://rustchoir.skyphusion.org/ws"
# east from the tunnels -> the Grid Gate tract
```
