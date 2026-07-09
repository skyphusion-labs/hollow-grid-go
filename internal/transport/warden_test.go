package transport

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func TestWardenCleared(t *testing.T) {
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := world.New("test", "")
	srv := NewServer(w, st, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))
	pit := w.Room("holding_pit")

	if srv.wardenCleared() {
		t.Fatal("expected not cleared with live warden before any kill")
	}

	warden := pit.Mob("warden")
	srv.killMob("holding_pit", warden)
	if !srv.wardenCleared() {
		t.Fatal("expected cleared while warden is dead")
	}

	srv.mu.Lock()
	srv.deadMobs["warden"] = pendingRespawn{templateID: "warden", roomID: "holding_pit", at: time.Now().UnixMilli() - 1}
	srv.mu.Unlock()
	srv.tickRespawns()
	if pit.Mob("warden") == nil {
		t.Fatal("expected warden respawned")
	}
	if !srv.wardenCleared() {
		t.Fatal("expected cleared within grace after respawn")
	}

	srv.mu.Lock()
	srv.mobSlainAt["warden"] = time.Now().Add(-time.Duration(wardenGraceMs)*time.Millisecond - time.Second).UnixMilli()
	srv.mu.Unlock()
	if srv.wardenCleared() {
		t.Fatal("expected not cleared after grace expires")
	}
}

func TestWardenGraceRescueAfterRespawn(t *testing.T) {
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := world.New("test", "")
	srv := NewServer(w, st, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Wait()
	})

	read, send, done := dial(t, ts)
	defer done()

	read()
	send("Liberator")
	read()
	send("human")
	read()

	send("north")
	read()
	send("north")
	mustContain(t, "pit", readUntil(t, read, `"id":"holding_pit"`), `"mobs":[{"id":"warden"`)

	send("attack warden")
	readUntil(t, read, "@event combat.start")
	killed := false
	for i := 0; i < 10 && !killed; i++ {
		if strings.Contains(read(), `"result":"killed"`) {
			killed = true
		}
	}
	if !killed {
		t.Skip("the warden won this run (combat variance)")
	}

	srv.mu.Lock()
	srv.deadMobs["warden"] = pendingRespawn{templateID: "warden", roomID: "holding_pit", at: time.Now().UnixMilli() - 1}
	srv.mu.Unlock()
	srv.tickRespawns()
	if w.Room("holding_pit").Mob("warden") == nil {
		t.Fatal("expected warden respawned for grace test")
	}

	send("free")
	out := readUntil(t, read, "@event grid.rescued")
	mustContain(t, "grace rescue", out, `"savedBy":"Liberator"`, `"freed":[`)
	mustContain(t, "antidote prose", out, "Antivenom")

	send("sense")
	out = readUntil(t, read, "@event room.actions")
	if strings.Contains(out, `"verb":"free"`) {
		t.Fatalf("room.actions should not offer free after rescue: %s", out)
	}
}
