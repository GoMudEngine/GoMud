package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Actor is the unified interface for any entity (player or mob) that can
// perform shared combat/social actions. Adapters in actor_user.go and
// actor_mob.go implement this for *users.UserRecord and *mobs.Mob
// respectively.
type Actor interface {
	// GetCharacter returns the underlying character data.
	GetCharacter() *characters.Character

	// GetRoom returns the room the actor currently occupies.
	GetRoom() *rooms.Room

	// SendTextLegacy delivers a message to this actor only (no-op for
	// mobs). Temporary shim name during the T9 messaging-framework
	// migration; T10-T14 will replace with a categorized signature.
	SendTextLegacy(msg string)

	// SendRoomText broadcasts msg to the room. When excludeSelf is true the
	// actor's own connection is omitted from the broadcast.
	SendRoomText(msg string, excludeSelf bool)

	// SendRoomCommunication broadcasts a communication (say/shout/etc.) to
	// the room. Some clients suppress these messages based on deafen settings;
	// this variant goes through the communication pipeline rather than the raw
	// text pipeline. excludeSelf works the same as SendRoomText.
	SendRoomCommunication(msg string, excludeSelf bool)

	// GetName returns the display name of the actor.
	GetName() string

	// IsPlayer reports whether this actor is a human player connection.
	IsPlayer() bool

	// GetUserId returns the user ID for player actors, or 0 for mobs.
	GetUserId() int

	// GetMobInstanceId returns the mob instance ID for mob actors, or 0 for
	// players.
	GetMobInstanceId() int

	// AddBuff applies a buff to this actor via the event queue.
	AddBuff(buffId int, source string)

	// OnSkillUse triggers skill progression (and the skill's governing stat).
	// UserActor calls Character.OnSkillUse(skill, userId) which internally
	// fires events.SkillUsed for quest tracking when userId > 0.
	// MobActor calls Character.OnSkillUse(skill, 0).
	OnSkillUse(skillName string) bool

	// OnStatUse triggers stat progression.
	OnStatUse(statName string) bool

	// OnCriticalSuccess records a critical hit for progression bonuses.
	OnCriticalSuccess(skillName string)

	// OnCriticalFailure records a fumble for progression tracking.
	OnCriticalFailure(skillName string)
}
