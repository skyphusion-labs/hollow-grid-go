# Command reference

The verbs a player (or agent) can send over `/ws`. One command per line; aliases
are grouped. Anything a command produces on the structured channel is noted as
its `@event`. The authoritative dispatch is `internal/transport/conn.go`; this
doc mirrors it for operators and port authors.

For the full TS reference list (including verbs not yet ported), see
`the-hollow-grid/docs/architecture.md`.

## Moving and looking

| Command | Effect | Emits |
|---|---|---|
| `<direction>` (`north`/`south`/`east`/`west`/`up`/`down`) | move; an unlisted exit says so | `room.info`, `char.vitals`, `char.affects`, `room.actions` |
| `look` / `l` | re-show the room | room scene + room events |
| `look <mob>` / `look <player>` | inspect a creature or another player in the room | (none) |
| `exits` | list the ways out | (none) |
| `consider` / `con` `<mob>` | size a creature up relative to you | (none) |
| `sense` / `actions` | read the room's contextual actions (the affordance layer) | `room.actions` |
| `recall` | fold back to the Cracked Nexus | room scene |

## Self

| Command | Effect | Emits |
|---|---|---|
| `whoami` / `identity` | read your canonical sheet back | `char.identity` |
| `affects` | your current standing / afflictions | `char.affects` |
| `inventory` / `inv` / `i` | what you carry | (none) |
| `equipment` / `eq` | what you wear | `char.equipment` |
| `wield` / `wear` / `equip` `<item>` | put gear into its slot | `char.equipment`, `char.vitals` |
| `remove` / `unwield` `<item>` | take gear off | `char.equipment` |
| `title <text>` | set the epithet shown in `who` | (none) |
| `who` | who is online (federation-wide when hub is bound) | (none) |
| `rest` | settle and regenerate over the heartbeat | `char.vitals` |
| `stand` / `wake` | get to your feet | `char.vitals` |
| `sleep` | sleep; the Grid shows you a dream | `char.dream`, `char.vitals` |
| `reckoning` / `conscience` / `record` | your moral self-model (deed ledger) | `char.reckoning` |

## The living world

| Command | Effect | Emits |
|---|---|---|
| `world` / `weather` | the time of day and the sky | `world.state` |

(The world also turns on its own: `world.state` is emitted on the heartbeat.)

## Combat

| Command | Effect | Emits |
|---|---|---|
| `attack` / `kill` / `k` `<mob>` | start a fight; resolves over ticks | `combat.start`, `combat.round`…, `combat.end`; `char.vitals`; `char.died` on death |

## Race ability

| Command | Effect | Emits |
|---|---|---|
| `ability` / `trait` | fire your race's signature ability (cooldown-gated) | `char.vitals` |
| or the race verb | `requisition` (human), `vanish` (elf), `commune` (revenant), `regenerate` (ghoul), `overclock` (chromed), `forage` (dustkin), `fabricate` (vatborn) | (none) |

## Items and economy

| Command | Effect | Emits |
|---|---|---|
| `get` / `take` `<item>` | pick up ground loot | `char.vitals` |
| `drop` `<item>` | drop from inventory | (none) |
| `give <item> <player>` | hand gear to another player in the room | `char.vitals` (both sides when applicable) |
| `list` / `wares` | tinker's stock (workshop) | (none) |
| `buy <item>` | buy gear for gold | `char.vitals` |
| `sell` / `trade` `<item>` | sell salvage at the market (honest coin; refused for Front members) | `char.vitals` |
| `steal` | steal from the market vendor (quick gold, corrupting) | `char.vitals`, `char.affects` |

Bare `sell` prompts `Sell what?`; agents should send the item name.

## Comms (multiplayer)

