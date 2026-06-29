package behaviortree

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// calcReactionDelay computes a perception-scaled delay for behavior tree actions.
// Formula: delay = base - (perception / scale), clamped to [min, max].
// High-perception mobs react faster; low-perception mobs react slower.
// Returns 0 if the mob instance doesn't exist.
func calcReactionDelay(mobInstanceId int) time.Duration {
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return 0
	}
	cfg := configs.GetBalanceConfig()
	base := float64(cfg.MobBTreeReactionBase)
	scale := float64(cfg.MobBTreeReactionPerceptionScale)
	minDelay := float64(cfg.MobReactionDelayMin)
	maxDelay := float64(cfg.MobReactionDelayMax)

	if scale <= 0 {
		scale = 100
	}

	perception := float64(mob.Character.Stats.Perception.ValueAdj)
	delay := base - (perception / scale)

	if delay < minDelay {
		delay = minDelay
	}
	if delay > maxDelay {
		delay = maxDelay
	}
	return time.Duration(delay * float64(time.Second))
}

// GetBehaviorPath constructs the filesystem path to a mob's behavior tree YAML.
// Path: {dataFiles}/../behaviors/{zone}/{mobId}-{convertedName}.yaml
// Behavior trees live in a top-level behaviors/ directory parallel to mobs/,
// NOT inside mobs/ (the mob loader panics on unknown YAML in its tree).
func GetBehaviorPath(mobId int, zone string, name string) string {
	dataFiles := configs.GetFilePathsConfig().DataFiles.String()
	zoneSafe := mobs.ZoneNameSanitize(zone)
	nameSafe := util.ConvertForFilename(name)
	return util.FilePath(dataFiles, `/`, `behaviors`, `/`, zoneSafe, `/`,
		strconv.Itoa(mobId)+`-`+nameSafe+`.yaml`)
}

// EnsureBTreeState lazily initializes the BehaviorState on a mob instance.
func EnsureBTreeState(mob *mobs.Mob) *BehaviorState {
	if mob.BTreeState == nil {
		mob.BTreeState = NewBehaviorState()
	}
	state, ok := mob.BTreeState.(*BehaviorState)
	if !ok {
		state = NewBehaviorState()
		mob.BTreeState = state
	}
	return state
}

// GetRoomBehaviorPath constructs the filesystem path to a room's behavior tree
// YAML. Path: {dataFiles}/behaviors/rooms/{zoneSafe}/{roomId}.yaml
func GetRoomBehaviorPath(roomId int, zone string) string {
	dataFiles := configs.GetFilePathsConfig().DataFiles.String()
	zoneSafe := rooms.ZoneNameSanitize(zone)
	return util.FilePath(dataFiles, `/`, `behaviors`, `/`, `rooms`, `/`, zoneSafe, `/`,
		strconv.Itoa(roomId)+`.yaml`)
}

// GetArchetypePath constructs the filesystem path to an archetype btree YAML.
// Path: {dataFiles}/behaviors/archetypes/{name}.yaml
// Archetype names are treated as filesystem-safe tokens; callers are
// responsible for sanitization (we do not apply ConvertForFilename
// because archetype names are authored, not user-derived).
func GetArchetypePath(name string) string {
	dataFiles := configs.GetFilePathsConfig().DataFiles.String()
	return util.FilePath(dataFiles, `/`, `behaviors`, `/`, `archetypes`, `/`,
		name+`.yaml`)
}

// TryRoomBehavior is the main entry point for room behavior tree event dispatch.
// For room_command events it returns ctx.Intercepted; for all others it returns
// true when the tree evaluates to Success.
func TryRoomBehavior(roomId int, event EventContext) bool {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return false
	}

	// Ephemeral room instances inherit their template's behavior file (which
	// is named by the template id). Resolve the tree by the original id; keep
	// per-instance STATE keyed by the instance roomId so each player's copy
	// gates independently.
	behaviorRoomId := roomId
	if orig, ok := rooms.OriginalRoomId(roomId); ok {
		behaviorRoomId = orig
	}

	// Lazy-load tree if not cached.
	tree := GetEngine().GetRoomTree(behaviorRoomId)
	if tree == nil {
		if GetEngine().HasNoRoomTree(behaviorRoomId) {
			return false
		}
		path := GetRoomBehaviorPath(behaviorRoomId, room.Zone)
		if _, err := os.Stat(path); err != nil {
			GetEngine().SetNoRoomTree(behaviorRoomId)
			return false
		}
		if err := GetEngine().LoadRoomTree(behaviorRoomId, path); err != nil {
			mudlog.Error("TryRoomBehavior", "error", fmt.Sprintf("failed to load behavior tree for room %d: %v", behaviorRoomId, err))
			return false
		}
		tree = GetEngine().GetRoomTree(behaviorRoomId)
		if tree == nil {
			return false
		}
	}

	state := EnsureRoomBTreeState(roomId)
	event.RoomId = roomId

	ctx := &EvalContext{
		Event:    event,
		MobState: state,
		RoomId:   roomId,
	}
	result := tree.Evaluate(ctx)

	if event.EventType == "room_command" {
		return ctx.Intercepted
	}
	return result == Success
}

// TryMobBehavior is the main entry point for event dispatch.
// Returns true if the behavior tree handled the event (Success).
//
// Resolution order:
//  1. Per-mob btree file (behaviors/<zone>/<mobId>-<name>.yaml)
//  2. Archetype tree (if mob.BehaviorArchetype is set)
//  3. No tree — returns false
func TryMobBehavior(mobInstanceId int, event EventContext) bool {
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return false
	}

	mobId := int(mob.MobId)
	engine := GetEngine()

	// 1. Try the per-mob btree (by mobId).
	tree := engine.GetTree(mobId)
	if tree == nil && !engine.HasNoTree(mobId) {
		path := GetBehaviorPath(mobId, mob.Zone, mob.Character.Name)
		if _, err := os.Stat(path); err != nil {
			engine.SetNoTree(mobId)
		} else if err := engine.LoadTree(mobId, path); err != nil {
			mudlog.Error("TryMobBehavior", "error", fmt.Sprintf("failed to load behavior tree for mob %d (%s): %v", mobId, path, err))
			engine.SetNoTree(mobId)
		} else {
			tree = engine.GetTree(mobId)
		}
	}

	// 2. Fall through to archetype if per-mob tree absent AND mob declares an archetype.
	if tree == nil && mob.BehaviorArchetype != "" {
		name := mob.BehaviorArchetype
		tree = engine.GetArchetype(name)
		if tree == nil && !engine.HasNoArchetype(name) {
			path := GetArchetypePath(name)
			if _, err := os.Stat(path); err != nil {
				engine.SetNoArchetype(name)
				mudlog.Warn("TryMobBehavior", "archetype", name, "warning", fmt.Sprintf("archetype file not found: %s", path))
			} else if err := engine.LoadArchetype(name, path); err != nil {
				mudlog.Error("TryMobBehavior", "error", fmt.Sprintf("failed to load archetype %s (%s): %v", name, path, err))
				engine.SetNoArchetype(name)
			} else {
				tree = engine.GetArchetype(name)
			}
		}
	}

	// 3. No tree — caller runs legacy path.
	if tree == nil {
		return false
	}

	state := EnsureBTreeState(mob)
	event.RoomId = mob.Character.RoomId

	ctx := &EvalContext{
		Event:      event,
		MobState:   state,
		MobId:      mobId,
		InstanceId: mobInstanceId,
		RoomId:     mob.Character.RoomId,
		MobName:    mob.Character.Name,
	}
	result := tree.Evaluate(ctx)
	return result == Success
}
