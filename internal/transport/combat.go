package transport

import (
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

// combatRound resolves one exchange against the player's target, driven by the
// heartbeat in handleConn. The player strikes first; a kill ends the fight,
// a lethal counter respawns the player at the Nexus.
func (s *session) combatRound() {
	s.srv.combatMu.Lock()
	defer s.srv.combatMu.Unlock()

	m := s.player.Target
	if m == nil {
		return
	}
	if s.room().Mob(m.ID) == nil {
		s.player.Target = nil
		s.event(event.CombatEnd, world.CombatEndPayload{Mob: m.ID, Result: "gone"})
		s.line("Your quarry is gone. You stand down.")
		s.event(event.CharVitals, s.player.Vitals())
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
		s.finishMobKill(m)
	case s.player.HP <= 0:
		s.player.Target = nil
		hubCall, hubCancel := hubCtx()
		defer hubCancel()
		_ = s.srv.grid.RecordFallen(hubCall, s.w.Name, s.player.Name, s.player.RoomID, time.Now().UnixMilli())
		s.player.HP = s.player.MaxHP
		s.player.RoomID = s.w.Start().ID
		s.player.Poisoned = false
		s.event(event.CombatEnd, world.CombatEndPayload{Mob: m.ID, Result: "died"})
		s.event(event.CharDied, world.CharDiedPayload{RespawnRoom: s.w.Start().ID, HP: s.player.HP, MaxHP: s.player.MaxHP})
		s.line("The dark takes you -- and the Grid, stubborn, reknits you at the Cracked Nexus.")
		s.event(event.CharVitals, s.player.Vitals())
		s.sendScene()
	default:
		if md > 0 {
			if tmpl, ok := world.MobTemplate(m.ID); ok && tmpl.PoisonChance > 0 &&
				!s.player.Poisoned && !world.RaceByID(s.player.Race).PoisonImmune &&
				rand.Float64() < tmpl.PoisonChance {
				s.player.Poisoned = true
				s.line("Venom courses through your veins; you are POISONED. Seek an antidote.")
				s.event(event.CharVitals, s.player.Vitals())
			}
		}
		s.event(event.CharVitals, s.player.Vitals())
	}
}

func (s *session) finishMobKill(m *world.Mob) {
	roomID := s.player.RoomID
	tmpl, _ := world.MobTemplate(m.ID)

	s.srv.displaceMobFightersLocked(roomID, m.ID, m.Name, s.player.Name)

	s.srv.killMob(roomID, m)
	s.player.Target = nil

	for _, drop := range world.RollLoot(m.ID) {
		s.srv.groundAdd(roomID, drop, 1)
		dropName := world.ItemName(drop)
		s.line(fmt.Sprintf("%s drops %s.", capitalize(m.Name), dropName))
		s.srv.hub.BroadcastRoom(roomID, fmt.Sprintf("%s drops %s.", capitalize(m.Name), dropName), s.player.Name)
	}

	xp := tmpl.XP
	if xp <= 0 {
		xp = 5
	}
	if leveled := s.player.AwardXP(xp); leveled {
		s.line(fmt.Sprintf("*** You reach level %d! Max HP is now %d. ***", s.player.Level, s.player.MaxHP))
	}
	if m.ID == "ashmonger" {
		s.player.Morality += 20
		s.srv.hub.BroadcastAll("Word races across the wastes: the Ashmonger is dead. The Cinder Front's heart is broken.\r\n")
	}

	s.deed("slain")
	s.recordTrace(roomID, "slain", s.player.Name+" slew "+m.Name+" here.")
	s.srv.hub.BroadcastRoom(roomID, s.player.Name+" has slain "+m.Name+".", s.player.Name)
	s.event(event.CombatEnd, world.CombatEndPayload{Mob: m.ID, Result: "killed"})
	s.line("You have slain " + m.Name + "!  (+" + itoa(xp) + " xp)")
	s.persist()
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
}

// displaceMobFightersLocked clears other players fighting the same mob and
// notifies them (TS killMob parity). Caller must hold combatMu.
func (s *Server) displaceMobFightersLocked(roomID, mobID, mobName, except string) {
	type victim struct {
		lp  *livePlayer
		msg string
	}
	var victims []victim

	s.hub.mu.Lock()
	for name, lp := range s.hub.players {
		if name == except || lp.room != roomID || lp.plr == nil || lp.plr.Target == nil {
			continue
		}
		if lp.plr.Target.ID != mobID {
			continue
		}
		lp.plr.Target = nil
		endLine, err := event.Line(event.CombatEnd, world.CombatEndPayload{Mob: mobID, Result: "gone"})
		if err != nil {
			continue
		}
		vitLine, err := event.Line(event.CharVitals, lp.plr.Vitals())
		if err != nil {
			continue
		}
		msg := capitalize(mobName) + " falls before you can finish it.\r\n" + endLine + crlf + vitLine + crlf
		victims = append(victims, victim{lp: lp, msg: msg})
	}
	s.hub.mu.Unlock()

	for _, v := range victims {
		pushReliable(v.lp, v.msg)
	}
}

func (s *session) cmdFlee() {
	s.srv.combatMu.Lock()
	defer s.srv.combatMu.Unlock()

	if s.player.Target == nil {
		s.line("You're not fighting anything.")
		return
	}
	fled := s.player.Target.ID
	s.player.Target = nil
	s.event(event.CombatEnd, world.CombatEndPayload{Mob: fled, Result: "fled"})
	s.event(event.CharVitals, s.player.Vitals())
	s.line("You break off and catch your breath.")
	s.srv.hub.BroadcastRoom(s.player.RoomID, s.player.Name+" flees the fight.", s.player.Name)
}

func (s *session) poisonTick() {
	if !s.player.Poisoned || world.RaceByID(s.player.Race).PoisonImmune {
		return
	}
	s.player.HP--
	if s.player.HP <= 0 {
		s.player.HP = s.player.MaxHP
		s.player.RoomID = s.w.Start().ID
		s.player.Target = nil
		s.player.Poisoned = false
		s.line("The venom finishes what the wastes started...")
		s.event(event.CharDied, world.CharDiedPayload{RespawnRoom: s.w.Start().ID, HP: s.player.HP, MaxHP: s.player.MaxHP})
		s.sendScene()
		return
	}
	s.line(fmt.Sprintf("The venom gnaws at you. (HP %d/%d)", s.player.HP, s.player.MaxHP))
	s.event(event.CharVitals, s.player.Vitals())
}

func standingLabel(morality int) string {
	switch {
	case morality >= 50:
		return "a beacon of the wastes"
	case morality >= 20:
		return "well-regarded"
	case morality > -20:
		return "unproven"
	case morality > -50:
		return "shady"
	default:
		return "reviled"
	}
}

func (s *session) cmdStatus() {
	flags := make([]string, 0, 4)
	if s.player.Poisoned {
		flags = append(flags, "AFFLICTED")
	}
	if s.player.Addiction >= 3 {
		flags = append(flags, "ADDICTED")
	}
	if s.player.Faction == "front" {
		flags = append(flags, "CINDER FRONT")
	}
	if s.player.Faction == "ally" {
		flags = append(flags, "FRIEND OF THE ELVES")
	}
	moralSign := "+"
	if s.player.Morality < 0 {
		moralSign = ""
	}
	line2 := fmt.Sprintf("Gold %d   Standing: %s (%s%d)", s.player.Gold, standingLabel(s.player.Morality), moralSign, s.player.Morality)
	if len(flags) > 0 {
		line2 += "   [" + strings.Join(flags, ", ") + "]"
	}
	s.line(fmt.Sprintf("HP %d/%d   Level %d   XP %d/%d", s.player.HP, s.player.MaxHP, s.player.Level, s.player.XP, s.player.Level*100))
	s.line(line2)
}
