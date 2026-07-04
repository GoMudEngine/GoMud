package hooks

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// File: NewRound_DoCombat_unified.go
//
// Stage 2a of combat-quadrant unification (see spec
// docs/superpowers/specs/2026-04-18-combat-quadrant-unification-design.md).
//
// This file introduces handleCombatRound (the unified replacement for the
// four handle{P,M}vs{P,M} functions) and its eight phase helpers. As of
// this commit the new code is PARALLEL and UNUSED — dispatchers in
// NewRound_DoCombat.go still call the old quadrant handlers. The flip
// happens in Stage 2b (Task 3).
//
// Routing strategy: atk.IsPlayer() / def.IsPlayer() checks at the leaf
// of each phase where divergence is required. No Quadrant enum.
//
// Each intentional divergence has a "// Divergence #N:" comment that
// references the numbered list in the design spec.

// handleCombatRound is the unified combat-round handler that will replace
// the four quadrant handlers in Stage 2b. Behavior is intended to be
// IDENTICAL to the old quadrant handlers at parity (see Stage 1 fixes in
// NewRound_DoCombat_parity_test.go). Any new combat logic added here
// applies to all four quadrants by default; quadrant-specific logic must
// be explicitly gated on atk.IsPlayer() / def.IsPlayer() AND commented
// with the reason for divergence.
//
// At end of Stage 2a this function is unreachable; tests should not
// observe any behavior change from its presence.
func handleCombatRound(
	atk actions.Actor,
	def actions.Actor,
	evt events.NewRound,
	moonMod float64,
	cfg *configs.Config,
	affectedPlayerIds *[]int,
	affectedMobInstanceIds *[]int,
	forceCrit bool,
) {
	// Phase 0: defender lookup + alive/in-room validation.
	if !resolveCombatTarget(atk, def, evt.RoundNumber) {
		return
	}

	// Defender's combat-cancel buffs always strip on combat engagement.
	// CancelCombatBuffs also strips permabuff entries so Validate() won't
	// re-apply them (notably: Hidden seeded via buffids on ambushers).
	def.GetCharacter().CancelCombatBuffs()

	// Chunk 1 follow-up (surfaced by chunk 4b smoke 2026-05-16):
	// CancelCombatBuffs strips buff #9 but the Awareness FSM is the
	// canonical source of truth for IsHidden post-chunk-1. The cascade
	// in Awareness_Cascades.go fires only when the defender's OWN
	// CombatPhase transitions Idle→Engaging — which doesn't happen
	// when a defender is targeted but never SetAggro's the attacker
	// (e.g. a hidden mob grappled while it has no aggro of its own).
	// Force the FSM out of Hidden if the buff strip left it stale; the
	// cascade re-strips the buff (no-op since already gone). Without
	// this, hidden ambushers stay stuck Hidden and the IsHidden check
	// below fires "can't seem to find your target" on every round.
	if defChar := def.GetCharacter(); defChar.Awareness != nil && defChar.IsHidden() {
		defChar.Awareness.ForceVisible(state.TransitionReason{
			Trigger: awareness.TriggerCombatEntered,
		})
	}

	// Phase 1: wait-round short-circuit.
	if phase1WaitRound(atk, def) {
		return
	}

	// Hidden defender bails the round (existing per-quadrant behavior).
	if def.GetCharacter().IsHidden() {
		// Divergence: only player attackers get the "can't seem to find
		// your target" feedback. Mob attackers silently bail (no chat).
		if atk.IsPlayer() {
			atk.SendText(messaging.CategorySystem, "You can't seem to find your target.")
		}
		return
	}

	// Append affected-id slots that the OLD per-quadrant handlers append
	// PRE-attack. Faithful to the per-quadrant ordering (PvP and PvM both
	// append the player attacker's Aggro target user id pre-attack).
	appendPreAttackAffected(atk, def, affectedPlayerIds, affectedMobInstanceIds)

	// Grapple progression is now handled by the per-round control-axis
	// tick in Position_GrappleTick.go (chunk 4b T6). The legacy single-
	// roll processGrappleProgression scanner was deleted in T10.

	// MvP-only: target switch AI runs after the grapple-progression hook,
	// before the swing.
	if !atk.IsPlayer() && def.IsPlayer() {
		// Divergence (MvP-only): the legacy mob handler considers
		// switching to a different player target before swinging.
		if mob, ok := atk.(*actions.MobActor); ok && mob.Mob != nil {
			if handleMobTargetSwitch(mob.Mob, atk.GetRoom()) {
				return
			}
			handleMobWeaponPickup(mob.Mob)
		}
	}

	// Phase 2: roll the attack (with moon-mod wrap).
	// Chunk 3.3: forceCrit is true when the defender was Sleeping at round
	// start; passed through to calculateCombat via rollCombatAttack so every
	// swing against the snapshotted target crits regardless of z-score.
	res := rollCombatAttack(atk, def, moonMod, forceCrit)

	// Phase 3: damage-layer bonuses (Conviction / Adrenaline / Return / Lifesteal).
	applyCombatDamageBonuses(atk, def, &res)

	// Pinnacle item procs: attacker's weapon on_hit. Fires for all four
	// quadrants (player and mob attackers) — the point of hooking the unified
	// orchestrator. Gated internally by ItemProcsEnabled + the per-proc
	// chance/cooldown; a no-op when the attacker carries no proc weapon.
	if res.Hit {
		dispatchItemProcs("on_hit", atk.GetCharacter(), def.GetCharacter(), atk.GetRoom(), res.DamageToTarget)
	}

	// Combat analytics (shared across all four quadrants).
	recordCombatAnalytics(atk, def, res, evt.RoundNumber)

	// Phase 4: crit effects + messaging dispatch.
	dispatchCritAndMessaging(atk, def, &res)

	// Phase 5: progression (attacker stats/skills/crits, defender D/P/B).
	applyCombatProgression(atk, def, &res)

	// Phase 6: defender behavior trigger (mob_hurt — no-op for player def).
	fireDefenderBehaviorTrigger(atk, def, res)

	// Phase 7: aggro propagation + companion/party assist.
	handleAggroAndAssist(atk, def, cfg)

	// Phase 8: end-of-round death/retarget resolution.
	resolveCombatRound(atk, def)
}

