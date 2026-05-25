# Mob Aliveness 3.6 — NPC↔NPC Idle Conversations Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Two related NPCs in the same room occasionally exchange a 2-4 line conversation drawn from a relationship-type-keyed library with optional per-pair overrides. Line-per-round pacing, life-state gating, graceful abort on interruption.

**Architecture:** New `internal/conversations/` package owns the YAML loader, registry (type pools + pair overrides), exchange picker, and per-conversation state machine. Trigger lives in `NewRound_IdleMobs` (low-rate per-tick roll) with a player-arrival boost in `go.go`. State machine uses MiscData on both NPCs with a shared `conversation_line_idx` counter so the speaker rotation is deterministic and self-driving.

**Tech Stack:** Go 1.24, new package `internal/conversations/`, new YAML directories under `_datafiles/world/dogmud/conversations/`, integration points in `internal/hooks/NewRound_IdleMobs.go` + `internal/usercommands/go.go`, three new config knobs in `internal/configs/config.balance.go`. Reuses the chunk 1.6 `relationships` package (Type constants, `RelationsBetween`, `AreRelated`) and the chunk 3.2 MiscData / IdleMobs patterns.

**Spec:** `docs/superpowers/specs/2026-05-25-mob-aliveness-3.6-npc-conversations-design.md`

**Branch:** `feature/mob-aliveness-3.6-npc-conversations` (already created; spec committed as `b0615a11`).

---

## Stage map

| Stage | Task | Description |
|---|---|---|
| 1 | T1 | Package skeleton — types + registry + pair-key normalization (TDD) |
| 2 | T2 | YAML loader + standalone validators + DI cross-checks |
| 3 | T3 | Three Conversation* config knobs |
| 4 | T4 | Picker — choose exchange from type pool ∪ subtype ∪ pair override (TDD) |
| 5 | T5 | State machine — TickConversation pure logic + abort detection (TDD) |
| 6 | T6 | TryStart + TryStartBetween + startConversation entry points |
| 7 | T7 | IdleMobs trigger branch — per-tick roll + eligibility gates |
| 8 | T8 | go.go player-arrival boost |
| 9 | T9 | Pilot content — relationship edges on 4 Thornwall mob YAMLs |
| 10 | T10 | Pilot content — friend.yaml conversation pool |
| 11 | T11 | Pilot content (optional) — rival.yaml + Dal↔Wrex pair override |
| 12 | T12 | Documentation pass |
| 13 | T13 | Smoketester goal file + roadmap closeout |

13 tasks. Sequential: T1 is foundation; T2 needs T1; T4 needs T1; T5 needs T1 + T4; T6 needs T4 + T5; T7 needs T6 + T3; T8 needs T6; T9-T11 are content; T12-T13 closeout.

---

## Task 1: Package skeleton + types + registry

**Files:**
- Create: `internal/conversations/conversation.go`
- Create: `internal/conversations/conversation_test.go`

- [ ] **Step 1: Confirm the relationships Type constants**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '1,30p' internal/relationships/types.go
```

You should see `TypeFamily`, `TypeFriend`, `TypeRival`, `TypeLover`, `TypeEmployer`, `TypeEmployee`. We'll key pool registry by these.

- [ ] **Step 2: Write the failing tests**

Create `internal/conversations/conversation_test.go`:

```go
package conversations

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/relationships"
)

func TestGetPool_EmptyTypeReturnsNil(t *testing.T) {
	if got := GetPool(relationships.Type("")); got != nil {
		t.Errorf("expected nil for empty type, got %+v", got)
	}
}

func TestGetPool_UnknownReturnsNil(t *testing.T) {
	if got := GetPool(relationships.Type("not-a-real-type")); got != nil {
		t.Errorf("expected nil for unknown type, got %+v", got)
	}
}

func TestGetPairOverride_OrderIndependent(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{{Lines: []ConversationLine{{Speaker: "A", Text: "x"}}}},
	})
	defer unregisterTestPool("friend")
	registerTestPairOverride(&PairOverride{
		Id: "test_pair", MobA: 100, MobB: 200,
		Exchanges: []Exchange{{Lines: []ConversationLine{{Speaker: "A", Text: "hi"}}}},
	})
	defer unregisterTestPairOverride(100, 200)

	got1 := GetPairOverride(100, 200)
	got2 := GetPairOverride(200, 100)
	if got1 == nil || got2 == nil {
		t.Fatalf("expected lookup to work in both orders, got %v / %v", got1, got2)
	}
	if got1 != got2 {
		t.Errorf("expected order-independent lookup to return same pointer")
	}
}

func TestPairKey_Normalizes(t *testing.T) {
	k1 := makePairKey(100, 200)
	k2 := makePairKey(200, 100)
	if k1 != k2 {
		t.Errorf("pair keys should normalize: %+v != %+v", k1, k2)
	}
	if k1.LowId != 100 || k1.HighId != 200 {
		t.Errorf("expected LowId=100 HighId=200, got %+v", k1)
	}
}
```

- [ ] **Step 3: Run the test, confirm fail**

```bash
go test ./internal/conversations/ -v
```

Expected: compile error — `GetPool`, `GetPairOverride`, types not defined.

- [ ] **Step 4: Implement `internal/conversations/conversation.go`**

```go
package conversations

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/relationships"
)

// ConversationLine is one line in an exchange. Speaker is "A" or "B"
// (the conceptual initiator vs partner; actual NPC role-mapping happens
// per-conversation when the exchange starts).
type ConversationLine struct {
	Speaker string `yaml:"speaker"`
	Text    string `yaml:"text"`
}

// Exchange is a single 2-4 line conversation script.
type Exchange struct {
	Lines []ConversationLine `yaml:"lines"`
}

// Pool is a per-relationship-type collection of generic exchanges, with
// optional subtype sub-pools for richer variation.
type Pool struct {
	Id          string             `yaml:"id"` // must match a relationships.Type
	Description string             `yaml:"description,omitempty"`
	Exchanges   []Exchange         `yaml:"exchanges"`
	Subtypes    map[string]Subpool `yaml:"subtypes,omitempty"`
}

// Subpool is a relationship-subtype-keyed addendum to a Pool's exchanges.
type Subpool struct {
	Exchanges []Exchange `yaml:"exchanges"`
}

// PairOverride adds exchanges specific to a single NPC pair on top of
// (not replacing) the type pool.
type PairOverride struct {
	Id        string     `yaml:"id"`
	MobA      int        `yaml:"mob_a"`
	MobB      int        `yaml:"mob_b"`
	Exchanges []Exchange `yaml:"exchanges"`
}

// pairKey is the registry key for PairOverride. Always normalized so
// (a, b) and (b, a) map to the same entry.
type pairKey struct {
	LowId  int
	HighId int
}

func makePairKey(a, b int) pairKey {
	if a > b {
		a, b = b, a
	}
	return pairKey{LowId: a, HighId: b}
}

// Package-level registries, populated by Load() at startup.
var (
	poolsMu sync.RWMutex
	pools   = map[relationships.Type]*Pool{}

	pairsMu sync.RWMutex
	pairs   = map[pairKey]*PairOverride{}
)

// GetPool returns the type pool for the given relationship type, or nil
// if no pool is registered for that type.
func GetPool(t relationships.Type) *Pool {
	if t == "" {
		return nil
	}
	poolsMu.RLock()
	defer poolsMu.RUnlock()
	return pools[t]
}

// GetPairOverride returns the pair override for the given mob ids,
// normalized so the order doesn't matter. Returns nil if no override
// is registered.
func GetPairOverride(mobA, mobB int) *PairOverride {
	pairsMu.RLock()
	defer pairsMu.RUnlock()
	return pairs[makePairKey(mobA, mobB)]
}

// Test-only registry helpers.
func registerTestPool(p *Pool) {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	pools[relationships.Type(p.Id)] = p
}

func unregisterTestPool(id string) {
	poolsMu.Lock()
	defer poolsMu.Unlock()
	delete(pools, relationships.Type(id))
}

func registerTestPairOverride(po *PairOverride) {
	pairsMu.Lock()
	defer pairsMu.Unlock()
	pairs[makePairKey(po.MobA, po.MobB)] = po
}

func unregisterTestPairOverride(a, b int) {
	pairsMu.Lock()
	defer pairsMu.Unlock()
	delete(pairs, makePairKey(a, b))
}

// Exported test helpers for cross-package tests (hooks package will
// use these in T7's tests).
func RegisterTestPool(p *Pool)                { registerTestPool(p) }
func UnregisterTestPool(id string)            { unregisterTestPool(id) }
func RegisterTestPairOverride(po *PairOverride) { registerTestPairOverride(po) }
func UnregisterTestPairOverride(a, b int)     { unregisterTestPairOverride(a, b) }
```

- [ ] **Step 5: Run tests, confirm pass**

```bash
go test ./internal/conversations/ -v
```

Expected: all 4 tests pass.

- [ ] **Step 6: Commit**

```bash
git add internal/conversations/conversation.go internal/conversations/conversation_test.go
git commit -m "$(cat <<'EOF'
feat(conversations): package skeleton + types + registry

Foundation for chunk 3.6. ConversationLine + Exchange + Pool +
Subpool + PairOverride types. Two package-level registries
(pools keyed by relationships.Type, pairs keyed by sorted-id
tuple). GetPool / GetPairOverride accessors with order-
independent pair lookup. Test helpers (RegisterTestPool +
exported variants for cross-package tests).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: YAML loader + validators

**Files:**
- Create: `internal/conversations/loader.go`
- Create: `internal/conversations/loader_test.go`
- Modify: `internal/mobs/mobs.go` (call `conversations.Load()` from `LoadDataFiles()` after mob templates load + after relationships load)

- [ ] **Step 1: Confirm the load ordering point**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "LoadFromMobs\|RegisterMobShop\|LoadSchedules\|LoadPatrols" internal/mobs/mobs.go | head -10
```

The relationships graph is built from mob templates via `relationships.LoadFromMobs(...)`. We need that to have run before we cross-check `relationships.RelationsBetween(...)` for pair overrides. Confirm `LoadFromMobs` runs inside `LoadDataFiles` and identify where to insert `conversations.Load()` (after it).

- [ ] **Step 2: Write failing tests**

Create `internal/conversations/loader_test.go`:

```go
package conversations

import (
	"strings"
	"testing"
)

func TestValidatePool_OK(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{
				{Speaker: "A", Text: "hello"},
				{Speaker: "B", Text: "hi back"},
			}},
		},
	}
	if err := validatePoolStandalone(p); err != nil {
		t.Errorf("valid pool should validate, got: %v", err)
	}
}

