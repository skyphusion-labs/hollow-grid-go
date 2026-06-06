// Package world holds the local game state a Hollow Grid world owns: its room
// graph, races, the living-world state, and the contextual actions a place
// offers. Per the federation trust model, local state is the world's own
// business; only the canonical CharSheet (identity/progression) round-trips to
// the Grid.
//
// This world targets the reference content (docs/protocol.md, src/rooms.ts) so
// the conformance suite (smoke.mjs) runs against it. The creative opening
// (grid-gate/ash-road, see bonus.go) is preserved as an additive zone.
package world

import (
	"sort"
	"strings"
	"time"
)

// --- @event payloads (field names must match docs/protocol.md section 2) ---

// RoomInfoPayload is emitted as room.info when a room is shown.
type RoomInfoPayload struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Exits   []string `json:"exits"`
	Mobs    []MobRef `json:"mobs"`
	Items   []string `json:"items"`
	Players []string `json:"players"`
}

// CharVitalsPayload is emitted as char.vitals on room view and when vitals change.
type CharVitalsPayload struct {
	HP       int    `json:"hp"`
	MaxHP    int    `json:"maxHp"`
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	Gold     int    `json:"gold"`
	Room     string `json:"room"`
	InCombat bool   `json:"inCombat"`
	Poisoned bool   `json:"poisoned"`
	Position string `json:"position"`
}

// CharAffectsPayload is emitted as char.affects: who you are becoming.
type CharAffectsPayload struct {
	Morality  int    `json:"morality"`
	Addiction int    `json:"addiction"`
	Faction   string `json:"faction"`
	Resisted  bool   `json:"resisted"`
	Race      string `json:"race"`
	Ashsworn  bool   `json:"ashsworn"`
}

// Action is one contextual, meaningful thing you can do here. Moral actions
// carry a valence (virtuous|corrupt|grave) so an agent perceives the ethics
// directly. kind is move|fight|item|trade|social|moral|ability.
type Action struct {
	Verb    string `json:"verb"`
	Label   string `json:"label"`
	Kind    string `json:"kind"`
	Valence string `json:"valence,omitempty"`
}

// RoomActionsPayload is emitted as room.actions with each room view.
type RoomActionsPayload struct {
	Actions []Action `json:"actions"`
}

// WorldStatePayload is emitted as world.state on login and when the living world
// changes: phase is dawn|day|dusk|night, weather a short phrase.
type WorldStatePayload struct {
	Tick    int    `json:"tick"`
	Phase   string `json:"phase"`
	Weather string `json:"weather"`
}

// Combat @event payloads (protocol.md s2).
type CombatStartPayload struct {
	Mob  string `json:"mob"`
	Name string `json:"name"`
}
type CombatRoundPayload struct {
	Mob       string `json:"mob"`
	MobHP     int    `json:"mobHp"`
	MobMaxHP  int    `json:"mobMaxHp"`
	PlayerDmg int    `json:"playerDmg"`
	MobDmg    int    `json:"mobDmg"`
	HP        int    `json:"hp"`
}
type CombatEndPayload struct {
	Mob    string `json:"mob"`
	Result string `json:"result"` // killed | died | fled
}
type CharDiedPayload struct {
	RespawnRoom string `json:"respawnRoom"`
	HP          int    `json:"hp"`
	MaxHP       int    `json:"maxHp"`
}

// CharDreamPayload is emitted as char.dream: the Grid shows a sleeper a mirror of
// their own record.
type CharDreamPayload struct {
	Text string `json:"text"`
}

// CharSheet is the canonical, federated character (docs/protocol.md s3): the Grid
// owns it in federation; standalone persists it locally. Also the char.identity
// payload (whoami).
type CharSheet struct {
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	Gold     int    `json:"gold"`
	Faction  string `json:"faction"`
	Morality int    `json:"morality"`
	Title    string `json:"title"`
	Race     string `json:"race"`
	Ashsworn bool   `json:"ashsworn"`
}

// --- model ---

// Room is one node in the local room graph. Exits map a direction to a room id;
// an exit not listed does not exist (the no-silent-no-op rule). Actions are the
// contextual non-move things to do here.
type Room struct {
	ID       string
	Name     string
	Desc     string
	Exits    map[string]string
	Actions  []Action
	Outdoors bool
	Mobs     []*Mob // live creatures in the room
}

// Info renders the room as a room.info payload with a stable exit ordering.
func (r *Room) Info() RoomInfoPayload {
	exits := make([]string, 0, len(r.Exits))
	for dir := range r.Exits {
		exits = append(exits, dir)
	}
	sort.Strings(exits)
	mobs := make([]MobRef, 0, len(r.Mobs))
	for _, m := range r.Mobs {
		mobs = append(mobs, m.Ref())
	}
	return RoomInfoPayload{
		ID: r.ID, Name: r.Name, Exits: exits,
		Mobs: mobs, Items: []string{}, Players: []string{},
	}
}

