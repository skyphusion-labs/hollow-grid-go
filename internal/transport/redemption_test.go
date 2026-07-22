package transport

import (
	"strings"
	"testing"
)

// TestDaisRedemptionArc: pledge at the Ashmonger's dais, stray, defect, become Returned.
func TestDaisRedemptionArc(t *testing.T) {
	read, send, done := dial(t, newWorldServer(t))
	defer done()

	loginNewCharacter(t, read, send, "Redeemer", "human")

	for _, dir := range []string{"east", "up", "north", "north", "north", "north", "north", "up"} {
		send(dir)
		out := read()
		if strings.Contains(out, `"id":"dais"`) {
			break
		}
	}

	mark := read()
	send("join")
	joinOut := read()
	if !strings.Contains(joinOut, "strayed a long way") && !strings.Contains(joinOut, joinOut) {
		combined := mark + joinOut
		if !strings.Contains(combined, "strayed a long way") {
			t.Fatalf("join should stray the soul: %q", combined)
		}
	}
	if !strings.Contains(joinOut, `"faction":"front"`) && !strings.Contains(read(), `"faction":"front"`) {
		// affects may land in the same read as join
		send("affects")
		out := read()
		if !strings.Contains(out, `"faction":"front"`) {
			t.Fatalf("join should brand front: %q", joinOut+out)
		}
	}

	send("defy")
	defyOut := read()
	if !strings.Contains(defyOut, "grid.redemption") {
		t.Fatalf("defy should emit grid.redemption: %q", defyOut)
	}
	if !strings.Contains(defyOut, `"faction":"ally"`) {
		t.Fatalf("defy should stand with free folk: %q", defyOut)
	}
}