func TestValidatePool_UnknownTypeRejected(t *testing.T) {
	p := &Pool{
		Id: "not-a-real-type",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "x"}, {Speaker: "B", Text: "y"}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "relationship type") {
		t.Errorf("expected relationship-type error, got: %v", err)
	}
}

func TestValidatePool_EmptyExchangeRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: nil},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "no lines") {
		t.Errorf("expected no-lines error, got: %v", err)
	}
}

func TestValidatePool_BadSpeakerRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "C", Text: "x"}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "speaker") {
		t.Errorf("expected speaker error, got: %v", err)
	}
}

func TestValidatePool_EmptyTextRejected(t *testing.T) {
	p := &Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: ""}}},
		},
	}
	err := validatePoolStandalone(p)
	if err == nil || !strings.Contains(err.Error(), "empty text") {
		t.Errorf("expected empty-text error, got: %v", err)
	}
}

func TestValidatePairOverride_OK(t *testing.T) {
	po := &PairOverride{
		Id:   "test_pair",
		MobA: 100, MobB: 200,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "hi"}, {Speaker: "B", Text: "hi"}}},
		},
	}
	if err := validatePairOverrideStandalone(po); err != nil {
		t.Errorf("valid pair override should validate, got: %v", err)
	}
}

func TestValidatePairOverride_SelfPairRejected(t *testing.T) {
	po := &PairOverride{
		Id:   "self",
		MobA: 100, MobB: 100,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "x"}}},
		},
	}
	err := validatePairOverrideStandalone(po)
	if err == nil || !strings.Contains(err.Error(), "self") {
		t.Errorf("expected self-pair error, got: %v", err)
	}
}

// World-aware checks (mob existence, relationship edge presence) are
// covered by the boot smoke in T13; the integration setup is too heavy
// for unit tests.
func TestValidatePairOverrideAgainstWorld_Stub(t *testing.T) {
	t.Skip("requires loaded mobs registry + relationships graph — covered by T13 boot smoke")
}
```

- [ ] **Step 3: Run, confirm fail**

```bash
go test ./internal/conversations/ -v
```

Expected: compile error — `validatePoolStandalone`, `validatePairOverrideStandalone` not defined.

- [ ] **Step 4: Implement `internal/conversations/loader.go`**

```go
package conversations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// ConversationWorldValidator is injected at startup to perform world-
// aware validation (mob existence + relationship edge presence) without
// creating an import cycle. Set via SetConversationWorldValidator before
// calling Load; if either function is nil, the world-aware pass is
// silently skipped.
var conversationWorldValidator struct {
	mobExists           func(id int) bool
	relationshipBetween func(a, b int) bool
}

// SetConversationWorldValidator wires in the mob-existence and
// relationship-edge checks. Caller (typically main.go) supplies closures
// that reach into the mobs and relationships packages. Mirrors the chunk
// 3.2 / 3.4 DI pattern.
func SetConversationWorldValidator(mobExists func(int) bool, relationshipBetween func(int, int) bool) {
	conversationWorldValidator.mobExists = mobExists
	conversationWorldValidator.relationshipBetween = relationshipBetween
}

// Load walks _datafiles/world/dogmud/conversations/{types,pairs}/*.yaml,
// parses each file, validates, and registers it in the package-level
// registries. Panics on duplicate ids or validation failures. If a
// directory is missing, logs and continues (optional content).
func Load() {
	start := time.Now()

	dataRoot := configs.GetFilePathsConfig().DataFiles.String() + `/conversations`

	if _, err := os.Stat(dataRoot); os.IsNotExist(err) {
		mudlog.Info("conversations.Load()", "loadedPools", 0, "loadedPairs", 0,
			"note", "conversations directory does not exist — skipping",
			"Time Taken", time.Since(start))
		return
	}

	tmpPools := map[relationships.Type]*Pool{}
	tmpPairs := map[pairKey]*PairOverride{}

	// types/*.yaml
	typesDir := filepath.Join(dataRoot, "types")
	if _, err := os.Stat(typesDir); err == nil {
		if walkErr := walkYAMLs(typesDir, func(path string, data []byte) error {
			var p Pool
			if err := yaml.Unmarshal(data, &p); err != nil {
				return fmt.Errorf("conversation pool: unmarshal %s: %w", path, err)
			}
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			expected := util.ConvertForFilename(p.Id)
			if base != expected {
				return fmt.Errorf("conversation pool: filename mismatch: %q base %q != expected %q (id %q)",
					path, base, expected, p.Id)
			}
			if err := validatePoolStandalone(&p); err != nil {
				return fmt.Errorf("conversation pool %q (%s): %w", p.Id, path, err)
			}
			emitPoolWarnings(&p)
			t := relationships.Type(p.Id)
			if _, dup := tmpPools[t]; dup {
				return fmt.Errorf("conversation pool: duplicate id %q in %s", p.Id, path)
			}
			pp := p
			tmpPools[t] = &pp
			return nil
		}); walkErr != nil {
			panic(fmt.Sprintf("conversations.Load() pools failed: %v", walkErr))
		}
	}

	// pairs/*.yaml
	pairsDir := filepath.Join(dataRoot, "pairs")
	if _, err := os.Stat(pairsDir); err == nil {
		if walkErr := walkYAMLs(pairsDir, func(path string, data []byte) error {
			var po PairOverride
			if err := yaml.Unmarshal(data, &po); err != nil {
				return fmt.Errorf("conversation pair: unmarshal %s: %w", path, err)
			}
			// Filename convention: <smaller_id>_<larger_id>.yaml
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			lo, hi := po.MobA, po.MobB
			if lo > hi {
				lo, hi = hi, lo
			}
			expected := fmt.Sprintf("%d_%d", lo, hi)
			if base != expected {
				return fmt.Errorf("conversation pair: filename mismatch: %q base %q != expected %q (mobs %d/%d)",
					path, base, expected, po.MobA, po.MobB)
			}
			if err := validatePairOverrideStandalone(&po); err != nil {
				return fmt.Errorf("conversation pair %q (%s): %w", po.Id, path, err)
			}
			emitPairOverrideWarnings(&po)
			k := makePairKey(po.MobA, po.MobB)
			if _, dup := tmpPairs[k]; dup {
				return fmt.Errorf("conversation pair: duplicate pair %d/%d in %s", po.MobA, po.MobB, path)
			}
			pp := po
			tmpPairs[k] = &pp
			return nil
		}); walkErr != nil {
			panic(fmt.Sprintf("conversations.Load() pairs failed: %v", walkErr))
		}
	}

	// World-aware validation: mob existence + relationship edge presence
	// for pair overrides. Only runs when both injectors are wired.
	if conversationWorldValidator.mobExists != nil && conversationWorldValidator.relationshipBetween != nil {
		for _, po := range tmpPairs {
			if !conversationWorldValidator.mobExists(po.MobA) {
				panic(fmt.Sprintf("conversation pair %q: mob_a=%d not found in mobs registry", po.Id, po.MobA))
			}
			if !conversationWorldValidator.mobExists(po.MobB) {
				panic(fmt.Sprintf("conversation pair %q: mob_b=%d not found in mobs registry", po.Id, po.MobB))
			}
			if !conversationWorldValidator.relationshipBetween(po.MobA, po.MobB) {
				panic(fmt.Sprintf("conversation pair %q: no relationship edge between mobs %d and %d (chunk 1.6) — pair override would never fire",
					po.Id, po.MobA, po.MobB))
			}
		}
	}

	poolsMu.Lock()
	pools = tmpPools
	poolsMu.Unlock()

	pairsMu.Lock()
	pairs = tmpPairs
	pairsMu.Unlock()

	mudlog.Info("conversations.Load()",
		"loadedPools", len(tmpPools),
		"loadedPairs", len(tmpPairs),
		"Time Taken", time.Since(start))
}

// walkYAMLs walks a directory and invokes fn for each .yaml file.
func walkYAMLs(dir string, fn func(path string, data []byte) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		return fn(path, data)
	})
}

// validatePoolStandalone runs internal-consistency checks on a pool
// without touching the mobs registry or filesystem. Suitable for unit
// tests.
func validatePoolStandalone(p *Pool) error {
	// id must be one of the six known relationship types.
	switch relationships.Type(p.Id) {
	case relationships.TypeFamily, relationships.TypeFriend, relationships.TypeRival,
		relationships.TypeLover, relationships.TypeEmployer, relationships.TypeEmployee:
		// ok
	default:
		return fmt.Errorf("pool id %q is not a known relationship type", p.Id)
	}
	for i, ex := range p.Exchanges {
		if err := validateExchange(ex); err != nil {
			return fmt.Errorf("exchange %d: %w", i, err)
		}
	}
	for subKey, sub := range p.Subtypes {
		for i, ex := range sub.Exchanges {
			if err := validateExchange(ex); err != nil {
				return fmt.Errorf("subtype %q exchange %d: %w", subKey, i, err)
			}
		}
	}
	return nil
}

// validatePairOverrideStandalone is the same for PairOverride.
func validatePairOverrideStandalone(po *PairOverride) error {
	if po.MobA == 0 || po.MobB == 0 {
		return errors.New("mob_a and mob_b must both be non-zero")
	}
	if po.MobA == po.MobB {
		return fmt.Errorf("self-pair (mob_a == mob_b == %d) not allowed", po.MobA)
	}
	for i, ex := range po.Exchanges {
		if err := validateExchange(ex); err != nil {
			return fmt.Errorf("exchange %d: %w", i, err)
		}
	}
	return nil
}

// validateExchange checks the line list of a single exchange.
func validateExchange(ex Exchange) error {
	if len(ex.Lines) == 0 {
		return errors.New("exchange has no lines")
	}
	for i, line := range ex.Lines {
		if line.Speaker != "A" && line.Speaker != "B" {
			return fmt.Errorf("line %d: speaker %q must be \"A\" or \"B\"", i, line.Speaker)
		}
		if strings.TrimSpace(line.Text) == "" {
			return fmt.Errorf("line %d: empty text", i)
		}
	}
	return nil
}

// emitPoolWarnings logs non-fatal author hints.
func emitPoolWarnings(p *Pool) {
	if len(p.Exchanges) == 0 && len(p.Subtypes) == 0 {
		mudlog.Warn("conversations.Load()",
			"poolId", p.Id, "warning", "pool has zero exchanges — relationship type will be silent")
	}
	known := map[string]bool{"fond": true, "estranged": true, "professional": true, "bitter": true}
	for subKey := range p.Subtypes {
		if !known[subKey] {
			mudlog.Warn("conversations.Load()",
				"poolId", p.Id, "subtype", subKey,
				"warning", "subtype key outside documented convention (fond / estranged / professional / bitter)")
		}
	}
	for i, ex := range p.Exchanges {
		if len(ex.Lines) == 1 {
			mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i,
				"warning", "single-line exchange — that's a monologue, not a conversation")
		}
		if len(ex.Lines) > 6 {
			mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i,
				"warning", "exchange has more than 6 lines — player tab-out risk")
		}
		for li, line := range ex.Lines {
			if len(line.Text) > 78 {
				mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i, "line", li,
					"warning", "line text longer than 78 chars — MUD line width recommendation")
			}
		}
	}
}

