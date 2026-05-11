package itemvalue

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/species"
)

// CanEquipFromGive returns true if this character is the kind of
// creature that can rationally equip gear from a player give.
// Returns false for:
//   - Animal species (Species.DisabledSlots contains "Weapon")
//   - Non-combat archetypes (noncombat_*, prey, combat_passive)
//
// Note: incorporeal-rank-4 mobs are NOT explicitly skipped here.
// Their IsUpgrade naturally returns false because all gear
// scores 0 via mutations.GearEffectivenessMultiplier (chunk 2.2a).
//
// Args: character - the character to check
//       behaviorArchetype - the mob's behavior archetype (passed separately
//                           since character has no such field)
func CanEquipFromGive(character *characters.Character, behaviorArchetype string) bool {
	if character == nil {
		return false
	}

	// Animal-species gate: species with Weapon slot disabled.
	if speciesInfo := species.GetSpecies(character.SpeciesId); speciesInfo != nil {
		for _, slot := range speciesInfo.DisabledSlots {
			if slot == "Weapon" {
				return false
			}
		}
	}

	// Non-combat archetypes silently skip equip-if-better.
	switch behaviorArchetype {
	case "noncombat_passive", "noncombat_questgiver",
		"noncombat_shopkeeper", "prey", "combat_passive":
		return false
	}

	return true
}

// CanScanFloorLoot returns true if this character should scan floor
// loot for upgrades on idle ticks. Equals CanEquipFromGive plus
// !character.IsCharmed(). Companions and mercs accept pushes
// from their owner but skip pull (owner has dibs on floor loot).
func CanScanFloorLoot(character *characters.Character, behaviorArchetype string) bool {
	if !CanEquipFromGive(character, behaviorArchetype) {
		return false
	}
	return !character.IsCharmed()
}
