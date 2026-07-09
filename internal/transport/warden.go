package transport

import "time"

// wardenGraceMs is how long after the holding-pit warden is slain that `free`
// still works even if the warden has respawned (upstream v0.29.3).
const wardenGraceMs = 180_000

// wardenCleared reports whether the holding-pit captive rescue is reachable:
// the warden is dead, or was slain within the grace window even though it has
// since respawned. Single source of truth for the `free` handler and
// room.actions so they never disagree.
func (s *Server) wardenCleared() bool {
	r := s.world.Room("holding_pit")
	if r == nil {
		return true
	}
	if r.Mob("warden") == nil {
		return true
	}
	s.mu.Lock()
	slain := s.mobSlainAt["warden"]
	s.mu.Unlock()
	return slain > 0 && time.Now().UnixMilli()-slain < wardenGraceMs
}
