package hooks

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// mkRaiseCorpse builds a mob corpse for raise-selection tests. training feeds
// the stat-pool metric (one stat carries it); gold>0 makes HasLoot() true.
func mkRaiseCorpse(name string, charmed bool, gold, training int) rooms.Corpse {
	c := rooms.Corpse{MobId: 1, WasCharmed: charmed}
	c.Character.Name = name
	c.Character.Stats.Strength.Training = training
	if gold > 0 {
		c.Loot.Gold = gold
	}
	return c
}

func TestSelectRaiseCorpse_SkipsLootHolder_WithReason(t *testing.T) {
	corpses := []rooms.Corpse{mkRaiseCorpse("goblin", false, 50, 300)} // still holds gold
	idx, _, reason := selectRaiseCorpse(corpses, "", 1, 0)
	if idx != -1 {
		t.Fatalf("idx=%d, want -1 (loot-holder skipped)", idx)
	}
	if reason != raiseHasLoot {
		t.Errorf("reason=%v, want raiseHasLoot", reason)
	}
}

func TestSelectRaiseCorpse_PicksValidAmongSkipped(t *testing.T) {
	corpses := []rooms.Corpse{
		mkRaiseCorpse("goblin", true, 0, 300),   // former companion -> skip
		mkRaiseCorpse("goblin", false, 10, 300), // still holds loot -> skip
		mkRaiseCorpse("goblin", false, 0, 300),  // valid
	}
	idx, pool, _ := selectRaiseCorpse(corpses, "", 1, 0)
	if idx != 2 {
		t.Errorf("idx=%d, want 2 (the valid corpse)", idx)
	}
	if pool != 300 {
		t.Errorf("pool=%d, want 300", pool)
	}
}

func TestSelectRaiseCorpse_TooWeak(t *testing.T) {
	corpses := []rooms.Corpse{mkRaiseCorpse("goblin", false, 0, 50)} // pool 50 < min 120
	idx, _, reason := selectRaiseCorpse(corpses, "", 1, 120)
	if idx != -1 || reason != raiseTooWeak {
		t.Errorf("idx=%d reason=%v, want -1/raiseTooWeak", idx, reason)
	}
}

func TestSelectRaiseCorpse_NameMismatchGivesNoNearReason(t *testing.T) {
	// A loot-holding corpse whose NAME doesn't match must not set a near-match
	// reason — the player asked for something that isn't here.
	corpses := []rooms.Corpse{mkRaiseCorpse("goblin", false, 50, 300)}
	idx, _, reason := selectRaiseCorpse(corpses, "rat", 1, 0)
	if idx != -1 || reason != raiseNoMatch {
		t.Errorf("idx=%d reason=%v, want -1/raiseNoMatch", idx, reason)
	}
}

func TestSelectRaiseCorpse_LootReasonBeatsCharmed(t *testing.T) {
	corpses := []rooms.Corpse{
		mkRaiseCorpse("goblin", true, 0, 300),   // charmed
		mkRaiseCorpse("goblin", false, 50, 300), // has loot (more actionable)
	}
	_, _, reason := selectRaiseCorpse(corpses, "", 1, 0)
	if reason != raiseHasLoot {
		t.Errorf("reason=%v, want raiseHasLoot (most actionable reason wins)", reason)
	}
}

func TestRaiseFailureMessage_LootIsActionable(t *testing.T) {
	msg := raiseFailureMessage(raiseHasLoot, "")
	if !strings.Contains(strings.ToLower(msg), "loot") {
		t.Errorf("raiseHasLoot message should mention looting first; got %q", msg)
	}
	// The generic no-match keeps distinct named vs unnamed phrasing.
	if raiseFailureMessage(raiseNoMatch, "wolf") == raiseFailureMessage(raiseNoMatch, "") {
		t.Error("named and unnamed no-match messages should differ")
	}
}
