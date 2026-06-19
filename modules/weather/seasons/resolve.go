package seasons

// monthOfDay mirrors the engine's month formula EXACTLY
// (internal/gametime/gametime.go:268-271): month = 1 + floor(day*24 /
// hoursPerMonth), hoursPerMonth = daysPerYear*24/numMonths, clamped to
// numMonths — so a season's month claims always agree with the 'time' command.
func monthOfDay(dayOfYear, daysPerYear, monthsPerYear int) int {
	hoursPerMonth := float64(daysPerYear) * 24.0 / float64(monthsPerYear)
	m := 1 + int(float64(dayOfYear)*24.0/hoursPerMonth)
	if m > monthsPerYear {
		m = monthsPerYear
	}
	return m
}

// firstDayOfMonth is the smallest day-of-year d with monthOfDay(d) == m.
func firstDayOfMonth(m, daysPerYear, monthsPerYear int) int {
	// Month 1 may be unreachable on degenerate calendars (daysPerYear <= monthsPerYear, where the engine's clamp skips it); "starts at day 1" stays engine-consistent for the prev-season and wrap arithmetic that call this.
	if m <= 1 {
		return 1
	}
	hoursPerMonth := float64(daysPerYear) * 24.0 / float64(monthsPerYear)
	d := int(float64(m-1)*hoursPerMonth/24.0) + 1
	// Guard float edges: walk to the exact boundary.
	for d > 1 && monthOfDay(d-1, daysPerYear, monthsPerYear) >= m {
		d--
	}
	for monthOfDay(d, daysPerYear, monthsPerYear) < m {
		d++
	}
	return d
}

// Resolve reports where this track stands on a day of the year: the current
// season, the season it is transitioning FROM, and the blend factor.
// blend = clamp(daysIntoSeason / transitionDays, 0, 1); 0 on the boundary day
// (odds still fully the previous season's — continuous across the flip), 1
// once the window has elapsed. transitionDays <= 0 means an immediate 1.
func (t Track) Resolve(dayOfYear int) (current, previous string, blend float64) {
	if len(t.Seasons) == 1 {
		s := t.Seasons[0].Name
		return s, s, 1.0
	}
	m := monthOfDay(dayOfYear, t.daysPerYear, t.monthsPerYear)

	curIdx := -1
	for i, s := range t.Seasons {
		for _, sm := range s.Months {
			if sm == m {
				curIdx = i
			}
		}
	}
	cur := t.Seasons[curIdx]

	// Previous season = the one claiming the month before cur's start month.
	prevMonth := cur.startMonth - 1
	if prevMonth < 1 {
		prevMonth = t.monthsPerYear
	}
	prevIdx := curIdx
	for i, s := range t.Seasons {
		for _, sm := range s.Months {
			if sm == prevMonth {
				prevIdx = i
			}
		}
	}

	daysInto := dayOfYear - firstDayOfMonth(cur.startMonth, t.daysPerYear, t.monthsPerYear)
	if daysInto < 0 { // season wraps the year boundary
		daysInto += t.daysPerYear
	}
	blend = 1.0
	if cur.TransitionDays > 0 {
		blend = float64(daysInto) / float64(cur.TransitionDays)
		if blend > 1 {
			blend = 1
		}
	}
	return cur.Name, t.Seasons[prevIdx].Name, blend
}

// season returns the named season (load-time validation guarantees presence).
func (t Track) season(name string) Season {
	for _, s := range t.Seasons {
		if s.Name == name {
			return s
		}
	}
	return Season{SpawnWeightMultiplier: 1.0, BaseWeightScale: 1.0}
}
