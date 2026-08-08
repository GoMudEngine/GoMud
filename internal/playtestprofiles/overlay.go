package playtestprofiles

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// WorldChecks validates overlay and start-room references against loaded world data.
type WorldChecks struct {
	RoomExists func(roomID int) bool
	SpellOK    func(spellID string) bool
	ItemOK     func(itemID int) bool
	FlagOK     func(key, value string) error
}

// DefaultWorldChecks uses the live rooms/spells/items/quest-flag registries.
func DefaultWorldChecks() WorldChecks {
	return WorldChecks{
		RoomExists: func(roomID int) bool {
			return loadRoomExists(roomID)
		},
		SpellOK: func(spellID string) bool {
			return spells.GetSpell(spellID) != nil
		},
		ItemOK: func(itemID int) bool {
			return items.GetItemSpec(itemID) != nil
		},
		FlagOK: quests.ValidateFlag,
	}
}

// ApplyOverlays mutates u with start room and overlays. Fail-closed on unknown refs.
func ApplyOverlays(u *users.UserRecord, startRoom int, o Overlays, world WorldChecks) error {
	if u == nil || u.Character == nil {
		return fmt.Errorf("playtestprofiles: apply overlays requires a character")
	}
	if startRoom <= 0 {
		return fmt.Errorf("playtestprofiles: start_room must be > 0")
	}
	if world.RoomExists != nil && !world.RoomExists(startRoom) {
		return fmt.Errorf("playtestprofiles: start_room %d does not exist", startRoom)
	}
	u.Character.RoomId = startRoom

	if u.Character.SpellBook == nil {
		u.Character.SpellBook = map[string]int{}
	}
	if u.Character.Skills == nil {
		u.Character.Skills = map[string]int{}
	}

	for spellID, rank := range o.GrantSpells {
		if rank < 1 {
			return fmt.Errorf("playtestprofiles: grant_spells %q rank must be >= 1", spellID)
		}
		if world.SpellOK != nil && !world.SpellOK(spellID) {
			return fmt.Errorf("playtestprofiles: unknown spell %q", spellID)
		}
		u.Character.SpellBook[spellID] = rank
	}
	for skill, rank := range o.GrantSkills {
		if rank < 1 {
			return fmt.Errorf("playtestprofiles: grant_skills %q rank must be >= 1", skill)
		}
		u.Character.Skills[skill] = rank
	}
	for _, itemID := range o.GrantItems {
		if world.ItemOK != nil && !world.ItemOK(itemID) {
			return fmt.Errorf("playtestprofiles: unknown item %d", itemID)
		}
		it := items.New(itemID)
		if !u.Character.StoreItem(it) {
			return fmt.Errorf("playtestprofiles: could not store item %d", itemID)
		}
	}
	for slot, itemID := range o.Equip {
		if world.ItemOK != nil && !world.ItemOK(itemID) {
			return fmt.Errorf("playtestprofiles: unknown equip item %d", itemID)
		}
		if err := equipSlot(u, slot, itemID); err != nil {
			return err
		}
	}
	for _, token := range o.SetQuestTokens {
		if !u.Character.GiveQuestToken(token) {
			if !u.Character.HasQuest(token) && !u.Character.IsQuestDone(token) {
				return fmt.Errorf("playtestprofiles: could not grant quest token %q", token)
			}
		}
	}
	for k, v := range o.SetQuestFlags {
		if world.FlagOK != nil {
			if err := world.FlagOK(k, v); err != nil {
				return fmt.Errorf("playtestprofiles: set_quest_flags: %w", err)
			}
		}
		u.Character.SetQuestFlag(k, v)
	}
	if o.SetGold != nil {
		if *o.SetGold < 0 {
			return fmt.Errorf("playtestprofiles: set_gold must be >= 0")
		}
		u.Character.Gold = *o.SetGold
	}
	return nil
}

func equipSlot(u *users.UserRecord, slot string, itemID int) error {
	key := strings.ToLower(strings.TrimSpace(slot))
	it := items.New(itemID)
	for _, s := range u.Character.Equipment.AllSlots() {
		if s.Key == key {
			*s.Item = it
			return nil
		}
	}
	return fmt.Errorf("playtestprofiles: unknown equip slot %q", slot)
}
