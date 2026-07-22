package grid

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

// RemoteHub calls a Grid Hub over HTTP JSON-RPC (POST /rpc). Federation never
// blocks play: callers treat errors as non-fatal.
type RemoteHub struct {
	url      string
	token    string
	world    string
	worldKey string
	client   *http.Client
}

// NewRemoteHub dials hubURL (the /rpc endpoint, e.g. https://grid-hub.example/rpc).
func NewRemoteHub(hubURL, token, world, worldKey string) *RemoteHub {
	return &RemoteHub{
		url:      strings.TrimRight(strings.TrimSpace(hubURL), "/"),
		token:    strings.TrimSpace(token),
		world:    strings.TrimSpace(world),
		worldKey: strings.TrimSpace(worldKey),
		client: &http.Client{
			Timeout: HubRPCTimeout,
		},
	}
}

func (h *RemoteHub) Remote() bool { return true }

type rpcRequest struct {
	Method string `json:"method"`
	Params []any  `json:"params"`
}

type rpcResponse struct {
	OK     bool            `json:"ok"`
	Result json.RawMessage `json:"result"`
	Error  string          `json:"error"`
}

func (h *RemoteHub) setWorldHeaders(req *http.Request, world string) {
	if world != "" {
		req.Header.Set("X-Grid-World", world)
	}
	if h.worldKey != "" {
		req.Header.Set("X-Grid-World-Key", h.worldKey)
	}
}

func (h *RemoteHub) call(ctx context.Context, method string, params []any, out any, authWorld string) error {
	body, err := json.Marshal(rpcRequest{Method: method, Params: params})
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, h.url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "hollow-grid-go/0.1.0")
	if h.token != "" {
		req.Header.Set("Authorization", "Bearer "+h.token)
	}
	h.setWorldHeaders(req, authWorld)
	resp, err := h.client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	var wrap rpcResponse
	if err := json.Unmarshal(b, &wrap); err != nil {
		return fmt.Errorf("grid rpc: %w", err)
	}
	if !wrap.OK {
		if wrap.Error == "" {
			wrap.Error = resp.Status
		}
		return fmt.Errorf("grid rpc %s: %s", method, wrap.Error)
	}
	if out == nil {
		return nil
	}
	if len(wrap.Result) == 0 || string(wrap.Result) == "null" {
		return nil
	}
	return json.Unmarshal(wrap.Result, out)
}

func (h *RemoteHub) Record(ctx context.Context, world, node, kind, text string, at int64) error {
	return h.call(ctx, "record", []any{world, node, kind, text, at}, nil, "")
}

func (h *RemoteHub) RecentAcross(ctx context.Context, world string, limit int) ([]Trace, error) {
	var out []Trace
	err := h.call(ctx, "recentAcross", []any{world, limit}, &out, "")
	return out, err
}

func (h *RemoteHub) Tide(ctx context.Context) (int, error) {
	var n int
	err := h.call(ctx, "tide", nil, &n, "")
	return n, err
}

func (h *RemoteHub) ShiftTide(ctx context.Context, delta int) (int, error) {
	var n int
	err := h.call(ctx, "shiftTide", []any{delta}, &n, "")
	return n, err
}

func (h *RemoteHub) LoadCharacter(ctx context.Context, name string) (CharSheet, bool, error) {
	var s CharSheet
	err := h.call(ctx, "loadCharacter", []any{name, h.world}, &s, h.world)
	if err != nil {
		return CharSheet{}, false, err
	}
	found := s.Race != "" || s.Level > 1 || s.XP > 0 || s.Faction != "" || s.Morality != 0
	return s, found, nil
}

func (h *RemoteHub) CommitCharacter(ctx context.Context, name string, sheet CharSheet) error {
	return h.call(ctx, "commitCharacter", []any{name, h.world, sheet}, nil, h.world)
}

func (h *RemoteHub) ClaimCharacterLease(ctx context.Context, name string) error {
	return h.call(ctx, "claimCharacterLease", []any{name, h.world}, nil, h.world)
}

func (h *RemoteHub) Register(ctx context.Context, world, url string) error {
	return h.call(ctx, "register", []any{world, url}, nil, world)
}

func (h *RemoteHub) ListWorlds(ctx context.Context) ([]WorldInfo, error) {
	var out []WorldInfo
	err := h.call(ctx, "listWorlds", nil, &out, "")
	return out, err
}

func (h *RemoteHub) GridCast(ctx context.Context, world, sender, text string) error {
	return h.call(ctx, "gridcast", []any{world, sender, text}, nil, "")
}

func (h *RemoteHub) CastsSince(ctx context.Context, sinceID, limit int) ([]Cast, error) {
	var out []Cast
	err := h.call(ctx, "castsSince", []any{sinceID, limit}, &out, "")
	return out, err
}

func (h *RemoteHub) LedgerStats(ctx context.Context) ([]LedgerKind, error) {
	var out []LedgerKind
	err := h.call(ctx, "ledgerStats", nil, &out, "")
	return out, err
}

func (h *RemoteHub) PruneLedgerKinds(ctx context.Context, kinds []string) (PruneResult, error) {
	var out PruneResult
	err := h.call(ctx, "pruneLedgerKinds", []any{kinds}, &out, "")
	return out, err
}

func (h *RemoteHub) ReportPresence(ctx context.Context, world string, entries []PresenceEntry, at int64) error {
	rows := make([]map[string]string, 0, len(entries))
	for _, e := range entries {
		rows = append(rows, map[string]string{"name": e.Name, "regard": e.Regard, "title": e.Title})
	}
	return h.call(ctx, "reportPresence", []any{world, rows, at}, nil, world)
}

func (h *RemoteHub) Presence(ctx context.Context, maxAgeMs int64) ([]Presence, error) {
	var out []Presence
	err := h.call(ctx, "presence", []any{maxAgeMs}, &out, "")
	return out, err
}

func (h *RemoteHub) RecordRescued(ctx context.Context, world, name, savedBy string, at int64) error {
	return h.call(ctx, "recordRescued", []any{world, name, savedBy, at}, nil, "")
}

func (h *RemoteHub) RecentRescued(ctx context.Context, limit int) ([]Rescued, error) {
	var out []Rescued
	err := h.call(ctx, "recentRescued", []any{limit}, &out, "")
	return out, err
}

func (h *RemoteHub) RecordFallen(ctx context.Context, world, name, room string, at int64) error {
	return h.call(ctx, "recordFallen", []any{world, name, room, at}, nil, "")
}

func (h *RemoteHub) RecentFallen(ctx context.Context, limit int) ([]Fallen, error) {
	var out []Fallen
	err := h.call(ctx, "recentFallen", []any{limit}, &out, "")
	return out, err
}

// Ping checks hub reachability (tide probe).
func (h *RemoteHub) Ping(ctx context.Context) error {
	_, err := h.Tide(ctx)
	return err
}