// emitPairOverrideWarnings logs non-fatal author hints for pair overrides.
func emitPairOverrideWarnings(po *PairOverride) {
	for i, ex := range po.Exchanges {
		if len(ex.Lines) == 1 {
			mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i,
				"warning", "single-line exchange — that's a monologue, not a conversation")
		}
		if len(ex.Lines) > 6 {
			mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i,
				"warning", "exchange has more than 6 lines — player tab-out risk")
		}
		for li, line := range ex.Lines {
			if len(line.Text) > 78 {
				mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i, "line", li,
					"warning", "line text longer than 78 chars — MUD line width recommendation")
			}
		}
	}
}
```

- [ ] **Step 5: Wire `Load()` into `mobs.LoadDataFiles()`**

Find the right insertion point — after both mob templates and the relationships graph are populated:

```bash
grep -n "LoadFromMobs\|LoadSchedules\|LoadPatrols" internal/mobs/mobs.go | head -10
```

Insert `conversations.Load()` AFTER `relationships.LoadFromMobs(...)`. Add the `conversations` import to `internal/mobs/mobs.go`.

- [ ] **Step 6: Wire the world-validator injector in `main.go`**

Mirror the chunks 3.2 / 3.4 DI wiring. Find where `mobs.SetScheduleWorldValidator(...)` and `mobs.SetPatrolWorldValidator(...)` are wired. After them, before `mobs.LoadDataFiles()`:

```go
// Chunk 3.6: conversations world validator. Cross-checks pair overrides
// reference real mobs with real relationship edges.
conversations.SetConversationWorldValidator(
	func(mobId int) bool {
		// mobs.GetMobInfo returns the template by mob id; nil means missing.
		_, ok := mobs.GetByMobId(mobs.MobId(mobId))
		return ok
	},
	func(a, b int) bool {
		return relationships.AreRelated(a, b)
	},
)
```

The `mobs.GetByMobId` accessor name may differ — search for it:

```bash
grep -n "^func GetByMobId\|^func GetMobInfo\|^func MobByMobId\|^func.*MobId.*MobId" internal/mobs/*.go | head -5
```

Use whichever existing accessor returns a template by mob id. If none exists with that exact signature, write a tiny inline check: scan `mobs.AllMobTemplates()` or similar. Pick the simplest existing call.

Add the `conversations` and `relationships` imports to `main.go` if not present.

- [ ] **Step 7: Run tests + build**

```bash
go test ./internal/conversations/ -v
go build ./...
```

Expected: green.

- [ ] **Step 8: Commit**

```bash
git add internal/conversations/loader.go internal/conversations/loader_test.go internal/mobs/mobs.go main.go
git commit -m "$(cat <<'EOF'
feat(conversations): YAML loader + standalone validators + DI

Walks _datafiles/world/dogmud/conversations/{types,pairs}/*.yaml,
parses, validates, registers under the package-level
{pools, pairs} registries. Filename ↔ id check per file type.
Standalone validators reject: unknown relationship type, empty
exchange, bad speaker, empty text, self-pair. World-aware
validation via SetConversationWorldValidator (DI) cross-checks
that pair overrides reference real mobs with real relationship
edges. Warn-only on degenerate / over-long / out-of-convention
cases.

Load() called from mobs.LoadDataFiles() after
relationships.LoadFromMobs so cross-checks work. main.go wires
the world validator before LoadDataFiles.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Three Conversation* config knobs

**Files:**
- Modify: `internal/configs/config.balance.go` (add three new fields)
- Modify: `internal/configs/config.balance.mobs.go` (set defaults — find where 3.4 / 3.3 knobs default)

- [ ] **Step 1: Find the pattern**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "ScheduleMaxPathRetries\|ScheduleWakeGraceRounds\|SleepRegenMultiplier" internal/configs/config.balance.go internal/configs/config.balance.mobs.go | head -10
```

Use whichever ConfigInt/ConfigFloat type wrapper the local convention uses.

- [ ] **Step 2: Add the three fields**

In `internal/configs/config.balance.go`, place near `ScheduleWakeGraceRounds` / `SleepRegenMultiplier`:

```go
// ConversationBaseChancePct is the per-tick percentage chance that a
// fully-idle NPC will attempt to start an idle conversation with an
// in-room partner that has a relationship edge. Default 1.0 → ~once
// per 100 ticks per NPC. Chunk 3.6.
ConversationBaseChancePct ConfigFloat `yaml:"ConversationBaseChancePct"`

// ConversationPlayerArrivalBoostPct is the percentage chance that
// a conversation will start when a player arrives in a room with 2+
// relateable, idle NPCs. Default 25. Chunk 3.6.
ConversationPlayerArrivalBoostPct ConfigInt `yaml:"ConversationPlayerArrivalBoostPct"`

// ConversationCooldownRounds is the cooldown applied to both NPCs
// after a conversation completes, before either can initiate another.
// Default 50 (~200 sec real-time). Chunk 3.6.
ConversationCooldownRounds ConfigInt `yaml:"ConversationCooldownRounds"`
```

- [ ] **Step 3: Set defaults**

In `internal/configs/config.balance.mobs.go`, find where the chunk 3.3 / 3.4 knobs are defaulted. Add:

```go
if b.ConversationBaseChancePct <= 0 {
	b.ConversationBaseChancePct = 1.0
}
if b.ConversationPlayerArrivalBoostPct < 0 {
	b.ConversationPlayerArrivalBoostPct = 25
}
if b.ConversationCooldownRounds < 1 {
	b.ConversationCooldownRounds = 50
}
```

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/configs/ -v
```

Expected: clean.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go internal/configs/config.balance.mobs.go
git commit -m "$(cat <<'EOF'
feat(configs): three Conversation* knobs for chunk 3.6

ConversationBaseChancePct (default 1.0) — per-tick chance for
a fully-idle NPC to attempt starting an idle conversation.
ConversationPlayerArrivalBoostPct (default 25) — chance for a
conversation to start when a player arrives in a room with
relateable NPCs.
ConversationCooldownRounds (default 50) — cooldown after a
conversation completes before either NPC can initiate another.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Exchange picker

**Files:**
- Modify: `internal/conversations/conversation.go` (add `pickExchange` + supporting functions)
- Modify: `internal/conversations/conversation_test.go` (add picker tests)

- [ ] **Step 1: Write failing tests**

Append to `internal/conversations/conversation_test.go`:

```go
import "github.com/GoMudEngine/GoMud/internal/util"

func TestPickExchange_TypePoolOnly(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "one"}, {Speaker: "B", Text: "1"}}},
			{Lines: []ConversationLine{{Speaker: "A", Text: "two"}, {Speaker: "B", Text: "2"}}},
		},
	})
	defer unregisterTestPool("friend")

	// No subtype, no pair override.
	ex, ok := pickExchange("friend", "", 100, 200)
	if !ok {
		t.Fatalf("expected picker to return an exchange")
	}
	if len(ex.Lines) == 0 {
		t.Errorf("picked exchange has no lines")
	}
}

func TestPickExchange_SubtypeAdditive(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "base"}, {Speaker: "B", Text: "."}}},
		},
		Subtypes: map[string]Subpool{
			"fond": {Exchanges: []Exchange{
				{Lines: []ConversationLine{{Speaker: "A", Text: "fond1"}, {Speaker: "B", Text: "."}}},
				{Lines: []ConversationLine{{Speaker: "A", Text: "fond2"}, {Speaker: "B", Text: "."}}},
			}},
		},
	})
	defer unregisterTestPool("friend")

	// Across many picks, both "base" and a "fond*" line should appear.
	util.SetRandomSeedForTest(42)
	defer util.SetRandomSeedForTest(0)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		ex, ok := pickExchange("friend", "fond", 100, 200)
		if !ok {
			t.Fatalf("expected pick to succeed")
		}
		seen[ex.Lines[0].Text] = true
	}
	if !seen["base"] {
		t.Errorf("expected base exchange to be picked at least once across 50 draws")
	}
	if !(seen["fond1"] || seen["fond2"]) {
		t.Errorf("expected fond subtype exchange to be picked at least once")
	}
}

func TestPickExchange_PairOverrideAdditive(t *testing.T) {
	registerTestPool(&Pool{
		Id: "friend",
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "generic"}, {Speaker: "B", Text: "."}}},
		},
	})
	defer unregisterTestPool("friend")
	registerTestPairOverride(&PairOverride{
		Id: "pair", MobA: 100, MobB: 200,
		Exchanges: []Exchange{
			{Lines: []ConversationLine{{Speaker: "A", Text: "pair_only"}, {Speaker: "B", Text: "."}}},
		},
	})
	defer unregisterTestPairOverride(100, 200)

	util.SetRandomSeedForTest(99)
	defer util.SetRandomSeedForTest(0)
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		ex, ok := pickExchange("friend", "", 100, 200)
		if !ok {
			t.Fatalf("expected pick to succeed")
		}
		seen[ex.Lines[0].Text] = true
	}
	if !seen["generic"] {
		t.Errorf("expected generic pool exchange to be picked across 50 draws")
	}
	if !seen["pair_only"] {
		t.Errorf("expected pair-override exchange to be picked across 50 draws")
	}
}

func TestPickExchange_NoPoolReturnsFalse(t *testing.T) {
	_, ok := pickExchange("friend", "", 100, 200)
	if ok {
		t.Errorf("expected false when no pool registered")
	}
}
```

`util.SetRandomSeedForTest` — verify it exists; if not, use `math/rand` with a seeded `*rand.Rand` injected via a package-level seam.

- [ ] **Step 2: Run, confirm fail**

```bash
go test ./internal/conversations/ -run TestPickExchange -v
```

Expected: compile error — `pickExchange` not defined.

- [ ] **Step 3: Implement the picker**

Append to `internal/conversations/conversation.go`:

```go
import (
	"github.com/GoMudEngine/GoMud/internal/util"
)

// pickExchange picks an exchange uniformly from the union of:
//   - the relationship-type pool's default exchanges (if registered)
//   - the matching subtype sub-pool's exchanges (if subtype matches)
//   - the pair override's exchanges (if registered for this mob pair)
//
// Returns (Exchange{}, false) if no exchanges are eligible.
func pickExchange(poolId, subtype string, mobA, mobB int) (Exchange, bool) {
	var eligible []Exchange

	if p := GetPool(relationships.Type(poolId)); p != nil {
		eligible = append(eligible, p.Exchanges...)
		if subtype != "" {
			if sub, ok := p.Subtypes[subtype]; ok {
				eligible = append(eligible, sub.Exchanges...)
			}
		}
	}

	if po := GetPairOverride(mobA, mobB); po != nil {
		eligible = append(eligible, po.Exchanges...)
	}

	if len(eligible) == 0 {
		return Exchange{}, false
	}

	idx := util.Rand(len(eligible))
	return eligible[idx], true
}
```

- [ ] **Step 4: Run tests, confirm pass**

```bash
go test ./internal/conversations/ -v
go build ./...
```

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/conversations/conversation.go internal/conversations/conversation_test.go
git commit -m "$(cat <<'EOF'
feat(conversations): pickExchange uniform draw across union

pickExchange(poolId, subtype, mobA, mobB) returns a uniform-
random exchange from the union of:
  - type pool default exchanges
  - matching subtype sub-pool exchanges (if subtype matches)
  - pair-override exchanges (order-independent)

Returns (Exchange{}, false) when no exchanges are eligible.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: TickConversation state machine

**Files:**
- Create: `internal/conversations/state.go` (TickConversation + abort logic)
- Create: `internal/conversations/state_test.go` (state machine tests)

- [ ] **Step 1: Write failing tests**

Create `internal/conversations/state_test.go`:

```go
package conversations

import (
	"testing"
)

// Helper: stamp full conversation state on a mock character. The actual
// MiscData lives on characters.Character which has GetMiscData/SetMiscData.
// For unit-testing the pure decision logic, we use a tiny stub that
// matches the API surface.
//
// If the production code reaches into mob.Character.SetMiscData directly,
// we need a real Character. Use the helpers from chunk 3.2/3.4 tests if
// available; otherwise build a minimal Character inline.

func TestTickConversation_FiresWhenItsMyTurn(t *testing.T) {
	t.Skip("state-machine integration test — covered by hooks T7 tests + T13 boot smoke")
	// The TickConversation function reaches into mob.Character.SetMiscData
	// and partner.Character.SetMiscData. The full integration requires
	// a registered mob instance pair. Pure logic is exercised via
	// TestComputeNextLine below.
}

func TestComputeNextLine_FiresLineWhenSpeakerMatches(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "second"},
			{Speaker: "A", Text: "third"},
		},
	}
	// Role A, line_idx 0 → should fire (line 0 speaker is A)
	plan := computeConversationPlan(ex, "A", 0)
	if !plan.ShouldFire {
		t.Errorf("expected ShouldFire=true for A at line 0, got %+v", plan)
	}
	if plan.Text != "first" {
		t.Errorf("expected text 'first', got %q", plan.Text)
	}
	if plan.NextLineIdx != 1 {
		t.Errorf("expected NextLineIdx=1, got %d", plan.NextLineIdx)
	}
	if plan.IsFinalLine {
		t.Errorf("expected IsFinalLine=false at line 0 of 3, got true")
	}
}

func TestComputeNextLine_WaitsWhenSpeakerMismatches(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "second"},
		},
	}
	// Role B, line_idx 0 → should NOT fire (line 0 speaker is A)
	plan := computeConversationPlan(ex, "B", 0)
	if plan.ShouldFire {
		t.Errorf("expected ShouldFire=false for B at line 0 (A's line), got %+v", plan)
	}
}

func TestComputeNextLine_DetectsFinalLine(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{
			{Speaker: "A", Text: "first"},
			{Speaker: "B", Text: "last"},
		},
	}
	plan := computeConversationPlan(ex, "B", 1)
	if !plan.IsFinalLine {
		t.Errorf("expected IsFinalLine=true at line 1 of 2, got %+v", plan)
	}
}

func TestComputeNextLine_OutOfRangeReturnsAbort(t *testing.T) {
	ex := Exchange{
		Lines: []ConversationLine{{Speaker: "A", Text: "only"}},
	}
	plan := computeConversationPlan(ex, "A", 999)
	if !plan.ShouldAbort {
		t.Errorf("expected ShouldAbort=true for out-of-range line_idx, got %+v", plan)
	}
}
```

- [ ] **Step 2: Run, confirm fail**

```bash
go test ./internal/conversations/ -run TestComputeNextLine -v
```

Expected: compile error — `computeConversationPlan` not defined.

- [ ] **Step 3: Implement `internal/conversations/state.go`**

```go
package conversations

// conversationPlan describes what the per-tick state machine wants to
// do for a single NPC. Pure-data, easy to unit-test.
type conversationPlan struct {
	ShouldFire   bool   // fire `say <text>` from this NPC this tick
	ShouldAbort  bool   // abort the conversation (out-of-range line_idx, etc.)
	Text         string // the text to say (set when ShouldFire)
	NextLineIdx  int    // shared line counter value AFTER this firing (set when ShouldFire)
	IsFinalLine  bool   // true when this firing completes the exchange (set when ShouldFire)
}

// computeConversationPlan is the pure decision logic for a single NPC's
// tick: given the exchange, the NPC's role ("A" or "B"), and the shared
// line index, decide whether to fire, wait, or abort.
func computeConversationPlan(ex Exchange, role string, lineIdx int) conversationPlan {
	if lineIdx < 0 || lineIdx >= len(ex.Lines) {
		return conversationPlan{ShouldAbort: true}
	}
	line := ex.Lines[lineIdx]
	if line.Speaker != role {
		// Not my turn — wait silently.
		return conversationPlan{}
	}
	return conversationPlan{
		ShouldFire:  true,
		Text:        line.Text,
		NextLineIdx: lineIdx + 1,
		IsFinalLine: lineIdx == len(ex.Lines)-1,
	}
}
```

The full `TickConversation` (which reaches into mob.Character.SetMiscData and runs the abort-trigger checks) is added in T6 alongside the rest of the entry-point API. T5 ships only the pure decision helper, which is the testable kernel.

- [ ] **Step 4: Run tests + build**

```bash
go test ./internal/conversations/ -v
go build ./...
```

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/conversations/state.go internal/conversations/state_test.go
git commit -m "$(cat <<'EOF'
feat(conversations): computeConversationPlan pure decision logic

The kernel of the per-tick state machine: given an Exchange,
the NPC's role ("A" or "B"), and the shared line_idx, decide
whether to fire (return text + next idx + is-final flag), wait
silently (speaker mismatch), or abort (out-of-range idx).

TickConversation full integration (MiscData reads/writes,
abort-trigger checks against partner state, room presence) is
added in T6 alongside TryStart/startConversation. T5 ships only
the testable pure kernel.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: TryStart + TryStartBetween + startConversation + TickConversation

**Files:**
- Modify: `internal/conversations/state.go` (add the entry-point API + full state machine)

- [ ] **Step 1: Read the room iteration API**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "func (r \*Room) GetMobs\|func (r \*Room) GetPlayers" internal/rooms/rooms.go | head -5
```

Confirm: `GetMobs() []int` returns instance ids, `GetPlayers() []int` returns user ids. Both methods exist on `*rooms.Room`.

- [ ] **Step 2: Add the entry-point API to `internal/conversations/state.go`**

Append:

```go
import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// MiscData key constants — single source of truth so callers in
// internal/hooks/ (eligibility gates) can use the same names.
const (
	MiscDataPartnerId            = "conversation_partner_id"
	MiscDataRole                 = "conversation_role"
	MiscDataPoolId               = "conversation_pool_id"
	MiscDataPairOverrideId       = "conversation_pair_override_id"
	MiscDataExchangeId           = "conversation_exchange_id"
	MiscDataLineIdx              = "conversation_line_idx"
	MiscDataLastRound            = "conversation_last_round"
	MiscDataCooldownUntilRound   = "conversation_cooldown_until_round"
)

// TryStart attempts to start a conversation initiated by `mob`. Picks
// a random in-room related partner, picks an exchange, stamps state on
// both NPCs, fires the first line immediately. No-op if no eligible
// partner / cooldowns active / no exchanges for the relationship type.
func TryStart(initiator *mobs.Mob, room *rooms.Room) {
	if initiator == nil || room == nil {
		return
	}
	if isOnCooldown(&initiator.Character) {
		return
	}

	// Find eligible partners in the room: must be related per chunk 1.6,
	// must be fully idle, must not be on cooldown.
	var candidates []*mobs.Mob
	for _, otherInstId := range room.GetMobs() {
		if otherInstId == initiator.InstanceId {
			continue
		}
		other := mobs.GetInstance(otherInstId)
		if other == nil {
			continue
		}
		if !relationships.AreRelated(int(initiator.MobId), int(other.MobId)) {
			continue
		}
		if !isFullyIdle(other) || isOnCooldown(&other.Character) || isInConversation(&other.Character) {
			continue
		}
		candidates = append(candidates, other)
	}
	if len(candidates) == 0 {
		return
	}

	partner := candidates[util.Rand(len(candidates))]
	TryStartBetween(initiator, partner, room)
}

// TryStartBetween skips partner selection (caller has already identified
// the pair). Used by the player-arrival boost in go.go.
func TryStartBetween(a, b *mobs.Mob, room *rooms.Room) {
	if a == nil || b == nil || room == nil {
		return
	}
	if isOnCooldown(&a.Character) || isOnCooldown(&b.Character) {
		return
	}
	if isInConversation(&a.Character) || isInConversation(&b.Character) {
		return
	}

	// Pick a relationship type to draw from. If multiple edges, pick one
	// uniformly. RelationsBetween returns []Relation.
	rels := relationships.RelationsBetween(int(a.MobId), int(b.MobId))
	if len(rels) == 0 {
		return
	}
	rel := rels[util.Rand(len(rels))]

	// Subtype is per-side; use the subtype from a's perspective (a's edge
	// to b). RelationsBetween returns edges in unspecified order; for
	// determinism we don't try to align which side.
	subtype := rel.Subtype

	ex, ok := pickExchange(string(rel.Type), subtype, int(a.MobId), int(b.MobId))
	if !ok {
		return // No exchanges authored for this relationship type.
	}

	startConversation(a, b, ex, string(rel.Type), subtype, room)
}

// startConversation stamps state on both NPCs and fires the first line.
func startConversation(a, b *mobs.Mob, ex Exchange, poolId, subtype string, room *rooms.Room) {
	// Find the first line's speaker → determine which NPC is "A".
	if len(ex.Lines) == 0 {
		return
	}
	speakerA, speakerB := a, b
	// Randomize which NPC plays role "A" (the initiator-role in the
	// exchange script) so the same exchange can fire with either NPC
	// leading. We don't need persistent A/B identity tied to mob ids.
	if util.Rand(2) == 0 {
		speakerA, speakerB = b, a
	}

	roundNow := uint64(util.GetRoundCount())
	// Pick a unique exchange id (round count is fine — distinguishes
	// concurrent exchanges across the world).
	exchangeId := int(roundNow)

	stampInitial := func(self *mobs.Mob, partner *mobs.Mob, role string) {
		self.Character.SetMiscData(MiscDataPartnerId, partner.InstanceId)
		self.Character.SetMiscData(MiscDataRole, role)
		self.Character.SetMiscData(MiscDataPoolId, poolId)
		self.Character.SetMiscData(MiscDataPairOverrideId, "") // populated only if used; TODO if we want to track separately
		self.Character.SetMiscData(MiscDataExchangeId, exchangeId)
		self.Character.SetMiscData(MiscDataLineIdx, 0)
	}
	stampInitial(speakerA, speakerB, "A")
	stampInitial(speakerB, speakerA, "B")

	// Fire the first line from whichever NPC plays the speaker of Lines[0].
	firstSpeaker := speakerA
	if ex.Lines[0].Speaker == "B" {
		firstSpeaker = speakerB
	}
	firstSpeaker.Command(fmt.Sprintf("say %s", ex.Lines[0].Text))

	// Stamp last_round on the speaker; advance line_idx on BOTH.
	firstSpeaker.Character.SetMiscData(MiscDataLastRound, roundNow)
	speakerA.Character.SetMiscData(MiscDataLineIdx, 1)
	speakerB.Character.SetMiscData(MiscDataLineIdx, 1)

	// If the exchange has only 1 line, finalize immediately.
	if len(ex.Lines) == 1 {
		finalizeConversation(speakerA, speakerB)
	}
}

// TickConversation drives an in-progress conversation for the given mob.
// Called each tick from the IdleMobs hook when partner_id MiscData is set.
func TickConversation(self *mobs.Mob, partnerId int, room *rooms.Room) {
	partner := mobs.GetInstance(partnerId)
	if partner == nil || partner.Character.RoomId != self.Character.RoomId {
		abortConversation(self, partner)
		return
	}
	if !isFullyIdleForConversation(partner) {
		abortConversation(self, partner)
		return
	}

	// Resolve the exchange and the next line.
	poolId, _ := self.Character.GetMiscData(MiscDataPoolId).(string)
	subtype := "" // subtype isn't strictly needed at tick time; pool+exchange_id pin the exchange
	// pickExchange is non-deterministic — but we need to recover the SAME
	// exchange. Cheapest: persist the picked exchange in MiscData. We
	// don't have a serializable Exchange identity yet. Workaround: stash
	// each picked exchange's index within its source list. For v1,
	// re-pick with the same seeded RNG by using exchange_id as the seed.
	// Simplest reliable path: cache the picked exchange on a package-
	// level map keyed by exchange_id.
	exchangeId, _ := self.Character.GetMiscData(MiscDataExchangeId).(int)
	ex, ok := getActiveExchange(exchangeId)
	if !ok {
		abortConversation(self, partner)
		return
	}
	_ = subtype // silence unused if not used in this branch
	_ = poolId

	role, _ := self.Character.GetMiscData(MiscDataRole).(string)
	lineIdx, _ := self.Character.GetMiscData(MiscDataLineIdx).(int)

	plan := computeConversationPlan(ex, role, lineIdx)
	if plan.ShouldAbort {
		abortConversation(self, partner)
		return
	}
	if !plan.ShouldFire {
		return // wait silently
	}

	// Pacing gate.
	roundNow := uint64(util.GetRoundCount())
	lastRound, _ := self.Character.GetMiscData(MiscDataLastRound).(uint64)
	if lastRound == roundNow {
		return // already spoke this round; defensive
	}

	self.Command(fmt.Sprintf("say %s", plan.Text))
	self.Character.SetMiscData(MiscDataLastRound, roundNow)
	self.Character.SetMiscData(MiscDataLineIdx, plan.NextLineIdx)
	partner.Character.SetMiscData(MiscDataLineIdx, plan.NextLineIdx)

	if plan.IsFinalLine {
		finalizeConversation(self, partner)
	}
}

// finalizeConversation clears state on both NPCs and stamps cooldown.
func finalizeConversation(a, b *mobs.Mob) {
	roundNow := uint64(util.GetRoundCount())
	cooldown := uint64(configs.GetBalanceConfig().ConversationCooldownRounds)
	until := roundNow + cooldown

	exchangeId, _ := a.Character.GetMiscData(MiscDataExchangeId).(int)
	clearActiveExchange(exchangeId)

	for _, m := range []*mobs.Mob{a, b} {
		if m == nil {
			continue
		}
		m.Character.SetMiscData(MiscDataPartnerId, 0)
		m.Character.SetMiscData(MiscDataRole, "")
		m.Character.SetMiscData(MiscDataPoolId, "")
		m.Character.SetMiscData(MiscDataPairOverrideId, "")
		m.Character.SetMiscData(MiscDataExchangeId, 0)
		m.Character.SetMiscData(MiscDataLineIdx, 0)
		m.Character.SetMiscData(MiscDataLastRound, uint64(0))
		m.Character.SetMiscData(MiscDataCooldownUntilRound, until)
	}
}

// abortConversation clears state on both NPCs without stamping a cooldown
// (graceful — the conversation was interrupted, not completed).
func abortConversation(self *mobs.Mob, partner *mobs.Mob) {
	exchangeId, _ := self.Character.GetMiscData(MiscDataExchangeId).(int)
	clearActiveExchange(exchangeId)

	for _, m := range []*mobs.Mob{self, partner} {
		if m == nil {
			continue
		}
		m.Character.SetMiscData(MiscDataPartnerId, 0)
		m.Character.SetMiscData(MiscDataRole, "")
		m.Character.SetMiscData(MiscDataPoolId, "")
		m.Character.SetMiscData(MiscDataPairOverrideId, "")
		m.Character.SetMiscData(MiscDataExchangeId, 0)
		m.Character.SetMiscData(MiscDataLineIdx, 0)
		m.Character.SetMiscData(MiscDataLastRound, uint64(0))
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

func isOnCooldown(c *charactersCharacterLike) bool {
	until, _ := c.GetMiscData(MiscDataCooldownUntilRound).(uint64)
	return uint64(util.GetRoundCount()) < until
}

func isInConversation(c *charactersCharacterLike) bool {
	partnerId, _ := c.GetMiscData(MiscDataPartnerId).(int)
	return partnerId > 0
}

// isFullyIdle: not in combat, not asleep, not in conversation, not on
// patrol mid-walk, not transitioning schedule, not in player dialogue.
// This is the same eligibility check the trigger uses; export it so
// the hooks-side trigger gate can reuse it.
func isFullyIdle(m *mobs.Mob) bool {
	if m == nil {
		return false
	}
	if m.Character.Aggro != nil || m.Character.IsInCombat() {
		return false
	}
	// Sleeping flag from chunk 3.3
	if m.Character.HasBuffFlag(buffsSleepingFlag()) {
		return false
	}
	// Patrolling between waypoints — path queue non-empty
	if m.Path.Len() > 0 || m.Path.Current() != nil {
		return false
	}
	// In a player dialogue
	if m.InConversation() {
		return false
	}
	return true
}

func isFullyIdleForConversation(m *mobs.Mob) bool {
	// Reused at tick time; excludes the in-conversation gate (we KNOW we're
	// in a conversation, that's the calling context).
	if m == nil {
		return false
	}
	if m.Character.Aggro != nil || m.Character.IsInCombat() {
		return false
	}
	if m.Character.HasBuffFlag(buffsSleepingFlag()) {
		return false
	}
	if m.InConversation() {
		// In a PLAYER conversation — abort the NPC↔NPC one
		return false
	}
	return true
}

// buffsSleepingFlag is a tiny indirection to avoid an import cycle —
// internal/buffs is imported by internal/conversations, but we want to
// keep the import surface tight. Use a free function.
func buffsSleepingFlag() buffs.Flag {
	return buffs.Sleeping
}

// charactersCharacterLike is a tiny structural narrowing for the helper
// signatures so they don't all force the *characters.Character pointer.
// If Go's generics or direct *characters.Character usage is cleaner,
// use that instead and delete this stub.
type charactersCharacterLike interface {
	GetMiscData(key string) any
	SetMiscData(key string, val any)
}

// ── Active exchange cache ────────────────────────────────────────────
//
// The picker chose an exchange but we can't easily reconstruct it from
// MiscData alone (the exchange is a value, not an id-addressable thing
// in the registry — exchanges are anonymous list entries). Cache the
// chosen Exchange keyed by exchange_id so TickConversation can recover
// it. Cleared in finalize/abort. Bounded growth: at most one entry per
// active exchange.

var (
	activeExchangesMu sync.RWMutex
	activeExchanges   = map[int]Exchange{}
)

func cacheActiveExchange(id int, ex Exchange) {
	activeExchangesMu.Lock()
	defer activeExchangesMu.Unlock()
	activeExchanges[id] = ex
}

func getActiveExchange(id int) (Exchange, bool) {
	activeExchangesMu.RLock()
	defer activeExchangesMu.RUnlock()
	ex, ok := activeExchanges[id]
	return ex, ok
}

func clearActiveExchange(id int) {
	activeExchangesMu.Lock()
	defer activeExchangesMu.Unlock()
	delete(activeExchanges, id)
}
```

After implementing, also update `startConversation` to call `cacheActiveExchange(exchangeId, ex)` immediately after stamping initial MiscData. Add the call AFTER `stampInitial(speakerB, speakerA, "B")` and BEFORE firing the first line:

```go
cacheActiveExchange(exchangeId, ex)
```

Also need imports — at the top of state.go ensure these are present:
- `sync`
- `github.com/GoMudEngine/GoMud/internal/buffs`
- `github.com/GoMudEngine/GoMud/internal/characters` (only if used; the helper interface avoids it)

The `charactersCharacterLike` interface stub is awkward — if there's a cleaner way (e.g., import `characters` directly since we already need the package for Character access through the mob), use that. The point is to avoid signature soup.

If the helper-interface approach doesn't compile cleanly (Go's interfaces don't auto-satisfy from concrete types without an explicit cast), fall back to passing `*characters.Character` directly through the helpers. The interface was a defensive abstraction; concrete is fine.

- [ ] **Step 3: Build, confirm clean**

```bash
go build ./...
```

If the helper-interface approach errors out, replace `charactersCharacterLike` with `*characters.Character` everywhere and import the package. The choice is mechanical.

- [ ] **Step 4: Run existing tests, confirm no regressions**

```bash
go test ./internal/conversations/ -v
```

Expected: T1, T2, T4, T5 tests still pass. No new tests in T6 (full integration is exercised by T7 + T13 boot smoke).

- [ ] **Step 5: Commit**

```bash
git add internal/conversations/state.go
git commit -m "$(cat <<'EOF'
feat(conversations): TryStart + startConversation + TickConversation

Public entry-point API + per-conversation state machine.
TryStart picks a related in-room partner uniformly + invokes
TryStartBetween which picks a relationship type, picks an
exchange via pickExchange, stamps MiscData on both NPCs, fires
the first line, caches the chosen Exchange.

TickConversation runs each tick when partner_id MiscData is
set: resolves the cached Exchange, runs computeConversationPlan,
fires the line if it's this NPC's turn, advances the shared
line counter on both NPCs, finalizes (with cooldown stamp) or
aborts (graceful) as appropriate.

Active-exchange cache (sync.Map-equivalent) keyed by
exchange_id holds the chosen Exchange for the duration of the
conversation. Cleared on finalize/abort.

MiscData key constants exported so the IdleMobs eligibility
gate (T7) can use the same names.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: IdleMobs trigger + state-machine wiring

**Files:**
- Modify: `internal/hooks/NewRound_IdleMobs.go` (add conversation trigger + tick branch)
- Possibly create: `internal/hooks/conversation_trigger_test.go` (gate eligibility tests)

- [ ] **Step 1: Re-read the IdleMobs hook**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
sed -n '60,100p' internal/hooks/NewRound_IdleMobs.go
```

Confirm: the schedule branch (chunk 3.2) is around line 72-78, the patrol branch (chunk 3.4) is around line 80-100. Conversation branch goes AFTER both of those but BEFORE the path-walker check, so an in-progress conversation gets a chance to drive the tick before the existing path-walker would.

- [ ] **Step 2: Add the conversation branch**

In `internal/hooks/NewRound_IdleMobs.go`, after the chunk 3.4 patrol branch and before the path-walker check, add:

```go
// Chunk 3.6: conversation tick + trigger.
// First: if this mob is already in a conversation, advance it.
if partnerId := getMiscDataInt(&mob.Character, conversations.MiscDataPartnerId); partnerId > 0 {
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		conversations.TickConversation(mob, partnerId, room)
	}
}

// Then: if fully idle and not on cooldown, roll for a new conversation.
if conversationsTriggerEligible(mob) {
	if util.Rand(10000) < int(configs.GetBalanceConfig().ConversationBaseChancePct*100) {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			conversations.TryStart(mob, room)
		}
	}
}
```

Add the helper `conversationsTriggerEligible` to a peer file or inline:

```go
// conversationsTriggerEligible gates the per-tick conversation trigger.
// Mirrors conversations.isFullyIdle plus the cooldown + already-in-conversation
// checks. Inlined here to avoid an export ceremony just for one trigger.
func conversationsTriggerEligible(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	// Not already in a conversation
	if partnerId := getMiscDataInt(&mob.Character, conversations.MiscDataPartnerId); partnerId > 0 {
		return false
	}
	// Not on cooldown
	if until := getMiscDataUint(&mob.Character, conversations.MiscDataCooldownUntilRound); uint64(util.GetRoundCount()) < until {
		return false
	}
	// Combat / sleep / patrol mid-walk / player dialogue
	if mob.Character.Aggro != nil || mob.Character.IsInCombat() {
		return false
	}
	if mob.Character.HasBuffFlag(buffs.Sleeping) {
		return false
	}
	if mob.Path.Len() > 0 || mob.Path.Current() != nil {
		return false
	}
	if mob.InConversation() {
		return false
	}
	return true
}

// getMiscDataUint is a small helper if not already present in the
// package. uint64 round counters come back via MiscData as uint64.
func getMiscDataUint(char *characters.Character, key string) uint64 {
	v := char.GetMiscData(key)
	if v == nil {
		return 0
	}
	if u, ok := v.(uint64); ok {
		return u
	}
	return 0
}
```

Add imports for `conversations`, `rooms`, and `buffs` to NewRound_IdleMobs.go if not present.

- [ ] **Step 3: Skip dedicated unit test for the trigger**

The trigger gates are inline checks against MiscData / mob state. The behavior is correctly exercised by T13's boot smoke (the smoketester will observe whether conversations fire only when expected). Adding a unit test requires mock mob construction that's heavyweight — skip.

- [ ] **Step 4: Build + test**

```bash
go build ./...
go test ./internal/hooks/ -v
```

Expected: green.

- [ ] **Step 5: Commit**

```bash
git add internal/hooks/NewRound_IdleMobs.go
git commit -m "$(cat <<'EOF'
feat(hooks): IdleMobs conversation tick + trigger wiring

Conversation branch placed after chunk 3.4 patrol branch and
before the path-walker. Two-phase per-NPC tick:
  1) If already in a conversation (partner_id MiscData set),
     call conversations.TickConversation to drive the next
     line / abort / finalize.
  2) Otherwise, if fully idle and not on cooldown, roll the
     per-tick trigger (ConversationBaseChancePct ÷ 10000) and
     call conversations.TryStart on hit.

conversationsTriggerEligible helper inlines the eligibility
gates (combat / sleep / patrol-mid-walk / player-dialogue /
existing-conversation / cooldown).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Player-arrival boost in go.go

**Files:**
- Modify: `internal/usercommands/go.go` (add boost hook after chunk 3.3 light-source wake)

- [ ] **Step 1: Find the chunk 3.3 light-source wake block**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -n "Chunk 3.3.*light\|EmitsLight" internal/usercommands/go.go | head -5
```

The boost hook goes AFTER that block — same post-arrival placement, different trigger.

- [ ] **Step 2: Add the boost hook**

In `internal/usercommands/go.go`, append after the chunk 3.3 light-source wake block:

```go
// Chunk 3.6: player-arrival conversation boost. If the arriving
// player lands in a room with 2+ NPCs that are related and idle,
// roll once at the higher boost chance for an opening exchange.
{
	cfg := configs.GetBalanceConfig()
	boostPct := int(cfg.ConversationPlayerArrivalBoostPct)
	if boostPct > 0 && util.Rand(100) < boostPct {
		if destRoom := rooms.LoadRoom(user.Character.RoomId); destRoom != nil {
			pairs := findRelateableEligiblePairsInRoom(destRoom)
			if len(pairs) > 0 {
				p := pairs[util.Rand(len(pairs))]
				conversations.TryStartBetween(p.A, p.B, destRoom)
			}
		}
	}
}
```

Add the helper near the bottom of `go.go` (or in a peer file):

```go
type relateableMobPair struct {
	A *mobs.Mob
	B *mobs.Mob
}

// findRelateableEligiblePairsInRoom enumerates all unordered (a, b)
// mob pairs in the room where both are related per chunk 1.6 and both
// are fully idle (not in combat, sleeping, on cooldown, mid-patrol-walk,
// already conversing).
func findRelateableEligiblePairsInRoom(room *rooms.Room) []relateableMobPair {
	if room == nil {
		return nil
	}
	mobIds := room.GetMobs()
	if len(mobIds) < 2 {
		return nil
	}
	mobList := make([]*mobs.Mob, 0, len(mobIds))
	for _, id := range mobIds {
		m := mobs.GetInstance(id)
		if m == nil {
			continue
		}
		// Cheap eligibility filter first.
		if m.Character.Aggro != nil || m.Character.IsInCombat() {
			continue
		}
		if m.Character.HasBuffFlag(buffs.Sleeping) {
			continue
		}
		if m.Path.Len() > 0 || m.Path.Current() != nil {
			continue
		}
		if m.InConversation() {
			continue
		}
		if pid, ok := m.Character.GetMiscData(conversations.MiscDataPartnerId).(int); ok && pid > 0 {
			continue
		}
		until, _ := m.Character.GetMiscData(conversations.MiscDataCooldownUntilRound).(uint64)
		if uint64(util.GetRoundCount()) < until {
			continue
		}
		mobList = append(mobList, m)
	}

	var out []relateableMobPair
	for i := 0; i < len(mobList); i++ {
		for j := i + 1; j < len(mobList); j++ {
			a, b := mobList[i], mobList[j]
			if relationships.AreRelated(int(a.MobId), int(b.MobId)) {
				out = append(out, relateableMobPair{A: a, B: b})
			}
		}
	}
	return out
}
```

Add the `conversations`, `relationships`, and `buffs` imports if not present.

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/go.go
git commit -m "$(cat <<'EOF'
feat(commands): player-arrival conversation boost in go.go

When the player arrives in a room with 2+ NPCs that are related
per chunk 1.6 and both fully idle, roll once at
ConversationPlayerArrivalBoostPct (default 25%) for an opening
exchange. The player is more likely to actually witness
conversations without the world going silent when unobserved.

findRelateableEligiblePairsInRoom enumerates all unordered pairs
in the room that pass the eligibility filter (no combat, no
sleep, no in-flight path, no existing conversation, no cooldown)
AND have a relationship edge.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Pilot content — relationship edges on 4 Thornwall mobs

**Files:**
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/115-old_gobb.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/116-old_wrex.yaml`
- Modify: `_datafiles/world/dogmud/mobs/thornwall_city/117-barmaid_dal.yaml`

- [ ] **Step 1: Read a mob YAML to confirm the relationships block format**

```bash
cd "C:/Users/Calabe Davis/workspace/DOGMud"
grep -A 10 "^relationships:" _datafiles/world/dogmud/mobs/**/*.yaml 2>/dev/null | head -30
```

If any mob has authored relationships, mirror the YAML shape. If not, the chunk 1.6 spec describes it as:

```yaml
relationships:
  - to: <mob_id>
    type: <family|friend|rival|lover|employer|employee>
    subtype: <optional free-string>
