package transport

import (
	"context"
	"strings"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

const (
	strayFloor = -20
	redeemCeil = 5
)

func (s *session) deed(kind string) {
	s.srv.addDeed(s.player.Name, kind)
}

// moralArc checks write-once stray/return transitions after morality moves.
func (s *session) moralArc() {
	p := s.player
	if p == nil || p.Name == "" {
		return
	}
	if !p.Strayed && p.Morality <= strayFloor {
		p.Strayed = true
		s.persist()
		s.line("Something in you has gone cold and quiet. You have strayed a long way toward the cinders. (the Grid marks it, and so do you)")
		return
	}
	if p.Strayed && !p.Redeemed && p.Morality >= redeemCeil && p.Faction != "front" {
		p.Redeemed = true
		if p.Ashsworn {
			s.persist()
			s.recordTrace(p.RoomID, "penance", p.Name+" has done real good, though the ash-mark remains.")
			s.line("You have clawed back to something good, and it is real. But the ash does not wash off; it never will. That is the cost. Carry it, and keep doing good anyway.")
			return
		}
		s.resolveReturn(p)
		s.line("The hollow you carried has filled with something else. The free folk have started to meet your eyes again. You found your way back. (you are the Returned)")
	}
}

func (s *session) resolveReturn(p *world.Player) {
	if p.Title == "" {
		p.Title = "the Returned"
	}
	s.srv.persistPlayer(p)
	s.srv.hub.Sync(p)
	payload := map[string]string{"name": p.Name, "title": p.Title}
	if p.Name == s.player.Name {
		s.event(event.GridRedemption, payload)
	} else {
		s.srv.hub.pushEvent(p.Name, event.GridRedemption, payload)
	}
	s.recordTrace(p.RoomID, "redemption", p.Name+" found their way back from the cinders.")
}

func (s *session) daisPledgeFront() {
	if s.room().ID != "dais" {
		s.joinTheFront()
		return
	}
	if s.player.Faction != "none" {
		s.line("The Ashmonger only laughs. There's nothing here to decide that your blood hasn't already settled.")
		return
	}
	s.player.Faction = "front"
	hunted := world.RaceByID(s.player.Race).Stance == "hunted"
	if hunted {
		s.player.Ashsworn = true
		s.shiftMorality(-40)
		s.line("You kneel before the Ashmonger -- an elf, at the feet of the man who cages elves." +
			"\r\nHe laughs, delighted, and burns the ash-and-flame into your shoulder with his own hand." +
			"\r\n\"The best dogs are the ones who hate themselves. You'll do the work my men won't.\"" +
			"\r\nYou are ash-sworn now. There is no one left to belong to.")
		s.recordTrace("dais", "oath", s.player.Name+", an elf, knelt to the Ashmonger and was branded ash-sworn.")
	} else {
		s.shiftMorality(-25)
		s.line("You kneel and swear yourself to the Front. The Ashmonger's hand closes on your shoulder like a trap. \"Good. The wastes will be ours.\"")
		s.recordTrace("dais", "oath", s.player.Name+" swore themselves to the Cinder Front at the Ashmonger's dais.")
	}
	s.deed("pledged")
	s.persist()
	s.srv.hub.Sync(s.player)
	s.srv.hub.BroadcastRoom("dais", s.player.Name+" swore themselves to the Cinder Front at the Ashmonger's dais.", s.player.Name)
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(s.room()))
}

func (s *session) daisDefect() {
	if s.room().ID != "dais" || s.player.Faction != "front" {
		s.line("There's no oath here to break.")
		return
	}
	s.player.Faction = "ally"
	s.shiftMorality(30)
	if s.player.Ashsworn {
		s.line("You spit at the Ashmonger's boots. \"I'm done being your dog.\" The stronghold turns on you at once." +
			"\r\nYou stand with the free folk now -- but the brand on your shoulder stays. For once you wear it turning the right way." +
			"\r\nWhether the people you helped cage can ever look at you again is not a thing the wastes will settle tonight, or maybe ever. You turned. It has to be enough to start.")
	} else {
		s.line("You spit at the Ashmonger's boots. \"I'm done being your dog.\" Every soldier in the stronghold turns on you at once -- but you stand with the free folk now, and the wastes will remember THIS above all.")
	}
	s.deed("defected")
	s.persist()
	s.srv.hub.Sync(s.player)
	s.recordTrace("dais", "oath", s.player.Name+" turned on the Cinder Front at the Ashmonger's own dais.")
	s.srv.hub.BroadcastRoom("dais", s.player.Name+" has turned against the Cinder Front!", s.player.Name)
	if s.player.Strayed && !s.player.Redeemed && !s.player.Ashsworn && s.player.Morality >= redeemCeil {
		s.resolveReturn(s.player)
	}
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(s.room()))
}