// ────────────────────────────────────────────────────────────────────────
// Phase 0: target resolution
// ────────────────────────────────────────────────────────────────────────

// resolveCombatTarget sanity-checks the (already-resolved) defender for
// liveness and room membership. Returns true if combat should proceed.
//
// Merges the first ~15 lines of each of handlePlayerVsPlayer /
// handlePlayerVsMob / handleMobVsPlayer / handleMobVsMob.
//
// roundNumber is used to seed the defender mob's CombatMemory when we
// emit a first-round combat_start signal (PvM-only — see body).
func resolveCombatTarget(atk, def actions.Actor, roundNumber uint64) bool {
	atkChar := atk.GetCharacter()
	if atkChar == nil {
		return false
	}
	if atk.GetRoom() == nil {
		atkChar.EndAggro()
		return false
	}
	if def == nil || def.GetCharacter() == nil {
		// Divergence: only player attackers get the "can't be found" message
		// (mobs have no connection).
		if atk.IsPlayer() {
			atk.SendText(messaging.CategorySystem, `Your target can't be found.`)
		}
		atkChar.EndAggro()
		return false
	}
	defChar := def.GetCharacter()

	// Defender room mismatch.
	if atkChar.RoomId != defChar.RoomId {
		// No cross-room pursuit. The old continuous remote-shoot path set
		// Aggro.ExitName so the round loop could keep swinging at a target
		// in an adjacent room; it has been retired in favor of the
		// immediate loaded-weapon `fire` action, which holds no aggro.
		// (Nothing writes Aggro.ExitName anymore.) Players get a notice;
		// mobs have no connection so stay silent.
		if atk.IsPlayer() {
			atk.SendText(messaging.CategorySystem, `Your target can't be found.`)
		}
		atkChar.EndAggro()
		return false
	}

	// PvM-only: initialize CombatMemory for the defender mob if this is
	// its first round in combat so grudge tracking works across flee/re-engage.
	if atk.IsPlayer() && !def.IsPlayer() {
		if defMob := asMobOrNil(def); defMob != nil {
			if defMob.CombatMemory == nil && defChar.Aggro != nil {
				defMob.CombatMemory = mobs.SetCombatMemory(
					defChar.Aggro.UserId,
					defChar.Aggro.MobInstanceId,
					defChar.RoomId,
					roundNumber,
				)
			}
		}
	}

	// Defender liveness.
	// Divergence: when the defender is a player, the legacy MvP handler
	// uses IsDisabled() (which post-bleedout-removal is equivalent to
	// Health <= 0). All other quadrants check Health < 1 directly. Both
	// reduce to the same condition today, so we use the simpler form.
	if defChar.Health < 1 {
		atkChar.EndAggro()
		return false
	}
	return true
}

// ────────────────────────────────────────────────────────────────────────
// Phase 1: wait-round adapter
// ────────────────────────────────────────────────────────────────────────