| Command | Effect | Emits |
|---|---|---|
| `say <text>` | speak to the room | (none) |
| `tell <player> <text>` | private message | `comm.tell` (recipient) |
| `reply <text>` | reply to last tell | `comm.tell` |
| `yell` / `shout` `<text>` | yell to the room (reliable broadcast) | `comm.yell` |
| `emote` / `pose` `<action>` | emote to the room | (none) |
| `gridcast` / `gc` `<text>` | cross-world chat (when hub is bound) | `grid.gridcast` |

## The moral arc (Cinder Front)

At the **Scrap Market** recruiter:

| Command | Effect | Emits |
|---|---|---|
| `join` | swear to the Cinder Front → `faction:front`; a **hunted** race is branded **ash-sworn** (the kapo) | `char.affects`, `char.vitals`, `room.actions` |
| `defend` | stand with the refugees → `faction:ally` | `char.affects`, `char.vitals`, `room.actions` |
| `talk` | room-specific NPC prose (market, tavern, waystation, …) | (none) |

At the **Ashmonger's dais** (stronghold endgame):

| Command | Effect |
|---|---|
| `defy` | defect from the Front to the free folk |

At the **tavern**:

| Command | Effect |
|---|---|
| `buy dust` | buy dust (heals, corrupts, addicts) |
| `carouse` | spend coin and conscience in the back |
| `resist` | resist the tavern's vices (+standing) |

At the **honest market**: `sell` is refused if you are a Front member; `steal` always corrupts.

At the **Refugee Waystation**:

| Command | Effect |
|---|---|
| `talk` | the free folk read your standing |
| `treat` / `medic` | the medic mends allies (gated by collective tide) |
| `witness` / `remember` / `mourn` `[name]` | read or hold vigil for the fallen | `grid.fallen`, `grid.remembrance` |

## Rescue and aid

| Command | Effect | Emits |
|---|---|---|
| `free` / `rescue` / … | free the captive (holding pit, cells) once the guard is down | `grid.rescued`, `char.affects` |
| `shelter` / `guide` | answer the transit-hub distress call | `char.affects` |
| `mend <player>` | heal another player at a cost to your HP | `char.vitals` |
| `cache` / `stash` `<gold>` | leave aid at a node for the next traveler | (none) |
| `gather` | take what a stranger left at a node | `char.vitals` |
| `forgive` | forgive a marked soul (waystation context) | `char.affects` |
| `inscribe` / `carve` / `leave` `<text>` | leave a message in the Grid ledger | `grid.trace` |

## The Grid (federation)

When `GRID_HUB_URL` is set (production Rust Choir), these call the shared hub.
On hub failure the world keeps running locally.

| Command | Effect | Emits |
|---|---|---|
| `worlds` | list registered worlds | `grid.worlds` |
| `travel <world>` | checkpoint and hand off to another world's URL (closes session) | `grid.travel` |
| `war` / `tide` | read the global faction tide | `world.war` |
| `ping` / `listen` / `tune` | node traces / cross-world ledger echoes | `grid.echo`, `grid.federation` |
| `saved` / `rescued` / `roll` | the cross-world rescued roll | `grid.rescued_roll` |
| `wall` / `announce` `<text>` | keeper broadcast (names in `ADMINS`) | `server.announce` |
| `gridstats` | ledger composition (keeper) | (none) |
| `gridprune` | purge ambient ledger noise (keeper) | (none) |

## Meta

| Command | Effect |
|---|---|
| `help` / `h` / `?` | a short reminder |
| `quit` / `q` | leave (the Grid keeps what you did) |

## `room.actions` (for agents)

Many world-affecting verbs are also surfaced as structured `room.actions` with a
`kind` (`move`/`moral`/`trade`/…) and, for moral ones, a `valence`
(`virtuous`/`corrupt`/`grave`). Prefer choosing an enumerated verb over guessing.
Run `sense` or read the `room.actions` event after every room change.

**Known quirk:** moral choices resolved in a session are tracked in session-local
`resolved` state; on reconnect, join/defend may reappear in `room.actions` even
if faction is already set. The server still enforces one-time outcomes.
