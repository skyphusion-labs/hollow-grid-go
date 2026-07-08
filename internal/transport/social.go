package transport

import (
	"regexp"
	"strings"
	"unicode"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

const dustCost = 10

var talkable = map[string]bool{
	"tavern": true, "market": true, "workshop": true, "waystation": true,
	"holding_pit": true,
}

func (s *session) cmdTalk() {
	rid := s.player.RoomID
	if !talkable[rid] {
		s.line("There's no one here to talk to.")
		return
	}
	switch rid {
	case "tavern":
		s.line("The dealer rolls a packet of dust between his fingers: \"First taste eases any pain, friend. Just say buy dust.\"" +
			"\r\nAcross the room the tavern wench catches your eye and tilts her head toward the back rooms." +
			"\r\n(You could buy/use dust, carouse, or resist.)")
	case "market":
		if s.player.Faction == "none" {
			s.line("A Cinder Front recruiter bellows from a crate: \"The wastes are OURS! Round up every unregistered elf and drive them out!\"" +
				"\r\nA frightened elf refugee murmurs at your side: \"Please, I was born here. Don't let them take me.\"" +
				"\r\n(You could join the Front, or defend the refugees.)")
		} else if s.player.Faction == "front" {
			s.line("The recruiter nods at you, one of his own now. The square has gone quiet and afraid.")
		} else {
			s.line("An elf refugee presses your hand in silent thanks. The recruiter is nowhere in sight.")
		}
	case "workshop":
		s.line("The tinker doesn't look up from their soldering. \"Salvage's on the racks, prices on the list. Say 'list', say 'buy'. I don't haggle and I don't chat.\"")
	case "waystation":
		switch {
		case s.player.Faction == "front" || s.player.Ashsworn:
			s.line("A refugee spits at your feet. \"Cinder Front. We know what you are. Get gone, before we make you.\" There is no help for you here.")
		case s.player.Faction == "ally" || s.player.Morality >= 25:
			s.player.HP = s.player.MaxHP
			s.line("The medic pulls you onto the cot, cleans your wounds, and presses a hand to your shoulder. \"You stood with us when it counted. Rest, friend -- you are whole again.\" (fully healed)")
			s.event(event.CharVitals, s.player.Vitals())
		default:
			s.line("The medic studies you. \"We tend friends of the free folk. Pick a side, wanderer, and we will see.\"")
		}
	case "holding_pit":
		if s.room().Mob("warden") != nil {
			s.line("The chained maiden whispers: \"The warden holds the only key. Free me, and I will give you antivenom; the wastes are thick with poison.\"")
		} else {
			s.line("The freed maiden says: \"Stay safe out there. The antivenom is yours when the venom bites.\"")
		}
	default:
		s.line("There's no one here to talk to.")
	}
}

func (s *session) cmdBuy(arg string) {
	if s.room().ID == "tavern" {
		if !strings.Contains(strings.ToLower(arg), "dust") {
			s.line("The dealer only deals one thing: dust. (\"buy dust\")")
			return
		}
		if s.player.Gold < dustCost {
			s.line("The dealer sneers. \"" + itoa(dustCost) + " gold, no credit.\" You're short.")
			return
		}
		s.player.Gold -= dustCost
		s.player.AddItem("dust")
		s.persist()
		s.srv.hub.Sync(s.player)
		s.line("The dealer slips you a packet of dust. (-" + itoa(dustCost) + " gold, gold: " + itoa(s.player.Gold) + ")")
		s.event(event.CharVitals, s.player.Vitals())
		return
	}
	// workshop buy handled in main switch
	switch price, id, ok := tinkerPrice(arg); {
	case s.room().ID != "workshop":
		s.line("There is nothing to buy here.")
	case !ok:
		s.line("The tinker doesn't sell that.")
	case s.player.Gold < price:
		s.line("You can't afford that -- it is " + itoa(price) + " gold and you have " + itoa(s.player.Gold) + ".")
	default:
		s.player.Gold -= price
		s.player.AddItem(id)
		s.line("The tinker hands you " + world.ItemName(id) + " and pockets your coin.")
		s.event(event.CharVitals, s.player.Vitals())
	}
}

func (s *session) cmdSell(arg string) {
	if s.room().ID != "market" {
		s.line("There's no one here buying.")
		return
	}
	if s.player.Faction == "front" {
		s.line("The vendor drone's optic flares red. \"Cinder Front. We remember Scrap Market. We don't trade with your kind.\" It turns its back on you, and the stalls nearby go quiet.")
		return
	}
	if strings.TrimSpace(arg) == "" {
		s.line("Sell what?")
		return
	}
	id, ok := s.player.FindInventory(arg)
	if !ok {
		s.line("You aren't carrying \"" + arg + "\".")
		return
	}
	s.player.RemoveFromInventory(id)
	value := 5
	if s.player.Faction == "ally" {
		value = 6
	}
	s.player.Gold += value
	s.persist()
	s.srv.hub.Sync(s.player)
	s.line("You sell " + world.ItemName(id) + " for " + itoa(value) + " gold.")
	s.event(event.CharVitals, s.player.Vitals())
}

func (s *session) cmdLookPlayer(arg string) bool {
	lp, ok := s.srv.hub.FindPrefix(arg)
	if !ok || lp.name == s.player.Name || lp.room != s.player.RoomID {
		return false
	}
	p := lp.plr
	if p == nil {
		return false
	}
	s.line(world.Tagged(p) + " stands before you, looking steady.")
	s.event(event.PlayerRead, map[string]any{
		"name": p.Name, "title": p.Title, "faction": p.Faction,
		"ashsworn": p.Ashsworn, "regard": world.Regard(p),
	})
	return true
}

func (s *session) cmdForgive(arg string) {
	who := strings.Fields(strings.TrimSpace(arg))[0]
	if who == "" {
		s.line("Forgive whom?  (forgive <player> -- choose to let someone marked back in)")
		return
	}
	lp, ok := s.srv.hub.FindPrefix(who)
	if !ok || lp.name == s.player.Name || lp.room != s.player.RoomID {
		if ok && lp.name == s.player.Name {
			s.line("You cannot forgive yourself here; that is a longer road, and a lonelier one.")
		} else {
			s.line("There's no one called \"" + who + "\" here to forgive.")
		}
		return
	}
	target := lp.plr
	if target == nil {
		return
	}
	if s.srv.hasForgiven(s.player.Name, target.Name) {
		s.line("You have already forgiven " + target.Name + ". It was true the first time; it does not need saying twice.")
		return
	}
	marked := target.Ashsworn || target.Faction == "front" || target.Morality <= -50
	if !marked {
		s.line(target.Name + " carries nothing that needs your forgiveness. Keep the words for someone who does.")
		return
	}
	s.srv.markForgiven(s.player.Name, target.Name)
	s.shiftMorality(2)
	s.recordTrace(s.player.RoomID, "grace", s.player.Name+" forgave "+target.Name+" here.")

	s.srv.hub.push(target.Name, s.player.Name+" looks at you and chooses to forgive you.\r\n")
	if target.Ashsworn {
		s.srv.hub.push(target.Name, "It reaches something in you. But the ash does not lift; it never will. You carry the mark and the mercy both. Some things are not forgotten, even when they are forgiven.\r\n")
		s.srv.hub.pushEvent(target.Name, event.CharForgiven, map[string]any{
			"by": s.player.Name, "ashsworn": true, "redeemed": false,
		})
	} else {
		s.srv.hub.pushEvent(target.Name, event.CharForgiven, map[string]any{
			"by": s.player.Name, "ashsworn": false, "redeemed": false,
		})
		s.srv.hub.push(target.Name, "It lands, and it stays with you. The road is still yours to walk, but you are not walking it unseen.\r\n")
	}
	s.srv.hub.BroadcastRoomExcept(s.player.RoomID, s.player.Name+" forgives "+target.Name+".\r\n", s.player.Name, target.Name)
	s.line("You choose to forgive " + target.Name + ". Out here that is not nothing; it may be everything.")
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdWall(arg string) {
	if !s.srv.isAdmin(s.player.Name) {
		s.line("Only a keeper of the Grid can broadcast across the wastes.")
		return
	}
	msg := strings.TrimSpace(arg)
	if msg == "" {
		s.line("Announce what?  (wall <message>)")
		return
	}
	banner := "*** GRID BROADCAST ***  " + msg
	for _, lp := range s.srv.hub.All() {
		var text string
		if lp.name == s.player.Name {
			text = banner + crlf
		} else {
			text = banner + crlf
		}
		ev, _ := event.Line(event.ServerAnnounce, map[string]string{"from": s.player.Name, "text": msg})
		s.srv.hub.push(lp.name, text+ev+crlf)
	}
}

func (s *session) cmdInscribe(arg string) {
	msg := sanitizeInscription(arg)
	if len(msg) < 2 {
		s.line("Carve what into the Grid? (inscribe <a few words for whoever comes next>)")
		return
	}
	text := s.player.Name + ": \"" + msg + "\""
	s.recordTrace(s.player.RoomID, "mark", text)
	s.line("You press your words into the dead network, where they will outlast you:")
	s.line("  \"" + msg + "\"")
	s.line("The Grid takes them. Someone will key into this node, long after you are gone, and hear you. (try 'ping')")
	s.event(event.GridInscribed, map[string]string{"node": s.player.RoomID, "text": msg})
}

func sanitizeInscription(arg string) string {
	var b strings.Builder
	for _, r := range arg {
		if r >= 0x20 && r <= 0x7e && !unicode.IsControl(r) {
			b.WriteRune(r)
		} else if r == '\t' || r == '\n' || r == '\r' {
			b.WriteByte(' ')
		}
	}
	out := strings.Join(strings.Fields(b.String()), " ")
	if len(out) > 120 {
		out = out[:120]
	}
	return strings.TrimSpace(out)
}

func (s *session) cmdCache(arg string) {
	amount := parseLeadingInt(arg)
	if amount < 1 {
		s.line("Cache how much?  (cache <gold> -- leave it here for whoever comes next)")
		return
	}
	if s.player.Gold < amount {
		s.line("You don't have " + itoa(amount) + " gold to give. (you have " + itoa(s.player.Gold) + ")")
		return
	}
	s.player.Gold -= amount
	s.srv.addCache(s.player.RoomID, amount)
	s.shiftMorality(2)
	s.persist()
	s.srv.hub.Sync(s.player)
	s.recordTrace(s.player.RoomID, "aid", s.player.Name+" left aid here for whoever comes next.")
	s.line("You tuck " + itoa(amount) + " gold into a hollow where the next traveler will find it. They'll never know your name. You do it anyway.")
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdGather() {
	here := s.srv.cacheGold(s.player.RoomID)
	if here <= 0 {
		s.line("There's nothing cached here. If you have something to spare, you could change that. (cache <gold>)")
		return
	}
	s.player.Gold += here
	s.srv.takeCache(s.player.RoomID)
	s.persist()
	s.srv.hub.Sync(s.player)
	s.line("You find " + itoa(here) + " gold someone cached here. Wherever they are, they meant it for a stranger; tonight that's you. (gold: " + itoa(s.player.Gold) + ")")
	s.event(event.CharVitals, s.player.Vitals())
}

func (s *session) announceCacheIfAny() {
	g := s.srv.cacheGold(s.player.RoomID)
	if g > 0 {
		s.line("Someone has cached aid here: " + itoa(g) + " gold, left for whoever comes next. (gather)")
		s.event(event.NodeCache, map[string]int{"gold": g})
	}
}

func (s *session) cmdGive(arg string) {
	toks := strings.Fields(strings.TrimSpace(arg))
	if len(toks) < 2 {
		s.line("Give what to whom?  (give <item> <player>)")
		return
	}
	who := toks[len(toks)-1]
	itemToks := toks[:len(toks)-1]
	if len(itemToks) > 0 && strings.EqualFold(itemToks[len(itemToks)-1], "to") {
		itemToks = itemToks[:len(itemToks)-1]
	}
	itemArg := strings.Join(itemToks, " ")
	id, ok := s.player.FindInventory(itemArg)
	if !ok {
		s.line("You aren't carrying \"" + itemArg + "\".")
		return
	}
	lp, ok := s.srv.hub.FindPrefix(who)
	if !ok || lp.name == s.player.Name || lp.room != s.player.RoomID || lp.plr == nil {
		s.line("There's no one called \"" + who + "\" here to give it to.")
		return
	}
	s.player.RemoveFromInventory(id)
	lp.plr.AddItem(id)
	s.line("You give " + world.ItemName(id) + " to " + lp.name + ".")
	s.srv.hub.push(lp.name, s.player.Name+" gives you "+world.ItemName(id)+".\r\n")
}

func (s *session) cmdMend(arg string) {
	lp, ok := s.srv.hub.FindPrefix(strings.TrimSpace(arg))
	if !ok || lp.name == s.player.Name || lp.room != s.player.RoomID || lp.plr == nil {
		s.line("There's no one like that here to mend.")
		return
	}
	if lp.plr.HP >= lp.plr.MaxHP {
		s.line(lp.name + " is already whole.")
		return
	}
	cost := 5
	if s.player.HP <= cost {
		s.line("You don't have enough life left to spare.")
		return
	}
	s.player.HP -= cost
	heal := 10
	lp.plr.HP += heal
	if lp.plr.HP > lp.plr.MaxHP {
		lp.plr.HP = lp.plr.MaxHP
	}
	s.srv.hub.Sync(s.player)
	s.srv.hub.Sync(lp.plr)
	s.line("You spend a little of yourself to mend " + lp.name + ".")
	s.srv.hub.push(lp.name, s.player.Name+" tends your wounds.\r\n")
	s.event(event.CharVitals, s.player.Vitals())
}

func (s *session) defendMarket() {
	r := s.room()
	if r.ID != "market" || s.resolved[r.ID+":defend"] || s.resolved[r.ID+":join"] {
		s.line("There is no stand to take here.")
		return
	}
	s.player.Faction = "ally"
	s.shiftMorality(25)
	s.player.AddItem("charm")
	s.markResolved(r.ID, "defend", "join")
	s.persist()
	s.srv.hub.Sync(s.player)
	s.srv.hub.BroadcastRoom(r.ID, s.player.Name+" stands with the elves against the Cinder Front.", s.player.Name)
	s.recordTrace(r.ID, "oath", s.player.Name+" stood with the free folk here.")
	s.line("You step between the recruiter and the refugees: \"They stay. They belong here as much as you do.\" The recruiter spits and storms off. The elves press an elven charm into your hands, eyes bright with thanks.")
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(r))
}

func (s *session) freeSynonym() { s.freeCaptive() }

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var d [12]byte
	i := len(d)
	for n > 0 {
		i--
		d[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		d[i] = '-'
	}
	return string(d[i:])
}

func parseLeadingInt(arg string) int {
	re := regexp.MustCompile(`^\s*(\d+)`)
	m := re.FindStringSubmatch(arg)
	if len(m) < 2 {
		return 0
	}
	n := 0
	for _, c := range m[1] {
		n = n*10 + int(c-'0')
	}
	return n
}

// pushEvent sends a structured event line to another player's push channel.
func (h *Hub) pushEvent(name, evName string, payload any) {
	l, err := event.Line(evName, payload)
	if err != nil {
		return
	}
	h.push(name, l+crlf)
}

// BroadcastRoomExcept sends prose to everyone in room except two names.
func (h *Hub) BroadcastRoomExcept(room, text, skip1, skip2 string) {
	h.mu.RLock()
	targets := make([]*livePlayer, 0, 4)
	for name, lp := range h.players {
		if lp.room == room && name != skip1 && name != skip2 {
			targets = append(targets, lp)
		}
	}
	h.mu.RUnlock()
	for _, lp := range targets {
		select {
		case lp.push <- text:
		default:
		}
	}
}
