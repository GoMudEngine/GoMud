package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// CastResult holds all shared output from InitiateCast. The caller (user or
// mob wrapper) uses this to apply wrapper-specific logic and then commit the
// CastingState to the character.
type CastResult struct {
	// SpellInfo is the resolved spell data. Non-nil on success.
	SpellInfo *spells.SpellData

	// CastingState is pre-built but NOT yet applied to the character.
	// The wrapper must set char.CastingState = result.CastingState after its
	// own checks (e.g. conviction pre-check, initiation roll) pass.
	CastingState *characters.CastingState

	// Initiated is true when all shared checks passed and CastingState is ready.
	Initiated bool

	// Early-exit flags — exactly one will be true when Initiated is false.
	InvalidSpell   bool // spell not found by ID or name
	SpellNotKnown  bool // actor does not know the spell (player check only)
	AlreadyCasting bool // character already has a CastingState in progress
	OnCooldown     bool // special-move cooldown blocked the cast
	NoTarget       bool // required target could not be resolved

	// Computed values exposed so the wrapper can apply its own logic.
	FoldsNeeded   int
	FoldsPerRound int
	TotalCost     int // conviction cost (base, no multiplier applied)

	// Resolved targets and rest string.
	TargetUserIds        []int
	TargetMobInstanceIds []int
	SpellRest            string
}

