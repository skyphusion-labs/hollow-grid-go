package grid

import (
	"context"
	"testing"
)

func TestLocalHubTideClamp(t *testing.T) {
	h := NewLocalHub("Dustfall", "ws://test/ws")
	ctx := context.Background()

	got, err := h.ShiftTide(ctx, 95)
	if err != nil || got != 95 {
		t.Fatalf("shift +95: got %d err %v", got, err)
	}
	got, err = h.ShiftTide(ctx, 10)
	if err != nil || got != 100 {
		t.Fatalf("shift +10 at 95: got %d want 100", got)
	}
	got, err = h.ShiftTide(ctx, -250)
	if err != nil || got != -100 {
		t.Fatalf("shift -250: got %d want -100", got)
	}
	tide, err := h.Tide(ctx)
	if err != nil || tide != -100 {
		t.Fatalf("tide read: got %d", tide)
	}
}

func TestLocalHubGridcastRelay(t *testing.T) {
	h := NewLocalHub("Dustfall", "ws://test/ws")
	ctx := context.Background()

	if err := h.GridCast(ctx, "Dustfall", "caster", "hello wastes"); err != nil {
		t.Fatal(err)
	}
	casts, err := h.CastsSince(ctx, 0, 10)
	if err != nil || len(casts) != 1 {
		t.Fatalf("casts: %v err %v", casts, err)
	}
	if casts[0].Sender != "caster" || casts[0].Text != "hello wastes" {
		t.Fatalf("cast payload: %+v", casts[0])
	}
	more, err := h.CastsSince(ctx, casts[0].ID, 10)
	if err != nil || len(more) != 0 {
		t.Fatalf("expected no casts after id %d, got %v", casts[0].ID, more)
	}
}

func TestLocalHubLedgerStats(t *testing.T) {
	h := NewLocalHub("Dustfall", "ws://test/ws")
	ctx := context.Background()
	stats, err := h.LedgerStats(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(stats) == 0 {
		t.Fatal("expected seeded traces in ledger stats")
	}
	total := 0
	for _, row := range stats {
		total += row.Count
	}
	if total < 3 {
		t.Fatalf("expected at least 3 seeded traces, got %d", total)
	}
}