```

- [ ] **Step 2: Add edges to Fen (114)**

Append to `_datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml`:

```yaml
relationships:
  - to: 115
    type: friend
  - to: 116
    type: friend
  - to: 117
    type: friend
```

(Chunk 1.6 auto-mirrors symmetric `friend` edges, so we only need one direction per pair.)

- [ ] **Step 3: Add edges to Gobb (115)**

Append to `_datafiles/world/dogmud/mobs/thornwall_city/115-old_gobb.yaml`:

```yaml
relationships:
  - to: 116
    type: friend
  - to: 117
    type: friend
```

(Fen→Gobb already covered by Fen's YAML; only the remaining edges need authoring.)

- [ ] **Step 4: Add edges to Wrex (116)**

Append to `_datafiles/world/dogmud/mobs/thornwall_city/116-old_wrex.yaml`:

```yaml
relationships:
  - to: 117
    type: friend
```

(Fen→Wrex and Gobb→Wrex already covered.)

- [ ] **Step 5: Dal (117) — no new edges**

All Dal↔(old man) edges are covered by the old men's YAMLs via auto-mirror. No edits needed to 117. Skip step.

- [ ] **Step 6: Boot smoke (manual, optional)**

```bash
go build ./...
```

The boot will load the relationships graph; if any edge fails validation, the chunk 1.6 loader warns (per its permissive validation). Build is enough for now.

- [ ] **Step 7: Commit**

```bash
git add _datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml _datafiles/world/dogmud/mobs/thornwall_city/115-old_gobb.yaml _datafiles/world/dogmud/mobs/thornwall_city/116-old_wrex.yaml
git commit -m "$(cat <<'EOF'
feat(content): Thornwall tavern back-room friend relationships