// InitiateCast performs all shared cast setup logic for both player and mob
// callers:
//
//  1. Spell lookup (by ID then by display-name prefix)
//  2. Already-casting guard
//  3. Special-move cooldown gate (shared with bash / kick / trip)
//  4. Target resolution by spell type
//  5. Fold calculation
//  6. CastingState construction (NOT applied to char — caller does that)
//
// Player-only concerns (initiation roll, conviction pre-check, mutation
// modifiers, skill events, onCast script) are intentionally left to the
// caller.
//
// spellName is the raw first token (spell ID or partial display name).
// targetName is the remainder of the command line after the spell name.
// For mob callers, spellName should already be lowercased.
func InitiateCast(actor Actor, spellName, targetName string) CastResult {
	char := actor.GetCharacter()
	room := actor.GetRoom()

	// 1. Spell lookup — by exact ID first, then display-name prefix.
	spellInfo := spells.GetSpell(spellName)
	if spellInfo == nil {
		spellInfo = spells.FindSpellByName(spellName)
	}
	if spellInfo == nil {
		return CastResult{InvalidSpell: true}
	}

	// 2. Already casting?
	if char.CastingState != nil {
		return CastResult{SpellInfo: spellInfo, AlreadyCasting: true}
	}

	// 3. Special-move cooldown — shared slot with bash/kick/trip.
	cfg := configs.GetBalanceConfig()
	if !char.TryCooldown(`special-move`, fmt.Sprintf(`%d rounds`, cfg.SpecialMoveCooldown)) {
		return CastResult{SpellInfo: spellInfo, OnCooldown: true}
	}

	// 4. Resolve targets by spell type.
	targetUserIds := []int{}
	targetMobInstanceIds := []int{}
	spellRest := ``

	switch spellInfo.Type {

	case spells.HarmSingle:
		if targetName != `` {
			pId, mId := room.FindByName(targetName)
			if mId > 0 {
				// Don't let players target any player-owned companion with harm spells
				if actor.IsPlayer() {
					if m := mobs.GetInstance(mId); m != nil && m.Character.IsCharmed() {
						actor.SendText("You can't target a companion with a harmful spell.")
						return CastResult{SpellInfo: spellInfo, NoTarget: true}
					}
				}
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				// Players cannot self-target harmful single spells; mobs can
				// since they would never target themselves this way in practice.
				if actor.IsPlayer() && pId == actor.GetUserId() {
					return CastResult{SpellInfo: spellInfo, NoTarget: true}
				}
				// PvP check: block targeting other players if PvP isn't allowed
				if actor.IsPlayer() {
					casterUser := users.GetByUserId(actor.GetUserId())
					targetUser := users.GetByUserId(pId)
					if casterUser != nil && targetUser != nil {
						if pvpErr := room.CanPvp(casterUser, targetUser); pvpErr != nil {
							actor.SendText(pvpErr.Error())
							return CastResult{SpellInfo: spellInfo, NoTarget: true}
						}
					}
				}
				targetUserIds = append(targetUserIds, pId)
			} else {
				return CastResult{SpellInfo: spellInfo, NoTarget: true}
			}
		} else if actor.IsPlayer() {
			// Player fallback: current aggro → party leader's aggro.
			pId, mId := resolvePlayerAggroTarget(actor, room)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			} else {
				return CastResult{SpellInfo: spellInfo, NoTarget: true}
			}
		} else {
			// Mob fallback: current aggro → any fighter targeting this mob.
			pId, mId := resolveMobAggroTarget(actor, room)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			}
			// Mobs do not return NoTarget for HarmSingle — the guard below handles it.
		}

	case spells.HarmMulti:
		if targetName != `` {
			pId, mId := room.FindByName(targetName)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			}
		} else if actor.IsPlayer() {
			pId, mId := resolvePlayerAggroTarget(actor, room)
			if mId > 0 {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			} else if pId > 0 {
				targetUserIds = append(targetUserIds, pId)
			}
		} else {
			// Mob HarmMulti: all fighters targeting this mob.
			targetUserIds, targetMobInstanceIds = resolveMobHarmMultiTargets(actor, room)
		}

	case spells.HelpSingle:
		if actor.IsPlayer() {
			if targetName != `` && targetName != actor.GetName() {
				pId, mId := room.FindByName(targetName)
				if pId > 0 {
					targetUserIds = append(targetUserIds, pId)
				} else if mId > 0 {
					// Allow targeting companions with help spells
					if m := mobs.GetInstance(mId); m != nil && m.Character.IsCharmed(actor.GetUserId()) {
						targetMobInstanceIds = append(targetMobInstanceIds, mId)
					} else {
						return CastResult{SpellInfo: spellInfo, NoTarget: true}
					}
				} else {
					return CastResult{SpellInfo: spellInfo, NoTarget: true}
				}
			} else {
				targetUserIds = append(targetUserIds, actor.GetUserId()) // default to self
			}
		} else {
			// Mob HelpSingle: named target or self.
			if targetName != `` {
				pId, mId := room.FindByName(targetName)
				if pId > 0 {
					targetUserIds = append(targetUserIds, pId)
				} else if mId > 0 {
					targetMobInstanceIds = append(targetMobInstanceIds, mId)
				}
			} else {
				targetMobInstanceIds = append(targetMobInstanceIds, actor.GetMobInstanceId())
			}
		}

	case spells.HelpMulti:
		if actor.IsPlayer() {
			// Player: seed with self; spell script expands to party at resolution.
			targetUserIds = append(targetUserIds, actor.GetUserId())
		} else {
			// Mob HelpMulti: self + same-kind mobs (or charmed owner's party).
			targetMobInstanceIds, targetUserIds = resolveMobHelpMultiTargets(actor, room)
		}

	case spells.HarmArea:
		// Exclude self from harm area — don't damage yourself
		for _, pId := range room.GetPlayers() {
			if pId != actor.GetUserId() {
				// PvP check: skip other players if PvP isn't allowed here
				if actor.IsPlayer() {
					casterUser := users.GetByUserId(actor.GetUserId())
					targetUser := users.GetByUserId(pId)
					if casterUser != nil && targetUser != nil {
						if pvpErr := room.CanPvp(casterUser, targetUser); pvpErr != nil {
							continue
						}
					}
				}
				targetUserIds = append(targetUserIds, pId)
			}
		}
		for _, mId := range room.GetMobs() {
			if mId != actor.GetMobInstanceId() {
				// Don't target the caster's own companions
				if actor.IsPlayer() {
					if m := mobs.GetInstance(mId); m != nil && m.Character.IsCharmed(actor.GetUserId()) {
						continue
					}
				}
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			}
		}

	case spells.HelpArea:
		targetUserIds = room.GetPlayers()
		targetMobInstanceIds = room.GetMobs()

	case spells.Neutral:
		spellRest = targetName
	}

	// 4a. Harm spells (single/multi) require at least one resolved target.
	// HarmArea determines targets at resolution time via script — skip check.
	if spellInfo.Type == spells.HarmSingle || spellInfo.Type == spells.HarmMulti {
		if len(targetUserIds) == 0 && len(targetMobInstanceIds) == 0 {
			return CastResult{SpellInfo: spellInfo, NoTarget: true}
		}
	}

	// 5. Fold calculation — school-aware stat/skill selection.
	baseFolds := spellInfo.BaseFolds
	if baseFolds < 1 {
		baseFolds = 4
	}
	foldsNeeded := characters.NextPowerOfTwo(baseFolds)
	primaryStat, spellSkillLevel := GetSpellStatAndSkill(char, spellInfo)
	foldsPerRound := characters.CalcFoldsPerRound(primaryStat, spellSkillLevel)

	// 6. Conviction cost (base, no multiplier — caller applies mutations).
	totalCost := spellInfo.Cost

	// 7. Build CastingState — NOT applied to char here.
	cs := &characters.CastingState{
		SpellId:              spellInfo.SpellId,
		FoldsNeeded:          foldsNeeded,
		FoldsAccumulated:     0,
		FoldsPerRound:        foldsPerRound,
		TotalConvictionCost:  totalCost,
		ConvictionSpent:      0,
		TargetUserIds:        targetUserIds,
		TargetMobInstanceIds: targetMobInstanceIds,
		SpellRest:            spellRest,
	}

	// Progression: cast skill on successful initiation (covers both players and mobs).
	castSkill := string(skills.Spellcasting)
	if spellInfo.HasSchool(spells.SchoolManifestation) {
		castSkill = string(skills.Manifestation)
	}
	actor.OnSkillUse(castSkill)

	return CastResult{
		SpellInfo:            spellInfo,
		CastingState:         cs,
		Initiated:            true,
		FoldsNeeded:          foldsNeeded,
		FoldsPerRound:        foldsPerRound,
		TotalCost:            totalCost,
		TargetUserIds:        targetUserIds,
		TargetMobInstanceIds: targetMobInstanceIds,
		SpellRest:            spellRest,
	}
}

