package transport

import (
	"fmt"
	"strings"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
)

func (s *session) cmdWar() {
	ctx, cancel := hubCtx()
	defer cancel()
	tide, err := s.srv.grid.Tide(ctx)
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
	msg := SanitizePlayerText(strings.TrimSpace(arg))
	if msg == "" {
		s.line("Gridcast what? (gridcast <message> -- the dead network carries it to every world)")
		return
	}
	ctx, cancel := hubCtx()
	defer cancel()
	if err := s.srv.grid.GridCast(ctx, s.w.Name, s.player.Name, msg); err != nil {
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
	ctx, cancel := hubCtx()
	defer cancel()
	stats, err := s.srv.grid.LedgerStats(ctx)
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
	beforeCtx, beforeCancel := hubCtx()
	before, err := s.srv.grid.LedgerStats(beforeCtx)
	beforeCancel()
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be tended.")
		return
	}
	beforeTotal := 0
	for _, r := range before {
		beforeTotal += r.Count
	}
	pruneCtx, pruneCancel := hubCtx()
	removed, err := s.srv.grid.PruneLedgerKinds(pruneCtx, ambientLedgerKinds)
	pruneCancel()
	if err != nil {
		s.line("The hub is unreachable; the deep memory cannot be tended.")
		return
	}
	afterCtx, afterCancel := hubCtx()
	after, err := s.srv.grid.LedgerStats(afterCtx)
	afterCancel()
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
	localFaction := s.player.Faction
	localMorality := s.player.Morality
	localTitle := s.player.Title
	if s.srv.grid.Remote() {
		ctx, cancel := hubCtx()
		canon, _, err := s.srv.grid.LoadCharacter(ctx, s.player.Name)
		cancel()
		if err == nil {
			applyHubSheet(s.player, canon)
			// Session-local standing wins over a stale hub read (commit is best-effort).
			if localFaction == "ally" || localFaction == "front" {
				s.player.Faction = localFaction
			}
			if localMorality != 0 && s.player.Morality == 0 && canon.Morality == 0 {
				s.player.Morality = localMorality
			}
			if localTitle != "" {
				s.player.Title = localTitle
			}
		} else {
			s.line("(the Grid is unreachable; showing your local self)")
		}
	}
	s.event(event.CharIdentity, s.player.Sheet())
	s.line("The Grid reads you back: " + identityLine(s.player))
}

func (s *session) mergeHubOnLogin() {
	if !s.srv.grid.Remote() {
		return
	}
	ctx, cancel := hubCtx()
	defer cancel()
	canon, _, err := s.srv.grid.LoadCharacter(ctx, s.player.Name)
	if err != nil {
		return
	}
	applyHubSheet(s.player, canon)
	go func() {
		regCtx, regCancel := hubCtx()
		defer regCancel()
		url := s.w.URL
		if url == "" {
			url = "ws://localhost:8790/ws"
		}
		_ = s.srv.grid.Register(regCtx, s.w.Name, url)
		_ = s.srv.grid.ClaimCharacterLease(regCtx, s.player.Name)
	}()
}

func (s *session) commitHub() {
	if !s.srv.grid.Remote() || s.player == nil {
		return
	}
	ctx, cancel := hubCtx()
	defer cancel()
	_ = s.srv.grid.CommitCharacter(ctx, s.player.Name, hubSheet(s.player))
}
