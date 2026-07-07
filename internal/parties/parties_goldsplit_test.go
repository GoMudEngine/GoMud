package parties

import "testing"

// TestParty_SettleGold_EvenSplitWithRemainder verifies SettleGold splits the
// shared gold pool evenly across members, routes the remainder to the leader,
// zeroes the pool, and conserves gold exactly.
func TestParty_SettleGold_EvenSplitWithRemainder(t *testing.T) {
	// Leader user 1 + two accepted members (2, 3).
	p := New(1)
	if p == nil {
		t.Fatal("New(1) returned nil")
	}
	defer p.Disband()

	p.InvitePlayer(2)
	p.AcceptInvite(2)
	p.InvitePlayer(3)
	p.AcceptInvite(3)

	if len(p.UserIds) != 3 {
		t.Fatalf("expected 3 members, got %d (%v)", len(p.UserIds), p.UserIds)
	}

	p.GoldPool = 10
	payouts := p.SettleGold()

	// Pool must be drained.
	if p.GoldPool != 0 {
		t.Errorf("GoldPool after settle = %d, want 0", p.GoldPool)
	}

	// Gold conservation: the payouts must sum to the original pool exactly.
	total := 0
	for _, amt := range payouts {
		total += amt
	}
	if total != 10 {
		t.Errorf("sum of payouts = %d, want 10 (gold not conserved)", total)
	}

	// base = 10/3 = 3, remainder = 1 to the leader.
	if payouts[1] != 4 {
		t.Errorf("leader (user 1) payout = %d, want 4 (base 3 + remainder 1)", payouts[1])
	}
	if payouts[2] != 3 {
		t.Errorf("member (user 2) payout = %d, want 3", payouts[2])
	}
	if payouts[3] != 3 {
		t.Errorf("member (user 3) payout = %d, want 3", payouts[3])
	}
}

// TestParty_SettleGold_EmptyPool verifies a zero pool yields an empty payout
// and leaves the pool at zero.
func TestParty_SettleGold_EmptyPool(t *testing.T) {
	p := New(4)
	if p == nil {
		t.Fatal("New(4) returned nil")
	}
	defer p.Disband()

	p.GoldPool = 0
	payouts := p.SettleGold()

	if len(payouts) != 0 {
		t.Errorf("empty pool payouts = %v, want empty map", payouts)
	}
	if p.GoldPool != 0 {
		t.Errorf("GoldPool = %d, want 0", p.GoldPool)
	}
}

// TestParty_SettleGold_SingleMemberExactSplit verifies a solo party's entire
// pool goes to the sole member (who is also the leader) with no remainder loss.
func TestParty_SettleGold_SingleMemberExactSplit(t *testing.T) {
	p := New(5)
	if p == nil {
		t.Fatal("New(5) returned nil")
	}
	defer p.Disband()

	p.GoldPool = 7
	payouts := p.SettleGold()

	if p.GoldPool != 0 {
		t.Errorf("GoldPool after settle = %d, want 0", p.GoldPool)
	}
	if len(payouts) != 1 {
		t.Fatalf("payouts = %v, want a single entry", payouts)
	}
	if payouts[5] != 7 {
		t.Errorf("sole member (user 5) payout = %d, want 7", payouts[5])
	}
}
