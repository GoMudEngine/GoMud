package mobai

import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

// pendingReactions holds queued reactions waiting to fire.
var pendingReactions []PendingReaction

// SignalData is passed to the reactor when a combat-relevant event occurs.
// The hooks layer builds this from concrete types and passes it in.
type SignalData struct {
	MobInstanceId int
	SignalType    string // "combat_start", "damage_taken", "action_complete", "player_entered", "target_fled"
	Detail        string // e.g. action name for "action_complete"
	RoomId        int
}

// ReactorConfig holds tuning values extracted from game config by the caller.
type ReactorConfig struct {
	Enabled          bool
	ReactionDelayMin float64
	ReactionDelayMax float64
	TurnsPerSecond   int
}

// GetEffectiveReactionDelay returns the mob's reaction delay, clamped
// to the configured min/max range. Returns 1.5 if not set.
func GetEffectiveReactionDelay(mobDelay float64, minDelay float64, maxDelay float64) float64 {
	if mobDelay <= 0 {
		mobDelay = 1.5
	}
	if mobDelay < minDelay {
		mobDelay = minDelay
	}
	if mobDelay > maxDelay {
		mobDelay = maxDelay
	}
	return mobDelay
}

// ShouldFollowTactic rolls against tactical discipline.
// Returns true if the mob should execute the tactic.
func ShouldFollowTactic(discipline float64) bool {
	if discipline <= 0 {
		return false
	}
	if discipline >= 1.0 {
		return true
	}
	return util.Rand(100) < int(discipline*100)
}

// ResolveTactics returns the effective tactic list for a mob,
// merging preset + custom tactics.
func ResolveTactics(preset string, custom []TacticRule) []TacticRule {
	var base []TacticRule
	if preset != "" {
		base = GetPreset(preset)
	}
	if len(custom) > 0 {
		if base == nil {
			return custom
		}
		return MergeTactics(base, custom)
	}
	return base
}

// ProcessSignal evaluates a mob's tactics against a signal and queues
// a reaction if appropriate. Called by the hooks layer.
//
// Returns the action string if a reaction was queued, "" otherwise.
func ProcessSignal(signal SignalData, ctx *TriggerContext, tactics []TacticRule,
	discipline float64, reactionDelay float64, lastReactionTurn uint64,
	cfg ReactorConfig) string {

	if !cfg.Enabled {
		return ""
	}

	if len(tactics) == 0 {
		return ""
	}

	// Check reaction cooldown. Skip if mob has never reacted (lastReactionTurn==0)
	// to avoid uint64 underflow when currentTurn is also 0 or small.
	currentTurn := util.GetTurnCount()
	delay := GetEffectiveReactionDelay(reactionDelay, cfg.ReactionDelayMin, cfg.ReactionDelayMax)
	cooldownTurns := uint64(delay * float64(cfg.TurnsPerSecond))
	if lastReactionTurn > 0 && currentTurn-lastReactionTurn < cooldownTurns {
		return ""
	}

	// Set signal-specific context fields
	switch signal.SignalType {
	case "combat_start":
		ctx.CombatJustStarted = true
	case "action_complete":
		ctx.LastAction = signal.Detail
	case "player_entered":
		ctx.PlayerEntered = true
	}

	// Evaluate tactics
	action := EvaluateTactics(tactics, ctx)
	if action == "" {
		return ""
	}

	// Roll discipline
	if !ShouldFollowTactic(discipline) {
		return ""
	}

	// Queue the reaction
	fireTurn := currentTurn + uint64(delay*float64(cfg.TurnsPerSecond))
	pendingReactions = append(pendingReactions, PendingReaction{
		MobInstanceId: signal.MobInstanceId,
		Action:        action,
		FireTurn:      fireTurn,
	})

	return action
}

// ProcessPendingReactions fires any queued reactions whose delay has
// elapsed. Called on every NewTurn (100ms tick) by the hooks layer.
//
// The mobResolver callback converts instance IDs to MobActors.
// The roomLoader callback loads a room by ID.
// These callbacks are provided by the hooks layer to avoid import cycles.
func ProcessPendingReactions(currentTurn uint64,
	mobResolver func(int) MobActor,
	roomLoader func(int) RoomAccessor,
	mobLookup MobLookup,
	playerLookup PlayerLookup) {

	if len(pendingReactions) == 0 {
		return
	}

	remaining := make([]PendingReaction, 0, len(pendingReactions))

	for _, pr := range pendingReactions {
		if currentTurn < pr.FireTurn {
			remaining = append(remaining, pr)
			continue
		}

		mob := mobResolver(pr.MobInstanceId)
		if mob == nil {
			continue
		}

		room := roomLoader(mob.GetRoomId())
		ExecuteAction(mob, pr.Action, room, mobLookup, playerLookup)
	}

	pendingReactions = remaining
}

// ClearPendingForMob removes all pending reactions for a specific mob
// (e.g. when the mob dies or despawns).
func ClearPendingForMob(mobInstanceId int) {
	remaining := make([]PendingReaction, 0, len(pendingReactions))
	for _, pr := range pendingReactions {
		if pr.MobInstanceId != mobInstanceId {
			remaining = append(remaining, pr)
		}
	}
	pendingReactions = remaining
}

// PendingCount returns the number of queued reactions (for testing/debugging).
func PendingCount() int {
	return len(pendingReactions)
}

// ClearAllPending removes all pending reactions (for testing).
func ClearAllPending() {
	pendingReactions = nil
}
