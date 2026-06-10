package engine

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// WeatherMutatorPrefix namespaces every mutator this module owns.
const WeatherMutatorPrefix = "weather-"

// mutatorSet is the slice of MutatorList behavior the applier needs; satisfied
// by *mutators.MutatorList and by test fakes (the real Add consults the global
// spec registry, so unit tests fake at this seam).
type mutatorSet interface {
	Add(string) bool
	Remove(string) bool
	Has(string) bool
}

// MutatorIdFor maps a sim weather type to its mutator id; "" for clear/unset
// (calm weather is the absence of a weather mutator).
func MutatorIdFor(w sim.WeatherType) string {
	if w == "" || w == sim.Clear {
		return ""
	}
	return WeatherMutatorPrefix + string(w)
}

// applyChange applies one zone weather transition. Add is guarded by Has
// because MutatorList.Add appends a duplicate entry when the mutator is
// already live. Returns false when the target spec id is unknown (data file
// missing or failed to load). Weather specs must not carry decayintoid: the
// engine's Remove resets SpawnedRound and runs Update, whose decay branch
// would instantly resurrect the entry as the decay target (a shipped-data
// test enforces this).
func applyChange(ms mutatorSet, from, to sim.WeatherType) bool {
	if id := MutatorIdFor(from); id != "" {
		ms.Remove(id)
	}
	if id := MutatorIdFor(to); id != "" {
		if ms.Has(id) {
			return true
		}
		return ms.Add(id)
	}
	return true
}

// reconcileZone forces a zone's weather mutators to exactly match target:
// every live weather-* id except the target is removed; the target is added if
// absent. current must hold the zone's live weather-* mutator ids.
func reconcileZone(ms mutatorSet, current []string, target sim.WeatherType) bool {
	want := MutatorIdFor(target)
	hasWant := false
	for _, id := range current {
		if id == want {
			hasWant = true
			continue
		}
		ms.Remove(id)
	}
	if want == "" || hasWant {
		return true
	}
	return ms.Add(want)
}

// warnedMutators tracks unknown-spec warnings so each id logs once. Touched
// only from the single game-loop goroutine — no mutex (see context.md).
var warnedMutators = map[string]bool{}

func warnUnknownMutator(w sim.WeatherType) {
	id := MutatorIdFor(w)
	if id == "" || warnedMutators[id] {
		return
	}
	warnedMutators[id] = true
	mudlog.Warn("Weather: no mutator spec loaded for weather type", "mutatorId", id)
}

// Apply walks a StateDiff and applies each change to its zone's zone-wide
// mutator list (spec §9.1 primary strategy). Zones missing from the live world
// (stale graph) are skipped.
func Apply(diff sim.StateDiff) {
	for _, ch := range diff.Changes {
		zc := rooms.GetZoneConfig(ch.Zone)
		if zc == nil {
			continue
		}
		if !applyChange(&zc.Mutators, ch.From, ch.To) {
			warnUnknownMutator(ch.To)
		}
	}
}

// Reconcile forces every zone's live weather mutators to match the resolved
// weather map — used at boot after restoring persisted state (zone mutators do
// not survive reboots) and after a graph rebuild.
func Reconcile(weather map[sim.ZoneId]sim.WeatherType) {
	for zone, w := range weather {
		zc := rooms.GetZoneConfig(zone)
		if zc == nil {
			continue
		}
		var current []string
		for _, mut := range zc.Mutators.GetActive() {
			if strings.HasPrefix(mut.MutatorId, WeatherMutatorPrefix) {
				current = append(current, mut.MutatorId)
			}
		}
		if !reconcileZone(&zc.Mutators, current, w) {
			warnUnknownMutator(w)
		}
	}
}

// StripBuffs clears the buff id lists on every loaded weather-* mutator spec —
// the BuffsEnabled=false path. GetMutatorSpec returns the registry's live
// pointer, so this affects all future applications. Returns the count stripped.
// Boot-time only: there is no restore path, so re-enabling buffs requires a
// reload.
func StripBuffs() int {
	n := 0
	for _, id := range mutators.GetAllMutatorIds() {
		if !strings.HasPrefix(id, WeatherMutatorPrefix) {
			continue
		}
		if spec := mutators.GetMutatorSpec(id); spec != nil {
			spec.PlayerBuffIds, spec.MobBuffIds, spec.NativeBuffIds = nil, nil, nil
			n++
		}
	}
	return n
}
