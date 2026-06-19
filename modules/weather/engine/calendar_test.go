package engine

import "testing"

// DOGMud's calendar is a fixed 12-month/365-day year (no named-calendar
// config). The seasons math relies on this shape.
func TestCalendarShape_DOGMud(t *testing.T) {
	m, d := CalendarShape()
	if m != 12 || d != 365 {
		t.Fatalf("CalendarShape() = (%d,%d), want (12,365)", m, d)
	}
}

// CalendarNow must report the fixed days-per-year and a non-negative
// day-of-year sourced from gametime.
func TestCalendarNow_DOGMud(t *testing.T) {
	pos := CalendarNow()
	if pos.DaysPerYear != 365 {
		t.Fatalf("CalendarNow().DaysPerYear = %d, want 365", pos.DaysPerYear)
	}
	if pos.DayOfYear < 0 {
		t.Fatalf("CalendarNow().DayOfYear = %d, want >= 0", pos.DayOfYear)
	}
}
