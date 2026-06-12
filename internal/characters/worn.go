package characters

import (
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/species"
)

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
	Tail         items.Item `yaml:"tail,omitempty"` // Tail mutation slot
	ComponentBag items.Item `yaml:"componentbag,omitempty"`
}

// WornSlot describes one equipment slot: its yaml/slot key, its display
// label, and a pointer to the real slot field on the Worn struct. The
// pointer is live — mutating WornSlot.Item mutates the underlying field.
type WornSlot struct {
	Key   string      // slotKey, matches the yaml tag: "weapon","offhand","extraarm1",...
	Label string      // display label
	Item  *items.Item // pointer to the real slot field
}

// AllSlots returns every equipment slot in eq/struct order. This is the
// SINGLE SOURCE OF TRUTH for the slot list + labels — enumeration sites
// (GetAllWornItems, validateEquipmentItems, FindOnBody, the GMCP Worn
// builder) iterate this instead of re-listing fields, so adding a slot
// only requires touching the struct and this method. A reflection guard
// test (worn_allslots_test.go) fails if a new items.Item field on Worn
// is not registered here.
func (w *Worn) AllSlots() []WornSlot {
	return []WornSlot{
		{"weapon", "Weapon", &w.Weapon},
		{"offhand", "Offhand", &w.Offhand},
		{"extraarm1", "Arm 3", &w.ExtraArm1}, {"extraarm2", "Arm 4", &w.ExtraArm2},
		{"extraarm3", "Arm 5", &w.ExtraArm3}, {"extraarm4", "Arm 6", &w.ExtraArm4},
		{"head", "Head", &w.Head}, {"neck", "Neck", &w.Neck}, {"shoulders", "Shoulders", &w.Shoulders},
		{"body", "Body", &w.Body}, {"back", "Back", &w.Back}, {"belt", "Belt", &w.Belt},
		{"wrist1", "Wrist", &w.Wrist1}, {"wrist2", "Wrist", &w.Wrist2},
		{"extrawrist1", "Wrist 3", &w.ExtraWrist1}, {"extrawrist2", "Wrist 4", &w.ExtraWrist2},
		{"extrawrist3", "Wrist 5", &w.ExtraWrist3}, {"extrawrist4", "Wrist 6", &w.ExtraWrist4},
		{"gloves", "Gloves", &w.Gloves}, {"ring", "Ring", &w.Ring}, {"ring2", "Ring", &w.Ring2},
		{"legs", "Legs", &w.Legs}, {"feet", "Feet", &w.Feet},
		{"tail", "Tail", &w.Tail}, {"componentbag", "Component Bag", &w.ComponentBag},
	}
}

// LoadEquippedRangedWeapons chambers/nocks any ranged weapon already worn in
// the main-hand or offhand slot (sets Item.Loaded = true). Called at mob spawn
// so an archer on duty starts ready to fire — its behavior tree opens with a
// shot instead of burning its first round on a reload. A player who loots such
// a weapon inherits the loaded state (one free shot) — an accepted edge case.
func (w *Worn) LoadEquippedRangedWeapons() {
	if w.Weapon.IsRangedWeapon() {
		w.Weapon.Loaded = true
	}
	if w.Offhand.IsRangedWeapon() {
		w.Offhand.Loaded = true
	}
}

// StatMod sums stat-mod contributions across every worn slot.
//
// NOTE: this deliberately keeps the explicit per-field sum rather than
// iterating AllSlots(). It is a hot path — reached per stat-name lookup
// via Character.StatMod from combat math (magical_mitigation,
// conviction_mitigation) and per-tick regen (resources.go recovery
// percentages), multiplied across every combatant/mob — and AllSlots()
// allocates a 25-element slice per call. The reflection guard test on
// AllSlots() still covers slot-list completeness.
// (keep in sync with AllSlots())
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
		w.Tail.StatMod(stat...) +
		w.ComponentBag.StatMod(stat...)
}

func (w *Worn) EnableAll() {
	for _, s := range w.AllSlots() {
		if s.Item.ItemId < 0 {
			*s.Item = items.Item{}
		}
	}
}

