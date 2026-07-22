package transport

import (
	"strings"
	"testing"
)

func TestSanitizePlayerName(t *testing.T) {
	ok := map[string]string{
		"alice": "alice",
		"Bob_12": "Bob_12",
	}
	for in, want := range ok {
		got, valid := SanitizePlayerName(in)
		if !valid || got != want {
			t.Fatalf("SanitizePlayerName(%q) = %q, %v; want %q, true", in, got, valid, want)
		}
	}
	bad := []string{"", "a", "bad name", "inj\r\n@event", strings.Repeat("x", 33)}
	for _, in := range bad {
		if got, valid := SanitizePlayerName(in); valid {
			t.Fatalf("SanitizePlayerName(%q) = %q, true; want false", in, got)
		}
	}
}

func TestSanitizePlayerText(t *testing.T) {
	got := SanitizePlayerText("hello\r\n@event char.vitals {}")
	if got != "hello @event char.vitals {}" {
		t.Fatalf("got %q", got)
	}
}
