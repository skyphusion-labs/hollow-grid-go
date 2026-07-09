// Package event implements the Hollow Grid structured @event channel.
//
// Every player-affecting state change is emitted as its own line:
//
//	@event <name> <json>
//
// interleaved with the human prose, so clients, bots, and the test suite parse
// machine-readable state instead of scraping English. See docs/protocol.md
// section 2 in the upstream the-hollow-grid repo for the event vocabulary.
package event

import (
	"encoding/json"
	"fmt"
)

// Prefix marks a structured event line. Clients that do not care may ignore any
// line beginning with it.
const Prefix = "@event "

// Canonical event names (a growing subset of the protocol vocabulary).
const (
	RoomInfo         = "room.info"
	RoomActions      = "room.actions"
	CharVitals       = "char.vitals"
	CharAffects      = "char.affects"
	CharEquipment    = "char.equipment"
	CharIdentity     = "char.identity"
	CharDream        = "char.dream"
	CharDied         = "char.died"
	CombatStart      = "combat.start"
	CombatRound      = "combat.round"
	CombatEnd        = "combat.end"
	WorldState       = "world.state"
	GridRescued      = "grid.rescued"
	GridRescuedRoll  = "grid.rescued_roll"
	GridRedemption   = "grid.redemption"
	GridFallen       = "grid.fallen"
	GridRemembrance  = "grid.remembrance"
	CharReckoning    = "char.reckoning"
	GridEcho         = "grid.echo"
	GridFederation   = "grid.federation"
	GridTransmission = "grid.transmission"
	GridWho          = "grid.who"
	GridWorlds       = "grid.worlds"
	GridTravel       = "grid.travel"
	CommGridcast     = "comm.gridcast"
	WorldWar         = "world.war"
	GridLedgerStats  = "grid.ledger_stats"
	GridLedgerPruned = "grid.ledger_pruned"
	CommTell         = "comm.tell"
	CommYell         = "comm.yell"
	CharForgiven     = "char.forgiven"
	PlayerRead       = "player.read"
	ServerAnnounce   = "server.announce"
	NodeCache        = "node.cache"
	GridInscribed    = "grid.inscribed"
)

// Line formats a single @event line (without the trailing CRLF, which the
// transport adds). payload is marshalled to single-line JSON.
func Line(name string, payload any) (string, error) {
	b, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("event %q: %w", name, err)
	}
	return Prefix + name + " " + string(b), nil
}