// phase1WaitRound delegates to the pre-existing handleCombatWaitRound.
// Returns true if the attacker was waiting and the round was consumed.
func phase1WaitRound(atk, def actions.Actor) bool {
	atkChar := atk.GetCharacter()
	if atkChar.Aggro == nil || atkChar.Aggro.RoundsWaiting <= 0 {
		return false
	}
	defChar := def.GetCharacter()
	atkRoom := atk.GetRoom()
	defRoom := def.GetRoom()

	// viewerUserId is the user-side participant (0 if neither is a user).
	viewerUserId := 0
	if atk.IsPlayer() {
		viewerUserId = atk.GetUserId()
	} else if def.IsPlayer() {
		viewerUserId = def.GetUserId()
	}

	return handleCombatWaitRound(
		atkChar, defChar,
		actorSourceTarget(atk), actorSourceTarget(def),
		userRecordOrNil(atk), userRecordOrNil(def),
		atkRoom, defRoom,
		viewerUserId,
	)
}

// ────────────────────────────────────────────────────────────────────────
// Phase 2: attack roll
// ────────────────────────────────────────────────────────────────────────

// rollCombatAttack runs the moon-mod-wrapped attack roll and returns the
// AttackResult. Polymorphic over combat.Attack{P,M}vs{P,M}.
// forceCrit is true when the defender was snapshotted as Sleeping at
// round start (chunk 3.3); all swings this round against them crit.
func rollCombatAttack(atk, def actions.Actor, moonMod float64, forceCrit bool) combat.AttackResult {
	restore := applyMoonMods(atk.GetCharacter(), moonMod)
	defer restore()

	switch {
	case atk.IsPlayer() && def.IsPlayer():
		return combat.AttackPlayerVsPlayer(asUser(atk), asUser(def), forceCrit)
	case atk.IsPlayer() && !def.IsPlayer():
		return combat.AttackPlayerVsMob(asUser(atk), asMob(def), forceCrit)
	case !atk.IsPlayer() && def.IsPlayer():
		return combat.AttackMobVsPlayer(asMob(atk), asUser(def), forceCrit)
	default:
		return combat.AttackMobVsMob(asMob(atk), asMob(def), forceCrit)
	}
}

// ────────────────────────────────────────────────────────────────────────
// Phase 3: damage-layer bonuses
// ────────────────────────────────────────────────────────────────────────

// applyCombatDamageBonuses applies all damage-layer buffs/debuffs that
// fire after the attack roll: Conviction Surge (DamageBonus flag),
// Adrenaline Surge (mutation), return damage (species + equipment),
// and lifesteal.
//
// Merges the 2 existing helpers (applyCombatDamageBonuses_PvM /
// applyCombatDamageBonuses_MvP in NewRound_DoCombat_resolution.go) and
// the 2 inline copies (PvP at helpers.go:993-1001 and MvM at
// helpers.go:1244-1300). Note: when this function replaces the PvP
// handler in Stage 2b, PvP gains Adrenaline / return / lifesteal — these
// were absent from the legacy PvP path. Per design spec the unified
// handler applies the same sequence to all four quadrants by default.
func applyCombatDamageBonuses(atk, def actions.Actor, res *combat.AttackResult) {
	if !res.Hit || res.DamageToTarget <= 0 {
		return
	}
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()

	// Conviction Surge: +15% damage on hit when DamageBonus buff flag set.
	if atkChar.HasBuffFlag(buffs.DamageBonus) {
		bonusDmg := int(math.Round(float64(res.DamageToTarget) * 0.15))
		if bonusDmg < 1 {
			bonusDmg = 1
		}
		defChar.Health -= bonusDmg
		res.DamageToTarget += bonusDmg
	}

	// Adrenaline Surge (mutation).
	if mutations.IsAdrenalSurgeActive(atkChar.Mutations, atkChar.Health, atkChar.HealthMax.Value) {
		if surgeBonus := mutations.GetAdrenalSurgeBonus(atkChar.Mutations); surgeBonus > 0 {
			bonusDmg := int(math.Round(float64(res.DamageToTarget) * surgeBonus))
			if bonusDmg < 1 {
				bonusDmg = 1
			}
			defChar.Health -= bonusDmg
			res.DamageToTarget += bonusDmg
		}
	}

	// Return damage (species + equipment).
	returnPct := defChar.StatMod("return_damage")
	if sp := species.GetSpecies(defChar.SpeciesId); sp != nil {
		returnPct += sp.ReturnDamage
	}
	if returnPct > 0 {
		returnDmg := int(float64(res.DamageToTarget) * float64(returnPct) / 100.0)
		if returnDmg > 0 {
			atkChar.Health -= returnDmg
			emitReturnDamageText(atk, def, returnDmg)
		}
	}

	// Lifesteal.
	if lifestealPct := atkChar.StatMod("lifesteal_pct"); lifestealPct > 0 {
		healAmt := int(float64(res.DamageToTarget) * float64(lifestealPct) / 100.0)
		if healAmt > 0 {
			atkChar.Health += healAmt
			if atkChar.Health > atkChar.HealthMax.Value {
				atkChar.Health = atkChar.HealthMax.Value
			}
			// Divergence: only player attackers get a private "your weapon
			// feeds" message — mobs have no connection.
			if atk.IsPlayer() {
				healDesc := combat.GetHealDescription(healAmt, atkChar.HealthMax.Value)
				atk.SendText(messaging.CategoryHitMelee, fmt.Sprintf(
					`<ansi fg="green">Your weapon feeds on the blow! (%s)</ansi>`,
					healDesc))
			}
		}
	}
}

