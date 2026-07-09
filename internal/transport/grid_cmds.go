package transport

import (
	"context"
	"fmt"
	"strings"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
)

func (s *session) cmdWar() {
	tide, err := s.srv.grid.Tide(context.Background())
	if err != nil {
		s.line("The deep Grid is silent; you can't read the war from here.")
		return
	}
	state := warState(tide)
	s.line(fmt.Sprintf("Across the whole Grid, the war for the wastes: %s (tide %s%d)", state, tideSign(tide), absInt(tide)))
	if tide >= 40 {
		s.line("  And you can see it in the world itself: the wastes are starting, here and there, to come back to life.")
	} else if tide <= -40 {
		s.line("  And you can see it in the world itself: everything is drawing in, going quiet and afraid.")
	}
	s.event(event.WorldWar, map[string]int{"tide": tide})
}

func warState(tide int) string {
	switch {
	case tide <= -50:
		return "the Cinder Front is ascendant -- the free folk are being driven under, across every world at once."
	case tide >= 50:
		return "the free folk are winning -- the Front is breaking, everywhere."
	case tide < 0:
		return "the Front holds the edge, for now."
	case tide > 0:
		return "the free folk are holding their ground."
	default:
		return "the war hangs in perfect, brutal balance."
	}
}

func tideSign(n int) string {
	if n >= 0 {
		return "+"
	}
	return ""
}

func absInt(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func (s *session) cmdGridcast(arg string) {
	msg := strings.TrimSpace(arg)
	if msg == "" {
		s.line("Gridcast what? (gridcast <message> -- the dead network carries it to every world)")
		return
	}
	if err := s.srv.grid.GridCast(context.Background(), s.w.Name, s.player.Name, msg); err != nil {
		s.line("The Grid swallows your words; the network is unreachable.")
		return
	}
	s.line(`You cast your voice into the dead Grid, out across every node: "` + msg + `"`)
}

func (s *session) cmdGridstats() {
	if !s.srv.isAdmin(s.player.Name) {
		s.line("Only a keeper of the Grid can read its deep memory.")
		return
	}
	stats, err := s.srv.grid.LedgerStats(context.Background())
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be read.")
		return
	}
	total := 0
	for _, r := range stats {
		total += r.Count
	}
	s.line(fmt.Sprintf("The Grid ledger holds %d trace(s):", total))
	for _, r := range stats {
		s.line(fmt.Sprintf("  %-10s %d", r.Kind, r.Count))
	}
	s.event(event.GridLedgerStats, map[string]any{"total": total, "kinds": stats})
}

func (s *session) cmdGridprune() {
	if !s.srv.isAdmin(s.player.Name) {
		s.line("Only a keeper of the Grid can tend its deep memory.")
		return
	}
	before, err := s.srv.grid.LedgerStats(context.Background())
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be tended.")
		return
	}
	beforeTotal := 0
	for _, r := range before {
		beforeTotal += r.Count
	}
	removed, err := s.srv.grid.PruneLedgerKinds(context.Background(), ambientLedgerKinds)
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be tended.")
		return
	}
	after, err := s.srv.grid.LedgerStats(context.Background())
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be tended.")
		return
	}
	afterTotal := 0
	for _, r := range after {
		afterTotal += r.Count
	}
	s.line(fmt.Sprintf("Pruned %d ambient trace(s) (%s).", removed.Removed, strings.Join(ambientLedgerKinds, ", ")))
	s.line(fmt.Sprintf("The ledger went from %d to %d trace(s); only meaningful memory remains.", beforeTotal, afterTotal))
	s.event(event.GridLedgerPruned, map[string]any{
		"removed": removed.Removed, "before": beforeTotal, "after": afterTotal, "kinds": after,
	})
}

func (s *session) cmdWhoami() {
	sheet := s.player.Sheet()
	if s.srv.grid.Remote() {
		if canon, _, err := s.srv.grid.LoadCharacter(context.Background(), s.player.Name); err == nil {
			applyHubSheet(s.player, canon)
			sheet = s.player.Sheet()
		} else {
			s.line("(the Grid is unreachable; showing your local self)")
		}
	}
	s.event(event.CharIdentity, sheet)
	s.line("The Grid reads you back: " + identityLine(s.player))
}

func (s *session) mergeHubOnLogin() {
	if !s.srv.grid.Remote() {
		return
	}
	canon, _, err := s.srv.grid.LoadCharacter(context.Background(), s.player.Name)
	if err != nil {
		return
	}
	applyHubSheet(s.player, canon)
	go func() {
		url := s.w.URL
		if url == "" {
			url = "ws://localhost:8790/ws"
		}
		_ = s.srv.grid.Register(context.Background(), s.w.Name, url)
	}()
}

func (s *session) commitHub() {
	if !s.srv.grid.Remote() || s.player == nil {
		return
	}
	_ = s.srv.grid.CommitCharacter(context.Background(), s.player.Name, hubSheet(s.player))
}