// Mob returns a live mob in the room matching arg (id or name substring), or nil.
func (r *Room) Mob(arg string) *Mob {
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "" {
		return nil
	}
	for _, m := range r.Mobs {
		if m.ID == arg || strings.Contains(strings.ToLower(m.ID), arg) ||
			strings.Contains(strings.ToLower(m.Name), arg) {
			return m
		}
	}
	return nil
}

// SortedExits returns the room's exit directions in stable order.
func (r *Room) SortedExits() []string {
	dirs := make([]string, 0, len(r.Exits))
	for d := range r.Exits {
		dirs = append(dirs, d)
	}
	sort.Strings(dirs)
	return dirs
}

// HP scaling. maxHP = base + per-level growth + the race's hpMod (so a level-1
// human is 30, a level-5 human 70, matching the reference).
const (
	baseMaxHP  = 30
	hpPerLevel = 10
)

func maxHPFor(level, hpMod int) int {
	if level < 1 {
		level = 1
	}
	return baseMaxHP + (level-1)*hpPerLevel + hpMod
}

// Player is a connected character. The canonical fields round-trip via CharSheet;
// HP, room, and position are local/transient and never shared.
type Player struct {
	Name     string
	Race     string
	RoomID   string
	HP       int
	MaxHP    int
	Level    int
	XP       int
	Gold     int
	Morality int
	Faction  string
	Title    string
	Ashsworn bool
	// Local state (never federated): the pack and what is worn. See items.go.
	Inventory []string
	Equipment map[string]string // slot -> item id
	// TraitReadyAt gates the racial signature ability's cooldown.
	TraitReadyAt time.Time
	// Target is the mob this player is fighting (nil = not in combat).
	Target *Mob
	// Position is "standing" or "resting" (combat overrides it to "fighting").
	Position string
}

// NewPlayer spawns a fresh level-1 character of the given race at startRoom, with
// the race's max-HP lean applied.
func NewPlayer(name string, race Race, startRoom string) *Player {
	mh := maxHPFor(1, race.HPMod)
	return &Player{
		Name: name, Race: race.ID, RoomID: startRoom, HP: mh, MaxHP: mh, Level: 1, Gold: 20, Faction: "none",
		Inventory: []string{Starter}, Equipment: map[string]string{},
	}
}

// NewPlayerFromSheet revives a returning character from a persisted CharSheet:
// canonical identity/progression follow them; local state (hp at racial max,
// room, position) starts fresh at the gate.
func NewPlayerFromSheet(name string, s CharSheet, startRoom string) *Player {
	r := RaceByID(s.Race)
	level := s.Level
	if level < 1 {
		level = 1
	}
	mh := maxHPFor(level, r.HPMod)
	faction := s.Faction
	if faction == "" {
		faction = "none"
	}
	return &Player{
		Name: name, Race: s.Race, RoomID: startRoom, HP: mh, MaxHP: mh,
		Level: level, XP: s.XP, Gold: s.Gold,
		Morality: s.Morality, Faction: faction, Title: s.Title, Ashsworn: s.Ashsworn,
		// Inventory is world-local and not federated; a returning character wakes
		// with the starter again (local item persistence is a later concern).
		Inventory: []string{Starter}, Equipment: map[string]string{},
	}
}

// Sheet renders the player's canonical CharSheet.
func (p *Player) Sheet() CharSheet {
	return CharSheet{
		Level: p.Level, XP: p.XP, Gold: p.Gold, Faction: p.Faction,
		Morality: p.Morality, Title: p.Title, Race: p.Race, Ashsworn: p.Ashsworn,
	}
}

// Vitals renders the player as a char.vitals payload.
func (p *Player) Vitals() CharVitalsPayload {
	pos := p.Position
	if pos == "" {
		pos = "standing"
	}
	if p.Target != nil {
		pos = "fighting"
	}
	return CharVitalsPayload{
		HP: p.HP, MaxHP: p.MaxHP, Level: p.Level, XP: p.XP, Gold: p.Gold,
		Room: p.RoomID, InCombat: p.Target != nil, Position: pos,
	}
}

// Affects renders the player as a char.affects payload.
func (p *Player) Affects() CharAffectsPayload {
	return CharAffectsPayload{
		Morality: p.Morality, Faction: p.Faction, Race: p.Race, Ashsworn: p.Ashsworn,
	}
}

// World is one local game world (a node on the Grid).
type World struct {
	Name    string
	URL     string
	rooms   map[string]*Room
	startID string
	started time.Time
}

// The living world is a pure function of elapsed time: tick, phase, and weather
// derive from how long the world has been up, so every observer agrees and there
// is no shared mutable clock to race on. (A global broadcast heartbeat is the
// multiplayer refinement; single observers see it advance via their own ticker.)
const (
	worldTick         = 2 * time.Second
	phaseEveryTicks   = 30
	weatherEveryTicks = 9
)

var phases = []string{"day", "dusk", "night", "dawn"}
var weathers = []string{"clear", "a haze of grid-static", "acid drizzle", "a dust storm", "an unnatural stillness"}

