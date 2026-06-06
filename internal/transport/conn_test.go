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

// TestLoginRoomMove drives the player transport end to end: connect, get the
// name prompt, log in, receive the start room + vitals on the @event channel,
// and move between rooms. This is the conformance check for protocol.md s1+s2.
func TestLoginRoomMove(t *testing.T) {
	srv := NewServer(world.New("Test World", ""), slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	defer ts.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.CloseNow()

	read := func() string {
		_, data, err := c.Read(ctx)
		if err != nil {
			t.Fatalf("read: %v", err)
		}
		return string(data)
	}
	send := func(s string) {
		if err := c.Write(ctx, websocket.MessageText, []byte(s)); err != nil {
			t.Fatalf("write: %v", err)
		}
	}

	if m := read(); !strings.Contains(m, "wanderer") {
		t.Fatalf("expected name prompt, got %q", m)
	}

	send("Tester")
	m := read()
	for _, want := range []string{"The Grid Gate", "@event room.info", "grid-gate", "@event char.vitals"} {
		if !strings.Contains(m, want) {
			t.Fatalf("login response missing %q in %q", want, m)
		}
	}

	send("north")
	if m := read(); !strings.Contains(m, "ash-road") {
		t.Fatalf("move north did not reach ash-road: %q", m)
	}

	send("look")
	if m := read(); !strings.Contains(m, "Ash Road") {
		t.Fatalf("look did not show Ash Road: %q", m)
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
