package knowledge

import (
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
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

// observerNameFor returns the mob template's display name for filename
// purposes. Tests can override via `observerNameFor = func(int) string {...}`.
var observerNameFor = func(mobId int) string {
	if spec := mobs.GetMobSpec(mobs.MobId(mobId)); spec != nil {
		return spec.Character.Name
	}
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

// Forget removes the entire record for subject from this observer's file.
// Intentional no-cascade: forgetting knowledge does NOT automatically
// cascade to opinion (1.1) or reputation (1.2/1.3) state. Those systems
// track faction-level trust derived from witnessed behaviour over time,
// not individual memory. Amnesia spells are the natural future moment to
// revisit cascade. See docs/superpowers/specs/2026-05-09-mob-aliveness-1.4-knowledge-model-design.md.
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

// ForgetFact clears a single field ("name", "observations", or "crimes")
// from the record without removing the record itself. Unknown fact keys
// are a no-op. Same no-cascade policy as Forget — see comment above.
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

// WitnessedCrime is the joined view of one crime ID from the knowledge
// layer enriched with live data from the 1.3 crimes substrate.
type WitnessedCrime struct {
	CrimeId       int
	Kind          crimes.Kind
	ResolvedRound uint64
}

// WitnessedCrimes returns the joined view of this observer's witnessed
// crime IDs against the 1.3 substrate. Lazy-filter-on-read: callers
// see all referenced crimes plus their current resolved status, and
// decide whether to surface stale entries.
//
// The 1.3 lookup walks all known factions because the knowledge layer
// doesn't store which faction the crime belongs to. For v1 this is
// fine — the underlying crimes.AllForFaction call walks an in-memory
// cache. If this becomes hot, a follow-on can store the faction id
// alongside the crime id on the knowledge record.
func WitnessedCrimes(observerMobId int, subject Subject) []WitnessedCrime {
	r := Get(observerMobId, subject)
	if r == nil || len(r.CrimesWitnessed) == 0 {
		return nil
	}
	want := make(map[int]bool, len(r.CrimesWitnessed))
	for _, id := range r.CrimesWitnessed {
		want[id] = true
	}

	out := make([]WitnessedCrime, 0, len(r.CrimesWitnessed))
	seen := make(map[int]bool, len(r.CrimesWitnessed))

	// Walk every loaded faction's crimes, keeping rows whose IDs we know.
	for _, def := range factions.AllDefinitions() {
		for _, c := range crimes.AllForFaction(def.FactionId, true /*includeResolved*/) {
			if !want[c.Id] || seen[c.Id] {
				continue
			}
			seen[c.Id] = true
			out = append(out, WitnessedCrime{
				CrimeId:       c.Id,
				Kind:          c.Kind,
				ResolvedRound: c.ResolvedRound,
			})
		}
	}
	return out
}

// AllForObserver returns a snapshot copy of every record this observer
// holds. Returns nil if the observer has no records.
func AllForObserver(observerMobId int) []*Record {
	fc := loadOrLazyInit(observerMobId, observerNameFor(observerMobId))
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	if len(fc.Records) == 0 {
		return nil
	}
	out := make([]*Record, len(fc.Records))
	copy(out, fc.Records)
	return out
}

// AllObserversOfPlayer returns the in-memory observer mob IDs that hold
// any record about this player. Best-effort: walks only loaded files.
func AllObserversOfPlayer(userId int) []int {
	subj := PlayerSubject(userId)
	knowledgeCacheMu.RLock()
	defer knowledgeCacheMu.RUnlock()
	out := make([]int, 0)
	for mobId, fc := range knowledgeCache {
		for _, r := range fc.Records {
			if r.Subject == subj {
				out = append(out, mobId)
				break
			}
		}
	}
	return out
}

// RecordRoutineObservers writes a knowledge observation for every other
// NPC mob currently in the room, treating the moving entity as the
// subject. Used by the forager/caravan room-change hook.
func RecordRoutineObservers(movingInstanceId int, movingTemplateId int, room *rooms.Room) {
	if room == nil {
		return
	}
	subject := MobSubject(movingTemplateId)
	for _, otherInstId := range room.GetMobs() {
		if otherInstId == movingInstanceId {
			continue
		}
		other := mobs.GetInstance(otherInstId)
		if other == nil {
			continue
		}
		RecordObservation(int(other.MobId), subject, room.RoomId)
		RecordMet(int(other.MobId), subject, room.RoomId, SourceWitnessed)
	}
}
