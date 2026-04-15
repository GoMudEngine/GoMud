package behaviortree

import (
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
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

// TryMobBehavior is the main entry point for event dispatch.
// Returns true if the behavior tree handled the event (Success).
func TryMobBehavior(mobInstanceId int, event EventContext) bool {
	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return false
	}

	mobId := int(mob.MobId)

	// Lazy-load tree if not cached
	tree := GetEngine().GetTree(mobId)
	if tree == nil {
		path := GetBehaviorPath(mobId, mob.Zone, mob.Character.Name)
		// Check if file exists
		if _, err := os.Stat(path); err != nil {
			return false // No behavior tree for this mob
		}
		if err := GetEngine().LoadTree(mobId, path); err != nil {
			mudlog.Error("TryMobBehavior", "error", fmt.Sprintf("failed to load behavior tree for mob %d: %v", mobId, err))
			return false
		}
		tree = GetEngine().GetTree(mobId)
		if tree == nil {
			return false
		}
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
