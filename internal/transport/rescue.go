package transport

import (
	"context"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func pickRefugeeNames(n int) []string {
	pool := append([]string(nil), world.RefugeeNames...)
	rand.Shuffle(len(pool), func(i, j int) { pool[i], pool[j] = pool[j], pool[i] })
	if n > len(pool) {
		n = len(pool)
	}
	return pool[:n]
}

func nameList(names []string) string {
	switch len(names) {
	case 0:
		return ""
	case 1:
		return names[0]
	case 2:
		return names[0] + " and " + names[1]
	default:
		return strings.Join(names[:len(names)-1], ", ") + ", and " + names[len(names)-1]
	}
}

func (s *session) emitRescued(freed []string) {
	payload := world.GridRescuedPayload{SavedBy: s.player.Name, Freed: freed}
	s.event(event.GridRescued, payload)
	ctx := context.Background()
	now := time.Now().UnixMilli()
	for _, name := range freed {
		_ = s.srv.grid.RecordRescued(ctx, s.w.Name, name, s.player.Name, now)
	}
	s.srv.rememberSaved(s.player.Name, freed...)
}

func (s *session) freeCells() {
	if !s.srv.cagesReady("cells") {
		s.line("The cages stand open and empty; someone already cut them loose. The Front will round up more soon enough -- it always does -- but not yet.")
		return
	}
	freed := pickRefugeeNames(rand.Intn(2) + 2)
	s.srv.setCageRefill("cells")
	s.deed("freed")
	s.shiftMorality(15)
	s.persist()
	s.emitRescued(freed)
	s.line("You wrench the cages open. " + nameList(freed) + " stumble out into the dark, some pausing only to grip your hand on the way past. Whatever else you are, whatever else you've done -- you did this.")
	s.srv.hub.BroadcastRoom("cells", s.player.Name+" throws open the Front's cages!", s.player.Name)
	s.recordTrace("cells", "quest", s.player.Name+" freed the caged refugees here.")
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdShelter() {
	if s.room().ID != "transit_hub" {
		s.line("There's no one here to shelter. The distress call comes from the old transit hub, south off the Scorch Road.")
		return
	}
	if !s.srv.cagesReady("transit_hub") {
		s.line("The platform is empty now. Whoever called, you got them moving -- toward the free camp, you have to believe. The Front will strand others here soon enough; it always does, and the call will go out again.")
		return
	}
	saved := pickRefugeeNames(rand.Intn(2) + 2)
	s.srv.setCageRefill("transit_hub")
	s.deed("sheltered")
	s.shiftMorality(15)
	s.persist()
	s.emitRescued(saved)
	s.line("You answer the call. You get " + nameList(saved) + " up and moving -- bottles filled at the tap, the youngest carried -- and stand watch on the cracked platform while they slip out the far side, toward the free camp and whatever the free folk can spare. The hand-radio goes quiet at last. Someone came.")
	s.srv.hub.BroadcastRoom("transit_hub", s.player.Name+" gets the stranded survivors moving toward safety.", s.player.Name)
	s.recordTrace("transit_hub", "aid", s.player.Name+" answered the transit-hub distress call and got the survivors out.")
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdSaved() {
	roll, err := s.srv.grid.RecentRescued(context.Background(), 12)
	if err != nil {
		s.line("The Grid is silent; its roll of the rescued is out of reach.")
		return
	}
	if len(roll) == 0 {
		s.line("No one has been pulled from the cages yet, or the Grid has forgotten. Find the Front's cages and change that.")
	} else {
		s.line("The Grid keeps these, pulled back out of the cages:")
		for _, r := range roll {
			place := ""
			if r.World != s.w.Name {
				place = ", on " + r.World
			}
			s.line(fmt.Sprintf("  %s  -- freed by %s%s", r.Name, r.SavedBy, place))
		}
	}
	s.event(event.GridRescuedRoll, map[string]any{"rescued": roll})
}