// emitReturnDamageText broadcasts the "X recoils from striking Y" room
// message for return damage. Handles all four quadrants by picking
// username vs mobname text based on each side's IsPlayer() and emits a
// private "you recoil" message to a player attacker.
func emitReturnDamageText(atk, def actions.Actor, returnDmg int) {
	atkRoom := atk.GetRoom()
	if atkRoom == nil {
		return
	}
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	dmgDesc := combat.GetDamageDescription(returnDmg, atkChar.HealthMax.Value)

	// Compose attacker / defender display tokens for the room broadcast.
	var atkToken string
	if atk.IsPlayer() {
		atkToken = fmt.Sprintf(`<ansi fg="username">%s</ansi>`, atkChar.Name)
	} else {
		if mob := asMobOrNil(atk); mob != nil {
			atkToken = mobDisplayName(mob, atkRoom, def.GetUserId())
		} else {
			atkToken = fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, atkChar.Name)
		}
	}
	var defToken string
	if def.IsPlayer() {
		defToken = fmt.Sprintf(`<ansi fg="username">%s</ansi>`, defChar.Name)
	} else {
		if mob := asMobOrNil(def); mob != nil {
			defToken = mobDisplayName(mob, def.GetRoom(), atk.GetUserId())
		} else {
			defToken = fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, defChar.Name)
		}
	}

	excludes := playerExcludeIds(atk, def)
	sendVisualRoomText(atkRoom, messaging.CategoryHitMelee, fmt.Sprintf(
		`<ansi fg="red">%s recoils from striking %s! (%s)</ansi>`,
		atkToken, defToken, dmgDesc), excludes...)

	// Player attacker: send the private "you recoil" message.
	if atk.IsPlayer() {
		atk.SendText(messaging.CategoryHitMelee, fmt.Sprintf(
			`<ansi fg="red">You recoil from striking %s! (%s)</ansi>`,
			defToken, dmgDesc))
	}
	// Player defender: send the "X recoils from striking you" message.
	if def.IsPlayer() {
		def.SendText(messaging.CategoryHitMelee, fmt.Sprintf(
			`<ansi fg="red">%s recoils from striking you! (%s)</ansi>`,
			atkToken, dmgDesc))
	}
}

// recordCombatAnalytics wraps combat.RecordAttack once for all four
// quadrants.
func recordCombatAnalytics(atk, def actions.Actor, res combat.AttackResult, roundNumber uint64) {
	atkType := "unarmed"
	if atk.GetCharacter().Equipment.Weapon.ItemId > 0 {
		atkType = "weapon"
	}
	combat.RecordAttack(res, actorSourceTarget(atk), actorSourceTarget(def), atkType,
		atk.GetCharacter(), def.GetCharacter(), roundNumber)
}

// ────────────────────────────────────────────────────────────────────────
// Phase 4: crit effects + messaging
// ────────────────────────────────────────────────────────────────────────

