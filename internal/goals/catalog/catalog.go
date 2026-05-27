// Package catalog registers the chunk-4.3 goal-type catalog with the
// goals package. Each <type>.go file's init() registers one type via
// goals.RegisterGoalType. Main.go pulls these registrations in via a
// blank import:
//
//	import _ "github.com/GoMudEngine/GoMud/internal/goals/catalog"
//
// Types in this catalog:
//
//	survival, wealth-gold, wealth-item, craft-item,
//	revenge-mob, revenge-faction, protection-mob, protection-faction,
//	befriend, befriend-faction, mastery-skill, mastery-equip,
//	visit-zone
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.3-goal-types-catalog-design.md.
package catalog
