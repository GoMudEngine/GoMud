package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestGiftToOpinionBoost_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["GiftAccepted"] {
		if reg.name == ruleNameGiftToOpinionBoost {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gift_to_opinion_boost not registered for GiftAccepted")
	}
}

func TestGiftValueToOpinionBump_Tiers(t *testing.T) {
	cases := []struct {
		value int
		want  int
	}{
		{0, 0},
		{1, 1},
		{49, 1},
		{50, 3},
		{199, 3},
		{200, 5},
		{999, 5},
		{1000, 8},
		{99999, 8},
	}
	for _, c := range cases {
		got := giftValueToOpinionBump(c.value)
		if got != c.want {
			t.Errorf("value=%d: got %d, want %d", c.value, got, c.want)
		}
	}
}

func TestGiftToOpinionBoost_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	giftToOpinionBoost(events.GiftAccepted{})
}

// End-to-end (item-value lookup + cooldown round-trip) requires live
// items registry + mob fixture; deferred to Task 15 smoke.
