// Package grid is the federation seam: a thin client for the shared Grid Hub.
// Federation never blocks play; when the hub is unreachable the world runs on
// local memory and FileStore alone.
package grid

import (
	"context"
	"sort"
	"sync"
	"time"
)

// Trace is one notable event in the shared Grid memory.
type Trace struct {
	World string `json:"world"`
	Node  string `json:"node"`
	Kind  string `json:"kind"`
	Text  string `json:"text"`
	At    int64  `json:"at"`
}

// EchoTrace is a local node memory row (grid.echo).
type EchoTrace struct {
	At   int64  `json:"at"`
	Kind string `json:"kind"`
	Text string `json:"text"`
}

// CharSheet mirrors the canonical federated character (shared/grid.ts).
type CharSheet struct {
	Level    int    `json:"level"`
	XP       int    `json:"xp"`
	Gold     int    `json:"gold"`
	Faction  string `json:"faction"`
	Morality int    `json:"morality"`
	Title    string `json:"title"`
	Race     string `json:"race"`
	Ashsworn bool   `json:"ashsworn"`
}

// Presence is one player reported live to the hub.
type Presence struct {
	World  string `json:"world"`
	Name   string `json:"name"`
	Regard string `json:"regard"`
	Title  string `json:"title"`
	Here   bool   `json:"here"`
	At     int64  `json:"at,omitempty"`
}

// WorldInfo is a registered federation destination.
type WorldInfo struct {
	ID       string `json:"id"`
	URL      string `json:"url"`
	LastSeen int64  `json:"last_seen"`
}

// Rescued is one entry on the cross-world rescued roll.
type Rescued struct {
	World   string `json:"world"`
	Name    string `json:"name"`
	SavedBy string `json:"savedBy"`
	At      int64  `json:"at"`
}

// Fallen is one entry on the memorial roll.
type Fallen struct {
	World string `json:"world"`
	Name  string `json:"name"`
	Room  string `json:"room"`
	At    int64  `json:"at"`
}

// Cast is one cross-world chat line from the shared hub feed.
type Cast struct {
	ID     int    `json:"id"`
	World  string `json:"world"`
	Sender string `json:"sender"`
	Text   string `json:"text"`
}

// LedgerKind is one row from ledgerStats().
type LedgerKind struct {
	Kind  string `json:"kind"`
	Count int    `json:"count"`
}

// PruneResult is the return value of pruneLedgerKinds().
type PruneResult struct {
	Removed int `json:"removed"`
}

// Hub is the GridHubApi surface worlds call over the federation backend.
type Hub interface {
	Record(ctx context.Context, world, node, kind, text string, at int64) error
	RecentAcross(ctx context.Context, world string, limit int) ([]Trace, error)
	Tide(ctx context.Context) (int, error)
	ShiftTide(ctx context.Context, delta int) (int, error)
	LoadCharacter(ctx context.Context, name string) (CharSheet, bool, error)
	CommitCharacter(ctx context.Context, name string, sheet CharSheet) error
	ClaimCharacterLease(ctx context.Context, name string) error
	Register(ctx context.Context, world, url string) error
	ListWorlds(ctx context.Context) ([]WorldInfo, error)
	GridCast(ctx context.Context, world, sender, text string) error
	CastsSince(ctx context.Context, sinceID, limit int) ([]Cast, error)
	LedgerStats(ctx context.Context) ([]LedgerKind, error)
	PruneLedgerKinds(ctx context.Context, kinds []string) (PruneResult, error)
	ReportPresence(ctx context.Context, world string, entries []PresenceEntry, at int64) error
	Presence(ctx context.Context, maxAgeMs int64) ([]Presence, error)
	RecordRescued(ctx context.Context, world, name, savedBy string, at int64) error
	RecentRescued(ctx context.Context, limit int) ([]Rescued, error)
	RecordFallen(ctx context.Context, world, name, room string, at int64) error
	RecentFallen(ctx context.Context, limit int) ([]Fallen, error)
	Remote() bool
}

// PresenceEntry is one player in a presence heartbeat.
type PresenceEntry struct {
	Name   string
	Regard string
	Title  string
}

// LocalHub is the standalone fallback: seeded federation echoes and an in-process
// ledger so ping/listen work without the Cloudflare hub binding.
type LocalHub struct {
	mu         sync.Mutex
	worldName  string
	worldURL   string
	traces     []Trace
	local      map[string][]EchoTrace
	rescued    []Rescued
	fallen     []Fallen
	tide       int
	casts      []Cast
	nextCastID int
}

// NewLocalHub builds a hub that satisfies federation-shaped calls offline.
func NewLocalHub(worldName, worldURL string) *LocalHub {
	h := &LocalHub{
		worldName: worldName,
		worldURL:  worldURL,
		local:     map[string][]EchoTrace{},
	}
	h.traces = []Trace{
		{World: "Saltreach", Node: "the drowned pier", Kind: "death", Text: "a runner called Mox bled out, cursing the tide.", At: 0},
		{World: "the Ninth Server", Node: "cell block C", Kind: "oath", Text: "someone swore off the dust for the ninth time.", At: 0},
		{World: "Dustfall", Node: "the long market", Kind: "slain", Text: "a trader put down a chrome-jackal with a length of pipe.", At: 0},
	}
	return h
}

func (h *LocalHub) Record(_ context.Context, world, node, kind, text string, at int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.recordTrace(world, node, kind, text, at)
	return nil
}

func (h *LocalHub) recordTrace(world, node, kind, text string, at int64) {
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	h.traces = append([]Trace{{World: world, Node: node, Kind: kind, Text: text, At: at}}, h.traces...)
	if len(h.traces) > 200 {
		h.traces = h.traces[:200]
	}
}

