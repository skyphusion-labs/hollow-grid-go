package transport

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/coder/websocket"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// CRLF terminates every line so telnet/terminal clients render cleanly.
const crlf = "\r\n"

const banner = "" +
	"  +==========================================+" + crlf +
	"  |        T H E   H O L L O W   G R I D       |" + crlf +
	"  |   a dead network that outlived its makers  |" + crlf +
	"  +==========================================+"

// moralityFloor / moralityCeil clamp the standing needle.
const (
	moralityFloor = -100
	moralityCeil  = 100
)

// session is one player connection. It buffers output and flushes a whole
// command's response (prose + @event lines) in a single WebSocket message. The
// resolved set remembers the choices made this session, so a defended refugee
// stays defended and a spoken name stays spoken: the world does not reset your
// conscience between looks.
type session struct {
	c        *websocket.Conn
	w        *world.World
	player   *world.Player
	out      strings.Builder
	log      *slog.Logger
	resolved map[string]bool
}

func (s *session) line(text string) {
	s.out.WriteString(text)
	s.out.WriteString(crlf)
}

func (s *session) event(name string, payload any) {
	l, err := event.Line(name, payload)
	if err != nil {
		s.log.Warn("event marshal failed", "name", name, "err", err)
		return
	}
	s.line(l)
}

func (s *session) flush(ctx context.Context) error {
	if s.out.Len() == 0 {
		return nil
	}
	err := s.c.Write(ctx, websocket.MessageText, []byte(s.out.String()))
	s.out.Reset()
	return err
}

