package hooks

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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

	// ── 1. Reservation + budget gate ────────────────────────────────────
	// The Conviction budget is the real limit (a soft count backstop lives in
	// CanAffordCompanion). Fail early, before consuming any component/corpse.
	baseReserve := spellData.SummonConvictionReserve
	if baseReserve <= 0 {
		baseReserve = int(configs.GetBalanceConfig().CompanionReserveDefault)
	}
	// Count cap and conviction budget both fail CanAffordCompanion — report them
	// separately so a player at their companion limit isn't wrongly told they
	// lack conviction.
	if len(ch.Companions) >= ch.GetMaxCompanions() {
		user.SendText(messaging.CategorySpellManifestation,
			"You are already sustaining as many companions as your will can hold.")
		return false
	}
	reserve := ch.CalcCompanionReserve(baseReserve)
	if !ch.CanAffordCompanion(reserve) {
		user.SendText(messaging.CategorySpellManifestation,
			"You cannot spare the conviction to bind another companion.")
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
			user.SendText(messaging.CategorySpellManifestation, "You lack the required component for this summoning.")
			return false
		}
	}

	// ── 3. Corpse consumption (if required) ─────────────────────────────
	var corpsePool int
	corpseConsumed := false

	if spellData.SummonRequiresCorpse {
		targetIdx, pool, reason := selectRaiseCorpse(room.Corpses, spellRest, targetIndex, int(spellData.SummonMinCorpsePool))
		if targetIdx < 0 {
			// Specific, actionable message (e.g. "still holds loot") rather than
			// a blanket "no suitable remains".
			user.SendText(messaging.CategorySpellManifestation, raiseFailureMessage(reason, spellRest))
			return false
		}
		corpsePool = pool

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
		user.SendText(messaging.CategorySpellDisruption, "The summoning fails — something is wrong.")
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
		MobId:             int(mob.MobId),
		InstanceId:        mob.InstanceId,
		SourceType:        sourceType,
		Name:              mob.Character.Name,
		BaseName:          mob.Character.Name,
		AutoAssist:        true,
		ConvictionReserve: reserve,
	}
	if !ch.AddCompanion(info) {
		// Should not happen since we checked the budget above, but be safe
		user.SendText(messaging.CategorySystem, "You cannot maintain any more companions.")
		return false
	}
	// Apply the new reservation immediately so usable Conviction drops now.
	ch.RecalculateStats()

	// Clear aggro from existing companions toward the new mob
	for _, charmId := range ch.GetCharmIds() {
		if charmId == mob.InstanceId {
			continue
		}
		if companion := mobs.GetInstance(charmId); companion != nil {
			if companion.Character.IsInCombat() &&
				companion.Character.CurrentCombatTarget().MobInstanceId == mob.InstanceId {
				companion.Character.EndAggro()
			}
		}
	}

	// Clear the owner's own aggro if targeting the new mob
	if ch.IsInCombat() && ch.CurrentCombatTarget().MobInstanceId == mob.InstanceId {
		ch.EndAggro()
	}

	return true
}

// raiseSkipReason explains why no corpse could be raised, so the failure
// message can be specific and actionable instead of a blanket "no remains".
type raiseSkipReason int

const (
	raiseNoMatch    raiseSkipReason = iota // no name/type-matching corpse present
	raiseWasCharmed                        // matched, but it was already a companion
	raiseTooWeak                           // matched, but its essence is too faint
	raiseHasLoot                           // matched, but it still holds loot
)

// raisePriority ranks reasons by how actionable they are; the most actionable
// (HasLoot — just loot it and retry) wins when several near-matches apply.
func raisePriority(r raiseSkipReason) int {
	switch r {
	case raiseHasLoot:
		return 3
	case raiseTooWeak:
		return 2
	case raiseWasCharmed:
		return 1
	default:
		return 0
	}
}

func worseReason(a, b raiseSkipReason) raiseSkipReason {
	if raisePriority(b) > raisePriority(a) {
		return b
	}
	return a
}

// corpseRaisePool is the stat metric the raise spell scores a corpse by — the
// same Training sum `assess` reports, so the two agree on a corpse's worth.
func corpseRaisePool(c rooms.Corpse) int {
	s := c.Character.Stats
	return s.Strength.Training + s.Dexterity.Training + s.Perception.Training +
		s.Vitality.Training + s.Willpower.Training + s.Charisma.Training
}

// selectRaiseCorpse picks the targetIndex-th valid mob corpse to raise (1-based
// among valid candidates). Returns (-1, 0, reason) when none qualifies, where
// reason describes why the best name/type-matching corpse was rejected — so the
// caller can show a helpful message. minPool <= 0 disables the stat-pool gate.
func selectRaiseCorpse(corpses []rooms.Corpse, nameFilter string, targetIndex, minPool int) (idx int, pool int, reason raiseSkipReason) {
	reason = raiseNoMatch
	validCount := 0
	for i, c := range corpses {
		// Not a raisable mob corpse at all — never counts as a near-match.
		if c.Prunable || c.UserId != 0 {
			continue
		}
		// Name filter: a non-matching corpse isn't what the player asked for.
		if nameFilter != "" && !strings.Contains(strings.ToLower(c.Character.Name), strings.ToLower(nameFilter)) {
			continue
		}
		// From here the corpse matched name/type; rejections carry a reason.
		if c.WasCharmed {
			reason = worseReason(reason, raiseWasCharmed)
			continue
		}
		if c.HasLoot() {
			reason = worseReason(reason, raiseHasLoot)
			continue
		}
		p := corpseRaisePool(c)
		if minPool > 0 && p < minPool {
			reason = worseReason(reason, raiseTooWeak)
			continue
		}
		validCount++
		if validCount == targetIndex {
			return i, p, raiseNoMatch
		}
	}
	return -1, 0, reason
}

// raiseFailureMessage renders the player-facing reason a raise found no corpse.
func raiseFailureMessage(reason raiseSkipReason, nameFilter string) string {
	switch reason {
	case raiseHasLoot:
		return "Those remains still hold spoils — clear the loot from them first, then raise what's left."
	case raiseTooWeak:
		return "Those remains are too faint to hold a bound spirit."
	case raiseWasCharmed:
		return "Those remains were already animated once; there is nothing left to raise."
	default:
		if nameFilter != "" {
			return "You cannot find suitable remains matching that description."
		}
		return "There are no suitable remains here to raise."
	}
}
