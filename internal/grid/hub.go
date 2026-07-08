// Package grid is the federation seam: a thin client for the shared Grid Hub.
// Federation never blocks play; when the hub is unreachable the world runs on
// local memory and FileStore alone.
package grid

import (
	"context"
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

// Hub is the GridHubApi surface worlds call over the federation backend.
type Hub interface {
	Record(ctx context.Context, world, node, kind, text string, at int64) error
	RecentAcross(ctx context.Context, world string, limit int) ([]Trace, error)
	Tide(ctx context.Context) (int, error)
	ShiftTide(ctx context.Context, delta int) (int, error)
	LoadCharacter(ctx context.Context, name string) (CharSheet, bool, error)
	CommitCharacter(ctx context.Context, name string, sheet CharSheet) error
	Register(ctx context.Context, world, url string) error
	ListWorlds(ctx context.Context) ([]WorldInfo, error)
	ReportPresence(ctx context.Context, world string, entries []PresenceEntry, at int64) error
	Presence(ctx context.Context, maxAgeMs int64) ([]Presence, error)
	RecordRescued(ctx context.Context, world, name, savedBy string, at int64) error
	RecentRescued(ctx context.Context, limit int) ([]Rescued, error)
	RecordFallen(ctx context.Context, world, name, room string, at int64) error
	RecentFallen(ctx context.Context, limit int) ([]Fallen, error)
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
	worldName string
	worldURL  string
	traces    []Trace
	local     map[string][]EchoTrace
	rescued   []Rescued
	fallen    []Fallen
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
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	h.traces = append([]Trace{{World: world, Node: node, Kind: kind, Text: text, At: at}}, h.traces...)
	if len(h.traces) > 200 {
		h.traces = h.traces[:200]
	}
	return nil
}

func (h *LocalHub) RecordLocal(node, kind, text string) {
	rows := h.local[node]
	rows = append([]EchoTrace{{At: time.Now().UnixMilli(), Kind: kind, Text: text}}, rows...)
	if len(rows) > 50 {
		rows = rows[:50]
	}
	h.local[node] = rows
}

func (h *LocalHub) LocalTraces(node string, limit int) []EchoTrace {
	rows := h.local[node]
	if limit <= 0 || limit > len(rows) {
		return rows
	}
	return rows[:limit]
}

func (h *LocalHub) RecentAcross(_ context.Context, world string, limit int) ([]Trace, error) {
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
	if limit <= 0 || limit > len(h.traces) {
		return h.traces
	}
	return h.traces[:limit]
}

func (h *LocalHub) Tide(context.Context) (int, error) { return 0, nil }

func (h *LocalHub) ShiftTide(_ context.Context, delta int) (int, error) { return delta, nil }

func (h *LocalHub) LoadCharacter(context.Context, string) (CharSheet, bool, error) {
	return CharSheet{}, false, nil
}

func (h *LocalHub) CommitCharacter(context.Context, string, CharSheet) error { return nil }

func (h *LocalHub) Register(_ context.Context, world, url string) error {
	for i, t := range h.traces {
		if t.World == world {
			h.traces[i].Node = url
			return nil
		}
	}
	h.traces = append([]Trace{{World: world, Node: url, Kind: "register", Text: "a new node joined the network.", At: time.Now().UnixMilli()}}, h.traces...)
	return nil
}

func (h *LocalHub) ListWorlds(context.Context) ([]WorldInfo, error) {
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
	if at == 0 {
		at = time.Now().UnixMilli()
	}
	h.rescued = append([]Rescued{{World: world, Name: name, SavedBy: savedBy, At: at}}, h.rescued...)
	if len(h.rescued) > 200 {
		h.rescued = h.rescued[:200]
	}
	return h.Record(context.Background(), world, "rescued", "rescue", name+" freed by "+savedBy, at)
}

func (h *LocalHub) RecentRescued(_ context.Context, limit int) ([]Rescued, error) {
	if limit <= 0 || limit > len(h.rescued) {
		return h.rescued, nil
	}
	return h.rescued[:limit], nil
}

func (h *LocalHub) RecordFallen(_ context.Context, world, name, room string, at int64) error {
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
	if len(h.fallen) == 0 {
		return []Fallen{}, nil
	}
	if limit <= 0 || limit > len(h.fallen) {
		return h.fallen, nil
	}
	return h.fallen[:limit], nil
}
