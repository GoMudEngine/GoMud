package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/awareness"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PlantOptions parameterizes a plant attempt.
// Exactly one of TargetMobInstanceId / TargetUserId / ContainerNoun
// must be set. ItemNoun is REQUIRED — names the item in the actor's
// backpack to plant.
type PlantOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
	ContainerNoun       string
	ItemNoun            string
}

// PlantResult is the structured outcome of a plant attempt.
type PlantResult struct {
	Succeeded     bool   // skill check passed and transfer happened
	Detected      bool   // detection roll fired and the defender noticed
	PlantedItemId int    // item id transferred (0 if failed)
	DefenderName  string // who/what was targeted
	OnCooldown    bool   // attempt was blocked by skullduggery cooldown
	Reason        string // when Succeeded==false and !OnCooldown, why
}

// Plant runs a skullduggery plant attempt from actor against the
// resolved target. Both UserActor and MobActor are supported.
// Shares the skullduggery cooldown key with Steal.
func Plant(actor Actor, opts PlantOptions) PlantResult {
	char := actor.GetCharacter()

	// Combat gate — can't plant while actively fighting.
	if char.Aggro != nil {
		actor.SendText(messaging.CategorySystem, "You can't do that while in combat!")
		return PlantResult{Reason: "in combat"}
	}

	room := actor.GetRoom()
	if room == nil {
		return PlantResult{Reason: "no room"}
	}

	// Under-attack gate — mobs with userId=0 short-circuit this harmlessly.
	if room.AreMobsAttacking(actor.GetUserId()) {
		actor.SendText(messaging.CategorySystem, "You can't do that while you are under attack!")
		return PlantResult{Reason: "under attack"}
	}

	// Require a target.
	if opts.TargetMobInstanceId == 0 && opts.TargetUserId == 0 &&
		opts.ContainerNoun == "" {
		actor.SendText(messaging.CategorySystem, "Plant on whom?")
		return PlantResult{Reason: "no target"}
	}

	// Require an item noun.
	if opts.ItemNoun == "" {
		actor.SendText(messaging.CategorySystem, "Plant what?")
		return PlantResult{Reason: "no item"}
	}

	// Find item in actor's backpack.
	plantItem, found := char.FindInBackpack(opts.ItemNoun)
	if !found {
		actor.SendText(messaging.CategorySystem, "You don't have that.")
		return PlantResult{Reason: "item not in backpack"}
	}

	cfg := configs.GetBalanceConfig()
	// Shares the steal cooldown key — intentional (same skullduggery cooldown).
	cooldownKey := skills.Skullduggery.String(`steal`)

	// Check cooldown before doing target resolution.
	if !char.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d real seconds`, int(cfg.StealCooldown))) {
		return PlantResult{
			OnCooldown: true,
			Reason: fmt.Sprintf("%d rounds remaining",
				char.GetCooldown(cooldownKey)),
		}
	}

	isHidden := char.IsHidden()

	// Compute attacker score (identical formula to steal).
	rank := char.GetSkillLevel(skills.Skullduggery)
	base := float64(char.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(rank)*25.0
	attackerScore := base * float64(cfg.StealSkillMultiplier)
	if isHidden {
		attackerScore += float64(cfg.StealHiddenBonus)
	}

	// Dispatch to the appropriate path.
	if opts.TargetMobInstanceId > 0 {
		return plantOnMob(actor, opts.TargetMobInstanceId, plantItem,
			attackerScore, rank)
	}

	if opts.TargetUserId > 0 {
		return plantOnPlayer(actor, opts.TargetUserId, plantItem,
			attackerScore, rank, cfg)
	}

	// Container path.
	return plantInContainer(actor, opts.ContainerNoun, plantItem,
		attackerScore, rank)
}

// plantOnMob handles slipping an item into a creature's inventory.
func plantOnMob(actor Actor, mobInstanceId int, plantItem items.Item,
	attackerScore float64, rank int) PlantResult {

	m := mobs.GetInstance(mobInstanceId)
	if m == nil {
		actor.SendText(messaging.CategorySystem, "They seem to have vanished.")
		return PlantResult{Reason: "target not found"}
	}

	// Skill-use and progression — always fire regardless of roll outcome.
	actor.OnSkillUse(string(skills.Skullduggery))

	// Quest engine notification — player actors only.
	if actor.IsPlayer() {
		if u := users.GetByUserId(actor.GetUserId()); u != nil {
			room := actor.GetRoom()
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  actor.GetUserId(),
				RoomId:  room.RoomId,
				Command: "plant",
			}, bridge, bridge)
		}
	}

	defenderScore := float64(m.Character.Stats.Perception.ValueAdj)
	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	room := actor.GetRoom()

	if success {
		m.Character.StoreItem(plantItem)
		actor.GetCharacter().RemoveItem(plantItem)

		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: m.InstanceId,
			Item:          plantItem,
			Gained:        true,
		})

		if actor.IsPlayer() {
			events.AddToQueue(events.ItemOwnership{
				UserId: actor.GetUserId(),
				Item:   plantItem,
				Gained: false,
			})
		} else {
			events.AddToQueue(events.ItemOwnership{
				MobInstanceId: actor.GetMobInstanceId(),
				Item:          plantItem,
				Gained:        false,
			})
		}

		actor.SendText(messaging.CategorySystem, fmt.Sprintf(
			`You deftly slip the <ansi fg="itemname">%s</ansi> into `+
				`<ansi fg="mobname">%s</ansi>'s belongings unnoticed.`,
			plantItem.DisplayName(), m.Character.Name))

		return PlantResult{
			Succeeded:     true,
			PlantedItemId: plantItem.ItemId,
			DefenderName:  m.Character.Name,
		}
	}

	// Failure — detected.
	actor.SendText(messaging.CategorySystem, fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> catches you in the act!`,
		m.Character.Name))

	if room != nil {
		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> gets caught trying to plant `+
					`something on <ansi fg="mobname">%s</ansi>!`,
				actor.GetName(), m.Character.Name),
			actor.GetUserId(),
		)
	}

	actor.GetCharacter().Awareness.TransitionToRevealing(state.TransitionReason{
		Trigger: awareness.TriggerSkullduggeryFailed,
	})

	// Record plant crime on faction-aligned victim.
	if factionIds := factions.FactionsForMob(m); len(factionIds) > 0 {
		// All witnesses including the victim (excludeInstanceId=0).
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		perp := crimes.IdentifiedPerp(actor.GetUserId(), witnesses)
		// External witnesses (excluding victim) for HadExternalWitness.
		externalWitnesses := crimes.WitnessesInRoom(factionIds, room,
			m.InstanceId)
		hadExternal := len(externalWitnesses) > 0
		delta := int(configs.GetBalanceConfig().CrimeRepDeltaTheft)
		for _, fid := range factionIds {
			crimeIds := crimes.Record([]string{fid}, crimes.KindTheft, perp,
				m, m.InstanceId, room.RoomId, m.Character.Zone, hadExternal)
			if perp.Type == crimes.PerpPlayer {
				factions.BumpRep(fid, actor.GetUserId(), delta)
				justice.MaybeDeclareBounty(fid, actor.GetUserId(), crimes.KindTheft)
				subject := knowledge.PlayerSubject(actor.GetUserId())
				for _, witnessInstId := range witnesses {
					w := mobs.GetInstance(witnessInstId)
					if w == nil {
						continue
					}
					for _, crimeId := range crimeIds {
						knowledge.RecordCrimeWitnessed(int(w.MobId), subject,
							crimeId)
					}
					knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
						knowledge.SourceWitnessed)
				}
			}
		}
	}

	m.Command(fmt.Sprintf(`attack @%d`, actor.GetUserId()))

	return PlantResult{
		Detected:     true,
		DefenderName: m.Character.Name,
		Reason:       "detected",
	}
}

