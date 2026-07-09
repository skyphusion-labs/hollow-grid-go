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

// mobTemplate is static bestiary data (a subset of src/mobs.ts).
type mobTemplate struct {
	ID        string
	Name      string
	Desc      string
	MaxHP     int
	Damage    int
	Room      string // spawn room (one instance per template id)
	RespawnMs int    // 0 = does not respawn
}

// mobCatalog is the local bestiary. Each key spawns exactly one live instance
// into Room when the world boots; slain mobs respawn after RespawnMs.
var mobCatalog = map[string]mobTemplate{
	"rat":       {ID: "rat", Name: "a glow-rat", Desc: "A bloated rodent, fur matted and faintly luminous with absorbed rads.", MaxHP: 12, Damage: 4, Room: "tunnels", RespawnMs: 20_000},
	"scav":      {ID: "scav", Name: "a feral scavenger", Desc: "A wiry figure in stitched rags, eyeing your gear like it's already theirs.", MaxHP: 26, Damage: 6},
	"drone":     {ID: "drone", Name: "a malfunctioning drone", Desc: "A dented quadcopter sparking at the rotors, its targeting laser twitching.", MaxHP: 18, Damage: 5},
	"scorpion":  {ID: "scorpion", Name: "a rad-scorpion", Desc: "A dog-sized arthropod of chitin and rust, tail arched and dripping venom.", MaxHP: 10, Damage: 5},
	"raider":    {ID: "raider", Name: "a wastes raider", Desc: "A scarred figure wrapped in sun-bleached rags and scavenged plate, hefting a length of rebar and grinning at the easy mark you make.", MaxHP: 22, Damage: 6, Room: "scorch_road", RespawnMs: 40_000},
	"warden":    {ID: "warden", Name: "the warden", Desc: "A chrome-masked jailer, broad as a doorway, the keys to the holding-pit cage hanging from their belt.", MaxHP: 18, Damage: 5, Room: "holding_pit", RespawnMs: 60_000},
	"leech":     {ID: "leech", Name: "a data-leech", Desc: "A pale, boneless thing clamped to a live rack, swollen with stolen current. It turns toward your warmth.", MaxHP: 18, Damage: 5, Room: "coldrow", RespawnMs: 30_000},
	"enforcer":  {ID: "enforcer", Name: "a Cinder Front enforcer", Desc: "A heavyset Front soldier in ash-grey plate -- more bully than soldier, but the gun on their hip is real enough.", MaxHP: 34, Damage: 7, Room: "checkpoint", RespawnMs: 90_000},
	"trooper":   {ID: "trooper", Name: "a Cinder Front trooper", Desc: "A drilled Front soldier in matched ash-grey gear, moving like someone who's done this killing before.", MaxHP: 30, Damage: 6, Room: "muster", RespawnMs: 60_000},
	"zealot":    {ID: "zealot", Name: "a Front zealot", Desc: "A true believer with the ash-and-flame branded into their own skin, eyes bright with the cause and nothing behind them.", MaxHP: 36, Damage: 7, Room: "warroom", RespawnMs: 75_000},
	"ashmonger": {ID: "ashmonger", Name: "the Ashmonger", Desc: "Commander of the Cinder Front: a slab-shouldered butcher in scorched plate, leaning on a cleaver as long as your leg, smiling like he's already won.", MaxHP: 100, Damage: 10, Room: "dais", RespawnMs: 180_000},
	"custodian": {ID: "custodian", Name: "the Custodian", Desc: "A hunched automaton of rusted chrome, still guarding the drowned core with a shard of light clutched in its claws.", MaxHP: 45, Damage: 8, Room: "corelab", RespawnMs: 120_000},
}

// RespawnFor returns the spawn room and respawn delay for a template id.
func RespawnFor(id string) (room string, respawnMs int, ok bool) {
	t, ok := mobCatalog[id]
	if !ok || t.RespawnMs <= 0 || t.Room == "" {
		return "", 0, false
	}
	return t.Room, t.RespawnMs, true
}

// newMob spawns a fresh full-HP instance of a template id (nil if unknown).
func newMob(id string) *Mob {
	t, ok := mobCatalog[id]
	if !ok {
		return nil
	}
	return &Mob{
		ID: t.ID, Name: t.Name, Desc: t.Desc,
		MaxHP: t.MaxHP, HP: t.MaxHP, Damage: t.Damage,
	}
}

// RemoveMob drops a live instance from a room (after a kill).
func (w *World) RemoveMob(roomID string, m *Mob) {
	r := w.Room(roomID)
	if r == nil || m == nil {
		return
	}
	for i, mm := range r.Mobs {
		if mm == m {
			r.Mobs = append(r.Mobs[:i], r.Mobs[i+1:]...)
			return
		}
	}
}

// HasMob reports whether a template id is alive in its spawn room.
func (w *World) HasMob(id string) bool {
	room, _, ok := RespawnFor(id)
	if !ok {
		return false
	}
	r := w.Room(room)
	if r == nil {
		return false
	}
	for _, m := range r.Mobs {
		if m.ID == id {
			return true
		}
	}
	return false
}

// SpawnMob places a fresh instance of id into its catalog spawn room.
func (w *World) SpawnMob(id string) *Mob {
	roomID, _, ok := RespawnFor(id)
	if !ok {
		return nil
	}
	r := w.Room(roomID)
	if r == nil {
		return nil
	}
	for _, m := range r.Mobs {
		if m.ID == id {
			return m
		}
	}
	m := newMob(id)
	if m == nil {
		return nil
	}
	r.Mobs = append(r.Mobs, m)
	return m
}