// dispatchCritAndMessaging applies crit effects and routes combat
// messages to the correct recipients per quadrant.
//
// Intentional divergences (spec §"Intentional Divergences"):
//   - #1 Crit message routing:
//   - Attacker text (MessagesToSource): only if atk.IsPlayer()
//   - Defender text (MessagesToTarget): only if def.IsPlayer()
//   - Room broadcasts: always, with text-receiving combatants excluded
//   - Darkness replacement: only matters for player viewers.
func dispatchCritAndMessaging(atk, def actions.Actor, res *combat.AttackResult) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkRoom := atk.GetRoom()
	defRoom := def.GetRoom()

	// Darkness replacement — applies for whichever side is a player viewer.
	srcCanSee := true
	tgtCanSee := true
	if atk.IsPlayer() {
		srcCanSee = messaging.CanSeeClearly(atkChar, atkRoom)
	}
	if def.IsPlayer() {
		tgtCanSee = messaging.CanSeeClearly(defChar, defRoom)
	}
	if !srcCanSee || !tgtCanSee {
		replaceDarknessMessages(res, srcCanSee, tgtCanSee)
	}

	// Crit effects (riposte / sweep / bash) compute side-specific text.
	critResult := applyCritEffects(atkChar, defChar, *res, atkRoom)

	// Crit message routing — Divergence #1.
	if critResult.AttackerMsg != `` && atk.IsPlayer() {
		atk.SendText(messaging.CategoryHitMelee, critResult.AttackerMsg)
	}
	if critResult.DefenderMsg != `` && def.IsPlayer() {
		def.SendText(messaging.CategoryHitMelee, critResult.DefenderMsg)
	}
	if critResult.RoomMsg != `` && atkRoom != nil {
		// Crit effects (riposte / sweep / bash) are melee follow-ups.
		atkRoom.SendText(messaging.CategoryHitMelee, critResult.RoomMsg, playerExcludeIds(atk, def)...)
	}

	// Buffs from the round (BuffSource → atk, BuffTarget → def).
	for _, buffId := range res.BuffSource {
		atk.AddBuff(buffId, `combat`)
	}
	for _, buffId := range res.BuffTarget {
		def.AddBuff(buffId, `combat`)
	}

	// Direct messages — Divergence #1, now verbosity-gated (spec:
	// 2026-06-10-combat-verbosity-design.md). AttackResult.MessagesTo*
	// carry per-line TaggedMessage data (Category + Text); the gate
	// suppresses by category per the viewer's setting. Floor rules:
	// damage-to-viewer lines always pass; categories outside the
	// suppress tables always pass.
	if atk.IsPlayer() {
		u := asUser(atk)
		lvl := u.GetCombatVerbosity()
		drainParticipantLines(u, res.MessagesToSource, lvl, false)
		// Gate on srcCanSee: a blind attacker's prose was already
		// darkness-substituted; the tally summary must not re-introduce
		// the named information they couldn't read per-swing.
		if lvl == messaging.VerbosityLight && srcCanSee {
			recordTallyFor(u.UserId, atk, def, res)
		}
	}
	if def.IsPlayer() {
		u := asUser(def)
		lvl := u.GetCombatVerbosity()
		drainParticipantLines(u, res.MessagesToTarget, lvl, true)
		// Same rationale: a blind defender's incoming text was already
		// darkness-substituted; only record the tally when they can see.
		if lvl == messaging.VerbosityLight && tgtCanSee {
			recordTallyFor(u.UserId, atk, def, res)
		}
	}

	// Room broadcasts, per-spectator gated one step below their setting.
	excludes := playerExcludeIds(atk, def)
	drainSpectatorLines(atkRoom, res.MessagesToSourceRoom, excludes)
	drainSpectatorLines(defRoom, res.MessagesToTargetRoom, excludes)
	recordSpectatorTallies(atkRoom, defRoom, atk, def, res, excludes)
	sendDarkRoomCombatFallback(atkRoom, excludes...)
	if defRoom != atkRoom {
		sendDarkRoomCombatFallback(defRoom, excludes...)
	}
}

// ────────────────────────────────────────────────────────────────────────
// Phase 5: progression
// ────────────────────────────────────────────────────────────────────────

// applyCombatProgression runs attacker-side stat/skill/crit progression
// plus defender-side OnCritReceived and dodge/parry/block progression.
//
// Intentional divergences (spec §"Intentional Divergences"):
//   - #6 TrackPlayerDamage only when both are players.
//   - #8 Stat-gain room messages: MvP/MvM use mob-flavor text; PvM/PvP
//     use player-side OnStatUse (no room broadcast historically).
//   - The userId arg passed to OnSkillUse / OnStatUse / OnCritReceived
//     is actor.GetUserId() (0 for mobs).
//   - Player concentration break is checked only when defender is a player.
func applyCombatProgression(atk, def actions.Actor, res *combat.AttackResult) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkUid := atk.GetUserId()
	defUid := def.GetUserId()

	// Cancel sleeping / cancel-on-damage buffs for any defender that took
	// damage this round (chunk 3.3). Fires before concentration-break so
	// that a waking defender's buffs are cleaned up in the same phase.
	if res.DamageToTarget > 0 {
		cancelDamageBuffs(defChar)
	}

	// Defender player concentration break (Divergence: player defender only).
	if def.IsPlayer() {
		handlePlayerConcentrationBreak(asUser(def), *res, def.GetRoom())
	}

	// Offhand break (quadrant-flavored — user/mob variants).
	if def.IsPlayer() {
		handleOffhandBreakUserDef(*res, asUser(def), def.GetRoom())
	} else {
		handleOffhandBreakMobDef(*res, asMob(def))
	}

	// PvP-only damage tracking — Divergence #6.
	if res.Hit && atk.IsPlayer() && def.IsPlayer() {
		defChar.TrackPlayerDamage(atkUid, res.DamageToTarget)
	}

	// Defender OnCritReceived (parity across all four quadrants — Stage 1
	// closed the MvM and PvM gaps).
	if res.Hit && res.Crit {
		defChar.OnCritReceived("physical", defUid)
	}

	// Attacker stat progression with quadrant-flavored room messages.
	emitAttackerStatGain(atk, "strength", atkUid)
	emitAttackerStatGain(atk, "dexterity", atkUid)

	// Attacker per-weapon skill + crit callbacks.
	for _, wh := range res.WeaponHits {
		if wh.Hit {
			atkChar.OnSkillUse(wh.SkillTag, atkUid)
			if wh.Crit {
				atkChar.OnCriticalSuccess(wh.SkillTag, atkUid)
			}
		} else if wh.Fumble {
			atkChar.OnCriticalFailure(wh.SkillTag, atkUid)
		}
	}
	if len(res.WeaponHits) == 0 && res.Hit {
		atkChar.OnSkillUse(string(skills.UnarmedCombat), atkUid)
	}

	// Defender dodge/parry/block.
	processDefenderProgression(defChar, defUid, *res)
}

