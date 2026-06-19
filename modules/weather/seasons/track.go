package seasons

import (
	"fmt"
	"io/fs"
	"path"
	"strings"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
	"gopkg.in/yaml.v2"
)

// Season is one phase of a track's cycle.
type Season struct {
	Name                     string
	Months                   []int // 1-based calendar months, one contiguous run (may wrap)
	TransitionDays           int   // blend window at the START of this season; 0 = hard flip
	WeatherWeightMultipliers map[sim.WeatherType]float64
	WeatherWeightAdditions   map[sim.WeatherType]float64 // absolute weights ADDED — may introduce types absent from the base climate (esoteric seasons, spec §3.1a)
	BaseWeightScale          float64                     // scales ALL base weights before additions; 1.0 = neutral, 0 = suppress normal weather
	SpawnWeightMultiplier    float64
	Influence                sim.WeatherInfluence // deltas ADDED to the biome's own

	startMonth int // derived at load: first month of the contiguous run
}

// Track is one season cycle plus the calendar shape it was validated against.
type Track struct {
	Name    string
	Seasons []Season

	monthsPerYear int
	daysPerYear   int
}

// Tracks maps track name -> track.
type Tracks map[string]Track

// trackFile mirrors the on-disk schema (design spec §3.1).
type trackFile struct {
	Track   string `yaml:"track"`
	Seasons []struct {
		Name                     string             `yaml:"name"`
		Months                   []int              `yaml:"months"`
		TransitionDays           int                `yaml:"transitionDays"`
		WeatherWeightMultipliers map[string]float64 `yaml:"weatherWeightMultipliers"`
		WeatherWeightAdditions   map[string]float64 `yaml:"weatherWeightAdditions"`
		BaseWeightScale          *float64           `yaml:"baseWeightScale"`
		SpawnWeightMultiplier    *float64           `yaml:"spawnWeightMultiplier"`
		Influence                struct {
			IntensityDelta     float64 `yaml:"intensityDelta"`
			MoistureDelta      float64 `yaml:"moistureDelta"`
			MovementResistance float64 `yaml:"movementResistance"`
		} `yaml:"influence"`
	} `yaml:"seasons"`
}

// Load parses every *.yaml track under dir and validates each against the
// world's calendar shape. Invalid tracks are dropped and reported; valid ones
// are returned — the caller logs the errors and continues (fail-soft). A
// missing dir is no tracks, no errors.
func Load(fsys fs.FS, dir string, monthsPerYear, daysPerYear int) (Tracks, []error) {
	tracks := Tracks{}
	var errs []error
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		return tracks, nil // dir not shipped
	}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, err := fs.ReadFile(fsys, path.Join(dir, e.Name()))
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
			continue
		}
		tr, err := parseTrack(b, monthsPerYear, daysPerYear)
		if err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", e.Name(), err))
			continue
		}
		if _, dup := tracks[tr.Name]; dup {
			errs = append(errs, fmt.Errorf("%s: duplicate track %q (first wins)", e.Name(), tr.Name))
			continue
		}
		tracks[tr.Name] = tr
	}
	return tracks, errs
}

