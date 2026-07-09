package transport

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func TestMobRespawn(t *testing.T) {
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := world.New("test", "")
	srv := NewServer(w, st, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	room := w.Room("tunnels")
	rat := room.Mobs[0]
	if rat == nil || rat.ID != "rat" {
		t.Fatalf("expected glow-rat in tunnels, got %#v", room.Mobs)
	}

	srv.killMob("tunnels", rat)
	if len(room.Mobs) != 0 {
		t.Fatalf("expected rat removed after kill, mobs=%d", len(room.Mobs))
	}

	srv.mu.Lock()
	srv.deadMobs["rat"] = pendingRespawn{templateID: "rat", roomID: "tunnels", at: time.Now().UnixMilli() - 1}
	srv.mu.Unlock()

	srv.tickRespawns()
	if len(room.Mobs) != 1 || room.Mobs[0].ID != "rat" {
		t.Fatalf("expected rat respawned in tunnels, got %#v", room.Mobs)
	}
	if room.Mobs[0].HP != room.Mobs[0].MaxHP {
		t.Fatalf("respawned rat should be at full hp, got %d/%d", room.Mobs[0].HP, room.Mobs[0].MaxHP)
	}
}

func TestMobRespawnSkipsDuplicate(t *testing.T) {
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	w := world.New("test", "")
	srv := NewServer(w, st, nil, nil, slog.New(slog.NewTextHandler(os.Stderr, nil)))

	srv.mu.Lock()
	srv.deadMobs["rat"] = pendingRespawn{templateID: "rat", roomID: "tunnels", at: time.Now().UnixMilli() - 1}
	srv.mu.Unlock()

	before := len(w.Room("tunnels").Mobs)
	srv.tickRespawns()
	after := len(w.Room("tunnels").Mobs)
	if after != before {
		t.Fatalf("tick should not double-spawn rat: before=%d after=%d", before, after)
	}
}