// plantOnPlayer handles the mob-on-player plant path. The plant
// mechanic is symmetric with plantOnMob: the actor's Dex+skill
// score is rolled against the target player's Perception. On
// success, the item is slipped into the player's inventory. An
// independent detection roll then decides whether the victim notices.
func plantOnPlayer(actor Actor, targetUserId int, plantItem items.Item,
	attackerScore float64, rank int, cfg configs.Balance) PlantResult {

	targetUser := users.GetByUserId(targetUserId)
	if targetUser == nil {
		actor.SendText(messaging.CategorySystem, "They seem to have vanished.")
		return PlantResult{Reason: "target not found"}
	}

	// Skill-use and progression.
	actor.OnSkillUse(string(skills.Skullduggery))

	defenderScore := float64(targetUser.Character.Stats.Perception.ValueAdj)
	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	if !success {
		actor.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="username">%s</ansi> catches you in the act!`,
			targetUser.Character.Name))
		actor.GetCharacter().Awareness.TransitionToRevealing(state.TransitionReason{
			Trigger: awareness.TriggerSkullduggeryFailed,
		})
		return PlantResult{
			Detected:     true,
			DefenderName: targetUser.Character.Name,
			Reason:       "detected",
		}
	}

	// Success — transfer item to target player.
	actor.GetCharacter().RemoveItem(plantItem)
	targetUser.Character.StoreItem(plantItem)

	if actor.IsPlayer() {
		events.AddToQueue(events.ItemOwnership{
			UserId: actor.GetUserId(),
			Item:   plantItem,
			Gained: false,
		})
	} else {
		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: actor.GetMobInstanceId(),
			Item:          plantItem,
			Gained:        false,
		})
	}
	events.AddToQueue(events.ItemOwnership{
		UserId: targetUserId,
		Item:   plantItem,
		Gained: true,
	})

	result := PlantResult{
		Succeeded:     true,
		PlantedItemId: plantItem.ItemId,
		DefenderName:  targetUser.Character.Name,
	}

	// Independent detection roll: victim may notice even on success.
	if !actor.IsPlayer() {
		searchScore := CalcSearchScore(targetUser.Character)
		sneakScore := CalcSneakScoreVsObserver(actor.GetCharacter(), targetUser.Character, actor.GetRoom())
		detected, _, _, _ := dice.OpposedRollStat(searchScore, sneakScore)
		if detected {
			targetUser.SendText(messaging.CategorySystem, fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> slips something into your `+
					`belongings!`,
				actor.GetName()))
			result.Detected = true
		}
	}

	return result
}

