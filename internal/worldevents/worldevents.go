// Package worldevents provides a ring-buffer–based recording system for
// notable in-world occurrences (mob mutations, pack milestones, rare crafts,
// etc.). Events carry a significance tier and region information so that
// future display systems (town crier, rumor NPCs) can filter by locality.
package worldevents

import (
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ─── Types ───────────────────────────────────────────────────────────────────

type WorldEventType int

const (
	MobStatMilestone        WorldEventType = iota // Mob reached a notable stat threshold
	MobMutationGained                             // Mob acquired a new mutation
	MobMutationAdvanced                           // Mob deepened an existing mutation
	MobCraftedRare                                // Crafter mob produced a rare item
	PackStrengthened                              // Pack group gained a training bonus
	PlayerMutationMilestone                       // Player reached a mutation milestone
	PlayerCraftedRare                             // Player produced a rare item
	PlayerDiedPvE                                 // Player was killed by a mob or condition
	MobKilledByPlayer                             // A mob was slain by a player
)

type Significance int

const (
	Local    Significance = iota // Routine — shown only in the origin zone
	Regional                    // Notable — shown in zones sharing the same region
	Global                      // Impressive — shown everywhere
)

// WorldEvent captures a single notable occurrence in the game world.
type WorldEvent struct {
	Type         WorldEventType
	Significance Significance
	ZoneName     string // Origin zone display name
	RegionName   string // Origin region (from ZoneConfig)
	MobName      string
	PlayerName   string
	Description  string // Human-readable summary for display
	Timestamp    time.Time
	Round        uint64
}

// WorldEventFilter controls which events are returned by GetRecentWorldEvents.
type WorldEventFilter struct {
	MinSignificance Significance   // Return events at this tier or higher
	ZoneName        string         // Exact zone match (for Local tier filtering)
	RegionName      string         // Region match (for Regional tier filtering)
	EventType       WorldEventType // 0 matches all; set to a specific type to filter
	TypeSet         bool           // Must be true for EventType filter to apply
}

// ─── Ring buffer ─────────────────────────────────────────────────────────────

var (
	eventBuffer []WorldEvent
	maxEvents   int
	ready       bool
)

// InitWorldEvents allocates the ring buffer from config. Safe to call
// multiple times; subsequent calls are no-ops.
func InitWorldEvents() {
	if ready {
		return
	}

	maxEvents = int(configs.GetBalanceConfig().WorldEventBufferSize)
	if maxEvents < 10 {
		maxEvents = 200
	}

	eventBuffer = make([]WorldEvent, 0, maxEvents)
	ready = true

	mudlog.Info("worldevents.InitWorldEvents", "state", "World event system enabled",
		"bufferSize", maxEvents)
}

// EmitWorldEvent appends an event to the ring buffer.
// Must be called under util.LockMud() (same thread-safety contract as combat analytics).
func EmitWorldEvent(evt WorldEvent) {
	if !ready {
		InitWorldEvents()
		if !ready {
			return
		}
	}

	if evt.Timestamp.IsZero() {
		evt.Timestamp = time.Now()
	}
	if evt.Round == 0 {
		evt.Round = util.GetRoundCount()
	}

	// Ring buffer: drop oldest when full (same pattern as combat/analytics.go)
	if len(eventBuffer) >= maxEvents {
		copy(eventBuffer, eventBuffer[1:])
		eventBuffer = eventBuffer[:len(eventBuffer)-1]
	}
	eventBuffer = append(eventBuffer, evt)
}

// GetRecentWorldEvents returns up to n events matching the optional filter.
// Events are returned newest-first. Pass nil filter to get all events.
func GetRecentWorldEvents(n int, filter *WorldEventFilter) []WorldEvent {
	if !ready || len(eventBuffer) == 0 {
		return nil
	}

	result := make([]WorldEvent, 0, n)
	for i := len(eventBuffer) - 1; i >= 0 && len(result) < n; i-- {
		evt := eventBuffer[i]
		if filter != nil && !passesFilter(evt, filter) {
			continue
		}
		result = append(result, evt)
	}
	return result
}

// passesFilter returns true if evt satisfies all filter criteria.
func passesFilter(evt WorldEvent, f *WorldEventFilter) bool {
	// Significance gate
	if evt.Significance < f.MinSignificance {
		return false
	}

	// Event type filter
	if f.TypeSet && evt.Type != f.EventType {
		return false
	}

	// Locality check — Global events always pass
	if evt.Significance == Global {
		return true
	}

	// Regional events: if filter specifies a region, event's region must match
	if evt.Significance == Regional && f.RegionName != "" {
		return evt.RegionName == f.RegionName
	}

	// Local events: if filter specifies a zone, event's zone must match
	if evt.Significance == Local && f.ZoneName != "" {
		return evt.ZoneName == f.ZoneName
	}

	return true
}
