package usercommands

import "testing"

func TestForageCore_UnknownBiomeReturnsEmpty(t *testing.T) {
	r := ForageCore(ForageAttempt{Biome: "nonexistent", SearchScore: 1000})
	if r.Found {
		t.Error("expected unknown biome to return Found=false")
	}
	if r.ItemId != 0 {
		t.Errorf("expected ItemId=0 for empty biome, got %d", r.ItemId)
	}
}

func TestForageCore_HighScoreFinds(t *testing.T) {
	// Score 1000 vs forest difficulty 120 — should always find.
	found := false
	for i := 0; i < 50 && !found; i++ {
		r := ForageCore(ForageAttempt{Biome: "forest", SearchScore: 1000})
		if r.Found {
			found = true
			if r.ItemId == 0 {
				t.Errorf("Found=true but ItemId=0")
			}
		}
	}
	if !found {
		t.Error("expected at least one find in 50 attempts at SearchScore 1000")
	}
}

func TestForageCore_LowScoreMisses(t *testing.T) {
	// Score 1 vs any biome difficulty (110+) — should always miss.
	for i := 0; i < 50; i++ {
		r := ForageCore(ForageAttempt{Biome: "forest", SearchScore: 1})
		if r.Found {
			t.Errorf("expected miss at SearchScore=1, got Found=true ItemId=%d", r.ItemId)
		}
	}
}

func TestForageCore_NightYieldsAppendedForForestAtNight(t *testing.T) {
	// At night, moonpetal (40046) is added to forest yields.
	// We can't deterministically force a moonpetal, but we can confirm
	// no out-of-table items appear and many runs eventually hit moonpetal.
	moonpetalSeen := false
	for i := 0; i < 200; i++ {
		r := ForageCore(ForageAttempt{Biome: "forest", SearchScore: 1000, AtNight: true})
		if !r.Found {
			continue
		}
		// Acceptable IDs: forest day (40004, 40005, 40049) + night (40046)
		switch r.ItemId {
		case 40004, 40005, 40049:
			// ok
		case 40046:
			moonpetalSeen = true
		default:
			t.Errorf("unexpected itemId %d in forest-night yields", r.ItemId)
		}
	}
	if !moonpetalSeen {
		t.Error("expected at least one moonpetal (40046) in 200 forest-night attempts")
	}
}
