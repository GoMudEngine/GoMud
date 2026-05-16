package hooks

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func DoCombat(e events.Event) events.ListenerReturn {

	evt := e.(events.NewRound)

	//
	// Combat rounds
	//
	affectedPlayers1, affectedMobs1 := handlePlayerCombat(evt)

	affectedPlayers2, affectedMobs2 := handleMobCombat(evt)

	// Do any resolution or extra checks based on everyone that has been involved in combat this round.
	handleAffected(append(affectedPlayers1, affectedPlayers2...), append(affectedMobs1, affectedMobs2...))

	// Post-combat retarget pass: players with no aggro who have mobs
	// attacking them (or their companions) should pick up a target.
	// This catches cases where mobs initiated combat during handleMobCombat
	// but the player's retarget scan in handlePlayerCombat ran too early.
	for _, userId := range users.GetOnlineUserIds() {
		user := users.GetByUserId(userId)
		if user == nil || user.Character.Aggro != nil {
			continue
		}
		uRoom := rooms.LoadRoom(user.Character.RoomId)
		if uRoom == nil {
			continue
		}
		if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
			targetName := "something"
			if user.Character.Aggro.MobInstanceId > 0 {
				if m := mobs.GetInstance(user.Character.Aggro.MobInstanceId); m != nil {
					targetName = m.Character.Name
				}
			} else if user.Character.Aggro.UserId > 0 {
				if u := users.GetByUserId(user.Character.Aggro.UserId); u != nil {
					targetName = u.Character.Name
				}
			}
			user.SendText(fmt.Sprintf("You shift your focus to <ansi fg=\"mobname\">%s</ansi>!", targetName))
		}
	}

	return events.Continue
}

func handlePlayerCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	moonMod := float64(configs.GetBalanceConfig().MoonStatModMax)

	tStart := time.Now()

	for _, userId := range users.GetOnlineUserIds() {

		user := users.GetByUserId(userId)

		if user == nil {
			continue
		}

		if user.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		// Task 15: tick the Combat Phase machine every round for every
		// player. This advances Engaging→Engaged countdowns and fires
		// registered DispatchTickEvent listeners.
		if user.Character.CombatPhase != nil {
			user.Character.CombatPhase.OnRoundTick()
			user.Character.CombatPhase.DispatchTickEvent()
		}

		handlePlayerShieldDecay(user)

		if handlePlayerFoldCasting(user, userId) {
			continue
		}

		if user.Character.Aggro == nil {
			continue
		}

		// Validate aggro target still exists and is alive; retarget if possible
		if !ValidateAggro(user.Character) {
			uRoom := rooms.LoadRoom(user.Character.RoomId)
			if uRoom != nil {
				if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
					if mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); mob != nil {
						user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", mob.Character.Name))
					} else if defUser := users.GetByUserId(user.Character.Aggro.UserId); defUser != nil {
						user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", defUser.Character.Name))
					}
				}
			}
			if user.Character.Aggro == nil {
				continue
			}
		}

		user.Character.CancelCombatBuffs()

		uRoom := rooms.LoadRoom(user.Character.RoomId)
		if uRoom == nil {
			continue
		}

		if handlePlayerFlee(user, uRoom, userId) {
			continue
		}

		// Unified combat dispatch (replaces PvP/PvM branch).
		if user.Character.Aggro != nil {
			var def actions.Actor
			if user.Character.Aggro.UserId > 0 {
				if defUser := users.GetByUserId(user.Character.Aggro.UserId); defUser != nil {
					defRoom := rooms.LoadRoom(defUser.Character.RoomId)
					def = actions.NewUserActorInRoom(defUser, defRoom)
				}
			} else if user.Character.Aggro.MobInstanceId > 0 {
				if defMob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); defMob != nil {
					defRoom := rooms.LoadRoom(defMob.Character.RoomId)
					def = actions.NewMobActorInRoom(defMob, defRoom)
				}
			}
			if def != nil {
				atk := actions.NewUserActorInRoom(user, uRoom)
				cfg := configs.GetConfig()
				handleCombatRound(atk, def, evt, moonMod, &cfg, &affectedPlayerIds, &affectedMobInstanceIds)
			}
		}
	}

	util.TrackTime(`DoCombat::handlePlayerCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

func handleMobCombat(evt events.NewRound) (affectedPlayerIds []int, affectedMobInstanceIds []int) {

	moonMod := float64(configs.GetBalanceConfig().MoonStatModMax)
	tStart := time.Now()
	activeZones := rooms.SnapshotActiveZones()

	// Sweep: kill any mob stuck at 0 HP from a previous round (e.g., DOT
	// damage, dismissed companion beaten down, or missed death check).
	// Sweep runs for every mob regardless of zone activity — dead mobs
	// should not linger even in idle zones.
	for _, mobId := range mobs.GetAllMobInstanceIds() {
		if mob := mobs.GetInstance(mobId); mob != nil && mob.Character.Health <= 0 && mob.Character.IsAlive() {
			// No killer attribution available in the sweep pass — the
			// mob died from a prior-round effect and the attacker is
			// no longer recoverable from context here.
			mob.Character.Die(state.ActorRef{}, life.TriggerHealthZero)
		}
	}

	for _, mobId := range mobs.GetAllMobInstanceIds() {

		mob := mobs.GetInstance(mobId)

		if mob == nil || mob.Character.Health <= 0 {
			continue
		}

		if mob.Character.HasBuffFlag(buffs.NoCombat) {
			continue
		}

		// Non-combatant mobs (merchants, quest NPCs) should never fight.
		// If they somehow got aggro (e.g., from pack scatter), clear it.
		if mob.IsNonCombatant() {
			if mob.Character.Aggro != nil {
				mob.Character.EndAggro()
			}
			continue
		}

		// Task 15: tick the Combat Phase machine every round for every mob.
		// Advances Engaging→Engaged countdowns and fires tick event listeners.
		if mob.Character.CombatPhase != nil {
			mob.Character.CombatPhase.OnRoundTick()
			mob.Character.CombatPhase.DispatchTickEvent()
		}

		mobRoom := rooms.LoadRoom(mob.Character.RoomId)
		if mobRoom == nil {
			if mob.Character.Aggro != nil {
				mob.Character.EndAggro()
			}
			continue
		}

		// Idle zone → skip combat entirely. Mobs "in combat" whose last
		// opponent left the zone sit frozen until combat-memory expiry
		// (handled in the idle lane of MobRoundTick) clears the flag.
		if !activeZones[mobRoom.Zone] {
			continue
		}

		// Only run the full combat prep (buff stripping, shield decay, etc.) when
		// actually fighting or when the mob might enter combat this round.
		if mob.Character.Aggro != nil {
			// Strip combat-cancelling buffs (Hidden, etc.) and remove
			// their permabuff entries so Validate() doesn't re-apply them.
			mob.Character.CancelCombatBuffs()

			// Mob shield decay (symmetric with handlePlayerShieldDecay)
			if mob.Character.HasCondition(characters.ConditionShield) {
				if mob.Character.GetConditionDuration(characters.ConditionShield) <= 1 {
					mob.Character.RemoveCondition(characters.ConditionShield)
					mobRoom.SendText(fmt.Sprintf(
						`<ansi fg="blue"><ansi fg="mobname">%s</ansi>'s Minor Shield dissipates.</ansi>`,
						mob.Character.Name))
				} else {
					mob.Character.DecrementCondition(characters.ConditionShield)
				}
			}

			if handleMobFoldCasting(mob, mobRoom) {
				continue
			}

			// Validate mob's aggro target still exists and is alive; retarget if possible
			if !ValidateAggro(&mob.Character) {
				if mobRoom != nil {
					RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
				}
			}
		}

		// Idle companions with autoassist scan for threats to owner
		if mob.Character.Aggro == nil && mob.Character.IsCharmed() {
			CompanionAutoTarget(mob, mobRoom)
			if mob.Character.Aggro == nil {
				continue
			}
		} else if mob.Character.Aggro == nil {
			continue
		}

		// Initialize CombatMemory on the first round of engagement so
		// the mob can track its aggro target across flee/re-engage cycles.
		if mob.CombatMemory == nil && mob.Character.Aggro != nil {
			mob.CombatMemory = mobs.SetCombatMemory(
				mob.Character.Aggro.UserId,
				mob.Character.Aggro.MobInstanceId,
				mob.Character.RoomId,
				evt.RoundNumber,
			)
		}

		// Fire mob_combat_round for the attacking mob BEFORE the legacy
		// handleMobAIDecision. Mobs with a matching btree (per-mob file or
		// archetype via BehaviorArchetype) get first shot at this round's
		// action; if the tree returns Success (e.g., initiated a cast) we
		// skip both the legacy AI and handleCombatRound for this mob.
		//
		// Legacy preferredSpell has a hardcoded priority (shield → heal →
		// harm-list) that would otherwise preempt archetype self-buffs every
		// round. Firing here makes the archetype authoritative.
		btCtx := behaviortree.EventContext{
			EventType: "mob_combat_round",
			RoomId:    mob.Character.RoomId,
		}
		if behaviortree.TryMobBehavior(mob.InstanceId, btCtx) {
			continue
		}

		c := configs.GetConfig()
		if handleMobAIDecision(mob, c) {
			continue
		}

		affectedMobInstanceIds = append(affectedMobInstanceIds, mob.InstanceId)

		// Unified combat dispatch (replaces MvP/MvM branch).
		if mob.Character.Aggro != nil {
			var def actions.Actor
			if mob.Character.Aggro.UserId > 0 {
				if defUser := users.GetByUserId(mob.Character.Aggro.UserId); defUser != nil {
					defRoom := rooms.LoadRoom(defUser.Character.RoomId)
					def = actions.NewUserActorInRoom(defUser, defRoom)
				}
			} else if mob.Character.Aggro.MobInstanceId > 0 {
				if defMob := mobs.GetInstance(mob.Character.Aggro.MobInstanceId); defMob != nil {
					defRoom := rooms.LoadRoom(defMob.Character.RoomId)
					def = actions.NewMobActorInRoom(defMob, defRoom)
				}
			}
			if def != nil {
				atk := actions.NewMobActorInRoom(mob, mobRoom)
				cfg := configs.GetConfig()
				handleCombatRound(atk, def, evt, moonMod, &cfg, &affectedPlayerIds, &affectedMobInstanceIds)
			}
		}
	}

	util.TrackTime(`World::handleMobCombat()`, time.Since(tStart).Seconds())

	return affectedPlayerIds, affectedMobInstanceIds
}

