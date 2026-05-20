package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/life"
)

func Suicide(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// ReviveOnDeath buff: heal and clear, no death — unless rest ==
	// "vanish" which forces unconditional despawn.
	if rest != `vanish` && mob.Character.HasBuffFlag(buffs.ReviveOnDeath) {
		mob.Character.Health = mob.Character.HealthMax.Value
		room.SendTextVisualLegacy(
			`<ansi fg="mobname">` + mob.Character.Name + `</ansi> is suddenly revived in a shower of sparks!`,
		)
		mob.Character.CancelBuffsWithFlag(buffs.ReviveOnDeath)
		return true, nil
	}

	mudlog.Debug(`Mob Death`, `name`, mob.Character.Name, `rest`, rest)

	// Vanish: bypass the Life machine and observer chain entirely.
	// This is an admin-style forced despawn — no loot, no corpse,
	// no kill credit, no broadcasts.
	if rest == `vanish` {
		mobs.DeleteMobInstance(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)
		mobs.DestroyInstance(mob.InstanceId)
		if r := rooms.LoadRoom(mob.HomeRoomId); r != nil {
			r.CleanupMobSpawns(false)
		}
		room.RemoveMob(mob.InstanceId)
		return true, nil
	}

	// Standard death: route through the Life machine.
	// All downstream work (broadcast, behavior-tree mob_die, kill
	// credit, charm cleanup, loot drop, corpse, MobDeath event, and
	// instance cleanup) fires via AfterTransition observers wired in:
	//   Death_MobBroadcast.go
	//   Death_MobBehaviorTree.go
	//   Death_MobKillCredit.go
	//   Death_MobCharmCleanup.go
	//   Death_MobLoot.go           (T7)
	//   Death_AlivenessSubstrate.go (T7)
	//   Death_MobInstanceCleanup.go (T7)
	//
	// Pick the first entry in PlayerDamage as the nominal killer for
	// attribution (matches historical suicide.go behaviour).
	var killer state.ActorRef
	for uid := range mob.Character.PlayerDamage {
		killer.UserId = uid
		break
	}
	mob.Character.Die(killer, life.TriggerHealthZero)

	return true, nil
}
