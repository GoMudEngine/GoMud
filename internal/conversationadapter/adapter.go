// Package conversationadapter bridges *mobs.Mob to
// conversations.MobConversant. It lives in its own package so both
// internal/hooks/ and internal/usercommands/ (and any other callers)
// can use it without code duplication.
//
// Import-cycle note: conversations doesn't import mobs; mobs imports
// conversations for Load(). This adapter package depends on both —
// which is fine because it sits above both in the dependency graph.
package conversationadapter

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

type mobAdapter struct {
	mob *mobs.Mob
}

// AdaptMob returns a conversations.MobConversant view of the given mob,
// or nil if mob is nil.
func AdaptMob(m *mobs.Mob) conversations.MobConversant {
	if m == nil {
		return nil
	}
	return &mobAdapter{mob: m}
}

func (a *mobAdapter) ConvInstanceId() int { return a.mob.InstanceId }
func (a *mobAdapter) ConvMobId() int      { return int(a.mob.MobId) }
func (a *mobAdapter) ConvRoomId() int     { return a.mob.Character.RoomId }

func (a *mobAdapter) ConvGetMiscData(key string) any {
	return a.mob.Character.GetMiscData(key)
}

func (a *mobAdapter) ConvSetMiscData(key string, val any) {
	a.mob.Character.SetMiscData(key, val)
}

func (a *mobAdapter) ConvIsInCombat() bool { return a.mob.Character.IsInCombat() }

func (a *mobAdapter) ConvHasBuffFlag(f buffs.Flag) bool {
	return a.mob.Character.HasBuffFlag(f)
}

// ConvAggro returns true when the mob has an active aggro target.
// mob.Character.Aggro is a *characters.Aggro; non-nil means combat is pending.
func (a *mobAdapter) ConvAggro() bool { return a.mob.Character.Aggro != nil }

func (a *mobAdapter) ConvPathLen() int            { return a.mob.Path.Len() }
func (a *mobAdapter) ConvPathCurrentNonNil() bool { return a.mob.Path.Current() != nil }

// ConvCommand fires a mob command (fire-and-forget; return value discarded).
func (a *mobAdapter) ConvCommand(text string) { a.mob.Command(text) }

// ConvGetPartner looks up another mob by instance id and returns it as a
// MobConversant, or nil if not found.
func (a *mobAdapter) ConvGetPartner(instanceId int) conversations.MobConversant {
	other := mobs.GetInstance(instanceId)
	if other == nil {
		return nil
	}
	return &mobAdapter{mob: other}
}
