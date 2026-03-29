package characters

import "github.com/GoMudEngine/GoMud/internal/items"

type Worn struct {
	Weapon       items.Item `yaml:"weapon,omitempty"`
	Offhand      items.Item `yaml:"offhand,omitempty"`
	ExtraArm1    items.Item `yaml:"extraarm1,omitempty"` // Extra Arms mutation slot 1
	ExtraArm2    items.Item `yaml:"extraarm2,omitempty"` // Extra Arms mutation slot 2
	ExtraArm3    items.Item `yaml:"extraarm3,omitempty"` // Extra Arms mutation slot 3
	ExtraArm4    items.Item `yaml:"extraarm4,omitempty"` // Extra Arms mutation slot 4
	Head         items.Item `yaml:"head,omitempty"`
	Neck         items.Item `yaml:"neck,omitempty"`
	Shoulders    items.Item `yaml:"shoulders,omitempty"`
	Body         items.Item `yaml:"body,omitempty"`
	Back         items.Item `yaml:"back,omitempty"`
	Belt         items.Item `yaml:"belt,omitempty"`
	Wrist1       items.Item `yaml:"wrist1,omitempty"`
	Wrist2       items.Item `yaml:"wrist2,omitempty"`
	ExtraWrist1  items.Item `yaml:"extrawrist1,omitempty"`
	ExtraWrist2  items.Item `yaml:"extrawrist2,omitempty"`
	ExtraWrist3  items.Item `yaml:"extrawrist3,omitempty"`
	ExtraWrist4  items.Item `yaml:"extrawrist4,omitempty"`
	Gloves       items.Item `yaml:"gloves,omitempty"`
	Ring         items.Item `yaml:"ring,omitempty"`
	Ring2        items.Item `yaml:"ring2,omitempty"`
	Legs         items.Item `yaml:"legs,omitempty"`
	Feet         items.Item `yaml:"feet,omitempty"`
	ComponentBag items.Item `yaml:"componentbag,omitempty"`
}

