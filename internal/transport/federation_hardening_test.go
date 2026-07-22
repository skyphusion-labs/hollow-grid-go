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

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func newWorldServerWithHub(t *testing.T, gh grid.Hub) (*httptest.Server, *Server) {
	t.Helper()
	st, err := store.NewFileStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	srv := NewServer(world.New("Test World", ""), st, gh, []string{"skyphusion"}, testAdminToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(func() {
		ts.Close()
		srv.Wait()
	})
	return ts, srv
}

func blackholeHubServer(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Hang longer than HubRPCTimeout; client must give up without stalling ticks.
		time.Sleep(10 * time.Second)
	}))
}

func TestFederationLoopSurvivesBlackholedHub(t *testing.T) {
	hubSrv := blackholeHubServer(t)
	defer hubSrv.Close()

	_, srv := newWorldServerWithHub(t, grid.NewRemoteHub(hubSrv.URL, ""))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.RunFederation(ctx)

	done := make(chan struct{})
	go func() {
		time.Sleep(5 * time.Second)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(7 * time.Second):
		t.Fatal("federation loop stalled on blackholed hub")
	}
}

func TestCombatResolvesWithBlackholedHub(t *testing.T) {
	hubSrv := blackholeHubServer(t)
	defer hubSrv.Close()

	ts, srv := newWorldServerWithHub(t, grid.NewRemoteHub(hubSrv.URL, ""))
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	srv.RunFederation(ctx)

	aRead, aSend, aDone := dial(t, ts)
	defer aDone()
	bRead, bSend, bDone := dial(t, ts)
	defer bDone()

	loginNewCharacter(t, aRead, aSend, "Alpha", "human")

	loginNewCharacter(t, bRead, bSend, "Beta", "human")

	aSend("down")
	aRead()
	bSend("down")
	bRead()

	aSend("attack rat")
	bSend("attack rat")

	deadline := time.Now().Add(15 * time.Second)
	var displaced string
	for time.Now().Before(deadline) {
		displaced += aRead()
		displaced += bRead()
		if strings.Contains(displaced, `@event combat.end`) &&
			strings.Contains(displaced, `"result":"gone"`) &&
			strings.Contains(displaced, `"inCombat":false`) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected combat.end with blackholed hub; got %q", displaced)
}
