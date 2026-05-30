package bounties

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test seams.
var (
	roundForTest          func() uint64
	goldMultiplierForTest func() float64
	goldFloorForTest      func() int
)

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func goldMultiplier() float64 {
	if goldMultiplierForTest != nil {
		return goldMultiplierForTest()
	}
	return float64(configs.GetBalanceConfig().BountyGoldDefaultMultiplier)
}

func goldFloor() int {
	if goldFloorForTest != nil {
		return goldFloorForTest()
	}
	return int(configs.GetBalanceConfig().BountyGoldFloor)
}

// computeDefaultGold returns floor(statpool * multiplier), with a
// floor of `BountyGoldFloor` so trivial mobs still pay a meaningful
// amount.
func computeDefaultGold(statpool int) int {
	g := int(float64(statpool) * goldMultiplier())
	if floor := goldFloor(); g < floor {
		return floor
	}
	return g
}

// DefaultGoldFor returns the statpool-derived default gold reward for a target,
// exposing the same computation Declare uses (so callers like internal/justice
// can scale it without duplicating the formula).
func DefaultGoldFor(target knowledge.Subject) int {
	return computeDefaultGold(statpoolFor(target))
}

// computeDefaultRep returns max(1, floor(statpool / 100)).
func computeDefaultRep(statpool int) int {
	r := statpool / 100
	if r < 1 {
		return 1
	}
	return r
}

// DeclareOpts carries optional overrides for Declare. Zero values mean
// "use auto-computed defaults".
type DeclareOpts struct {
	GoldOverride   int
	RepOverride    int
	DeclaredReason string
}

// Test seam: replace statpool lookup in unit tests so no real mob/user
// fixtures are needed.
var statpoolForTest func(target knowledge.Subject) int

// statpoolFor resolves the target's effective statpool.
//
//   - Mob targets: uses spec.StatPool (the pre-spawn point budget).
//   - Player targets: sums ValueAdj across all 6 stats for online users.
//     Offline player lookup is not supported in v1; returns 0, which causes
//     Declare to fall back to the gold floor and rep floor of 1.
func statpoolFor(target knowledge.Subject) int {
	if statpoolForTest != nil {
		return statpoolForTest(target)
	}
	switch target.Type {
	case knowledge.SubjectMob:
		spec := mobs.GetMobSpec(mobs.MobId(target.Id))
		if spec == nil {
			return 0
		}
		return spec.StatPool
	case knowledge.SubjectPlayer:
		// Online lookup only. If the player is offline, returns 0 so that
		// the gold floor (BountyGoldFloor) and rep floor (1) still apply.
		u := users.GetByUserId(target.Id)
		if u == nil {
			return 0
		}
		s := u.Character.Stats
		return s.Strength.ValueAdj +
			s.Dexterity.ValueAdj +
			s.Perception.ValueAdj +
			s.Vitality.ValueAdj +
			s.Willpower.ValueAdj +
			s.Charisma.ValueAdj
	}
	return 0
}

// Declare creates a new bounty and persists it. Returns the assigned bounty
// ID. Rewards are auto-computed from the target's statpool unless overridden
// via DeclareOpts.
func Declare(issuer Issuer, target knowledge.Subject, condition Condition,
	expiryRound uint64, opts DeclareOpts) (int, error) {

	r := loadOrLazyInit()
	now := currentRound()
	statpool := statpoolFor(target)

	gold := opts.GoldOverride
	if gold == 0 {
		gold = computeDefaultGold(statpool)
	}
	rep := opts.RepOverride
	if rep == 0 {
		rep = computeDefaultRep(statpool)
	}

	registryMu.Lock()
	id := r.NextId
	r.NextId++
	b := &Bounty{
		Id:             id,
		Issuer:         issuer,
		Target:         target,
		GoldReward:     gold,
		RepReward:      rep,
		Condition:      condition,
		DeclaredRound:  now,
		ExpiryRound:    expiryRound,
		Status:         StatusOpen,
		DeclaredReason: opts.DeclaredReason,
	}
	r.Bounties = append(r.Bounties, b)
	registryMu.Unlock()

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("bounties.Declare: save failed", "id", id, "error", err)
		return id, err
	}
	return id, nil
}

