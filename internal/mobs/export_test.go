package mobs

// This file re-exports package-private symbols for tests within the mobs
// package. Cross-package test injection uses RegisterScheduleForTest and
// UnregisterScheduleForTest, which are exported from schedule.go.
//
// Nothing to add here currently — kept as a placeholder for future
// white-box test exports.