func (h *LocalHub) RecordLocal(node, kind, text string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	rows := h.local[node]
	rows = append([]EchoTrace{{At: time.Now().UnixMilli(), Kind: kind, Text: text}}, rows...)
	if len(rows) > 50 {
		rows = rows[:50]
	}
	h.local[node] = rows
}

func (h *LocalHub) LocalTraces(node string, limit int) []EchoTrace {
	h.mu.Lock()
	defer h.mu.Unlock()
	rows := h.local[node]
	if limit <= 0 || limit > len(rows) {
		return append([]EchoTrace(nil), rows...)
	}
	return append([]EchoTrace(nil), rows[:limit]...)
}

func (h *LocalHub) RecentAcross(_ context.Context, world string, limit int) ([]Trace, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Trace, 0, limit)
	for _, t := range h.traces {
		if t.World == world {
			continue
		}
		out = append(out, t)
		if len(out) >= limit {
			break
		}
	}
	return out, nil
}

func (h *LocalHub) AllTraces(limit int) []Trace {
	h.mu.Lock()
	defer h.mu.Unlock()
	if limit <= 0 || limit > len(h.traces) {
		return append([]Trace(nil), h.traces...)
	}
	return append([]Trace(nil), h.traces[:limit]...)
}

func (h *LocalHub) Tide(context.Context) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.tide, nil
}

func (h *LocalHub) ShiftTide(_ context.Context, delta int) (int, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.tide = clampTide(h.tide + delta)
	return h.tide, nil
}

func clampTide(n int) int {
	if n < -100 {
		return -100
	}
	if n > 100 {
		return 100
	}
	return n
}

func (h *LocalHub) LoadCharacter(context.Context, string) (CharSheet, bool, error) {
	return CharSheet{}, false, nil
}

func (h *LocalHub) CommitCharacter(context.Context, string, CharSheet) error { return nil }

func (h *LocalHub) ClaimCharacterLease(context.Context, string) error { return nil }

func (h *LocalHub) Register(_ context.Context, world, url string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	for i, t := range h.traces {
		if t.World == world {
			h.traces[i].Node = url
			return nil
		}
	}
	h.recordTrace(world, url, "register", "a new node joined the network.", time.Now().UnixMilli())
	return nil
}

func (h *LocalHub) ListWorlds(context.Context) ([]WorldInfo, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := time.Now().UnixMilli()
	return []WorldInfo{
		{ID: "Saltreach", URL: "wss://saltreach.example/ws", LastSeen: 0},
		{ID: "Dustfall", URL: "wss://dustfall.skyphusion.org/ws", LastSeen: now},
		{ID: h.worldName, URL: h.worldURL, LastSeen: now},
	}, nil
}

func (h *LocalHub) ReportPresence(context.Context, string, []PresenceEntry, int64) error {
	return nil
}

func (h *LocalHub) Presence(context.Context, int64) ([]Presence, error) { return nil, nil }

func (h *LocalHub) RecordRescued(_ context.Context, world, name, savedBy string, at int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	h.rescued = append([]Rescued{{World: world, Name: name, SavedBy: savedBy, At: at}}, h.rescued...)
	if len(h.rescued) > 200 {
		h.rescued = h.rescued[:200]
	}
	h.recordTrace(world, "rescued", "rescue", name+" freed by "+savedBy, at)
	return nil
}

func (h *LocalHub) RecentRescued(_ context.Context, limit int) ([]Rescued, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if limit <= 0 || limit > len(h.rescued) {
		return append([]Rescued(nil), h.rescued...), nil
	}
	return append([]Rescued(nil), h.rescued[:limit]...), nil
}

func (h *LocalHub) RecordFallen(_ context.Context, world, name, room string, at int64) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	h.fallen = append([]Fallen{{World: world, Name: name, Room: room, At: at}}, h.fallen...)
	if len(h.fallen) > 200 {
		h.fallen = h.fallen[:200]
	}
	return nil
}

func (h *LocalHub) RecentFallen(_ context.Context, limit int) ([]Fallen, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if len(h.fallen) == 0 {
		return []Fallen{}, nil
	}
	if limit <= 0 || limit > len(h.fallen) {
		return append([]Fallen(nil), h.fallen...), nil
	}
	return append([]Fallen(nil), h.fallen[:limit]...), nil
}

func (h *LocalHub) Remote() bool { return false }

func (h *LocalHub) GridCast(_ context.Context, world, sender, text string) error {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextCastID++
	h.casts = append(h.casts, Cast{ID: h.nextCastID, World: world, Sender: sender, Text: text})
	return nil
}

func (h *LocalHub) CastsSince(_ context.Context, sinceID, limit int) ([]Cast, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]Cast, 0, len(h.casts))
	for _, c := range h.casts {
		if c.ID > sinceID {
			out = append(out, c)
		}
	}
	if limit > 0 && len(out) > limit {
		out = out[:limit]
	}
	return out, nil
}

func (h *LocalHub) LedgerStats(context.Context) ([]LedgerKind, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	counts := map[string]int{}
	for _, t := range h.traces {
		counts[t.Kind]++
	}
	out := make([]LedgerKind, 0, len(counts))
	for kind, count := range counts {
		out = append(out, LedgerKind{Kind: kind, Count: count})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Kind < out[j].Kind })
	return out, nil
}

func (h *LocalHub) PruneLedgerKinds(context.Context, []string) (PruneResult, error) {
	return PruneResult{}, nil
}
