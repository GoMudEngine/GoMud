package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// WitnessResponse is how a mob reacts to witnessing (or being the victim of)
// a crime against another mob committed by a player.
type WitnessResponse int

const (
	// ResponseRevenge: combat-capable non-guard — seed a personal revenge goal.
	ResponseRevenge WitnessResponse = iota
	// ResponseAlarm: noncombatant civilian — momentary fright reaction; the
	// law (5.1 crime record + guard enforcement) handles the actual response.
	ResponseAlarm
	// ResponseReportOnly: guard (or nil) — seed nothing; 5.1 crime record +
	// RunGuardEnforcement enforce. A personal revenge goal would derail proper
	// enforcement.
	ResponseReportOnly
)

// classifyWitnessResponse decides how a mob should respond. Pure (no side
// effects). Guard takes precedence over the noncombatant check.
func classifyWitnessResponse(m *mobs.Mob) WitnessResponse {
	if m == nil {
		return ResponseReportOnly
	}
	if mobs.IsGuardMob(m.Groups) {
		return ResponseReportOnly
	}
	if m.IsNonCombatant() {
		return ResponseAlarm
	}
	return ResponseRevenge
}
