package transport

import (
	"context"
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

// session is one player connection. It buffers output and flushes a whole
// command's response (prose + @event lines) in a single WebSocket message.
type session struct {
	c      *websocket.Conn
	w      *world.World
	player *world.Player
	out    strings.Builder
	log    *slog.Logger
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
	s := &session{c: c, w: w, log: log}

	s.line(banner)
	s.line("By what name are you known, wanderer?")
	if err := s.flush(ctx); err != nil {
		return
	}

	// The first line a new connection sends is the character name. (Race menu
	// for brand-new characters arrives in Phase 1.)
	name, err := s.read(ctx)
	if err != nil || name == "" {
		return
	}
	s.player = world.NewPlayer(name, w.Start().ID)
	log.Info("player connected", "name", name, "world", w.Name)

	s.line("The Grid remembers you, " + name + ".")
	s.sendRoom()
	s.sendVitals()
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

func (s *session) room() *world.Room { return s.w.Room(s.player.RoomID) }

func (s *session) sendRoom() {
	r := s.room()
	s.line("")
	s.line(r.Name)
	s.line(r.Desc)
	s.event(event.RoomInfo, r.Info())
}

func (s *session) sendVitals() { s.event(event.CharVitals, s.player.Vitals()) }

// handle runs one command line; returns true if the session should close.
func (s *session) handle(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	switch verb := strings.ToLower(fields[0]); verb {
	case "quit", "q":
		s.line("The Grid goes quiet. Farewell.")
		return true
	case "look", "l":
		s.sendRoom()
	case "help", "h", "?":
		s.line("Commands: look, <direction>, help, quit. (More as the port grows.)")
	default:
		if dest, ok := s.room().Exits[verb]; ok {
			s.player.RoomID = dest
			s.sendRoom()
			s.sendVitals()
		} else {
			// No-silent-no-op: an unusable exit/command says so.
			s.line("You can't do that here. (Try: look, help.)")
		}
	}
	return false
}