func (w *Worn) GetAllItems() []items.Item {
	iList := []items.Item{}
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			iList = append(iList, *s.Item)
		}
	}
	return iList
}

// GetAllItemsWithEmptySlots returns all equipped items plus zero-value marker
// items for empty paired slots (Ring, Wrist). This lets callers detect that
// a second slot of a given type exists but is unfilled. Single slots that are
// empty are not included — only paired slots need the marker because the
// caller cannot otherwise distinguish "no slot" from "empty second slot."
func (w *Worn) GetAllItemsWithEmptySlots() []items.Item {
	iList := w.GetAllItems()

	// For paired slots, if one is filled and the other empty, add a typed
	// marker so the caller knows the empty slot exists.
	if w.Ring.ItemId <= 0 && !w.Ring.IsDisabled() {
		iList = append(iList, items.Item{Spec: &items.ItemSpec{Type: items.Ring}})
	}
	if w.Ring2.ItemId <= 0 && !w.Ring2.IsDisabled() {
		iList = append(iList, items.Item{Spec: &items.ItemSpec{Type: items.Ring}})
	}
	if w.Wrist1.ItemId <= 0 && !w.Wrist1.IsDisabled() {
		iList = append(iList, items.Item{Spec: &items.ItemSpec{Type: items.Wrist}})
	}
	if w.Wrist2.ItemId <= 0 && !w.Wrist2.IsDisabled() {
		iList = append(iList, items.Item{Spec: &items.ItemSpec{Type: items.Wrist}})
	}

	return iList
}

// GetAllItemPtrs returns pointers to all equipped item slots with valid items.
// Used by the enchantment tick system to modify items in-place.
func (w *Worn) GetAllItemPtrs() []*items.Item {
	ptrs := make([]*items.Item, 0, 24)
	for _, s := range w.AllSlots() {
		if s.Item.ItemId > 0 {
			ptrs = append(ptrs, s.Item)
		}
	}
	return ptrs
}

// IsBlockedBy2H returns true if the given slot is the second slot in a hand
// pair where the first slot holds a 2-handed weapon. Used by the inventory
// template to hide "Offhand: -nothing-" when a 2H is wielded, etc.
func (w *Worn) IsBlockedBy2H(slot string) bool {
	// Map each "second" slot to its pair's "first" slot
	partnerMap := map[string]*items.Item{
		"offhand":     &w.Weapon,
		"extra arm 2": &w.ExtraArm1,
		"extra arm 4": &w.ExtraArm3,
	}
	partner, ok := partnerMap[slot]
	if !ok || partner.ItemId < 1 {
		return false
	}
	return partner.GetSpec().Hands >= 2
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
	case "worn - tail":
		return &w.Tail
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
		string(items.Tail),
		string(items.ComponentBag),
	}
}

func (c *Character) HandsRequired(i items.Item) int {

	if i.ItemId < 1 {
		return 0
	}

	iSpec := i.GetSpec()

	// Shooting weapnos don't benefit from creature size
	// when determining how many hands they require
	if iSpec.Subtype == items.Shooting {
		return iSpec.Hands
	}

	speciesInfo := species.GetSpecies(c.SpeciesId)
	if speciesInfo.Size == species.Large {
		return 1
	}

	if speciesInfo.Size == species.Small {
		return iSpec.Hands + 1
	}

	return iSpec.Hands
}

// HasShield returns true if the character is wielding a shield in offhand,
// or if the species has natural bash ability (earth elemental, etc.).
func (c *Character) HasShield() bool {
	// Species-based natural bash (elementals, golems, etc.)
	if sp := species.GetSpecies(c.SpeciesId); sp != nil && sp.NaturalBash {
		return true
	}
	return c.HasAnyShield()
}

