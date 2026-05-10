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
