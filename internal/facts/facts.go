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
