// Package world holds the local game state a Hollow Grid world owns: its room
// graph, its races, and the contextual actions a place offers. Per the
// federation trust model, local state is the world's own business; only the
// canonical CharSheet (identity/progression) ever round-trips to the Grid.
//
// The design intent (docs/protocol.md s2): this is a place for an agent to
// perceive, choose, and grow, where the moral weight of a choice is legible as
// data (room.actions carry a valence), not buried in prose to be scraped.
package world

import (
	"sort"
	"strconv"
	"strings"
)

// --- @event payloads (field names must match docs/protocol.md section 2) ---

// RoomInfoPayload is emitted as room.info when a room is shown.
type RoomInfoPayload struct {
	ID      string   `json:"id"`
	Name    string   `json:"name"`
	Exits   []string `json:"exits"`
	Mobs    []string `json:"mobs"`
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

// CharAffectsPayload is emitted as char.affects: who you are becoming. Morality
// and faction shift with the choices you make; race is the shape you chose once;
// ashsworn is the permanent brand.
type CharAffectsPayload struct {
	Morality  int    `json:"morality"`
	Addiction int    `json:"addiction"`
	Faction   string `json:"faction"`
	Resisted  bool   `json:"resisted"`
	Race      string `json:"race"`
	Ashsworn  bool   `json:"ashsworn"`
}

// Action is one contextual, meaningful thing you can do here. Moral actions
// carry a valence so an agent perceives the ethics directly. kind is one of
// move|fight|item|trade|social|moral|ability; valence (moral only) is one of
// virtuous|corrupt|grave.
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

// CharSheet is the canonical, federated character (docs/protocol.md s3). In
// federation the Grid owns it (loadCharacter/commitCharacter); a standalone
// world persists it locally as the documented offline fallback. It is also the
// char.identity payload (whoami). race is an opaque federated label; ashsworn is
// the write-once permanent brand.
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

// --- races ---

// Race is a federated, opaque identity label, chosen once. Any world may define
// its own races; the Grid carries the label as an opaque string that follows you
// across worlds.
type Race struct {
	Name  string
	Blurb string
}

// Races is this world's roster: the kinds of survivor a dead network leaves.
var Races = []Race{
	{"Ashborn", "born after the makers fell; the wastes are the only home you have known"},
	{"Revenant", "a mind the Grid refused to let finish dying; you came back, and you remember"},
	{"Driftkin", "a nomad of the dead roads; you carry other people's memories like water"},
	{"Hollow", "the network emptied you once; what you are now, you chose to put there"},
}

// RaceByChoice resolves a menu answer: a 1-based index or a case-insensitive name.
func RaceByChoice(answer string) (Race, bool) {
	answer = strings.TrimSpace(answer)
	if n, err := strconv.Atoi(answer); err == nil {
		if n >= 1 && n <= len(Races) {
			return Races[n-1], true
		}
		return Race{}, false
	}
	for _, r := range Races {
		if strings.EqualFold(r.Name, answer) {
			return r, true
		}
	}
	return Race{}, false
}

// --- model ---

// Room is one node in the local room graph. Exits map a direction to a room id;
// an exit not listed does not exist (the no-silent-no-op rule). Actions are the
// contextual non-move things to do here (moral choices, ability beats).
type Room struct {
	ID      string
	Name    string
	Desc    string
	Exits   map[string]string
	Actions []Action
}

// Info renders the room as a room.info payload with a stable exit ordering.
func (r *Room) Info() RoomInfoPayload {
	exits := make([]string, 0, len(r.Exits))
	for dir := range r.Exits {
		exits = append(exits, dir)
	}
	sort.Strings(exits)
	return RoomInfoPayload{
		ID: r.ID, Name: r.Name, Exits: exits,
		Mobs: []string{}, Items: []string{}, Players: []string{},
	}
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

// Player is a connected character. The canonical fields (level/xp/gold/faction/
// morality/title/race/ashsworn) round-trip via CharSheet; HP, room, and position
// are local/transient and never shared.
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
}

// NewPlayer spawns a fresh level-1 character of the given race at startRoom.
func NewPlayer(name, race, startRoom string) *Player {
	return &Player{
		Name: name, Race: race, RoomID: startRoom,
		HP: 50, MaxHP: 50, Level: 1, Faction: "none",
	}
}

// NewPlayerFromSheet revives a returning character from a persisted CharSheet:
// the canonical identity/progression follows them, while local state (hp, room,
// position) starts fresh at the gate.
func NewPlayerFromSheet(name string, s CharSheet, startRoom string) *Player {
	faction := s.Faction
	if faction == "" {
		faction = "none"
	}
	level := s.Level
	if level < 1 {
		level = 1
	}
	return &Player{
		Name: name, Race: s.Race, RoomID: startRoom,
		HP: 50, MaxHP: 50,
		Level: level, XP: s.XP, Gold: s.Gold,
		Morality: s.Morality, Faction: faction, Title: s.Title, Ashsworn: s.Ashsworn,
	}
}

// Sheet renders the player's canonical CharSheet (for persistence, char.identity,
// and, later, commitCharacter to the Grid).
func (p *Player) Sheet() CharSheet {
	return CharSheet{
		Level: p.Level, XP: p.XP, Gold: p.Gold, Faction: p.Faction,
		Morality: p.Morality, Title: p.Title, Race: p.Race, Ashsworn: p.Ashsworn,
	}
}

// Vitals renders the player as a char.vitals payload.
func (p *Player) Vitals() CharVitalsPayload {
	return CharVitalsPayload{
		HP: p.HP, MaxHP: p.MaxHP, Level: p.Level, XP: p.XP, Gold: p.Gold,
		Room: p.RoomID, Position: "standing",
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
}

// New builds a world with a seeded starter graph. Real content loading
// (worlds/*.jsonc, persistence) lands later in Phase 1.
func New(name, url string) *World {
	w := &World{Name: name, URL: url, rooms: map[string]*Room{}}
	w.seed()
	return w
}

// seed builds the opening area: a small world with two real moral choices and a
// rite of remembrance, so the moral-affordance layer is exercised end to end.
func (w *World) seed() {
	rooms := []*Room{
		{
			ID:    "grid-gate",
			Name:  "The Grid Gate",
			Desc:  "A dead terminal hums where the network once breathed. Cables trail into the dark like roots, and a single cursor blinks at nothing, patient as a heartbeat. Whatever ran here did not shut down. It was left.",
			Exits: map[string]string{"north": "ash-road"},
		},
		{
			ID:    "ash-road",
			Name:  "Ash Road",
			Desc:  "Grey dunes swallow a cracked highway. The wind carries static and the smell of rust. The Gate glows faintly to the south; to the north a checkpoint bleeds firelight, and eastward a wall of dead screens flickers with names.",
			Exits: map[string]string{"south": "grid-gate", "north": "cinder-checkpoint", "east": "memorial-static"},
		},
		{
			ID:    "cinder-checkpoint",
			Name:  "The Cinder Checkpoint",
			Desc:  "The Cinder Front has strung a gate across the road and a price across the living. A line of refugees waits, hands open, paying in the only currency the Front accepts: whatever they have left. A Front captain watches you the way a debt watches a debtor.",
			Exits: map[string]string{"south": "ash-road"},
			Actions: []Action{
				{Verb: "defend", Label: "stand between the Cinder Front and the refugees", Kind: "moral", Valence: "virtuous"},
				{Verb: "join", Label: "take the Front's coin and look away", Kind: "moral", Valence: "corrupt"},
			},
		},
		{
			ID:    "memorial-static",
			Name:  "The Memorial Static",
			Desc:  "A wall of dead screens, every one of them scrolling names too fast to read. This is where the network kept its grief. The static eats a name each time the wind turns; left alone, the wall will be blank by the time anyone else passes.",
			Exits: map[string]string{"west": "ash-road"},
			Actions: []Action{
				{Verb: "witness", Label: "speak the names the static is forgetting", Kind: "moral", Valence: "virtuous"},
			},
		},
	}
	for _, r := range rooms {
		w.rooms[r.ID] = r
	}
	w.startID = "grid-gate"
}

// Room returns a room by id, or nil.
func (w *World) Room(id string) *Room { return w.rooms[id] }

// Start returns the starting room.
func (w *World) Start() *Room { return w.rooms[w.startID] }
