package gmcp

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mutations"
)

func TestBuildMutationPayload(t *testing.T) {
	restore := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"chameleon-skin": {
			MutationId:  "chameleon-skin",
			Name:        "Chameleon Skin",
			Description: "Your skin drinks the colors around it.",
		},
	})
	defer restore()

	p, ok := buildMutationPayload(mutations.Gained{
		UserId: 1, MutationId: "chameleon-skin", Rank: 2, IsNew: false,
	})
	if !ok {
		t.Fatal("expected payload for a registered mutation")
	}
	if p.Id != "chameleon-skin" || p.Name != "Chameleon Skin" || p.Rank != 2 || p.IsNew {
		t.Errorf("payload fields wrong: %+v", p)
	}
	if p.Art != "/static/images/mutations/chameleon-skin.png" {
		t.Errorf("art path = %q", p.Art)
	}
	if p.Description == "" {
		t.Error("description must ship")
	}

	if _, ok := buildMutationPayload(mutations.Gained{MutationId: "nope"}); ok {
		t.Error("unknown mutation must not build a payload")
	}
}
