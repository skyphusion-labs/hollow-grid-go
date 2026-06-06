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

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// newWorldServer stands up a world with a fresh temp-dir character store.
func newWorldServer(t *testing.T) *httptest.Server {
	t.Helper()
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	srv := NewServer(world.New("Test World", ""), st, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	// Drain in-flight sessions (and their final persists) before the temp dir is
	// removed, so a disconnect-time write never races cleanup.
	t.Cleanup(func() {
		ts.Close()
		srv.Wait()
	})
	return ts
}

// dial opens a player WebSocket and returns read/send/close helpers.
func dial(t *testing.T, ts *httptest.Server) (read func() string, send func(string), closeConn func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	wsURL := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws"
	c, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
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
	closeConn = func() { c.CloseNow(); cancel() }
	return read, send, closeConn
}

func mustContain(t *testing.T, where, got string, wants ...string) {
	t.Helper()
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Fatalf("%s: missing %q in %q", where, w, got)
		}
	}
}

// TestLoginRaceAndMove drives the login flow with the race menu, then movement
// and the perception frame (protocol.md s1+s2).
func TestLoginRaceAndMove(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	mustContain(t, "name prompt", read(), "wanderer")
	send("Tester")
	mustContain(t, "race menu", read(), "choose what you are", "Ashborn", "Revenant")
	send("Revenant")
	mustContain(t, "entry scene", read(),
		"The Grid Gate", "@event room.info", "grid-gate",
		"@event char.vitals", "@event char.affects", `"race":"Revenant"`, "@event room.actions")
	send("north")
	mustContain(t, "ash road", read(), "ash-road")
	send("look")
	mustContain(t, "look", read(), "Ash Road")
}

// TestMoralChoiceIsRememberedAndLegible: a moral action carries a valence,
// taking it shifts char.affects, and the world remembers (it leaves the set).
func TestMoralChoiceIsRememberedAndLegible(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Tester")
	read()
	send("Ashborn")
	read()
	send("north")
	read()
	send("north")
	mustContain(t, "checkpoint affordances", read(),
		`"verb":"defend"`, `"kind":"moral"`, `"valence":"virtuous"`,
		`"verb":"join"`, `"valence":"corrupt"`)

	send("defend")
	afterDefend := read()
	mustContain(t, "defend consequence", afterDefend, "@event char.affects", `"morality":10`, "refugees")
	if strings.Contains(afterDefend, `"verb":"defend"`) || strings.Contains(afterDefend, `"verb":"join"`) {
		t.Fatalf("resolved choice still offered in room.actions: %q", afterDefend)
	}
}

// TestResumePersistsTheCharacter: a returning name resumes its persisted
// CharSheet, skips the race menu, and carries its standing. "The world remembers
// you" made literal.
func TestResumePersistsTheCharacter(t *testing.T) {
	ts := newWorldServer(t)

	read, send, done := dial(t, ts)
	read()
	send("Mara")
	read() // race menu (new)
	send("Revenant")
	read() // entry scene
	send("north")
	read()
	send("north")
	read() // at the checkpoint
	send("defend")
	mustContain(t, "defend persisted", read(), `"morality":10`)
	done()

	read2, send2, done2 := dial(t, ts)
	defer done2()
	read2()
	send2("Mara")
	resumed := read2()
	mustContain(t, "resume", resumed, "remembers you", `"race":"Revenant"`, `"morality":10`)
	if strings.Contains(resumed, "choose what you are") {
		t.Fatalf("resume should skip the race menu: %q", resumed)
	}
}

// TestWhoamiEmitsIdentity: whoami emits char.identity carrying the CharSheet.
func TestWhoamiEmitsIdentity(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Wren")
	read()
	send("Hollow")
	read()
	send("whoami")
	mustContain(t, "whoami", read(), "@event char.identity", `"race":"Hollow"`)
}

// TestHealth checks the liveness probe shape (protocol.md s1).
func TestHealth(t *testing.T) {
	ts := newWorldServer(t)
	resp, err := http.Get(ts.URL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status %d", resp.StatusCode)
	}
}
