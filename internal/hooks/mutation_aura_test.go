package hooks

import "testing"

func TestAuraRecipients(t *testing.T) {
	got := auraRecipients([]int{1, 2, 3}, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Fatalf("auraRecipients = %v, want [1 3] (owner 2 excluded)", got)
	}
	if len(auraRecipients([]int{5}, 5)) != 0 {
		t.Fatal("a lone owner buffs nobody")
	}
}
