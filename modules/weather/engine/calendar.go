package engine

import (
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/modules/weather/seasons"
)

// DOGMud's gametime hardcodes a 365-day, 12-month year: month =
// 1 + floor(day*24/730), where 730 = 365*24/12 (gametime.go ReCalculate).
// There is no named-calendar config (upstream GoMud added one; DOGMud did
// not), so the calendar shape is constant. The seasons package's monthOfDay
// reduces to the same 730-hour month at (12, 365), so season month claims
// agree with the in-game `time`/date.
const (
	dogmudMonthsPerYear = 12
	dogmudDaysPerYear   = 365
)

// CalendarShape reports the active calendar's (monthsPerYear, daysPerYear) —
// the shape seasons.Load validates track files against.
func CalendarShape() (monthsPerYear, daysPerYear int) {
	return dogmudMonthsPerYear, dogmudDaysPerYear
}

// CalendarNow is the current calendar position for season resolution.
// GameDate.Day is the 1-based day-of-year (gametime subtracts whole years).
func CalendarNow() seasons.CalendarPos {
	return seasons.CalendarPos{DayOfYear: gametime.GetDate().Day, DaysPerYear: dogmudDaysPerYear}
}
