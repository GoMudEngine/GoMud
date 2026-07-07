package skills

import "testing"

// allSkillRanks builds an allRanks map with every canonical skill set to rank.
func allSkillRanks(rank int) map[string]int {
	m := make(map[string]int, len(SkillPrimaryStats))
	for skill := range SkillPrimaryStats {
		m[skill] = rank
	}
	return m
}

func TestGetSkillTier_GrandmasterAndDemigod(t *testing.T) {
	// Demigod: every profession at the master rank (50).
	if got := GetSkillTier(allSkillRanks(50)); got != "demigod" {
		t.Fatalf("all skills at 50 should be demigod, got %q", got)
	}

	// Grandmaster: very high total (pct >= 0.85) but NOT all mastered — knock one
	// skill below 50 so it isn't demigod.
	gm := allSkillRanks(60) // 16*60 = 960 / 850 = 1.13 pct
	for skill := range SkillPrimaryStats {
		gm[skill] = 0 // drop exactly one skill below master
		break
	}
	if got := GetSkillTier(gm); got != "grandmaster" {
		t.Fatalf("high-total-but-not-all-mastered should be grandmaster, got %q", got)
	}

	// Master still applies below the grandmaster threshold: ~0.6 pct.
	// 10 skills at ~51 ≈ 510/850 = 0.60.
	mid := map[string]int{}
	i := 0
	for skill := range SkillPrimaryStats {
		if i < 10 {
			mid[skill] = 51
		}
		i++
	}
	if got := GetSkillTier(mid); got != "master" {
		t.Fatalf("~0.60 pct should be master, got %q", got)
	}
}
