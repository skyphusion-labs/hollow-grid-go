package world

import (
	"strconv"
	"strings"
)

// Ability is a race's signature active: a verb on a cooldown.
type Ability struct {
	Verb       string `json:"verb"`
	Name       string `json:"name"`
	Desc       string `json:"desc"`
	CooldownMs int    `json:"cooldownMs"`
}

// Race is a federated, opaque identity label with light local mechanical leans.
// Stance is how the Cinder Front treats this people -- "accepted", "tolerated", or
// "hunted" -- which is the heart of the moral system: the race you choose places
// you in the Front's machinery before you have done anything. (src/races.ts.)
type Race struct {
	ID           string
	Name         string
	Blurb        string
	Stance       string
	HPMod        int
	Damage       int
	Armor        int
	Regen        int
	PoisonImmune bool
	Trait        string
	Ability      Ability
}

// Races is the canonical roster in menu order (src/races.ts RACE_ORDER). The race
// is the opaque federated label the Grid carries across worlds; these match the
// reference so the conformance suite (which logs in as "human") and cross-world
// identity line up. Any world may define its own races.
var Races = []Race{
	{ID: "human", Name: "Human", Blurb: "the Registered -- the Front's idea of a real person", Stance: "accepted",
		Trait:   "Unmarked. The registry, the vendors, and the checkpoints treat you as a person by default.",
		Ability: Ability{"requisition", "Requisition", "call in a registry payout; the system pays its own", 180000}},
	{ID: "elf", Name: "Elf", Blurb: "the Unregistered -- the people the Cinder Front hunts", Stance: "hunted", Regen: 1,
		Trait:   "Quick and resilient; you recover a little faster. The Front's cages, rallies, and checkpoints are about you.",
		Ability: Ability{"vanish", "Vanish", "slip the net: break off any fight and disappear", 45000}},
	{ID: "revenant", Name: "Revenant", Blurb: "a mind the network kept after the body failed", Stance: "hunted", PoisonImmune: true,
		Trait:   "No flesh to rot: poison and the pox cannot touch you. The Front calls you an abomination, not a citizen.",
		Ability: Ability{"commune", "Commune", "reach into the dead Grid for its memory and a little of its cold life", 120000}},
	{ID: "ghoul", Name: "Ghoul", Blurb: "rad-scoured human, hard to kill", Stance: "tolerated", HPMod: 10,
		Trait:   "You carry more hit points than flesh should. The Front works you, and never lets you forget you are not 'real'.",
		Ability: Ability{"regenerate", "Regenerate", "rad-scoured flesh knits itself back: a heavy self-heal", 120000}},
	{ID: "chromed", Name: "Chromed", Blurb: "flesh half-replaced with salvage augments", Stance: "tolerated", Damage: 1, Armor: 1,
		Trait:   "Chrome under the skin: a little more bite, a little more plate. The Front's muscle is chromed too, until you go too far.",
		Ability: Ability{"overclock", "Overclock", "vent your augments past every safety into one devastating strike", 30000}},
	{ID: "dustkin", Name: "Dustkin", Blurb: "born to the open pan, owing the registry nothing", Stance: "hunted", Regen: 2,
		Trait:   "At home where others die: you heal faster out in the world. The Front hunts you as a vagrant.",
		Ability: Ability{"forage", "Forage", "scavenge the open wastes for supplies (outdoors only)", 90000}},
	{ID: "vatborn", Name: "Vatborn", Blurb: "grown, not born, in the old fabrication vats", Stance: "hunted", HPMod: 5,
		Trait:   "Printed sturdy: a little extra frame. No lineage the Front recognizes, so they call you property.",
		Ability: Ability{"fabricate", "Fabricate", "print a field stim from raw salvage", 120000}},
}

// RaceByChoice resolves a menu answer: a 1-based index in menu order, or an
// id/name (case-insensitive).
func RaceByChoice(answer string) (Race, bool) {
	answer = strings.TrimSpace(answer)
	if n, err := strconv.Atoi(answer); err == nil {
		if n >= 1 && n <= len(Races) {
			return Races[n-1], true
		}
		return Race{}, false
	}
	for _, r := range Races {
		if strings.EqualFold(r.ID, answer) || strings.EqualFold(r.Name, answer) {
			return r, true
		}
	}
	return Race{}, false
}

// RaceByID re-resolves a (resumed) character's racial leans from its opaque
// label, defaulting to human for an unknown label.
func RaceByID(id string) Race {
	for _, r := range Races {
		if strings.EqualFold(r.ID, id) {
			return r
		}
	}
	return Races[0]
}
