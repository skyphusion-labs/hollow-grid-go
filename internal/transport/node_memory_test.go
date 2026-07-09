package transport

import (
	"testing"

	"github.com/SkyPhusion/hollow-grid-go/internal/grid"
)

// RemoteHub federation must still keep per-node memory for grid.echo ping.
func TestLocalNodeMemoryWithRemoteHub(t *testing.T) {
	srv := NewServer(nil, nil, grid.NewRemoteHub("http://127.0.0.1:9/rpc", "x"), nil, nil)
	srv.recordLocalTrace("tunnels", "slain", "smoke slew the glow-rat here.")
	rows := srv.localTracesFor("tunnels", 6)
	if len(rows) != 1 || rows[0].Kind != "slain" {
		t.Fatalf("local traces: %+v", rows)
	}
}