// parseTrack parses and validates one track file against the calendar shape.
func parseTrack(b []byte, monthsPerYear, daysPerYear int) (Track, error) {
	var tf trackFile
	if err := yaml.Unmarshal(b, &tf); err != nil {
		return Track{}, err
	}
	if tf.Track == "" {
		return Track{}, fmt.Errorf("missing required 'track' key")
	}
	if monthsPerYear < 1 || daysPerYear < 1 {
		return Track{}, fmt.Errorf("invalid calendar shape (%d months, %d days)", monthsPerYear, daysPerYear)
	}

	tr := Track{Name: tf.Track, monthsPerYear: monthsPerYear, daysPerYear: daysPerYear}
	claimed := make(map[int]string, monthsPerYear) // month -> season name

	for _, s := range tf.Seasons {
		if s.Name == "" {
			return Track{}, fmt.Errorf("track %q: a season is missing a name", tf.Track)
		}
		if len(s.Months) == 0 {
			return Track{}, fmt.Errorf("track %q: season %q claims no months", tf.Track, s.Name)
		}
		for _, m := range s.Months {
			if m < 1 || m > monthsPerYear {
				return Track{}, fmt.Errorf("track %q: season %q month %d outside 1..%d", tf.Track, s.Name, m, monthsPerYear)
			}
			if prev, dup := claimed[m]; dup {
				return Track{}, fmt.Errorf("track %q: month %d claimed twice (%s and %s)", tf.Track, m, prev, s.Name)
			}
			claimed[m] = s.Name
		}
		start, contiguous := contiguousStart(s.Months, monthsPerYear)
		if !contiguous {
			return Track{}, fmt.Errorf("track %q: season %q months must form one contiguous run (modulo the year)", tf.Track, s.Name)
		}
		spawnMult := 1.0
		if s.SpawnWeightMultiplier != nil {
			spawnMult = *s.SpawnWeightMultiplier
		}
		baseScale := 1.0
		if s.BaseWeightScale != nil {
			baseScale = *s.BaseWeightScale
		}
		if baseScale < 0 {
			return Track{}, fmt.Errorf("track %q: season %q baseWeightScale is negative", tf.Track, s.Name)
		}
		mults := make(map[sim.WeatherType]float64, len(s.WeatherWeightMultipliers))
		for k, v := range s.WeatherWeightMultipliers {
			if v < 0 {
				return Track{}, fmt.Errorf("track %q: season %q multiplier for %q is negative", tf.Track, s.Name, k)
			}
			mults[sim.WeatherType(k)] = v
		}
		adds := make(map[sim.WeatherType]float64, len(s.WeatherWeightAdditions))
		for k, v := range s.WeatherWeightAdditions {
			if v < 0 {
				return Track{}, fmt.Errorf("track %q: season %q addition for %q is negative", tf.Track, s.Name, k)
			}
			adds[sim.WeatherType(k)] = v
		}
		tr.Seasons = append(tr.Seasons, Season{
			Name:                     s.Name,
			Months:                   append([]int(nil), s.Months...),
			TransitionDays:           max(0, s.TransitionDays),
			WeatherWeightMultipliers: mults,
			WeatherWeightAdditions:   adds,
			BaseWeightScale:          baseScale,
			SpawnWeightMultiplier:    spawnMult,
			Influence: sim.WeatherInfluence{
				IntensityDelta:     s.Influence.IntensityDelta,
				MoistureDelta:      s.Influence.MoistureDelta,
				MovementResistance: s.Influence.MovementResistance,
			},
			startMonth: start,
		})
	}
	if len(tr.Seasons) == 0 {
		return Track{}, fmt.Errorf("track %q: no seasons defined", tf.Track)
	}
	for m := 1; m <= monthsPerYear; m++ {
		if _, ok := claimed[m]; !ok {
			return Track{}, fmt.Errorf("track %q: month %d claimed by no season", tf.Track, m)
		}
	}
	return tr, nil
}

// contiguousStart reports whether months form one contiguous run modulo the
// year, and returns the run's first month. A run wraps when month n is
// followed by month 1 (e.g. [12 1 2]).
func contiguousStart(months []int, monthsPerYear int) (start int, ok bool) {
	in := make(map[int]bool, len(months))
	for _, m := range months {
		in[m] = true
	}
	if len(in) == monthsPerYear { // whole year = trivially contiguous
		return months[0], true
	}
	// The start month is the unique claimed month whose predecessor is unclaimed.
	start = 0
	for m := range in {
		prev := m - 1
		if prev < 1 {
			prev = monthsPerYear
		}
		if !in[prev] {
			if start != 0 {
				return 0, false // two run-starts = not contiguous
			}
			start = m
		}
	}
	return start, start != 0
}
