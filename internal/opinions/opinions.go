package opinions

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// TierOf bucket-maps a raw disposition score to its Tier.
func TierOf(score int) Tier {
	switch {
	case score <= -50:
		return TierHostile
	case score <= -15:
		return TierCold
	case score <= 14:
		return TierNeutral
	case score <= 49:
		return TierWarm
	default:
		return TierFriendly
	}
}

// clampScore restricts a score to the [ScoreMin, ScoreMax] window.
func clampScore(s int) int {
	if s < ScoreMin {
		return ScoreMin
	}
	if s > ScoreMax {
		return ScoreMax
	}
	return s
}

// Test-only seams. Production code never sets these — they let the
// unit tests inject a mob template lookup, force the round count,
// and override the half-life without standing up the full mob/config
// stack. Each is consulted via a tiny helper that prefers the seam
// when non-nil.
var (
	defaultProviderForTest func(mobId int) (namesimple string, def int, ok bool)
	roundForTest           func() uint64
	halfLifeForTest        func() uint64
)

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func currentHalfLife() uint64 {
	if halfLifeForTest != nil {
		return halfLifeForTest()
	}
	return uint64(configs.GetBalanceConfig().DispositionDecayHalfLifeRounds)
}

// resolveTemplate returns (namesimple, default_disposition, ok) for
// a mob template. In production it consults internal/mobs; tests
// override via defaultProviderForTest.
func resolveTemplate(mobId int) (string, int, bool) {
	if defaultProviderForTest != nil {
		return defaultProviderForTest(mobId)
	}
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return "", 0, false
	}
	return util.ConvertForFilename(spec.Character.Name), spec.DefaultDisposition, true
}

// loadOrLazyInit returns the cached *MobOpinions for mobId,
// loading from disk on first access. If neither cache nor disk
// has data, an empty MobOpinions seeded from the mob template's
// default disposition is created and cached, but no row is added
// to the inner Opinions map. Returns (nil, "", false) if the mob
// template is unknown.
func loadOrLazyInit(mobId int) (*MobOpinions, string, bool) {
	opinionCacheMu.RLock()
	if mo, ok := opinionCache[mobId]; ok {
		name := nameByMobId[mobId]
		opinionCacheMu.RUnlock()
		return mo, name, true
	}
	opinionCacheMu.RUnlock()

	name, def, ok := resolveTemplate(mobId)
	if !ok {
		return nil, "", false
	}

	if mo := loadFromDisk(mobId, name); mo != nil {
		opinionCacheMu.Lock()
		opinionCache[mobId] = mo
		nameByMobId[mobId] = name
		opinionCacheMu.Unlock()
		return mo, name, true
	}

	mo := &MobOpinions{
		MobId:              mobId,
		DefaultDisposition: def,
		Opinions:           map[int]*Opinion{},
	}
	opinionCacheMu.Lock()
	opinionCache[mobId] = mo
	nameByMobId[mobId] = name
	opinionCacheMu.Unlock()
	return mo, name, true
}

// Get returns the (decay-adjusted) score this NPC has of the given
// user. Returns the NPC's default disposition if no row exists.
// Does not mutate the score or write to disk; lazy cache priming
// may occur on first access for a mobId (an empty MobOpinions is
// inserted into the cache, but no opinion row is added and no file
// is created).
func Get(mobId int, userId int) int {
	mo, _, ok := loadOrLazyInit(mobId)
	if !ok {
		return 0
	}
	op, has := mo.Opinions[userId]
	if !has {
		return mo.DefaultDisposition
	}
	return decayedScore(op.Score, mo.DefaultDisposition,
		op.LastUpdatedRound, currentRound(), currentHalfLife())
}

// Set assigns an absolute score, clamped to [-100, +100], stamps
// last_updated_round, and persists synchronously.
func Set(mobId int, userId int, score int) {
	mo, name, ok := loadOrLazyInit(mobId)
	if !ok {
		return
	}
	clamped := clampScore(score)
	now := currentRound()

	opinionCacheMu.Lock()
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	mo.Opinions[userId] = &Opinion{Score: clamped, LastUpdatedRound: now}
	opinionCacheMu.Unlock()

	if err := saveToDisk(mobId, name); err != nil {
		mudlog.Warn("opinions.Set: saveToDisk", "mobId", mobId, "userId", userId, "error", err)
	}
}

// Bump adds delta to the current decay-adjusted score, clamps,
// re-stamps the round, and persists synchronously. The everyday
// mutator — combat, dialogue, quest hooks all call this.
func Bump(mobId int, userId int, delta int) {
	mo, name, ok := loadOrLazyInit(mobId)
	if !ok {
		return
	}
	now := currentRound()
	hl := currentHalfLife()

	opinionCacheMu.Lock()
	if mo.Opinions == nil {
		mo.Opinions = map[int]*Opinion{}
	}
	op, has := mo.Opinions[userId]
	var base int
	if has {
		base = decayedScore(op.Score, mo.DefaultDisposition, op.LastUpdatedRound, now, hl)
	} else {
		base = mo.DefaultDisposition
	}
	newScore := clampScore(base + delta)
	mo.Opinions[userId] = &Opinion{Score: newScore, LastUpdatedRound: now}
	opinionCacheMu.Unlock()

	if err := saveToDisk(mobId, name); err != nil {
		mudlog.Warn("opinions.Bump: saveToDisk", "mobId", mobId, "userId", userId, "error", err)
	}
}

// TierFor is sugar for TierOf(Get(mobId, userId)).
func TierFor(mobId int, userId int) Tier {
	return TierOf(Get(mobId, userId))
}
