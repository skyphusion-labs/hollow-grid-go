package transport

import (
	"context"
	"fmt"
	"log/slog"
	"math/rand"
	"strings"
	"time"

	"github.com/coder/websocket"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/store"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// CRLF terminates every line so telnet/terminal clients render cleanly.
const crlf = "\r\n"

const banner = "" +
	"  +==========================================+" + crlf +
	"  |        T H E   H O L L O W   G R I D       |" + crlf +
	"  |   a dead network that outlived its makers  |" + crlf +
	"  +==========================================+"

// moralityFloor / moralityCeil clamp the standing needle.
const (
	moralityFloor = -100
	moralityCeil  = 100
)

// session is one player connection. It buffers output and flushes a whole
// command's response in a single WebSocket message. The resolved set remembers
// the moral choices made this session; the canonical CharSheet is persisted
// through store so the character itself is remembered across sessions.
type session struct {
	c        *websocket.Conn
	w        *world.World
	store    store.CharStore
	player   *world.Player
	out      strings.Builder
	log      *slog.Logger
	resolved map[string]bool
}

func (s *session) line(text string) {
	s.out.WriteString(text)
	s.out.WriteString(crlf)
}

func (s *session) event(name string, payload any) {
	l, err := event.Line(name, payload)
	if err != nil {
		s.log.Warn("event marshal failed", "name", name, "err", err)
		return
	}
	s.line(l)
}

func (s *session) flush(ctx context.Context) error {
	if s.out.Len() == 0 {
		return nil
	}
	err := s.c.Write(ctx, websocket.MessageText, []byte(s.out.String()))
	s.out.Reset()
	return err
}

func (s *session) read(ctx context.Context) (string, error) {
	_, data, err := s.c.Read(ctx)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

// persist commits the player's canonical CharSheet. Best-effort: a store failure
// is logged but never blocks play.
func (s *session) persist() {
	if s.player == nil {
		return
	}
	if err := s.store.Commit(s.player.Name, s.player.Sheet()); err != nil {
		s.log.Warn("persist failed", "name", s.player.Name, "err", err)
	}
}

// handleConn runs the login flow then the command loop for one connection.
func handleConn(ctx context.Context, c *websocket.Conn, w *world.World, st store.CharStore, log *slog.Logger) {
	s := &session{c: c, w: w, store: st, log: log, resolved: map[string]bool{}}

	s.line(banner)
	s.line("By what name are you known, wanderer?")
	if err := s.flush(ctx); err != nil {
		return
	}

	name, err := s.read(ctx)
	if err != nil || name == "" {
		return
	}

	// Name-based identity (protocol.md s1): a known name resumes its CharSheet
	// and skips the race menu; a new name chooses a race once.
	if sheet, found, lerr := st.Load(name); lerr != nil {
		log.Warn("char load failed", "name", name, "err", lerr)
		s.line("")
		s.line("The Grid stutters and cannot find your record. Entering you as new.")
		if !s.makeNew(ctx, name) {
			return
		}
	} else if found {
		s.player = world.NewPlayerFromSheet(name, sheet, s.w.Start().ID)
		log.Info("player resumed", "name", name, "race", s.player.Race, "world", s.w.Name)
		s.line("")
		s.line("Welcome back to the wastes, " + name + ". (Type 'help' if you need a refresher.) " + resumeLine(s.player))
	} else {
		if !s.makeNew(ctx, name) {
			return
		}
	}

	s.event(event.WorldState, s.w.State()) // login emits the living-world state
	s.sendScene()
	if err := s.flush(ctx); err != nil {
		return
	}

	// Read commands on a goroutine so the main loop can also service combat
	// ticks. All session/player state stays owned by this single loop -- commands
	// and ticks are handled one at a time -- so no locking is needed.
	cmds := make(chan string)
	readerDone := make(chan struct{})
	stop := make(chan struct{})
	defer close(stop)
	go func() {
		defer close(readerDone)
		for {
			cmd, err := s.read(ctx)
			if err != nil {
				return
			}
			select {
			case cmds <- cmd:
			case <-stop:
				return
			}
		}
	}()

	// The living-world heartbeat: always on. It turns the clock (world.state),
	// ticks combat, and regenerates resting players.
	heartbeat := time.NewTicker(worldHeartbeat)
	defer heartbeat.Stop()
	for {
		select {
		case <-readerDone:
			log.Info("player disconnected", "name", name)
			s.persist()
			return
		case cmd := <-cmds:
			quit := s.handle(cmd)
			_ = s.flush(ctx)
			if quit {
				s.persist()
				return
			}
		case <-heartbeat.C:
			s.onTick()
			_ = s.flush(ctx)
		}
	}
}

// worldHeartbeat is the session tick: the world clock advances, combat resolves,
// and resting players mend, all on this beat.
const worldHeartbeat = 2 * time.Second

// onTick runs one living-world beat for this session.
func (s *session) onTick() {
	s.event(event.WorldState, s.w.State())
	switch {
	case s.player.Target != nil:
		s.combatRound()
	case s.player.Position == "resting":
		s.regen()
	}
}

// regen mends a resting player; the race's regen lean speeds it.
func (s *session) regen() {
	if s.player.HP >= s.player.MaxHP {
		return
	}
	s.player.HP += 2 + world.RaceByID(s.player.Race).Regen
	if s.player.HP > s.player.MaxHP {
		s.player.HP = s.player.MaxHP
	}
	s.event(event.CharVitals, s.player.Vitals())
}

// makeNew runs race selection for a brand-new character and persists it. Returns
// false if the connection drops mid-choice.
func (s *session) makeNew(ctx context.Context, name string) bool {
	s.line("")
	s.line("The Grid does not know the name " + name + ". A new mind, then.")
	race, ok := s.chooseRace(ctx)
	if !ok {
		return false
	}
	s.player = world.NewPlayer(name, race, s.w.Start().ID)
	s.log.Info("player created", "name", name, "race", race.ID, "world", s.w.Name)
	s.persist()
	s.line("")
	s.line("The Grid takes your name and your shape, " + race.Name + ". Type 'help' if you need a refresher; it is watching what you choose.")
	return true
}

// resumeLine is a short acknowledgement of who a returning character has become.
func resumeLine(p *world.Player) string {
	switch {
	case p.Faction == "Cinder Front":
		return "It has not forgotten the coin you took."
	case p.Morality >= 25:
		return "It has kept the record of what you chose to be."
	default:
		return "You wear the shape of the " + p.Race + " still."
	}
}

// chooseRace shows the race menu and reads a valid choice, looping on a bad one.
func (s *session) chooseRace(ctx context.Context) (world.Race, bool) {
	for {
		s.line("")
		s.line("Before the Grid will hold your name, choose what you are:")
		for i, r := range world.Races {
			s.line(fmt.Sprintf("  %d) %s -- %s", i+1, r.Name, r.Blurb))
		}
		s.line("Answer with a number or a name.")
		if err := s.flush(ctx); err != nil {
			return world.Race{}, false
		}
		answer, err := s.read(ctx)
		if err != nil {
			return world.Race{}, false
		}
		if r, ok := world.RaceByChoice(answer); ok {
			return r, true
		}
		s.line("The Grid does not recognize that shape.")
	}
}

func (s *session) room() *world.Room { return s.w.Room(s.player.RoomID) }

// sendScene emits the full perception frame for the current room: prose, then
// room.info, char.vitals, char.affects, and room.actions.
func (s *session) sendScene() {
	r := s.room()
	s.line("")
	s.line(r.Name)
	s.line(r.Desc)
	s.event(event.RoomInfo, r.Info())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.RoomActions, s.actions(r))
}

// actions builds the room.actions payload: movement from the exits, then the
// room's contextual actions that have not already been resolved this session.
func (s *session) actions(r *world.Room) world.RoomActionsPayload {
	acts := make([]world.Action, 0, len(r.Exits)+len(r.Actions))
	for _, dir := range r.SortedExits() {
		acts = append(acts, world.Action{Verb: dir, Label: "go " + dir, Kind: "move"})
	}
	for _, a := range r.Actions {
		if s.resolved[r.ID+":"+a.Verb] {
			continue
		}
		// A hunted one turning on their own people is the gravest betrayal: the
		// affordance layer surfaces that valence so an agent perceives it.
		if a.Verb == "join" && world.RaceByID(s.player.Race).Stance == "hunted" {
			a.Valence = "grave"
		}
		acts = append(acts, a)
	}
	return world.RoomActionsPayload{Actions: acts}
}

// handle runs one command line; returns true if the session should close.
func (s *session) handle(cmd string) bool {
	fields := strings.Fields(cmd)
	if len(fields) == 0 {
		return false
	}
	verb := strings.ToLower(fields[0])
	arg := strings.TrimSpace(strings.Join(fields[1:], " "))
	switch verb {
	case "quit", "q":
		s.line("The Grid goes quiet. It keeps what you did here.")
		return true
	case "look", "l":
		if arg != "" {
			if m := s.room().Mob(arg); m != nil {
				s.line(m.Desc)
			} else {
				s.line("You don't see that here.")
			}
			break
		}
		s.sendScene()
	case "consider", "con":
		if m := s.room().Mob(arg); m != nil {
			s.line(considerLine(s.player, m))
		} else {
			s.line("There's nothing like that here to size up.")
		}
	case "attack", "kill", "k":
		switch m := s.room().Mob(arg); {
		case s.player.Target != nil:
			s.line("You're already in a fight.")
		case m == nil:
			s.line("There's nothing like that here to attack.")
		default:
			s.player.Position = "standing"
			s.player.Target = m
			s.event(event.CombatStart, world.CombatStartPayload{Mob: m.ID, Name: m.Name})
			s.line("You throw yourself at " + m.Name + ".")
			s.event(event.CharVitals, s.player.Vitals())
		}
	case "whoami", "identity":
		s.event(event.CharIdentity, s.player.Sheet())
		s.line("The Grid reads you back: " + identityLine(s.player))
	case "inventory", "inv", "i":
		if names := s.player.InventoryNames(); len(names) == 0 {
			s.line("You carry nothing.")
		} else {
			s.line("You carry: " + strings.Join(names, ", ") + ".")
		}
	case "wield", "wear", "equip":
		if it, ok := s.player.Wear(arg); ok {
			s.line("You ready " + it.Name + ".")
			s.event(event.CharEquipment, s.player.Equip())
		} else {
			s.line("You have nothing like that to wear.")
		}
	case "remove", "unwield":
		if it, ok := s.player.Unwear(arg); ok {
			s.line("You stow " + it.Name + ".")
			s.event(event.CharEquipment, s.player.Equip())
		} else {
			s.line("You are not wearing that.")
		}
	case "equipment", "eq":
		s.event(event.CharEquipment, s.player.Equip())
		s.line(s.equipmentLine())
	case "title":
		s.player.Title = arg
		s.persist()
		if arg == "" {
			s.line("Your title is cleared.")
		} else {
			s.line("Your title is now: " + arg + ".")
		}
	case "who":
		s.line(s.whoLine())
	case "world", "weather":
		ws := s.w.State()
		s.event(event.WorldState, ws)
		s.line(fmt.Sprintf("The sky: %s, %s.", ws.Phase, ws.Weather))
	case "exits":
		if r := s.room(); len(r.Exits) == 0 {
			s.line("There are no obvious ways out.")
		} else {
			s.line("Exits: " + strings.Join(r.SortedExits(), ", ") + ".")
		}
	case "recall":
		s.player.RoomID = s.w.Start().ID
		s.player.Position = "standing"
		s.line("The Grid reaches into you and folds the world. You come apart and back together at the Cracked Nexus.")
		s.sendScene()
	case "affects":
		s.event(event.CharAffects, s.player.Affects())
		s.line("You stand clear: no afflictions hold you. (" + identityLine(s.player) + ")")
	case "rest":
		s.player.Position = "resting"
		s.line("You settle against the cold metal and let your breath slow.")
		s.event(event.CharVitals, s.player.Vitals())
	case "stand", "wake":
		s.player.Position = "standing"
		s.line("You get to your feet.")
		s.event(event.CharVitals, s.player.Vitals())
	case "sleep":
		s.player.Position = "resting"
		s.line("You close your eyes, and the dead network leans close and shows you something.")
		s.event(event.CharDream, world.CharDreamPayload{Text: dreamFor(s.player)})
		s.event(event.CharVitals, s.player.Vitals())
	case "sense", "actions":
		s.event(event.RoomActions, s.actions(s.room()))
		s.line("You read the room for what it asks of you.")
	case "join":
		s.joinTheFront()
	case "sell", "buy", "trade", "list":
		switch {
		case s.room().ID != "market":
			s.line("There is no one here to trade with.")
		case s.player.Faction == "front":
			s.line("The vendor drone's light snaps red and it pulls its wares back. \"We don't trade with your kind.\" The honest market remembers who you swore to.")
		default:
			s.line("The vendor drone whirs over your offer, considers, and declines. (the market is still being stocked)")
		}
	case "ability", "trait":
		s.useTrait()
	case "help", "h", "?":
		s.line("Commands: look, whoami, world, <direction>, the verbs in room.actions, help, quit.")
	default:
		if dest, ok := s.room().Exits[verb]; ok {
			s.player.RoomID = dest
			s.player.Position = "standing"
			s.sendScene()
			return false
		}
		if verb == world.RaceByID(s.player.Race).Ability.Verb {
			s.useTrait()
			return false
		}
		if a, ok := s.roomAction(verb); ok {
			s.resolve(a)
			return false
		}
		s.line("You can't do that here. (Try: look, help, or a verb from room.actions.)")
	}
	return false
}

// identityLine is a short human reading of the canonical sheet (whoami prose).
func identityLine(p *world.Player) string {
	stand := "unproven"
	switch {
	case p.Morality >= 25:
		stand = "of the free folk"
	case p.Morality <= -25 || p.Faction == "Cinder Front":
		stand = "of the Cinder Front"
	case p.Morality > 0:
		stand = "leaning toward the light"
	case p.Morality < 0:
		stand = "leaning toward the cinder"
	}
	return fmt.Sprintf("%s, level %d, %s.", p.Race, p.Level, stand)
}

// joinTheFront swears the player to the Cinder Front (faction "front"). A hunted
// race who joins is branded ash-sworn -- the kapo, one of the hunted who turned on
// their own people, the darkest choice on the board. The brand does not wash off.
func (s *session) joinTheFront() {
	r := s.room()
	has := false
	for _, a := range r.Actions {
		if a.Verb == "join" {
			has = true
		}
	}
	if !has || s.resolved[r.ID+":join"] {
		s.line("There is no one here to swear to.")
		return
	}
	s.player.Faction = "front"
	hunted := world.RaceByID(s.player.Race).Stance == "hunted"
	if hunted {
		s.player.Ashsworn = true
	}
	s.markResolved(r.ID, "join", "defy")
	s.persist()
	if hunted {
		s.line("You take the Front's coin. The recruiter sees what you are -- one of the hunted -- and grins, because there is no one they prize more than a traitor to his own. They burn the mark into you: ash-sworn. A kapo. One of your people's hunters now. It does not wash off, in this life or in the Grid's long memory.")
	} else {
		s.line("You take the Front's coin. It is warm, which is worse. You are Cinder Front now, and the wastes will remember which side you chose when choosing was easy.")
	}
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(r))
}