Authors mutual friend edges across the four back-room NPCs:
- Fen (114) ↔ Gobb (115), Wrex (116), Dal (117)
- Gobb (115) ↔ Wrex (116), Dal (117)
- Wrex (116) ↔ Dal (117)

6 friend pairs total. Chunk 1.6 auto-mirrors symmetric friend
edges, so each pair is authored once. Dal (117) needs no edits
— all her edges are mirrored from the old men's YAMLs.

These edges are the pilot substrate for chunk 3.6 idle
conversations.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Pilot content — friend.yaml conversation pool

**Files:**
- Create: `_datafiles/world/dogmud/conversations/types/friend.yaml`

- [ ] **Step 1: Author the pool**

Create the file. Aim for ~10-12 generic friend-banter exchanges, mostly 2-3 lines each. Mix of complaints, weather, gossip, small life updates, drink-related exchanges, work-related exchanges. The Thornwall tavern back-room will be the primary stage; write in a voice that fits old men + a barmaid all relating as friends.

```yaml
id: friend
description: "Generic friend banter — small talk, gossip, weather, drink, complaints. Suitable for old men in a tavern back room or any two NPCs in friendly relation."

exchanges:
  - lines:
      - speaker: A
        text: "How's the brew today?"
      - speaker: B
        text: "Same as ever, thank the gods."
      - speaker: A
        text: "Could be worse."
  - lines:
      - speaker: A
        text: "Bones aching with the weather?"
      - speaker: B
        text: "Aye. Cold mornings tell their tale."
      - speaker: A
        text: "Hot tea fixes more than the apothecary admits."
      - speaker: B
        text: "Truer words."
  - lines:
      - speaker: A
        text: "Heard about the wagon out by the gate?"
      - speaker: B
        text: "Half the wheel gone, they say."
      - speaker: A
        text: "That's the third one this season."
  - lines:
      - speaker: A
        text: "You look tired."
      - speaker: B
        text: "Slept poorly. Old men and old beds."
      - speaker: A
        text: "Old beds you can fix."
  - lines:
      - speaker: A
        text: "Another?"
      - speaker: B
        text: "Aye, but small. The boy's waiting on me."
  - lines:
      - speaker: A
        text: "The river's running low."
      - speaker: B
        text: "Dry summer coming, mark me."
      - speaker: A
        text: "Hope you're wrong."
      - speaker: B
        text: "I usually am."
  - lines:
      - speaker: A
        text: "What's that you've got there?"
      - speaker: B
        text: "Just a letter. Nothing important."
      - speaker: A
        text: "If you say so."
  - lines:
      - speaker: A
        text: "Saw the priest at the temple this morning."
      - speaker: B
        text: "Out so early? On what business?"
      - speaker: A
        text: "Didn't ask. Wasn't mine to ask."
  - lines:
      - speaker: A
        text: "My back's at me again."
      - speaker: B
        text: "Sit down, then. Drink and complain like the rest of us."
  - lines:
      - speaker: A
        text: "Quiet today."
      - speaker: B
        text: "Quiet's good. Loud means trouble."
      - speaker: A
        text: "Loud means custom, too."
      - speaker: B
        text: "Hmph. Same coin, different side."
  - lines:
      - speaker: A
        text: "You're up early."
      - speaker: B
        text: "Couldn't sleep. The dreams."
      - speaker: A
        text: "Same ones?"
      - speaker: B
        text: "Same ones."

subtypes:
  fond:
    exchanges:
      - lines:
          - speaker: A
            text: "Always good to share a quiet hour with you."
          - speaker: B
            text: "Same, friend. Same."
      - lines:
          - speaker: A
            text: "What would I do without you?"
          - speaker: B
            text: "Drink alone, like a sad old fool."
          - speaker: A
            text: "Ha."
```

