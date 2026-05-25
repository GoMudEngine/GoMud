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
