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

func RecordName(observerMobId int, subject Subject, name string, source Source) {
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
	}
	if r.NameLearned == name {
		knowledgeCacheMu.Unlock()
		return
	}
	r.NameLearned = name
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordName: save failed", "observer", observerMobId, "error", err)
	}
}

func RecordCrimeWitnessed(observerMobId int, subject Subject, crimeId int) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

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
	for _, existing := range r.CrimesWitnessed {
		if existing == crimeId {
			knowledgeCacheMu.Unlock()
			return
		}
	}
	r.CrimesWitnessed = append(r.CrimesWitnessed, crimeId)
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.RecordCrimeWitnessed: save failed", "observer", observerMobId, "error", err)
	}
}

func Forget(observerMobId int, subject Subject) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))

	knowledgeCacheMu.Lock()
	mutated := false
	for i, r := range fc.Records {
		if r.Subject == subject {
			fc.Records = append(fc.Records[:i], fc.Records[i+1:]...)
			mutated = true
			break
		}
	}
	knowledgeCacheMu.Unlock()

	if mutated {
		if err := saveObserverFile(fc); err != nil {
			mudlog.Warn("knowledge.Forget: save failed", "observer", observerMobId, "error", err)
		}
	}
}

func ForgetFact(observerMobId int, subject Subject, fact string) {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	knowledgeCacheMu.Lock()
	r := findRecord(fc, subject)
	if r == nil {
		knowledgeCacheMu.Unlock()
		return
	}
	switch fact {
	case "name":
		r.NameLearned = ""
	case "observations":
		r.Observations = nil
	case "crimes":
		r.CrimesWitnessed = nil
	default:
		// Unknown fact: no-op.
		knowledgeCacheMu.Unlock()
		return
	}
	r.LastUpdatedRound = now
	knowledgeCacheMu.Unlock()

	if err := saveObserverFile(fc); err != nil {
		mudlog.Warn("knowledge.ForgetFact: save failed", "observer", observerMobId, "error", err)
	}
}

func HasMet(observerMobId int, subject Subject) bool {
	r := Get(observerMobId, subject)
	return r != nil && r.HasMet
}

func NameOf(observerMobId int, subject Subject) (string, bool) {
	r := Get(observerMobId, subject)
	if r == nil || r.NameLearned == "" {
		return "", false
	}
	return r.NameLearned, true
}

func LastSeen(observerMobId int, subject Subject) (int, uint64, bool) {
	r := Get(observerMobId, subject)
	if r == nil || r.LastSeenRound == 0 {
		return 0, 0, false
	}
	return r.LastSeenRoom, r.LastSeenRound, true
}
