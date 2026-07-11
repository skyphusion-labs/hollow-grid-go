// Package transport implements the Hollow Grid player transport: a WebSocket
// endpoint at /ws speaking plain UTF-8 text with CRLF lines, plus the two
// unauthenticated health probes. See docs/protocol.md section 1.
package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// Server wires the HTTP surface for one world.
type Server struct {
	world       *world.World
	store       store.CharStore
	grid        grid.Hub
	hub         *Hub
	log         *slog.Logger
	conns       sync.WaitGroup
	admins      map[string]bool
	caches      map[string]int              // room id -> gold left for strangers
	localTraces map[string][]grid.EchoTrace // room id -> node memory for grid.echo
	forgiven    map[forgivenPair]bool
	cages       map[string]int64    // room id -> unix ms when refill completes (0 = ready)
	saved       map[string][]string // player name -> people they rescued
	deeds       map[string]map[string]int
	kept        map[keptPair]bool
	deadMobs    map[string]pendingRespawn // template id -> respawn schedule
	mobSlainAt  map[string]int64          // template id -> unix ms when last slain
	ground      map[string]map[string]int // room id -> item id -> qty
	lastTide    int                       // cached collective tide from the hub
	lastCast    int
	combatMu    sync.Mutex
	mu          sync.Mutex
}

type keptPair struct{ keeper, fallen string }

type forgivenPair struct{ forgiver, subject string }

// NewServer builds a transport server for the given world and character store.
func NewServer(w *world.World, st store.CharStore, gh grid.Hub, admins []string, log *slog.Logger) *Server {
	if gh == nil {
		gh = grid.NewLocalHub(w.Name, w.URL)
	}
	adm := map[string]bool{}
	for _, a := range admins {
		a = strings.TrimSpace(strings.ToLower(a))
		if a != "" {
			adm[a] = true
		}
	}
	return &Server{
		world: w, store: st, grid: gh, hub: NewHub(), log: log,
		admins: adm, caches: map[string]int{}, localTraces: map[string][]grid.EchoTrace{},
		forgiven: map[forgivenPair]bool{},
		cages:    map[string]int64{}, saved: map[string][]string{},
		deeds: map[string]map[string]int{}, kept: map[keptPair]bool{},
		deadMobs: map[string]pendingRespawn{}, mobSlainAt: map[string]int64{},
		ground: map[string]map[string]int{},
	}
}

func (s *Server) isAdmin(name string) bool {
	return s.admins[strings.ToLower(name)]
}

// tide reads the collective war tide, caching the last good value when the hub
// is reachable (mirrors upstream lastTide on the world DO).
func (s *Server) tide(ctx context.Context) (int, error) {
	t, err := s.grid.Tide(ctx)
	if err == nil {
		s.mu.Lock()
		s.lastTide = t
		s.mu.Unlock()
		return t, nil
	}
	s.mu.Lock()
	cached := s.lastTide
	s.mu.Unlock()
	return cached, nil
}

func (s *Server) cachedTide() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.lastTide
}

func (s *Server) cacheGold(room string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.caches[room]
}

func (s *Server) addCache(room string, amount int) {
	s.mu.Lock()
	s.caches[room] += amount
	s.mu.Unlock()
}

func (s *Server) takeCache(room string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	g := s.caches[room]
	s.caches[room] = 0
	return g
}

func (s *Server) hasForgiven(forgiver, subject string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.forgiven[forgivenPair{forgiver, subject}]
}

func (s *Server) markForgiven(forgiver, subject string) {
	s.mu.Lock()
	s.forgiven[forgivenPair{forgiver, subject}] = true
	s.mu.Unlock()
}

const cageRefillMs = 4 * 60 * 1000

func (s *Server) cagesReady(room string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	at := s.cages[room]
	return at == 0 || time.Now().UnixMilli() >= at
}

func (s *Server) setCageRefill(room string) {
	s.mu.Lock()
	s.cages[room] = time.Now().UnixMilli() + cageRefillMs
	s.mu.Unlock()
}

func (s *Server) rememberSaved(player string, names ...string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.saved[player] = append(names, s.saved[player]...)
	if len(s.saved[player]) > 24 {
		s.saved[player] = s.saved[player][:24]
	}
}

func (s *Server) savedSouls(player string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.saved[player]...)
}

func (s *Server) addDeed(player, kind string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.deeds[player] == nil {
		s.deeds[player] = map[string]int{}
	}
	s.deeds[player][kind]++
}

func (s *Server) deedsFor(player string) map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	src := s.deeds[player]
	out := map[string]int{}
	for k, v := range src {
		out[k] = v
	}
	return out
}

func (s *Server) hasKept(keeper, fallen string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.kept[keptPair{keeper, fallen}]
}

func (s *Server) markKept(keeper, fallen string) {
	s.mu.Lock()
	s.kept[keptPair{keeper, fallen}] = true
	s.mu.Unlock()
}

func (s *Server) persistPlayer(p *world.Player) {
	if p == nil || p.Name == "" {
		return
	}
	_ = s.store.Commit(p.Name, p.Sheet())
}

// Handler returns the world's HTTP handler (/ws, /health, /health/deep, /map.svg, play page).
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /health", s.health)
	mux.HandleFunc("GET /health/deep", s.healthDeep)
	mux.HandleFunc("GET /map.svg", s.mapSVG)
	mux.HandleFunc("/ws", s.ws)
	mux.HandleFunc("/", s.playPage)
	return mux
}

// playPage serves the browser xterm client for any unmatched GET (TS parity).
func (s *Server) playPage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write([]byte(playPage(s.world.Name)))
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
	hubOK := true
	hubLatency := 0
	if s.grid.Remote() {
		start := time.Now()
		if err := gridPing(r.Context(), s.grid); err != nil {
			hubOK = false
		}
		hubLatency = int(time.Since(start).Milliseconds())
	}
	checks := map[string]any{
		"world":    map[string]any{"ok": worldOK, "latency_ms": 0, "critical": true},
		"grid_hub": map[string]any{"ok": hubOK, "latency_ms": hubLatency, "critical": false},
	}
	code := http.StatusOK
	if !worldOK {
		code = http.StatusServiceUnavailable
	}
	writeJSON(w, code, map[string]any{
		"ok": worldOK, "ts": time.Now().UnixMilli(), "world": s.world.Name, "checks": checks,
	})
}

func gridPing(ctx context.Context, h grid.Hub) error {
	if rh, ok := h.(*grid.RemoteHub); ok {
		return rh.Ping(ctx)
	}
	_, err := h.Tide(ctx)
	return err
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
	handleConn(r.Context(), c, s)
}

func (s *Server) mapSVG(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "image/svg+xml; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(mapSVG))
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}
