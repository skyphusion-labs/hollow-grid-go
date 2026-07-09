package transport

import (
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
)

// recordLocalTrace keeps node-scoped memory for grid.echo ping even when the
// federation backend is a remote Grid Hub (Record goes to the hub; ping reads
// local node memory, matching the TS World's SQLite caches table pattern).
func (s *Server) recordLocalTrace(node, kind, text string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.localTraces[node]
	rows = append([]grid.EchoTrace{{At: time.Now().UnixMilli(), Kind: kind, Text: text}}, rows...)
	if len(rows) > 50 {
		rows = rows[:50]
	}
	s.localTraces[node] = rows
}

func (s *Server) localTracesFor(node string, limit int) []grid.EchoTrace {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows := s.localTraces[node]
	if limit <= 0 || limit >= len(rows) {
		return append([]grid.EchoTrace(nil), rows...)
	}
	return append([]grid.EchoTrace(nil), rows[:limit]...)
}