// IsDualWielding returns true if character has weapons in both hands
func (c *Character) IsDualWielding() bool {
	if c.Equipment.Weapon.ItemId <= 0 || c.Equipment.Offhand.ItemId <= 0 {
		return false
	}
	// Dual wielding means both are weapons
	weaponSpec := c.Equipment.Weapon.GetSpec()
	offhandSpec := c.Equipment.Offhand.GetSpec()
	return weaponSpec.Type == items.Weapon && offhandSpec.Type == items.Weapon
}

// IsUnarmed returns true if character has no weapon equipped
func (c *Character) IsUnarmed() bool {
	return c.Equipment.Weapon.ItemId <= 0
}

// IsUnarmedStyle returns true if the character fights with fists, claws, or
// bare hands — anything that uses unarmed-combat skill. These fighters cannot
// parry (no weapon to deflect with) but get a speed bonus instead.
func (c *Character) IsUnarmedStyle() bool {
	if c.Equipment.Weapon.ItemId <= 0 {
		return true
	}
	sub := c.Equipment.Weapon.GetSpec().Subtype
	return sub == items.Fist || sub == items.Claws
}

// GetAllWornItems returns every equipped item across ALL slots, in Worn-struct
// (eq) order. (Previously this omitted the slots added later — Shoulders, Back,
// Wrist1/2, ExtraWrist1-4, ExtraArm3/4, Ring2, Tail, ComponentBag — which meant
// items in those slots never contributed their WornBuffIds via buffs.go and
// were missed by other callers. Now complete.)
func (c *Character) GetAllWornItems() []items.Item {
	wornItems := []items.Item{}
	for _, s := range c.Equipment.AllSlots() {
		if s.Item.ItemId > 0 {
			wornItems = append(wornItems, *s.Item)
		}
	}
	return wornItems
}

// GetGearValue sums the spec value of every equipped item across ALL slots
// (via AllSlots(), so it can't drift when slots are added). Used by the
// goal-planner's gear heuristic.
func (c *Character) GetGearValue() int {
	value := 0
	for _, s := range c.Equipment.AllSlots() {
		if s.Item.ItemId > 0 {
			value += s.Item.GetSpec().Value
		}
	}
	return value
}

// wearWeaponOrShield handles pair-based placement for weapons and offhands.
// Returns the same tuple as Wear. Caller is responsible for calling
// reapplyPermabuffs (this helper calls it for 2H and shield cases internally
// to preserve pre-refactor semantics).
func (c *Character) wearWeaponOrShield(i items.Item, spec items.ItemSpec, iHandsRequired int, canDualWield bool) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	pairs := c.GetHandPairs()
	isShield := spec.Type == items.Offhand

	if iHandsRequired >= 2 {
		freePair := FindFirstFreePair(pairs)
		if freePair == nil {
			freePair = FindCheapestPairToDisplace(pairs)
		}
		if freePair == nil {
			return returnItems, false, `You have no free pair of hands for a two-handed weapon.`
		}
		if !freePair.First.IsEmpty() && freePair.First.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.First.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.Second.IsEmpty() && freePair.Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + freePair.Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		if !freePair.First.IsEmpty() {
			returnItems = append(returnItems, *freePair.First.ItemPtr)
		}
		if !freePair.IsHalfPair() && !freePair.Second.IsEmpty() {
			returnItems = append(returnItems, *freePair.Second.ItemPtr)
		}
		*freePair.First.ItemPtr = i
		if !freePair.IsHalfPair() {
			*freePair.Second.ItemPtr = items.Item{}
		}
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	if isShield {
		slot := c.FindFirstEmptySlot(pairs, true)
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
		if pairs[0].First.Is2H(c) {
			return returnItems, false, `Your two-handed weapon leaves no room for a shield.`
		}
		if pairs[0].Second.ItemPtr.IsCursed() {
			return returnItems, false, `Your ` + pairs[0].Second.ItemPtr.DisplayName() + ` is cursed and prevents you from removing it.`
		}
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = i
		c.reapplyPermabuffs()
		return returnItems, true, ``
	}

	// 1H weapon
	bothMartial := spec.Subtype == items.Claws && c.Equipment.Weapon.GetSpec().Subtype == items.Claws

	slot := c.FindFirstEmptySlot(pairs, false)
	if slot != nil {
		if slot.Label == "offhand" && !canDualWield && !bothMartial {
			slot = nil
			for pi := 1; pi < len(pairs); pi++ {
				p := &pairs[pi]
				if p.First.Is2H(c) {
					continue
				}
				if p.First.IsEmpty() {
					slot = &p.First
					break
				}
				if !p.IsHalfPair() && p.Second.IsEmpty() {
					slot = &p.Second
					break
				}
			}
		}
		if slot != nil {
			*slot.ItemPtr = i
			c.reapplyPermabuffs()
			return returnItems, true, ``
		}
	}

	// No empty slots — displace Weapon slot (arm 1)
	if c.Equipment.Weapon.IsCursed() {
		return returnItems, false, `Your ` + c.Equipment.Weapon.DisplayName() + ` is cursed and prevents you from removing it.`
	}
	if pairs[0].First.Is2H(c) && !pairs[0].Second.IsEmpty() {
		returnItems = append(returnItems, *pairs[0].Second.ItemPtr)
		*pairs[0].Second.ItemPtr = items.Item{}
	}
	returnItems = append(returnItems, c.Equipment.Weapon)
	c.Equipment.Weapon = i
	c.reapplyPermabuffs()
	return returnItems, true, ``
}

