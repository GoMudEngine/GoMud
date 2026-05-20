package bounties

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestIssuerHelpers(t *testing.T) {
	if FactionIssuer("thornwall_guards") != (Issuer{Type: IssuerFaction, Id: "thornwall_guards"}) {
		t.Errorf("FactionIssuer mismatch")
	}
	if QuestIssuer(14) != (Issuer{Type: IssuerQuest, Id: "14"}) {
		t.Errorf("QuestIssuer mismatch")
	}
	if NPCIssuer(357) != (Issuer{Type: IssuerNPC, Id: "357"}) {
		t.Errorf("NPCIssuer mismatch")
	}
}

func TestBountyDefaultStatus(t *testing.T) {
	b := &Bounty{
		Id:     1,
		Issuer: FactionIssuer("thornwall_guards"),
		Target: knowledge.PlayerSubject(17),
		Status: StatusOpen,
	}
	if b.Status != StatusOpen {
		t.Errorf("Status default mismatch: got %s", b.Status)
	}
}
