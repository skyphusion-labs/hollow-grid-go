package world

// Regard returns the short standing token for player.read and grid.who.
func Regard(p *Player) string {
	if p.Ashsworn {
		return "branded"
	}
	if p.Morality >= 50 {
		return "honored"
	}
	if p.Morality <= -50 {
		return "feared"
	}
	if p.Faction == "ally" {
		return "trusted"
	}
	if p.Faction == "front" {
		return "front"
	}
	return "neutral"
}

// Tagged formats a player name with their public brand for look prose.
func Tagged(p *Player) string {
	name := p.Name
	if p.Title != "" {
		name += ", " + p.Title
	}
	if b := Brand(p); b != "" {
		return name + " (" + b + ")"
	}
	return name
}
