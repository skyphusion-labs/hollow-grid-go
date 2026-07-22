package transport

import (
	"strings"
	"sync"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func canonicalPlayerName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// livePlayer is one connected character tracked for multiplayer presence.
type livePlayer struct {
	name     string
	room     string
	title    string
	faction  string
	race     string
	ashsworn bool
	morality int
	hp       int
	maxHP    int
	push     chan string
	replyTo  string
	plr      *world.Player
}

// Hub tracks live sessions and routes room/global messages between them.
type Hub struct {
	mu      sync.RWMutex
	players map[string]*livePlayer
	pending map[string]struct{}
}

// NewHub builds an empty session registry.
func NewHub() *Hub {
	return &Hub{players: map[string]*livePlayer{}, pending: map[string]struct{}{}}
}

// TryReserve holds a name during login before Register.
func (h *Hub) TryReserve(name string) bool {
	key := canonicalPlayerName(name)
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.players[key]; ok {
		return false
	}
	if _, ok := h.pending[key]; ok {
		return false
	}
	h.pending[key] = struct{}{}
	return true
}

// Release drops a login reservation when auth fails before Register.
func (h *Hub) Release(name string) {
	key := canonicalPlayerName(name)
	h.mu.Lock()
	delete(h.pending, key)
	h.mu.Unlock()
}

// Register adds a player and returns their outbound push channel.
// Returns nil,false when the name is already connected elsewhere on this world.
func (h *Hub) Register(p *world.Player) (chan string, bool) {
	key := canonicalPlayerName(p.Name)
	ch := make(chan string, 256)
	lp := &livePlayer{
		name: p.Name, room: p.RoomID, title: p.Title,
		faction: p.Faction, race: p.Race, ashsworn: p.Ashsworn, morality: p.Morality,
		hp: p.HP, maxHP: p.MaxHP,
		push: ch, plr: p,
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.pending, key)
	if _, ok := h.players[key]; ok {
		return nil, false
	}
	h.players[key] = lp
	return ch, true
}

// Unregister removes a player from the registry.
func (h *Hub) Unregister(name string) {
	key := canonicalPlayerName(name)
	h.mu.Lock()
	delete(h.players, key)
	h.mu.Unlock()
}

// Sync updates cached fields used for presence/branding after local changes.
func (h *Hub) Sync(p *world.Player) {
	key := canonicalPlayerName(p.Name)
	h.mu.Lock()
	if lp, ok := h.players[key]; ok {
		lp.room = p.RoomID
		lp.title = p.Title
		lp.faction = p.Faction
		lp.race = p.Race
		lp.ashsworn = p.Ashsworn
		lp.morality = p.Morality
		lp.hp = p.HP
		lp.maxHP = p.MaxHP
	}
	h.mu.Unlock()
}

// SetReplyTo remembers who last told this player (for reply).
func (h *Hub) SetReplyTo(name, from string) {
	key := canonicalPlayerName(name)
	h.mu.Lock()
	if lp, ok := h.players[key]; ok {
		lp.replyTo = from
	}
	h.mu.Unlock()
}

// ReplyTo returns the last private messenger, if any.
func (h *Hub) ReplyTo(name string) string {
	key := canonicalPlayerName(name)
	h.mu.RLock()
	defer h.mu.RUnlock()
	if lp, ok := h.players[key]; ok {
		return lp.replyTo
	}
	return ""
}

// Find returns a live player by name (case-insensitive).
func (h *Hub) Find(name string) (*livePlayer, bool) {
	key := canonicalPlayerName(name)
	h.mu.RLock()
	defer h.mu.RUnlock()
	lp, ok := h.players[key]
	return lp, ok
}

// PlayersInRoom lists others in the same room for room.info.
func (h *Hub) PlayersInRoom(room, except string) []world.PlayerRef {
	exceptKey := canonicalPlayerName(except)
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]world.PlayerRef, 0, 4)
	for key, lp := range h.players {
		if key == exceptKey || lp.room != room {
			continue
		}
		out = append(out, world.PlayerRef{Name: lp.name, Standing: brandLive(lp)})
	}
	return out
}

