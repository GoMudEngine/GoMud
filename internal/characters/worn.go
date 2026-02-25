package characters

import "github.com/GoMudEngine/GoMud/internal/items"

type Worn struct {
	Weapon    items.Item `yaml:"weapon,omitempty"`
	Offhand   items.Item `yaml:"offhand,omitempty"`
	ExtraArm1 items.Item `yaml:"extraarm1,omitempty"` // Extra Arms mutation slot 1
	ExtraArm2 items.Item `yaml:"extraarm2,omitempty"` // Extra Arms mutation slot 2
	Head      items.Item `yaml:"head,omitempty"`
	Neck      items.Item `yaml:"neck,omitempty"`
	Body      items.Item `yaml:"body,omitempty"`
	Belt      items.Item `yaml:"belt,omitempty"`
	Gloves    items.Item `yaml:"gloves,omitempty"`
	Ring      items.Item `yaml:"ring,omitempty"`
	Legs      items.Item `yaml:"legs,omitempty"`
	Feet      items.Item `yaml:"feet,omitempty"`
}

func (w *Worn) StatMod(stat ...string) int {

	return w.Weapon.StatMod(stat...) +
		w.Offhand.StatMod(stat...) +
		w.ExtraArm1.StatMod(stat...) +
		w.ExtraArm2.StatMod(stat...) +
		w.Head.StatMod(stat...) +
		w.Neck.StatMod(stat...) +
		w.Body.StatMod(stat...) +
		w.Belt.StatMod(stat...) +
		w.Gloves.StatMod(stat...) +
		w.Ring.StatMod(stat...) +
		w.Legs.StatMod(stat...) +
		w.Feet.StatMod(stat...)
}

func (w *Worn) EnableAll() {
	if w.Weapon.ItemId < 0 {
		w.Weapon = items.Item{}
	}
	if w.Offhand.ItemId < 0 {
		w.Offhand = items.Item{}
	}
	if w.ExtraArm1.ItemId < 0 {
		w.ExtraArm1 = items.Item{}
	}
	if w.ExtraArm2.ItemId < 0 {
		w.ExtraArm2 = items.Item{}
	}
	if w.Head.ItemId < 0 {
		w.Head = items.Item{}
	}
	if w.Neck.ItemId < 0 {
		w.Neck = items.Item{}
	}
	if w.Body.ItemId < 0 {
		w.Body = items.Item{}
	}
	if w.Belt.ItemId < 0 {
		w.Belt = items.Item{}
	}
	if w.Gloves.ItemId < 0 {
		w.Gloves = items.Item{}
	}
	if w.Ring.ItemId < 0 {
		w.Ring = items.Item{}
	}
	if w.Legs.ItemId < 0 {
		w.Legs = items.Item{}
	}
	if w.Feet.ItemId < 0 {
		w.Feet = items.Item{}
	}
}

func (w *Worn) GetAllItems() []items.Item {
	iList := []items.Item{}
	if w.Weapon.ItemId > 0 {
		iList = append(iList, w.Weapon)
	}
	if w.Offhand.ItemId > 0 {
		iList = append(iList, w.Offhand)
	}
	if w.ExtraArm1.ItemId > 0 {
		iList = append(iList, w.ExtraArm1)
	}
	if w.ExtraArm2.ItemId > 0 {
		iList = append(iList, w.ExtraArm2)
	}
	if w.Head.ItemId > 0 {
		iList = append(iList, w.Head)
	}
	if w.Neck.ItemId > 0 {
		iList = append(iList, w.Neck)
	}
	if w.Body.ItemId > 0 {
		iList = append(iList, w.Body)
	}
	if w.Belt.ItemId > 0 {
		iList = append(iList, w.Belt)
	}
	if w.Gloves.ItemId > 0 {
		iList = append(iList, w.Gloves)
	}
	if w.Ring.ItemId > 0 {
		iList = append(iList, w.Ring)
	}
	if w.Legs.ItemId > 0 {
		iList = append(iList, w.Legs)
	}
	if w.Feet.ItemId > 0 {
		iList = append(iList, w.Feet)
	}
	return iList
}

func GetAllSlotTypes() []string {
	return []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Body),
		string(items.Belt),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
	}
}