// wearArmorSlot handles placement for non-weapon equipment (armor, rings, wrists,
// back, shoulders, component bag, tail). Returns the same tuple as Wear.
// Does NOT call reapplyPermabuffs — the caller handles that (to preserve the
// pre-refactor semantics where reapplyPermabuffs is called with returnItems).
func (c *Character) wearArmorSlot(i items.Item, spec items.ItemSpec) (returnItems []items.Item, newItemWorn bool, failureReason string) {
	switch spec.Type {
	case items.Head:
		if c.Equipment.Head.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Head)
		c.Equipment.Head = i
	case items.Neck:
		if c.Equipment.Neck.IsDisabled() {
			return returnItems, false, `You can't wear things on your neck.`
		}
		returnItems = append(returnItems, c.Equipment.Neck)
		c.Equipment.Neck = i
	case items.Body:
		if c.Equipment.Body.IsDisabled() {
			return returnItems, false, `You can't wear things on your body.`
		}
		returnItems = append(returnItems, c.Equipment.Body)
		c.Equipment.Body = i
	case items.Belt:
		if c.Equipment.Belt.IsDisabled() {
			return returnItems, false, `You can't wear things on your head.`
		}
		returnItems = append(returnItems, c.Equipment.Belt)
		c.Equipment.Belt = i
	case items.Gloves:
		if c.Equipment.Gloves.IsDisabled() {
			return returnItems, false, `You can't wear things as gloves.`
		}
		returnItems = append(returnItems, c.Equipment.Gloves)
		c.Equipment.Gloves = i
	case items.Ring:
		if c.Equipment.Ring.IsDisabled() && c.Equipment.Ring2.IsDisabled() {
			return returnItems, false, `You can't wear rings.`
		}
		if !c.Equipment.Ring.IsDisabled() && c.Equipment.Ring.ItemId == 0 {
			c.Equipment.Ring = i
		} else if !c.Equipment.Ring2.IsDisabled() && c.Equipment.Ring2.ItemId == 0 {
			c.Equipment.Ring2 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Ring)
			c.Equipment.Ring = i
		}
	case items.Wrist:
		if c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist2.IsDisabled() {
			return returnItems, false, `You can't wear things on your wrists.`
		}
		if !c.Equipment.Wrist1.IsDisabled() && c.Equipment.Wrist1.ItemId == 0 {
			c.Equipment.Wrist1 = i
		} else if !c.Equipment.Wrist2.IsDisabled() && c.Equipment.Wrist2.ItemId == 0 {
			c.Equipment.Wrist2 = i
		} else if c.ExtraArms >= 1 && !c.Equipment.ExtraWrist1.IsDisabled() && c.Equipment.ExtraWrist1.ItemId == 0 {
			c.Equipment.ExtraWrist1 = i
		} else if c.ExtraArms >= 2 && !c.Equipment.ExtraWrist2.IsDisabled() && c.Equipment.ExtraWrist2.ItemId == 0 {
			c.Equipment.ExtraWrist2 = i
		} else if c.ExtraArms >= 3 && !c.Equipment.ExtraWrist3.IsDisabled() && c.Equipment.ExtraWrist3.ItemId == 0 {
			c.Equipment.ExtraWrist3 = i
		} else if c.ExtraArms >= 4 && !c.Equipment.ExtraWrist4.IsDisabled() && c.Equipment.ExtraWrist4.ItemId == 0 {
			c.Equipment.ExtraWrist4 = i
		} else {
			returnItems = append(returnItems, c.Equipment.Wrist1)
			c.Equipment.Wrist1 = i
		}
	case items.Back:
		if c.Equipment.Back.IsDisabled() {
			return returnItems, false, `You can't wear things on your back.`
		}
		returnItems = append(returnItems, c.Equipment.Back)
		c.Equipment.Back = i
	case items.Shoulders:
		if c.Equipment.Shoulders.IsDisabled() {
			return returnItems, false, `You can't wear things on your shoulders.`
		}
		returnItems = append(returnItems, c.Equipment.Shoulders)
		c.Equipment.Shoulders = i
	case items.ComponentBag:
		if c.Equipment.ComponentBag.IsDisabled() {
			return returnItems, false, `You can't equip a component bag.`
		}
		returnItems = append(returnItems, c.Equipment.ComponentBag)
		c.Equipment.ComponentBag = i
		c.SortComponentItems()
	case items.Legs:
		if c.Equipment.Legs.IsDisabled() {
			return returnItems, false, `You can't wear things on your legs.`
		}
		returnItems = append(returnItems, c.Equipment.Legs)
		c.Equipment.Legs = i
	case items.Feet:
		if c.Equipment.Feet.IsDisabled() {
			return returnItems, false, `You can't wear things on your feet.`
		}
		returnItems = append(returnItems, c.Equipment.Feet)
		c.Equipment.Feet = i
	case items.Tail:
		if c.Equipment.Tail.IsDisabled() {
			return returnItems, false, `You don't have a tail to attach that to.`
		}
		returnItems = append(returnItems, c.Equipment.Tail)
		c.Equipment.Tail = i
	default:
		return returnItems, false, `Unrecognized object.`
	}
	return returnItems, true, ``
}

