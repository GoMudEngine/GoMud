package facts

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

// Test seam.
var roundForTest func() uint64

func currentRound() uint64 {
	if roundForTest != nil {
		return roundForTest()
	}
	return util.GetRoundCount()
}

func heardEventsMax() int {
	return configs.GetBalanceConfig().FactsHeardEventsMax
}

type DeclareOpts struct {
	Description         string
	Significance        worldevents.Significance
	Zone                string
	Region              string
	ExpiryRound         uint64
	Tags                []string
	WithdrawOnRespawnOf int
}

// Declare adds a new active fact. Returns error if a fact with the
// same id already exists.
func Declare(factId string, opts DeclareOpts) error {
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	for _, f := range r.Facts {
		if f.Id == factId {
			registryMu.Unlock()
			return fmt.Errorf("fact collision: %q already exists", factId)
		}
	}
	now := currentRound()
	f := &Fact{
		Id:                  factId,
		Description:         opts.Description,
		Significance:        opts.Significance,
		Zone:                opts.Zone,
		Region:              opts.Region,
		DeclaredRound:       now,
		ExpiryRound:         opts.ExpiryRound,
		Tags:                opts.Tags,
		WithdrawOnRespawnOf: opts.WithdrawOnRespawnOf,
		Status:              StatusActive,
	}
	r.Facts = append(r.Facts, f)
	registryMu.Unlock()

	if err := saveRegistry(r); err != nil {
		mudlog.Warn("facts.Declare: save failed", "factId", factId, "error", err)
		return err
	}
	return nil
}

// GetFact returns the fact with the given id, regardless of status,
// or nil if not found.
func GetFact(factId string) *Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	for _, f := range r.Facts {
		if f.Id == factId {
			return f
		}
	}
	return nil
}

// Withdraw flips a fact to status=withdrawn. Idempotent.
func Withdraw(factId string) {
	setStatus(factId, StatusWithdrawn)
}

// Expire flips a fact to status=expired. Idempotent.
func Expire(factId string) {
	setStatus(factId, StatusExpired)
}

func setStatus(factId string, newStatus Status) {
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	mutated := false
	for _, f := range r.Facts {
		if f.Id == factId && f.Status == StatusActive {
			f.Status = newStatus
			mutated = true
			break
		}
	}
	registryMu.Unlock()

	if mutated {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.setStatus: save failed", "factId", factId, "status", newStatus, "error", err)
		}
	}
}

// PruneExpired walks active facts; flips any past expiry_round to
// expired. Returns count.
func PruneExpired() int {
	r := loadOrLazyInitRegistry()
	now := currentRound()

	registryMu.Lock()
	count := 0
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		if f.ExpiryRound == 0 {
			continue // never
		}
		if f.ExpiryRound <= now {
			f.Status = StatusExpired
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.PruneExpired: save failed", "error", err)
		}
	}
	return count
}

// WithdrawAllBoundTo flips active facts whose WithdrawOnRespawnOf
// matches the given mob template id to status=withdrawn. Returns
// count flipped.
func WithdrawAllBoundTo(mobTemplateId int) int {
	if mobTemplateId == 0 {
		return 0
	}
	r := loadOrLazyInitRegistry()

	registryMu.Lock()
	count := 0
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		if f.WithdrawOnRespawnOf == mobTemplateId {
			f.Status = StatusWithdrawn
			count++
		}
	}
	registryMu.Unlock()

	if count > 0 {
		if err := saveRegistry(r); err != nil {
			mudlog.Warn("facts.WithdrawAllBoundTo: save failed", "mobId", mobTemplateId, "error", err)
		}
	}
	return count
}

// AllActiveFacts returns every fact currently in StatusActive.
func AllActiveFacts() []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, 0)
	for _, f := range r.Facts {
		if f.Status == StatusActive {
			out = append(out, f)
		}
	}
	return out
}

// AllFactsByTag returns active facts that include the given tag.
func AllFactsByTag(tag string) []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, 0)
	for _, f := range r.Facts {
		if f.Status != StatusActive {
			continue
		}
		for _, t := range f.Tags {
			if t == tag {
				out = append(out, f)
				break
			}
		}
	}
	return out
}

// AllRows returns every fact regardless of status. Admin/debug use.
func AllRows() []*Fact {
	r := loadOrLazyInitRegistry()
	registryMu.RLock()
	defer registryMu.RUnlock()
	out := make([]*Fact, len(r.Facts))
	copy(out, r.Facts)
	return out
}

// Test seam for bounded FIFO cap.
var heardEventsMaxForTest func() int

