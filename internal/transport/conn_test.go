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
	// Drain in-flight sessions (and their final persists) before temp-dir cleanup.
	t.Cleanup(func() {
		ts.Close()
		srv.Wait()
	})
	return ts
}

// dial opens a player WebSocket and returns read/send/close helpers.
func dial(t *testing.T, ts *httptest.Server) (read func() string, send func(string), closeConn func()) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
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

// TestLoginRaceMoveAndScene drives the canonical login: name -> race menu ->
// the Cracked Nexus, with the full perception frame and world.state, then a
// move down into the service tunnels (protocol.md s1+s2, reference content).
func TestLoginRaceMoveAndScene(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	mustContain(t, "name prompt", read(), "wanderer")
	send("Tester")
	mustContain(t, "race menu", read(), "choose what you are", "Human", "Revenant")

	send("human")
	entry := read()
	mustContain(t, "entry scene", entry,
		"The Cracked Nexus", "Type 'help'",
		`@event room.info`, `"id":"nexus"`, `"north"`,
		`@event char.vitals`, `"maxHp":30`, `"inCombat":false`,
		`@event char.affects`, `"addiction":0`, `"faction":"none"`, `"race":"human"`,
		`@event room.actions`,
		`@event world.state`, `"phase":"day"`, `"weather":"clear"`)

	send("down")
	mustContain(t, "tunnels", read(), `"id":"tunnels"`, "Service Tunnels")
}

// TestResumePersistsTheCharacter: a returning name resumes its persisted sheet,
// skips the race menu, and carries its race/standing.
func TestResumePersistsTheCharacter(t *testing.T) {
	ts := newWorldServer(t)

	read, send, done := dial(t, ts)
	read()
	send("Mara")
	read() // race menu
	send("revenant")
	read() // entry scene; makeNew persisted the new sheet
	done()

	read2, send2, done2 := dial(t, ts)
	defer done2()
	read2()
	send2("Mara")
	resumed := read2()
	mustContain(t, "resume", resumed, "Welcome back", "Type 'help'", `"race":"revenant"`)
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
	send("ghoul")
	read()
	send("whoami")
	mustContain(t, "whoami", read(), "@event char.identity", `"race":"ghoul"`)
}

// TestEquipmentAndTitle: the starter shiv is in the pack, wield/remove move it
// in and out of the weapon slot on char.equipment, and a title shows in who.
func TestEquipmentAndTitle(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Ash")
	read()
	send("human")
	read() // at the nexus

	send("inventory")
	mustContain(t, "inventory", read(), "rusted shiv")

	send("wield shiv")
	mustContain(t, "wield", read(), "@event char.equipment", `"weapon":"shiv"`)

	send("remove shiv")
	mustContain(t, "remove", read(), "@event char.equipment", `"weapon":null`)

	send("title the Ash-Walker")
	read()
	send("who")
	mustContain(t, "who", read(), "Ash the Ash-Walker")
}

// TestRacialAbilityRequisition: a human's Requisition grants gold, and the
// ability respects its cooldown on an immediate second use.
func TestRacialAbilityRequisition(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Reg")
	read()
	send("human")
	read()

	send("requisition")
	first := read()
	mustContain(t, "requisition", first, "@event char.vitals", "gold")
	if strings.Contains(first, `"gold":0`) {
		t.Fatalf("requisition granted no gold: %q", first)
	}

	send("requisition")
	mustContain(t, "cooldown", read(), "recharging")
}

// TestMobsConsiderLook: the glow-rat shows up in room.info as an object, and
// consider/look <mob> read it (the synchronous half of the combat phase).
func TestMobsConsiderLook(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Hunter")
	read()
	send("human")
	read()

	send("down") // nexus -> tunnels, where the glow-rat lives
	mustContain(t, "tunnels mobs", read(), `"id":"tunnels"`, `"mobs":[{"id":"rat"`)

	send("consider rat")
	mustContain(t, "consider", read(), "sweat") // a glow-rat is weak vs a level-1

	send("look rat")
	mustContain(t, "look rat", read(), "rodent")
}

// TestCombatKillsTheGlowRat: attack starts combat (combat.start + inCombat), the
// fight resolves over combat ticks (combat.round), and the kill ends it
// (combat.end killed, inCombat false) with the player surviving.
func TestCombatKillsTheGlowRat(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	read()
	send("Brawler")
	read()
	send("human")
	read()
	send("down")
	read() // tunnels, where the rat is

	send("attack rat")
	mustContain(t, "combat start", read(), "@event combat.start", `"name":"a glow-rat"`, `"inCombat":true`)

	sawRound, killed := false, false
	for i := 0; i < 6 && !killed; i++ {
		msg := read()
		if strings.Contains(msg, "@event combat.round") {
			sawRound = true
		}
		if strings.Contains(msg, "@event combat.end") && strings.Contains(msg, `"result":"killed"`) {
			killed = true
			mustContain(t, "post-kill vitals", msg, `"inCombat":false`)
		}
	}
	if !sawRound {
		t.Fatal("no combat.round event was emitted")
	}
	if !killed {
		t.Fatal("the glow-rat was not killed")
	}
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
