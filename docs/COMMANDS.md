# Command reference

The verbs a player (or agent) can send over `/ws`. Send one command per line.
Aliases are listed together. Anything a command produces on the structured
channel is noted as its `@event`.

## Moving & looking

| Command | Effect | Emits |
|---|---|---|
| `<direction>` (`north`/`south`/`east`/`west`/`up`/`down`) | move; an unlisted exit says so | `room.info`, `char.vitals`, `char.affects`, `room.actions` |
| `look` / `l` | re-show the room | the room scene |
| `look <mob>` | read a creature's description | — |
| `exits` | list the ways out | — |
| `consider` / `con` `<mob>` | size a creature up relative to you | — |
| `sense` / `actions` | read the room's contextual actions (the affordance layer) | `room.actions` |
| `recall` | fold back to the Cracked Nexus | the room scene |

## Self

| Command | Effect | Emits |
|---|---|---|
| `whoami` / `identity` | read your canonical sheet back | `char.identity` |
| `affects` | your current standing / afflictions | `char.affects` |
| `inventory` / `inv` / `i` | what you carry | — |
| `equipment` / `eq` | what you wear | `char.equipment` |
| `wield` / `wear` / `equip` `<item>` | put gear into its slot | `char.equipment`, `char.vitals` |
| `remove` / `unwield` `<item>` | take gear off | `char.equipment` |
| `title <text>` | set the epithet shown in `who` | — |
| `who` | who is online (with titles) | — |
| `rest` | settle and regenerate over the heartbeat | `char.vitals` |
| `stand` / `wake` | get to your feet | `char.vitals` |
| `sleep` | sleep, and the Grid shows you a dream | `char.dream`, `char.vitals` |

## The living world

| Command | Effect | Emits |
|---|---|---|
| `world` / `weather` | the time of day and the sky | `world.state` |

(The world also turns on its own: `world.state` is emitted on the heartbeat.)

## Combat

| Command | Effect | Emits |
|---|---|---|
| `attack` / `kill` / `k` `<mob>` | start a fight; it resolves over ticks | `combat.start`, then `combat.round`…, `combat.end`; `char.vitals`; `char.died` on death |

## Race ability

| Command | Effect | Emits |
|---|---|---|
| `ability` / `trait` | fire your race's signature ability (cooldown-gated) | `char.vitals` |
| or the race's own verb | `requisition` (human), `vanish` (elf), `commune` (revenant), `regenerate` (ghoul), `overclock` (chromed), `forage` (dustkin), `fabricate` (vatborn) | — |

## The moral arc (Cinder Front)

At the **Scrap Market** recruiter:

| Command | Effect | Emits |
|---|---|---|
| `join` | swear to the Cinder Front → `faction:front`; a **hunted** race is branded **ash-sworn** (the kapo). It does not wash off. | `char.affects`, `char.vitals`, `room.actions` |
| `defy` | spit on the offer; +standing | `char.affects`, `char.vitals`, `room.actions` |

At the **honest market**: `sell` / `trade` — refused if you are a Front member.

## The wastes & the Refugee Waystation

| Command | Effect |
|---|---|
| `talk` | at the waystation, the free folk read your standing (the unaligned are told to pick a side; the just are welcomed; the ash-sworn are met with silence) |
| `treat` / `medic` | at the waystation, the medic mends the free/unaligned and turns their back on a Front collaborator; nowhere else |

## Rescue

| Command | Effect | Emits |
|---|---|---|
| `free` / `rescue` | free the captive in the room — but only once the warden is down. A real virtuous act (+morality), named, unfarmable. | `grid.rescued`, `char.affects` |

## The tinker (workshop)

| Command | Effect | Emits |
|---|---|---|
| `list` / `wares` | the tinker's stock and prices | — |
| `buy <item>` | buy gear for gold; it lands in your pack | `char.vitals` |

## Meta

| Command | Effect |
|---|---|
| `help` / `h` / `?` | a short reminder |
| `quit` / `q` | leave (the Grid keeps what you did) |

> Many world-affecting actions are also surfaced as structured `room.actions` with
> a `kind` and, for moral ones, a `valence` — so an agent can read what it may do,
> and what it would mean, without scraping prose. See `sense`.