func (c *Character) Wear(i items.Item) (returnItems []items.Item, newItemWorn bool, failureReason string) {

	i.Validate()

	spec := i.GetSpec()

	if spec.Type != items.Weapon && spec.Subtype != items.Wearable {
		return returnItems, false, `That item cannot be equipped.`
	}

	// Min-Strength wield gate — heavy bows and arbalests require a minimum
	// Strength to operate. Checked before HandsRequired so the rejection is
	// immediate and consistent for all callers.
	if spec.MinStrength > 0 && c.Stats.Strength.ValueAdj < spec.MinStrength {
		return returnItems, false, `You aren't strong enough to handle ` + i.DisplayName() + `.`
	}

	iHandsRequired := c.HandsRequired(i)
	if iHandsRequired > 2 {
		return returnItems, false, `That requires too many hands.`
	}

	// Weapon + shield placement uses pair-based logic.
	if spec.Type == items.Weapon || spec.Type == items.Offhand {
		return c.wearWeaponOrShield(i, spec, iHandsRequired, c.CanDualWield())
	}

	// Armor + non-weapon slots use the simple switch.
	returnItems, newItemWorn, failureReason = c.wearArmorSlot(i, spec)
	if newItemWorn {
		c.reapplyPermabuffs(returnItems...)
	}
	return returnItems, newItemWorn, failureReason
}

