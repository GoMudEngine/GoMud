package sim

// ZoneId names a zone (matches Graph node keys).
type ZoneId = string

// WeatherType is an open, data-driven weather label (climate profiles define
// which are valid per biome). Clear is the calm baseline for frontless zones.
type WeatherType string

const Clear WeatherType = "clear"

// FrontId uniquely identifies a weather front within a run.
type FrontId uint64

// Front is a discrete weather system with a location and a trajectory.
type Front struct {
	Id        FrontId     `json:"id"`
	Type      WeatherType `json:"type"`
	Zone      ZoneId      `json:"zone"`
	Intensity float64     `json:"intensity"` // 0..1; <=0 means death
	Moisture  float64     `json:"moisture"`  // 0..1
	Age       int         `json:"age"`       // ticks alive
	MaxAge    int         `json:"maxAge"`    // soft cap; older fronts decay faster
	History   []ZoneId    `json:"history"`   // recent path (bounded), newest last
}

// State is the full simulation state: the RNG cursor, the front-id counter,
// active fronts, and the resolved per-zone weather. It is serializable.
type State struct {
	Round    uint64                 `json:"round"`
	RNGState uint64                 `json:"rngState"`
	NextID   FrontId                `json:"nextId"`
	Fronts   []Front                `json:"fronts"`
	Weather  map[ZoneId]WeatherType `json:"weather"`
}

// Clock carries the current coarse tick (and, later, season). Step stamps the
// next State with it.
type Clock struct {
	Round uint64 `json:"round"`
}

// ZoneChange records one zone's weather transition for a tick. A From of "" (the
// zero WeatherType) means the zone had no previously-recorded weather — e.g. on
// the first tick — not that it was literally empty; the engine layer should
// treat "" as "unset".
type ZoneChange struct {
	Zone ZoneId      `json:"zone"`
	From WeatherType `json:"from"`
	To   WeatherType `json:"to"`
}

// StateDiff is the set of per-zone weather changes a Step produced — what the
// engine layer applies (and nothing more).
type StateDiff struct {
	Changes []ZoneChange `json:"changes"`
}

// clamp01 constrains x to [0, 1].
func clamp01(x float64) float64 {
	if x < 0 {
		return 0
	}
	if x > 1 {
		return 1
	}
	return x
}
