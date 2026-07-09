package transport

import (
	"sort"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func (s *Server) groundItems(roomID string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	pile := s.ground[roomID]
	if len(pile) == 0 {
		return nil
	}
	ids := make([]string, 0, len(pile))
	for id, qty := range pile {
		if qty > 0 {
			ids = append(ids, id)
		}
	}
	sort.Strings(ids)
	return ids
}

func (s *Server) groundAdd(roomID, item string, qty int) {
	if roomID == "" || item == "" || qty <= 0 {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ground[roomID] == nil {
		s.ground[roomID] = map[string]int{}
	}
	s.ground[roomID][item] += qty
}

func (s *Server) groundRemove(roomID, item string, qty int) bool {
	if roomID == "" || item == "" || qty <= 0 {
		return false
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pile := s.ground[roomID]
	if pile == nil || pile[item] < qty {
		return false
	}
	pile[item] -= qty
	if pile[item] <= 0 {
		delete(pile, item)
	}
	if len(pile) == 0 {
		delete(s.ground, roomID)
	}
	return true
}

func (s *Server) findGroundItem(roomID, arg string) (string, bool) {
	return world.MatchItemArg(arg, s.groundItems(roomID))
}
