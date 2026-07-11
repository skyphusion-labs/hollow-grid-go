package transport

import (
	"net/http"
	"strings"
	"testing"
)

func TestPlayPageContainsXtermAndWorldName(t *testing.T) {
	body := playPage("Rust Choir")
	for _, want := range []string{"Rust Choir", "xterm", "@event ", "/ws", "the hollow grid network"} {
		if !strings.Contains(body, want) {
			t.Fatalf("playPage missing %q", want)
		}
	}
}

func TestPlayPageEscapesTitle(t *testing.T) {
	body := playPage(`<script>alert("x")</script>`)
	if strings.Contains(body, "<script>alert") {
		t.Fatal("playPage did not escape world name")
	}
}

func TestGETRootServesPlayPage(t *testing.T) {
	ts := newWorldServer(t)
	resp, err := http.Get(ts.URL + "/")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / status %d", resp.StatusCode)
	}
	ct := resp.Header.Get("Content-Type")
	if !strings.Contains(ct, "text/html") {
		t.Fatalf("content-type %q", ct)
	}
	buf := make([]byte, 4096)
	n, _ := resp.Body.Read(buf)
	body := string(buf[:n])
	if !strings.Contains(body, "xterm") {
		t.Fatal("GET / body missing xterm")
	}
}
