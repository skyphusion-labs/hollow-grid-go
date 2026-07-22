package transport

import (
	"strings"
	"testing"
	"time"
)

func TestFleeEndsCombat(t *testing.T) {
	ts := newWorldServer(t)
	read, send, done := dial(t, ts)
	defer done()

	loginNewCharacter(t, read, send, "Fleer", "human")

	send("down")
	read()

	send("attack rat")
	out := readUntil(t, read, "@event combat.start")
	mustContain(t, "combat start", out, `"inCombat":true`)

	send("flee")
	out = read()
	mustContain(t, "flee end", out, `@event combat.end`, `"result":"fled"`, `"inCombat":false`)
}

func TestGetDropGroundItem(t *testing.T) {
	ts := newWorldServer(t)
	read, send, done := dial(t, ts)
	defer done()

	loginNewCharacter(t, read, send, "Picker", "human")

	send("drop shiv")
	out := read()
	mustContain(t, "drop shiv", out, "drop")

	send("get shiv")
	out = read()
	mustContain(t, "get shiv", out, "pick up")
}

func TestStolenKillDisplacesFighter(t *testing.T) {
	ts := newWorldServer(t)

	aRead, aSend, aDone := dial(t, ts)
	defer aDone()
	bRead, bSend, bDone := dial(t, ts)
	defer bDone()

	loginNewCharacter(t, aRead, aSend, "Alpha", "human")

	loginNewCharacter(t, bRead, bSend, "Beta", "human")

	aSend("down")
	aRead()
	bSend("down")
	bRead()

	// Both declare the fight before either heartbeat can finish the rat alone.
	aSend("attack rat")
	bSend("attack rat")

	deadline := time.Now().Add(15 * time.Second)
	var displaced string
	for time.Now().Before(deadline) {
		displaced += aRead()
		displaced += bRead()
		if strings.Contains(displaced, `@event combat.end`) &&
			strings.Contains(displaced, `"result":"gone"`) &&
			strings.Contains(displaced, `"inCombat":false`) {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("expected a displaced fighter to receive combat.end gone; got %q", displaced)
}
