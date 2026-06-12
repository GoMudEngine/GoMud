package behaviortree

// actions_archer.go — ranged-weapons T8 archer AI.
//
// Three behavior-tree actions that turn a ranged-weapon-carrying mob into a
// kiting archer:
//
//   try_fire      — fire a LOADED ranged weapon at the aggro target (same room,
//                   or one exit away via a directional shot).
//   try_reload    — chamber a fresh projectile when UNLOADED, ammo is on hand,
//                   and the mob is NOT pinned in melee (slip into cover first
//                   when capable of stealth).
//   keep_distance — when pinned in melee and still healthy, fall back through a
//                   passable exit so the next tick can shoot from range.
//
// These mirror the command-enqueue convention used elsewhere in the package:
// every "do X" is issued as a mob command string via mob.Command(...), which
// the engine routes through the normal input pipeline (here: the shoot/reload/
// go/sneak mob commands). Gating is read off live engine state — the mob's
// equipped weapon (items.Item.Loaded / IsRangedWeapon), its inventory ammo
// bundles, and its aggro target's current room.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func init() {
	actionRegistry["try_fire"] = actTryFire
	actionRegistry["try_reload"] = actTryReload
	actionRegistry["keep_distance"] = actKeepDistance
}

// loadedRangedWeapon returns the mob's equipped, LOADED ranged weapon (main
// hand first, then offhand), or nil. Mirrors actions.findRangedWeaponSlot's
// slot ordering, with the additional Loaded gate.
func loadedRangedWeapon(char *characters.Character) *items.Item {
	if char.Equipment.Weapon.IsRangedWeapon() && char.Equipment.Weapon.Loaded {
		return &char.Equipment.Weapon
	}
	if char.Equipment.Offhand.IsRangedWeapon() && char.Equipment.Offhand.Loaded {
		return &char.Equipment.Offhand
	}
	return nil
}

// unloadedRangedWeapon returns the mob's equipped but UNLOADED ranged weapon
// (main hand first, then offhand), or nil.
func unloadedRangedWeapon(char *characters.Character) *items.Item {
	if char.Equipment.Weapon.IsRangedWeapon() && !char.Equipment.Weapon.Loaded {
		return &char.Equipment.Weapon
	}
	if char.Equipment.Offhand.IsRangedWeapon() && !char.Equipment.Offhand.Loaded {
		return &char.Equipment.Offhand
	}
	return nil
}

// hasMatchingAmmo reports whether the character carries an ammo bundle whose
// AmmoTag matches the supplied weapon ammo tag. Mirrors the bundle-find loop
// in actions.ExecuteReload (the actual consume happens there).
func hasMatchingAmmo(char *characters.Character, ammoTag string) bool {
	for idx := range char.Items {
		spec := char.Items[idx].GetSpec()
		if spec.Type == items.Ammo && spec.AmmoTag == ammoTag {
			return true
		}
	}
	return false
}

// archerTarget resolves the mob's current aggro target into a display name and
// the room it is currently in. ok=false when there is no resolvable target.
func archerTarget(char *characters.Character) (name string, roomId int, ok bool) {
	if char.Aggro == nil {
		return "", 0, false
	}
	// Mob target before player target, matching actions.ResolveAggroTarget.
	if char.Aggro.MobInstanceId > 0 {
		if m := mobs.GetInstance(char.Aggro.MobInstanceId); m != nil {
			return m.Character.Name, m.Character.RoomId, true
		}
	}
	if char.Aggro.UserId > 0 {
		if u := users.GetByUserId(char.Aggro.UserId); u != nil {
			return u.Character.Name, u.Character.RoomId, true
		}
	}
	return "", 0, false
}