// processGrappleProgression was the chunk-2-era binary single-roll
// progression scanner: every round, take a clinched pair to Grounded
// or break it. Chunk 4b replaces it with the per-round drift tick in
// Position_GrappleTick.go (T6) — graduated multi-round ControlLevel
// shifts that fire threshold-triggered transitions when either side
// hits Controlled. Deleted here in T10.

func handleAffected(affectedPlayerIds []int, affectedMobInstanceIds []int) {

	playersHandled := map[int]struct{}{}
	for _, userId := range affectedPlayerIds {
		if _, ok := playersHandled[userId]; ok {
			continue
		}
		playersHandled[userId] = struct{}{}

		if user := users.GetByUserId(userId); user != nil {
			if user.Character.Health < 1 && user.Character.IsAlive() {
				// Death on zero: route through Life cascade for same-tick
				// observer firing (teleport, stat decay, loot, etc.).
				// Killer attribution: the attacker is tracked via
				// PlayerDamage (snapshotted inside Die).
				user.Character.Die(state.ActorRef{}, life.TriggerHealthZero)
			}
		}
	}

	mobsHandled := map[int]struct{}{}
	for _, mobId := range affectedMobInstanceIds {
		if _, ok := mobsHandled[mobId]; ok {
			continue
		}
		mobsHandled[mobId] = struct{}{}

		if mob := mobs.GetInstance(mobId); mob != nil {
			if mob.Character.Health < 1 && mob.Character.IsAlive() {
				mob.Character.Die(state.ActorRef{}, life.TriggerHealthZero)
			}
		}

	}

}

// applyMoonMods temporarily adds moon phase stat deltas to a mutated character's
// six combat stats (DEX/STR via Swiftmoon, VIT/WIL via Wanderer, PER/CHA via The Eye).
// Returns a no-op restore function when the character has no mutations or moonMod is zero.
// Always call the returned function immediately after the combat roll to undo the changes.
func applyMoonMods(ch *characters.Character, moonMod float64) func() {
	if len(ch.Mutations) == 0 || moonMod == 0 {
		return func() {}
	}
	swift, wander, eye := gametime.GetAllPhases()
	dex := gametime.MoonStatDelta(swift, moonMod, ch.Stats.Dexterity.ValueAdj)
	str := gametime.MoonStatDelta(swift, moonMod, ch.Stats.Strength.ValueAdj)
	vit := gametime.MoonStatDelta(wander, moonMod, ch.Stats.Vitality.ValueAdj)
	wil := gametime.MoonStatDelta(wander, moonMod, ch.Stats.Willpower.ValueAdj)
	per := gametime.MoonStatDelta(eye, moonMod, ch.Stats.Perception.ValueAdj)
	cha := gametime.MoonStatDelta(eye, moonMod, ch.Stats.Charisma.ValueAdj)
	ch.Stats.Dexterity.ValueAdj += dex
	ch.Stats.Strength.ValueAdj += str
	ch.Stats.Vitality.ValueAdj += vit
	ch.Stats.Willpower.ValueAdj += wil
	ch.Stats.Perception.ValueAdj += per
	ch.Stats.Charisma.ValueAdj += cha
	return func() {
		ch.Stats.Dexterity.ValueAdj -= dex
		ch.Stats.Strength.ValueAdj -= str
		ch.Stats.Vitality.ValueAdj -= vit
		ch.Stats.Willpower.ValueAdj -= wil
		ch.Stats.Perception.ValueAdj -= per
		ch.Stats.Charisma.ValueAdj -= cha
	}
}
