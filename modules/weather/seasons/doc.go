// Package seasons resolves data-defined season tracks (temperate, monsoon,
// anything a builder writes) against the game calendar and applies them as a
// pure transform over sim.Climate. It is the architecture's regression
// guarantee for v2: sim.Step never changes — it just receives this tick's
// effective climate. Pure: no engine imports (enforced by arch_test.go);
// season state is never persisted because it is always derivable from the
// calendar position.
package seasons