// Get returns the bounty with the given ID, or nil if not found.
func Get(bountyId int) *Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, b := range r.Bounties {
		if b.Id == bountyId {
			return b
		}
	}
	return nil
}

// Withdraw transitions an open bounty to withdrawn status. Idempotent on
// already-withdrawn bounties.
func Withdraw(bountyId int) {
	r := loadOrLazyInit()

	registryMu.Lock()
	mutated := false
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusWithdrawn
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.Withdraw: save failed", "id", bountyId, "error", err)
		}
	}
}

// MarkExpired transitions an open bounty to expired status. Idempotent on
// already-expired bounties.
func MarkExpired(bountyId int) {
	r := loadOrLazyInit()

	registryMu.Lock()
	mutated := false
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusExpired
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.MarkExpired: save failed", "id", bountyId, "error", err)
		}
	}
}

// PruneExpired walks open bounties and flips any past their expiry
// round to status=expired. Returns the count flipped.
func PruneExpired() int {
	r := loadOrLazyInit()
	now := currentRound()

	registryMu.Lock()
	count := 0
	for _, b := range r.Bounties {
		if b.Status != StatusOpen {
			continue
		}
		if b.ExpiryRound == 0 {
			continue // never expires
		}
		if b.ExpiryRound <= now {
			b.Status = StatusExpired
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("bounties.PruneExpired: save failed", "error", err)
		}
	}
	return count
}

// TryClaim records a claim on the given bounty. Returns (bounty, true)
// on success (status was open, now claimed). Returns (nil, false) if
// the bounty was already non-open.
func TryClaim(bountyId int, claimer knowledge.Subject) (*Bounty, bool) {
	r := loadOrLazyInit()
	now := currentRound()

	registryMu.Lock()
	var claimed *Bounty
	for _, b := range r.Bounties {
		if b.Id == bountyId && b.Status == StatusOpen {
			b.Status = StatusClaimed
			b.ClaimedBy = claimer
			b.ClaimedRound = now
			claimed = b
			break
		}
	}
	registryMu.Unlock()

	if claimed == nil {
		return nil, false
	}

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("bounties.TryClaim: save failed", "id", bountyId, "error", err)
	}
	return claimed, true
}

// AllOpen returns all bounties with status=open.
func AllOpen() []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen {
			out = append(out, b)
		}
	}
	return out
}

// OpenForTarget returns all bounties with status=open targeting the given subject.
func OpenForTarget(target knowledge.Subject) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen && b.Target == target {
			out = append(out, b)
		}
	}
	return out
}

// OpenForIssuer returns all bounties with status=open issued by the given issuer.
func OpenForIssuer(issuer Issuer) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Status == StatusOpen && b.Issuer == issuer {
			out = append(out, b)
		}
	}
	return out
}

// OpenAgainstPlayer is a convenience wrapper for OpenForTarget(PlayerSubject(userId)).
func OpenAgainstPlayer(userId int) []*Bounty {
	return OpenForTarget(knowledge.PlayerSubject(userId))
}

// AllRows returns a snapshot of every row in the registry,
// regardless of status. Used by the admin --all listing.
func AllRows() []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, len(r.Bounties))
	copy(out, r.Bounties)
	return out
}

// AllForTarget returns all bounties targeting the given subject. If includeNonOpen is false,
// only status=open bounties are included. If true, all statuses are included.
func AllForTarget(target knowledge.Subject, includeNonOpen bool) []*Bounty {
	r := loadOrLazyInit()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Bounty, 0)
	for _, b := range r.Bounties {
		if b.Target != target {
			continue
		}
		if !includeNonOpen && b.Status != StatusOpen {
			continue
		}
		out = append(out, b)
	}
	return out
}