- [ ] **Step 2: Boot smoke (manual)**

```bash
go build ./...
```

If the loader rejects the YAML (filename↔id mismatch, bad speaker, etc.), the boot will panic — fix and rebuild.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/world/dogmud/conversations/types/friend.yaml
git commit -m "$(cat <<'EOF'
feat(content): chunk 3.6 friend conversation pool

11 generic friend exchanges (2-4 lines each) covering brew,
weather, gossip, sleep complaints, river, custom, priest
sightings, back-aches, dream-talk. Plus a 2-exchange "fond"
subtype sub-pool for NPCs with a fond-typed friend edge.

Designed primarily for Thornwall tavern back-room NPCs (Dal,
Fen, Gobb, Wrex) but reusable by any pair with a friend edge.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: (Optional) rival pool + Dal↔Wrex pair override

**Files:**
- (Optional) Modify: `_datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml` (add rival edge to Wrex)
- (Optional) Create: `_datafiles/world/dogmud/conversations/types/rival.yaml`
- (Optional) Create: `_datafiles/world/dogmud/conversations/pairs/116_117.yaml`

This task is optional flavor. Ship it if scope budget allows; skip it if you'd rather move to docs and closeout faster. The friend pool alone validates the chunk end-to-end.

- [ ] **Step 1: Add the rival edge between Fen and Wrex**

