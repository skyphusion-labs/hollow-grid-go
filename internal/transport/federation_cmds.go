package transport

import (
	"strings"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
)

func (s *session) cmdWorlds() {
	ctx, cancel := hubCtx()
	defer cancel()
	worlds, err := s.srv.grid.ListWorlds(ctx)
	if err != nil {
		s.line("The Grid is silent; the registry is out of reach.")
		return
	}
	now := time.Now().UnixMilli()
	rows := make([]map[string]any, 0, len(worlds))
	lines := []string{"Worlds linked on the Grid (say 'travel <world>'):"}
	for _, w := range worlds {
		reachable := w.LastSeen > 0
		active := w.LastSeen > now-60_000
		here := w.ID == s.w.Name
		tag := "seeded (not yet live)"
		switch {
		case here:
			tag = "you are here"
		case reachable && active:
			tag = "reachable, active now"
		case reachable:
			tag = "reachable, quiet"
		}
		lines = append(lines, "  "+w.ID+"  ["+tag+"]")
		rows = append(rows, map[string]any{
			"id": w.ID, "reachable": reachable, "active": active,
			"lastSeen": w.LastSeen, "here": here,
		})
	}
	s.line(strings.Join(lines, "\r\n"))
	s.event(event.GridWorlds, map[string]any{"worlds": rows})
}

// cmdTravel checkpoints the character and hands off to another world. Returns
// true when the session should close (protocol.md travel flow).
func (s *session) cmdTravel(arg string) bool {
	target := strings.TrimSpace(arg)
	if target == "" {
		s.line("Travel where? (say 'worlds' to see the Grid)")
		return false
	}
	if s.player.Target != nil {
		s.line("You can't key out through the Grid in the middle of a fight.")
		return false
	}
	ctx, cancel := hubCtx()
	defer cancel()
	worlds, err := s.srv.grid.ListWorlds(ctx)
	if err != nil {
		s.line("The Grid won't answer; travel is impossible right now.")
		return false
	}
	t := strings.ToLower(target)
	var dest *grid.WorldInfo
	for i := range worlds {
		if strings.EqualFold(worlds[i].ID, target) {
			dest = &worlds[i]
			break
		}
	}
	if dest == nil {
		for i := range worlds {
			if strings.Contains(strings.ToLower(worlds[i].ID), t) {
				dest = &worlds[i]
				break
			}
		}
	}
	if dest == nil {
		s.line("No world called \"" + target + "\" answers on the Grid. (try 'worlds')")
		return false
	}
	if dest.ID == s.w.Name {
		s.line("You're already in " + s.w.Name + ".")
		return false
	}
	s.persist()
	s.srv.hub.BroadcastRoom(s.player.RoomID, s.player.Name+" keys into the Grid and is routed away, off the edge of the world.", s.player.Name)
	s.line("The Grid takes you apart, packet by packet, and routes you toward " + dest.ID + ".")
	s.line("Reconnect there and you arrive as yourself -- your name, level, and standing all travel with you:")
	s.line("    " + dest.URL)
	s.line("(This world is letting you go. See you on the other side.)")
	s.event(event.GridTravel, map[string]string{"to": dest.ID, "url": dest.URL})
	return true
}
