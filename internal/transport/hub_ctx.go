package transport

import (
	"context"

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
)

// hubCtx returns a context capped at grid.HubRPCTimeout. Federation is
// best-effort and must never block play.
func hubCtx() (context.Context, context.CancelFunc) {
	return grid.WithHubTimeout(context.Background())
}