// emitAttackerStatGain calls OnStatUse on the attacker and, if the stat
// gained AND the attacker is a mob, emits the room-visible mob-flavor
// stat-gain text. Players historically get only their own private
// progression message via OnStatUse; no room broadcast is emitted for
// player attackers (matches legacy PvP/PvM behavior).
//
// Divergence #8: room broadcast only for mob attackers.
func emitAttackerStatGain(atk actions.Actor, statName string, uid int) {
	gained := atk.GetCharacter().OnStatUse(statName, uid)
	if !gained || atk.IsPlayer() {
		return
	}
	atkRoom := atk.GetRoom()
	if atkRoom == nil {
		return
	}
	mob := asMobOrNil(atk)
	if mob == nil {
		return
	}
	if tmpl, ok := characters.MobStatGainMessages[statName]; ok {
		// Mob-side stat-gain flavor text — visible mob emote.
		atkRoom.SendText(messaging.CategoryMobEmote, fmt.Sprintf(tmpl, mobDisplayName(mob, atkRoom, 0)))
	}
}

// ────────────────────────────────────────────────────────────────────────
// Phase 6: defender behavior trigger
// ────────────────────────────────────────────────────────────────────────

// fireDefenderBehaviorTrigger fires the mob_hurt behavior tree when the
// defender is a mob. No-op for player defenders — Divergence #2: players
// have no behavior tree.
func fireDefenderBehaviorTrigger(atk, def actions.Actor, res combat.AttackResult) {
	if def.IsPlayer() || !res.Hit {
		return
	}
	defMob := asMob(def)
	ctx := behaviortree.EventContext{
		EventType: "mob_hurt",
		RoomId:    defMob.Character.RoomId,
	}
	if atk.IsPlayer() {
		ctx.UserId = atk.GetUserId()
	} else {
		ctx.MobId = atk.GetMobInstanceId()
	}
	behaviortree.TryMobBehavior(defMob.InstanceId, ctx)
}

// ────────────────────────────────────────────────────────────────────────
// Phase 7: aggro propagation + companion / party assist
// ────────────────────────────────────────────────────────────────────────

