// Package sim holds the pure, engine-independent core of the weather module:
// the geography Graph that the crawler produces and the weather simulation
// consumes. Nothing in this package may import the GoMud engine
// (internal/*); that rule is enforced by arch_test.go.
package sim
