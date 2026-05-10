package facts

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

func TestDeclare_HappyPath(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	err := Declare("king-dead", DeclareOpts{
		Description:  "The king is dead.",
		Significance: worldevents.Global,
		Tags:         []string{"politics", "death"},
	})
	if err != nil {
		t.Fatalf("Declare returned error: %v", err)
	}

	f := GetFact("king-dead")
	if f == nil {
		t.Fatalf("GetFact returned nil")
	}
	if f.Status != StatusActive {
		t.Errorf("status not active: %s", f.Status)
	}
	if f.DeclaredRound != 100 {
		t.Errorf("DeclaredRound mismatch: %d", f.DeclaredRound)
	}
}

func TestDeclare_CollisionRejected(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	if err := Declare("foo", DeclareOpts{Description: "first"}); err != nil {
		t.Fatalf("first Declare: %v", err)
	}
	if err := Declare("foo", DeclareOpts{Description: "second"}); err == nil {
		t.Errorf("collision should return error")
	}
}

func TestWithdraw(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("foo", DeclareOpts{Description: "t"})
	Withdraw("foo")
	if f := GetFact("foo"); f.Status != StatusWithdrawn {
		t.Errorf("expected withdrawn, got %s", f.Status)
	}
	// Idempotent.
	Withdraw("foo")
	if f := GetFact("foo"); f.Status != StatusWithdrawn {
		t.Errorf("idempotent expected, got %s", f.Status)
	}
}

func TestPruneExpired(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", ExpiryRound: 200})
	Declare("b", DeclareOpts{Description: "b"}) // never expires
	Declare("c", DeclareOpts{Description: "c", ExpiryRound: 5000})

	roundForTest = func() uint64 { return 300 }
	count := PruneExpired()
	if count != 1 {
		t.Errorf("expected 1 pruned, got %d", count)
	}
	if GetFact("a").Status != StatusExpired {
		t.Errorf("a should be expired")
	}
	if GetFact("b").Status != StatusActive {
		t.Errorf("b should still be active (never expires)")
	}
	if GetFact("c").Status != StatusActive {
		t.Errorf("c should still be active (future expiry)")
	}
}

func TestWithdrawAllBoundTo(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", WithdrawOnRespawnOf: 500})
	Declare("b", DeclareOpts{Description: "b", WithdrawOnRespawnOf: 500})
	Declare("c", DeclareOpts{Description: "c"})

	count := WithdrawAllBoundTo(500)
	if count != 2 {
		t.Errorf("expected 2 withdrawn, got %d", count)
	}
	if GetFact("a").Status != StatusWithdrawn || GetFact("b").Status != StatusWithdrawn {
		t.Errorf("a/b should be withdrawn")
	}
	if GetFact("c").Status != StatusActive {
		t.Errorf("c should be active")
	}
}

func TestRegistryReads(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	Declare("a", DeclareOpts{Description: "a", Tags: []string{"politics"}})
	Declare("b", DeclareOpts{Description: "b", Tags: []string{"politics", "war"}})
	Declare("c", DeclareOpts{Description: "c", Tags: []string{"travel"}})
	Withdraw("c")

	if got := len(AllActiveFacts()); got != 2 {
		t.Errorf("AllActiveFacts: %d want 2", got)
	}
	if got := len(AllRows()); got != 3 {
		t.Errorf("AllRows: %d want 3", got)
	}
	if got := len(AllFactsByTag("politics")); got != 2 {
		t.Errorf("politics tag count: %d", got)
	}
	if got := len(AllFactsByTag("war")); got != 1 {
		t.Errorf("war tag count: %d", got)
	}
	if got := len(AllFactsByTag("travel")); got != 0 {
		t.Errorf("travel tag (withdrawn): %d, want 0 (active only)", got)
	}
}

func TestRecordHeardEvent_BoundedFIFO(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil; heardEventsMaxForTest = nil }()
	roundForTest = func() uint64 { return 100 }
	heardEventsMaxForTest = func() int { return 4 }

	for i := uint64(1); i <= 6; i++ {
		RecordHeardEvent(1141, i)
	}
	a := loadOrLazyInitAwareness(1141, "")
	if len(a.HeardEvents) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(a.HeardEvents))
	}
	want := []uint64{3, 4, 5, 6}
	for i, e := range a.HeardEvents {
		if e != want[i] {
			t.Errorf("entry %d: got %d, want %d", i, e, want[i])
		}
	}
}

func TestHeardEvent(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	if HeardEvent(1142, 42) {
		t.Errorf("HeardEvent should be false for unknown")
	}
	RecordHeardEvent(1142, 42)
	if !HeardEvent(1142, 42) {
		t.Errorf("HeardEvent should be true after Record")
	}
}

func TestRecordHeardEvent_DedupSameId(t *testing.T) {
	resetCaches()
	defer func() { roundForTest = nil }()
	roundForTest = func() uint64 { return 100 }

	RecordHeardEvent(1143, 42)
	RecordHeardEvent(1143, 42) // duplicate
	a := loadOrLazyInitAwareness(1143, "")
	if len(a.HeardEvents) != 1 {
		t.Errorf("duplicate event id should dedupe, got %d", len(a.HeardEvents))
	}
}
