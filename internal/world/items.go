package world

import "strings"

// Item is a piece of local salvage. Slot is one of the equip slots (weapon, head,
// body, hands, feet) or "" for a non-equippable consumable. Damage/armor are the
// combat leans it grants when worn. (src/items.ts.)
type Item struct {
	ID     string
	Name   string
	Slot   string
	Damage int
	Armor  int
}

// EquipSlots is the canonical equipment layout (the char.equipment fields).
var EquipSlots = []string{"weapon", "head", "body", "hands", "feet"}

// Starter is the weapon a fresh character wakes clutching, in this world.
const Starter = "shiv"

// Items is the local item table (a growing subset of src/items.ts).
var Items = map[string]Item{
	"shiv":     {ID: "shiv", Name: "a rusted shiv", Slot: "weapon", Damage: 3},
	"rebar":    {ID: "rebar", Name: "a length of rebar", Slot: "weapon", Damage: 6},
	"helm":     {ID: "helm", Name: "a dented scrap helm", Slot: "head", Armor: 1},
	"plating":  {ID: "plating", Name: "a sheet of scrap plating", Slot: "body", Armor: 2},
	"charm":    {ID: "charm", Name: "an elven charm", Slot: ""},
	"antidote": {ID: "antidote", Name: "an antidote vial", Slot: ""},
	"dust":     {ID: "dust", Name: "a packet of dust", Slot: ""},
	"shard":    {ID: "shard", Name: "the core shard", Slot: ""},
}

// ItemByID returns the item definition for an id.
func ItemByID(id string) (Item, bool) { it, ok := Items[id]; return it, ok }

// ItemName returns an item's display name (or the raw id if unknown).
func ItemName(id string) string {
	if it, ok := Items[id]; ok {
		return it.Name
	}
	return id
}

// CharEquipmentPayload is emitted as char.equipment: each slot holds an item id
// or null. Pointers so an empty slot serialises as JSON null (what the contract
// and the conformance suite expect).
type CharEquipmentPayload struct {
	Weapon *string `json:"weapon"`
	Head   *string `json:"head"`
	Body   *string `json:"body"`
	Hands  *string `json:"hands"`
	Feet   *string `json:"feet"`
}

// Equip renders the player's worn gear as a char.equipment payload.
func (p *Player) Equip() CharEquipmentPayload {
	slot := func(name string) *string {
		if id, ok := p.Equipment[name]; ok {
			v := id
			return &v
		}
		return nil
	}
	return CharEquipmentPayload{
		Weapon: slot("weapon"), Head: slot("head"), Body: slot("body"),
		Hands: slot("hands"), Feet: slot("feet"),
	}
}

// FindInventory resolves a typed arg to an inventory item id.
func (p *Player) FindInventory(arg string) (string, bool) { return p.findInventory(arg) }

// findInventory resolves a player's typed arg to an inventory item id: an exact
// id, or a case-insensitive substring of the id or display name.
func (p *Player) findInventory(arg string) (string, bool) {
	arg = strings.ToLower(strings.TrimSpace(arg))
	if arg == "" {
		return "", false
	}
	for _, id := range p.Inventory {
		if id == arg || strings.Contains(strings.ToLower(id), arg) ||
			strings.Contains(strings.ToLower(ItemName(id)), arg) {
			return id, true
		}
	}
	return "", false
}

// findEquipped resolves a typed arg to a worn item's slot+id.
func (p *Player) findEquipped(arg string) (slot, id string, ok bool) {
	arg = strings.ToLower(strings.TrimSpace(arg))
	for sl, id := range p.Equipment {
		if id == arg || strings.Contains(strings.ToLower(id), arg) ||
			strings.Contains(strings.ToLower(ItemName(id)), arg) {
			return sl, id, true
		}
	}
	return "", "", false
}

// removeFromInventory drops one copy of id from the pack.
func (p *Player) removeFromInventory(id string) {
	for i, have := range p.Inventory {
		if have == id {
			p.Inventory = append(p.Inventory[:i], p.Inventory[i+1:]...)
			return
		}
	}
}

// Wear moves an inventory item into its slot, returning the item and what it
// displaced (if anything) back into the pack.
func (p *Player) Wear(arg string) (Item, bool) {
	id, ok := p.findInventory(arg)
	if !ok {
		return Item{}, false
	}
	it, ok := ItemByID(id)
	if !ok || it.Slot == "" {
		return Item{}, false
	}
	if prev, worn := p.Equipment[it.Slot]; worn {
		p.Inventory = append(p.Inventory, prev)
	}
	p.removeFromInventory(id)
	p.Equipment[it.Slot] = id
	return it, true
}

// Unwear takes a worn item off and returns it to the pack.
func (p *Player) Unwear(arg string) (Item, bool) {
	slot, id, ok := p.findEquipped(arg)
	if !ok {
		return Item{}, false
	}
	delete(p.Equipment, slot)
	p.Inventory = append(p.Inventory, id)
	it, _ := ItemByID(id)
	return it, true
}

// HasItem reports whether the pack holds at least one copy of id.
func (p *Player) HasItem(id string) bool {
	for _, have := range p.Inventory {
		if have == id {
			return true
		}
	}
	return false
}

// AddItem puts one copy of id into the pack.
func (p *Player) AddItem(id string) { p.Inventory = append(p.Inventory, id) }

// RemoveFromInventory drops one copy of id from the pack.
func (p *Player) RemoveFromInventory(id string) { p.removeFromInventory(id) }

func (p *Player) InventoryNames() []string {
	names := make([]string, 0, len(p.Inventory))
	for _, id := range p.Inventory {
		names = append(names, ItemName(id))
	}
	return names
}
