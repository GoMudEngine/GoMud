package seasons

import (
	"os"
	"testing"
)

// TestShippedTracks validates the default track files against the stock
// 12-month/365-day calendar (worlds with other calendars must author their
// own — design spec S-R3).
func TestShippedTracks(t *testing.T) {
	tracks, errs := Load(os.DirFS("../../../_datafiles/world/dogmud"), "weather/seasons", 12, 365)
	if len(errs) != 0 {
		t.Fatalf("shipped tracks invalid: %v", errs)
	}
	tr, ok := tracks["temperate"]
	if !ok || len(tr.Seasons) != 4 {
		t.Fatalf("temperate track missing or wrong shape: %+v", tracks)
	}
	if mo, ok := tracks["monsoon"]; !ok || len(mo.Seasons) != 2 {
		t.Fatalf("monsoon track missing or wrong shape: %+v", tracks)
	}
	// Every multiplier references a weather type some default climate knows,
	// so a typo'd type can't silently no-op.
	known := map[string]bool{"clear": true, "overcast": true, "rain": true, "storm": true,
		"fog": true, "snow": true, "blizzard": true, "dust": true, "heatwave": true}
	for name, track := range tracks {
		for _, s := range track.Seasons {
			for w := range s.WeatherWeightMultipliers {
				if !known[string(w)] {
					t.Errorf("track %s season %s: unknown weather type %q", name, s.Name, w)
				}
			}
			// Shipped tracks must not add types we ship no mutator/emote for.
			for w := range s.WeatherWeightAdditions {
				if !known[string(w)] {
					t.Errorf("track %s season %s: addition for unshipped weather type %q", name, s.Name, w)
				}
			}
		}
	}
}
