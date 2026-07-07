package hooks

import "testing"

func TestComputeCorpseOwners_Solo(t *testing.T) {
	// Single damager, not in a party -> just that user.
	got := computeCorpseOwners(map[int]int{7: 100}, 0)
	if len(got) != 1 || got[0] != 7 {
		t.Fatalf("computeCorpseOwners(solo) = %v, want [7]", got)
	}
}

func TestComputeCorpseOwners_Empty(t *testing.T) {
	// Mob/environment kill: no player damage -> nobody owns (anyone loots).
	got := computeCorpseOwners(map[int]int{}, 0)
	if len(got) != 0 {
		t.Fatalf("computeCorpseOwners(empty) = %v, want []", got)
	}
}

func TestLootTimeoutRound_Advances(t *testing.T) {
	// The helper's job is to advance `now` by the configured real-time
	// duration. We assert with an explicit real-time unit ("4 real minutes")
	// because that is the intent behind CorpseLootTimeout (its config comment
	// reads "Real-time duration ...").
	//
	// NOTE / FINDING: the Task 6 default value is the bare string "4 minutes",
	// which gametime.AddPeriod parses as GAME-time minutes, not real minutes.
	// At the default RoundsPerDay (20) that floors to a 0-round advance
	// (lootTimeoutRound(1, "4 minutes") == 1), and even at prod RoundsPerDay
	// (900) it is only ~2 rounds (~8 real seconds) — nowhere near 4 real
	// minutes. The default should be "4 real minutes" for the corpse-loot
	// timeout to behave as documented. That fix belongs to the config (Task 6),
	// not this helper, so it is flagged here rather than worked around.
	if r := lootTimeoutRound(1, "4 real minutes"); r <= 1 {
		t.Fatalf("lootTimeoutRound(1, \"4 real minutes\") = %d, want > 1", r)
	}
}
