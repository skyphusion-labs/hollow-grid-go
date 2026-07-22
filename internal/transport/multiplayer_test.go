package transport

import (
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func TestWallBroadcast(t *testing.T) {
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	srv := NewServer(world.New("test", ""), st, nil, []string{"skyphusion"}, testAdminToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Wait()
	})

	adminRead, adminSend, adminDone := dial(t, ts)
	defer adminDone()
	obsRead, obsSend, obsDone := dial(t, ts)
	defer obsDone()

	loginNewCharacter(t, adminRead, adminSend, "skyphusion", "human")

	loginNewCharacter(t, obsRead, obsSend, "watcher", "human")

	obsSend("wall I should not be able to do this")
	time.Sleep(200 * time.Millisecond)
	if !strings.Contains(obsRead(), "keeper of the Grid") {
		t.Fatal("non-admin should be refused wall")
	}

	beacon := "The Grid stirs in the deep dark."
	adminSend("wall " + beacon)
	time.Sleep(300 * time.Millisecond)
	out := obsRead()
	if !strings.Contains(out, beacon) {
		t.Fatalf("observer should receive wall broadcast: %q", out)
	}
	if !strings.Contains(out, "GRID BROADCAST") {
		t.Fatal("wall should be marked as broadcast")
	}
	if !strings.Contains(out, "@event server.announce") {
		t.Fatal("wall should emit server.announce")
	}
}

func TestGiveItem(t *testing.T) {
	ts := newWorldServer(t)

	pRead, pSend, pDone := dial(t, ts)
	defer pDone()
	qRead, qSend, qDone := dial(t, ts)
	defer qDone()

	loginNewCharacter(t, pRead, pSend, "Giver", "human")

	loginNewCharacter(t, qRead, qSend, "Taker", "human")

	pSend("north")
	pRead()
	qSend("north")
	qRead()

	pSend("defend")
	pReadUntil := func(sub string) {
		t.Helper()
		deadline := time.Now().Add(5 * time.Second)
		for time.Now().Before(deadline) {
			if strings.Contains(pRead(), sub) {
				return
			}
			time.Sleep(50 * time.Millisecond)
		}
		t.Fatalf("Giver never saw %q after defend", sub)
	}
	pReadUntil("elven charm")

	pSend("give charm Taker")
	pReadUntil("give")

	qSend("inventory")
	out := readUntil(t, qRead, "carry")
	if !strings.Contains(out, "charm") {
		t.Fatalf("recipient inventory should include charm: %q", out)
	}
}
