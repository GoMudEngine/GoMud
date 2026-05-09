package knowledge

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Test-only seam — overrides util.GetRoundCount(). Production never sets this.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

// findRecord returns the record for the given subject, or nil. Caller must
// hold knowledgeCacheMu (read or write).
func findRecord(fc *ObserverFile, subject Subject) *Record {
	for _, r := range fc.Records {
		if r.Subject == subject {
			return r
		}
	}
	return nil
}

// observerNameFor looks up the mob template name for filename purposes. If
// the lookup fails, returns "" — saveObserverFile tolerates blank names.
// Implementations may use mobs.GetMobSpec(mobs.MobId(id)).Name; this helper
// keeps the persistence layer decoupled from mobs imports.
var observerNameFor = func(mobId int) string {
	// In production this will be wired in init() or via a setter; for tests
	// we can override directly.
	return ""
}

func RecordMet(observerMobId int, subject Subject, room int, source Source) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       source,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	} else {
		r.HasMet = true
	}
	r.LastSeenRoom = room
	r.LastSeenRound = now
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordMet: save failed", "observer", observerMobId, "error", err)
	}
}

func Get(observerMobId int, subject Subject) *Record {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	return findRecord(fc, subject)
}

// Test-only override for the observation log max size.
var observationLogMaxForTest func() int

func observationLogMax() int {
	if observationLogMaxForTest != nil {
		return observationLogMaxForTest()
	}
	return int(configs.GetBalanceConfig().KnowledgeObservationLogMax)
}

func RecordObservation(observerMobId int, subject Subject, room int) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()
	maxLog := observationLogMax()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		r = &Record{
			Subject:      subject,
			HasMet:       true,
			Source:       SourceWitnessed,
			Confidence:   ConfidenceHigh,
			LearnedRound: now,
		}
		fc.Records = append(fc.Records, r)
	}

	// Same-round dedup at tail.
	if n := len(r.Observations); n > 0 {
		tail := r.Observations[n-1]
		if tail.Room == room && tail.Round == now {
			r.LastSeenRoom = room
			r.LastSeenRound = now
			r.LastUpdatedRound = now
			knowledgeCacheMu.Unlock()
			return
		}
	}

	r.Observations = append(r.Observations, Observation{Room: room, Round: now})
	if maxLog > 0 && len(r.Observations) > maxLog {
		r.Observations = r.Observations[len(r.Observations)-maxLog:]
	}
	r.LastSeenRoom = room
	r.LastSeenRound = now
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordObservation: save failed", "observer", observerMobId, "error", err)
	}
}
