package grid

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRemoteHubUserAgent(t *testing.T) {
	var ua string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ua = r.Header.Get("User-Agent")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true,"result":0}`))
	}))
	defer srv.Close()

	hub := NewRemoteHub(srv.URL, "")
	if _, err := hub.Tide(context.Background()); err != nil {
		t.Fatalf("tide: %v", err)
	}
	if ua != "hollow-grid-go/0.1.0" {
		t.Fatalf("User-Agent = %q, want hollow-grid-go/0.1.0", ua)
	}
}

func TestRemoteHubTimesOutHungRPC(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
	}))
	defer srv.Close()

	hub := NewRemoteHub(srv.URL, "")
	start := time.Now()
	_, err := hub.Tide(context.Background())
	elapsed := time.Since(start)
	if err == nil {
		t.Fatal("expected timeout error from hung hub RPC")
	}
	if elapsed > 3*time.Second {
		t.Fatalf("RPC took %v; expected ~%v cap", elapsed, HubRPCTimeout)
	}
}