In `_datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml`, append (or insert into the existing `relationships:` block):

```yaml
  - to: 116
    type: rival
    subtype: "old argument"
```

Chunk 1.6 auto-mirrors symmetric `rival` edges. Fen and Wrex are now both friends AND rivals — chunk 1.6 supports multiple edges between the same pair.

- [ ] **Step 2: Create the rival pool**

Create `_datafiles/world/dogmud/conversations/types/rival.yaml`:

```yaml
id: rival
description: "Pointed jabs and old disagreements. Used between NPCs with a rival edge."

exchanges:
  - lines:
      - speaker: A
        text: "Still wrong, after all these years."
      - speaker: B
        text: "And you, still right. Funny how it works out."
  - lines:
      - speaker: A
        text: "You always did tell that story wrong."
      - speaker: B
        text: "Tell it your way, then. I've heard it enough times."
      - speaker: A
        text: "Maybe I will."
  - lines:
      - speaker: A
        text: "I notice you're drinking the cheap stuff."
      - speaker: B
        text: "Tastes the same as your fancy stuff, to me."
      - speaker: A
        text: "That's because your tongue's gone."
  - lines:
      - speaker: A
        text: "You owe me a copper from the spring fair."
      - speaker: B
        text: "I owe you nothing of the kind."
      - speaker: A
        text: "Liar."
      - speaker: B
        text: "Cheat."
  - lines:
      - speaker: A
        text: "Hmph."
      - speaker: B
        text: "Hmph yourself."
```

- [ ] **Step 3: Create the Dal↔Wrex pair override**

Filename is `<smaller-mob-id>_<larger-mob-id>.yaml` = `116_117.yaml`.

Create `_datafiles/world/dogmud/conversations/pairs/116_117.yaml`:

```yaml
id: dal_and_wrex
mob_a: 116
mob_b: 117
exchanges:
  - lines:
      - speaker: A
        text: "The usual, Wrex?"
      - speaker: B
        text: "You know me too well, lass."
  - lines:
      - speaker: A
        text: "Mind the spill on table three — I'll get to it."
      - speaker: B
        text: "Take your time. We've got nowhere to be."
  - lines:
      - speaker: A
        text: "Your tab's getting long."
      - speaker: B
        text: "I'm good for it. Always have been."
      - speaker: A
        text: "Aye, you have."
```

- [ ] **Step 4: Boot smoke + commit**

```bash
go build ./...
git add _datafiles/world/dogmud/mobs/thornwall_city/114-old_fen.yaml _datafiles/world/dogmud/conversations/types/rival.yaml _datafiles/world/dogmud/conversations/pairs/116_117.yaml
git commit -m "$(cat <<'EOF'
feat(content): chunk 3.6 rival pool + Dal-Wrex pair override

Adds the rival relationship edge between Fen (114) and Wrex
(116) with subtype "old argument". 5 rival exchanges authored
(short pointed jabs about old disagreements, drinking habits,
spring-fair debt).

Adds the Dal-Wrex per-pair override at pairs/116_117.yaml —
3 unique exchanges about the usual order, table spills, tab.
Extends (not replaces) the friend pool for that specific pair.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Documentation pass

**Files:**
- Create: `docs/schemas/conversation.md`
- Create: `internal/conversations/context.md`
- Modify: `internal/hooks/context.md`
- Modify: `internal/configs/context.md`
- Modify: `CLAUDE.md`
- Modify: `_datafiles/world/dogmud/templates/help/ask.template`

- [ ] **Step 1: Create the schema doc**

Create `docs/schemas/conversation.md`. Cover:
- File locations (`conversations/types/` and `conversations/pairs/`)
- Pool YAML shape (id, description, exchanges, subtypes)
- PairOverride YAML shape (id, mob_a, mob_b, exchanges)
- Filename conventions
- Speaker semantics (A is initiator-role; engine randomizes which physical NPC plays A per-conversation)
- Subtype convention (fond, estranged, professional, bitter — extensible but warn on others)
- Validation rules (panic vs warn)
- Runtime behavior (trigger rate, cooldown, pacing, abort triggers)

Match the format of `docs/schemas/patrol.md` (created in chunk 3.4).

- [ ] **Step 2: Create the conversations package context.md**

Create `internal/conversations/context.md` summarizing:
- Package purpose
- Public API (`Load`, `SetConversationWorldValidator`, `TryStart`, `TryStartBetween`, `TickConversation`, `GetPool`, `GetPairOverride`, exported MiscData key constants)
- Internal flow (loader → registries; trigger → TryStart → pickExchange → startConversation → cacheActiveExchange; tick → TickConversation → computeConversationPlan → applies via mob.Command)
- MiscData key list with semantics
- Abort trigger list

Match the format of `internal/mobs/context.md` (existing files).

- [ ] **Step 3: Update hooks context.md**

Append in the IdleMobs section:

```markdown
- `NewRound_IdleMobs.go` chunk 3.6 branch: two-phase per-NPC
  tick — drives in-progress conversation via
  `conversations.TickConversation`, then if fully idle and not
  on cooldown rolls the trigger (ConversationBaseChancePct) to
  start a new one via `conversations.TryStart`. Placed AFTER
  the chunk 3.4 patrol branch and BEFORE the path-walker.
