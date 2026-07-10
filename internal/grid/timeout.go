package grid

import (
	"context"
	"time"
)

// HubRPCTimeout caps every hub poll/RPC so a hung federation backend cannot
// stall world ticks (mirrors the-hollow-grid GRIDCAST_POLL_MS = 2000).
const HubRPCTimeout = 2 * time.Second

// WithHubTimeout returns ctx bounded by HubRPCTimeout. Federation is
// best-effort: callers treat errors as non-fatal.
func WithHubTimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithTimeout(ctx, HubRPCTimeout)
}
