package seasons

import (
	"math"
	"testing"
	"testing/fstest"
)

// Stock calendar: 365 days, 12 months, hoursPerMonth = 365*24/12 = 730.
// Engine month formula: month = 1 + floor(day*24/730), clamped to 12.
// Month 1 = days 1..30, month 12 = days 335..365 (clamp).
func TestMonthOfDayMirrorsEngine(t *testing.T) {
	cases := []struct{ day, want int }{
		{1, 1}, {30, 1}, {31, 2}, {304, 10}, {305, 11}, {334, 11}, {335, 12}, {365, 12},
	}
	for _, c := range cases {
		if got := monthOfDay(c.day, 365, 12); got != c.want {
			t.Errorf("day %d: month %d, want %d", c.day, got, c.want)
		}
	}
}

func TestResolveSeasonsAndBlend(t *testing.T) {
	tr := loadOne(t, temperateYAML)["temperate"]

	// Mid-summer (month 7 ≈ days 183..212): fully summer.
	cur, prev, blend := tr.Resolve(200)
	if cur != "summer" || prev != "spring" || blend != 1.0 {
		t.Errorf("day 200: %s/%s/%v", cur, prev, blend)
	}

	// Winter starts at month 12 (day 335). transitionDays=6:
	// day 335 -> daysInto 0 -> blend 0.0 (continuous with autumn).
	cur, prev, blend = tr.Resolve(335)
	if cur != "winter" || prev != "autumn" || blend != 0.0 {
		t.Errorf("day 335: %s/%s/%v", cur, prev, blend)
	}
	// day 338 -> daysInto 3 -> blend 0.5.
	if _, _, b := tr.Resolve(338); math.Abs(b-0.5) > 1e-9 {
		t.Errorf("day 338 blend: %v", b)
	}
	// day 341+ -> blend 1.0.
	if _, _, b := tr.Resolve(341); b != 1.0 {
		t.Errorf("day 341 blend: %v", b)
	}

	// Wraparound: winter spans 12,1,2 — day 10 is still winter, fully blended
	// (daysInto = 10-1 + (365-335+1) = 40 >= 6).
	cur, prev, blend = tr.Resolve(10)
	if cur != "winter" || prev != "autumn" || blend != 1.0 {
		t.Errorf("day 10: %s/%s/%v", cur, prev, blend)
	}

	// Spring starts month 3 (day 61): blend 0 at start, prev = winter.
	cur, prev, blend = tr.Resolve(61)
	if cur != "spring" || prev != "winter" || blend != 0.0 {
		t.Errorf("day 61: %s/%s/%v", cur, prev, blend)
	}
}

// Non-12-month calendars must resolve identically to the engine's month math
// (hoursPerMonth = 100*24/4 = 600; month m starts at the smallest day d with
// d*24 >= (m-1)*600, i.e. days 1, 25, 50, 75 — exact divisors land ON the
// boundary).
func TestResolveNonTwelveMonthCalendar(t *testing.T) {
	yaml := "track: q\nseasons:\n  - name: a\n    months: [4, 1]\n    transitionDays: 4\n  - name: b\n    months: [2, 3]\n    transitionDays: 4\n"
	fsys := fstest.MapFS{"seasons/q.yaml": {Data: []byte(yaml)}}
	tracks, errs := Load(fsys, "seasons", 4, 100)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	tr := tracks["q"]

	if got := monthOfDay(24, 100, 4); got != 1 {
		t.Errorf("day 24: month %d, want 1", got)
	}
	if got := monthOfDay(25, 100, 4); got != 2 {
		t.Errorf("day 25: month %d, want 2", got)
	}
	// Season a wraps (months 4,1): day 5 is inside it, started day 75 ->
	// daysInto = 5-75+100 = 30 -> fully blended.
	cur, prev, blend := tr.Resolve(5)
	if cur != "a" || prev != "b" || blend != 1.0 {
		t.Errorf("day 5: %s/%s/%v", cur, prev, blend)
	}
	// Season b starts day 25; day 27 -> daysInto 2 -> blend 0.5.
	cur, prev, blend = tr.Resolve(27)
	if cur != "b" || prev != "a" || blend != 0.5 {
		t.Errorf("day 27: %s/%s/%v", cur, prev, blend)
	}
	// Boundary day itself: blend 0 (continuous with season a).
	cur, _, blend = tr.Resolve(25)
	if cur != "b" || blend != 0.0 {
		t.Errorf("day 25: %s blend %v, want b/0", cur, blend)
	}
}

func TestResolveHardFlip(t *testing.T) {
	yaml := "track: t\nseasons:\n  - name: a\n    months: [1,2,3,4,5,6]\n  - name: b\n    months: [7,8,9,10,11,12]\n"
	tr := loadOne(t, yaml)["t"]
	// transitionDays 0 => blend 1.0 immediately on the boundary day.
	// Month 7's first day: smallest d with 1+floor(d*24/730) == 7, i.e.
	// d*24/730 >= 6 → d >= 182.5 → day 183.
	if cur, _, blend := tr.Resolve(183); cur != "b" || blend != 1.0 {
		t.Errorf("hard flip: %s/%v", cur, blend)
	}
}

func TestSingleSeasonTrack(t *testing.T) {
	yaml := "track: t\nseasons:\n  - name: always\n    months: [1,2,3,4,5,6,7,8,9,10,11,12]\n"
	tr := loadOne(t, yaml)["t"]
	cur, prev, blend := tr.Resolve(100)
	if cur != "always" || prev != "always" || blend != 1.0 {
		t.Errorf("single season: %s/%s/%v", cur, prev, blend)
	}
}