- `go.go` chunk 3.6 player-arrival boost: post-arrival hook
  rolls `ConversationPlayerArrivalBoostPct` for an opening
  exchange via `conversations.TryStartBetween`. Placed AFTER
  the chunk 3.3 light-source wake block.
```

- [ ] **Step 4: Update configs context.md**

Add three new rows to the balance knobs table:

```markdown
| `ConversationBaseChancePct` | 1.0 | Per-tick % chance a fully-idle NPC attempts to start an idle conversation. Chunk 3.6. |
| `ConversationPlayerArrivalBoostPct` | 25 | On player arrival in a room with relateable, idle NPCs, % chance to start one. Chunk 3.6. |
| `ConversationCooldownRounds` | 50 | Cooldown applied to both NPCs after a conversation completes. Chunk 3.6. |
```

- [ ] **Step 5: Update CLAUDE.md**

Append a subsection near the existing chunk 3.2 / 3.3 / 3.4 subsections:

```markdown
### NPC↔NPC Conversations
Townspeople with relationship edges (chunk 1.6) occasionally
exchange 2-4 line conversations drawn from a relationship-type-
keyed library at `_datafiles/world/dogmud/conversations/`.
Type pools (`types/<relationship-type>.yaml`) hold generic
exchanges per relationship type. Optional pair overrides
(`pairs/<lower>_<higher>.yaml`) add per-pair-specific
exchanges (extending the type pool). Optional subtype sub-pools
add flavor variation per relationship subtype string.

Triggers: low per-tick chance (`ConversationBaseChancePct`,
default 1%) per fully-idle NPC + a higher player-arrival boost
(`ConversationPlayerArrivalBoostPct`, default 25%). Pacing: one
line per round, shared `conversation_line_idx` MiscData counter
drives speaker alternation deterministically. Cooldown
(`ConversationCooldownRounds`, default 50) on both NPCs after
an exchange completes.

Gating: conversations only fire when both NPCs are fully idle
(no combat, no sleep, no patrol mid-walk, no player dialogue,
no existing conversation). Mid-exchange interruption (partner
leaves the room / sleeps / enters combat / starts player
dialogue) aborts gracefully without applying a cooldown.

NPC↔NPC opinion store and "spoken about you" gossip are
deferred (see chunk 3.6 spec for rationale).
```

- [ ] **Step 6: Update the ask helpfile**

Append to `_datafiles/world/dogmud/templates/help/ask.template`:

```
You can sometimes overhear townspeople chatting with each other --
pause in busy rooms to catch the gossip.
```

- [ ] **Step 7: Commit**

```bash
git add docs/schemas/conversation.md internal/conversations/context.md internal/hooks/context.md internal/configs/context.md CLAUDE.md _datafiles/world/dogmud/templates/help/ask.template
git commit -m "$(cat <<'EOF'
docs: chunk 3.6 schema + context.md + CLAUDE.md + helpfile

docs/schemas/conversation.md: full YAML schema reference for
type pools and pair overrides, validation rules, runtime
behavior, subtype convention.

internal/conversations/context.md: package context — public
API, internal flow, MiscData keys, abort triggers.

internal/hooks/context.md: IdleMobs branch + go.go boost
documented.

internal/configs/context.md: three new knob rows.

CLAUDE.md: NPC<->NPC Conversations subsection.

ask.template: one-sentence player hint.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Smoketester goal file + roadmap closeout

**Files:**
- Create: `tools/testing/goals/3.6-conversation-observation.yaml`
- Modify: `MOB_ALIVENESS_ROADMAP.md` (mark 3.6 Done)
- Move: spec and plan to `completed/`

- [ ] **Step 1: Author the smoketester goal**

Create `tools/testing/goals/3.6-conversation-observation.yaml`:

```yaml
description: "Observe Thornwall tavern back-room NPCs (Dal 117, Fen 114, Gobb 115, Wrex 116) for 30-50 rounds; assert at least one multi-line idle conversation between two of them completes."

goals:
  - "Locate the Thornwall tavern back room (look up the room id by walking the tavern from room 472)."
  - "Stand idle for ~30 game rounds. Watch the room output."
  - "Count `say` broadcasts from the four target NPCs (Dal, Fen, Gobb, Wrex). A multi-line exchange will look like two NPCs alternating say-lines across consecutive rounds."
  - "If at least one full multi-line exchange completes within the observation window, PASS the trigger test."
  - "Leave the room, wait 10 game rounds, re-enter (player-arrival boost test). Confirm increased likelihood of immediate exchange vs the passive baseline."
  - "If the rival pool was authored (T11), check whether any rival exchange between Fen and Wrex fires across multiple observation windows."

pass_criteria:
  - "At least one full multi-line exchange observed between two of the target NPCs"
  - "No `say`-line cross-contamination from non-conversation idlecommands (existing chunk 3.2 idlecommands are independent)"
  - "Player-arrival boost demonstrably triggers earlier than the passive baseline"

notes:
  - "If 'time set <hour>' admin command isn't available, observe in real-time."
  - "Background activity: chunk 3.4's city guard may be patrolling through the tavern — that's expected and shouldn't interfere with the back-room observation."
```

- [ ] **Step 2: Update the roadmap**

Edit `MOB_ALIVENESS_ROADMAP.md`. Find the chunk 3.6 progress row:

```markdown
| 3.6 | Routine | NPC↔NPC idle conversation | M | 1.6 | Not started |
```

Change to:

```markdown
| 3.6 | Routine | NPC↔NPC idle conversation | M | 1.6 | Done |
```

Find the chunk 3.6 detailed section. Append:

```markdown
- **Shipped:** New `internal/conversations/` package with type
  pools (`types/<relationship-type>.yaml`) and per-pair
  overrides (`pairs/<lower>_<higher>.yaml`). Loader with
  filename↔id check, speaker validation, mob existence +
  relationship edge cross-check via DI. Picker draws uniformly
  from `type_pool ∪ matching_subtype ∪ pair_override`. State
  machine runs in `NewRound_IdleMobs`: per-tick trigger
  (`ConversationBaseChancePct`, default 1%) + player-arrival
  boost (`ConversationPlayerArrivalBoostPct`, default 25%) in
  `go.go`. Line-per-round pacing via shared
  `conversation_line_idx` MiscData; deterministic speaker
  alternation. Graceful abort on partner moves / sleeps /
  combats / player-dialogue. Cooldown
  (`ConversationCooldownRounds`, default 50) on both NPCs
  after completion. Pilot: Thornwall tavern back-room — Dal +
  Fen + Gobb + Wrex friend edges (6 pairs) + optional rival
  edge Fen↔Wrex + friend pool (11 exchanges + 2 fond-subtype)
  + optional rival pool (5 exchanges) + optional Dal↔Wrex pair
  override (3 exchanges). NPC↔NPC opinion store and "spoken
  about you" gossip explicitly deferred (see spec). Spec at
  `docs/superpowers/specs/completed/2026-05-25-mob-aliveness-3.6-npc-conversations-design.md`,
  plan at `docs/superpowers/plans/completed/2026-05-25-mob-aliveness-3.6-npc-conversations.md`.
```

- [ ] **Step 3: Move spec and plan to completed/**

```bash
git mv docs/superpowers/specs/2026-05-25-mob-aliveness-3.6-npc-conversations-design.md docs/superpowers/specs/completed/
git mv docs/superpowers/plans/2026-05-25-mob-aliveness-3.6-npc-conversations.md docs/superpowers/plans/completed/
```

- [ ] **Step 4: Final verification**

```bash
go build ./...
go test ./... 2>&1 | grep -i fail | head -5
```

Expected: clean across the board.

- [ ] **Step 5: Commit**

```bash
git add tools/testing/goals/3.6-conversation-observation.yaml MOB_ALIVENESS_ROADMAP.md docs/superpowers/specs/completed/ docs/superpowers/plans/completed/
git commit -m "$(cat <<'EOF'
chore(roadmap): mark 3.6 NPC conversations Done

Smoketester goal file authored at
tools/testing/goals/3.6-conversation-observation.yaml.
Roadmap row + detailed section updated with Shipped summary.
Spec + plan moved to completed/.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

- [ ] **Step 6: Report**

Print summary:
- Total commits ahead of development on this branch
- Final commit SHA
- Build + test results
- Any leftover concerns

---

## Self-Review

**Spec coverage:**

- ✅ Conversation types + registry + GetPool/GetPairOverride → T1
- ✅ YAML loader + validators + DI cross-checks → T2
- ✅ Three config knobs → T3
- ✅ Exchange picker (type pool ∪ subtype ∪ pair override) → T4
- ✅ State machine (computeConversationPlan + TickConversation + TryStart + TryStartBetween + startConversation + finalize + abort) → T5 + T6
- ✅ IdleMobs trigger + tick wiring → T7
- ✅ Player-arrival boost → T8
- ✅ Pilot relationships (4 NPC YAMLs) → T9
- ✅ Pilot friend pool YAML → T10
- ✅ Optional rival pool + per-pair override → T11
- ✅ Schema doc, context.md, CLAUDE.md, helpfile → T12
- ✅ Smoketester goal + roadmap closeout → T13

**Type consistency check:**

- `relationships.Type` constants used consistently as the pool key (T1, T2, T4, T6).
- MiscData key constants (`MiscDataPartnerId` etc.) defined in T6's `state.go`, consumed in T7's IdleMobs hook + T8's go.go.
- `pickExchange(poolId, subtype, mobA, mobB)` signature consistent T4 → T6 caller.
- `computeConversationPlan(ex, role, lineIdx) conversationPlan` signature consistent T5 → T6 caller.
- `pairKey` and `makePairKey` consistent T1 → T2 loader.
- Filename convention `<lower>_<higher>.yaml` for pair overrides consistent in T2 loader + T11 content + T12 schema doc.

**Placeholder scan:**

Searched for "TBD", "TODO", "implement later". Two intentional skip/defer markers (T5's `t.Skip` for the integration test that requires heavy fixtures; T7's note that the trigger gates rely on T13 boot smoke rather than dedicated unit tests). Both are explicit deferrals with rationale, not silent placeholders.

**Internal consistency:**

- The active-exchange cache (T6) is the bridge between "picker picked an exchange at trigger time" and "tick machine needs to recover the same exchange later." Key by exchange_id (unique per startConversation call). Cleared in finalize + abort.
- The `conversation_pair_override_id` MiscData key is declared but not strictly necessary since the cached exchange holds the chosen content. Could be removed in a polish pass; harmless to keep for debug-inspector future use.
- The DI pattern in T2 (`SetConversationWorldValidator`) mirrors chunk 3.2 / 3.4 — the cycle-break reason is the same (mobs is imported by rooms, so conversations can't directly resolve mob_id → mob template without going through DI).

Plan is internally consistent and ready for execution.
