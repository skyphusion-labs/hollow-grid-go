package transport

import (
	"context"
	"fmt"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

var ambientLedgerKinds = []string{"ghost", "passage", "recall"}

// RunFederation starts background hub heartbeats when connected to a remote Grid.
func (s *Server) RunFederation(ctx context.Context) {
	if !s.grid.Remote() {
		return
	}
	go s.registerWorld(ctx)
	ticker := time.NewTicker(2 * time.Second)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.pollGridcasts(ctx)
				s.reportPresence(ctx)
			}
		}
	}()
}

func (s *Server) registerWorld(ctx context.Context) {
	url := s.world.URL
	if url == "" {
		url = "ws://localhost:8790/ws"
	}
	if err := s.grid.Register(ctx, s.world.Name, url); err != nil {
		s.log.Warn("grid register failed", "world", s.world.Name, "err", err)
	} else {
		s.log.Info("registered on the Grid", "world", s.world.Name, "url", url)
	}
}

func (s *Server) contributeTide(delta int) {
	go func() {
		if _, err := s.grid.ShiftTide(context.Background(), delta); err != nil {
			s.log.Debug("shiftTide failed", "delta", delta, "err", err)
		}
	}()
}

func (s *Server) pollGridcasts(ctx context.Context) {
	s.mu.Lock()
	since := s.lastCast
	s.mu.Unlock()
	casts, err := s.grid.CastsSince(ctx, since, 20)
	if err != nil || len(casts) == 0 {
		return
	}
	maxID := since
	for _, c := range casts {
		if c.ID > maxID {
			maxID = c.ID
		}
		ev, err := event.Line(event.CommGridcast, map[string]string{
			"world": c.World, "from": c.Sender, "text": c.Text,
		})
		if err != nil {
			continue
		}
		prose := fmt.Sprintf("\r\n[Grid] [%s] %s: %s\r\n", c.World, c.Sender, c.Text)
		s.hub.BroadcastAll(prose + ev + "\r\n> ")
	}
	s.mu.Lock()
	s.lastCast = maxID
	s.mu.Unlock()
}

func (s *Server) reportPresence(ctx context.Context) {
	if len(s.hub.All()) == 0 {
		return
	}
	entries := make([]grid.PresenceEntry, 0, 8)
	for _, lp := range s.hub.All() {
		entries = append(entries, grid.PresenceEntry{
			Name: lp.name, Regard: brandLive(lp), Title: lp.title,
		})
	}
	_ = s.grid.ReportPresence(ctx, s.world.Name, entries, time.Now().UnixMilli())
}

func hubSheet(p *world.Player) grid.CharSheet {
	faction := p.Faction
	if faction == "Cinder Front" {
		faction = "front"
	}
	return grid.CharSheet{
		Level: p.Level, XP: p.XP, Gold: p.Gold, Faction: faction,
		Morality: p.Morality, Title: p.Title, Race: p.Race, Ashsworn: p.Ashsworn,
	}
}

func applyHubSheet(p *world.Player, c grid.CharSheet) {
	if c.Level > 0 {
		p.Level = c.Level
	}
	p.XP = c.XP
	if c.Gold > 0 || c.Race != "" {
		p.Gold = c.Gold
	}
	if c.Faction != "" {
		p.Faction = c.Faction
	}
	p.Morality = c.Morality
	p.Title = c.Title
	if c.Race != "" {
		p.Race = c.Race
	}
	if c.Ashsworn {
		p.Ashsworn = true
	}
	p.RecalcMaxHP()
}