// handleAggroAndAssist manages aggro propagation (defender attacks back)
// and companion/party auto-assist.
//
// Intentional divergences (spec §"Intentional Divergences"):
//   - #3 Hostility groups (mobs.MakeHostile) only for player→mob.
//   - #4 Mob-aggro-on-attack with exit-walk only when atk is player;
//     mob attackers set Aggro directly (already in same room).
//   - #5 Party auto-assist: gated on def.GetCharacter().PartyId, NOT on
//     def.IsPlayer(). Today only players can form parties so this
//     naturally no-ops for mob defenders, but the gate is structured
//     so a future mob-party feature routes through the same path.
func handleAggroAndAssist(atk, def actions.Actor, cfg *configs.Config) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()
	atkRoom := atk.GetRoom()

	// Defender retaliation aggro.
	switch {
	case atk.IsPlayer() && def.IsPlayer():
		// PvP: mutual aggro is established by the attack command itself
		// elsewhere; nothing to do here.

	case atk.IsPlayer() && !def.IsPlayer():
		// PvM:
		//   - Hostility group assignment (Divergence #3) — charisma-scaled.
		//   - Mob aggro with exit-walk if attacker is in a different room
		//     (Divergence #4).
		//   - Companion-owner assist if defender mob is charmed.
		defMob := asMob(def)
		// Pack-tactics revamp: same-room routine-matching packmates
		// receive packmate_hurt. Replaces the former mobs.MakeHostile
		// group-flag propagation (which flagged every taxonomic group
		// the defender belonged to, causing priests to aggro on
		// respawning players who had attacked bandits).
		dispatchPackmateHurt(defMob, atk.GetUserId(), 0)
		if defChar.Aggro == nil {
			if atkChar.RoomId != defChar.RoomId {
				if mobRoom := rooms.LoadRoom(defChar.RoomId); mobRoom != nil {
					for exitName, exitInfo := range mobRoom.Exits {
						if exitInfo.RoomId == atkChar.RoomId {
							defMob.Command(fmt.Sprintf(`go %s`, exitName))
							if actionStr := defMob.GetAngryCommand(); actionStr != `` {
								defMob.Command(actionStr)
							}
							break
						}
					}
				}
			}
			defMob.Command(fmt.Sprintf("attack @%d", atk.GetUserId()))
		}
		// Companion-owner assist: if the defender mob is a charmed
		// companion, its owner (and other companions) fight back.
		handleCompanionOwnerAssist(defMob, fmt.Sprintf("@%d", atk.GetUserId()))

	case !atk.IsPlayer() && def.IsPlayer():
		// MvP:
		//   - Reciprocal aggro on the player defender (skip if dead/disabled).
		//   - Party auto-assist (Divergence #5) gated on PartyId.
		//   - Charmed-mob assist for the defender player.
		if defChar.Health > 0 && defChar.Aggro == nil {
			defChar.SetAggro(0, atk.GetMobInstanceId(), characters.DefaultAttack)
		}
		// Divergence #5: party-assist gate keys on party-membership
		// (parties.Get(userId)) — the engine's equivalent of a PartyId
		// field. Today only players can join parties, so this naturally
		// no-ops for mob defenders. When mob parties become a real
		// mechanic, the same gate will route through this code path.
		if partyExistsFor(def) {
			handlePartyAutoAttack(asMob(atk), asUser(def))
		}
		if atkRoom != nil {
			handleCharmedMobAssist(atkRoom, def.GetUserId(), fmt.Sprintf("#%d", atk.GetMobInstanceId()))
		}

	default: // !atk.IsPlayer() && !def.IsPlayer() — MvM
		// MvM:
		//   - Defender mob gets aggro and issues its own attack command.
		//   - Companion-owner assist if defender mob is charmed.
		defMob := asMob(def)
		if defChar.Aggro == nil {
			defChar.Aggro = &characters.Aggro{
				Type: characters.DefaultAttack,
			}
			defMob.Command(fmt.Sprintf("attack #%d", atk.GetMobInstanceId()))
		}
		handleCompanionOwnerAssist(defMob, fmt.Sprintf("#%d", atk.GetMobInstanceId()))
	}

	// PvP: charmed-mob assist for the defender player happens in
	// the legacy handler BEFORE dispatchCombatMessages. We mirror it
	// here in the same phase but use def.GetUserId() / @atk.UserId.
	if atk.IsPlayer() && def.IsPlayer() && atkRoom != nil {
		handleCharmedMobAssist(atkRoom, def.GetUserId(), fmt.Sprintf("@%d", atk.GetUserId()))
	}
}

// partyExistsFor returns true if the actor has a registered party (used
// as a defensive secondary check alongside PartyId, since parties are
// stored in a side-table keyed by user id).
func partyExistsFor(a actions.Actor) bool {
	if !a.IsPlayer() {
		return false
	}
	return parties.Get(a.GetUserId()) != nil
}

// ────────────────────────────────────────────────────────────────────────
// Phase 8: end-of-round resolution
// ────────────────────────────────────────────────────────────────────────

// resolveCombatRound handles end-of-round death processing and retarget
// messaging.
//
// Intentional divergence:
//   - #9 Retarget-on-death "You turn your attention to..." only when the
//     survivor is a player.
func resolveCombatRound(atk, def actions.Actor) {
	atkChar := atk.GetCharacter()
	defChar := def.GetCharacter()

	if atkChar.Health <= 0 || defChar.Health <= 0 {
		defChar.EndAggro()
		if atkChar.Health > 0 {
			atkRoom := atk.GetRoom()
			if atk.IsPlayer() {
				if RetargetOrEnd(atkChar, atkRoom, atk.GetUserId(), 0) {
					// Divergence #9: player-only retarget message.
					emitRetargetMessage(atk)
				}
			} else {
				RetargetOrEnd(atkChar, atkRoom, 0, atk.GetMobInstanceId())
			}
		} else {
			atkChar.EndAggro()
		}
		return
	}

	// Both alive: re-assert attacker's aggro on the current defender.
	// PvP does this implicitly via SetAggro on the user; PvM/MvP/MvM
	// explicitly re-set aggro at end of round.
	if def.IsPlayer() {
		atkChar.SetAggro(def.GetUserId(), 0, characters.DefaultAttack)
	} else {
		atkChar.SetAggro(0, def.GetMobInstanceId(), characters.DefaultAttack)
	}
}

