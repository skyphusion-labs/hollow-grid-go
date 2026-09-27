package transport

import (
	"errors"
	"io"
	"log/slog"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// failingLoadStore reports a real store fault on Load and records every Commit.
type failingLoadStore struct {
	mu      sync.Mutex
	commits int
}

func (f *failingLoadStore) Load(string) (world.CharSheet, bool, error) {
	return world.CharSheet{}, false, errors.New("disk read fault")
}

func (f *failingLoadStore) Commit(string, world.CharSheet) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.commits++
	return nil
}

func (f *failingLoadStore) commitCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.commits
}

// TestLoadErrorDoesNotEnterAsNew: a store fault on Load is not "no such
// character". The login must end without offering character creation, and
// nothing may be committed over the record the store could not read.
func TestLoadErrorDoesNotEnterAsNew(t *testing.T) {
	st := &failingLoadStore{}
	srv := NewServer(world.New("Test World", ""), st, nil, []string{"skyphusion"}, testAdminToken, slog.New(slog.NewTextHandler(io.Discard, nil)))
	ts := httptest.NewServer(srv.Handler())
	defer func() {
		ts.Close()
		srv.Wait()
	}()

	read, send, done := dial(t, ts)
	defer done()

	mustContain(t, "name prompt", read(), "wanderer")
	send("Tester")
	out := read()
	if strings.Contains(out, "Entering you as new") || strings.Contains(out, "choose what you are") {
		t.Fatalf("load error must not start character creation, got %q", out)
	}
	mustContain(t, "load error notice", out, "cannot read your record")

	done()
	ts.Close()
	srv.Wait()
	if n := st.commitCount(); n != 0 {
		t.Fatalf("no commit may follow a failed load, got %d", n)
	}
}
