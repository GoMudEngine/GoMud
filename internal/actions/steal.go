package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/dice"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// StealOptions parameterizes a theft attempt.
// Exactly one of TargetMobInstanceId / TargetUserId / ContainerNoun
// must be set. ItemNoun narrows the steal to a specific item;
// when empty, the action defaults to gold-or-random-item per the
// existing player-side logic.
type StealOptions struct {
	TargetMobInstanceId int
	TargetUserId        int
	ContainerNoun       string
	ItemNoun            string
}

// StealResult is the structured outcome of a steal attempt.
type StealResult struct {
	Succeeded     bool   // skill check passed and transfer happened
	Detected      bool   // detection roll fired and the defender noticed
	StoleGold     int    // gold transferred (0 if item-only or failed)
	StoleItemId   int    // item id transferred (0 if gold-only or failed)
	StoleItemName string // for messaging
	DefenderName  string // who/what was robbed
	OnCooldown    bool   // attempt was blocked by skullduggery cooldown
	Reason        string // when Succeeded==false and !OnCooldown, why
}

// Steal runs a skullduggery theft attempt from actor against the
// resolved target. Both UserActor and MobActor are supported.
func Steal(actor Actor, opts StealOptions) StealResult {
	char := actor.GetCharacter()

	// Combat gate — can't steal while actively fighting.
	if char.Aggro != nil {
		actor.SendText("You can't do that while in combat!")
		return StealResult{Reason: "in combat"}
	}

	room := actor.GetRoom()
	if room == nil {
		return StealResult{Reason: "no room"}
	}

	// Under-attack gate — mobs with userId=0 short-circuit this harmlessly.
	if room.AreMobsAttacking(actor.GetUserId()) {
		actor.SendText("You can't do that while you are under attack!")
		return StealResult{Reason: "under attack"}
	}

	// Require a target.
	if opts.TargetMobInstanceId == 0 && opts.TargetUserId == 0 &&
		opts.ContainerNoun == "" {
		actor.SendText("Steal from whom?")
		return StealResult{Reason: "no target"}
	}

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`steal`)

	// Check cooldown before doing target resolution.
	if !char.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d real seconds`, int(cfg.StealCooldown))) {
		return StealResult{
			OnCooldown: true,
			Reason: fmt.Sprintf("%d rounds remaining",
				char.GetCooldown(cooldownKey)),
		}
	}

	isHidden := char.HasBuffFlag(buffs.Hidden)

	// Compute attacker score.
	rank := char.GetSkillLevel(skills.Skullduggery)
	base := float64(char.Stats.Dexterity.ValueAdj) +
		combat.SkillMultiplier(rank)*25.0
	attackerScore := base * float64(cfg.StealSkillMultiplier)
	if isHidden {
		attackerScore += float64(cfg.StealHiddenBonus)
	}

	// Dispatch to the appropriate path.
	if opts.TargetMobInstanceId > 0 {
		return stealFromMob(actor, opts.TargetMobInstanceId, attackerScore, rank, cfg)
	}

	if opts.TargetUserId > 0 {
		return stealFromPlayer(actor, opts.TargetUserId, attackerScore, rank, cfg)
	}

	// Container path.
	return stealFromContainer(actor, opts.ContainerNoun, attackerScore, rank)
}

// stealFromMob handles the creature steal path.
func stealFromMob(actor Actor, mobInstanceId int, attackerScore float64,
	rank int, cfg configs.Balance) StealResult {

	m := mobs.GetInstance(mobInstanceId)
	if m == nil {
		actor.SendText("They seem to have vanished.")
		return StealResult{Reason: "target not found"}
	}

	if m.IsNonCombatant() || m.PlayerAttackImmune {
		actor.SendText(fmt.Sprintf(
			`You can't steal from <ansi fg="mobname">%s</ansi>.`,
			m.Character.Name))
		return StealResult{
			DefenderName: m.Character.Name,
			Reason:       "immune",
		}
	}

	// Skill rank 2 required for the actual steal mechanic. Checked
	// AFTER the immune gate so a low-rank thief targeting an immune mob
	// gets the immune rebuff (which doesn't imply skill would help)
	// rather than the skill rebuff (which does).
	if rank < 2 {
		actor.SendText("You aren't advanced enough at skullduggery for that.")
		return StealResult{
			DefenderName: m.Character.Name,
			Reason:       "not advanced enough",
		}
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
				Command: "steal",
			}, bridge, bridge)
		}
	}

	defenderScore := float64(m.Character.Stats.Perception.ValueAdj)
	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	room := actor.GetRoom()

	if success {
		result := StealResult{
			Succeeded:    true,
			DefenderName: m.Character.Name,
		}

		stolenStuff := []string{}

		if m.Character.Gold > 0 {
			quarterGold := m.Character.Gold >> 2
			minGold := m.Character.Gold - quarterGold
			goldStolen := util.Rand(quarterGold) + minGold
			if goldStolen > 0 {
				m.Character.Gold -= goldStolen
				actor.GetCharacter().Gold += goldStolen
				result.StoleGold = goldStolen
				stolenStuff = append(stolenStuff,
					fmt.Sprintf(`<ansi fg="yellow-bold">%d gold</ansi>`, goldStolen))

				if actor.IsPlayer() {
					events.AddToQueue(events.EquipmentChange{
						UserId:     actor.GetUserId(),
						GoldChange: goldStolen,
					})
				}
			}
		}

		if itemStolen, found := m.Character.GetRandomItem(); found {
			m.Character.RemoveItem(itemStolen)
			actor.GetCharacter().StoreItem(itemStolen)
			result.StoleItemId = itemStolen.ItemId
			result.StoleItemName = itemStolen.DisplayName()

			events.AddToQueue(events.ItemOwnership{
				MobInstanceId: m.InstanceId,
				Item:          itemStolen,
				Gained:        false,
			})

			if actor.IsPlayer() {
				events.AddToQueue(events.ItemOwnership{
					UserId: actor.GetUserId(),
					Item:   itemStolen,
					Gained: true,
				})
			} else {
				events.AddToQueue(events.ItemOwnership{
					MobInstanceId: actor.GetMobInstanceId(),
					Item:          itemStolen,
					Gained:        true,
				})
			}

			stolenStuff = append(stolenStuff,
				fmt.Sprintf(`<ansi fg="itemname">%s</ansi>`, itemStolen.DisplayName()))
		}

		if len(stolenStuff) == 0 {
			actor.SendText(fmt.Sprintf(
				`You deftly rifle through <ansi fg="mobname">%s</ansi>'s `+
					`belongings but find nothing worth taking.`,
				m.Character.Name))
		} else {
			actor.SendText(fmt.Sprintf(
				`You successfully steal %s from <ansi fg="mobname">%s</ansi>.`,
				strings.Join(stolenStuff, ` and `), m.Character.Name))
		}

		return result
	}

	// Failure — detected.
	actor.SendText(fmt.Sprintf(
		`<ansi fg="mobname">%s</ansi> catches you in the act!`,
		m.Character.Name))

	if room != nil {
		room.SendTextVisual(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> gets caught trying to steal `+
					`from <ansi fg="mobname">%s</ansi>!`,
				actor.GetName(), m.Character.Name),
			actor.GetUserId(),
		)
	}

	actor.GetCharacter().CancelBuffsWithFlag(buffs.Hidden)

	// chunk 1.3: record theft crime on faction-aligned victim.
	if factionIds := factions.FactionsForMob(m); len(factionIds) > 0 {
		// All witnesses including the victim (excludeInstanceId=0).
		witnesses := crimes.WitnessesInRoom(factionIds, room, 0)
		perp := crimes.IdentifiedPerp(actor.GetUserId(), witnesses)
		// External witnesses (excluding victim) for HadExternalWitness.
		externalWitnesses := crimes.WitnessesInRoom(factionIds, room, m.InstanceId)
		hadExternal := len(externalWitnesses) > 0
		delta := int(configs.GetBalanceConfig().CrimeRepDeltaTheft)
		for _, fid := range factionIds {
			crimeIds := crimes.Record([]string{fid}, crimes.KindTheft, perp,
				m, m.InstanceId, room.RoomId, m.Character.Zone, hadExternal)
			if perp.Type == crimes.PerpPlayer {
				factions.BumpRep(fid, actor.GetUserId(), delta)
				// Knowledge: each witness records the player as the perp of
				// these crimes.
				subject := knowledge.PlayerSubject(actor.GetUserId())
				for _, witnessInstId := range witnesses {
					w := mobs.GetInstance(witnessInstId)
					if w == nil {
						continue
					}
					for _, crimeId := range crimeIds {
						knowledge.RecordCrimeWitnessed(int(w.MobId), subject, crimeId)
					}
					knowledge.RecordMet(int(w.MobId), subject, room.RoomId,
						knowledge.SourceWitnessed)
				}
			}
		}
	}

	m.Command(fmt.Sprintf(`attack @%d`, actor.GetUserId()))

	return StealResult{
		Detected:     true,
		DefenderName: m.Character.Name,
		Reason:       "detected",
	}
}

// stealFromPlayer handles the mob-on-player theft path. The steal
// mechanic is symmetric with stealFromMob: the actor's Dex+skill
// score is rolled against the target player's Perception. On
// success, gold is lifted (no item steal against players). An
// independent detection roll then decides whether the victim
// notices.
func stealFromPlayer(actor Actor, targetUserId int, attackerScore float64,
	rank int, cfg configs.Balance) StealResult {

	targetUser := users.GetByUserId(targetUserId)
	if targetUser == nil {
		actor.SendText("They seem to have vanished.")
		return StealResult{Reason: "target not found"}
	}

	// Skill rank 2 required.
	if rank < 2 {
		actor.SendText("You aren't advanced enough at skullduggery for that.")
		return StealResult{
			DefenderName: targetUser.Character.Name,
			Reason:       "not advanced enough",
		}
	}

	// Skill-use and progression.
	actor.OnSkillUse(string(skills.Skullduggery))

	defenderScore := float64(targetUser.Character.Stats.Perception.ValueAdj)
	success, _, _, _ := dice.OpposedRollStat(attackerScore, defenderScore)

	if !success {
		actor.SendText(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> catches you in the act!`,
			targetUser.Character.Name))
		actor.GetCharacter().CancelBuffsWithFlag(buffs.Hidden)
		return StealResult{
			Detected:     true,
			DefenderName: targetUser.Character.Name,
			Reason:       "detected",
		}
	}

	result := StealResult{
		Succeeded:    true,
		DefenderName: targetUser.Character.Name,
	}

	if targetUser.Character.Gold > 0 {
		quarterGold := targetUser.Character.Gold >> 2
		minGold := targetUser.Character.Gold - quarterGold
		goldStolen := util.Rand(quarterGold) + minGold
		if goldStolen > 0 {
			targetUser.Character.Gold -= goldStolen
			actor.GetCharacter().Gold += goldStolen
			result.StoleGold = goldStolen

			if actor.IsPlayer() {
				events.AddToQueue(events.EquipmentChange{
					UserId:     actor.GetUserId(),
					GoldChange: goldStolen,
				})
			}
			events.AddToQueue(events.EquipmentChange{
				UserId:     targetUserId,
				GoldChange: -goldStolen,
			})
		}
	}

	// Independent detection roll: victim may notice even on success.
	if !actor.IsPlayer() {
		searchScore := CalcSearchScore(targetUser.Character)
		sneakScore := CalcSneakScore(actor.GetCharacter())
		detected, _, _, _ := dice.OpposedRollStat(searchScore, sneakScore)
		if detected {
			targetUser.SendText(fmt.Sprintf(
				`<ansi fg="mobname">%s</ansi> lifts `+
					`<ansi fg="gold">%d gold</ansi> from your pocket!`,
				actor.GetName(), result.StoleGold))
			result.Detected = true
		}
	}

	return result
}

