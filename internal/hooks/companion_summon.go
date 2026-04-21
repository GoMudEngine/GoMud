package hooks

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// resolveCompanionSummon handles YAML-driven companion summoning spells.
// It replaces 13 nearly-identical JS onMagic scripts with one parameterized
// Go function. Called when spellData.SummonMobId > 0.
//
// Steps: companion cap check, optional component consumption, optional corpse
// consumption, stat scaling, mob spawn, charm, companion registration.
//
// Returns true on success, false on failure (error already sent to user).
func resolveCompanionSummon(user *users.UserRecord, spellData *spells.SpellData, spellRest string, room *rooms.Room) bool {

	// Handle corpse targeting: "corpse", "2.corpse", "corpse#2", etc.
	// Use the engine's standard disambiguation parser.
	targetWord, targetIndex := util.GetMatchNumber(spellRest)
	genericTargets := map[string]bool{"corpse": true, "remains": true, "body": true, "corpses": true, "bones": true, "": true}
	if genericTargets[targetWord] {
		spellRest = "" // Generic word — match by index, not name
	}
	// targetIndex: 1 = first valid, 2 = second valid, etc.

	ch := user.Character

	// ── 1. Companion cap ────────────────────────────────────────────────
	if len(ch.Companions) >= ch.GetMaxCompanions() {
		user.SendText("You cannot maintain any more companions.")
		return false
	}

	// ── 2. Component consumption (if required) ──────────────────────────
	if spellData.SummonComponentId > 0 {
		found := false
		for i, itm := range ch.Items {
			if itm.ItemId == spellData.SummonComponentId {
				ch.Items = append(ch.Items[:i], ch.Items[i+1:]...)
				found = true
				break
			}
		}
		if !found {
			user.SendText("You lack the required component for this summoning.")
			return false
		}
	}

	// ── 3. Corpse consumption (if required) ─────────────────────────────
	var corpsePool int
	corpseConsumed := false

	if spellData.SummonRequiresCorpse {
		targetIdx := -1
		validCount := 0
		for idx, c := range room.Corpses {
			if c.Prunable {
				continue
			}
			// Skip player corpses
			if c.UserId != 0 {
				continue
			}
			// Skip former companions
			if c.WasCharmed {
				continue
			}
			// Name filter if spellRest is a specific mob name
			if spellRest != "" && !strings.Contains(strings.ToLower(c.Character.Name), strings.ToLower(spellRest)) {
				continue
			}
			// Check minimum corpse stat pool
			pool := c.Character.Stats.Strength.Training +
				c.Character.Stats.Dexterity.Training +
				c.Character.Stats.Perception.Training +
				c.Character.Stats.Vitality.Training +
				c.Character.Stats.Willpower.Training +
				c.Character.Stats.Charisma.Training

			if spellData.SummonMinCorpsePool > 0 && pool < spellData.SummonMinCorpsePool {
				continue
			}

			validCount++
			if validCount == targetIndex {
				targetIdx = idx
				corpsePool = pool
				break
			}
		}

		if targetIdx < 0 {
			if spellRest != "" {
				user.SendText("You cannot find suitable remains matching that description.")
			} else {
				user.SendText("There are no suitable remains here to raise.")
			}
			return false
		}

		// Remove the corpse
		room.Corpses = append(room.Corpses[:targetIdx], room.Corpses[targetIdx+1:]...)
		corpseConsumed = true
	}

	// ── 4. Stat scaling ─────────────────────────────────────────────────
	charisma := ch.Stats.Charisma.ValueAdj
	manifestSkill := ch.GetSkillLevel(skills.Manifestation)

	pool := characters.CalcCompanionStatPool(spellData.SummonBasePool, charisma, manifestSkill)

	// If a corpse was consumed, average the scaled pool with the corpse pool
	if corpseConsumed {
		pool = (pool + corpsePool) / 2
	}

	// ── 5. Spawn and register ───────────────────────────────────────────
	mob := mobs.NewMobByIdFresh(mobs.MobId(spellData.SummonMobId), room.RoomId, pool)
	if mob == nil {
		user.SendText("The summoning fails — something is wrong.")
		return false
	}
	room.AddMob(mob.InstanceId)

	// Charm permanently (effectively infinite duration)
	mob.Character.Charm(user.UserId, 99999, "")
	mob.Character.EndAggro()
	user.Character.TrackCharmed(mob.InstanceId, true)

	// Determine source type based on whether a corpse was consumed
	sourceType := characters.CompanionSummoned
	if spellData.SummonRequiresCorpse {
		sourceType = characters.CompanionRaised
	}

	// Register as companion
	info := characters.CompanionInfo{
		MobId:      int(mob.MobId),
		InstanceId: mob.InstanceId,
		SourceType: sourceType,
		Name:       mob.Character.Name,
		BaseName:   mob.Character.Name,
		AutoAssist: true,
	}
	if !ch.AddCompanion(info) {
		// Should not happen since we checked cap above, but be safe
		user.SendText("You cannot maintain any more companions.")
		return false
	}

	// Clear aggro from existing companions toward the new mob
	for _, charmId := range ch.GetCharmIds() {
		if charmId == mob.InstanceId {
			continue
		}
		if companion := mobs.GetInstance(charmId); companion != nil {
			if companion.Character.Aggro != nil &&
				companion.Character.Aggro.MobInstanceId == mob.InstanceId {
				companion.Character.EndAggro()
			}
		}
	}

	// Clear the owner's own aggro if targeting the new mob
	if ch.Aggro != nil && ch.Aggro.MobInstanceId == mob.InstanceId {
		ch.EndAggro()
	}

	return true
}