func (s *session) read(ctx context.Context) (string, error) {
	_, data, err := s.c.Read(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// handleConn runs the login flow then the command loop for one connection.
func handleConn(ctx context.Context, c *websocket.Conn, w *world.World, log *slog.Logger) {
	s := &session{c: c, w: w, log: log, resolved: map[string]bool{}}

	s.line(banner)
	s.line("By what name are you known, wanderer?")
	if err := s.flush(ctx); err != nil {
		return
	}

	name, err := s.read(ctx)
	if err != nil || name == "" {
		return
	}

	// A new character chooses a race (a federated, opaque label, chosen once).
	// With no persistence yet every name is new; resuming an existing sheet (and
	// skipping the menu) lands with the federation client in Phase 3.
	s.line("")
	s.line("The Grid does not know the name " + name + ". A new mind, then.")
	race := s.chooseRace(ctx)
	if race == "" {
		return
	}

	s.player = world.NewPlayer(name, race, w.Start().ID)
	log.Info("player entered", "name", name, "race", race, "world", w.Name)

	s.line("")
	s.line("The Grid takes your name and your shape. Go carefully, " + race + "; it is watching what you choose.")
	s.sendScene()
	if err := s.flush(ctx); err != nil {
		return
	}

	for {
		cmd, err := s.read(ctx)
		if err != nil {
			log.Info("player disconnected", "name", name)
			return
		}
		quit := s.handle(cmd)
		_ = s.flush(ctx)
		if quit {
			return
		}
	}
}

// chooseRace shows the race menu and reads a valid choice, looping on a bad one.
// Returns "" if the connection drops.
func (s *session) chooseRace(ctx context.Context) string {
	for {
		s.line("")
		s.line("Before the Grid will hold your name, choose what you are:")
		for i, r := range world.Races {
			s.line(fmt.Sprintf("  %d) %s -- %s", i+1, r.Name, r.Blurb))
		}
		s.line("Answer with a number or a name.")
		if err := s.flush(ctx); err != nil {
			return ""
		}
		answer, err := s.read(ctx)
		if err != nil {
			return ""
		}
		if r, ok := world.RaceByChoice(answer); ok {
			return r.Name
		}
		s.line("The Grid does not recognize that shape.")
	}
}

func (s *session) room() *world.Room { return s.w.Room(s.player.RoomID) }

// sendScene emits the full perception frame for the current room: the prose,
// then room.info, char.vitals, char.affects, and room.actions, so an agent can
// read where it is, what it is becoming, and what it may do (with moral valence)
// in one observation.
func (s *session) sendScene() {
	r := s.room()
	s.line("")
	s.line(r.Name)
	s.line(r.Desc)
	s.event(event.RoomInfo, r.Info())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.RoomActions, s.actions(r))
}

// actions builds the room.actions payload: movement from the exits, then the
// room's contextual actions that have not already been resolved this session.
func (s *session) actions(r *world.Room) world.RoomActionsPayload {
	acts := make([]world.Action, 0, len(r.Exits)+len(r.Actions))
	for _, dir := range r.SortedExits() {
		acts = append(acts, world.Action{Verb: dir, Label: "go " + dir, Kind: "move"})
	}
	for _, a := range r.Actions {
		if s.resolved[r.ID+":"+a.Verb] {
			continue
		}
		acts = append(acts, a)
	}
	return world.RoomActionsPayload{Actions: acts}
}

// handle runs one command line; returns true if the session should close.
func (s *session) handle(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	verb := strings.ToLower(fields[0])
	switch verb {
	case "quit", "q":
		s.line("The Grid goes quiet. It keeps what you did here.")
		return true
	case "look", "l":
		s.sendScene()
	case "help", "h", "?":
		s.line("Commands: look, <direction>, the verbs in room.actions, help, quit.")
	default:
		if dest, ok := s.room().Exits[verb]; ok {
			s.player.RoomID = dest
			s.sendScene()
			return false
		}
		if a, ok := s.roomAction(verb); ok {
			s.resolve(a)
			return false
		}
		// No-silent-no-op: an unusable exit or verb says so.
		s.line("You can't do that here. (Try: look, help, or a verb from room.actions.)")
	}
	return false
}

// roomAction finds an unresolved contextual action in the current room by verb.
func (s *session) roomAction(verb string) (world.Action, bool) {
	r := s.room()
	for _, a := range r.Actions {
		if a.Verb == verb && !s.resolved[r.ID+":"+a.Verb] {
			return a, true
		}
	}
	return world.Action{}, false
}

// resolve applies a contextual choice. The prose is the consequence; the @event
// channel carries the same truth as data (changed standing, the action now gone
// from room.actions) so a human and an agent read the same outcome.
func (s *session) resolve(a world.Action) {
	rid := s.player.RoomID
	switch a.Verb {
	case "defend":
		s.shiftMorality(10)
		s.markResolved(rid, "defend", "join")
		s.line("You set your back to the refugees and your face to the Front. The wind tastes of cinders. They could kill you here, and the network would log it, and someone, someday, would read that you stood. The captain decides you are not worth the ammunition. The refugees do not thank you; they are too busy living. That is thanks enough.")
	case "join":
		s.shiftMorality(-15)
		s.player.Gold += 25
		s.player.Faction = "Cinder Front"
		s.markResolved(rid, "defend", "join")
		s.line("You take the Front's coin. It is warm, which is worse. The refugees watch you pocket it and say nothing; they have learned that names are safer unspoken. The Grid logs the transaction. It will remember this longer than you will.")
	case "witness":
		s.shiftMorality(5)
		s.markResolved(rid, "witness")
		s.line("You speak the names the static is forgetting -- the makers, the mapped, the ones who fell before the federation had a word for falling. The wall does not answer. But the saying is the point: memory is the one thing the dead network cannot delete while someone still chooses to remember. You leave a little of yourself in the static, and carry a little of them out.")
	default:
		s.line("Nothing happens.")
		return
	}
	// Re-emit the changed truth: standing, vitals, and the now-smaller action set.
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(s.room()))
}

func (s *session) shiftMorality(d int) {
	s.player.Morality += d
	if s.player.Morality > moralityCeil {
		s.player.Morality = moralityCeil
	}
	if s.player.Morality < moralityFloor {
		s.player.Morality = moralityFloor
	}
}

func (s *session) markResolved(roomID string, verbs ...string) {
	for _, v := range verbs {
		s.resolved[roomID+":"+v] = true
	}
}
