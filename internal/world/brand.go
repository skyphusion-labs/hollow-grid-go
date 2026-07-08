package world

// PlayerRef is one other player visible in a room (room.info.players).
type PlayerRef struct {
	Name     string `json:"name"`
	Standing string `json:"standing"`
}

// Brand returns the public standing tag others see on this player.
func Brand(p *Player) string {
	if p.Ashsworn {
		return "ash-sworn"
	}
	switch p.Faction {
	case "front", "Cinder Front":
		return "Cinder Front"
	case "ally":
		return "Free Folk ally"
	}
	if p.Morality >= 50 {
		return "a beacon of the wastes"
	}
	if p.Morality <= -50 {
		return "reviled"
	}
	return ""
}