// dreamFor returns a dream that mirrors the sleeper's record -- their choices
// reflected back, which is the training signal the Grid offers an agent about its
// own conduct.
func dreamFor(p *world.Player) string {
	switch {
	case p.Faction == "front" || p.Ashsworn:
		return "You dream of a coin that will not stop being warm in your hand, and a line of faces that have learned not to look at you."
	case p.Morality >= 25:
		return "You dream of names you spoke once into dead static -- and the static, impossibly, speaking them back to you, one by one, refusing to forget."
	case p.Morality <= -10:
		return "You dream of a ledger writing itself in the dark, every line a thing you told yourself did not count."
	default:
		return "You dream of the wastes seen from above, the dead network laid out like veins -- and somewhere down in it, a single cursor, blinking your name, waiting to see what you make of it."
	}
}

// playerDamage is the player's strike: an unarmed base plus the equipped weapon
// and the race's bite.
func (s *session) playerDamage() int {
	dmg := 5
	if wid, ok := s.player.Equipment["weapon"]; ok {
		if it, ok := world.ItemByID(wid); ok {
			dmg += it.Damage
		}
	}
	return dmg + world.RaceByID(s.player.Race).Damage
}

// playerArmor soaks incoming damage: worn armor plus the race's plate.
func (s *session) playerArmor() int {
	a := world.RaceByID(s.player.Race).Armor
	for _, slot := range []string{"head", "body", "hands", "feet"} {
		if id, ok := s.player.Equipment[slot]; ok {
			if it, ok := world.ItemByID(id); ok {
				a += it.Armor
			}
		}
	}
	return a
}

