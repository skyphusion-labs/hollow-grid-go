package transport

import (
	"strings"
	"sync"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

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
}

// NewHub builds an empty session registry.
func NewHub() *Hub {
	return &Hub{players: map[string]*livePlayer{}}
}

// Register adds a player and returns their outbound push channel.
func (h *Hub) Register(p *world.Player) chan string {
	ch := make(chan string, 64)
	lp := &livePlayer{
		name: p.Name, room: p.RoomID, title: p.Title,
		faction: p.Faction, race: p.Race, ashsworn: p.Ashsworn, morality: p.Morality,
		hp: p.HP, maxHP: p.MaxHP,
		push: ch, plr: p,
	}
	h.mu.Lock()
	h.players[p.Name] = lp
	h.mu.Unlock()
	return ch
}

// Unregister removes a player from the registry.
func (h *Hub) Unregister(name string) {
	h.mu.Lock()
	delete(h.players, name)
	h.mu.Unlock()
}

// Sync updates cached fields used for presence/branding after local changes.
func (h *Hub) Sync(p *world.Player) {
	h.mu.Lock()
	if lp, ok := h.players[p.Name]; ok {
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
	h.mu.Lock()
	if lp, ok := h.players[name]; ok {
		lp.replyTo = from
	}
	h.mu.Unlock()
}

// ReplyTo returns the last private messenger, if any.
func (h *Hub) ReplyTo(name string) string {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if lp, ok := h.players[name]; ok {
		return lp.replyTo
	}
	return ""
}

// Find returns a live player by name (case-insensitive).
func (h *Hub) Find(name string) (*livePlayer, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	if lp, ok := h.players[name]; ok {
		return lp, true
	}
	lower := strings.ToLower(strings.TrimSpace(name))
	for n, lp := range h.players {
		if strings.ToLower(n) == lower {
			return lp, true
		}
	}
	return nil, false
}

// PlayersInRoom lists others in the same room for room.info.
func (h *Hub) PlayersInRoom(room, except string) []world.PlayerRef {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]world.PlayerRef, 0, 4)
	for name, lp := range h.players {
		if name == except || lp.room != room {
			continue
		}
		out = append(out, world.PlayerRef{Name: name, Standing: brandLive(lp)})
	}
	return out
}

// All returns every connected player snapshot for who/presence.
func (h *Hub) All() []*livePlayer {
	h.mu.RLock()
	defer h.mu.RUnlock()
	out := make([]*livePlayer, 0, len(h.players))
	for _, lp := range h.players {
		out = append(out, lp)
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
	h.mu.RLock()
	lp, ok := h.players[name]
	h.mu.RUnlock()
	if !ok {
		return
	}
	deliver(lp, text)
}

// deliver enqueues outbound text for a live player. Blocks until the session
// reads it so tell/reply and gridcasts are never dropped on a full buffer.
func deliver(lp *livePlayer, text string) {
	lp.push <- text
}

// BroadcastRoom sends prose to everyone in a room except skip (if non-empty).
func (h *Hub) BroadcastRoom(room, text, skip string) {
	h.mu.RLock()
	targets := make([]*livePlayer, 0, 4)
	for name, lp := range h.players {
		if lp.room == room && name != skip {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		deliver(lp, text)
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
		deliver(lp, text)
	}
}

// BroadcastAllExcept sends prose to every player except skip.
func (h *Hub) BroadcastAllExcept(text, skip string) {
	h.mu.RLock()
	targets := make([]*livePlayer, 0, len(h.players))
	for name, lp := range h.players {
		if name != skip {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		deliver(lp, text)
	}
}
