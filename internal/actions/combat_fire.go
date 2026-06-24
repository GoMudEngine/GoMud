package actions

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// FireResult holds the outcome of a shoot/fire attempt.
type FireResult struct {
	WeaponName string
	TargetName string

	TargetUserId        int
	TargetMobInstanceId int
	TargetRoomId        int
	IsTargetMob         bool

	CrossRoom  bool
	ExitName   string
	IsSneaking bool

	MoveResult combat.SkillMoveResult
	Executed   bool

	NoWeapon       bool
	NotLoaded      bool
	BadSyntax      bool
	ExitLocked     bool
	NoTarget       bool
	IsCharmed      bool
	IsNonCombatant bool
	Crafting       bool
}

// rangedDefenseScore computes the defender's score against a ranged shot:
// Dexterity + combat skill, plus the configured shield bonus when an offhand
// with a block rating is equipped. Parry contributes nothing to ranged
// defense by design (you can't parry a bolt).
func rangedDefenseScore(defender *characters.Character) float64 {
	score := float64(defender.GetEffectiveDexterity()) + float64(defender.GetCombatSkillLevel())
	if defender.Equipment.Offhand.ItemId > 0 {
		// NOTE: Item.GetSpec() returns an ItemSpec by VALUE, not a pointer, so
		// there is intentionally no nil check here (one would not even compile).
		// A registry miss yields the zero-value ItemSpec, whose BlockRating is 0,
		// so the guard below simply skips the shield bonus. This is safe by
		// value semantics — please don't "fix" it by adding a nil check.
		if spec := defender.Equipment.Offhand.GetSpec(); spec.BlockRating > 0 {
			score += float64(configs.GetBalanceConfig().RangedShieldDefenseBonus)
		}
	}
	return score
}

// ExecuteFire resolves a ranged shot immediately. rest is either "<target>"
// (same room) or "<target words...> <direction>" (adjacent room). The weapon
// must be loaded; firing unloads it (even on a miss). Firing does NOT consume
// the special-move cooldown — reloading does. It DOES consume the attacker's
// combat round via RecordAndWait.
//
// Callers are responsible for: messaging, OnSkillUse/OnStatUse progression,
// retaliation aggro on the target, crime recording, and combat-initiation
// aggro for same-room shots.
func ExecuteFire(actor Actor, rest string) FireResult {
	char := actor.GetCharacter()

	// Don't interrupt any active activity (cast/craft/salvage) to fire.
	if char.IsActing() {
		return FireResult{Crafting: true}
	}

	weapon := findRangedWeaponSlot(actor)
	if weapon == nil {
		return FireResult{NoWeapon: true}
	}
	if !weapon.Loaded {
		return FireResult{WeaponName: weapon.DisplayName(), NotLoaded: true}
	}

	args := strings.Fields(rest)
	if len(args) < 1 {
		return FireResult{WeaponName: weapon.DisplayName(), BadSyntax: true}
	}

	// Try the last word as a direction (cross-room shot); fall back to a
	// same-room target if it isn't a usable exit.
	room := actor.GetRoom()
	if room == nil {
		return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
	}

	crossRoom := false
	exitName := ""
	targetRoom := room
	targetWords := args

	if len(args) >= 2 {
		if name, roomId := room.FindExitByName(args[len(args)-1]); name != "" {
			exitInfo, _ := room.GetExitInfo(name)
			if exitInfo.Lock.IsLocked() {
				return FireResult{WeaponName: weapon.DisplayName(), ExitName: name, ExitLocked: true}
			}
			if adj := rooms.LoadRoom(roomId); adj != nil {
				crossRoom = true
				exitName = name
				targetRoom = adj
				targetWords = args[:len(args)-1]
			}
		}
	}

	targetUserId, targetMobInstanceId := targetRoom.FindByName(strings.Join(targetWords, " "))
	if targetUserId == 0 && targetMobInstanceId == 0 && crossRoom {
		// The trailing word may have been part of the target name after all;
		// retry as a same-room shot using the full argument string.
		crossRoom, exitName, targetRoom = false, "", room
		targetUserId, targetMobInstanceId = room.FindByName(strings.Join(args, " "))
	}
	if targetUserId == 0 && targetMobInstanceId == 0 {
		return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
	}

	result := FireResult{
		WeaponName:          weapon.DisplayName(),
		TargetUserId:        targetUserId,
		TargetMobInstanceId: targetMobInstanceId,
		TargetRoomId:        targetRoom.RoomId,
		CrossRoom:           crossRoom,
		ExitName:            exitName,
		IsSneaking:          char.IsHidden(),
	}

	// Resolve the defender character. Charmed / non-combatant checks happen
	// BEFORE unloading — a refused shot is not wasted.
	var defChar *characters.Character
	if targetMobInstanceId > 0 {
		m := mobs.GetInstance(targetMobInstanceId)
		if m == nil {
			return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
		}

		// Charmed-mob friendly-fire prevention. Player actors charm by userId;
		// mob actors charm by instanceId.
		charmerKey := actor.GetUserId()
		if charmerKey == 0 {
			charmerKey = actor.GetMobInstanceId()
		}
		if m.Character.IsCharmed(charmerKey) {
			result.IsCharmed = true
			result.TargetName = m.Character.Name
			return result
		}
		if m.IsNonCombatant() || m.PlayerAttackImmune {
			result.IsNonCombatant = true
			result.TargetName = m.Character.Name
			return result
		}

		result.IsTargetMob = true
		result.TargetName = m.Character.Name
		defChar = &m.Character
	} else {
		u := users.GetByUserId(targetUserId)
		if u == nil {
			return FireResult{WeaponName: weapon.DisplayName(), NoTarget: true}
		}
		result.TargetName = u.Character.Name
		defChar = u.Character
	}

	// The shot: unload first (fires even on a miss), then resolve.
	weapon.Loaded = false

	cfg := configs.GetBalanceConfig()
	shotMult := weapon.GetSpec().DamageMultiplier * float64(cfg.RangedShotScale)
	rangedRank := char.GetSkillLevel(skills.RangedCombat)

	result.MoveResult = combat.ExecuteSkillMove(combat.SkillMoveParams{
		Attacker:             char,
		Defender:             defChar,
		AttackStat:           char.GetEffectivePerception(),
		AttackSkill:          rangedRank,
		DefenseStat:          0, // folded into DefenseSkill via rangedDefenseScore
		DefenseSkill:         int(rangedDefenseScore(defChar)),
		DamagePercent:        shotMult,
		KnockdownChance:      0,
		SkillRank:            rangedRank,
		DamageStat:           char.GetEffectivePerception(),
		MitigationMultiplier: 1.0,
	})
	result.Executed = true

	// Analytics + round consumption (same pattern as kick). Fire never burns
	// the special-move cooldown — only the combat round.
	sourceType := combat.User
	if !actor.IsPlayer() {
		sourceType = combat.Mob
	}
	targetType := combat.User
	if result.IsTargetMob {
		targetType = combat.Mob
	}
	dmg := 0
	if result.MoveResult.Hit {
		dmg = result.MoveResult.Damage
	}
	RecordAndWait(char, "shoot", sourceType, defChar, targetType, result.MoveResult.Hit, dmg, util.GetRoundCount())

	return result
}
