package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// conversationMobAdapter wraps *mobs.Mob to satisfy conversations.MobConversant
// without forcing the conversations package to import mobs (which would close
// an import cycle since mobs imports conversations for Load()).
type conversationMobAdapter struct {
	mob *mobs.Mob
}

// adaptMob returns a conversations.MobConversant view of the given mob,
// or nil if mob is nil.
func adaptMob(m *mobs.Mob) conversations.MobConversant {
	if m == nil {
		return nil
	}
	return &conversationMobAdapter{mob: m}
}

func (a *conversationMobAdapter) ConvInstanceId() int { return a.mob.InstanceId }
func (a *conversationMobAdapter) ConvMobId() int      { return int(a.mob.MobId) }
func (a *conversationMobAdapter) ConvRoomId() int     { return a.mob.Character.RoomId }

func (a *conversationMobAdapter) ConvGetMiscData(key string) any {
	return a.mob.Character.GetMiscData(key)
}

func (a *conversationMobAdapter) ConvSetMiscData(key string, val any) {
	a.mob.Character.SetMiscData(key, val)
}

func (a *conversationMobAdapter) ConvIsInCombat() bool { return a.mob.Character.IsInCombat() }

func (a *conversationMobAdapter) ConvHasBuffFlag(f buffs.Flag) bool {
	return a.mob.Character.HasBuffFlag(f)
}

// ConvAggro returns true when the mob has an active aggro target.
// mob.Character.Aggro is a *characters.Aggro; non-nil means combat is pending.
func (a *conversationMobAdapter) ConvAggro() bool { return a.mob.Character.Aggro != nil }

func (a *conversationMobAdapter) ConvPathLen() int            { return a.mob.Path.Len() }
func (a *conversationMobAdapter) ConvPathCurrentNonNil() bool { return a.mob.Path.Current() != nil }

// ConvCommand fires a mob command (fire-and-forget; return value discarded).
func (a *conversationMobAdapter) ConvCommand(text string) { a.mob.Command(text) }

// ConvGetPartner looks up another mob by instance id and returns it as a
// MobConversant, or nil if not found.
func (a *conversationMobAdapter) ConvGetPartner(instanceId int) conversations.MobConversant {
	other := mobs.GetInstance(instanceId)
	return adaptMob(other)
}