// stealFromContainer handles the room-container steal path.
func stealFromContainer(actor Actor, containerName string,
	attackerScore float64, rank int) StealResult {

	room := actor.GetRoom()
	container, ok := room.Containers[containerName]
	if !ok {
		actor.SendText("You don't see that here.")
		return StealResult{Reason: "not found"}
	}

	// Skill rank 2 required for the actual steal mechanic. Mirrors the
	// gate in stealFromMob; checked AFTER target validation so the rebuff
	// order is consistent.
	if rank < 2 {
		actor.SendText("You aren't advanced enough at skullduggery for that.")
		return StealResult{Reason: "not advanced enough"}
	}

	if len(container.Items) == 0 && container.Gold == 0 {
		actor.SendText(fmt.Sprintf(
			`You root around inside the <ansi fg="itemname">%s</ansi> `+
				`but find it empty.`,
			containerName))
		return StealResult{Reason: "empty"}
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
				Command: "steal",
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
		actor.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> spots you reaching into the `+
				`<ansi fg="itemname">%s</ansi>!`,
			spotterName, containerName))

		room.SendTextVisual(
			fmt.Sprintf(
				`<ansi fg="username">%s</ansi> is caught stealing from `+
					`the <ansi fg="itemname">%s</ansi>!`,
				actor.GetName(), containerName),
			actor.GetUserId(),
		)

		actor.GetCharacter().CancelBuffsWithFlag(buffs.Hidden)
		return StealResult{
			Detected:     true,
			DefenderName: spotterName,
			Reason:       "detected",
		}
	}

	// Success — take a random item (or gold) from the container.
	result := StealResult{
		Succeeded: true,
	}

	if len(container.Items) > 0 {
		idx := util.Rand(len(container.Items))
		stolen := container.Items[idx]
		container.RemoveItem(stolen)
		room.Containers[containerName] = container
		actor.GetCharacter().StoreItem(stolen)
		result.StoleItemId = stolen.ItemId
		result.StoleItemName = stolen.DisplayName()

		if actor.IsPlayer() {
			events.AddToQueue(events.ItemOwnership{
				UserId: actor.GetUserId(),
				Item:   stolen,
				Gained: true,
			})
		} else {
			events.AddToQueue(events.ItemOwnership{
				MobInstanceId: actor.GetMobInstanceId(),
				Item:          stolen,
				Gained:        true,
			})
		}

		actor.SendText(fmt.Sprintf(
			`You quietly slip <ansi fg="itemname">%s</ansi> from the `+
				`<ansi fg="itemname">%s</ansi>.`,
			stolen.DisplayName(), containerName))
	} else {
		// Container only has gold.
		quarterGold := container.Gold >> 2
		minGold := container.Gold - quarterGold
		goldStolen := util.Rand(quarterGold) + minGold
		if goldStolen > 0 {
			container.Gold -= goldStolen
			room.Containers[containerName] = container
			actor.GetCharacter().Gold += goldStolen
			result.StoleGold = goldStolen

			if actor.IsPlayer() {
				events.AddToQueue(events.EquipmentChange{
					UserId:     actor.GetUserId(),
					GoldChange: goldStolen,
				})
			}

			actor.SendText(fmt.Sprintf(
				`You quietly lift <ansi fg="yellow-bold">%d gold</ansi> `+
					`from the <ansi fg="itemname">%s</ansi>.`,
				goldStolen, containerName))
		}
	}

	return result
}
