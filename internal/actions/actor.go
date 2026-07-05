package actions

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
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

	// SendText delivers a categorized message to this actor only (no-op
	// for mobs). Routes through the centralized messaging pipeline.
	SendText(cat messaging.Category, msg string)

	// SendRoomCommunication broadcasts a communication (say/shout/etc.) to
	// the room. Some clients suppress these messages based on deafen settings;
	// this variant goes through the communication pipeline rather than the raw
	// text pipeline. excludeSelf works the same as the deleted SendRoomText.
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

// FunctionExporter mirrors usercommands.FunctionExporter — a decoupling
// seam so this package can reach plugin-exported functions (e.g. the
// weather module's GetWeather, used by forage.go) without importing
// internal/plugins directly, which would create an import cycle
// (plugins -> mobcommands -> actions). main.go wires the real
// plugin registry in via AddFunctionExporter at startup.
type FunctionExporter interface {
	GetExportedFunction(funcName string) (any, bool)
}

var functionExporters = []FunctionExporter{}

// AddFunctionExporter registers a source of plugin-exported functions.
func AddFunctionExporter(f FunctionExporter) {
	functionExporters = append(functionExporters, f)
}

// GetExportedFunction looks up a plugin-exported function by name across
// all registered exporters. Returns ok=false if none match (e.g. in unit
// tests, where no exporter has been registered) — callers must treat
// this as a safe, silent degrade.
func GetExportedFunction(fName string) (any, bool) {
	for _, x := range functionExporters {
		if f, ok := x.GetExportedFunction(fName); ok {
			return f, ok
		}
	}
	return nil, false
}
