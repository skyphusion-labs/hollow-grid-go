package transport

import (
	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

const carouseCost = 10

func (s *session) cmdResist() {
	if s.room().ID != "tavern" {
		s.line("There's no temptation here to resist.")
		return
	}
	if s.player.Resisted {
		s.line("You've already made your peace with this place. You keep your coin and your wits.")
		return
	}
	s.player.Resisted = true
	s.shiftMorality(5)
	s.persist()
	s.line("You wave off the dust and the wench both, jaw set. Your head stays clear. There's pride in that.")
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdCarouse() {
	if s.room().ID != "tavern" {
		s.line("There's no one here to keep you company.")
		return
	}
	if s.player.Gold < carouseCost {
		s.line("The wench looks you over, sees empty pockets, and moves on.")
		return
	}
	s.player.Gold -= carouseCost
	s.shiftMorality(-8)
	immune := world.RaceByID(s.player.Race).PoisonImmune
	msg := "You spend coin and an hour in the back; the details stay between you and the rafters."
	if !s.player.Poisoned && !immune {
		s.player.Poisoned = true
		msg += "\r\nBy morning, though, something burns that shouldn't. You've caught the pox. (afflicted)"
	}
	s.persist()
	s.line(msg)
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
}