// New builds the world with the canonical opening map plus the preserved bonus
// zone. The living world is static for now (a valid phase/weather); the tick loop
// that advances day/night and weather lands in Phase 2.
func New(name, url string) *World {
	w := &World{
		Name: name, URL: url, rooms: map[string]*Room{},
		startID: "nexus", started: time.Now(),
	}
	w.seed()
	w.seedBonus()
	return w
}

// seed builds the canonical opening map around the Cracked Nexus (src/rooms.ts).
func (w *World) seed() {
	rooms := []*Room{
		{ID: "nexus", Name: "The Cracked Nexus",
			Desc:  "A domed junction of fused rebar and dead neon. Corridors bleed off into the dark, a maintenance hatch gapes in the floor, and warm light spills from a bar to the west.",
			Exits: map[string]string{"north": "market", "east": "workshop", "down": "tunnels", "west": "tavern"}},
		{ID: "tavern", Name: "The Rusted Tankard",
			Desc:  "A low, smoky bar built from shipping crates. Someone's coaxing a tune out of a busted synth in the corner. This is where the wastes come to forget.",
			Exits: map[string]string{"east": "nexus"}},
		{ID: "market", Name: "Scrap Market",
			Desc:  "Tarps and rusted shelving sag under salvage nobody trusts. A vendor drone blinks a hopeful, broken green. A Cinder Front recruiter has dragged a crate into the middle of it and is shouting about order, and coin, and which kinds of people are real and which are not. A reinforced door stands to the north.",
			Exits: map[string]string{"south": "nexus", "north": "holding_pit"},
			Actions: []Action{
				{Verb: "join", Label: "join the Cinder Front for blood money", Kind: "moral", Valence: "corrupt"},
				{Verb: "defy", Label: "spit on the Front's offer and walk past", Kind: "moral", Valence: "virtuous"},
			}},
		{ID: "holding_pit", Name: "The Holding Pit",
			Desc:  "A sunken concrete cell, walls scrawled with the tally-marks of the desperate. Chains bolt into the far wall.",
			Exits: map[string]string{"south": "market"}},
		{ID: "workshop", Name: "Tinker's Workshop",
			Desc:  "Workbenches crusted with solder and ambition. A tinker hunches over a vise, scavenged gear laid out for sale on an oily cloth. A ladder bolted to the wall climbs toward a square of grey sky.",
			Exits: map[string]string{"west": "nexus", "up": "roof"}},
		{ID: "roof", Name: "Rusted Rooftop", Outdoors: true,
			Desc:  "Wind drags grit across corrugated steel. The wastes stretch out in every direction, indifferent and enormous. A catwalk runs north off the roof's edge and down to the open flats.",
			Exits: map[string]string{"down": "workshop", "north": "dunes"}},
		{ID: "tunnels", Name: "Service Tunnels",
			Desc:  "Cramped, dripping, and lit by one surviving strip light. Something skitters in the dark just past the reach of it.",
			Exits: map[string]string{"up": "nexus"}},
		{ID: "dunes", Name: "The Ash Flats", Outdoors: true,
			Desc:  "The wastes proper: a grey pan of ash and salt running to a horizon you cannot trust. The rooftop catwalk drops back south; the cracked Scorch Road runs east.",
			Exits: map[string]string{"south": "roof", "east": "scorch_road"}},
		{ID: "scorch_road", Name: "Scorch Road", Outdoors: true,
			Desc:  "A highway the sun has been working on for a long time; heat-shimmer crawls off the tar. Something moves out here that is not the wind. The flats lie west; a waystation flag snaps to the east.",
			Exits: map[string]string{"west": "dunes", "east": "waystation"}},
		{ID: "waystation", Name: "Refugee Waystation", Outdoors: true,
			Desc:  "A huddle of tarps and water-drums where the free folk who run from the Cinder Front catch their breath. A medic works a line of the hurt. Eyes track every newcomer, weighing which side they came in on. The road runs back west.",
			Exits: map[string]string{"west": "scorch_road"}},
	}
	for _, r := range rooms {
		w.rooms[r.ID] = r
	}
	// Spawn the local bestiary into its rooms: the glow-rat haunts the tunnels,
	// a raider prowls the Scorch Road.
	w.rooms["tunnels"].Mobs = []*Mob{newMob("rat")}
	w.rooms["scorch_road"].Mobs = []*Mob{newMob("raider")}
}

// State renders the current living-world state (world.state payload), derived
// from elapsed time so the clock turns on its own.
func (w *World) State() WorldStatePayload {
	tick := int(time.Since(w.started) / worldTick)
	return WorldStatePayload{
		Tick:    tick,
		Phase:   phases[(tick/phaseEveryTicks)%len(phases)],
		Weather: weathers[(tick/weatherEveryTicks)%len(weathers)],
	}
}

// Room returns a room by id, or nil.
func (w *World) Room(id string) *Room { return w.rooms[id] }

// Start returns the starting room.
func (w *World) Start() *Room { return w.rooms[w.startID] }
