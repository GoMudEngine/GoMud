package guilds

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestRegistry_CreateAndLookup(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()

	g, err := Create("QC", "Questing Cajuns", 1, "Founder")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if g.Tag != "QC" || !g.IsLeader(1) || len(g.Members) != 1 {
		t.Fatalf("bad new guild: %+v", g)
	}
	if TagForUser(1) != "QC" {
		t.Errorf("TagForUser(1) = %q, want QC", TagForUser(1))
	}
	if _, ok := Get("qc"); !ok {
		t.Error("Get should be case-insensitive")
	}

	// Uniqueness.
	if _, err := Create("QC", "Other", 2, "B"); err == nil {
		t.Error("duplicate tag should fail")
	}
	if _, err := Create("ZZ", "Questing Cajuns", 2, "B"); err == nil {
		t.Error("duplicate name should fail")
	}

	// Add / rank / remove.
	if err := AddMember("QC", 2, "Second"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if TagForUser(2) != "QC" {
		t.Error("member 2 not indexed")
	}
	if err := SetRank("QC", 2, RankOfficer); err != nil {
		t.Fatalf("setrank: %v", err)
	}
	if r, _ := g.MemberRank(2); r != RankOfficer {
		t.Errorf("rank = %q, want officer", r)
	}
	RemoveMember("QC", 2)
	if TagForUser(2) != "" {
		t.Error("member 2 should be de-indexed after removal")
	}

	// Transfer + delete.
	AddMember("QC", 3, "Third")
	if err := TransferLeader("QC", 3); err != nil {
		t.Fatalf("transfer: %v", err)
	}
	if !g.IsLeader(3) {
		t.Error("3 should be leader after transfer")
	}
	Delete("QC")
	if _, ok := Get("QC"); ok {
		t.Error("guild should be gone after delete")
	}
}

func TestRegistry_Treasury(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()

	if _, err := Create("TR", "Treasurers", 1, "Lead"); err != nil {
		t.Fatalf("create: %v", err)
	}

	if err := DepositGold("TR", 500); err != nil {
		t.Fatalf("deposit: %v", err)
	}
	if g, _ := Get("TR"); g.Treasury != 500 {
		t.Errorf("treasury = %d, want 500", g.Treasury)
	}
	// Withdraw more than held should fail and not change the balance.
	if err := WithdrawGold("TR", 900); err == nil {
		t.Error("over-withdraw should fail")
	}
	if g, _ := Get("TR"); g.Treasury != 500 {
		t.Errorf("treasury after failed withdraw = %d, want 500", g.Treasury)
	}
	if err := WithdrawGold("TR", 200); err != nil {
		t.Fatalf("withdraw: %v", err)
	}
	if g, _ := Get("TR"); g.Treasury != 300 {
		t.Errorf("treasury = %d, want 300", g.Treasury)
	}

	// Delegation toggle.
	if err := SetTreasuryDelegated("TR", true); err != nil {
		t.Fatalf("delegate: %v", err)
	}
	if g, _ := Get("TR"); !g.TreasuryDelegated {
		t.Error("delegation flag not set")
	}
}

func TestRegistry_Vault(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()

	if _, err := Create("VA", "Vaulters", 1, "Lead"); err != nil {
		t.Fatalf("create: %v", err)
	}

	it := items.Item{ItemId: 1}
	if err := DonateItem("VA", it); err != nil {
		t.Fatalf("donate: %v", err)
	}
	if g, _ := Get("VA"); len(g.Vault) != 1 {
		t.Fatalf("vault len = %d, want 1", len(g.Vault))
	}
	// Take index 0 back out.
	taken, err := TakeItem("VA", 0)
	if err != nil {
		t.Fatalf("take: %v", err)
	}
	if taken.ItemId != 1 {
		t.Errorf("taken item id = %d, want 1", taken.ItemId)
	}
	if g, _ := Get("VA"); len(g.Vault) != 0 {
		t.Errorf("vault len after take = %d, want 0", len(g.Vault))
	}
	// Out-of-range take errors.
	if _, err := TakeItem("VA", 0); err == nil {
		t.Error("take from empty vault should fail")
	}
}

func TestRegistry_RankTitle(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()
	if _, err := Create("RT", "Ranktitlers", 1, "L"); err != nil {
		t.Fatal(err)
	}
	if err := SetRankTitle("RT", RankOfficer, "Lieutenant"); err != nil {
		t.Fatal(err)
	}
	if g, _ := Get("RT"); g.RankTitle(RankOfficer) != "Lieutenant" {
		t.Errorf("title not set")
	}
	// Reset (empty) removes the key -> default.
	if err := SetRankTitle("RT", RankOfficer, ""); err != nil {
		t.Fatal(err)
	}
	if g, _ := Get("RT"); g.RankTitle(RankOfficer) != "officer" {
		t.Errorf("reset should restore default")
	}
}

func TestRegistry_Invites(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()

	if _, err := Create("IN", "Inviters", 1, "Lead"); err != nil {
		t.Fatalf("create: %v", err)
	}
	if err := AddInvite("IN", 5); err != nil {
		t.Fatalf("addinvite: %v", err)
	}
	g, ok := GuildWithInvite(5)
	if !ok || g.Tag != "IN" {
		t.Fatal("GuildWithInvite should find the invite")
	}
	// Accepting (AddMember) clears the invite.
	if err := AddMember("IN", 5, "Fifth"); err != nil {
		t.Fatalf("add: %v", err)
	}
	if _, ok := GuildWithInvite(5); ok {
		t.Error("invite should clear on join")
	}
}

func TestRegistry_JoinClearsCrossGuildInvites(t *testing.T) {
	defer SetDataDirForTest(t.TempDir())()
	resetRegistry()

	if _, err := Create("AA", "Alpha", 1, "L1"); err != nil {
		t.Fatal(err)
	}
	if _, err := Create("BB", "Beta", 2, "L2"); err != nil {
		t.Fatal(err)
	}
	AddInvite("AA", 9)
	AddInvite("BB", 9)

	if err := AddMember("AA", 9, "Bob"); err != nil {
		t.Fatal(err)
	}
	if g, _ := Get("BB"); g.HasInvite(9) {
		t.Error("joining AA should clear BB's stale invite")
	}
	if _, ok := GuildWithInvite(9); ok {
		t.Error("no pending invites should remain anywhere after joining")
	}
}