// emitRetargetMessage sends the "You turn your attention to..." message
// to a player attacker who was just retargeted by RetargetOrEnd.
func emitRetargetMessage(atk actions.Actor) {
	if !atk.IsPlayer() {
		return
	}
	atkChar := atk.GetCharacter()
	if atkChar.Aggro == nil {
		return
	}
	if mob := mobs.GetInstance(atkChar.Aggro.MobInstanceId); mob != nil {
		atk.SendText(messaging.CategorySystem, fmt.Sprintf(
			"You turn your attention to <ansi fg=\"mobname\">%s</ansi>!",
			mob.Character.Name))
	} else if newDef := users.GetByUserId(atkChar.Aggro.UserId); newDef != nil {
		atk.SendText(messaging.CategorySystem, fmt.Sprintf(
			"You turn your attention to <ansi fg=\"username\">%s</ansi>!",
			newDef.Character.Name))
	}
}

// ────────────────────────────────────────────────────────────────────────
// Affected-id bookkeeping
// ────────────────────────────────────────────────────────────────────────

// appendPreAttackAffected mirrors the per-quadrant pre-attack append
// pattern in the legacy handlers. Each quadrant pushes specific ids onto
// the affected lists at distinct points in the round; we faithfully
// reproduce the legacy ordering so that the post-round suicide sweep
// (handleAffected) sees the same set of ids.
//
// Note: handleAggroAndAssist may further mutate aggro fields between
// pre-attack and end-of-round, but the legacy code captures these ids
// pre-attack — so we do the same.
func appendPreAttackAffected(
	atk, def actions.Actor,
	affectedPlayerIds *[]int,
	affectedMobInstanceIds *[]int,
) {
	atkChar := atk.GetCharacter()
	switch {
	case atk.IsPlayer() && def.IsPlayer():
		// Legacy PvP: append defender userId (atkChar.Aggro.UserId).
		if atkChar.Aggro != nil {
			*affectedPlayerIds = append(*affectedPlayerIds, atkChar.Aggro.UserId)
		}

	case atk.IsPlayer() && !def.IsPlayer():
		// Legacy PvM: appendsmobInstanceId (line 1070) + AggroUserId (line
		// 1092). The latter is 0 in PvM; preserved verbatim.
		if atkChar.Aggro != nil {
			*affectedMobInstanceIds = append(*affectedMobInstanceIds, atkChar.Aggro.MobInstanceId)
			*affectedPlayerIds = append(*affectedPlayerIds, atkChar.Aggro.UserId)
		}

	case !atk.IsPlayer() && def.IsPlayer():
		// Legacy MvP: append defender userId (mob.Aggro.UserId).
		if atkChar.Aggro != nil {
			*affectedPlayerIds = append(*affectedPlayerIds, atkChar.Aggro.UserId)
		}

	default: // MvM
		// Legacy MvM: append defender mobInstanceId (mob.Aggro.MobInstanceId).
		if atkChar.Aggro != nil {
			*affectedMobInstanceIds = append(*affectedMobInstanceIds, atkChar.Aggro.MobInstanceId)
		}
	}
}

// ────────────────────────────────────────────────────────────────────────
// Small adapters
// ────────────────────────────────────────────────────────────────────────

// asUser unwraps an Actor known to be a player. Panics if misused — only
// call after an IsPlayer() == true gate.
func asUser(a actions.Actor) *users.UserRecord {
	if u, ok := a.(*actions.UserActor); ok && u != nil {
		return u.User
	}
	panic("asUser called on non-player Actor")
}

// asMob unwraps an Actor known to be a mob. Panics if misused — only
// call after an IsPlayer() == false gate.
func asMob(a actions.Actor) *mobs.Mob {
	if m, ok := a.(*actions.MobActor); ok && m != nil {
		return m.Mob
	}
	panic("asMob called on non-mob Actor")
}

// asMobOrNil is the safe form of asMob — returns nil if the actor is not
// a *actions.MobActor (e.g., a future test double or scripted entity).
func asMobOrNil(a actions.Actor) *mobs.Mob {
	if m, ok := a.(*actions.MobActor); ok && m != nil {
		return m.Mob
	}
	return nil
}

// actorSourceTarget returns the combat.SourceTarget tag for an Actor.
func actorSourceTarget(a actions.Actor) combat.SourceTarget {
	if a.IsPlayer() {
		return combat.User
	}
	return combat.Mob
}

// userRecordOrNil returns the actor's *users.UserRecord when it's a
// player, nil otherwise. Used by the wait-round adapter.
func userRecordOrNil(a actions.Actor) *users.UserRecord {
	if a.IsPlayer() {
		return asUser(a)
	}
	return nil
}

// playerExcludeIds returns the user ids of any player participants in
// the round, suitable for a room broadcast exclusion list. Mobs return 0
// from GetUserId so they're naturally filtered.
func playerExcludeIds(atk, def actions.Actor) []int {
	excludes := []int{}
	if id := atk.GetUserId(); id != 0 {
		excludes = append(excludes, id)
	}
	if id := def.GetUserId(); id != 0 {
		excludes = append(excludes, id)
	}
	return excludes
}
