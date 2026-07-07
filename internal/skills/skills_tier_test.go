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
	// Demigod: raw rank-total >= 1200. 16 skills * 75 = 1200.
	if got := GetSkillTier(allSkillRanks(75)); got != "demigod" {
		t.Fatalf("16 skills at 75 (total 1200) should be demigod, got %q", got)
	}

	// Grandmaster: every profession mastered (all at 50) but below the demigod
	// total (16*50 = 800 < 1200).
	if got := GetSkillTier(allSkillRanks(50)); got != "grandmaster" {
		t.Fatalf("all skills at 50 should be grandmaster, got %q", got)
	}

	// Not-all-mastered (one skill below 50) and under the demigod total → falls to
	// the pct ladder. 15 skills at 60 = 900 (pct ~1.06) → master.
	gm := allSkillRanks(60)
	for skill := range SkillPrimaryStats {
		gm[skill] = 0 // drop exactly one skill below master
		break
	}
	if got := GetSkillTier(gm); got != "master" {
		t.Fatalf("not-all-mastered, under 1200 total should be master, got %q", got)
	}

	// Master: ~0.60 pct, not all mastered — 10 skills at 51 ≈ 510/850.
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
