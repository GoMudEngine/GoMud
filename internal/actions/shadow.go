package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ShadowOptions parameterizes a shadow attempt.
// Exactly one of TargetMobInstanceId / TargetUserId must be set.
type ShadowOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
}

// ShadowResult is the structured outcome of a shadow attempt.
type ShadowResult struct {
	Succeeded  bool   // target id was stored and shadow tracking began
	Detected   bool   // target won the initial detection roll
	TargetName string // display name of the target
	OnCooldown bool   // attempt was blocked by shadow cooldown
	Reason     string // when Succeeded==false and !OnCooldown, why
}

// Shadow attempts to track a target while hidden. Requires the actor to
// already carry buff 9 (Hidden). On success, stores the target's id in
// the actor's misc-data so the engine's existing shadow-follow event
// listener can auto-follow on the target's room-change events. An initial
// detection roll is performed for player targets: if the target wins, they
// sense pursuit (Detected=true) but shadow still begins.
func Shadow(actor Actor, opts ShadowOptions) ShadowResult {
	char := actor.GetCharacter()

	// Must be hidden to shadow.
	if !char.IsHidden() {
		actor.SendText(messaging.CategorySystem,
			"You must be hidden to shadow someone. "+
				`Try <ansi fg="command">sneak</ansi> first.`)
		return ShadowResult{Reason: "not hidden"}
	}

	// Combat gate.
	if char.Aggro != nil {
		actor.SendText(messaging.CategorySystem, "You can't do that while in combat!")
		return ShadowResult{Reason: "in combat"}
	}

	// Require a target.
	if opts.TargetMobInstanceId == 0 && opts.TargetUserId == 0 {
		actor.SendText(messaging.CategorySystem, "Shadow whom?")
		return ShadowResult{Reason: "no target"}
	}

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`shadow`)

	// Check cooldown before doing target resolution.
	if !char.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d rounds`, cfg.ShadowCooldown)) {
		return ShadowResult{
			OnCooldown: true,
			Reason: fmt.Sprintf("%d rounds remaining",
				char.GetCooldown(cooldownKey)),
		}
	}

	if opts.TargetMobInstanceId > 0 {
		return shadowMob(actor, opts.TargetMobInstanceId, cfg)
	}

	return shadowPlayer(actor, opts.TargetUserId, cfg)
}

// shadowMob handles the mob-target shadow path.
func shadowMob(actor Actor, mobInstanceId int, cfg configs.Balance) ShadowResult {
	m := mobs.GetInstance(mobInstanceId)
	if m == nil {
		actor.SendText(messaging.CategorySystem, "They seem to have vanished.")
		return ShadowResult{Reason: "target not found"}
	}

	char := actor.GetCharacter()
	char.SetMiscData("shadow-target-user", nil)
	char.SetMiscData("shadow-target-mob", m.InstanceId)
	actor.AddBuff(87, "skill")

	actor.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You begin shadowing <ansi fg="mobname">%s</ansi>, `+
			`moving silently in their wake.`,
		m.Character.Name))

	// Skill-use and progression.
	actor.OnSkillUse(string(skills.Skullduggery))

	// Quest engine notification — player actors only.
	if actor.IsPlayer() {
		if u := users.GetByUserId(actor.GetUserId()); u != nil {
			room := actor.GetRoom()
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  actor.GetUserId(),
				RoomId:  room.RoomId,
				Command: "shadow",
			}, bridge, bridge)

			events.AddToQueue(events.SkillUsed{
				UserId:  actor.GetUserId(),
				Skill:   skills.Skullduggery,
				Details: `shadow`,
			})
		}
	}

	return ShadowResult{
		Succeeded:  true,
		TargetName: m.Character.Name,
	}
}

// shadowPlayer handles the player-target shadow path. An initial detection
// roll is performed: if the target wins (their Per+Search beats the actor's
// Dex+Skullduggery), the target senses pursuit. Shadow still begins
// regardless — the Detected flag is informational.
func shadowPlayer(actor Actor, targetUserId int, cfg configs.Balance) ShadowResult {
	targetUser := users.GetByUserId(targetUserId)
	if targetUser == nil {
		actor.SendText(messaging.CategorySystem, "They seem to have vanished.")
		return ShadowResult{Reason: "target not found"}
	}

	char := actor.GetCharacter()
	char.SetMiscData("shadow-target-user", targetUser.UserId)
	char.SetMiscData("shadow-target-mob", nil)
	actor.AddBuff(87, "skill")

	actor.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You begin shadowing <ansi fg="username">%s</ansi>, `+
			`watching their every move.`,
		targetUser.Character.Name))

	// Skill-use and progression.
	actor.OnSkillUse(string(skills.Skullduggery))

	// Quest engine notification — player actors only.
	if actor.IsPlayer() {
		if u := users.GetByUserId(actor.GetUserId()); u != nil {
			room := actor.GetRoom()
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  actor.GetUserId(),
				RoomId:  room.RoomId,
				Command: "shadow",
			}, bridge, bridge)

			events.AddToQueue(events.SkillUsed{
				UserId:  actor.GetUserId(),
				Skill:   skills.Skullduggery,
				Details: `shadow`,
			})
		}
	}

	// Initial detection roll: target wins if their search score beats the
	// actor's sneak score. Uses the same formula as shadowDetectionRoll in
	// go.go (Per+Search vs Dex+Skullduggery; OpposedRollStat: first arg wins).
	sneakScore := CalcSneakScoreVsObserver(char, targetUser.Character, actor.GetRoom())
	searchScore := CalcSearchScore(targetUser.Character)
	detected, _, _, _ := dice.OpposedRollStat(searchScore, sneakScore)
	if detected {
		targetUser.SendText(messaging.CategorySystem, "You sense someone following close behind you.")
	}

	return ShadowResult{
		Succeeded:  true,
		Detected:   detected,
		TargetName: targetUser.Character.Name,
	}
}
