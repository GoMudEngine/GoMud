package engine

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutators"
	"gopkg.in/yaml.v2"
)

// The shipped weather mutator specs must obey the engine-lifecycle rules the
// module depends on: no respawnrate (fights the reconciler), no decayintoid
// (MutatorList.Remove instantly resurrects the decay target), decayrate
// present (self-heal if the module is disabled mid-storm), outdooronly set
// (indoor rooms must not render weather), no buff ids (presentation-only),
// weather- namespace, and loader-compatible filenames.
func TestShippedWeatherMutatorSpecs(t *testing.T) {
	dir := filepath.Join("..", "..", "..", "_datafiles", "world", "dogmud", "mutators")
	matches, err := filepath.Glob(filepath.Join(dir, "weather_*.yaml"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}
	if len(matches) != 8 {
		t.Fatalf("expected 8 weather mutator specs, found %d: %v", len(matches), matches)
	}

	for _, path := range matches {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		var spec mutators.MutatorSpec
		if err := yaml.Unmarshal(b, &spec); err != nil {
			t.Fatalf("%s: unmarshal: %v", path, err)
		}
		name := filepath.Base(path)

		if !strings.HasPrefix(spec.MutatorId, WeatherMutatorPrefix) {
			t.Errorf("%s: mutatorid %q not in %s namespace", name, spec.MutatorId, WeatherMutatorPrefix)
		}
		wantFile := strings.Map(func(r rune) rune {
			if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
				return r
			}
			return '_'
		}, strings.ToLower(spec.MutatorId)) + ".yaml"
		if name != wantFile {
			t.Errorf("%s: filename should be %q for id %q", name, wantFile, spec.MutatorId)
		}
		if spec.RespawnRate != "" {
			t.Errorf("%s: respawnrate must be empty (fights the weather reconciler)", name)
		}
		if spec.DecayIntoId != "" {
			t.Errorf("%s: decayintoid must be empty (Remove would resurrect it)", name)
		}
		if spec.DecayRate == "" {
			t.Errorf("%s: decayrate required as the self-heal safety net", name)
		}
		if !spec.OutdoorOnly {
			t.Errorf("%s: outdooronly must be true", name)
		}
		if len(spec.PlayerBuffIds)+len(spec.MobBuffIds)+len(spec.NativeBuffIds) != 0 {
			t.Errorf("%s: buff ids must be empty (presentation-only)", name)
		}
	}
}
