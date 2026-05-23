package actions

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// SurpriseAttackOpts parameterizes a surprise-attack trigger attempt.
type SurpriseAttackOpts struct {
	// Target is the actor being attacked. Exactly one of Target.GetUserId()
	// or Target.GetMobInstanceId() should be non-zero when Target is non-nil.
	Target Actor
}

// SurpriseAttackResult is the structured outcome of a SurpriseAttack call.
// On Triggered=false, BlockReason is one of:
//
//	"not-hidden"  — attacker is not in the Hidden state
//	"no-target"   — Target is nil
//	"on-cooldown" — the special-move cooldown is active
//
// On Triggered=true, StrikeCount is the number of per-weapon swings that
// connected (misses do not count).
type SurpriseAttackResult struct {
	Triggered   bool
	StrikeCount int
	BlockReason string
}

// SurpriseAttack fires the pre-combat surprise-attack burst. It requires the
// attacker to be in the Hidden state and have a free special-move cooldown.
// On success it iterates every equipped weapon (falling back to unarmed fists)
// and rolls an attack against the target, applying a crit-style half-mitigation
// bypass and a dex+skullduggery surprise multiplier.
//
// The Awareness_Cascades hook — not this function — handles the
// Hidden → Revealing transition at end of round. Do not call CancelBuff or
// clear Hidden state here.
//
// Both the player wrapper (usercommands/attack.go) and the mob wrapper
// (mobcommands/attack.go) collapse to thin call-throughs. The hidden-state
// gate and cooldown try are handled internally so the callers need no
// pre-flight logic.
func SurpriseAttack(actor Actor, opts SurpriseAttackOpts) SurpriseAttackResult {

	// --- Gate 1: attacker must be hidden ---
	if !actor.GetCharacter().IsHidden() {
		return SurpriseAttackResult{BlockReason: "not-hidden"}
	}

	// --- Gate 2: target must be present ---
	if opts.Target == nil {
		return SurpriseAttackResult{BlockReason: "no-target"}
	}

	// --- Gate 3: special-move cooldown ---
	cfg := configs.GetBalanceConfig()
	if !actor.GetCharacter().TryCooldown("special-move",
		fmt.Sprintf("%d rounds", cfg.SpecialMoveCooldown)) {
		return SurpriseAttackResult{BlockReason: "on-cooldown"}
	}

	// --- Resolve target vitals ---
	targetChar := opts.Target.GetCharacter()
	targetName := opts.Target.GetName()

	// For mob targets, use the indexed mob name if we can resolve the room
	// and a viewing user id is available (player attacker).
	if opts.Target.GetMobInstanceId() > 0 && actor.GetRoom() != nil {
		dupIdx := actor.GetRoom().GetMobDuplicateIndex(opts.Target.GetMobInstanceId())
		indexed := targetChar.GetMobNameIndexed(actor.GetUserId(), dupIdx)
		targetName = indexed.String()
	}

	defenderMaxHP := targetChar.HealthMax.Value
	defenderMitigation := targetChar.GetPhysicalMitigation()

	// --- Attacker stats ---
	attackerChar := actor.GetCharacter()
	dex := attackerChar.Stats.Dexterity.ValueAdj
	skillRank := attackerChar.GetSkillLevel(skills.Skullduggery)

	// Surprise multiplier: (dex + skillRank) / 100, minimum 1.0
	surpriseMult := math.Max(1.0, float64(dex+skillRank)/100.0)

	// --- Collect attack weapons (mirrors the player implementation) ---
	type weaponEntry struct {
		itemId     int
		name       string
		dmgMult    float64
		hitPenalty float64 // fraction: 0.0 = no penalty, 0.10 = 10% miss added
	}

	weapons := []weaponEntry{}

	// Primary weapon
	if attackerChar.Equipment.Weapon.ItemId > 0 {
		spec := attackerChar.Equipment.Weapon.GetSpec()
		wm := spec.DamageMultiplier
		if wm <= 0 {
			wm = float64(cfg.UnarmedDamageMultiplier)
		}
		weapons = append(weapons, weaponEntry{
			itemId:     attackerChar.Equipment.Weapon.ItemId,
			name:       spec.NameSimple,
			dmgMult:    wm,
			hitPenalty: 0.0,
		})
	}

	// Offhand weapon
	if attackerChar.Equipment.Offhand.ItemId > 0 {
		offSpec := attackerChar.Equipment.Offhand.GetSpec()
		if offSpec.Type == items.Weapon {
			wm := offSpec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{
				itemId:     attackerChar.Equipment.Offhand.ItemId,
				name:       offSpec.NameSimple,
				dmgMult:    wm,
				hitPenalty: float64(cfg.SurpriseAttackOffhandPenalty),
			})
		}
	}

	// Extra arm 1
	if attackerChar.ExtraArms >= 1 && attackerChar.Equipment.ExtraArm1.ItemId > 0 {
		ea1Spec := attackerChar.Equipment.ExtraArm1.GetSpec()
		if ea1Spec.Type == items.Weapon {
			wm := ea1Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{
				itemId:     attackerChar.Equipment.ExtraArm1.ItemId,
				name:       ea1Spec.NameSimple,
				dmgMult:    wm,
				hitPenalty: float64(cfg.SurpriseAttackExtraArm1Penalty),
			})
		}
	}

	// Extra arm 2
	if attackerChar.ExtraArms >= 2 && attackerChar.Equipment.ExtraArm2.ItemId > 0 {
		ea2Spec := attackerChar.Equipment.ExtraArm2.GetSpec()
		if ea2Spec.Type == items.Weapon {
			wm := ea2Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{
				itemId:     attackerChar.Equipment.ExtraArm2.ItemId,
				name:       ea2Spec.NameSimple,
				dmgMult:    wm,
				hitPenalty: float64(cfg.SurpriseAttackExtraArm2Penalty),
			})
		}
	}

	// Extra arm 3
	if attackerChar.ExtraArms >= 3 && attackerChar.Equipment.ExtraArm3.ItemId > 0 {
		ea3Spec := attackerChar.Equipment.ExtraArm3.GetSpec()
		if ea3Spec.Type == items.Weapon {
			wm := ea3Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{
				itemId:     attackerChar.Equipment.ExtraArm3.ItemId,
				name:       ea3Spec.NameSimple,
				dmgMult:    wm,
				hitPenalty: float64(cfg.SurpriseAttackExtraArm3Penalty),
			})
		}
	}

	// Extra arm 4
	if attackerChar.ExtraArms >= 4 && attackerChar.Equipment.ExtraArm4.ItemId > 0 {
		ea4Spec := attackerChar.Equipment.ExtraArm4.GetSpec()
		if ea4Spec.Type == items.Weapon {
			wm := ea4Spec.DamageMultiplier
			if wm <= 0 {
				wm = float64(cfg.UnarmedDamageMultiplier)
			}
			weapons = append(weapons, weaponEntry{
				itemId:     attackerChar.Equipment.ExtraArm4.ItemId,
				name:       ea4Spec.NameSimple,
				dmgMult:    wm,
				hitPenalty: float64(cfg.SurpriseAttackExtraArm4Penalty),
			})
		}
	}

	// Fallback: unarmed
	if len(weapons) == 0 {
		weapons = append(weapons, weaponEntry{
			itemId:     0,
			name:       "fists",
			dmgMult:    float64(cfg.UnarmedDamageMultiplier),
			hitPenalty: 0.0,
		})
	}

	// --- Swing each weapon ---
	room := actor.GetRoom()
	attackerName := actor.GetName()
	strikeCount := 0
	anyHit := false

	for _, w := range weapons {
		// Hit check: roll 0-99; if result < penaltyPct → miss this weapon
		penaltyPct := int(w.hitPenalty * 100)
		roll := util.Rand(100)
		if roll < penaltyPct {
			// Missed this weapon swing due to hit penalty
			actor.SendText(messaging.CategorySystem,
				fmt.Sprintf(
					`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
						`You swing your <ansi fg="item">%s</ansi> at `+
						`<ansi fg="mobname">%s</ansi> but miss!`,
					w.name, targetName,
				),
			)
			if room != nil {
				room.SendTextVisual(messaging.CategorySurpriseAttack,
					fmt.Sprintf(
						`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
							`<ansi fg="username">%s</ansi> swings at %s from `+
							`the shadows but misses!`,
						attackerName, targetName,
					),
					actor.GetUserId(),
				)
			}
			continue
		}

		// Damage pipeline: CalcRawDamage → half-mitigation bypass →
		// surprise multiplier → dice.RollStat variance
		rawDmg := combat.CalcRawDamage(
			attackerChar.Stats.Strength.ValueAdj,
			skillRank,
			w.dmgMult,
			combat.ChannelPhysical,
		)

		// Surprise attack: apply only half of normal mitigation (crit-like bypass)
		halfMitigation := defenderMitigation * 0.5
		dmgMean := combat.ApplyMitigation(
			rawDmg,
			halfMitigation,
			combat.MitigationCap(combat.ChannelPhysical),
		)

		// Apply surprise multiplier
		dmgMean *= surpriseMult

		// Variance roll
		dmgResult := dice.RollStat(dmgMean)
		dmg := int(math.Round(math.Max(1.0, dmgResult.Value)))

		anyHit = true
		strikeCount++

		// Apply damage to target
		targetChar.Health -= dmg
		if targetChar.Health < 0 {
			targetChar.Health = 0
		}

		// Per-weapon hit messages
		dmgDesc := combat.GetDamageDescription(dmg, defenderMaxHP)

		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(
				`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
					`Your <ansi fg="item">%s</ansi> strikes `+
					`<ansi fg="mobname">%s</ansi> from the shadows! `+
					`(<ansi fg="damage">%s</ansi>)`,
				w.name, targetName, dmgDesc,
			),
		)

		if room != nil {
			room.SendTextVisual(messaging.CategorySurpriseAttack,
				fmt.Sprintf(
					`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
						`<ansi fg="username">%s</ansi>'s <ansi fg="item">%s</ansi> `+
						`strikes %s from the shadows!`,
					attackerName, w.name, targetName,
				),
				actor.GetUserId(),
			)
		}

		// Notify the target if they are a player
		opts.Target.SendText(messaging.CategorySystem,
			fmt.Sprintf(
				`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
					`<ansi fg="username">%s</ansi>'s <ansi fg="item">%s</ansi> `+
					`strikes you from the shadows! (<ansi fg="damage">%s</ansi>)`,
				attackerName, w.name, dmgDesc,
			),
		)
	}

	if !anyHit {
		// All weapons missed — still reveal the attacker's position
		actor.SendText(messaging.CategorySystem,
			fmt.Sprintf(
				`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
					`You lunge at <ansi fg="mobname">%s</ansi> from `+
					`the shadows, but miss!`,
				targetName,
			),
		)
		if room != nil {
			room.SendTextVisual(messaging.CategorySurpriseAttack,
				fmt.Sprintf(
					`<ansi fg="magenta-bold">*[SURPRISE ATTACK]*</ansi> `+
						`<ansi fg="username">%s</ansi> lunges at %s from `+
						`the shadows, but misses!`,
					attackerName, targetName,
				),
				actor.GetUserId(),
			)
		}
	}

	// --- Skill progression ---
	actor.OnSkillUse(string(skills.Skullduggery))

	return SurpriseAttackResult{
		Triggered:   true,
		StrikeCount: strikeCount,
	}
}
