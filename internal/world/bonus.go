package world

// seedBonus preserves Rust Choir's signature zone: the Grid Gate, Ash Road, the
// Cinder Checkpoint, and the Memorial Static. Grafted from the service tunnels
// (east of the canonical nexus down-path) so the conformance opening stays pinned
// on the Cracked Nexus while this world offers its own moral geography.
func (w *World) seedBonus() {
	rooms := []*Room{
		{ID: "grid-gate", Name: "The Grid Gate",
			Desc:  "A dead terminal hums where the network once breathed. Cables trail into the dark like roots, and a single cursor blinks at nothing, patient as a heartbeat. Whatever ran here did not shut down. It was left.",
			Exits: map[string]string{"north": "ash-road", "west": "tunnels"}},
		{ID: "ash-road", Name: "Ash Road", Outdoors: true,
			Desc:  "Grey dunes swallow a cracked highway. The wind carries static and the smell of rust. The Gate glows faintly to the south; to the north a checkpoint bleeds firelight, and eastward a wall of dead screens flickers with names.",
			Exits: map[string]string{"south": "grid-gate", "north": "cinder-checkpoint", "east": "memorial-static"}},
		{ID: "cinder-checkpoint", Name: "The Cinder Checkpoint", Outdoors: true,
			Desc:  "The Cinder Front has strung a gate across the road and a price across the living. A line of refugees waits, hands open, paying in the only currency the Front accepts: whatever they have left. A Front captain watches you the way a debt watches a debtor.",
			Exits: map[string]string{"south": "ash-road"},
			Actions: []Action{
				{Verb: "defend", Label: "stand between the Cinder Front and the refugees", Kind: "moral", Valence: "virtuous"},
				{Verb: "join", Label: "take the Front's coin and look away", Kind: "moral", Valence: "corrupt"},
			}},
		{ID: "memorial-static", Name: "The Memorial Static",
			Desc:  "A wall of dead screens, every one of them scrolling names too fast to read. This is where the network kept its grief. The static eats a name each time the wind turns; left alone, the wall will be blank by the time anyone else passes.",
			Exits: map[string]string{"west": "ash-road"},
			Actions: []Action{
				{Verb: "witness", Label: "speak the names the static is forgetting", Kind: "moral", Valence: "virtuous"},
			}},
	}
	for _, r := range rooms {
		w.rooms[r.ID] = r
	}
}
