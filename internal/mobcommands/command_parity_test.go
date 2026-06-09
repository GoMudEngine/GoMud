package mobcommands

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// TestCommandIsReadyNamesAreMobCommands verifies every command that
// actions.CommandIsReady knows about is registered in the mob command
// dispatcher. If this test fails, a btree's command_best_of action
// could "fire" a command that is actually just a confused-emote
// fallback — the exact T7 smoke-test bug that required taunt to be
// registered as a Howl alias.
//
// If CommandIsReady gains a new case, either:
//  1. Register a matching mob command in mobcommands.go, or
//  2. If the command is intentionally player-only (e.g. a spell-path
//     that mobs can't cast), remove its case from CommandIsReady.
func TestCommandIsReadyNamesAreMobCommands(t *testing.T) {
	// Known list of command names CommandIsReady supports. Keep this
	// in sync with the switch cases in
	// internal/actions/command_readiness.go.
	supported := []string{
		"taunt", "rally", "warcry", "trip", "bash", "grapple", "kick", "rake",
	}

	// Verify each name IS in the mob command dispatcher.
	mobCmdSet := make(map[string]struct{})
	for _, name := range GetAllMobCommands() {
		mobCmdSet[name] = struct{}{}
	}

	for _, cmd := range supported {
		_, ok := mobCmdSet[cmd]
		assert.True(t, ok,
			"CommandIsReady supports %q but the mob command dispatcher does not — "+
				"btree command_best_of would fire a confused emote. "+
				"Either register %q in mobcommands.go or remove the case from "+
				"actions.CommandIsReady.", cmd, cmd)
	}
}
