package actions

import (
	"errors"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ResolveTargetOptions configures target resolution behavior for
// ResolveTargetActor.
type ResolveTargetOptions struct {
	// FindFlags filters which entities are eligible. When empty (zero value)
	// the helper falls back to rooms.FindAll.
	FindFlags []rooms.FindFlag
	// ExcludeUserId, when > 0, hides the named user from results. Used for
	// self-exclusion in commands like "consider".
	ExcludeUserId int
	// ExcludeMobInstanceId, when > 0, hides the named mob from results.
	ExcludeMobInstanceId int
}

// Sentinel errors returned by ResolveTargetActor. Callers can use
// errors.Is to distinguish causes for messaging purposes.
var (
	// ErrTargetNotFound is returned when no entity in the room matches the
	// search name (or the only matches were excluded by options).
	ErrTargetNotFound = errors.New("target not found")
	// ErrTargetVanished is returned when FindByName resolved an ID but the
	// corresponding pointer lookup (mobs.GetInstance / users.GetByUserId)
	// returned nil. Race condition: the entity left between the find and
	// the lookup, or the registry was mutated mid-call.
	ErrTargetVanished = errors.New("target vanished")
	// ErrTargetSelfExcluded is reserved for future use when distinguishing
	// "no match" from "matched but excluded by self-exclusion" matters for
	// user-facing messaging. Currently ResolveTargetActor returns
	// ErrTargetNotFound in both cases — no caller needs the differentiation
	// today.
	ErrTargetSelfExcluded = errors.New("target is self")
)

// ResolveTargetActor looks up a player or mob by name in the given room and
// returns it wrapped in an Actor. Returns ErrTargetNotFound if no match
// (or only matches were excluded), ErrTargetVanished if FindByName returned
// an ID but the registry pointer lookup returned nil.
//
// The returned Actor is a concrete *UserActor or *MobActor — callers can
// type-assert when they need type-specific behavior (mob.IsNonCombatant,
// user.PartyId, etc.).
//
// Players take precedence over mobs when both match the same name. This
// matches the implicit convention in pre-refactor call sites and aligns
// with the intuition that named players are usually the intended target.
// Known limitation: nothing prevents player-mob name collisions; see
// project_name_collision_prevention.md (future work).
//
// Note: the design spec calls this method out as r.ResolveTargetActor(name)
// living on Room. That would create an import cycle (internal/actions
// already imports internal/rooms), so the helper lives in actions as a
// free function taking the room as the first argument. Caller pattern:
//
//	target, err := actions.ResolveTargetActor(room, name)
//	if err != nil {
//	    user.SendText(messaging.CategoryError, "You don't see them here.")  // caller controls wording
//	    return true, nil
//	}
//	// ... use target uniformly ...
//	if !target.IsPlayer() {
//	    mob := target.(*MobActor).Mob
//	    if mob.IsNonCombatant() { /* ... */ }
//	}
func ResolveTargetActor(r *rooms.Room, name string, opts ...ResolveTargetOptions) (Actor, error) {
	var o ResolveTargetOptions
	if len(opts) > 0 {
		o = opts[0]
	}

	flags := o.FindFlags
	if len(flags) == 0 {
		flags = []rooms.FindFlag{rooms.FindAll}
	}

	playerId, mobInstanceId := r.FindByName(name, flags...)

	// Apply exclusions.
	if o.ExcludeUserId > 0 && playerId == o.ExcludeUserId {
		playerId = 0
	}
	if o.ExcludeMobInstanceId > 0 && mobInstanceId == o.ExcludeMobInstanceId {
		mobInstanceId = 0
	}

	// Players take precedence over mobs (see docstring). Wrap with the
	// InRoom variants so target.GetRoom() / SendRoomText() work without
	// callers having to re-resolve the room.
	if playerId > 0 {
		u := users.GetByUserId(playerId)
		if u == nil {
			return nil, ErrTargetVanished
		}
		return NewUserActorInRoom(u, r), nil
	}
	if mobInstanceId > 0 {
		m := mobs.GetInstance(mobInstanceId)
		if m == nil {
			return nil, ErrTargetVanished
		}
		return NewMobActorInRoom(m, r), nil
	}
	return nil, ErrTargetNotFound
}