// archerMemoryTarget resolves the mob's CombatMemory into a display name and
// the room the remembered target is CURRENTLY in. It is the fallback for when
// the mob's live Aggro has been ended by the round driver's same-room validator
// after a kiting retreat — the CombatMemory survives EndAggro and lets the
// archer keep firing on the same foe. ok=false when there is no memory or the
// target can no longer be resolved (left the world / despawned) — the firing
// guard against shooting at someone who has left the area entirely.
//
// The target's LIVE room is preferred (it is authoritative); LastSeenRoomId is
// only consulted when the target object is resolvable but reports no room.
func archerMemoryTarget(mob *mobs.Mob) (name string, roomId int, ok bool) {
	mem := mob.CombatMemory
	if mem == nil {
		return "", 0, false
	}
	// Mob target before player target, matching archerTarget's ordering.
	if mem.TargetMobId > 0 {
		if m := mobs.GetInstance(mem.TargetMobId); m != nil {
			liveRoom := m.Character.RoomId
			if liveRoom == 0 {
				liveRoom = mem.LastSeenRoomId
			}
			return m.Character.Name, liveRoom, true
		}
		// Mob target has left the world (despawned / killed) — clear memory
		// immediately so the archer stops burning btree evals every round
		// until the 300-round expiry.
		mob.CombatMemory = nil
		return "", 0, false
	}
	if mem.TargetUserId > 0 {
		if u := users.GetByUserId(mem.TargetUserId); u != nil {
			liveRoom := u.Character.RoomId
			if liveRoom == 0 {
				liveRoom = mem.LastSeenRoomId
			}
			return u.Character.Name, liveRoom, true
		}
		// Player target has left the world (logged out / deleted) — clear
		// memory immediately for the same idle-CPU reason.
		mob.CombatMemory = nil
		return "", 0, false
	}
	return "", 0, false
}

// archerMeleeEngaged reports whether the mob is pinned in melee — defined as:
// its aggro target stands in the SAME room, OR some other actor in the room is
// aggroed onto this mob. This is the simplest correct "toe-to-toe" check the
// engine exposes without a dedicated melee-range concept: ranged combat is
// cross-room-capable, so "an enemy shares my room" is the cue that an archer
// has been closed on and should either kite (keep_distance) or hold its
// reload.
func archerMeleeEngaged(mob *mobs.Mob) bool {
	myRoom := mob.Character.RoomId

	// (1) My own aggro target is standing here.
	if _, targetRoomId, ok := archerTarget(&mob.Character); ok && targetRoomId == myRoom {
		return true
	}

	// (2) Someone in my room is targeting me.
	room := rooms.LoadRoom(myRoom)
	if room == nil {
		return false
	}
	for _, uid := range room.GetPlayers() {
		if u := users.GetByUserId(uid); u != nil && u.Character.Aggro != nil &&
			u.Character.Aggro.MobInstanceId == mob.InstanceId {
			return true
		}
	}
	for _, iid := range room.GetMobs() {
		if iid == mob.InstanceId {
			continue
		}
		if m := mobs.GetInstance(iid); m != nil && m.Character.Aggro != nil &&
			m.Character.Aggro.MobInstanceId == mob.InstanceId {
			return true
		}
	}
	return false
}

// hpPercent returns the mob's current health as a percentage of its max
// (0..100). A non-positive max is treated as full health.
func hpPercent(char *characters.Character) float64 {
	if char.HealthMax.Value <= 0 {
		return 100
	}
	return float64(char.Health) * 100 / float64(char.HealthMax.Value)
}

// actTryFire fires a loaded ranged weapon at the mob's aggro target. Same-room
// targets get an immediate `shoot <name>`; a target exactly one exit away gets
// a directional `shoot <name> <dir>` (the shoot mob command resolves the rest).
// Returns Failure when there is nothing loaded, no target, or the target is not
// reachable by a single shot — so the surrounding selector can try try_reload /
// the melee fallback instead.
//
// Target resolution prefers the LIVE aggro target, then falls back to
// CombatMemory: after keep_distance retreats the archer one exit, the round
// driver's ValidateAggro ends its aggro (the target is no longer in the room),
// so on the next tick the archer has NO aggro but a fresh CombatMemory pointing
// at the same foe. That fallback is what keeps the kite-and-shoot loop alive.
func actTryFire(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if loadedRangedWeapon(&mob.Character) == nil {
		return Failure // nothing chambered to fire
	}
	name, targetRoomId, ok := archerTarget(&mob.Character)
	if !ok {
		// Live aggro gone (ended by ValidateAggro after a retreat) — fall
		// back to the remembered target so the archer keeps firing on it.
		// archerMemoryTarget refuses when the target has left the world,
		// guarding against shooting at a foe that left the area entirely.
		name, targetRoomId, ok = archerMemoryTarget(mob)
		if !ok {
			return Failure
		}
	}

	// (a) Same-room target — fire straight away.
	if targetRoomId == mob.Character.RoomId {
		mob.Command("shoot " + name)
		return Success
	}

	// (b) Cross-room target — fire through the exit that leads to its room,
	// if it is exactly one exit away. A target more than one exit off is not
	// shootable this tick (and not pursued — bounded engagement window).
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	dir := room.FindExitTo(targetRoomId)
	if dir == "" {
		return Failure // not adjacent — can't line up a shot this tick
	}
	mob.Command(fmt.Sprintf("shoot %s %s", name, dir))
	return Success
}

