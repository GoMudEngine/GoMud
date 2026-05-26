package conversations

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/relationships"
)

func TestGetPool_EmptyTypeReturnsNil(t *testing.T) {
	if got := GetPool(relationships.Type("")); got != nil {
		t.Errorf("expected nil for empty type, got %+v", got)
	}
}

func TestGetPool_UnknownReturnsNil(t *testing.T) {
	if got := GetPool(relationships.Type("not-a-real-type")); got != nil {
		t.Errorf("expected nil for unknown type, got %+v", got)
	}
}

func TestGetPairOverride_OrderIndependent(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{{Lines: []ConversationLine{{Speaker: "A", Text: "x"}}}},
	})
	defer unregisterTestPool("friend")
	registerTestPairOverride(&PairOverride{
		Id: "test_pair", MobA: 100, MobB: 200,
		Exchanges: []Exchange{{Lines: []ConversationLine{{Speaker: "A", Text: "hi"}}}},
	})
	defer unregisterTestPairOverride(100, 200)

	got1 := GetPairOverride(100, 200)
	got2 := GetPairOverride(200, 100)
	if got1 == nil || got2 == nil {
		t.Fatalf("expected lookup to work in both orders, got %v / %v", got1, got2)
	}
	if got1 != got2 {
		t.Errorf("expected order-independent lookup to return same pointer")
	}
}

func TestPairKey_Normalizes(t *testing.T) {
	k1 := makePairKey(100, 200)
	k2 := makePairKey(200, 100)
	if k1 != k2 {
		t.Errorf("pair keys should normalize: %+v != %+v", k1, k2)
	}
	if k1.LowId != 100 || k1.HighId != 200 {
		t.Errorf("expected LowId=100 HighId=200, got %+v", k1)
	}
}

func TestPickExchange_TypePoolOnly(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "one"}, {Speaker: "B", Text: "1"}}},
			{Lines: []ConversationLine{{Speaker: "A", Text: "two"}, {Speaker: "B", Text: "2"}}},
		},
	})
	defer unregisterTestPool("friend")

	// No subtype, no pair override.
	ex, ok := pickExchange("friend", "", 100, 200)
	if !ok {
		t.Fatalf("expected picker to return an exchange")
	}
	if len(ex.Lines) == 0 {
		t.Errorf("picked exchange has no lines")
	}
}

func TestPickExchange_SubtypeAdditive(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "base"}, {Speaker: "B", Text: "."}}},
		},
		Subtypes: map[string]Subpool{
			"fond": {Exchanges: []Exchange{
				{Lines: []ConversationLine{{Speaker: "A", Text: "fond1"}, {Speaker: "B", Text: "."}}},
				{Lines: []ConversationLine{{Speaker: "A", Text: "fond2"}, {Speaker: "B", Text: "."}}},
			}},
		},
	})
	defer unregisterTestPool("friend")

	// Across many picks, both "base" and a "fond*" line should appear.
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		ex, ok := pickExchange("friend", "fond", 100, 200)
		if !ok {
			t.Fatalf("expected pick to succeed")
		}
		seen[ex.Lines[0].Text] = true
	}
	if !seen["base"] {
		t.Errorf("expected base exchange to be picked at least once across 50 draws")
	}
	if !(seen["fond1"] || seen["fond2"]) {
		t.Errorf("expected fond subtype exchange to be picked at least once")
	}
}

func TestPickExchange_PairOverrideAdditive(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "generic"}, {Speaker: "B", Text: "."}}},
		},
	})
	defer unregisterTestPool("friend")
	registerTestPairOverride(&PairOverride{
		Id: "pair", MobA: 100, MobB: 200,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "pair_only"}, {Speaker: "B", Text: "."}}},
		},
	})
	defer unregisterTestPairOverride(100, 200)

	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		ex, ok := pickExchange("friend", "", 100, 200)
		if !ok {
			t.Fatalf("expected pick to succeed")
		}
		seen[ex.Lines[0].Text] = true
	}
	if !seen["generic"] {
		t.Errorf("expected generic pool exchange to be picked across 50 draws")
	}
	if !seen["pair_only"] {
		t.Errorf("expected pair-override exchange to be picked across 50 draws")
	}
}

func TestPickExchange_NoPoolReturnsFalse(t *testing.T) {
	_, ok := pickExchange("friend", "", 100, 200)
	if ok {
		t.Errorf("expected false when no pool registered")
	}
}
