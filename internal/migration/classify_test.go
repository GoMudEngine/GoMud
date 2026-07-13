package migration

import "testing"

// TestClassifyPlayer uses the real §4 anchor profiles (values from
// tools/analyze_players.py over the 34 prod accounts).
func TestClassifyPlayer(t *testing.T) {
	cases := []struct {
		name string
		in   PlayerSignals
		want string
	}{
		// Megalomania: role admin.
		{"megalomania-admin", PlayerSignals{Role: "admin", CraftSum: 206, TopCombat: 103, TopCombatCluster: "colossus"}, "admin"},
		// meirok: 2 companions.
		{"meirok-manifester", PlayerSignals{Companions: 2, Manifestation: 48, SpellbookDepth: 43, WillpowerTrain: 35, TopCombat: 69, TopCombatCluster: "colossus", CraftSum: 382}, "manifester"},
		// Crafters: deep spellbook + high wil, but craftsum dominates.
		{"fyttyn-chrysifier", PlayerSignals{Companions: 1, Manifestation: 37, SpellbookDepth: 41, WillpowerTrain: 128, TopCombat: 72, TopCombatCluster: "stalker", CraftSum: 476}, "chrysifier"},
		{"duard-chrysifier", PlayerSignals{Companions: 1, SpellbookDepth: 41, WillpowerTrain: 93, TopCombat: 73, TopCombatCluster: "stalker", CraftSum: 477}, "chrysifier"},
		{"oriana-chrysifier", PlayerSignals{Companions: 1, SpellbookDepth: 33, WillpowerTrain: 74, TopCombat: 60, TopCombatCluster: "stalker", CraftSum: 417}, "chrysifier"},
		// Deios: caster, low craft.
		{"deios-ethereal", PlayerSignals{SpellbookDepth: 16, WillpowerTrain: 88, TopCombat: 11, TopCombatCluster: "ravener", CraftSum: 36}, "ethereal"},
		// Saphira: combat standout 27.
		{"saphira-ravener", PlayerSignals{SpellbookDepth: 3, TopCombat: 27, TopCombatCluster: "ravener", CraftSum: 9}, "ravener"},
		// Calabe: light — spellbook clears depth but not will-train; no combat standout.
		{"calabe-generalist", PlayerSignals{SpellbookDepth: 8, WillpowerTrain: 30, TopCombat: 6, TopCombatCluster: "colossus", CraftSum: 11}, "generalist"},
		// Newbie/dabbler.
		{"newbie-generalist", PlayerSignals{SpellbookDepth: 2, TopCombat: 1, TopCombatCluster: "zealot", CraftSum: 6}, "generalist"},
		{"aitester-generalist", PlayerSignals{TopCombat: 17, TopCombatCluster: "colossus", CraftSum: 8}, "generalist"},
	}
	for _, c := range cases {
		if got := ClassifyPlayer(c.in); got != c.want {
			t.Errorf("%s: ClassifyPlayer = %q, want %q", c.name, got, c.want)
		}
	}
}