// presenceSnap is a point-in-time copy of hub fields used for who/presence.
type presenceSnap struct {
	name   string
	title  string
	regard string
}

// HasPlayers reports whether any sessions are registered.
func (h *Hub) HasPlayers() bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.players) > 0
}

// PlayerNames returns connected player names (snapshot under hub lock).
func (h *Hub) PlayerNames() []string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]string, 0, len(h.players))
	for name := range h.players {
		out = append(out, name)
	}
	return out
}

// PresenceSnapshots returns who/presence fields copied under hub lock.
func (h *Hub) PresenceSnapshots() []presenceSnap {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]presenceSnap, 0, len(h.players))
	for _, lp := range h.players {
		out = append(out, presenceSnap{
			name: lp.name, title: lp.title, regard: brandLive(lp),
		})
	}
	return out
}

func brandLive(lp *livePlayer) string {
	p := &world.Player{
		Name: lp.name, Faction: lp.faction, Morality: lp.morality,
		Ashsworn: lp.ashsworn,
	}
	return world.Brand(p)
}

// FindPrefix returns a live player whose name starts with prefix (case-insensitive).
func (h *Hub) FindPrefix(prefix string) (*livePlayer, bool) {
	prefix = strings.ToLower(strings.TrimSpace(prefix))
	h.mu.RLock()
	defer h.mu.RUnlock()
	for name, lp := range h.players {
		if strings.HasPrefix(strings.ToLower(name), prefix) {
			return lp, true
		}
	}
	return nil, false
}

func (h *Hub) push(name, text string) {
	lp, ok := h.Find(name)
	if !ok {
		return
	}
	pushBestEffort(lp, text)
}

// PushReliable retries until the session reads the message or times out.
// Used for tell/reply so private comms are not dropped on a full buffer.
func (h *Hub) PushReliable(name, text string) {
	lp, ok := h.Find(name)
	if !ok {
		return
	}
	pushReliable(lp, text)
}

// pushReliable retries a direct push until the session reads it or times out.
// Used for tell/reply so comms are not dropped when the buffer is momentarily full.
func pushReliable(lp *livePlayer, text string) {
	deadline := time.Now().Add(5 * time.Second)
	for {
		select {
		case lp.push <- text:
			return
		default:
			if time.Now().After(deadline) {
				return
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
}

// pushBestEffort enqueues text without blocking the caller (drops if full).
func pushBestEffort(lp *livePlayer, text string) {
	select {
	case lp.push <- text:
	default:
	}
}

// PushReliableRoom sends prose to everyone in a room except skip, retrying until
// each session reads it or times out (used for emotes and other room-visible comms).
func (h *Hub) PushReliableRoom(room, text, skip string) {
	skipKey := canonicalPlayerName(skip)
	h.mu.RLock()
	targets := make([]*livePlayer, 0, 4)
	for key, lp := range h.players {
		if lp.room == room && key != skipKey {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		pushReliable(lp, text)
	}
}

// BroadcastRoom sends prose to everyone in a room except skip (if non-empty).
func (h *Hub) BroadcastRoom(room, text, skip string) {
	skipKey := canonicalPlayerName(skip)
	h.mu.RLock()
	targets := make([]*livePlayer, 0, 4)
	for key, lp := range h.players {
		if lp.room == room && key != skipKey {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		pushBestEffort(lp, text)
	}
}

// BroadcastAll sends prose to every connected player.
func (h *Hub) BroadcastAll(text string) {
	h.mu.RLock()
	targets := make([]*livePlayer, 0, len(h.players))
	for _, lp := range h.players {
		targets = append(targets, lp)
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		pushBestEffort(lp, text)
	}
}

// BroadcastAllExcept sends prose to every player except skip.
func (h *Hub) BroadcastAllExcept(text, skip string) {
	skipKey := canonicalPlayerName(skip)
	h.mu.RLock()
	targets := make([]*livePlayer, 0, len(h.players))
	for key, lp := range h.players {
		if key != skipKey {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		pushBestEffort(lp, text)
	}
}