// plantInContainer handles slipping an item into a room container.
func plantInContainer(actor Actor, containerName string, plantItem items.Item,
	attackerScore float64, rank int) PlantResult {

	room := actor.GetRoom()
	container, ok := room.Containers[containerName]
	if !ok {
		actor.SendText(messaging.CategorySystem, "You don't see that here.")
		return PlantResult{Reason: "not found"}
	}

	// Skill-use and progression — always fire regardless of roll outcome.
	actor.OnSkillUse(string(skills.Skullduggery))

	// Quest engine notification — player actors only.
	if actor.IsPlayer() {
		if u := users.GetByUserId(actor.GetUserId()); u != nil {
			bridge := questengine.NewGameBridge(u, room.RoomId)
			questengine.GetEngine().Notify("command", questengine.EventDetails{
				UserId:  actor.GetUserId(),
				RoomId:  room.RoomId,
				Command: "plant",
			}, bridge, bridge)
		}
	}

	// Find highest Perception observer (players + mobs, excluding party).
	partySet := map[int]bool{}
	if uid := actor.GetUserId(); uid > 0 {
		partySet[uid] = true
		if party := parties.Get(uid); party != nil {
			for _, memberId := range party.GetMembers() {
				partySet[memberId] = true
			}
		}
	}
	selfMobId := actor.GetMobInstanceId()

	highestPerception := 0.0
	spotterName := ""
	hasObserver := false

	for _, observerId := range room.GetPlayers() {
		if partySet[observerId] {
			continue
		}
		observer := users.GetByUserId(observerId)
		if observer == nil {
			continue
		}
		perScore := float64(observer.Character.Stats.Perception.ValueAdj)
		if perScore > highestPerception {
			highestPerception = perScore
			spotterName = observer.Character.Name
			hasObserver = true
		}
	}

	for _, mobInstanceId := range room.GetMobs() {
		if mobInstanceId == selfMobId {
			continue
		}
		m := mobs.GetInstance(mobInstanceId)
		if m == nil {
			continue
		}
		perScore := float64(m.Character.Stats.Perception.ValueAdj)
		if perScore > highestPerception {
			highestPerception = perScore
			spotterName = m.Character.Name
			hasObserver = true
		}
	}

	success := true
	if hasObserver {
		success, _, _, _ = dice.OpposedRollStat(attackerScore, highestPerception)
	}

	if !success {
		actor.SendText(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> spots you slipping something into `+
				`the <ansi fg="itemname">%s</ansi>!`,
			spotterName, containerName))

		room.SendTextVisual(messaging.CategoryMobEmote,
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is caught planting something `+
					`in the <ansi fg="itemname">%s</ansi>!`,
				actor.GetName(), containerName),
			actor.GetUserId(),
		)

		actor.GetCharacter().Awareness.TransitionToRevealing(state.TransitionReason{
			Trigger: awareness.TriggerSkullduggeryFailed,
		})
		return PlantResult{
			Detected:     true,
			DefenderName: spotterName,
			Reason:       "detected",
		}
	}

	// Success — place item into container.
	container.AddItem(plantItem)
	actor.GetCharacter().RemoveItem(plantItem)
	room.Containers[containerName] = container

	if actor.IsPlayer() {
		events.AddToQueue(events.ItemOwnership{
			UserId: actor.GetUserId(),
			Item:   plantItem,
			Gained: false,
		})
	} else {
		events.AddToQueue(events.ItemOwnership{
			MobInstanceId: actor.GetMobInstanceId(),
			Item:          plantItem,
			Gained:        false,
		})
	}

	actor.SendText(messaging.CategorySystem, fmt.Sprintf(
		`You quietly slip your <ansi fg="itemname">%s</ansi> into `+
			`the <ansi fg="itemname">%s</ansi>.`,
		plantItem.DisplayName(), containerName))

	return PlantResult{
		Succeeded:     true,
		PlantedItemId: plantItem.ItemId,
		DefenderName:  containerName,
	}
}