func (w *Worn) StatMod(stat ...string) int {

	return w.Weapon.StatMod(stat...) +
		w.Offhand.StatMod(stat...) +
		w.ExtraArm1.StatMod(stat...) +
		w.ExtraArm2.StatMod(stat...) +
		w.ExtraArm3.StatMod(stat...) +
		w.ExtraArm4.StatMod(stat...) +
		w.Head.StatMod(stat...) +
		w.Neck.StatMod(stat...) +
		w.Shoulders.StatMod(stat...) +
		w.Body.StatMod(stat...) +
		w.Back.StatMod(stat...) +
		w.Belt.StatMod(stat...) +
		w.Wrist1.StatMod(stat...) +
		w.Wrist2.StatMod(stat...) +
		w.ExtraWrist1.StatMod(stat...) +
		w.ExtraWrist2.StatMod(stat...) +
		w.ExtraWrist3.StatMod(stat...) +
		w.ExtraWrist4.StatMod(stat...) +
		w.Gloves.StatMod(stat...) +
		w.Ring.StatMod(stat...) +
		w.Ring2.StatMod(stat...) +
		w.Legs.StatMod(stat...) +
		w.Feet.StatMod(stat...) +
		w.ComponentBag.StatMod(stat...)
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
	if w.ExtraArm3.ItemId < 0 {
		w.ExtraArm3 = items.Item{}
	}
	if w.ExtraArm4.ItemId < 0 {
		w.ExtraArm4 = items.Item{}
	}
	if w.Head.ItemId < 0 {
		w.Head = items.Item{}
	}
	if w.Neck.ItemId < 0 {
		w.Neck = items.Item{}
	}
	if w.Shoulders.ItemId < 0 {
		w.Shoulders = items.Item{}
	}
	if w.Body.ItemId < 0 {
		w.Body = items.Item{}
	}
	if w.Back.ItemId < 0 {
		w.Back = items.Item{}
	}
	if w.Belt.ItemId < 0 {
		w.Belt = items.Item{}
	}
	if w.Wrist1.ItemId < 0 {
		w.Wrist1 = items.Item{}
	}
	if w.Wrist2.ItemId < 0 {
		w.Wrist2 = items.Item{}
	}
	if w.ExtraWrist1.ItemId < 0 {
		w.ExtraWrist1 = items.Item{}
	}
	if w.ExtraWrist2.ItemId < 0 {
		w.ExtraWrist2 = items.Item{}
	}
	if w.ExtraWrist3.ItemId < 0 {
		w.ExtraWrist3 = items.Item{}
	}
	if w.ExtraWrist4.ItemId < 0 {
		w.ExtraWrist4 = items.Item{}
	}
	if w.Gloves.ItemId < 0 {
		w.Gloves = items.Item{}
	}
	if w.Ring.ItemId < 0 {
		w.Ring = items.Item{}
	}
	if w.Ring2.ItemId < 0 {
		w.Ring2 = items.Item{}
	}
	if w.Legs.ItemId < 0 {
		w.Legs = items.Item{}
	}
	if w.Feet.ItemId < 0 {
		w.Feet = items.Item{}
	}
	if w.ComponentBag.ItemId < 0 {
		w.ComponentBag = items.Item{}
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
	if w.ExtraArm3.ItemId > 0 {
		iList = append(iList, w.ExtraArm3)
	}
	if w.ExtraArm4.ItemId > 0 {
		iList = append(iList, w.ExtraArm4)
	}
	if w.Head.ItemId > 0 {
		iList = append(iList, w.Head)
	}
	if w.Neck.ItemId > 0 {
		iList = append(iList, w.Neck)
	}
	if w.Shoulders.ItemId > 0 {
		iList = append(iList, w.Shoulders)
	}
	if w.Body.ItemId > 0 {
		iList = append(iList, w.Body)
	}
	if w.Back.ItemId > 0 {
		iList = append(iList, w.Back)
	}
	if w.Belt.ItemId > 0 {
		iList = append(iList, w.Belt)
	}
	if w.Wrist1.ItemId > 0 {
		iList = append(iList, w.Wrist1)
	}
	if w.Wrist2.ItemId > 0 {
		iList = append(iList, w.Wrist2)
	}
	if w.ExtraWrist1.ItemId > 0 {
		iList = append(iList, w.ExtraWrist1)
	}
	if w.ExtraWrist2.ItemId > 0 {
		iList = append(iList, w.ExtraWrist2)
	}
	if w.ExtraWrist3.ItemId > 0 {
		iList = append(iList, w.ExtraWrist3)
	}
	if w.ExtraWrist4.ItemId > 0 {
		iList = append(iList, w.ExtraWrist4)
	}
	if w.Gloves.ItemId > 0 {
		iList = append(iList, w.Gloves)
	}
	if w.Ring.ItemId > 0 {
		iList = append(iList, w.Ring)
	}
	if w.Ring2.ItemId > 0 {
		iList = append(iList, w.Ring2)
	}
	if w.Legs.ItemId > 0 {
		iList = append(iList, w.Legs)
	}
	if w.Feet.ItemId > 0 {
		iList = append(iList, w.Feet)
	}
	if w.ComponentBag.ItemId > 0 {
		iList = append(iList, w.ComponentBag)
	}
	return iList
}

// GetAllItemPtrs returns pointers to all equipped item slots with valid items.
// Used by the enchantment tick system to modify items in-place.
func (w *Worn) GetAllItemPtrs() []*items.Item {
	ptrs := make([]*items.Item, 0, 24)
	if w.Weapon.ItemId > 0 {
		ptrs = append(ptrs, &w.Weapon)
	}
	if w.Offhand.ItemId > 0 {
		ptrs = append(ptrs, &w.Offhand)
	}
	if w.ExtraArm1.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraArm1)
	}
	if w.ExtraArm2.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraArm2)
	}
	if w.ExtraArm3.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraArm3)
	}
	if w.ExtraArm4.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraArm4)
	}
	if w.Head.ItemId > 0 {
		ptrs = append(ptrs, &w.Head)
	}
	if w.Neck.ItemId > 0 {
		ptrs = append(ptrs, &w.Neck)
	}
	if w.Shoulders.ItemId > 0 {
		ptrs = append(ptrs, &w.Shoulders)
	}
	if w.Body.ItemId > 0 {
		ptrs = append(ptrs, &w.Body)
	}
	if w.Back.ItemId > 0 {
		ptrs = append(ptrs, &w.Back)
	}
	if w.Belt.ItemId > 0 {
		ptrs = append(ptrs, &w.Belt)
	}
	if w.Wrist1.ItemId > 0 {
		ptrs = append(ptrs, &w.Wrist1)
	}
	if w.Wrist2.ItemId > 0 {
		ptrs = append(ptrs, &w.Wrist2)
	}
	if w.ExtraWrist1.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraWrist1)
	}
	if w.ExtraWrist2.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraWrist2)
	}
	if w.ExtraWrist3.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraWrist3)
	}
	if w.ExtraWrist4.ItemId > 0 {
		ptrs = append(ptrs, &w.ExtraWrist4)
	}
	if w.Gloves.ItemId > 0 {
		ptrs = append(ptrs, &w.Gloves)
	}
	if w.Ring.ItemId > 0 {
		ptrs = append(ptrs, &w.Ring)
	}
	if w.Ring2.ItemId > 0 {
		ptrs = append(ptrs, &w.Ring2)
	}
	if w.Legs.ItemId > 0 {
		ptrs = append(ptrs, &w.Legs)
	}
	if w.Feet.ItemId > 0 {
		ptrs = append(ptrs, &w.Feet)
	}
	if w.ComponentBag.ItemId > 0 {
		ptrs = append(ptrs, &w.ComponentBag)
	}
	return ptrs
}

