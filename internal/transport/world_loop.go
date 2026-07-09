package transport

import (
	"context"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

type pendingRespawn struct {
	templateID string
	roomID     string
	at         int64 // unix ms
}

// RunWorldLoop ticks server-wide world mechanics (mob respawns) on the same
// beat as session heartbeats. Runs even with zero players connected.
func (s *Server) RunWorldLoop(ctx context.Context) {
	ticker := time.NewTicker(worldHeartbeat)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				s.tickRespawns()
			}
		}
	}()
}

func (s *Server) killMob(roomID string, m *world.Mob) {
	if m == nil {
		return
	}
	s.world.RemoveMob(roomID, m)
	spawnRoom, respawnMs, ok := world.RespawnFor(m.ID)
	if !ok {
		return
	}
	if spawnRoom == "" {
		spawnRoom = roomID
	}
	s.mu.Lock()
	s.deadMobs[m.ID] = pendingRespawn{
		templateID: m.ID,
		roomID:     spawnRoom,
		at:         time.Now().Add(time.Duration(respawnMs) * time.Millisecond).UnixMilli(),
	}
	s.mu.Unlock()
}

func (s *Server) tickRespawns() {
	now := time.Now().UnixMilli()
	var due []pendingRespawn
	s.mu.Lock()
	for id, p := range s.deadMobs {
		if p.at <= now {
			due = append(due, p)
			delete(s.deadMobs, id)
		}
	}
	s.mu.Unlock()
	for _, p := range due {
		if s.world.HasMob(p.templateID) {
			continue
		}
		m := s.world.SpawnMob(p.templateID)
		if m == nil {
			continue
		}
		prose := capitalize(m.Name) + " stalks into view.\r\n"
		s.hub.BroadcastRoom(p.roomID, prose, "")
	}
}
