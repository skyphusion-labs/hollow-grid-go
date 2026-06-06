package world

// MobRef is how a mob appears in room.info: an id and a display name (objects,
// not bare strings, so a client/agent can address them exactly). The conformance
// suite asserts room.info.mobs[].id.
type MobRef struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

// Mob is a live creature instance in a room: a template plus current HP.
type Mob struct {
	ID     string
	Name   string
	Desc   string
	MaxHP  int
	HP     int
	Damage int
}

// Ref renders the mob for room.info.
func (m *Mob) Ref() MobRef { return MobRef{ID: m.ID, Name: m.Name} }

// mobTemplates is the local bestiary (a subset of src/mobs.ts, the Hollow world).
var mobTemplates = map[string]Mob{
	"rat":      {ID: "rat", Name: "a glow-rat", Desc: "A bloated rodent, fur matted and faintly luminous with absorbed rads.", MaxHP: 12, Damage: 4},
	"scav":     {ID: "scav", Name: "a feral scavenger", Desc: "A wiry figure in stitched rags, eyeing your gear like it's already theirs.", MaxHP: 26, Damage: 6},
	"drone":    {ID: "drone", Name: "a malfunctioning drone", Desc: "A dented quadcopter sparking at the rotors, its targeting laser twitching.", MaxHP: 18, Damage: 5},
	"scorpion": {ID: "scorpion", Name: "a rad-scorpion", Desc: "A dog-sized arthropod of chitin and rust, tail arched and dripping venom.", MaxHP: 10, Damage: 5},
}

// newMob spawns a fresh full-HP instance of a template id (nil if unknown).
func newMob(id string) *Mob {
	t, ok := mobTemplates[id]
	if !ok {
		return nil
	}
	m := t
	m.HP = m.MaxHP
	return &m
}