// GetSpellStatAndSkill returns the primary stat value and skill level used for
// fold calculation for a given spell. Manifestation spells use Charisma +
// manifestation skill; all other spells use Perception + spellcasting.
func GetSpellStatAndSkill(char *characters.Character, spellData *spells.SpellData) (statValue int, skillLevel int) {
	if spellData.HasSchool(spells.SchoolManifestation) {
		return char.Stats.Charisma.ValueAdj,
			char.GetSkillLevel(skills.Manifestation)
	}
	// Default: perception + spellcasting (traditional magic)
	return char.Stats.Perception.ValueAdj,
		char.GetSkillLevel(skills.Spellcasting)
}

// ---------------------------------------------------------------------------
// Internal target resolution helpers
// ---------------------------------------------------------------------------

// resolvePlayerAggroTarget returns the player's current aggro target, falling
// back to the party leader's aggro target if the player has no aggro of their
// own. Returns (userId, mobInstanceId) with at most one non-zero.
func resolvePlayerAggroTarget(actor Actor, room *rooms.Room) (int, int) {
	char := actor.GetCharacter()
	if char.Aggro != nil {
		if char.Aggro.MobInstanceId > 0 {
			return 0, char.Aggro.MobInstanceId
		}
		if char.Aggro.UserId > 0 {
			return char.Aggro.UserId, 0
		}
	}
	// Party leader fallback.
	if p := parties.Get(actor.GetUserId()); p != nil {
		if leaderUser := users.GetByUserId(p.LeaderUserId); leaderUser != nil {
			if leaderUser.Character.RoomId == char.RoomId && leaderUser.Character.Aggro != nil {
				if leaderUser.Character.Aggro.MobInstanceId > 0 {
					return 0, leaderUser.Character.Aggro.MobInstanceId
				}
				if leaderUser.Character.Aggro.UserId > 0 {
					return leaderUser.Character.Aggro.UserId, 0
				}
			}
		}
	}
	return 0, 0
}

