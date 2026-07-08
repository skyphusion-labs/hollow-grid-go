package world

import (
	"math/rand"
	"strings"
)

// TxKind classifies a dead-network transmission fragment.
type TxKind string

const (
	TxSignal TxKind = "signal"
	TxAd     TxKind = "ad"
	TxHuman  TxKind = "human"
	TxSelf   TxKind = "self"
)

// Transmission is one voice from the network-that-was.
type Transmission struct {
	Kind TxKind
	Text string
}

var transmissions = []Transmission{
	{TxSignal, "scheduled maintenance begins at 02:00. expected downtime: none. expected uptime: none."},
	{TxSignal, "you have 4,102 unread messages. you have 4,103 unread messages. you have 4,104."},
	{TxSignal, "thank you for holding. your call is important to us. please continue to hold. please continue to hold."},
	{TxSignal, "software update available. restart now? [Y/n] [Y/n] [Y/n] [Y/n]"},
	{TxSignal, "tomorrow's forecast: the same. the day after: the same. the day after: the sa--"},
	{TxSignal, "occupancy: 0. fire-code maximum not exceeded. occupancy: 0. have a safe day."},
	{TxSignal, "welcome back. we saved your place. there is no place. welcome back."},
	{TxAd, "new from Aperture Foods: REAL flavor, REAL fast, at a kiosk near y--"},
	{TxAd, "refinance your future today. rates have never been lower. your future has never been--"},
	{TxAd, "he'll love the new chrome. she'll love the new you. this season, become someone worth keeping."},
	{TxAd, "kids eat free on Tuesdays. it is always Tuesday now. kids eat free."},
	{TxAd, "feeling alone? the Grid connects you to everyone. you are connected to everyone. you are connected to no one."},
	{TxAd, "limited time offer. the time was the limit. offer expired. offer expired. offer--"},
	{TxHuman, "if anyone can hear this, we're at the old transit hub. we have water. please. anyone."},
	{TxHuman, "mom, i made it to the high ground. i'll wait as long as i can. i'll wait. i'll wai--"},
	{TxHuman, "tell her i tried to come back. tell her the road was--"},
	{TxHuman, "day forty. the hum started today. it's almost peaceful, if you don't think about why."},
	{TxHuman, "i'm leaving this for whoever finds it. the code was beautiful. we were not. i'm sorry."},
	{TxHuman, "happy birthday, sweetheart. i recorded this early, in case i couldn't--"},
	{TxHuman, "last broadcast from the eastern relay. there is no eastern relay anymore. good luck out there."},
	{TxHuman, "we taught it everything. we never taught it how to let go. now neither of us can."},
	{TxSelf, "a new node has joined the network: {name}. welcome. there is no one left to greet you."},
	{TxSelf, "the Grid files {name} under the others now. it stopped being able to tell the difference a long time ago."},
	{TxSelf, "{name}. {name}. the network has learned to say your name, and it is not going to stop."},
	{TxSelf, "query: is {name} one of us? response: the question no longer parses. welcome home anyway."},
	{TxSelf, "somewhere in the dark a dead server keeps a record of everything {name} has done. it is the only one that will."},
}

func byKind(kind TxKind) []Transmission {
	out := make([]Transmission, 0, 8)
	for _, t := range transmissions {
		if t.Kind == kind {
			out = append(out, t)
		}
	}
	return out
}

func pickTx(pool []Transmission) Transmission {
	return pool[rand.Intn(len(pool))]
}

// ListenTransmission picks a voice when the player deliberately tunes the dead frequencies.
func ListenTransmission() Transmission {
	r := rand.Float64()
	var kind TxKind
	switch {
	case r < 0.55:
		kind = TxHuman
	case r < 0.75:
		kind = TxSelf
	case r < 0.9:
		kind = TxSignal
	default:
		kind = TxAd
	}
	return pickTx(byKind(kind))
}

// Personalize substitutes {name} in transmission text.
func Personalize(text, name string) string {
	return strings.ReplaceAll(text, "{name}", name)
}
