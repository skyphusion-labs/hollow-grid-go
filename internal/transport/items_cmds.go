package transport

import (
	"fmt"

	"github.com/SkyPhusion/hollow-grid-go/internal/event"
	"github.com/SkyPhusion/hollow-grid-go/internal/world"
)

func (s *session) cmdGet(arg string) {
	if arg == "" {
		s.line("Get what?")
		return
	}
	roomID := s.player.RoomID
	id, ok := s.srv.findGroundItem(roomID, arg)
	if !ok {
		s.line(`There's no "` + arg + `" here to take.`)
		return
	}
	if !s.srv.groundRemove(roomID, id, 1) {
		s.line(`There's no "` + arg + `" here to take.`)
		return
	}
	s.player.AddItem(id)
	name := world.ItemName(id)
	s.line("You pick up " + name + ".")
	s.srv.hub.BroadcastRoom(roomID, s.player.Name+" picks up "+name+".", s.player.Name)
}

func (s *session) cmdDrop(arg string) {
	if arg == "" {
		s.line("Drop what?")
		return
	}
	id, ok := s.player.FindInventory(arg)
	if !ok {
		s.line(`You aren't carrying "` + arg + `".`)
		return
	}
	s.player.RemoveFromInventory(id)
	s.srv.groundAdd(s.player.RoomID, id, 1)
	name := world.ItemName(id)
	s.line("You drop " + name + ".")
	s.srv.hub.BroadcastRoom(s.player.RoomID, s.player.Name+" drops "+name+".", s.player.Name)
}

func (s *session) cmdUse(arg string) {
	if arg == "" {
		s.line("Use what?")
		return
	}
	id, ok := s.player.FindInventory(arg)
	if !ok {
		s.line(`You aren't carrying "` + arg + `".`)
		return
	}
	it, ok := world.ItemByID(id)
	if !ok || it.Use == nil {
		s.line("You can't figure out how to use " + world.ItemName(id) + ".")
		return
	}
	switch it.Use.Effect {
	case "cure_poison":
		if !s.player.Poisoned {
			s.line("You aren't poisoned. Best to save it.")
			return
		}
		s.player.Poisoned = false
		s.player.RemoveFromInventory(id)
		s.line("The antivenom burns cold down your throat; the venom recedes. You are cured.")
	case "heal":
		if s.player.HP >= s.player.MaxHP {
			s.line("You're already at full health.")
			return
		}
		s.player.HP += it.Use.Amount
		if s.player.HP > s.player.MaxHP {
			s.player.HP = s.player.MaxHP
		}
		s.player.RemoveFromInventory(id)
		s.line(fmt.Sprintf("You jolt yourself with %s. (HP %d/%d)", it.Name, s.player.HP, s.player.MaxHP))
	case "drug":
		s.player.RemoveFromInventory(id)
		s.player.HP = s.player.MaxHP
		s.player.Morality -= 10
		s.player.Addiction++
		s.line(fmt.Sprintf(
			"The dust hits like a sunrise behind your eyes. Pain forgotten, body humming, you feel whole again. (HP %d/%d)",
			s.player.HP, s.player.MaxHP,
		))
		if s.player.Addiction >= 3 {
			s.line("But the wanting is louder now. Your hands won't stop shaking when it fades.")
		}
	default:
		s.line("You can't figure out how to use " + it.Name + ".")
		return
	}
	s.persist()
	s.srv.hub.Sync(s.player)
	s.event(event.CharVitals, s.player.Vitals())
	s.event(event.CharAffects, s.player.Affects())
}

func (s *session) cmdExamine(arg string) {
	if arg == "" {
		s.line("Examine what?")
		return
	}
	if id, ok := s.player.FindInventory(arg); ok {
		if it, ok := world.ItemByID(id); ok && it.Desc != "" {
			s.line(it.Desc)
			return
		}
	}
	if id, ok := s.srv.findGroundItem(s.player.RoomID, arg); ok {
		if it, ok := world.ItemByID(id); ok && it.Desc != "" {
			s.line(it.Desc)
			return
		}
	}
	s.line(`You don't see any "` + arg + `" to examine.`)
}

func (s *session) cmdSay(msg string) {
	msg = SanitizePlayerText(msg)
	if msg == "" {
		s.line("Say what?")
		return
	}
	s.line(`You say, "` + msg + `"`)
	s.srv.hub.BroadcastRoom(s.player.RoomID, s.player.Name+` says, "`+msg+`"`, s.player.Name)
}
