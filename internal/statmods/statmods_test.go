package statmods

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRegression_AddOnNilMap locks the fix for the 2026-07-20 audit finding:
// Add had a value receiver and "handled" a nil map with
//
//	if s == nil { s = make(StatMods) }
//
// which reassigns the local parameter. The new map never escaped to the caller,
// so Add on a nil StatMods silently discarded the bonus — no panic, no error,
// the stat modifier simply vanished.
//
// Every live caller happened to pre-allocate, so this never fired in
// production, but nothing enforced that for the next enchant/affix feature.
func TestRegression_AddOnNilMap(t *testing.T) {
	t.Run("add_to_nil_map_persists", func(t *testing.T) {
		var m StatMods // nil
		require.Nil(t, m, "precondition: map starts nil")

		m.Add("strength", 5)

		assert.Equal(t, 5, m.Get("strength"),
			"Add on a nil StatMods must persist the value, not discard it")
	})

	t.Run("add_to_nil_map_via_struct_field", func(t *testing.T) {
		// The shape every real caller uses: spec.StatMods.Add(...) where the
		// field was never initialised.
		type spec struct{ StatMods StatMods }
		s := &spec{}

		s.StatMods.Add("dexterity", 3)

		assert.Equal(t, 3, s.StatMods.Get("dexterity"),
			"Add through an uninitialised struct field must persist")
	})
}

func TestStatMods_Add(t *testing.T) {
	t.Run("accumulates_across_calls", func(t *testing.T) {
		m := StatMods{}
		m.Add("strength", 2)
		m.Add("strength", 3)

		assert.Equal(t, 5, m.Get("strength"), "repeated Add calls must accumulate")
	})

	t.Run("tracks_stats_independently", func(t *testing.T) {
		m := StatMods{}
		m.Add("strength", 2)
		m.Add("dexterity", 7)

		assert.Equal(t, 2, m.Get("strength"))
		assert.Equal(t, 7, m.Get("dexterity"))
	})

	t.Run("negative_values_subtract", func(t *testing.T) {
		m := StatMods{}
		m.Add("strength", 5)
		m.Add("strength", -8)

		assert.Equal(t, -3, m.Get("strength"), "negative modifiers must apply")
	})
}

func TestStatMods_Get(t *testing.T) {
	t.Run("unknown_stat_returns_zero", func(t *testing.T) {
		m := StatMods{"strength": 4}

		assert.Equal(t, 0, m.Get("nonexistent"),
			"an unknown stat must read as 0, not panic")
	})

	t.Run("nil_map_returns_zero", func(t *testing.T) {
		var m StatMods

		assert.Equal(t, 0, m.Get("strength"), "a nil map must read as 0, not panic")
	})

	t.Run("sums_multiple_stats", func(t *testing.T) {
		m := StatMods{"strength": 2, "dexterity": 3, "vitality": 10}

		assert.Equal(t, 5, m.Get("strength", "dexterity"),
			"Get must sum every named stat")
	})

	t.Run("sum_skips_unknown_names", func(t *testing.T) {
		m := StatMods{"strength": 2}

		assert.Equal(t, 2, m.Get("strength", "nonexistent"),
			"unknown names contribute 0 rather than breaking the sum")
	})

	t.Run("no_names_returns_zero", func(t *testing.T) {
		m := StatMods{"strength": 2}

		assert.Equal(t, 0, m.Get(), "Get with no arguments sums nothing")
	})
}
