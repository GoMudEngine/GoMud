package justice

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/opinions"
)

func installSeams(t *testing.T) func() {
	t.Helper()
	origRep, origAllies, origBounty, origCrime, origNow, origLook :=
		repTierFn, alliesFn, openFactionBountyFn, unresolvedCrimeFn, nowRoundFn, lookbackFn
	return func() {
		repTierFn, alliesFn, openFactionBountyFn, unresolvedCrimeFn, nowRoundFn, lookbackFn =
			origRep, origAllies, origBounty, origCrime, origNow, origLook
	}
}

func TestVerdict_NeutralRep_None(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityNone {
		t.Errorf("got %v, want None", got)
	}
}

func TestVerdict_ColdRep_Warn(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierCold }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityWarn {
		t.Errorf("got %v, want Warn", got)
	}
}

// 5.1c: Verdict now yields Arrest (not Attack) for the wanted signals;
// Attack is produced only by the resist fork at enforcement time.
func TestVerdict_HostileRep_Arrest(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierHostile }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityArrest {
		t.Errorf("got %v, want Arrest", got)
	}
}

func TestVerdict_OpenBounty_Arrest(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return true }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityArrest {
		t.Errorf("got %v, want Arrest (bounty)", got)
	}
}

func TestVerdict_AllyCrime_Arrest(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierNeutral }
	alliesFn = func(f string) []string {
		if f == "thornwall_guards" {
			return []string{"thornwall_citizens"}
		}
		return nil
	}
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(faction string, _ int, _ uint64, _ uint64) bool {
		return faction == "thornwall_citizens"
	}
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityArrest {
		t.Errorf("got %v, want Arrest (ally crime)", got)
	}
}

func TestVerdict_EmptyGuardFactions_None(t *testing.T) {
	defer installSeams(t)()
	// repTierFn would say Attack, but with no factions the loop never runs.
	repTierFn = func(string, int) opinions.Tier { return opinions.TierHostile }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict(nil, 17); got != SeverityNone {
		t.Errorf("nil guardFactions: got %v, want None", got)
	}
	if got := Verdict([]string{}, 17); got != SeverityNone {
		t.Errorf("empty guardFactions: got %v, want None", got)
	}
}

func TestVerdict_FriendlyRep_None(t *testing.T) {
	defer installSeams(t)()
	repTierFn = func(string, int) opinions.Tier { return opinions.TierFriendly }
	alliesFn = func(string) []string { return nil }
	openFactionBountyFn = func(int, map[string]bool) bool { return false }
	unresolvedCrimeFn = func(string, int, uint64, uint64) bool { return false }
	nowRoundFn = func() uint64 { return 1000 }
	lookbackFn = func() uint64 { return 1000 }
	if got := Verdict([]string{"thornwall_guards"}, 17); got != SeverityNone {
		t.Errorf("friendly rep: got %v, want None", got)
	}
}