func effectiveHeardEventsMax() int {
	if heardEventsMaxForTest != nil {
		return heardEventsMaxForTest()
	}
	return heardEventsMax()
}

// RecordHeardEvent appends an event id to the observer's heard_events
// FIFO. Bounded; oldest entries fall off when over cap. Dedups: if
// the eventId is already in the list, no-op.
func RecordHeardEvent(observerMobId int, eventId uint64) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()
	maxLog := effectiveHeardEventsMax()

	awarenessCacheMu.Lock()
	for _, e := range a.HeardEvents {
		if e == eventId {
			awarenessCacheMu.Unlock()
			return
		}
	}
	a.HeardEvents = append(a.HeardEvents, eventId)
	if maxLog > 0 && len(a.HeardEvents) > maxLog {
		a.HeardEvents = a.HeardEvents[len(a.HeardEvents)-maxLog:]
	}
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.RecordHeardEvent: save failed", "observer", observerMobId, "error", err)
	}
}

// HeardEvent reports whether the observer has gossiped about this event.
func HeardEvent(observerMobId int, eventId uint64) bool {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	defer awarenessCacheMu.RUnlock()
	for _, e := range a.HeardEvents {
		if e == eventId {
			return true
		}
	}
	return false
}

// observerNameFor looks up the mob template name for filename
// purposes. Stub returns empty string by default; T11 wires it
// to a real lookup.
var observerNameFor = func(mobId int) string {
	return ""
}

// RecordKnowsFact appends a fact-knowledge entry to the observer's
// known_facts. Idempotent — if the same factId is already present,
// no-op (existing entry preserved).
func RecordKnowsFact(observerMobId int, factId string, source Source) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			awarenessCacheMu.Unlock()
			return // already known; no change
		}
	}
	a.KnownFacts = append(a.KnownFacts, FactKnowledge{
		FactId:       factId,
		Source:       source,
		LearnedRound: now,
	})
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.RecordKnowsFact: save failed", "observer", observerMobId, "factId", factId, "error", err)
	}
}

// KnowsFact reports whether the observer has the given fact in their
// known list, regardless of fact status.
func KnowsFact(observerMobId int, factId string) bool {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	defer awarenessCacheMu.RUnlock()
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			return true
		}
	}
	return false
}

type KnownFact struct {
	Fact         *Fact
	Source       Source
	LearnedRound uint64
}

// KnownFactsOf returns the joined view: walks the observer's known
// fact ids against the active fact registry. Lazy filter — withdrawn
// or expired facts are skipped.
func KnownFactsOf(observerMobId int) []KnownFact {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	awarenessCacheMu.RLock()
	known := make([]FactKnowledge, len(a.KnownFacts))
	copy(known, a.KnownFacts)
	awarenessCacheMu.RUnlock()

	out := make([]KnownFact, 0, len(known))
	for _, fk := range known {
		f := GetFact(fk.FactId)
		if f == nil || f.Status != StatusActive {
			continue
		}
		out = append(out, KnownFact{
			Fact:         f,
			Source:       fk.Source,
			LearnedRound: fk.LearnedRound,
		})
	}
	return out
}

// ForgetFact drops a single fact awareness entry.
func ForgetFact(observerMobId int, factId string) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	out := a.KnownFacts[:0]
	mutated := false
	for _, fk := range a.KnownFacts {
		if fk.FactId == factId {
			mutated = true
			continue
		}
		out = append(out, fk)
	}
	a.KnownFacts = out
	if mutated {
		a.LastUpdatedRound = now
	}
	awarenessCacheMu.Unlock()

	if mutated {
		if err := saveAwareness(a); err != nil {
			mudlog.Warn("facts.ForgetFact: save failed", "observer", observerMobId, "factId", factId, "error", err)
		}
	}
}

// ForgetAll drops every awareness entry (heard events + known facts)
// for an observer.
func ForgetAll(observerMobId int) {
	a := loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
	now := currentRound()

	awarenessCacheMu.Lock()
	a.HeardEvents = nil
	a.KnownFacts = nil
	a.LastUpdatedRound = now
	awarenessCacheMu.Unlock()

	if err := saveAwareness(a); err != nil {
		mudlog.Warn("facts.ForgetAll: save failed", "observer", observerMobId, "error", err)
	}
}

// AllForObserver returns the raw awareness record for an observer.
// Used by admin commands.
func AllForObserver(observerMobId int) *Awareness {
	return loadOrLazyInitAwareness(observerMobId, observerNameFor(observerMobId))
}