// actTryReload chambers a fresh projectile. Gating: an UNLOADED ranged weapon,
// a matching ammo bundle on hand, and NOT being pinned in melee (reloading
// toe-to-toe is suicidal — keep_distance runs first to break contact). A
// skullduggery-capable archer that is not already hidden slips into cover
// (`sneak`) before reloading.
//
// NOTE: there is no mob `hide` command in this engine; `sneak` is the
// skullduggery stealth verb (internal/mobcommands/sneak.go) and serves the
// same "get into cover before reloading" intent the task describes.
func actTryReload(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	weapon := unloadedRangedWeapon(&mob.Character)
	if weapon == nil {
		return Failure // no ranged weapon, or it's already loaded
	}
	if !hasMatchingAmmo(&mob.Character, weapon.GetSpec().AmmoTag) {
		return Failure // out of ammo
	}
	if archerMeleeEngaged(mob) {
		return Failure // break contact first (keep_distance), don't reload in melee
	}

	// Slip into cover before reloading when capable and not already hidden.
	if mob.Character.GetSkillLevel(skills.Skullduggery) > 0 && !mob.Character.IsHidden() {
		mob.Command("sneak")
	}
	mob.Command("reload")
	return Success
}

// actKeepDistance kites the mob away from melee. It fires only when the mob is
// pinned in melee AND still healthy enough to choose flight over a desperate
// stand (health above the `health_percent` arg, default 50). It picks a
// passable exit — preferring the one toward home — and moves through it.
//
// Before moving it refreshes the mob's CombatMemory to point at the current
// aggro target, last seen in the room being retreated FROM (where the enemy
// still stands). This is the linchpin of the kite-and-shoot loop: stepping out
// of the room makes the next round's ValidateAggro end this mob's aggro, so the
// CombatMemory is the ONLY surviving handle on the foe. The round driver's
// ranged-mob exemption reads it to keep driving the btree, and try_fire reads
// it to fire back through the reverse exit.
func actKeepDistance(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	if !archerMeleeEngaged(mob) {
		return Failure // nothing closed on us — no need to kite
	}
	threshold := getFloatParam(params, "health_percent", 50)
	if hpPercent(&mob.Character) <= threshold {
		return Failure // too hurt to safely disengage — let the combat cascade handle it
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return Failure
	}
	dir := pickRetreatExit(mob, room)
	if dir == "" {
		return Failure // boxed in — stand and fight
	}

	// Refresh CombatMemory pointing at the current target, last seen in the
	// room we're retreating FROM, so the engagement survives the imminent
	// aggro loss (see doc comment above).
	if mob.Character.Aggro != nil {
		fromRoomId := mob.Character.RoomId
		round := util.GetRoundCount()
		if mob.CombatMemory == nil {
			mob.CombatMemory = mobs.SetCombatMemory(
				mob.Character.Aggro.UserId,
				mob.Character.Aggro.MobInstanceId,
				fromRoomId,
				round,
			)
		} else {
			mob.CombatMemory.TargetUserId = mob.Character.Aggro.UserId
			mob.CombatMemory.TargetMobId = mob.Character.Aggro.MobInstanceId
			mob.CombatMemory.Grudge = true
			mobs.UpdateCombatMemoryLocation(mob.CombatMemory, fromRoomId, round)
		}
	}

	mob.Command("go " + dir)
	return Success
}

// pickRetreatExit chooses a passable (non-locked) exit to fall back through,
// preferring the exit that leads toward the mob's home room when one is
// directly available, else the first passable exit encountered. Returns ""
// when every exit is locked (boxed in).
func pickRetreatExit(mob *mobs.Mob, room *rooms.Room) string {
	homeId := mob.HomeRoomId
	fallback := ""
	for dir, info := range room.Exits {
		if info.Lock.IsLocked() {
			continue
		}
		if homeId > 0 && info.RoomId == homeId {
			return dir // prefer heading home
		}
		if fallback == "" {
			fallback = dir
		}
	}
	return fallback
}