// GetSlotPointer returns a mutable pointer to the item in the named slot.
func (w *Worn) GetSlotPointer(label string) *items.Item {
	switch label {
	case "wielded":
		return &w.Weapon
	case "offhand":
		return &w.Offhand
	case "extra arm 1":
		return &w.ExtraArm1
	case "extra arm 2":
		return &w.ExtraArm2
	case "extra arm 3":
		return &w.ExtraArm3
	case "extra arm 4":
		return &w.ExtraArm4
	case "worn - head":
		return &w.Head
	case "worn - neck":
		return &w.Neck
	case "worn - shoulders":
		return &w.Shoulders
	case "worn - body":
		return &w.Body
	case "worn - back":
		return &w.Back
	case "worn - belt":
		return &w.Belt
	case "worn - wrist", "worn - wrist1":
		return &w.Wrist1
	case "worn - wrist2":
		return &w.Wrist2
	case "extra wrist 1":
		return &w.ExtraWrist1
	case "extra wrist 2":
		return &w.ExtraWrist2
	case "extra wrist 3":
		return &w.ExtraWrist3
	case "extra wrist 4":
		return &w.ExtraWrist4
	case "worn - gloves":
		return &w.Gloves
	case "worn - ring":
		return &w.Ring
	case "worn - ring2":
		return &w.Ring2
	case "worn - legs":
		return &w.Legs
	case "worn - feet":
		return &w.Feet
	case "worn - componentbag":
		return &w.ComponentBag
	}
	return nil
}

func GetAllSlotTypes() []string {
	return []string{
		string(items.Weapon),
		string(items.Offhand),
		string(items.Head),
		string(items.Neck),
		string(items.Shoulders),
		string(items.Body),
		string(items.Back),
		string(items.Belt),
		string(items.Wrist),
		string(items.Gloves),
		string(items.Ring),
		string(items.Legs),
		string(items.Feet),
		string(items.ComponentBag),
	}
}