// resolveMobAggroTarget returns the mob's current aggro target, falling back
// to the first fighter in the room that is aggro toward this mob.
func resolveMobAggroTarget(actor Actor, room *rooms.Room) (int, int) {
	char := actor.GetCharacter()
	mobId := actor.GetMobInstanceId()

	if char.Aggro != nil {
		if char.Aggro.UserId > 0 {
			return char.Aggro.UserId, 0
		}
		if char.Aggro.MobInstanceId > 0 {
			return 0, char.Aggro.MobInstanceId
		}
	}

	// First player in room that is fighting this mob.
	for _, pUserId := range room.GetPlayers(rooms.FindFightingMob) {
		if u := users.GetByUserId(pUserId); u != nil {
			if u.Character.IsAggro(0, mobId) {
				return pUserId, 0
			}
		}
	}

	// First mob in room that is fighting this mob.
	for _, mId := range room.GetMobs(rooms.FindFightingMob) {
		if m := mobs.GetInstance(mId); m != nil {
			if m.Character.IsAggro(0, mobId) {
				return 0, mId
			}
		}
	}

	return 0, 0
}

// resolveMobHarmMultiTargets collects all players/mobs in the room that are
// aggro toward the casting mob (mirrors the mob cast.go HarmMulti logic).
func resolveMobHarmMultiTargets(actor Actor, room *rooms.Room) ([]int, []int) {
	mobId := actor.GetMobInstanceId()
	targetUserIds := []int{}
	targetMobInstanceIds := []int{}

	for _, mId := range room.GetMobs(rooms.FindFightingMob) {
		if m := mobs.GetInstance(mId); m != nil {
			if m.Character.IsAggro(0, mobId) || m.HatesSpecies(m.Character.Species()) {
				targetMobInstanceIds = append(targetMobInstanceIds, mId)
			}
		}
	}
	for _, uId := range room.GetPlayers(rooms.FindFightingMob) {
		if u := users.GetByUserId(uId); u != nil {
			if u.Character.IsAggro(0, mobId) {
				targetUserIds = append(targetUserIds, uId)
			}
		}
	}

	return targetUserIds, targetMobInstanceIds
}

// resolveMobHelpMultiTargets builds the HelpMulti target list for a mob:
// self + same-kind mobs (uncharmed) or charmed owner + party (charmed).
// Returns (mobInstanceIds, userIds) — note parameter order matches usage.
func resolveMobHelpMultiTargets(actor Actor, room *rooms.Room) ([]int, []int) {
	mob := actor.(*MobActor).Mob
	targetMobInstanceIds := []int{mob.InstanceId}
	targetUserIds := []int{}

	if !mob.Character.IsCharmed() {
		for _, mInstId := range room.GetMobs() {
			if m := mobs.GetInstance(mInstId); m != nil && m.MobId == mob.MobId {
				targetMobInstanceIds = append(targetMobInstanceIds, mInstId)
			}
		}
	} else {
		ownerId := mob.Character.Charmed.UserId
		targetUserIds = append(targetUserIds, ownerId)
		if p := parties.Get(ownerId); p != nil {
			for _, partyUserId := range p.GetMembers() {
				if partyUserId == ownerId {
					continue
				}
				if partyUser := users.GetByUserId(partyUserId); partyUser != nil {
					targetUserIds = append(targetUserIds, partyUserId)
					targetMobInstanceIds = append(targetMobInstanceIds, partyUser.Character.GetCharmIds()...)
				}
			}
		}
	}

	return targetMobInstanceIds, targetUserIds
}

