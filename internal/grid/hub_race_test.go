package grid

import (
	"context"
	"sync"
	"testing"
)

func TestLocalHubConcurrentRecordAndLedgerStats(t *testing.T) {
	h := NewLocalHub("Dustfall", "ws://test/ws")
	ctx := context.Background()
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func(n int) {
			defer wg.Done()
			_ = h.Record(ctx, "Dustfall", "room", "quest", "trace", int64(n))
		}(i)
	}
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = h.LedgerStats(ctx)
		}()
	}
	wg.Wait()
}
