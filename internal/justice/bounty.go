package justice

import (
	"fmt"
	"math"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const hostileRepFloor = 50 // |rep| at the Hostile tier boundary (rep <= -50)

// bountyGold = powerBase * max(crimeMult, repMult). crimeMult is murderMult on
// an identified murder, else 1.0. repMult ramps 1.0 (rep -50) -> repMultMax
// (rep -100); 1.0 if not yet Hostile.
func bountyGold(powerBase int, isMurder bool, rep int, murderMult, repMultMax float64) int {
	crimeMult := 1.0
	if isMurder {
		crimeMult = murderMult
	}
	repMult := 1.0
	if absRep := -rep; absRep > hostileRepFloor {
		frac := float64(absRep-hostileRepFloor) / float64(100-hostileRepFloor)
		if frac > 1 {
			frac = 1
		}
		repMult = 1.0 + frac*(repMultMax-1.0)
	}
	mult := crimeMult
	if repMult > mult {
		mult = repMult
	}
	return int(math.Round(float64(powerBase) * mult))
}

// shouldDeclare gates auto-bounty creation: skip if one is already open;
// otherwise declare on an identified murder OR Hostile rep.
func shouldDeclare(triggerKind crimes.Kind, tier opinions.Tier, alreadyOpen bool) bool {
	if alreadyOpen {
		return false
	}
	return triggerKind == crimes.KindMurder || tier == opinions.TierHostile
}

// Seams (production wiring; tests override).
var (
	bDefaultGoldFn = bounties.DefaultGoldFor
	bRepFn         = factions.GetRep
	bTierFn        = func(faction string, userId int) opinions.Tier {
		return factions.TierFor(faction, userId)
	}
	bDeclareFn      = bounties.Declare
	bExistingFn     = existingFactionBounty
	bExpiryRoundsFn = func() uint64 {
		v := configs.GetBalanceConfig().JusticeBountyExpiryRounds
		if v < 1 {
			return 5000
		}
		return uint64(v)
	}
	bMurderMultFn = func() float64 {
		return float64(configs.GetBalanceConfig().JusticeBountyMurderMult)
	}
	bRepMultMaxFn = func() float64 {
		return float64(configs.GetBalanceConfig().JusticeBountyRepMultMax)
	}
	bNowFn = nowRoundFn // reuse justice.go's round seam
)

// existingFactionBounty reports whether an open bounty issued by `faction`
// already targets the player.
func existingFactionBounty(faction string, userId int) bool {
	for _, b := range bounties.OpenAgainstPlayer(userId) {
		if b.Issuer.Type == bounties.IssuerFaction && b.Issuer.Id == faction {
			return true
		}
	}
	return false
}

// MaybeDeclareBounty posts a town-faction kill-bounty on a player when an
// identified murder or Hostile rep warrants it. Idempotent per (faction,
// player). Called from the crime-recording sites after their rep hit.
func MaybeDeclareBounty(faction string, userId int, triggerKind crimes.Kind) {
	tier := bTierFn(faction, userId)
	if !shouldDeclare(triggerKind, tier, bExistingFn(faction, userId)) {
		return
	}
	gold := bountyGold(
		bDefaultGoldFn(knowledge.PlayerSubject(userId)),
		triggerKind == crimes.KindMurder,
		bRepFn(faction, userId),
		bMurderMultFn(), bRepMultMaxFn(),
	)
	reason := fmt.Sprintf("Crimes against %s", faction)
	if triggerKind == crimes.KindMurder {
		reason = fmt.Sprintf("Murder (faction %s)", faction)
	}
	_, _ = bDeclareFn(
		bounties.FactionIssuer(faction),
		knowledge.PlayerSubject(userId),
		bounties.ConditionKill,
		bNowFn()+bExpiryRoundsFn(),
		bounties.DeclareOpts{GoldOverride: gold, DeclaredReason: reason},
	)
}