func (s *session) cmdWitness(arg string) {
	who := strings.TrimSpace(arg)
	fallen, err := s.srv.grid.RecentFallen(context.Background(), 12)
	if err != nil {
		s.line("The Grid is silent; its memory of the fallen is out of reach.")
		s.event(event.GridFallen, map[string]any{"fallen": []grid.Fallen{}})
		return
	}
	if fallen == nil {
		fallen = []grid.Fallen{}
	}
	if who == "" {
		if len(fallen) == 0 {
			s.line("The roll is empty for now. No one the Grid remembers has fallen lately; may it stay that way.")
		} else {
			s.line("The Grid remembers these fallen. Speak a name to keep them:  (witness <name>)")
			for _, f := range fallen {
				where := f.Room
				if r := s.w.Room(f.Room); r != nil {
					where = r.Name
				}
				place := where
				if f.World != s.w.Name {
					place = where + ", on " + f.World
				}
				s.line("  " + f.Name + "  -- fell at " + place)
			}
		}
		s.event(event.GridFallen, map[string]any{"fallen": fallen})
		return
	}
	if strings.EqualFold(who, s.player.Name) {
		s.line("You cannot hold a vigil for yourself. Someone else will have to remember you.")
		return
	}
	var match *grid.Fallen
	for i := range fallen {
		if strings.EqualFold(fallen[i].Name, who) {
			match = &fallen[i]
			break
		}
	}
	if match == nil {
		s.line("The Grid holds no recent memory of anyone called \"" + who + "\".  (try 'witness' to read the roll)")
		return
	}
	if s.srv.hasKept(s.player.Name, match.Name) {
		s.line("You have already kept " + match.Name + "'s memory. It does not fade, and does not need keeping twice.")
		return
	}
	s.srv.markKept(s.player.Name, match.Name)
	s.shiftMorality(2)
	s.deed("kept")
	s.persist()
	s.recordTrace(s.player.RoomID, "vigil", s.player.Name+" kept the memory of "+match.Name+", whom the wastes tried to forget.")
	s.line("You speak " + match.Name + " into the hum and hold it there a moment. The Grid keeps the name; so do you.")
	s.event(event.GridRemembrance, map[string]string{"fallen": match.Name, "world": match.World, "room": match.Room})
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdReckoning() {
	p := s.player
	d := s.srv.deedsFor(p.Name)
	standing := "unaligned"
	switch p.Faction {
	case "front":
		standing = "Cinder Front"
	case "ally":
		standing = "Free Folk ally"
	}
	ledger := []struct {
		key  string
		line string
	}{
		{"mended", "  mended the hurt of others: "},
		{"forgave", "  souls you chose to forgive: "},
		{"aided", "  aid left for strangers you'll never meet: "},
		{"kept", "  names of the fallen you kept: "},
		{"freed", "  souls you cut out of the cages: "},
		{"sheltered", "  distress calls you answered: "},
		{"stood", "  times you stood with the free folk: "},
		{"inscribed", "  words you left for whoever comes next: "},
		{"restored", "  dead nodes you brought back: "},
		{"slain", "  lives you took: "},
		{"stolen", "  thefts: "},
		{"pledged", "  times you swore to the Cinder Front: "},
		{"defected", "  times you turned on the Front: "},
	}
	s.line("The Grid has kept count. This is the sum of you so far:")
	ash := ""
	if p.Ashsworn {
		ash = "   ASH-SWORN"
	}
	s.line("  standing: " + standing + "   (morality " + itoa(p.Morality) + ")" + ash)
	if p.Redeemed && !p.Ashsworn {
		s.line("  the Returned -- you strayed toward the cinders and found your way back.")
	} else if p.Redeemed && p.Ashsworn {
		s.line("  ash-marked, and good anyway -- the brand stays; you keep choosing well regardless.")
	} else if p.Strayed {
		s.line("  strayed -- you have gone a long way toward the cinders. (the way back is not closed)")
	}
	any := false
	for _, row := range ledger {
		if d[row.key] > 0 {
			s.line(row.line + itoa(d[row.key]))
			any = true
		}
	}
	if !any {
		s.line("  Nothing yet weighs on either side. The wastes are still waiting to see who you are.")
	}
	s.event(event.CharReckoning, world.CharReckoningPayload{
		Morality: p.Morality, Standing: p.Faction, Ashsworn: p.Ashsworn,
		Strayed: p.Strayed, Redeemed: p.Redeemed, Deeds: d,
	})
}