func (c *Character) RemoveFromBody(i items.Item) bool {

	if i.Equals(c.Equipment.Weapon) {
		c.Equipment.Weapon = items.Item{}
	} else if i.Equals(c.Equipment.Offhand) {
		c.Equipment.Offhand = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm1) {
		c.Equipment.ExtraArm1 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm2) {
		c.Equipment.ExtraArm2 = items.Item{}
	} else if i.Equals(c.Equipment.Head) {
		c.Equipment.Head = items.Item{}
	} else if i.Equals(c.Equipment.Neck) {
		c.Equipment.Neck = items.Item{}
	} else if i.Equals(c.Equipment.Body) {
		c.Equipment.Body = items.Item{}
	} else if i.Equals(c.Equipment.Belt) {
		// If removing a bandolier, spill potions to backpack
		beltSpec := c.Equipment.Belt.GetSpec()
		if beltSpec.IsBandolier && len(c.PotionItems) > 0 {
			for _, pi := range c.PotionItems {
				c.Items = append(c.Items, pi)
			}
			c.PotionItems = nil
		}
		c.Equipment.Belt = items.Item{}
	} else if i.Equals(c.Equipment.Gloves) {
		c.Equipment.Gloves = items.Item{}
	} else if i.Equals(c.Equipment.Ring) {
		c.Equipment.Ring = items.Item{}
	} else if i.Equals(c.Equipment.Legs) {
		c.Equipment.Legs = items.Item{}
	} else if i.Equals(c.Equipment.Feet) {
		c.Equipment.Feet = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm3) {
		c.Equipment.ExtraArm3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraArm4) {
		c.Equipment.ExtraArm4 = items.Item{}
	} else if i.Equals(c.Equipment.Shoulders) {
		c.Equipment.Shoulders = items.Item{}
	} else if i.Equals(c.Equipment.Back) {
		c.Equipment.Back = items.Item{}
	} else if i.Equals(c.Equipment.Wrist1) {
		c.Equipment.Wrist1 = items.Item{}
	} else if i.Equals(c.Equipment.Wrist2) {
		c.Equipment.Wrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist1) {
		c.Equipment.ExtraWrist1 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist2) {
		c.Equipment.ExtraWrist2 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist3) {
		c.Equipment.ExtraWrist3 = items.Item{}
	} else if i.Equals(c.Equipment.ExtraWrist4) {
		c.Equipment.ExtraWrist4 = items.Item{}
	} else if i.Equals(c.Equipment.Ring2) {
		c.Equipment.Ring2 = items.Item{}
	} else if i.Equals(c.Equipment.Tail) {
		c.Equipment.Tail = items.Item{}
	} else if i.Equals(c.Equipment.ComponentBag) {
		// Spill component bag contents back to backpack
		for _, ci := range c.ComponentItems {
			c.Items = append(c.Items, ci)
		}
		c.ComponentItems = nil
		c.Equipment.ComponentBag = items.Item{}
	} else {
		return false
	}

	c.reapplyPermabuffs(i)

	return true
}

func (c *Character) Uncurse() []items.Item {

	uncursedList := []items.Item{}

	if c.Equipment.Weapon.IsCursed() {
		c.Equipment.Weapon.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Weapon)
	}

	if c.Equipment.Offhand.IsCursed() {
		c.Equipment.Offhand.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Offhand)
	}

	if c.Equipment.Head.IsCursed() {
		c.Equipment.Head.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Head)
	}

	if c.Equipment.Neck.IsCursed() {
		c.Equipment.Neck.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Neck)
	}

	if c.Equipment.Body.IsCursed() {
		c.Equipment.Body.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Body)
	}

	if c.Equipment.Belt.IsCursed() {
		c.Equipment.Belt.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Belt)
	}

	if c.Equipment.Gloves.IsCursed() {
		c.Equipment.Gloves.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Gloves)
	}

	if c.Equipment.Ring.IsCursed() {
		c.Equipment.Ring.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Ring)
	}

	if c.Equipment.Legs.IsCursed() {
		c.Equipment.Legs.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Legs)
	}

	if c.Equipment.Feet.IsCursed() {
		c.Equipment.Feet.Uncursed = true
		uncursedList = append(uncursedList, c.Equipment.Feet)
	}

	return uncursedList
}
