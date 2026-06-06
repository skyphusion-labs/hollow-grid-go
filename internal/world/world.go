// Package world holds the local game state a Hollow Grid world owns: its room
// graph, and (later) mobs, items, inventory, and positions. Per the federation
// trust model, local state is the world's own business; only the canonical
// CharSheet (identity/progression) ever round-trips to the Grid.
package world

import "sort"

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

// CharVitalsPayload is emitted as char.vitals on room view and whenever vitals change.
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

// --- model ---

// Room is one node in the local room graph. Exits map a direction to a room id;
// an exit not listed does not exist (the no-silent-no-op rule).
type Room struct {
	ID    string
	Name  string
	Desc  string
	Exits map[string]string
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

// Player is a connected character. Identity/progression here is local until the
// federation client (Phase 3) makes it the canonical, Grid-owned CharSheet.
type Player struct {
	Name   string
	RoomID string
	HP     int
	MaxHP  int
	Level  int
	XP     int
	Gold   int
}

// NewPlayer spawns a fresh level-1 character at startRoom.
func NewPlayer(name, startRoom string) *Player {
	return &Player{Name: name, RoomID: startRoom, HP: 50, MaxHP: 50, Level: 1}
}

// Vitals renders the player as a char.vitals payload.
func (p *Player) Vitals() CharVitalsPayload {
	return CharVitalsPayload{
		HP: p.HP, MaxHP: p.MaxHP, Level: p.Level, XP: p.XP, Gold: p.Gold,
		Room: p.RoomID, Position: "standing",
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
// (worlds/*.jsonc, persistence) lands in Phase 1.
func New(name, url string) *World {
	w := &World{Name: name, URL: url, rooms: map[string]*Room{}}
	w.seed()
	return w
}

func (w *World) seed() {
	rooms := []*Room{
		{
			ID:    "grid-gate",
			Name:  "The Grid Gate",
			Desc:  "A dead terminal hums where the network once breathed. Cables trail into the dark like roots, and a single cursor blinks at nothing.",
			Exits: map[string]string{"north": "ash-road"},
		},
		{
			ID:    "ash-road",
			Name:  "Ash Road",
			Desc:  "Grey dunes swallow a cracked highway. The wind carries static and the smell of rust. The Gate glows faintly to the south.",
			Exits: map[string]string{"south": "grid-gate"},
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