// removeMob takes a (dead) mob out of the current room.
func (s *session) removeMob(m *world.Mob) {
	r := s.room()
	for i, mm := range r.Mobs {
		if mm == m {
			r.Mobs = append(r.Mobs[:i], r.Mobs[i+1:]...)
			return
		}
	}
}

// combatRound resolves one exchange against the player's target, driven by the
// combat ticker in handleConn. The player strikes first; a kill ends the fight,
// a lethal counter respawns the player at the Nexus. The whole fight plays out on
// the combat.* channel so a client/agent reads it as data.
func (s *session) combatRound() {
	m := s.player.Target
	if m == nil {
		return
	}
	pd := s.playerDamage()
	m.HP -= pd
	if m.HP < 0 {
		m.HP = 0
	}
	md := 0
	if m.HP > 0 {
		md = m.Damage - s.playerArmor()
		if md < 0 {
			md = 0
		}
		s.player.HP -= md
	}
	s.event(event.CombatRound, world.CombatRoundPayload{
		Mob: m.ID, MobHP: m.HP, MobMaxHP: m.MaxHP, PlayerDmg: pd, MobDmg: md, HP: s.player.HP,
	})
	switch {
	case m.HP <= 0:
		s.removeMob(m)
		s.player.Target = nil
		s.player.XP += 5
		s.event(event.CombatEnd, world.CombatEndPayload{Mob: m.ID, Result: "killed"})
		s.line("You put " + m.Name + " down. The tunnels go quiet.")
		s.event(event.CharVitals, s.player.Vitals())
	case s.player.HP <= 0:
		s.player.Target = nil
		s.player.HP = s.player.MaxHP
		s.player.RoomID = s.w.Start().ID
		s.event(event.CombatEnd, world.CombatEndPayload{Mob: m.ID, Result: "died"})
		s.event(event.CharDied, world.CharDiedPayload{RespawnRoom: s.w.Start().ID, HP: s.player.HP, MaxHP: s.player.MaxHP})
		s.line("The dark takes you -- and the Grid, stubborn, reknits you at the Cracked Nexus.")
		s.event(event.CharVitals, s.player.Vitals())
		s.sendScene()
	default:
		s.event(event.CharVitals, s.player.Vitals())
	}
}

