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

// allLocalTraces returns recent node memory across every room (newest first).
func (s *Server) allLocalTraces(limit int) []grid.EchoTrace {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]grid.EchoTrace, 0, limit)
	for _, rows := range s.localTraces {
		out = append(out, rows...)
	}
	if len(out) <= 1 {
		return out
	}
	// Insertion sort by At desc (small n).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].At > out[j-1].At; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out
}
