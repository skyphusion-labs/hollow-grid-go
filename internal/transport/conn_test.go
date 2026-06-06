package transport

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// dialWorld stands up a world over a real WebSocket and returns helpers to read
// server messages and send client lines.
func dialWorld(t *testing.T) (read func() string, send func(string), done func()) {
	t.Helper()
	srv := NewServer(world.New("Test World", ""), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		ts.Close()
		cancel()
		t.Fatalf("dial: %v", err)
	}
	read = func() string {
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return string(data)
	}
	send = func(s string) {
		if err := c.Write(ctx, websocket.MessageText, []byte(s)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	done = func() { c.CloseNow(); ts.Close(); cancel() }
	return read, send, done
}

func mustContain(t *testing.T, where, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("%s: missing %q in %q", where, w, got)
		}
	}
}

// TestLoginRaceAndMove drives the transport through the full login flow with the
// race menu, then movement and the perception frame (protocol.md s1+s2).
func TestLoginRaceAndMove(t *testing.T) {
	read, send, done := dialWorld(t)
	defer done()

	mustContain(t, "name prompt", read(), "wanderer")

	send("Tester")
	mustContain(t, "race menu", read(), "choose what you are", "Ashborn", "Revenant")

	send("Revenant")
	mustContain(t, "entry scene", read(),
		"The Grid Gate", "@event room.info", "grid-gate",
		"@event char.vitals", "@event char.affects", `"race":"Revenant"`,
		"@event room.actions")

	send("north")
	mustContain(t, "ash road", read(), "ash-road")

	send("look")
	mustContain(t, "look", read(), "Ash Road")
}

// TestMoralChoiceIsRememberedAndLegible verifies the heart of Phase 1: a moral
// action carries a valence in room.actions, taking it shifts char.affects, and
// the world remembers the choice (the action leaves the affordance set).
func TestMoralChoiceIsRememberedAndLegible(t *testing.T) {
	read, send, done := dialWorld(t)
	defer done()

	read()          // name prompt
	send("Tester")  //
	read()          // race menu
	send("Ashborn") //
	read()          // entry scene at the Grid Gate

	send("north") // ash-road
	read()
	send("north") // cinder-checkpoint
	atCheckpoint := read()
	mustContain(t, "checkpoint affordances", atCheckpoint,
		`"verb":"defend"`, `"kind":"moral"`, `"valence":"virtuous"`,
		`"verb":"join"`, `"valence":"corrupt"`)

	send("defend")
	afterDefend := read()
	// Standing moved, and it is on the structured channel as data.
	mustContain(t, "defend consequence", afterDefend, "@event char.affects", `"morality":10`)
	mustContain(t, "defend prose", afterDefend, "refugees")
	// The world remembers: the resolved choice is gone from room.actions, and so
	// is its mutually-exclusive opposite.
	if strings.Contains(afterDefend, `"verb":"defend"`) || strings.Contains(afterDefend, `"verb":"join"`) {
		t.Fatalf("resolved choice still offered in room.actions: %q", afterDefend)
	}

	// A fresh look confirms the choice stuck across observations.
	send("look")
	relook := read()
	if strings.Contains(relook, `"verb":"defend"`) {
		t.Fatalf("defend reappeared on re-look: %q", relook)
	}
}

// TestHealth checks the liveness probe shape (protocol.md s1).
func TestHealth(t *testing.T) {
	srv := NewServer(world.New("Test World", ""), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status %d", resp.StatusCode)
	}
}
