// Package transport implements the Hollow Grid player transport: a WebSocket
// endpoint at /ws speaking plain UTF-8 text with CRLF lines, plus the two
// unauthenticated health probes. See docs/protocol.md section 1.
package transport

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// Server wires the HTTP surface for one world.
type Server struct {
	world *world.World
	store store.CharStore
	log   *slog.Logger
	conns sync.WaitGroup // in-flight player sessions, for graceful drain
}

// NewServer builds a transport server for the given world and character store.
func NewServer(w *world.World, st store.CharStore, log *slog.Logger) *Server {
	return &Server{world: w, store: st, log: log}
}

// Handler returns the world's HTTP handler (/ws, /health, /health/deep).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /health/deep", s.healthDeep)
	mux.HandleFunc("/ws", s.ws)
	return mux
}

// Wait blocks until all in-flight player sessions have ended. Pair it with the
// HTTP server's Shutdown for a clean stop (and to drain final persists).
func (s *Server) Wait() { s.conns.Wait() }

// health is sub-millisecond liveness; always 200. (protocol.md s1)
func (s *Server) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{
		"ok": true, "ts": time.Now().UnixMilli(), "world": s.world.Name,
	})
}

// healthDeep exercises dependencies once each. Only the world is critical; the
// grid hub is reported but non-critical because federation never blocks play.
// Returns 503 only when a critical check fails.
func (s *Server) healthDeep(w http.ResponseWriter, r *http.Request) {
	worldOK := s.world.Start() != nil
	checks := map[string]any{
		"world": map[string]any{"ok": worldOK, "latency_ms": 0, "critical": true},
		// Non-critical: federation never blocks play (docs/federation.md s8).
		// Stubbed reachable while standalone; becomes a real tide() probe when the
		// federation client lands (Phase 3).
		"grid_hub": map[string]any{"ok": true, "latency_ms": 0, "critical": false},
	}
	code := http.StatusOK
	if !worldOK {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"ok": worldOK, "ts": time.Now().UnixMilli(), "world": s.world.Name, "checks": checks,
	})
}

// ws upgrades to a WebSocket and runs one player session. Origin checks are
// skipped: a raw client (wscat, a bot) is a first-class game client.
func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{InsecureSkipVerify: true})
	if err != nil {
		s.log.Warn("ws accept failed", "err", err)
		return
	}
	s.conns.Add(1)
	defer s.conns.Done()
	defer c.CloseNow()
	handleConn(r.Context(), c, s.world, s.store, s.log)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
