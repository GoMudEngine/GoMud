package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// BenchmarkMobRoundTick_AllActive simulates a world where every zone
// has a player in it. All mobs run the full active-path code.
func BenchmarkMobRoundTick_AllActive(b *testing.B) {
	// Setup requires a mob + room fixture builder that is independent
	// of the full data-file load. None exists in the hooks test package
	// today. Once a fixture lands, un-skip and populate the benchmark.
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")

	_ = events.NewRound{}
	_ = rooms.SnapshotActiveZones
	for i := 0; i < b.N; i++ {
		// MobRoundTick(events.NewRound{RoundNumber: uint64(i)})
	}
}

// BenchmarkMobRoundTick_MostlyIdle simulates 10% active zones — the
// common real-world case. Target metric: ≥5× faster than AllActive.
func BenchmarkMobRoundTick_MostlyIdle(b *testing.B) {
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")
}

// BenchmarkMobRoundTick_AllIdle simulates no players anywhere.
func BenchmarkMobRoundTick_AllIdle(b *testing.B) {
	b.Skip("requires mob+room fixture infrastructure — see spec §Testing")
}
