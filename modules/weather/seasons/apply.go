package seasons

import "github.com/GoMudEngine/GoMud/modules/weather/sim"

// CalendarPos is the calendar position the engine resolves from the round.
type CalendarPos struct {
	DayOfYear   int
	DaysPerYear int
}

// ZoneSeason is one zone's resolved season.
type ZoneSeason struct {
	Track  string
	Season string
	Blend  float64
}

func lerp(p, c, blend float64) float64 { return p + (c-p)*blend }

// mult returns a season's multiplier for one weather type (missing = 1.0).
func mult(m map[sim.WeatherType]float64, key sim.WeatherType) float64 {
	if v, ok := m[key]; ok {
		return v
	}
	return 1.0
}

// EffectiveClimate returns a season-adjusted COPY of base for this calendar
// position. Biomes without a track (or with an unknown track) pass through
// unchanged. Per weather type:
//
//	eff(w) = base(w) × lerp(prevScale×prevMult(w), curScale×curMult(w))
//	         + lerp(prevAdd(w), curAdd(w))
//
// Additions may introduce types absent from the base climate (esoteric
// seasons, spec §3.1a) and lerp from/to 0 across the window. Spawn weight is
// multiplied by the blended seasonal multiplier; influence deltas are blended
// then ADDED to the biome's own values; MovementResistance is clamped to
// [0,1] afterwards (the sim treats it as a probability).
func EffectiveClimate(base sim.Climate, tracks Tracks, pos CalendarPos) sim.Climate {
	out := make(sim.Climate, len(base))
	for biome, p := range base {
		tr, ok := tracks[p.Track]
		if p.Track == "" || !ok {
			out[biome] = p // pass-through; maps shared but never written below
			continue
		}
		curName, prevName, blend := tr.Resolve(pos.DayOfYear)
		cur, prev := tr.season(curName), tr.season(prevName)

		np := p // copy struct
		np.Weather = make(map[sim.WeatherType]float64, len(p.Weather)+len(cur.WeatherWeightAdditions)+len(prev.WeatherWeightAdditions))
		for w, weight := range p.Weather {
			factor := lerp(prev.BaseWeightScale*mult(prev.WeatherWeightMultipliers, w),
				cur.BaseWeightScale*mult(cur.WeatherWeightMultipliers, w), blend)
			np.Weather[w] = weight * factor
		}
		// Additions: union of both seasons' addition keys, lerped (missing = 0).
		for w := range prev.WeatherWeightAdditions {
			np.Weather[w] += lerp(prev.WeatherWeightAdditions[w], cur.WeatherWeightAdditions[w], blend)
		}
		for w := range cur.WeatherWeightAdditions {
			if _, done := prev.WeatherWeightAdditions[w]; done {
				continue // already lerped above
			}
			np.Weather[w] += lerp(0, cur.WeatherWeightAdditions[w], blend)
		}
		np.SpawnWeight = p.SpawnWeight * lerp(prev.SpawnWeightMultiplier, cur.SpawnWeightMultiplier, blend)
		np.Influence.IntensityDelta = p.Influence.IntensityDelta + lerp(prev.Influence.IntensityDelta, cur.Influence.IntensityDelta, blend)
		np.Influence.MoistureDelta = p.Influence.MoistureDelta + lerp(prev.Influence.MoistureDelta, cur.Influence.MoistureDelta, blend)
		mr := p.Influence.MovementResistance + lerp(prev.Influence.MovementResistance, cur.Influence.MovementResistance, blend)
		if mr < 0 {
			mr = 0
		} else if mr > 1 {
			mr = 1
		}
		np.Influence.MovementResistance = mr
		out[biome] = np
	}
	return out
}

// ZoneSeasons maps every zone to its resolved season via its biome's track.
// Zones whose biome has no (known) track are absent. Biome resolution uses
// base.For's default fallback, matching how the sim resolves climate.
func ZoneSeasons(g *sim.Graph, base sim.Climate, tracks Tracks, pos CalendarPos) map[sim.ZoneId]ZoneSeason {
	out := map[sim.ZoneId]ZoneSeason{}
	for _, z := range g.Zones() {
		p := base.For(g.Nodes[z].Biome)
		tr, ok := tracks[p.Track]
		if p.Track == "" || !ok {
			continue
		}
		cur, _, blend := tr.Resolve(pos.DayOfYear)
		out[z] = ZoneSeason{Track: tr.Name, Season: cur, Blend: blend}
	}
	return out
}