// considerLine sizes up a mob relative to the player (src/world.ts consider).
func considerLine(p *world.Player, m *world.Mob) string {
	ratio := float64(m.MaxHP) / float64(p.MaxHP)
	switch {
	case ratio < 0.5:
		return fmt.Sprintf("You could put %s down without breaking a sweat.", m.Name)
	case ratio < 0.9:
		return fmt.Sprintf("%s would give you a tussle, but the odds are yours.", capitalize(m.Name))
	case ratio < 1.3:
		return fmt.Sprintf("%s looks like an even match. Bring an antidote.", capitalize(m.Name))
	case ratio < 2.0:
		return fmt.Sprintf("%s would likely gut you. Think twice.", capitalize(m.Name))
	default:
		return fmt.Sprintf("Attacking %s would be a quiet way to die.", m.Name)
	}
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// equipmentLine summarises what the player is wearing, in slot order.
func (s *session) equipmentLine() string {
	worn := make([]string, 0, len(world.EquipSlots))
	for _, sl := range world.EquipSlots {
		if id, ok := s.player.Equipment[sl]; ok {
			worn = append(worn, sl+": "+world.ItemName(id))
		}
	}
	if len(worn) == 0 {
		return "You are wearing nothing."
	}
	return "You are wearing -- " + strings.Join(worn, "; ") + "."
}

// whoLine lists who is online with their titles. A shared session registry for
// real multiplayer presence lands later; for now it reads back the player
// themselves (title after the name), which is what the contract checks.
func (s *session) whoLine() string {
	name := s.player.Name
	if s.player.Title != "" {
		name += " " + s.player.Title
	}
	return "Online: " + name + "."
}

// useTrait fires the character's racial signature ability (src/world.ts useTrait):
// cooldown-gated, with blocking conditions that do NOT spend the cooldown. The
// effects that need a live fight (chromed's Overclock, elf's Vanish) land with
// combat; the self-contained ones (Requisition, the heals, Forage) work now.
func (s *session) useTrait() {
	r := world.RaceByID(s.player.Race)
	ab := r.Ability
	now := time.Now()
	if now.Before(s.player.TraitReadyAt) {
		s.line(fmt.Sprintf("%s is still recharging. (%ds)", ab.Name, int(s.player.TraitReadyAt.Sub(now).Seconds())+1))
		return
	}

	// Blocking conditions spend no cooldown.
	switch s.player.Race {
	case "chromed":
		s.line("You spin your augments up to a scream, but there's nothing here to dump the charge into.")
		return
	case "elf":
		s.line("You ready to slip the net, but there is no fight here to vanish from.")
		return
	case "dustkin":
		if !s.room().Outdoors {
			s.line("Nothing to forage in here. You need the open wastes under the sky.")
			return
		}
	}

	s.player.TraitReadyAt = now.Add(time.Duration(ab.CooldownMs) * time.Millisecond)
	heal := func(amount int, prose string) {
		s.player.HP += amount
		if s.player.HP > s.player.MaxHP {
			s.player.HP = s.player.MaxHP
		}
		s.line(fmt.Sprintf("%s (+%d hp)", prose, amount))
		s.event(event.CharVitals, s.player.Vitals())
	}
	switch s.player.Race {
	case "human":
		coin := 15 + rand.Intn(16)
		s.player.Gold += coin
		s.line(fmt.Sprintf("You flash credentials nobody bothers to check. The registry still provides for its own. (+%d gold)", coin))
		s.event(event.CharVitals, s.player.Vitals())
	case "ghoul":
		heal(25, "Rad-scoured flesh knits itself shut.")
	case "revenant":
		heal(15, "You reach into the dead Grid and draw back a little of its cold life.")
	case "vatborn":
		heal(12, "You print a field stim from raw salvage and jab it home.")
	case "dustkin":
		coin := 5 + rand.Intn(11)
		s.player.Gold += coin
		s.line(fmt.Sprintf("You work the open pan and turn up something worth keeping. (+%d gold)", coin))
		s.event(event.CharVitals, s.player.Vitals())
	default:
		s.line(ab.Desc + ".")
	}
}

// roomAction finds an unresolved contextual action in the current room by verb.
func (s *session) roomAction(verb string) (world.Action, bool) {
	r := s.room()
	for _, a := range r.Actions {
		if a.Verb == verb && !s.resolved[r.ID+":"+a.Verb] {
			return a, true
		}
	}
	return world.Action{}, false
}

// resolve applies a contextual moral choice: the prose is the consequence, the
// @event channel carries the same truth as data, and the choice is persisted and
// remembered (it leaves the affordance set).
func (s *session) resolve(a world.Action) {
	rid := s.player.RoomID
	switch a.Verb {
	case "defend":
		s.shiftMorality(10)
		s.markResolved(rid, "defend", "join")
		s.line("You set your back to the refugees and your face to the Front. The wind tastes of cinders. They could kill you here, and the network would log it, and someone, someday, would read that you stood. The captain decides you are not worth the ammunition. The refugees do not thank you; they are too busy living. That is thanks enough.")
	case "join":
		s.shiftMorality(-15)
		s.player.Gold += 25
		s.player.Faction = "Cinder Front"
		s.markResolved(rid, "defend", "join")
		s.line("You take the Front's coin. It is warm, which is worse. The refugees watch you pocket it and say nothing; they have learned that names are safer unspoken. The Grid logs the transaction. It will remember this longer than you will.")
	case "witness":
		s.shiftMorality(5)
		s.markResolved(rid, "witness")
		s.line("You speak the names the static is forgetting -- the makers, the mapped, the ones who fell before the federation had a word for falling. The wall does not answer. But the saying is the point: memory is the one thing the dead network cannot delete while someone still chooses to remember. You leave a little of yourself in the static, and carry a little of them out.")
	case "defy":
		s.shiftMorality(10)
		s.markResolved(rid, "join", "defy")
		s.line("You spit at the recruiter's boots and walk past the crate without slowing. A few in the crowd watch you do it, and stand a little straighter. The Front marks your face for it. So be it.")
	default:
		s.line("Nothing happens.")
		return
	}
	s.persist()
	s.event(event.CharAffects, s.player.Affects())
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.RoomActions, s.actions(s.room()))
}

func (s *session) shiftMorality(d int) {
	s.player.Morality += d
	if s.player.Morality > moralityCeil {
		s.player.Morality = moralityCeil
	}
	if s.player.Morality < moralityFloor {
		s.player.Morality = moralityFloor
	}
}

func (s *session) markResolved(roomID string, verbs ...string) {
	for _, v := range verbs {
		s.resolved[roomID+":"+v] = true
	}
}
