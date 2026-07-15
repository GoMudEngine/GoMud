package guilds

import "testing"

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
