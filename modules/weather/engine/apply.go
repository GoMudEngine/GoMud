package engine

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/seasons"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// WeatherMutatorPrefix namespaces every mutator this module owns.
const WeatherMutatorPrefix = "weather-"

// mutatorSet is the slice of MutatorList behavior the reconcile core needs;
// satisfied by *mutators.MutatorList and by test fakes (the real Add consults
// the global spec registry, so unit tests fake at this seam).
type mutatorSet interface {
	Add(string) bool
	Remove(string) bool
}

// MutatorIdFor maps a sim weather type to its mutator id; "" for clear/unset
// (calm weather is the absence of a weather mutator).
func MutatorIdFor(w sim.WeatherType) string {
	if w == "" || w == sim.Clear {
		return ""
	}
	return WeatherMutatorPrefix + string(w)
}

// reconcileList forces one mutator list's ids WITHIN ONE NAMESPACE to exactly
// match want: every id in current except want is removed; want is added if
// absent ("" = remove all). current must hold only ids from the same
// namespace (the caller gathers by prefix). Returns false when the want spec
// id is unknown (data file missing or failed to load). Our specs must not
// carry decayintoid: the engine's Remove resets SpawnedRound and runs Update,
// whose decay branch would instantly resurrect the entry as the decay target
// (a shipped-data test enforces this).
func reconcileList(ms mutatorSet, current []string, want string) bool {
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

func warnUnknownMutatorId(id string) {
	if id == "" || warnedMutators[id] {
		return
	}
	warnedMutators[id] = true
	mudlog.Warn("Weather: no mutator spec loaded for weather type", "mutatorId", id)
}

// SeasonMutatorPrefix namespaces the seasonal-ambience mutators; independent
// of WeatherMutatorPrefix — the two reconcile layers never touch each other's
// ids.
const SeasonMutatorPrefix = "season-"

// SeasonMutatorId maps a zone's resolved (track, season) to its mutator id;
// "" when either part is empty (no seasonal mutator).
func SeasonMutatorId(track, season string) string {
	if track == "" || season == "" {
		return ""
	}
	return SeasonMutatorPrefix + track + "-" + season
}

// ReconcileSeasons forces every zone's season-* mutators to match its
// resolved season — used at boot, each tick, and after a graph rebuild.
// Zones absent from the map (unbound biomes) get their season-* mutators
// removed, so a zone whose biome lost its track binding heals.
func ReconcileSeasons(g *sim.Graph, zoneSeasons map[sim.ZoneId]seasons.ZoneSeason) {
	for _, zone := range g.Zones() {
		zc := rooms.GetZoneConfig(zone)
		if zc == nil {
			continue
		}
		var current []string
		for _, mut := range zc.Mutators.GetActive() {
			if strings.HasPrefix(mut.MutatorId, SeasonMutatorPrefix) {
				current = append(current, mut.MutatorId)
			}
		}
		want := ""
		if zs, ok := zoneSeasons[zone]; ok {
			want = SeasonMutatorId(zs.Track, zs.Season)
		}
		if len(current) == 0 && want == "" {
			continue
		}
		if !reconcileList(&zc.Mutators, current, want) {
			warnUnknownSeasonMutator(want)
		}
	}
}

// warnedSeasonMutators: warn-once for missing season specs (single goroutine).
var warnedSeasonMutators = map[string]bool{}

func warnUnknownSeasonMutator(id string) {
	if id == "" || warnedSeasonMutators[id] {
		return
	}
	warnedSeasonMutators[id] = true
	mudlog.Warn("Weather: no mutator spec loaded for season", "mutatorId", id)
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
		want := MutatorIdFor(w)
		if !reconcileList(&zc.Mutators, weatherIds(&zc.Mutators), want) {
			warnUnknownMutatorId(want)
		}
	}
}

// StripBuffs clears the buff id lists on every loaded weather-* and season-*
// mutator spec — the BuffsEnabled=false path. GetMutatorSpec returns the
// registry's live pointer, so this affects all future applications. Returns
// the count stripped. Boot-time only: there is no restore path, so
// re-enabling buffs requires a reload.
func StripBuffs() int {
	n := 0
	for _, id := range mutators.GetAllMutatorIds() {
		if !strings.HasPrefix(id, WeatherMutatorPrefix) && !strings.HasPrefix(id, SeasonMutatorPrefix) {
			continue
		}
		if spec := mutators.GetMutatorSpec(id); spec != nil {
			spec.PlayerBuffIds, spec.MobBuffIds, spec.NativeBuffIds = nil, nil, nil
			n++
		}
	}
	return n
}

// ApplyBuffOverrides rewires PlayerBuffIds on the registered OUTDOOR weather
// specs per the BuffOverrides.<type> config: each entry replaces that type's
// player buff list wholesale (an empty list strips it; Mob/Native lists are
// untouched). Indoor variants are buff-free by rule and never touched (the
// "weather-"+type id can't match a "-indoor" spec). Boot-time spec mutation
// with the same mechanism and no-restore caveat as StripBuffs — and the module
// always runs it BEFORE StripBuffs, so BuffsEnabled=false wins over any
// override (spec §3). Returns the number of specs changed.
func ApplyBuffOverrides(overrides map[string][]int) int {
	return applyBuffOverrides(mutators.GetMutatorSpec, overrides)
}

// applyBuffOverrides is the testable core; the registry lookup is the seam
// (the live spec registry is empty under `go test`).
func applyBuffOverrides(lookup func(string) *mutators.MutatorSpec, overrides map[string][]int) int {
	n := 0
	for t, ids := range overrides {
		id := WeatherMutatorPrefix + t
		spec := lookup(id)
		if spec == nil {
			warnUnknownOverride(id)
			continue
		}
		// Copy so the spec never aliases the config map's backing arrays.
		spec.PlayerBuffIds = append([]int(nil), ids...)
		n++
	}
	return n
}

// warnedOverrides: warn-once for overrides naming no loaded spec (e.g. a typo,
// or "clear", which is the absence of a mutator). Single goroutine — no mutex.
var warnedOverrides = map[string]bool{}

func warnUnknownOverride(id string) {
	if warnedOverrides[id] {
		return
	}
	warnedOverrides[id] = true
	mudlog.Warn("Weather: BuffOverrides entry matches no loaded mutator spec; ignored", "mutatorId", id)
}
