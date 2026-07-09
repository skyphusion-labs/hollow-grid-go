package world

// seedEndgame grafts the Sunken Server Farm, open-wastes links, transit hub, and
// Cinder Front stronghold onto the canonical map (src/rooms.ts).
func (w *World) seedEndgame() {
	rooms := []*Room{
		{ID: "sump", Name: "The Sump",
			Desc:  "Ankle-deep in oily runoff that glows a sick green. The walls sweat. Whatever lives down here, lives hungry. A buckled bulkhead gapes below, and cold air pours up out of it.",
			Exits: map[string]string{"up": "tunnels", "down": "floodgate"}},
		{ID: "floodgate", Name: "The Breached Floodgate",
			Desc:  "A bulkhead the size of a truck, buckled open. The sump's runoff pours through it and down into a drowned cathedral of machines. A stranded operator huddles by a dead console, watching you with wary hope. (try 'talk')",
			Exits: map[string]string{"up": "sump", "north": "coldrow"}},
		{ID: "coldrow", Name: "Cold Storage Row",
			Desc:  "Aisle after aisle of server racks stand hip-deep in black water, their status lights long dead. Something pale flickers between them, feeding on whatever current is left.",
			Exits: map[string]string{"south": "floodgate", "east": "cooling", "north": "fiber"}},
		{ID: "cooling", Name: "The Cooling Pools",
			Desc:  "Great square pools of coolant gone to scum and rust. A maintenance unit lurches through the shallows on three working legs, still trying to do its job.",
			Exits: map[string]string{"west": "coldrow"}},
		{ID: "fiber", Name: "The Fiber Vault",
			Desc:  "A cathedral nave of severed fiber-optic trunks, each thick as your arm, hanging dead from the ceiling. This was the spine of the Grid once. Something cold still moves along the cables, where the light used to.",
			Exits: map[string]string{"south": "coldrow", "down": "corelab"}},
		{ID: "corelab", Name: "The Core Lab",
			Desc:  "The drowned heart of the data center. A single black monolith of a server still hums, impossibly, in the dark, and something has made itself its keeper. It turns to face you. (the Custodian guards it)",
			Exits: map[string]string{"up": "fiber", "west": "archive"}},
		{ID: "archive", Name: "The Cold Archive",
			Desc:  "A sealed vault of tape spools and frozen drives, untouched by the flood. The air is bone-dry and very cold. Whatever the Grid wanted to keep forever, it kept here.",
			Exits: map[string]string{"east": "corelab"}},
		{ID: "checkpoint", Name: "The Cinder Front Checkpoint", Outdoors: true,
			Desc:  "Sandbags, razor-wire, and a banner stamped with the Front's ash-and-flame mark. An enforcer mans the barrier, and the road runs north toward the Front's stronghold.",
			Exits: map[string]string{"south": "dunes", "north": "gate"}},
		{ID: "transit_hub", Name: "The Old Transit Hub", Outdoors: true,
			Desc:  "A derelict transit station, platforms cracked, the departure board frozen on a destination that does not exist anymore. Survivors huddle around a water tap; one of them still works a hand-radio, sending the looping call you followed here. (try 'shelter')",
			Exits: map[string]string{"north": "scorch_road"}},
		{ID: "gate", Name: "The Cinder Gate", Outdoors: true,
			Desc:  "A fortress wall of welded scrap and old shipping containers, the ash-and-flame banner snapping overhead. Troopers watch from firing slits. This is the heart of the Front.",
			Exits: map[string]string{"south": "checkpoint", "north": "muster"}},
		{ID: "muster", Name: "The Muster Yard", Outdoors: true,
			Desc:  "A packed-dirt parade ground where the Front drills, ringed by barracks. Cages line the west wall; the war room looms to the north.",
			Exits: map[string]string{"south": "gate", "west": "cells", "north": "warroom"}},
		{ID: "cells", Name: "The Cages",
			Desc:  "A row of welded cages, packed with elf refugees the Front has rounded up. They press to the bars when you enter, hope and terror warring in their faces. (try 'free')",
			Exits: map[string]string{"east": "muster"}},
		{ID: "warroom", Name: "The War Room",
			Desc:  "A blast-shelter strung with maps of the wastes, every refugee settlement circled in red. A zealot pores over the plans. A ladder climbs to the commander's dais above.",
			Exits: map[string]string{"south": "muster", "up": "dais"}},
		{ID: "dais", Name: "The Ashmonger's Dais", Outdoors: true,
			Desc:  "A raised platform of stacked rubble crowned with the Front's banner. The Ashmonger himself stands here, commander of the Cinder Front, surveying the wastes he means to own.",
			Exits: map[string]string{"down": "warroom"}},
	}
	for _, r := range rooms {
		w.rooms[r.ID] = r
	}
	// Link the canonical graph.
	if t := w.rooms["tunnels"]; t != nil {
		t.Exits["down"] = "sump"
		t.Desc = "Cramped, dripping, and lit by one surviving strip light. Something skitters in the dark just past the reach of it. A flooded shaft drops away below."
	}
	if d := w.rooms["dunes"]; d != nil {
		d.Exits["north"] = "checkpoint"
		d.Desc = "Open desert under a bleached sky, dunes of grey ash rolling to the horizon. A cracked highway runs east, and the silhouette of a checkpoint stands to the north."
	}
	if s := w.rooms["scorch_road"]; s != nil {
		s.Exits["south"] = "transit_hub"
		s.Desc = "A ruined stretch of pre-collapse highway, asphalt buckled and tar-black, burned-out hulks lining the shoulder. A faded sign points south to an old transit hub; that's where the distress call is coming from."
	}
	// Spawn the endgame bestiary.
	w.rooms["coldrow"].Mobs = []*Mob{newMob("leech")}
	w.rooms["checkpoint"].Mobs = []*Mob{newMob("enforcer")}
	w.rooms["muster"].Mobs = []*Mob{newMob("trooper")}
	w.rooms["warroom"].Mobs = []*Mob{newMob("zealot")}
	w.rooms["dais"].Mobs = []*Mob{newMob("ashmonger")}
	w.rooms["corelab"].Mobs = []*Mob{newMob("custodian")}
}

// RefugeeNames are procedural names given to the rescued (src/world.ts).
var RefugeeNames = []string{
	"Sera", "Tomas", "old Wick", "Bex", "Halden", "the Marsh twins", "Ona", "Pavel",
	"little Resh", "Caro", "Dunne", "Yusa", "the smith's boy", "Mira", "Teo", "Nell",
}
