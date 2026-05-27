# Mob Aliveness 4.3 — Goal Types Catalog Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Author 13 narrow goal types over the 4.1 substrate + 4.2 selection engine, plus three engine deltas (ParamSchema validation, AllowMultiple + DedupKey for multi-instance types, archetype lazy-seed sentinel) and sparse archetype defaults that seed `survival` everywhere with `wealth-gold` extras for thief / shopkeeper.

**Architecture:** Pure-function Predicate + ContextScore + DedupKey per type, each in its own file under `internal/goals/catalog/`. The catalog subpackage's `init()` funcs register types via the existing `goals.RegisterGoalType` API; `main.go` adds a blank import to fire all the inits. Archetype YAML grows a `default_goals:` block parsed by `behaviortree.LoadArchetypeYAMLFromFile`; a `SetArchetypeDefaultsLookup` callback bridges goals → behaviortree without an import cycle (mirrors 4.2's `SetWeightsLookup`).

**Tech Stack:** Go 1.25 · `gopkg.in/yaml.v3` (goals) + `gopkg.in/yaml.v2` (behaviortree) · existing `mudlog`, `configs`, `util`, `mobs`, `behaviortree`, `factions`, `opinions`, `rooms` packages.

**Spec:** `docs/superpowers/specs/2026-05-27-mob-aliveness-4.3-goal-types-catalog-design.md`

---

## Task 1 — ParamSchema struct + ValidateParams + Add wiring

Engine delta #1. Declarative param validation lets per-type Predicates assume well-typed params at read time. Schema-less types (4.1 behavior) pass through unchanged.

**Files:**
- Modify: `internal/goals/types.go` (add `ParamSchema` struct + `Params` field on `GoalTypeMeta` + `ErrBadParams` typed error)
- Create: `internal/goals/validation.go` (`ValidateParams` function)
- Create: `internal/goals/validation_test.go` (table-driven tests)
- Modify: `internal/goals/store.go` (wire `ValidateParams` into `Add` before conflict check)
- Modify: `internal/goals/store_test.go` (integration test: Add rejects bad params)

- [ ] **Step 1.1: Write failing validation tests**

Create `internal/goals/validation_test.go`:

```go
package goals

import (
	"errors"
	"testing"
)

func TestValidateParams_NoSchema_AllParamsPass(t *testing.T) {
	g := &Goal{Type: "freeform", Params: map[string]any{"anything": 1, "goes": "here"}}
	if err := ValidateParams(g, nil); err != nil {
		t.Errorf("got err=%v, want nil (no schema = no validation)", err)
	}
}

func TestValidateParams_RequiredKeyPresent_Pass(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil", err)
	}
}

func TestValidateParams_RequiredKeyMissing_Fails(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{}}
	err := ValidateParams(g, schema)
	if err == nil {
		t.Fatalf("got nil, want ErrBadParams")
	}
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if bpe.Key != "target" {
		t.Errorf("bpe.Key=%q, want %q", bpe.Key, "target")
	}
}

func TestValidateParams_WrongType_Fails(t *testing.T) {
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "wealth-gold", Params: map[string]any{"target": "five hundred"}}
	err := ValidateParams(g, schema)
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if bpe.ExpectedType != "int" {
		t.Errorf("bpe.ExpectedType=%q, want int", bpe.ExpectedType)
	}
	if bpe.GotType != "string" {
		t.Errorf("bpe.GotType=%q, want string", bpe.GotType)
	}
}

func TestValidateParams_OptionalKeyMissing_Pass(t *testing.T) {
	schema := []ParamSchema{
		{Key: "target", Required: true, GoType: "int"},
		{Key: "threshold", Required: false, GoType: "int"},
	}
	g := &Goal{Type: "x", Params: map[string]any{"target": 1}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil (optional key absent is fine)", err)
	}
}

func TestValidateParams_IntFromInt64_Pass(t *testing.T) {
	// YAML round-trips integers as int64; the validator should accept either.
	schema := []ParamSchema{{Key: "target", Required: true, GoType: "int"}}
	g := &Goal{Type: "x", Params: map[string]any{"target": int64(500)}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("got err=%v, want nil (int64 should satisfy int schema)", err)
	}
}

func TestValidateParams_StringSlice_AcceptsBothShapes(t *testing.T) {
	// YAML can unmarshal a string list as []interface{} OR []string depending on path.
	schema := []ParamSchema{{Key: "tags", Required: true, GoType: "[]string"}}

	g1 := &Goal{Type: "x", Params: map[string]any{"tags": []any{"a", "b"}}}
	if err := ValidateParams(g1, schema); err != nil {
		t.Errorf("[]any{string}: got err=%v, want nil", err)
	}

	g2 := &Goal{Type: "x", Params: map[string]any{"tags": []string{"a", "b"}}}
	if err := ValidateParams(g2, schema); err != nil {
		t.Errorf("[]string: got err=%v, want nil", err)
	}

	g3 := &Goal{Type: "x", Params: map[string]any{"tags": []any{"a", 5}}}
	if err := ValidateParams(g3, schema); err == nil {
		t.Errorf("[]any with non-string element: want err, got nil")
	}
}

func TestValidateParams_FloatFromInt_Pass(t *testing.T) {
	schema := []ParamSchema{{Key: "ratio", Required: true, GoType: "float64"}}
	g := &Goal{Type: "x", Params: map[string]any{"ratio": 5}}
	if err := ValidateParams(g, schema); err != nil {
		t.Errorf("int → float64: got err=%v, want nil", err)
	}
}

func TestValidateParams_BoolStrict(t *testing.T) {
	schema := []ParamSchema{{Key: "flag", Required: true, GoType: "bool"}}
	if err := ValidateParams(&Goal{Type: "x", Params: map[string]any{"flag": true}}, schema); err != nil {
		t.Errorf("bool true: got err=%v, want nil", err)
	}
	if err := ValidateParams(&Goal{Type: "x", Params: map[string]any{"flag": 1}}, schema); err == nil {
		t.Errorf("int as bool: want err, got nil")
	}
}
```

- [ ] **Step 1.2: Run tests to verify they fail**

Run: `go test ./internal/goals/ -run "TestValidateParams_" -v`
Expected: FAIL — `ParamSchema` / `ValidateParams` / `ErrBadParams` undefined.

- [ ] **Step 1.3: Add `ParamSchema` + `Params` field + `ErrBadParams`**

In `internal/goals/types.go`, add after `GoalTypeMeta` (currently around lines 44-48 from chunk 4.2):

```go
// ParamSchema declares one expected key on Goal.Params for type-aware
// validation. Used by ValidateParams (chunk 4.3).
type ParamSchema struct {
	Key      string
	Required bool
	GoType   string // "int" | "string" | "[]string" | "float64" | "bool"
}

// ErrBadParams is returned by Add when a goal's params don't match its
// type's declared ParamSchema.
type ErrBadParams struct {
	Key          string
	ExpectedType string
	GotType      string
	Reason       string // optional — e.g. "missing required key"
}

func (e *ErrBadParams) Error() string {
	if e.Reason != "" {
		return fmt.Sprintf("goals.ErrBadParams: key=%q %s", e.Key, e.Reason)
	}
	return fmt.Sprintf("goals.ErrBadParams: key=%q expected=%s got=%s",
		e.Key, e.ExpectedType, e.GotType)
}
```

Then extend `GoalTypeMeta` to add the `Params` field. Replace the existing struct:

```go
// GoalTypeMeta is registered once per goal type by chunk 4.3's catalog.
type GoalTypeMeta struct {
	Predicate     PredicateFn
	ConflictsWith []string
	ContextScore  ContextScoreFn // chunk 4.2 — optional; nil = always 1.0
	Params        []ParamSchema  // chunk 4.3 — optional; nil = no validation
}
```

Add `"fmt"` to imports if not already present.

- [ ] **Step 1.4: Create `internal/goals/validation.go`**

```go
package goals

// ValidateParams checks g.Params against the registered type's schema.
// Returns *ErrBadParams on failure (use errors.As to inspect).
// nil schema → no validation (matches 4.1 freeform behavior).
//
// Chunk 4.3.
func ValidateParams(g *Goal, schema []ParamSchema) error {
	if len(schema) == 0 {
		return nil
	}
	for _, ps := range schema {
		raw, present := g.Params[ps.Key]
		if !present {
			if ps.Required {
				return &ErrBadParams{Key: ps.Key, ExpectedType: ps.GoType, Reason: "missing required key"}
			}
			continue
		}
		if !matchesGoType(raw, ps.GoType) {
			return &ErrBadParams{Key: ps.Key, ExpectedType: ps.GoType, GotType: goTypeName(raw)}
		}
	}
	return nil
}

// matchesGoType reports whether raw satisfies the declared GoType.
// Permissive on numeric widening (int64 → int, int → float64) since
// YAML round-trips integers as int64 and floats can absorb ints.
func matchesGoType(raw any, goType string) bool {
	switch goType {
	case "int":
		switch raw.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		}
		return false
	case "string":
		_, ok := raw.(string)
		return ok
	case "[]string":
		switch v := raw.(type) {
		case []string:
			return true
		case []any:
			for _, e := range v {
				if _, ok := e.(string); !ok {
					return false
				}
			}
			return true
		}
		return false
	case "float64":
		switch raw.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		}
		return false
	case "bool":
		_, ok := raw.(bool)
		return ok
	}
	return false // unknown GoType in schema is treated as no-match
}

// goTypeName returns a printable Go type name for error messages.
func goTypeName(raw any) string {
	switch raw.(type) {
	case int, int8, int16, int32, int64:
		return "int"
	case uint, uint8, uint16, uint32, uint64:
		return "uint"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case bool:
		return "bool"
	case []any:
		return "[]any"
	case []string:
		return "[]string"
	}
	return "unknown"
}
```

- [ ] **Step 1.5: Wire ValidateParams into Add**

In `internal/goals/store.go`, find `Add` (currently at line 86). Insert validation immediately after `mg := loadOrLazyInit(...)` and before `cacheMu.Lock()`:

```go
func Add(mobId int, namesimple string, g *Goal) (AddResult, error) {
	mg := loadOrLazyInit(mobId, namesimple)

	// Chunk 4.3: validate params against the registered type's schema.
	if meta, ok := lookupMeta(g.Type); ok {
		if err := ValidateParams(g, meta.Params); err != nil {
			return AddResult{}, err
		}
	}

	cacheMu.Lock()
	// ...rest unchanged...
```

- [ ] **Step 1.6: Add integration test that Add rejects bad params**

Append to `internal/goals/store_test.go`:

```go
func TestAdd_ParamSchemaViolation_RejectsGoal(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-paramreq", GoalTypeMeta{
		Params: []ParamSchema{{Key: "target", Required: true, GoType: "int"}},
	})
	defer resetRegistry()

	mobId := 99301
	name := "param_test_mob"
	bad := &Goal{Type: "test-paramreq", Priority: 50, Params: map[string]any{}}
	_, err := Add(mobId, name, bad)
	if err == nil {
		t.Fatalf("Add returned nil error for missing required param")
	}
	var bpe *ErrBadParams
	if !errors.As(err, &bpe) {
		t.Fatalf("err type: got %T, want *ErrBadParams", err)
	}
	if got := goals.GoalsOf(mobId, name); len(got) != 0 {
		t.Errorf("goal added despite validation failure: %v", got)
	}
}
```

⚠️ The `goals.GoalsOf` reference at the end is **inside** the goals package itself — drop the `goals.` prefix (just `GoalsOf(mobId, name)`). Also add `"errors"` to the test file's imports if not present.

- [ ] **Step 1.7: Run tests to verify they pass**

Run: `go test ./internal/goals/ -run "TestValidateParams_|TestAdd_ParamSchemaViolation" -v`
Expected: PASS.

Run: `go test ./internal/goals/...`
Expected: PASS (existing 4.1 + 4.2 tests still pass — schema-less types are unaffected).

- [ ] **Step 1.8: Commit**

```bash
git add internal/goals/types.go internal/goals/validation.go internal/goals/validation_test.go internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): declarative ParamSchema validation in Add (4.3)" -m "GoalTypeMeta gains a Params []ParamSchema field. Add() validates a
goal's Params against its type's declared schema before the conflict
check; failures return ErrBadParams with key + expected/got types.
Numeric widening (int64→int, int→float64) and []any-of-strings paths
are tolerated to absorb YAML unmarshal quirks. Schema-less types
(4.1 freeform behavior) pass through unchanged.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2 — AllowMultiple + DedupKey on GoalTypeMeta + Add conflict logic

Engine delta #2. Lets multi-instance types (revenge against different targets, training multiple skills) coexist on one mob while still de-duping equal-target instances.

**Files:**
- Modify: `internal/goals/types.go` (add `AllowMultiple` + `DedupKey` fields on `GoalTypeMeta`)
- Modify: `internal/goals/store.go` (extend conflict-detection in `Add`; wrap DedupKey in panic recovery)
- Modify: `internal/goals/store_test.go` (multi-instance + dedup tests)

- [ ] **Step 2.1: Write failing tests**

Append to `internal/goals/store_test.go`:

```go
func TestAdd_AllowMultiple_DifferentDedupKeys_BothAdded(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if t, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(t)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99401
	name := "multi_diff_mob"
	g1 := &Goal{Type: "test-multi", Priority: 50, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi", Priority: 50, Params: map[string]any{"target": 2}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	if _, err := Add(mobId, name, g2); err != nil {
		t.Fatalf("Add g2 (different dedup key): %v", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 2 {
		t.Errorf("expected 2 coexisting goals, got %d: %v", len(got), got)
	}
}

func TestAdd_AllowMultiple_SameDedupKey_ConflictsByPriority(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi2", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if t, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(t)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99402
	name := "multi_same_mob"
	g1 := &Goal{Type: "test-multi2", Priority: 90, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi2", Priority: 30, Params: map[string]any{"target": 1}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	_, err := Add(mobId, name, g2)
	if err == nil {
		t.Fatalf("Add g2 (same dedup key, lower priority) should have conflicted")
	}
	var ce *ConflictError
	if !errors.As(err, &ce) {
		t.Fatalf("err type: got %T, want *ConflictError", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 1 {
		t.Errorf("expected 1 goal (blocker preserved), got %d", len(got))
	}
}

func TestAdd_AllowMultiple_SameDedupKey_HigherPriority_Displaces(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-multi3", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			if t, ok := g.Params["target"].(int); ok {
				return strconv.Itoa(t)
			}
			return ""
		},
	})
	defer resetRegistry()

	mobId := 99403
	name := "multi_displace_mob"
	g1 := &Goal{Type: "test-multi3", Priority: 30, Params: map[string]any{"target": 1}}
	g2 := &Goal{Type: "test-multi3", Priority: 90, Params: map[string]any{"target": 1}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	res, err := Add(mobId, name, g2)
	if err != nil {
		t.Fatalf("Add g2 (same dedup, higher priority): %v", err)
	}
	if len(res.Displaced) != 1 {
		t.Errorf("expected 1 displaced, got %v", res.Displaced)
	}
}

func TestAdd_AllowMultipleFalse_StillBlocksSameType(t *testing.T) {
	// Confirms 4.1 behavior preserved for types that don't opt in.
	ClearCache()
	RegisterGoalType("test-singleton", GoalTypeMeta{})
	defer resetRegistry()

	mobId := 99404
	name := "single_mob"
	g1 := &Goal{Type: "test-singleton", Priority: 50}
	g2 := &Goal{Type: "test-singleton", Priority: 50}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1: %v", err)
	}
	_, err := Add(mobId, name, g2)
	if err == nil {
		t.Fatalf("expected ConflictError on second add of singleton type")
	}
}

func TestAdd_DedupKey_PanicRecovered_FallsThroughToNoKeyCollision(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-panicky", GoalTypeMeta{
		AllowMultiple: true,
		DedupKey: func(g *Goal) string {
			panic("dedup-key boom")
		},
	})
	defer resetRegistry()

	mobId := 99405
	name := "panicky_mob"
	g1 := &Goal{Type: "test-panicky", Priority: 50, Params: map[string]any{"a": 1}}
	g2 := &Goal{Type: "test-panicky", Priority: 50, Params: map[string]any{"a": 2}}
	if _, err := Add(mobId, name, g1); err != nil {
		t.Fatalf("Add g1 with panicking dedup-key: %v", err)
	}
	if _, err := Add(mobId, name, g2); err != nil {
		t.Fatalf("Add g2 with panicking dedup-key: %v", err)
	}
	if got := GoalsOf(mobId, name); len(got) != 2 {
		t.Errorf("expected 2 goals (panic → empty key → coexist), got %d", len(got))
	}
}
```

Add `"strconv"` to the test file's imports if not already present (`"errors"` was added in Task 1).

- [ ] **Step 2.2: Run tests to verify they fail**

Run: `go test ./internal/goals/ -run "TestAdd_AllowMultiple_|TestAdd_AllowMultipleFalse|TestAdd_DedupKey_Panic" -v`
Expected: FAIL — `AllowMultiple` / `DedupKey` fields undefined.

- [ ] **Step 2.3: Add the two new fields to GoalTypeMeta**

In `internal/goals/types.go`, replace the `GoalTypeMeta` struct:

```go
// GoalTypeMeta is registered once per goal type by chunk 4.3's catalog.
type GoalTypeMeta struct {
	Predicate     PredicateFn
	ConflictsWith []string
	ContextScore  ContextScoreFn       // chunk 4.2
	Params        []ParamSchema        // chunk 4.3
	AllowMultiple bool                 // chunk 4.3 — multi-instance allowed
	DedupKey      func(g *Goal) string // chunk 4.3 — when AllowMultiple, returns dedup key
}
```

- [ ] **Step 2.4: Add the panic-recovered DedupKey invocation helper**

Append to `internal/goals/store.go` (near the bottom, next to `instanceForRecompute`):

```go
// invokeDedupKey calls a registered DedupKey func under panic recovery.
// A panic logs a single-line warning and returns "" (collapses to
// "no key" — same-type goals fall through to coexist freely under
// AllowMultiple semantics). Mirrors how invokeContextScore handles
// panics. Chunk 4.3.
func invokeDedupKey(fn func(g *Goal) string, g *Goal) (key string) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("goals.dedup_key panic",
				"type", g.Type,
				"goal_id", g.Id,
				"panic", fmt.Sprintf("%v", r))
			key = ""
		}
	}()
	return fn(g)
}
```

- [ ] **Step 2.5: Extend Add's conflict-detection loop**

In `internal/goals/store.go`, find the section in `Add` that builds the `conflicting` slice (currently lines ~95-101):

```go
	newMeta, _ := lookupMeta(g.Type)
	var conflicting []*Goal
	for _, e := range mg.Goals {
		if isConflict(g.Type, e.Type, newMeta) {
			conflicting = append(conflicting, e)
		}
	}
```

Replace with:

```go
	newMeta, _ := lookupMeta(g.Type)
	var newKey string
	if newMeta.AllowMultiple && newMeta.DedupKey != nil {
		newKey = invokeDedupKey(newMeta.DedupKey, g)
	}
	var conflicting []*Goal
	for _, e := range mg.Goals {
		if g.Type == e.Type {
			// Chunk 4.3: same-type pair. AllowMultiple=true + matching
			// DedupKey collides; AllowMultiple=false keeps 4.1's
			// "same type always conflicts" semantics.
			if newMeta.AllowMultiple {
				if newMeta.DedupKey != nil && newKey != "" {
					existingKey := invokeDedupKey(newMeta.DedupKey, e)
					if existingKey == newKey {
						conflicting = append(conflicting, e)
					}
				}
				// AllowMultiple + no key match → coexist (do nothing).
				continue
			}
			conflicting = append(conflicting, e)
			continue
		}
		// Cross-type: 4.1's type-name-based ConflictsWith lookup.
		if isConflict(g.Type, e.Type, newMeta) {
			conflicting = append(conflicting, e)
		}
	}
```

⚠️ Note: existing `isConflict` (4.1) handles cross-type via `ConflictsWith`. We keep that exact same call for the cross-type branch. The same-type path is what this task rewrites.

- [ ] **Step 2.6: Run tests to verify they pass**

Run: `go test ./internal/goals/ -run "TestAdd_" -v`
Expected: all PASS, including existing 4.1 / 4.2 / Task 1 / new Task 2 tests.

Run: `go test ./internal/goals/...`
Expected: PASS.

- [ ] **Step 2.7: Commit**

```bash
git add internal/goals/types.go internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): AllowMultiple + DedupKey for multi-instance types (4.3)" -m "GoalTypeMeta gains AllowMultiple bool and DedupKey func. Add's
conflict-detection branches: same-type pair with AllowMultiple=true
collides only when DedupKey matches; cross-type still uses 4.1's
ConflictsWith. Panic-recovered DedupKey invocation mirrors 4.2's
ContextScore wrapper — a panic logs and collapses to empty key,
which under AllowMultiple semantics means coexist.

Schema-less / opt-out types (AllowMultiple=false) preserve 4.1
'same type always conflicts' behavior verbatim.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3 — ArchetypeDefaultsLookupFn callback (goals side)

Engine delta #3a. Defines the goals → behaviortree bridge for archetype default-goal lookups. Mirrors 4.2's `SetWeightsLookup` pattern. No archetype data parsed yet (Task 5) — this task ships the API.

**Files:**
- Modify: `internal/goals/types.go` (add `GoalDefault` struct)
- Modify: `internal/goals/lookup.go` (extend with `ArchetypeDefaultsLookupFn` + `SetArchetypeDefaultsLookup` + `resolveArchetypeDefaults`)
- Modify: `internal/goals/lookup_test.go` (registration test)

- [ ] **Step 3.1: Write the failing test**

Append to `internal/goals/lookup_test.go`:

```go
func TestSetArchetypeDefaultsLookup_Registered_Resolves(t *testing.T) {
	called := false
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		called = true
		return []GoalDefault{
			{Type: "survival", Priority: 80},
			{Type: "wealth-gold", Priority: 40, Params: map[string]any{"target": 500}},
		}
	})
	defer SetArchetypeDefaultsLookup(nil)

	got := resolveArchetypeDefaults(&mobs.Mob{})
	if !called {
		t.Errorf("defaults lookup callback not invoked")
	}
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Type != "survival" || got[1].Type != "wealth-gold" {
		t.Errorf("types: got [%s, %s], want [survival, wealth-gold]", got[0].Type, got[1].Type)
	}
}

func TestResolveArchetypeDefaults_NoLookup_ReturnsNil(t *testing.T) {
	SetArchetypeDefaultsLookup(nil)
	got := resolveArchetypeDefaults(&mobs.Mob{})
	if got != nil {
		t.Errorf("got=%v, want nil", got)
	}
}
```

- [ ] **Step 3.2: Run tests to verify they fail**

Run: `go test ./internal/goals/ -run "TestSetArchetypeDefaultsLookup_|TestResolveArchetypeDefaults_" -v`
Expected: FAIL — `SetArchetypeDefaultsLookup` / `resolveArchetypeDefaults` / `GoalDefault` undefined.

- [ ] **Step 3.3: Add `GoalDefault` to types.go**

In `internal/goals/types.go`, append at the bottom:

```go
// GoalDefault is the goals-package mirror of behaviortree.GoalDefault.
// Kept separate (rather than imported across the package boundary) to
// avoid an internal/goals → internal/behaviortree import cycle.
// Chunk 4.3.
type GoalDefault struct {
	Type     string
	Priority int
	Params   map[string]any
}
```

- [ ] **Step 3.4: Extend lookup.go**

Append to `internal/goals/lookup.go`:

```go
// ArchetypeDefaultsLookupFn returns the archetype's default goal list
// for the given mob. Registered once at boot from main.go as a thin
// adapter over behaviortree.GetEngine().GetArchetypeDefaultGoals
// (returning the goals-package mirror type). Chunk 4.3.
type ArchetypeDefaultsLookupFn func(mob *mobs.Mob) []GoalDefault

var archetypeDefaultsLookup ArchetypeDefaultsLookupFn // guarded by lookupMu

// SetArchetypeDefaultsLookup registers the archetype-defaults resolver.
// Pass nil to unregister (tests use this for isolation). Chunk 4.3.
func SetArchetypeDefaultsLookup(fn ArchetypeDefaultsLookupFn) {
	lookupMu.Lock()
	archetypeDefaultsLookup = fn
	lookupMu.Unlock()
}

// resolveArchetypeDefaults returns the archetype defaults for a mob,
// or nil if no lookup is registered. Internal — called by the lazy-
// seed path in loadOrLazyInit. Chunk 4.3.
func resolveArchetypeDefaults(mob *mobs.Mob) []GoalDefault {
	lookupMu.RLock()
	fn := archetypeDefaultsLookup
	lookupMu.RUnlock()
	if fn == nil {
		return nil
	}
	return fn(mob)
}
```

(The `lookupMu sync.RWMutex` is the existing 4.2 lookup mutex — reuse it for both weights and defaults to keep the API minimal.)

- [ ] **Step 3.5: Run tests to verify they pass**

Run: `go test ./internal/goals/ -run "TestSetArchetypeDefaultsLookup_|TestResolveArchetypeDefaults_" -v`
Expected: PASS.

- [ ] **Step 3.6: Commit**

```bash
git add internal/goals/types.go internal/goals/lookup.go internal/goals/lookup_test.go
git commit -m "feat(goals): ArchetypeDefaultsLookupFn callback (4.3)" -m "Mirrors chunk 4.2's SetWeightsLookup pattern. Defines GoalDefault
struct (goals-package mirror of the behaviortree type — kept
separate to avoid import cycle) + SetArchetypeDefaultsLookup
registration + internal resolveArchetypeDefaults helper. Main.go
will register the adapter once at boot in a later task.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 4 — SeededFromArchetype field + lazy-seed branch

Engine delta #3b. New mobs (no existing MobGoals file) get seeded from archetype defaults on first `GoalsOf` access. Sentinel field on the file prevents re-seeding.

**Files:**
- Modify: `internal/goals/types.go` (add `SeededFromArchetype` field)
- Modify: `internal/goals/persistence_test.go` (round-trip + legacy-file tests)
- Modify: `internal/goals/store.go` (extend `loadOrLazyInit`; add `seedFromArchetype` helper)
- Modify: `internal/goals/store_test.go` (lazy-seed integration tests)

- [ ] **Step 4.1: Write failing persistence round-trip + legacy-load tests**

Append to `internal/goals/persistence_test.go`:

```go
func TestMobGoals_SeededFromArchetype_RoundTrip(t *testing.T) {
	mg := &MobGoals{
		MobId:               371,
		NextGoalId:          2,
		SeededFromArchetype: true,
		Goals: []*Goal{
			{Id: "g1", Type: "survival", Priority: 80,
				CreatedAt: time.Date(2026, 5, 27, 10, 0, 0, 0, time.UTC)},
		},
	}
	out, err := yaml.Marshal(mg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var got MobGoals
	if err := yaml.Unmarshal(out, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if !got.SeededFromArchetype {
		t.Errorf("SeededFromArchetype: got false, want true")
	}
}

func TestMobGoals_LegacyFile_LoadsWithSentinelFalse(t *testing.T) {
	legacy := `mob_id: 371
next_goal_id: 2
goals:
  - id: g1
    type: survival
    priority: 80
    created_at: 2026-05-27T10:00:00Z
`
	var got MobGoals
	if err := yaml.Unmarshal([]byte(legacy), &got); err != nil {
		t.Fatalf("unmarshal legacy: %v", err)
	}
	if got.SeededFromArchetype {
		t.Errorf("SeededFromArchetype: got true, want false (legacy file)")
	}
}
```

- [ ] **Step 4.2: Add `SeededFromArchetype` field to `MobGoals`**

In `internal/goals/types.go`, replace `MobGoals`:

```go
// MobGoals is the on-disk shape — one file per mob template.
type MobGoals struct {
	MobId               int     `yaml:"mob_id"`
	NextGoalId          int     `yaml:"next_goal_id"`
	CurrentGoalId       string  `yaml:"current_goal_id,omitempty"`         // 4.2
	CurrentSinceRound   uint64  `yaml:"current_since_round,omitempty"`     // 4.2
	LastSwitchRound     uint64  `yaml:"last_switch_round,omitempty"`       // 4.2
	SeededFromArchetype bool    `yaml:"seeded_from_archetype,omitempty"`   // 4.3
	Goals               []*Goal `yaml:"goals"`
}
```

- [ ] **Step 4.3: Run round-trip tests to verify pass**

Run: `go test ./internal/goals/ -run "TestMobGoals_SeededFromArchetype_|TestMobGoals_LegacyFile_LoadsWithSentinelFalse" -v`
Expected: PASS.

- [ ] **Step 4.4: Write failing lazy-seed integration tests**

Append to `internal/goals/store_test.go`:

```go
func TestLoadOrLazyInit_FreshMob_NoLookup_SetsSentinelTrueNoSeeds(t *testing.T) {
	ClearCache()
	SetArchetypeDefaultsLookup(nil)
	mobId := 99501
	name := "fresh_no_lookup"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("SeededFromArchetype=false, want true (sentinel must flip even with nil lookup)")
	}
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals, got %d", len(mg.Goals))
	}
}

func TestLoadOrLazyInit_FreshMob_WithLookup_SeedsAndPersists(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-seedable", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{{Type: "test-seedable", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99502
	name := "fresh_with_seeds"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("SeededFromArchetype=false, want true")
	}
	if len(mg.Goals) != 1 {
		t.Fatalf("expected 1 seeded goal, got %d", len(mg.Goals))
	}
	if mg.Goals[0].Type != "test-seedable" || mg.Goals[0].Priority != 80 {
		t.Errorf("seeded goal mismatch: %+v", mg.Goals[0])
	}
}

func TestLoadOrLazyInit_ExistingFileWithSentinelTrue_SkipsSeed(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-skipseed", GoalTypeMeta{})
	defer resetRegistry()
	called := 0
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		called++
		return []GoalDefault{{Type: "test-skipseed", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99503
	name := "skip_seed_mob"
	cacheStoreForTest(name, &MobGoals{MobId: mobId, NextGoalId: 1, SeededFromArchetype: true})
	mg := loadOrLazyInit(mobId, name)
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals (no seed), got %d", len(mg.Goals))
	}
	if called > 0 {
		t.Errorf("lookup called %d times; expected 0", called)
	}
}

func TestClear_PreservesSentinel(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-preserve", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{{Type: "test-preserve", Priority: 80}}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99504
	name := "clear_preserve_mob"
	_ = loadOrLazyInit(mobId, name)
	if err := Clear(mobId, name); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("sentinel cleared by Clear — should be preserved")
	}
	if len(mg.Goals) != 0 {
		t.Errorf("expected 0 goals after Clear+reload, got %d", len(mg.Goals))
	}
}

func TestLoadOrLazyInit_SeededDefaultFailsValidation_LogsAndContinues(t *testing.T) {
	ClearCache()
	RegisterGoalType("test-strict", GoalTypeMeta{
		Params: []ParamSchema{{Key: "target", Required: true, GoType: "int"}},
	})
	RegisterGoalType("test-loose", GoalTypeMeta{})
	defer resetRegistry()
	SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []GoalDefault {
		return []GoalDefault{
			{Type: "test-strict", Priority: 80}, // missing required param
			{Type: "test-loose", Priority: 40},  // seeds successfully
		}
	})
	defer SetArchetypeDefaultsLookup(nil)

	mobId := 99505
	name := "partial_seed_mob"
	mg := loadOrLazyInit(mobId, name)
	if !mg.SeededFromArchetype {
		t.Errorf("sentinel should flip even when one default failed")
	}
	if len(mg.Goals) != 1 || mg.Goals[0].Type != "test-loose" {
		t.Errorf("expected only test-loose to seed, got %v", mg.Goals)
	}
}
```

- [ ] **Step 4.5: Run tests to verify they fail**

Run: `go test ./internal/goals/ -run "TestLoadOrLazyInit_|TestClear_PreservesSentinel" -v`
Expected: FAIL — seeding logic not wired yet.

- [ ] **Step 4.6: Wire `seedFromArchetype` into `loadOrLazyInit`**

In `internal/goals/store.go`, modify `loadOrLazyInit`. After a fresh `*MobGoals` is created and stored in the cache (but BEFORE the function returns), call `seedFromArchetype(mobId, namesimple, mg)`.

**Critical sequencing:** the seed runs AFTER `cache[mobId] = mg` and AFTER any write lock is released. The seed calls `Add`, which calls `loadOrLazyInit` again — to avoid infinite recursion, the cache entry must be present and the lock free before invoking the seed. Read the current `loadOrLazyInit` body before editing to confirm the right insertion point.

Add the helper near the bottom of `store.go`:

```go
// seedFromArchetype runs the chunk-4.3 lazy seed for a fresh MobGoals.
// Idempotent under the sentinel. Always flips the sentinel + persists
// once at the end, regardless of seed outcome. Add failures are logged
// at warn level and skipped — partial seeding is preferable to bailing.
func seedFromArchetype(mobId int, namesimple string, mg *MobGoals) {
	if mg.SeededFromArchetype {
		return
	}
	mob := instanceForRecompute(mobId)
	defaults := resolveArchetypeDefaults(mob)
	for _, d := range defaults {
		g := &Goal{Type: d.Type, Priority: d.Priority, Params: d.Params}
		if _, err := Add(mobId, namesimple, g); err != nil {
			mudlog.Warn("goals.seedFromArchetype: Add failed (skipping)",
				"mob_id", mobId, "type", d.Type, "error", err)
		}
	}
	cacheMu.Lock()
	mg.SeededFromArchetype = true
	cacheMu.Unlock()
	if err := saveToDisk(mobId, namesimple); err != nil {
		mudlog.Warn("goals.seedFromArchetype: save failed",
			"mob_id", mobId, "error", err)
	}
}
```

- [ ] **Step 4.7: Run tests to verify they pass**

Run: `go test ./internal/goals/ -run "TestLoadOrLazyInit_|TestClear_PreservesSentinel" -v`
Expected: PASS.

Run: `go test ./internal/goals/...`
Expected: PASS.

- [ ] **Step 4.8: Commit**

```bash
git add internal/goals/types.go internal/goals/persistence_test.go internal/goals/store.go internal/goals/store_test.go
git commit -m "feat(goals): lazy archetype-default seeding + sentinel (4.3)" -m "MobGoals gains SeededFromArchetype bool. loadOrLazyInit calls
seedFromArchetype on fresh mobs: invokes the registered
ArchetypeDefaultsLookupFn, routes each default through Add (so
ParamSchema validation + dedup fire naturally), then flips the
sentinel and persists once. Add failures log + skip.

Sentinel preserved across admin Clear. Backward compat: existing
4.1/4.2 files load with sentinel=false; first post-deploy GoalsOf
triggers seeding (intentional — appends survival to existing goal
lists for most mobs).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5 — Behaviortree side: parse `default_goals:` from archetype YAML

The behaviortree archetype loader gains a third optional top-level field (`default_goals:`) alongside `tree:` and `goal_weights:`. Cache it on Engine; expose via `GetArchetypeDefaultGoals`.

**Files:**
- Modify: `internal/behaviortree/types.go` (extend `TreeDef`; declare `GoalDefault`)
- Modify: `internal/behaviortree/loader.go` (extend `LoadArchetypeYAMLFromFile`)
- Modify: `internal/behaviortree/engine.go` (cache + accessor; extend `LoadArchetype`)
- Modify: `internal/behaviortree/test_export.go` (cleanup for new cache)
- Modify: `internal/behaviortree/engine_goal_weights_test.go` (extend `newEngineForTest`)
- Create: `internal/behaviortree/engine_default_goals_test.go`

- [ ] **Step 5.1: Write failing accessor tests**

Create `internal/behaviortree/engine_default_goals_test.go`:

```go
package behaviortree

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGetArchetypeDefaultGoals_ParsedFromYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "defaults_archetype.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
default_goals:
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 40
    params:
      target: 500
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("defaults_archetype", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeDefaultGoals("defaults_archetype")
	if len(got) != 2 {
		t.Fatalf("len=%d, want 2", len(got))
	}
	if got[0].Type != "survival" || got[0].Priority != 80 {
		t.Errorf("got[0]=%+v, want survival/80", got[0])
	}
	if got[1].Type != "wealth-gold" || got[1].Priority != 40 {
		t.Errorf("got[1]=%+v, want wealth-gold/40", got[1])
	}
}

func TestGetArchetypeDefaultGoals_AbsentField_EmptyList(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "no_defaults.yaml")
	yaml := []byte(`tree:
  type: action
  do: set_state
`)
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	e := newEngineForTest()
	if err := e.LoadArchetype("no_defaults", path); err != nil {
		t.Fatalf("LoadArchetype: %v", err)
	}
	got := e.GetArchetypeDefaultGoals("no_defaults")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}

func TestGetArchetypeDefaultGoals_UnknownArchetype_EmptyList(t *testing.T) {
	e := newEngineForTest()
	got := e.GetArchetypeDefaultGoals("never_loaded")
	if len(got) != 0 {
		t.Errorf("len=%d, want 0", len(got))
	}
}
```

In `internal/behaviortree/engine_goal_weights_test.go`, extend `newEngineForTest` to initialize the new map (add as the last line in the struct literal):

```go
archetypeDefaultGoals: map[string][]GoalDefault{},
```

- [ ] **Step 5.2: Run tests to verify they fail**

Run: `go test ./internal/behaviortree/ -run "TestGetArchetypeDefaultGoals_" -v`
Expected: FAIL.

- [ ] **Step 5.3: Extend `TreeDef` + declare `GoalDefault`**

In `internal/behaviortree/types.go`, replace `TreeDef`:

```go
// TreeDef is the top-level YAML structure for archetype + room +
// per-mob behavior trees.
type TreeDef struct {
	Tree         NodeDef            `yaml:"tree"`
	GoalWeights  map[string]float64 `yaml:"goal_weights,omitempty"`  // chunk 4.2
	DefaultGoals []GoalDefault      `yaml:"default_goals,omitempty"` // chunk 4.3
}

// GoalDefault declares one default goal to seed on a fresh mob whose
// template uses this archetype. Consumed by internal/goals/ via the
// SetArchetypeDefaultsLookup callback. Chunk 4.3.
type GoalDefault struct {
	Type     string         `yaml:"type"`
	Priority int            `yaml:"priority"`
	Params   map[string]any `yaml:"params,omitempty"`
}
```

- [ ] **Step 5.4: Update `LoadArchetypeYAMLFromFile`**

In `internal/behaviortree/loader.go`, change the signature + body:

```go
// LoadArchetypeYAMLFromFile reads an archetype YAML file and returns
// the compiled tree Node, the chunk-4.2 goal_weights map, AND any
// chunk-4.3 default_goals list.
func LoadArchetypeYAMLFromFile(path string) (Node, map[string]float64, []GoalDefault, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, nil, err
	}
	var def TreeDef
	if err := yaml.Unmarshal(data, &def); err != nil {
		return nil, nil, nil, fmt.Errorf("parse error: %w", err)
	}
	tree, err := compileNode(def.Tree, "root")
	if err != nil {
		return nil, nil, nil, err
	}
	return tree, def.GoalWeights, def.DefaultGoals, nil
}
```

⚠️ Replace `compileNode` with whatever helper the chunk-4.2 implementation actually used. The only intra-package caller of this function is `Engine.LoadArchetype` (next step) — extend both signatures together.

- [ ] **Step 5.5: Extend `LoadArchetype` + add cache + accessor in `engine.go`**

In `internal/behaviortree/engine.go`, find `Engine` and add the new field next to `archetypeGoalWeights`:

```go
archetypeDefaultGoals map[string][]GoalDefault // chunk 4.3
```

Replace `LoadArchetype`:

```go
func (e *Engine) LoadArchetype(name string, path string) error {
	tree, weights, defaults, err := LoadArchetypeYAMLFromFile(path)
	if err != nil {
		return err
	}
	e.mu.Lock()
	e.archetypes[name] = tree
	if e.archetypeGoalWeights == nil {
		e.archetypeGoalWeights = map[string]map[string]float64{}
	}
	e.archetypeGoalWeights[name] = weights
	if e.archetypeDefaultGoals == nil {
		e.archetypeDefaultGoals = map[string][]GoalDefault{}
	}
	e.archetypeDefaultGoals[name] = defaults
	delete(e.noArchetype, name)
	e.mu.Unlock()
	return nil
}
```

Add the accessor:

```go
// GetArchetypeDefaultGoals returns the cached default_goals list for
// the named archetype, or an empty slice if the archetype is unknown
// or declared no defaults. Returns a shallow copy. Chunk 4.3.
func (e *Engine) GetArchetypeDefaultGoals(name string) []GoalDefault {
	e.mu.RLock()
	defer e.mu.RUnlock()
	raw := e.archetypeDefaultGoals[name]
	if len(raw) == 0 {
		return []GoalDefault{}
	}
	out := make([]GoalDefault, len(raw))
	copy(out, raw)
	return out
}
```

In `internal/behaviortree/test_export.go`, if `LoadArchetypeForTest`'s cleanup deletes from `archetypeGoalWeights`, add the parallel:

```go
delete(e.archetypeDefaultGoals, name)
```

- [ ] **Step 5.6: Run tests to verify they pass**

Run: `go test ./internal/behaviortree/ -run "TestGetArchetypeDefaultGoals_" -v`
Expected: PASS.

Run: `go test ./internal/behaviortree/...`
Expected: PASS.

- [ ] **Step 5.7: Commit**

```bash
git add internal/behaviortree/types.go internal/behaviortree/loader.go internal/behaviortree/engine.go internal/behaviortree/engine_default_goals_test.go internal/behaviortree/test_export.go internal/behaviortree/engine_goal_weights_test.go
git commit -m "feat(behaviortree): parse default_goals from archetype YAML (4.3)" -m "Extends TreeDef with default_goals []GoalDefault. LoadArchetypeYAMLFromFile
now returns (tree, weights, defaults, err). New per-Engine
archetypeDefaultGoals cache + GetArchetypeDefaultGoals(name) accessor
returns a shallow copy of the list.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6 — main.go boot wiring for archetype defaults

Register `goals.SetArchetypeDefaultsLookup`. Mirrors chunk-4.2's `SetWeightsLookup`.

**Files:**
- Modify: `main.go`

- [ ] **Step 6.1: Add the defaults-lookup wiring**

In `main.go`, immediately after the existing `goals.SetWeightsLookup(...)` block (from chunk 4.2), insert:

```go
// Wire the goals → behaviortree archetype-defaults resolver. Mirrors
// SetWeightsLookup (chunk 4.2) — bridges goals → behaviortree without
// an import cycle. Chunk 4.3.
goals.SetArchetypeDefaultsLookup(func(mob *mobs.Mob) []goals.GoalDefault {
	if mob == nil || mob.BehaviorArchetype == "" {
		return nil
	}
	btDefaults := behaviortree.GetEngine().GetArchetypeDefaultGoals(mob.BehaviorArchetype)
	if len(btDefaults) == 0 {
		return nil
	}
	out := make([]goals.GoalDefault, len(btDefaults))
	for i, d := range btDefaults {
		out[i] = goals.GoalDefault{Type: d.Type, Priority: d.Priority, Params: d.Params}
	}
	return out
})
```

The translation loop converts `behaviortree.GoalDefault` → `goals.GoalDefault` (deliberate type duplication to avoid the import cycle, per Task 3).

- [ ] **Step 6.2: Build to confirm no typo / cycle**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 6.3: Commit**

```bash
git add main.go
git commit -m "feat(boot): register goals.SetArchetypeDefaultsLookup adapter (4.3)" -m "Bridges goals → behaviortree via the callback from Task 3. Translates
behaviortree.GoalDefault → goals.GoalDefault per the deliberate type
duplication.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Catalog tasks (7–19): conventions

The next 13 tasks each create one type file under `internal/goals/catalog/`. All follow the same shape — Predicate + (optional) ContextScore + (optional) DedupKey + `init()` that registers the type. Per-type tests live in `<type>_test.go` alongside.

**Shared conventions:**
- Each file is `package catalog`.
- Import path for the goals API: `goals "github.com/GoMudEngine/GoMud/internal/goals"`.
- Each `init()` calls `goals.RegisterGoalType(name, goals.GoalTypeMeta{...})`.
- Test files use the existing 4.1 test pattern. To test in isolation, register the type once per test file via `init()` (it's already loaded once at package init, but a `defer goals.UnregisterGoalType(name)` pattern doesn't exist — tests just verify the registered behavior).
- The first per-type task (Task 7) also creates `internal/goals/catalog/catalog.go` with the package documentation comment.

⚠️ **Note on test isolation:** the goals package's `resetRegistry()` test helper wipes the registry — but it's package-internal to `goals`, not exposed to consumers. The catalog package's tests therefore can NOT use `resetRegistry()`. They rely on the fact that catalog `init()` registrations are idempotent (subsequent registrations replace the first). If a per-type test needs to override behavior temporarily, use a separate test-only type name (e.g. `test-survival-x`) rather than overriding the production registration.

---

## Task 7 — `survival` type + catalog package setup

**Files:**
- Create: `internal/goals/catalog/catalog.go` (package doc + shared helpers — empty for now)
- Create: `internal/goals/catalog/survival.go`
- Create: `internal/goals/catalog/survival_test.go`

- [ ] **Step 7.1: Create the catalog package doc file**

Create `internal/goals/catalog/catalog.go`:

```go
// Package catalog registers the chunk-4.3 goal-type catalog with the
// goals package. Each <type>.go file's init() registers one type via
// goals.RegisterGoalType. Main.go pulls these registrations in via a
// blank import:
//
//	import _ "github.com/GoMudEngine/GoMud/internal/goals/catalog"
//
// Types in this catalog:
//
//	survival, wealth-gold, wealth-item, craft-item,
//	revenge-mob, revenge-faction, protection-mob, protection-faction,
//	befriend, befriend-faction, mastery-skill, mastery-equip,
//	visit-zone
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.3-goal-types-catalog-design.md.
package catalog
```

- [ ] **Step 7.2: Write the failing `survival` tests**

Create `internal/goals/catalog/survival_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSurvival_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("survival"); !ok {
		t.Fatalf("survival not registered")
	}
}

func TestSurvival_Predicate_FullHPNotInCombat_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax = 100
	g := &goals.Goal{Type: "survival", Priority: 80}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate at full HP: got false, want true (recovered = satisfied)")
	}
}

func TestSurvival_Predicate_LowHP_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 20
	mob.Character.HealthMax = 100
	g := &goals.Goal{Type: "survival", Priority: 80}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate at 20%% HP: got true, want false")
	}
}

func TestSurvival_ContextScore_FullHP_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 100
	mob.Character.HealthMax = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	if got != 0 {
		t.Errorf("context score at full HP: got %f, want 0 (filtered)", got)
	}
}

func TestSurvival_ContextScore_MidWound_Linear(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 40
	mob.Character.HealthMax = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	// Between flee (25) and safe (60) thresholds → linear from 1.0 to 3.0.
	// At 40%: roughly (60-40)/(60-25) = 0.57 fraction → 1.0 + 0.57*(3.0-1.0) ≈ 2.14
	if got < 1.5 || got > 2.5 {
		t.Errorf("context score at 40%% HP: got %f, want ~2.14 (1.5-2.5)", got)
	}
}

func TestSurvival_ContextScore_Critical_FivePointZero(t *testing.T) {
	meta, _ := goals.LookupGoalType("survival")
	mob := &mobs.Mob{}
	mob.Character.Health = 5
	mob.Character.HealthMax = 100
	got := meta.ContextScore(&goals.Goal{Type: "survival"}, mob)
	if got != 5.0 {
		t.Errorf("context score at 5%% HP: got %f, want 5.0", got)
	}
}
```

⚠️ This test uses `goals.LookupGoalType(name)` — the existing 4.1 helper is internal (`lookupMeta`). **Add an exported wrapper** in `internal/goals/registry.go` (Task 7 also includes this small engine addition; it's used by every per-type test):

```go
// LookupGoalType returns the registered metadata for a goal type and
// whether it was found. Exported variant of the internal lookupMeta —
// catalog tests + future consumers may need to inspect the registry.
// Chunk 4.3.
func LookupGoalType(name string) (GoalTypeMeta, bool) {
	return lookupMeta(name)
}
```

- [ ] **Step 7.3: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestSurvival_" -v`
Expected: FAIL — `survival` type not registered (and possibly `LookupGoalType` undefined if the wrapper isn't yet added).

- [ ] **Step 7.4: Add `LookupGoalType` wrapper to `internal/goals/registry.go`**

Append (with the comment block from Step 7.2). Build verifies no regression.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 7.5: Create `internal/goals/catalog/survival.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const (
	survivalDefaultSafeThresholdPct = 60
	survivalDefaultFleeThresholdPct = 25
)

func init() {
	goals.RegisterGoalType("survival", goals.GoalTypeMeta{
		Predicate:    survivalPredicate,
		ContextScore: survivalContextScore,
		Params: []goals.ParamSchema{
			{Key: "safe_threshold_pct", Required: false, GoType: "int"},
			{Key: "flee_threshold_pct", Required: false, GoType: "int"},
		},
		// AllowMultiple: false — only one survival goal per mob.
	})
}

// survivalPredicate: satisfied (mob has recovered) when HP is at or above
// safe_threshold_pct AND mob is not in combat.
func survivalPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil || mob.Character.HealthMax <= 0 {
		return false
	}
	safe := paramIntOr(g, "safe_threshold_pct", survivalDefaultSafeThresholdPct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax
	return hpPct >= safe && !mob.Character.Aggro.Hostile()
}

// survivalContextScore:
//   - 0 if HP at/above safe threshold (filtered — predicate fires next tick)
//   - 5.0 if HP at/below flee threshold (critical)
//   - linear interpolation from 1.0 (at safe) to 3.0 (at flee) in between
func survivalContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil || mob.Character.HealthMax <= 0 {
		return 0
	}
	safe := paramIntOr(g, "safe_threshold_pct", survivalDefaultSafeThresholdPct)
	flee := paramIntOr(g, "flee_threshold_pct", survivalDefaultFleeThresholdPct)
	hpPct := (mob.Character.Health * 100) / mob.Character.HealthMax
	if hpPct >= safe {
		return 0
	}
	if hpPct <= flee {
		return 5.0
	}
	// Linear: at safe → 1.0, at flee → 3.0
	fraction := float64(safe-hpPct) / float64(safe-flee)
	return 1.0 + fraction*(3.0-1.0)
}

// paramIntOr reads an int param from g.Params with a fallback default.
// Tolerates the int64-from-YAML case (matches ValidateParams widening).
func paramIntOr(g *goals.Goal, key string, def int) int {
	raw, ok := g.Params[key]
	if !ok {
		return def
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}
```

⚠️ The `mob.Character.Aggro.Hostile()` call assumes the existing aggro API has a method that reports "currently in combat with someone." Verify via codegraph (`codegraph_node Aggro` or `codegraph_search Hostile`). If the actual API differs (e.g., `mob.Character.Aggro.UserId != 0` or similar), adapt the predicate. The intent is "any active combat target = not satisfied."

- [ ] **Step 7.6: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestSurvival_" -v`
Expected: PASS.

- [ ] **Step 7.7: Commit**

```bash
git add internal/goals/registry.go internal/goals/catalog/catalog.go internal/goals/catalog/survival.go internal/goals/catalog/survival_test.go
git commit -m "feat(catalog): survival goal type + catalog package setup (4.3)" -m "Catalog subpackage doc file + first type (survival). Predicate: HP ≥
safe threshold AND not in combat. ContextScore: 0 above safe; linear
1.0→3.0 between safe and flee thresholds; 5.0 below flee. Also adds
goals.LookupGoalType exported wrapper for per-type tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8 — `wealth-gold` type

**Files:**
- Create: `internal/goals/catalog/wealth_gold.go`
- Create: `internal/goals/catalog/wealth_gold_test.go`

- [ ] **Step 8.1: Write the failing tests**

Create `internal/goals/catalog/wealth_gold_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthGold_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("wealth-gold"); !ok {
		t.Fatalf("wealth-gold not registered")
	}
}

func TestWealthGold_Predicate_AtTarget_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 500
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate at target: got false, want true")
	}
}

func TestWealthGold_Predicate_BelowTarget_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 200
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate below target: got true, want false")
	}
}

func TestWealthGold_ContextScore_Satisfied_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 500
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score satisfied: got %f, want 0", got)
	}
}

func TestWealthGold_ContextScore_HalfWay_Scaled(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 250
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	got := meta.ContextScore(g, mob)
	// Baseline 1.0 + (target-gold)/target = 0.5 added → 1.5
	if got < 1.4 || got > 1.6 {
		t.Errorf("score at 50%% target: got %f, want ~1.5", got)
	}
}

func TestWealthGold_ContextScore_Empty_MaxTwo(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-gold")
	mob := &mobs.Mob{}
	mob.Character.Gold = 0
	g := &goals.Goal{Type: "wealth-gold", Params: map[string]any{"target": 500}}
	got := meta.ContextScore(g, mob)
	// Baseline 1.0 + (target-gold)/target = 1.0 added, capped at 2.0
	if got != 2.0 {
		t.Errorf("score empty: got %f, want 2.0", got)
	}
}
```

- [ ] **Step 8.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestWealthGold_" -v`
Expected: FAIL — type not registered.

- [ ] **Step 8.3: Create `internal/goals/catalog/wealth_gold.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("wealth-gold", goals.GoalTypeMeta{
		Predicate:    wealthGoldPredicate,
		ContextScore: wealthGoldContextScore,
		Params: []goals.ParamSchema{
			{Key: "target", Required: true, GoType: "int"},
		},
		// AllowMultiple: false — one gold target per mob.
	})
}

func wealthGoldPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	target := paramIntOr(g, "target", 0)
	return mob.Character.Gold >= target
}

// wealthGoldContextScore: 0 if satisfied; otherwise baseline 1.0 plus
// (target - gold) / target, capped at 2.0.
func wealthGoldContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	target := paramIntOr(g, "target", 0)
	if target <= 0 || mob.Character.Gold >= target {
		return 0
	}
	gap := float64(target-mob.Character.Gold) / float64(target)
	score := 1.0 + gap
	if score > 2.0 {
		return 2.0
	}
	return score
}
```

- [ ] **Step 8.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestWealthGold_" -v`
Expected: PASS.

- [ ] **Step 8.5: Commit**

```bash
git add internal/goals/catalog/wealth_gold.go internal/goals/catalog/wealth_gold_test.go
git commit -m "feat(catalog): wealth-gold goal type (4.3)" -m "Predicate: Gold ≥ target. ContextScore: 0 if satisfied; baseline 1.0
+ (target-gold)/target, capped at 2.0. AllowMultiple false — one gold
target per mob.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9 — `wealth-item` type

**Files:**
- Create: `internal/goals/catalog/wealth_item.go`
- Create: `internal/goals/catalog/wealth_item_test.go`

- [ ] **Step 9.1: Write the failing tests**

Create `internal/goals/catalog/wealth_item_test.go`:

```go
package catalog

import (
	"strconv"
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestWealthItem_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("wealth-item"); !ok {
		t.Fatalf("wealth-item not registered")
	}
}

func TestWealthItem_DedupKey_ByItemId(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	g1 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	g2 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 99}}
	if k1, k2 := meta.DedupKey(g1), meta.DedupKey(g2); k1 == k2 {
		t.Errorf("dedup keys collide: %s == %s (different ids)", k1, k2)
	}
}

func TestWealthItem_DedupKey_ByTag(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	g1 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_tag": "iron-ingot"}}
	g2 := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_tag": "iron-ingot"}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same tag")
	}
}

func TestWealthItem_Predicate_ItemAbsent_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	// Empty inventory.
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate with absent item: got true, want false")
	}
}

func TestWealthItem_ContextScore_Present_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	mob.Character.Items = []items.Item{{ItemId: 42}}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score when item present: got %f, want 0", got)
	}
	_ = strconv.Itoa(0) // silence import-unused if test rearranges
}

func TestWealthItem_ContextScore_Absent_OnePointZero(t *testing.T) {
	meta, _ := goals.LookupGoalType("wealth-item")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "wealth-item", Params: map[string]any{"item_id": 42}}
	got := meta.ContextScore(g, mob)
	// 1.0 baseline; the +0.5 shop-in-zone bump requires a real zone scan
	// which we don't have in unit tests, so the baseline is what fires.
	if got != 1.0 {
		t.Errorf("score when item absent (no zone shop scan): got %f, want 1.0", got)
	}
}
```

⚠️ The `items.Item` struct's field name for the item id might be `ItemId` or something else — verify via codegraph (`codegraph_node Item`) and adapt the test fixture. Also verify the location of the backpack list on `*mobs.Mob` (likely `mob.Character.Items` but confirm).

- [ ] **Step 9.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestWealthItem_" -v`
Expected: FAIL.

- [ ] **Step 9.3: Create `internal/goals/catalog/wealth_item.go`**

```go
package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("wealth-item", goals.GoalTypeMeta{
		Predicate:     wealthItemPredicate,
		ContextScore:  wealthItemContextScore,
		AllowMultiple: true,
		DedupKey:      wealthItemDedupKey,
		Params: []goals.ParamSchema{
			{Key: "item_tag", Required: false, GoType: "string"},
			{Key: "item_id", Required: false, GoType: "int"},
		},
	})
}

// wealthItemDedupKey: "tag:<tag>" or "id:<n>". Caller is responsible
// for providing exactly one (Params schema allows either; mob authors
// can violate that — the dedup key just picks tag if present).
func wealthItemDedupKey(g *goals.Goal) string {
	if tag, ok := g.Params["item_tag"].(string); ok && tag != "" {
		return "tag:" + tag
	}
	if id := paramIntOr(g, "item_id", 0); id > 0 {
		return "id:" + strconv.Itoa(id)
	}
	return ""
}

func wealthItemPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	tag, _ := g.Params["item_tag"].(string)
	id := paramIntOr(g, "item_id", 0)
	return mobHasItem(mob, tag, id)
}

// wealthItemContextScore: 0 if present; 1.0 baseline if absent.
// Spec also describes a +0.5 bump if a shop in the mob's zone sells
// the item — implementable via a future shops-in-zone scan; deferred
// here since the engine surface for that scan isn't in 4.3's scope.
// 4.4's planner will add the shop-aware bump.
func wealthItemContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	tag, _ := g.Params["item_tag"].(string)
	id := paramIntOr(g, "item_id", 0)
	if mobHasItem(mob, tag, id) {
		return 0
	}
	return 1.0
}

// mobHasItem checks backpack + equipment for a matching item.
func mobHasItem(mob *mobs.Mob, tag string, id int) bool {
	for _, it := range mob.Character.Items {
		if matchesItem(it, tag, id) {
			return true
		}
	}
	for _, it := range mob.Character.Equipment.GetAllItems() {
		if matchesItem(it, tag, id) {
			return true
		}
	}
	return false
}

func matchesItem(it items.Item, tag string, id int) bool {
	if id > 0 && it.ItemId == id {
		return true
	}
	if tag != "" {
		spec := it.GetSpec()
		if spec.ComponentTag == tag {
			return true
		}
	}
	return false
}
```

⚠️ Verify via codegraph: `items.Item.ItemId`, `items.Item.GetSpec()`, `ItemSpec.ComponentTag`, and `mob.Character.Equipment.GetAllItems()`. Adapt names to actual API.

- [ ] **Step 9.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestWealthItem_" -v`
Expected: PASS.

- [ ] **Step 9.5: Commit**

```bash
git add internal/goals/catalog/wealth_item.go internal/goals/catalog/wealth_item_test.go
git commit -m "feat(catalog): wealth-item goal type (4.3)" -m "Predicate: item with matching tag or id is in backpack or equipped.
ContextScore: 1.0 if absent, 0 if present. AllowMultiple yes; DedupKey
prefers tag, falls back to id. Spec's +0.5 shop-in-zone bump deferred
to 4.4's planner.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10 — `craft-item` type

**Files:**
- Create: `internal/goals/catalog/craft_item.go`
- Create: `internal/goals/catalog/craft_item_test.go`

- [ ] **Step 10.1: Write the failing tests**

Create `internal/goals/catalog/craft_item_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestCraftItem_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("craft-item"); !ok {
		t.Fatalf("craft-item not registered")
	}
}

func TestCraftItem_DedupKey_ByRecipeId(t *testing.T) {
	meta, _ := goals.LookupGoalType("craft-item")
	g1 := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "iron-sword"}}
	g2 := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "steel-sword"}}
	if k1, k2 := meta.DedupKey(g1), meta.DedupKey(g2); k1 == k2 {
		t.Errorf("dedup keys collide: %s == %s", k1, k2)
	}
}

func TestCraftItem_ContextScore_RecipeUnknown_Zero(t *testing.T) {
	meta, _ := goals.LookupGoalType("craft-item")
	mob := &mobs.Mob{}
	// mob.Character.KnownRecipes empty — recipe unknown.
	g := &goals.Goal{Type: "craft-item", Params: map[string]any{"recipe_id": "iron-sword"}}
	if got := meta.ContextScore(g, mob); got != 0 {
		t.Errorf("score with unknown recipe: got %f, want 0 (filtered)", got)
	}
}

// Additional tests (skill-too-low → 0.3; known+skilled+missing materials → 1.0;
// known+skilled+materials on hand → 2.0) require a richer test fixture with
// the recipes/skills/items registry. Defer those to integration testing
// once the catalog package is wired (Task 23 smoke).
```

- [ ] **Step 10.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestCraftItem_" -v`
Expected: FAIL.

- [ ] **Step 10.3: Create `internal/goals/catalog/craft_item.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("craft-item", goals.GoalTypeMeta{
		Predicate:     craftItemPredicate,
		ContextScore:  craftItemContextScore,
		AllowMultiple: true,
		DedupKey:      craftItemDedupKey,
		Params: []goals.ParamSchema{
			{Key: "recipe_id", Required: true, GoType: "string"},
		},
	})
}

func craftItemDedupKey(g *goals.Goal) string {
	if rid, ok := g.Params["recipe_id"].(string); ok {
		return rid
	}
	return ""
}

// craftItemPredicate: satisfied when the item produced by the recipe
// is in mob's inventory or equipment. Resolves the recipe → output item
// via the crafting registry.
func craftItemPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	rid, _ := g.Params["recipe_id"].(string)
	if rid == "" {
		return false
	}
	outputId, ok := craftingRecipeOutputId(rid)
	if !ok {
		return false
	}
	return mobHasItem(mob, "", outputId)
}

// craftItemContextScore tiers:
//   - Recipe unknown to mob → 0 (filtered)
//   - Skill rank below recipe's required minimum → 0.3 (let mastery-skill win)
//   - Known + skilled + materials missing → 1.0
//   - Known + skilled + materials on hand → 2.0
func craftItemContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	rid, _ := g.Params["recipe_id"].(string)
	if rid == "" {
		return 0
	}
	if !mobKnowsRecipe(mob, rid) {
		return 0
	}
	if !mobMeetsRecipeSkill(mob, rid) {
		return 0.3
	}
	if mobHasRecipeMaterials(mob, rid) {
		return 2.0
	}
	return 1.0
}

// ─── Adapters to the existing crafting registry ─────────────────────
// These are thin shims; verify exact crafting API via codegraph and
// adapt. The crafting package likely exposes a Recipes map or a getter.

// craftingRecipeOutputId returns the item id produced by a recipe.
// Returns (0, false) if recipe is unknown to the registry.
func craftingRecipeOutputId(recipeId string) (int, bool) {
	// TODO-ADAPT: call into internal/crafting to look up the recipe's
	// output item id. Example shape:
	//   r, ok := crafting.GetRecipe(recipeId)
	//   if !ok { return 0, false }
	//   return r.OutputItemId, true
	return 0, false
}

func mobKnowsRecipe(mob *mobs.Mob, recipeId string) bool {
	// TODO-ADAPT: walk mob.Character.KnownRecipes (or whatever the actual
	// field is called) for an entry matching recipeId.
	return false
}

func mobMeetsRecipeSkill(mob *mobs.Mob, recipeId string) bool {
	// TODO-ADAPT: read the recipe's required skill + rank from crafting
	// registry, compare to mob.Character.Skills[skillName].Rank.
	return false
}

func mobHasRecipeMaterials(mob *mobs.Mob, recipeId string) bool {
	// TODO-ADAPT: walk recipe.Ingredients[] and check mob's inventory.
	return false
}
```

⚠️ **Critical for the implementer:** the four "TODO-ADAPT" helpers above are placeholders. **Replace them with the actual crafting-package calls before marking this task complete.** Use codegraph to discover the real API: `codegraph_search Recipe`, `codegraph_node KnownRecipes`, `codegraph_search GetRecipe`. The plan can't pre-write the adapters because the crafting package's exact shape isn't verified yet — implementer must discover and wire them. If the API doesn't have a clean lookup, raise it as `DONE_WITH_CONCERNS` and the followup will land in 4.4 or a follow-on chunk.

For the test suite: only the registration + DedupKey + recipe-unknown ContextScore tests are written above because they don't need the adapters. The other branches require a working crafting wiring; defer those tests to integration as noted.

- [ ] **Step 10.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestCraftItem_" -v`
Expected: PASS for the three tests written.

- [ ] **Step 10.5: Commit**

```bash
git add internal/goals/catalog/craft_item.go internal/goals/catalog/craft_item_test.go
git commit -m "feat(catalog): craft-item goal type (4.3)" -m "Predicate: recipe's output item in inventory. ContextScore tiers: 0
if recipe unknown; 0.3 if skill too low; 1.0 if missing materials; 2.0
if materials on hand. AllowMultiple yes; DedupKey by recipe_id.

Crafting-registry adapters are stubbed (TODO-ADAPT) — implementer
wires them to internal/crafting using codegraph to discover the
actual API shape. Test coverage is partial pending those adapters.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11 — `revenge-mob` type

**Files:**
- Create: `internal/goals/catalog/revenge_mob.go`
- Create: `internal/goals/catalog/revenge_mob_test.go`

- [ ] **Step 11.1: Write the failing tests**

Create `internal/goals/catalog/revenge_mob_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestRevengeMob_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("revenge-mob"); !ok {
		t.Fatalf("revenge-mob not registered")
	}
}

func TestRevengeMob_DedupKey_DifferentTargets_Distinct(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 5}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 7}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different targets")
	}
}

func TestRevengeMob_DedupKey_DifferentKinds_Distinct(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "mob", "target_id": 5}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 5}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for mob:5 vs player:5")
	}
}

func TestRevengeMob_DedupKey_SameTarget_Match(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-mob")
	g1 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 3}}
	g2 := &goals.Goal{Type: "revenge-mob", Params: map[string]any{"target_kind": "player", "target_id": 3}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same target")
	}
}

// Predicate + ContextScore behavior depend on live mobs/users + combat-memory
// state. Integration coverage lands at Task 23 smoke. Unit-test only the
// registration + DedupKey shape here.
```

- [ ] **Step 11.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestRevengeMob_" -v`
Expected: FAIL.

- [ ] **Step 11.3: Create `internal/goals/catalog/revenge_mob.go`**

```go
package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("revenge-mob", goals.GoalTypeMeta{
		Predicate:     revengeMobPredicate,
		ContextScore:  revengeMobContextScore,
		AllowMultiple: true,
		DedupKey:      revengeMobDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"}, // "mob" | "player"
			{Key: "target_id", Required: true, GoType: "int"},
		},
	})
}

func revengeMobDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// revengeMobPredicate: satisfied when target is dead.
// "mob" kind: any instance of that template is gone (or recently died).
// "player" kind: user is in death-flag state.
func revengeMobPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return false
	}
	switch kind {
	case "mob":
		// TODO-ADAPT: confirm whether "target dead" means "no instances
		// of templateId loaded" or "recently died per combat log".
		// Cheapest read: scan mobs.GetAllMobInstanceIds + filter by MobId.
		// If zero instances → dead.
		for _, instId := range mobs.GetAllMobInstanceIds() {
			if inst := mobs.GetInstance(instId); inst != nil && int(inst.MobId) == id {
				return false
			}
		}
		return true
	case "player":
		// TODO-ADAPT: check users.GetByUserId(id).Character.Health <= 0
		// (or whatever the death-flag API actually is).
		return false
	}
	return false
}

// revengeMobContextScore:
//   - 0 if target not seen by this mob's CombatMemory in last 1000 rounds
//   - 2.0 if target currently in same room
//   - 1.5 if target in adjacent room
//   - 0.5 if target elsewhere in zone
//   - 0.1 if target out of zone
func revengeMobContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	// TODO-ADAPT: read mob.CombatMemory or a similar "last seen" tracker
	// to confirm the target has been seen recently. Default conservative:
	// if no memory data is available, fall through to room/zone heuristic.
	if !targetRecentlySeen(mob, kind, id) {
		return 0
	}
	return targetProximityScore(mob, kind, id)
}

// ─── Shared helpers used by revenge / protection / befriend types ───
// (extract to a catalog/helpers.go file if duplication grows.)

func targetRecentlySeen(mob *mobs.Mob, kind string, id int) bool {
	// TODO-ADAPT: inspect mob.CombatMemory (4.x or earlier substrate).
	// Cheapest stub: always true (lets the proximity score drive).
	// Implementer: replace with real lookup before commit.
	return true
}

func targetProximityScore(mob *mobs.Mob, kind string, id int) float64 {
	// TODO-ADAPT: resolve target's current room, compare to mob's room.
	// Step-down by hops: same room → 2.0, adjacent → 1.5, same zone → 0.5,
	// out-of-zone → 0.1.
	// Stub: return 0.5 (mid-range) until the room lookup is wired.
	return 0.5
}
```

⚠️ **Critical for implementer:** the three TODO-ADAPT helpers (player death lookup, recently-seen check, proximity score) must be wired before commit. Use codegraph: `codegraph_node CombatMemory`, `codegraph_node GetByUserId`, `codegraph_search room` for the adjacency logic. If `targetProximityScore` is shared with `protection-mob` and `befriend` (Tasks 13 + 15), extract into `internal/goals/catalog/helpers.go` rather than duplicating.

- [ ] **Step 11.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestRevengeMob_" -v`
Expected: PASS.

- [ ] **Step 11.5: Commit**

```bash
git add internal/goals/catalog/revenge_mob.go internal/goals/catalog/revenge_mob_test.go
git commit -m "feat(catalog): revenge-mob goal type (4.3)" -m "Predicate: target dead. ContextScore tiers by proximity (same room
2.0, adjacent 1.5, zone 0.5, out-of-zone 0.1); 0 if target not seen
recently per CombatMemory. AllowMultiple yes; DedupKey
target_kind:target_id.

TODO-ADAPT helpers stubbed for combat-memory lookup, player-death
check, and zone-proximity heuristic — implementer wires before commit.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12 — `revenge-faction` type

**Files:**
- Create: `internal/goals/catalog/revenge_faction.go`
- Create: `internal/goals/catalog/revenge_faction_test.go`

- [ ] **Step 12.1: Write the failing tests**

Create `internal/goals/catalog/revenge_faction_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestRevengeFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("revenge-faction"); !ok {
		t.Fatalf("revenge-faction not registered")
	}
}

func TestRevengeFaction_DedupKey_ByFactionId(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-faction")
	g1 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 5}}
	g2 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "guards", "target_kill_count": 5}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different faction_ids")
	}
}

func TestRevengeFaction_DedupKey_SameFactionDifferentCount_Collides(t *testing.T) {
	meta, _ := goals.LookupGoalType("revenge-faction")
	g1 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 5}}
	g2 := &goals.Goal{Type: "revenge-faction", Params: map[string]any{"faction_id": "bandits", "target_kill_count": 10}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same faction (count is not part of key)")
	}
}
```

- [ ] **Step 12.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestRevengeFaction_" -v`
Expected: FAIL.

- [ ] **Step 12.3: Create `internal/goals/catalog/revenge_faction.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const factionKillsMiscPrefix = "faction_kills_inflicted:"

func init() {
	goals.RegisterGoalType("revenge-faction", goals.GoalTypeMeta{
		Predicate:     revengeFactionPredicate,
		ContextScore:  revengeFactionContextScore,
		AllowMultiple: true,
		DedupKey:      revengeFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
			{Key: "target_kill_count", Required: true, GoType: "int"},
		},
	})
}

func revengeFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// revengeFactionPredicate: per-mob counter (MiscData) reaches target.
// Counter is incremented by a 4.5 reactive hook on kill events; 4.3
// only defines the read path.
func revengeFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	fid, _ := g.Params["faction_id"].(string)
	target := paramIntOr(g, "target_kill_count", 0)
	if fid == "" || target == 0 {
		return false
	}
	current := mobMiscInt(mob, factionKillsMiscPrefix+fid)
	return current >= target
}

// revengeFactionContextScore: 0 if no faction members in zone; 1.0 +
// 0.1 × member_count otherwise, capped at 2.0.
func revengeFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	count := factionMembersInZone(mob, fid)
	if count == 0 {
		return 0
	}
	score := 1.0 + 0.1*float64(count)
	if score > 2.0 {
		return 2.0
	}
	return score
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────

// mobMiscInt reads an integer value from mob.MiscData (or wherever
// per-mob arbitrary KV state lives in 4.x). Returns 0 if absent.
func mobMiscInt(mob *mobs.Mob, key string) int {
	// TODO-ADAPT: implement against the real MiscData API.
	// e.g., if mob.MiscData is map[string]any: return mob.MiscData[key].(int)
	return 0
}

// factionMembersInZone counts live mob instances in the same zone
// as `mob` that belong to faction_id.
func factionMembersInZone(mob *mobs.Mob, factionId string) int {
	// TODO-ADAPT: walk mobs.GetAllMobInstanceIds, filter by zone +
	// faction membership via factions.IsMember (or similar).
	return 0
}
```

⚠️ Implementer: wire `mobMiscInt` (probably reads from a `mob.Character.MiscData map[string]any` or similar) and `factionMembersInZone` (factions package + mob/zone walk). Use codegraph to find the actual APIs. The `factions.IsMember(mobId, factionId)` signature may differ; verify.

- [ ] **Step 12.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestRevengeFaction_" -v`
Expected: PASS.

- [ ] **Step 12.5: Commit**

```bash
git add internal/goals/catalog/revenge_faction.go internal/goals/catalog/revenge_faction_test.go
git commit -m "feat(catalog): revenge-faction goal type (4.3)" -m "Predicate: per-mob MiscData counter ≥ target_kill_count. ContextScore:
0 if no faction members in zone, else 1.0+0.1*count capped at 2.0.
AllowMultiple yes; DedupKey by faction_id only (count is not part of
identity — multiple counts against the same faction collapse).

Counter-incrementing hook ships in 4.5.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13 — `protection-mob` type

**Files:**
- Create: `internal/goals/catalog/protection_mob.go`
- Create: `internal/goals/catalog/protection_mob_test.go`

- [ ] **Step 13.1: Write the failing tests**

Create `internal/goals/catalog/protection_mob_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionMob_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("protection-mob"); !ok {
		t.Fatalf("protection-mob not registered")
	}
}

func TestProtectionMob_DedupKey_PerTarget(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-mob")
	g1 := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 100}}
	g2 := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 200}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup keys collide for different targets")
	}
}

func TestProtectionMob_Predicate_NeverSatisfied(t *testing.T) {
	// Protection is ongoing — never satisfied. 4.6 will remove the goal
	// via expiry/dead-target logic.
	meta, _ := goals.LookupGoalType("protection-mob")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "protection-mob", Params: map[string]any{"target_kind": "mob", "target_id": 100}}
	if meta.Predicate(g, mob) {
		t.Errorf("protection-mob predicate should never satisfy at 4.3 (got true)")
	}
}
```

- [ ] **Step 13.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestProtectionMob_" -v`
Expected: FAIL.

- [ ] **Step 13.3: Create `internal/goals/catalog/protection_mob.go`**

```go
package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("protection-mob", goals.GoalTypeMeta{
		Predicate:     protectionMobPredicate,
		ContextScore:  protectionMobContextScore,
		AllowMultiple: true,
		DedupKey:      protectionMobDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"},
			{Key: "target_id", Required: true, GoType: "int"},
		},
	})
}

func protectionMobDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// protectionMobPredicate: never satisfied (ongoing). 4.6's pruning
// sweep removes the goal when the target has been dead for ≥ N rounds.
func protectionMobPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	return false
}

// protectionMobContextScore:
//   - 0 if target dead
//   - 2.5 if target currently in combat
//   - 1.5 if target in same room (not in combat)
//   - 0.8 if target in same zone
//   - 0.2 if target alive in different zone
func protectionMobContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	if !targetAlive(kind, id) {
		return 0
	}
	if targetInCombat(kind, id) {
		return 2.5
	}
	return targetProximityScoreProtection(mob, kind, id)
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────

func targetAlive(kind string, id int) bool {
	// TODO-ADAPT: lookup based on kind ("mob" → mobs.GetInstance(id);
	// "player" → users.GetByUserId(id) + Character.Health > 0).
	return false
}

func targetInCombat(kind string, id int) bool {
	// TODO-ADAPT: lookup target, check target.Character.Aggro state.
	return false
}

func targetProximityScoreProtection(mob *mobs.Mob, kind string, id int) float64 {
	// TODO-ADAPT: same-room → 1.5, same-zone → 0.8, other-zone → 0.2.
	// Likely shares the resolution path with targetProximityScore in
	// revenge_mob.go; extract a shared helper when duplication shows up.
	return 0.8
}
```

⚠️ Implementer: wire `targetAlive`, `targetInCombat`, `targetProximityScoreProtection`. The protection scoring is identical-shape to revenge-mob's but with different magnitudes — strong candidate for extracting `targetRoomProximityHops(mob, targetRoomId) int` into `helpers.go` and applying per-type magnitude tables.

- [ ] **Step 13.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestProtectionMob_" -v`
Expected: PASS.

- [ ] **Step 13.5: Commit**

```bash
git add internal/goals/catalog/protection_mob.go internal/goals/catalog/protection_mob_test.go
git commit -m "feat(catalog): protection-mob goal type (4.3)" -m "Predicate never satisfies (4.6 prunes on dead target). ContextScore:
2.5 if target in combat, 1.5 same room, 0.8 same zone, 0.2 elsewhere,
0 dead. AllowMultiple yes; DedupKey target_kind:target_id.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14 — `protection-faction` type

**Files:**
- Create: `internal/goals/catalog/protection_faction.go`
- Create: `internal/goals/catalog/protection_faction_test.go`

- [ ] **Step 14.1: Write the failing tests**

Create `internal/goals/catalog/protection_faction_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestProtectionFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("protection-faction"); !ok {
		t.Fatalf("protection-faction not registered")
	}
}

func TestProtectionFaction_DedupKey_ByFactionId(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-faction")
	g1 := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	g2 := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.DedupKey(g1) != meta.DedupKey(g2) {
		t.Errorf("dedup keys differ for same faction_id")
	}
}

func TestProtectionFaction_Predicate_NeverSatisfied(t *testing.T) {
	meta, _ := goals.LookupGoalType("protection-faction")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "protection-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.Predicate(g, mob) {
		t.Errorf("protection-faction predicate should never satisfy")
	}
}
```

- [ ] **Step 14.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestProtectionFaction_" -v`
Expected: FAIL.

- [ ] **Step 14.3: Create `internal/goals/catalog/protection_faction.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("protection-faction", goals.GoalTypeMeta{
		Predicate:     protectionFactionPredicate,
		ContextScore:  protectionFactionContextScore,
		AllowMultiple: true,
		DedupKey:      protectionFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
		},
	})
}

func protectionFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// protectionFactionPredicate: never satisfied (ongoing).
func protectionFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	return false
}

// protectionFactionContextScore:
//   - 0 if no faction members in zone
//   - 2.0 if any faction member is in combat in zone
//   - 1.0 if hostile mobs in zone but no member-in-combat
//   - 0.3 if zone calm (members present but quiet)
func protectionFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	memberCount := factionMembersInZone(mob, fid)
	if memberCount == 0 {
		return 0
	}
	if factionMemberInCombatInZone(mob, fid) {
		return 2.0
	}
	if hostileMobsInZone(mob) {
		return 1.0
	}
	return 0.3
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────
// (factionMembersInZone already declared in revenge_faction.go)

func factionMemberInCombatInZone(mob *mobs.Mob, factionId string) bool {
	// TODO-ADAPT: walk faction members in zone, check Character.Aggro state.
	return false
}

func hostileMobsInZone(mob *mobs.Mob) bool {
	// TODO-ADAPT: walk mob instances in zone, find any with AutoAggro=true
	// (or whatever the canonical "hostile" check is post-bcompat).
	return false
}
```

⚠️ Implementer: `factionMembersInZone` is already declared in `revenge_faction.go` — DO NOT redeclare it in this file (Go compile error). Reuse the existing helper. If extracting to `helpers.go`, do so as part of this task and delete the local copy in `revenge_faction.go`.

- [ ] **Step 14.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestProtectionFaction_" -v`
Expected: PASS.

- [ ] **Step 14.5: Commit**

```bash
git add internal/goals/catalog/protection_faction.go internal/goals/catalog/protection_faction_test.go
git commit -m "feat(catalog): protection-faction goal type (4.3)" -m "Predicate never satisfies. ContextScore: 2.0 member-in-combat in zone,
1.0 hostiles present without member-combat, 0.3 calm with members
present, 0 no members in zone. AllowMultiple yes; DedupKey faction_id.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15 — `befriend` type

**Files:**
- Create: `internal/goals/catalog/befriend.go`
- Create: `internal/goals/catalog/befriend_test.go`

- [ ] **Step 15.1: Write failing tests**

Create `internal/goals/catalog/befriend_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestBefriend_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("befriend"); !ok {
		t.Fatalf("befriend not registered")
	}
}

func TestBefriend_DedupKey_PerTarget(t *testing.T) {
	meta, _ := goals.LookupGoalType("befriend")
	g1 := &goals.Goal{Type: "befriend", Params: map[string]any{"target_kind": "player", "target_id": 5}}
	g2 := &goals.Goal{Type: "befriend", Params: map[string]any{"target_kind": "player", "target_id": 6}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different targets")
	}
}
```

- [ ] **Step 15.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestBefriend_" -v`
Expected: FAIL.

- [ ] **Step 15.3: Create `internal/goals/catalog/befriend.go`**

```go
package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const befriendDefaultThreshold = 60

func init() {
	goals.RegisterGoalType("befriend", goals.GoalTypeMeta{
		Predicate:     befriendPredicate,
		ContextScore:  befriendContextScore,
		AllowMultiple: true,
		DedupKey:      befriendDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_kind", Required: true, GoType: "string"},
			{Key: "target_id", Required: true, GoType: "int"},
			{Key: "opinion_threshold", Required: false, GoType: "int"},
		},
	})
}

func befriendDedupKey(g *goals.Goal) string {
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return ""
	}
	return kind + ":" + strconv.Itoa(id)
}

// befriendPredicate: satisfied when opinion of target ≥ threshold.
func befriendPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	thr := paramIntOr(g, "opinion_threshold", befriendDefaultThreshold)
	if kind == "" || id == 0 {
		return false
	}
	return mobOpinionOf(mob, kind, id) >= thr
}

// befriendContextScore: 0 if target not in same zone; 1.5 same room;
// 0.8 same zone (different room).
func befriendContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	kind, _ := g.Params["target_kind"].(string)
	id := paramIntOr(g, "target_id", 0)
	if kind == "" || id == 0 {
		return 0
	}
	switch targetProximityHops(mob, kind, id) {
	case 0:
		return 1.5 // same room
	case 1, 2, 3:
		return 0.8 // same zone, near
	}
	return 0
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────

func mobOpinionOf(mob *mobs.Mob, kind string, id int) int {
	// TODO-ADAPT: call opinions.Of(int(mob.MobId), kind, id) (or whatever
	// the chunk 1.1 API signature actually is).
	return 0
}

func targetProximityHops(mob *mobs.Mob, kind string, id int) int {
	// TODO-ADAPT: 0 = same room, 1-N = adjacent / nearby, -1 = out of zone.
	// Shared candidate with revenge-mob's targetProximityScore — extract
	// to helpers.go.
	return -1
}
```

⚠️ Implementer: `mobOpinionOf` and `targetProximityHops` need real wiring against the opinions package + room/zone graph. `targetProximityHops` is third use of the same proximity-resolution idea (after revenge-mob and protection-mob) — extract a `helpers.go` with `targetRoomId(kind, id) int` + `hopsBetweenRooms(a, b int) int` now to avoid 4+ copies.

- [ ] **Step 15.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestBefriend_" -v`
Expected: PASS.

- [ ] **Step 15.5: Commit**

```bash
git add internal/goals/catalog/befriend.go internal/goals/catalog/befriend_test.go
git commit -m "feat(catalog): befriend goal type (4.3)" -m "Predicate: opinion ≥ threshold. ContextScore: 1.5 same room, 0.8 same
zone, 0 elsewhere. AllowMultiple yes; DedupKey target_kind:target_id.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 16 — `befriend-faction` type

**Files:**
- Create: `internal/goals/catalog/befriend_faction.go`
- Create: `internal/goals/catalog/befriend_faction_test.go`

- [ ] **Step 16.1: Write failing tests**

Create `internal/goals/catalog/befriend_faction_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestBefriendFaction_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("befriend-faction"); !ok {
		t.Fatalf("befriend-faction not registered")
	}
}

func TestBefriendFaction_DedupKey_ByFaction(t *testing.T) {
	meta, _ := goals.LookupGoalType("befriend-faction")
	g1 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "merchants"}}
	g2 := &goals.Goal{Type: "befriend-faction", Params: map[string]any{"faction_id": "watch"}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different factions")
	}
}
```

- [ ] **Step 16.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestBefriendFaction_" -v`
Expected: FAIL.

- [ ] **Step 16.3: Create `internal/goals/catalog/befriend_faction.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const befriendFactionDefaultThreshold = 60

func init() {
	goals.RegisterGoalType("befriend-faction", goals.GoalTypeMeta{
		Predicate:     befriendFactionPredicate,
		ContextScore:  befriendFactionContextScore,
		AllowMultiple: true,
		DedupKey:      befriendFactionDedupKey,
		Params: []goals.ParamSchema{
			{Key: "faction_id", Required: true, GoType: "string"},
			{Key: "rep_threshold", Required: false, GoType: "int"},
		},
	})
}

func befriendFactionDedupKey(g *goals.Goal) string {
	if fid, ok := g.Params["faction_id"].(string); ok {
		return fid
	}
	return ""
}

// befriendFactionPredicate: satisfied when rep with faction ≥ threshold.
func befriendFactionPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	fid, _ := g.Params["faction_id"].(string)
	thr := paramIntOr(g, "rep_threshold", befriendFactionDefaultThreshold)
	if fid == "" {
		return false
	}
	return factionRepOf(mob, fid) >= thr
}

// befriendFactionContextScore: 0 if no faction members in zone; otherwise
// 1.0 + 0.1 × (threshold - current) / threshold, capped at 1.8.
func befriendFactionContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	fid, _ := g.Params["faction_id"].(string)
	if fid == "" {
		return 0
	}
	if factionMembersInZone(mob, fid) == 0 {
		return 0
	}
	thr := paramIntOr(g, "rep_threshold", befriendFactionDefaultThreshold)
	current := factionRepOf(mob, fid)
	if thr <= 0 || current >= thr {
		return 1.0
	}
	gap := float64(thr-current) / float64(thr)
	score := 1.0 + 0.1*gap
	if score > 1.8 {
		return 1.8
	}
	return score
}

// ─── TODO-ADAPT helper ──────────────────────────────────────────────

func factionRepOf(mob *mobs.Mob, factionId string) int {
	// TODO-ADAPT: call factions.GetRep(int(mob.MobId), factionId) — verify
	// signature via codegraph.
	return 0
}
```

⚠️ Implementer: `factionRepOf` wraps the factions package. Verify signature with codegraph (`codegraph_search GetRep`).

- [ ] **Step 16.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestBefriendFaction_" -v`
Expected: PASS.

- [ ] **Step 16.5: Commit**

```bash
git add internal/goals/catalog/befriend_faction.go internal/goals/catalog/befriend_faction_test.go
git commit -m "feat(catalog): befriend-faction goal type (4.3)" -m "Predicate: rep ≥ threshold. ContextScore: 0 if no faction members in
zone; baseline 1.0 + small bump scaling with rep gap, capped at 1.8.
AllowMultiple yes; DedupKey faction_id.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 17 — `mastery-skill` type

**Files:**
- Create: `internal/goals/catalog/mastery_skill.go`
- Create: `internal/goals/catalog/mastery_skill_test.go`

- [ ] **Step 17.1: Write failing tests**

Create `internal/goals/catalog/mastery_skill_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestMasterySkill_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("mastery-skill"); !ok {
		t.Fatalf("mastery-skill not registered")
	}
}

func TestMasterySkill_DedupKey_BySkillName(t *testing.T) {
	meta, _ := goals.LookupGoalType("mastery-skill")
	g1 := &goals.Goal{Type: "mastery-skill", Params: map[string]any{"skill_name": "weapon-combat", "target_rank": 30}}
	g2 := &goals.Goal{Type: "mastery-skill", Params: map[string]any{"skill_name": "spellcasting", "target_rank": 30}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different skills")
	}
}
```

- [ ] **Step 17.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestMasterySkill_" -v`
Expected: FAIL.

- [ ] **Step 17.3: Create `internal/goals/catalog/mastery_skill.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("mastery-skill", goals.GoalTypeMeta{
		Predicate:     masterySkillPredicate,
		ContextScore:  masterySkillContextScore,
		AllowMultiple: true,
		DedupKey:      masterySkillDedupKey,
		Params: []goals.ParamSchema{
			{Key: "skill_name", Required: true, GoType: "string"},
			{Key: "target_rank", Required: true, GoType: "int"},
		},
	})
}

func masterySkillDedupKey(g *goals.Goal) string {
	if name, ok := g.Params["skill_name"].(string); ok {
		return name
	}
	return ""
}

// masterySkillPredicate: satisfied when current rank ≥ target.
func masterySkillPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	name, _ := g.Params["skill_name"].(string)
	target := paramIntOr(g, "target_rank", 0)
	if name == "" || target == 0 {
		return false
	}
	return mobSkillRank(mob, name) >= target
}

// masterySkillContextScore tiers:
//   - 0 if already at/above target (predicate fires next tick)
//   - 0.2 baseline if no training opportunity in zone
//   - 1.0 if opportunity in zone
//   - 2.0 if opportunity in current room
//   - Multiplied by (1 - rank/target) * 0.5 + 0.5 so closer-to-target is less urgent
func masterySkillContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	name, _ := g.Params["skill_name"].(string)
	target := paramIntOr(g, "target_rank", 0)
	if name == "" || target == 0 {
		return 0
	}
	current := mobSkillRank(mob, name)
	if current >= target {
		return 0
	}
	var base float64
	switch skillTrainingProximity(mob, name) {
	case 0: // in current room
		base = 2.0
	case 1: // in zone
		base = 1.0
	default:
		base = 0.2
	}
	urgency := (1.0-float64(current)/float64(target))*0.5 + 0.5
	return base * urgency
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────

func mobSkillRank(mob *mobs.Mob, skillName string) int {
	// TODO-ADAPT: read mob.Character.Skills[skillName].Rank (verify field).
	return 0
}

func skillTrainingProximity(mob *mobs.Mob, skillName string) int {
	// TODO-ADAPT: per-skill table mapping skill → training context kind
	// (combat for weapon skills, crafting stations for crafting skills, etc.).
	// Return 0 = in current room, 1 = in zone, 2+ = elsewhere.
	return 2
}
```

⚠️ Implementer: `mobSkillRank` reads the existing skill registry; verify field path. `skillTrainingProximity` needs a per-skill heuristic table — start with a simple "all skills train via combat for now" and refine when 4.4 wires actual planning.

- [ ] **Step 17.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestMasterySkill_" -v`
Expected: PASS.

- [ ] **Step 17.5: Commit**

```bash
git add internal/goals/catalog/mastery_skill.go internal/goals/catalog/mastery_skill_test.go
git commit -m "feat(catalog): mastery-skill goal type (4.3)" -m "Predicate: rank ≥ target. ContextScore tiered by training-opportunity
proximity (current room 2.0, zone 1.0, elsewhere 0.2), scaled by urgency
(less urgent as rank approaches target). AllowMultiple yes; DedupKey
skill_name.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 18 — `mastery-equip` type

**Files:**
- Create: `internal/goals/catalog/mastery_equip.go`
- Create: `internal/goals/catalog/mastery_equip_test.go`

- [ ] **Step 18.1: Write failing tests**

Create `internal/goals/catalog/mastery_equip_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestMasteryEquip_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("mastery-equip"); !ok {
		t.Fatalf("mastery-equip not registered")
	}
}

func TestMasteryEquip_DedupKey_BySlot(t *testing.T) {
	meta, _ := goals.LookupGoalType("mastery-equip")
	g1 := &goals.Goal{Type: "mastery-equip", Params: map[string]any{"slot": "weapon", "min_rarity_tier": 60}}
	g2 := &goals.Goal{Type: "mastery-equip", Params: map[string]any{"slot": "head", "min_rarity_tier": 60}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different slots")
	}
}
```

- [ ] **Step 18.2: Run tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestMasteryEquip_" -v`
Expected: FAIL.

- [ ] **Step 18.3: Create `internal/goals/catalog/mastery_equip.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const rarityFallbackTier = 50 // engine fallback per project memory

func init() {
	goals.RegisterGoalType("mastery-equip", goals.GoalTypeMeta{
		Predicate:     masteryEquipPredicate,
		ContextScore:  masteryEquipContextScore,
		AllowMultiple: true,
		DedupKey:      masteryEquipDedupKey,
		Params: []goals.ParamSchema{
			{Key: "slot", Required: true, GoType: "string"},
			{Key: "min_rarity_tier", Required: true, GoType: "int"},
		},
	})
}

func masteryEquipDedupKey(g *goals.Goal) string {
	if s, ok := g.Params["slot"].(string); ok {
		return s
	}
	return ""
}

// masteryEquipPredicate: satisfied when equipped item in slot has
// rarity_tier ≥ target. Untagged items use the engine fallback (50).
func masteryEquipPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	slot, _ := g.Params["slot"].(string)
	target := paramIntOr(g, "min_rarity_tier", 0)
	if slot == "" || target == 0 {
		return false
	}
	tier := mobSlotRarityTier(mob, slot)
	return tier >= target
}

// masteryEquipContextScore tiers:
//   - 0 if predicate satisfied
//   - 1.5 if slot empty (very motivated)
//   - 1.5 if shop selling slot items is in current room
//   - 1.0 if shop in zone
//   - 0.5 reachable via patrol/wander
//   - 0.3 no shop in zone (planner can wander)
func masteryEquipContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	slot, _ := g.Params["slot"].(string)
	target := paramIntOr(g, "min_rarity_tier", 0)
	if slot == "" || target == 0 {
		return 0
	}
	tier := mobSlotRarityTier(mob, slot)
	if tier >= target {
		return 0
	}
	if mobSlotIsEmpty(mob, slot) {
		return 1.5
	}
	switch shopForSlotProximity(mob, slot) {
	case 0: // in current room
		return 1.5
	case 1: // in zone
		return 1.0
	}
	return 0.3
}

// ─── TODO-ADAPT helpers ─────────────────────────────────────────────

// mobSlotRarityTier returns the rarity tier of the item in mob's slot,
// or rarityFallbackTier if the item lacks the rarity_tier field.
// Returns 0 if the slot is empty.
func mobSlotRarityTier(mob *mobs.Mob, slot string) int {
	// TODO-ADAPT: walk mob.Character.Equipment for the named slot;
	// resolve its rarity_tier via items.GetSpec(item.ItemId).RarityTier;
	// fall back to rarityFallbackTier if missing.
	return 0
}

func mobSlotIsEmpty(mob *mobs.Mob, slot string) bool {
	// TODO-ADAPT: inspect mob.Character.Equipment for the slot.
	return true
}

func shopForSlotProximity(mob *mobs.Mob, slot string) int {
	// TODO-ADAPT: scan shops in mob's zone, return 0/1/2 by distance.
	return 2
}
```

⚠️ Implementer: three TODO-ADAPT helpers. The `mobSlotRarityTier` is the most subtle — verify rarity-tier field is on `ItemSpec` (the project memory references it at `ItemSpec.RarityTier`) and apply the tier-50 fallback per the existing `474afd98` engine logic.

- [ ] **Step 18.4: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestMasteryEquip_" -v`
Expected: PASS.

- [ ] **Step 18.5: Commit**

```bash
git add internal/goals/catalog/mastery_equip.go internal/goals/catalog/mastery_equip_test.go
git commit -m "feat(catalog): mastery-equip goal type (4.3)" -m "Predicate: slot item rarity_tier ≥ target (untagged items use tier-50
fallback per 474afd98). ContextScore: 1.5 if slot empty or shop in
room, 1.0 if shop in zone, 0.3 otherwise. AllowMultiple yes; DedupKey
slot.

Tuning patchy until project_rarity_tier_audit lands (145/213 items
lack tags). Recommend authors avoid this type in untagged-heavy zones.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 19 — `visit-zone` type

**Files:**
- Create: `internal/goals/catalog/visit_zone.go`
- Create: `internal/goals/catalog/visit_zone_test.go`
- Modify: `internal/mobs/mobs.go` (add `VisitedZones` field on `*mobs.Mob`)
- Create or modify: `internal/hooks/<some_room_change_hook>.go` (record visits on room change)

⚠️ This task adds engine state (a `VisitedZones` tracker on Mob instances) and a hook that records visits — more than a pure catalog type. If the room-change hook plumbing turns out to be larger than expected, split into 19a (catalog type + field) and 19b (hook integration) at implementer discretion.

- [ ] **Step 19.1: Write failing catalog tests**

Create `internal/goals/catalog/visit_zone_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestVisitZone_Registered(t *testing.T) {
	if _, ok := goals.LookupGoalType("visit-zone"); !ok {
		t.Fatalf("visit-zone not registered")
	}
}

func TestVisitZone_DedupKey_ByTargetZone(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	g1 := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	g2 := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "thornwall_city"}}
	if meta.DedupKey(g1) == meta.DedupKey(g2) {
		t.Errorf("dedup collide for different zones")
	}
}

func TestVisitZone_Predicate_Visited_True(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	mob := &mobs.Mob{}
	mob.VisitedZones = map[string]bool{"stillwater": true}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	if !meta.Predicate(g, mob) {
		t.Errorf("predicate when visited: got false, want true")
	}
}

func TestVisitZone_Predicate_Unvisited_False(t *testing.T) {
	meta, _ := goals.LookupGoalType("visit-zone")
	mob := &mobs.Mob{}
	g := &goals.Goal{Type: "visit-zone", Params: map[string]any{"target_zone": "stillwater"}}
	if meta.Predicate(g, mob) {
		t.Errorf("predicate when unvisited: got true, want false")
	}
}
```

- [ ] **Step 19.2: Add `VisitedZones` field to `mobs.Mob`**

In `internal/mobs/mobs.go`, find the `Mob` struct definition and add:

```go
// VisitedZones tracks zone names this instance has entered. Persisted
// via mobs.instances/ alongside other instance state. Read by the
// chunk-4.3 visit-zone goal-type Predicate. Updated by the room-change
// hook in internal/hooks/. Lazily initialized — nil counts as empty.
// Chunk 4.3.
VisitedZones map[string]bool `yaml:"visited_zones,omitempty"`
```

Place it near the other instance-state fields (PackFleeImmune, LastIdleCommand, etc.) so authors notice it.

- [ ] **Step 19.3: Run catalog tests to verify they fail**

Run: `go test ./internal/goals/catalog/ -run "TestVisitZone_" -v`
Expected: FAIL — `visit-zone` not registered.

- [ ] **Step 19.4: Create `internal/goals/catalog/visit_zone.go`**

```go
package catalog

import (
	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("visit-zone", goals.GoalTypeMeta{
		Predicate:     visitZonePredicate,
		ContextScore:  visitZoneContextScore,
		AllowMultiple: true,
		DedupKey:      visitZoneDedupKey,
		Params: []goals.ParamSchema{
			{Key: "target_zone", Required: true, GoType: "string"},
		},
	})
}

func visitZoneDedupKey(g *goals.Goal) string {
	if z, ok := g.Params["target_zone"].(string); ok {
		return z
	}
	return ""
}

// visitZonePredicate: satisfied when mob has visited target_zone.
func visitZonePredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	zone, _ := g.Params["target_zone"].(string)
	if zone == "" {
		return false
	}
	if mob.VisitedZones == nil {
		return false
	}
	return mob.VisitedZones[zone]
}

// visitZoneContextScore:
//   - 0 if mob currently in target_zone (predicate fires next tick on hook)
//   - 0 if no path known from current zone (defensive)
//   - 1.5 if adjacent (1 hop)
//   - 0.8 if 2-3 hops
//   - 0.3 if 4+ hops
func visitZoneContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	zone, _ := g.Params["target_zone"].(string)
	if zone == "" {
		return 0
	}
	if mob.Character.Zone == zone {
		return 0
	}
	hops := zoneGraphDistance(mob.Character.Zone, zone)
	switch {
	case hops < 0:
		return 0
	case hops == 1:
		return 1.5
	case hops <= 3:
		return 0.8
	}
	return 0.3
}

// ─── TODO-ADAPT helper ──────────────────────────────────────────────

// zoneGraphDistance returns hop count between two zones, or -1 if no
// path is known. Uses the existing zone adjacency graph (or computes
// from room-exit adjacency lazily).
func zoneGraphDistance(from, to string) int {
	// TODO-ADAPT: rooms package likely exposes zone adjacency via
	// GetZoneConfig or similar. Cheapest stub: hop count = 99 (treated
	// as "far"). Implementer should wire a BFS over zone-adjacency
	// (computed from inter-zone room exits) and cache the result.
	return 99
}
```

- [ ] **Step 19.5: Add room-change visit tracker hook**

The room-change hook should record `mob.VisitedZones[newZone] = true` whenever a mob enters a room in a zone it hasn't visited before.

Find the existing room-change hook for mobs (look for `MoveToRoom` callers in `internal/hooks/` and `internal/mobs/`). The exact insertion point depends on current plumbing — verify via codegraph.

Pattern to insert (probably in `mobs.SetRoom` or whatever the canonical room-update method is):

```go
// Chunk 4.3: visit-zone goal tracker. Record zone visits on room change.
if newZone := /* ...resolve new zone from new room... */; newZone != "" {
	if mob.VisitedZones == nil {
		mob.VisitedZones = map[string]bool{}
	}
	mob.VisitedZones[newZone] = true
}
```

⚠️ Implementer: this hook plumbing requires careful placement. Locate via `codegraph_search MoveToRoom` and confirm where mob room transitions actually fire. If no clean hook exists, extract one.

- [ ] **Step 19.6: Run tests to verify they pass**

Run: `go test ./internal/goals/catalog/ -run "TestVisitZone_" -v`
Expected: PASS.

Run: `go test ./internal/goals/... ./internal/mobs/...`
Expected: PASS.

- [ ] **Step 19.7: Commit**

```bash
git add internal/goals/catalog/visit_zone.go internal/goals/catalog/visit_zone_test.go internal/mobs/mobs.go internal/hooks/
git commit -m "feat(catalog): visit-zone goal type + VisitedZones tracker (4.3)" -m "Predicate: mob.VisitedZones[target_zone] true. ContextScore: 0 if
already in target zone, 1.5 if adjacent, 0.8 if 2-3 hops, 0.3 if 4+.
AllowMultiple yes; DedupKey target_zone.

New mob.Mob.VisitedZones map[string]bool field persisted via
mobs.instances/. Room-change hook records visits on mob room
transitions. Wanderlust effect emerges from authoring multiple
visit-zone goals on a 'wanderer' archetype.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 20 — Catalog blank-import in main.go

Fires all the per-type `init()` registrations. Without this import, the catalog package's source compiles fine but nothing in it ever runs.

**Files:**
- Modify: `main.go`

- [ ] **Step 20.1: Add the blank import**

In `main.go`, add to the imports block:

```go
import (
	// ...existing imports...
	_ "github.com/GoMudEngine/GoMud/internal/goals/catalog" // chunk 4.3 — fire type registrations
)
```

The underscore alias signals "import for side-effects only" (the init() funcs).

- [ ] **Step 20.2: Build to confirm**

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 20.3: Sanity check — confirm types are registered at runtime**

Add a quick smoke test in `internal/goals/catalog/catalog_test.go`:

```go
package catalog

import (
	"testing"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
)

func TestAllCatalogTypes_Registered(t *testing.T) {
	want := []string{
		"survival",
		"wealth-gold", "wealth-item", "craft-item",
		"revenge-mob", "revenge-faction",
		"protection-mob", "protection-faction",
		"befriend", "befriend-faction",
		"mastery-skill", "mastery-equip",
		"visit-zone",
	}
	for _, name := range want {
		if _, ok := goals.LookupGoalType(name); !ok {
			t.Errorf("type %q not registered", name)
		}
	}
}
```

Run: `go test ./internal/goals/catalog/ -run "TestAllCatalogTypes_" -v`
Expected: PASS — all 13 registered.

- [ ] **Step 20.4: Commit**

```bash
git add main.go internal/goals/catalog/catalog_test.go
git commit -m "feat(boot): blank-import goals/catalog to fire type registrations (4.3)" -m "Adds the blank import that pulls all 13 type init() funcs. Without
this import, the catalog package compiles fine but never registers
anything at runtime. Sanity test confirms all 13 types are registered
after import.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 21 — Archetype YAML edits (16 archetypes)

Add the `default_goals:` block per Section 5.1 of the spec.

**Files:**
- Modify: 16 archetype YAMLs in `_datafiles/world/dogmud/behaviors/archetypes/`

**Defaults table** (from spec §5.1):

| File | Defaults |
|------|----------|
| `ambusher.yaml` | `survival` |
| `combat_passive.yaml` | `survival` |
| `defensive_caster.yaml` | `survival` |
| `forager.yaml` | `survival` |
| `generic_fighter.yaml` | `survival` |
| `leader.yaml` | `survival` |
| `lookout.yaml` | `survival` |
| `melee_self_buff.yaml` | `survival` |
| `predator.yaml` | `survival` |
| `prey.yaml` | `survival` |
| `pure_caster.yaml` | `survival` |
| `scout.yaml` | `survival` |
| `support_caster.yaml` | `survival` |
| `tank_taunter.yaml` | `survival` |
| `thief.yaml` | `survival` + `wealth-gold(target=500, priority=40)` |
| `noncombat_shopkeeper.yaml` | `survival` + `wealth-gold(target=1000, priority=30)` |

**Untouched:** `noncombat_passive.yaml`, `noncombat_questgiver.yaml`, all 5 boss YAMLs.

- [ ] **Step 21.1: Add `survival` to the 14 single-default archetypes**

For each of the 14 files listed above with just `survival`, append the following block at the top level (after `tree:` and `goal_weights:` if either is present — order doesn't matter semantically; pick whichever reads cleanest):

```yaml
default_goals:
  - type: survival
    priority: 80
```

- [ ] **Step 21.2: Add `survival` + `wealth-gold` to `thief.yaml`**

```yaml
default_goals:
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 40
    params:
      target: 500
```

- [ ] **Step 21.3: Add `survival` + `wealth-gold` to `noncombat_shopkeeper.yaml`**

```yaml
default_goals:
  - type: survival
    priority: 80
  - type: wealth-gold
    priority: 30
    params:
      target: 1000
```

- [ ] **Step 21.4: Boot to confirm archetypes parse cleanly**

Run: `go build ./... && timeout 60 go run . 2>&1 | grep -iE "panic|error|loadedcount" | head -30`
Expected: no panics; `mobs.LoadDataFiles()` reports a normal `loadedCount`; archetype loads (look for log lines from `behaviortree`) succeed.

Stop the server with Ctrl+C once boot completes.

- [ ] **Step 21.5: Commit**

```bash
git add _datafiles/world/dogmud/behaviors/archetypes/
git commit -m "content(archetypes): add 4.3 default_goals blocks (16 archetypes)" -m "Per spec §5.1: survival universal for all combat-capable archetypes
(14), plus generic wealth-gold for thief (target=500) and shopkeeper
(target=1000). Quest-givers, passive townspeople, and bosses get no
defaults (per spec — defer to per-mob YAML when that ships).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 22 — context.md authoring (goals + catalog)

Per spec §10.7 task 22, close the documentation gap from 4.1/4.2 and create the new catalog package's context.md.

**Files:**
- Create: `internal/goals/context.md`
- Create: `internal/goals/catalog/context.md`

- [ ] **Step 22.1: Author `internal/goals/context.md`**

Follow the structure of `internal/relationships/context.md` and `internal/knowledge/context.md`. Cover:

- Package purpose: the goal substrate + selection engine (4.1 + 4.2).
- Key types: `Goal`, `MobGoals`, `GoalTypeMeta`, `ParamSchema`, `GoalDefault`, `SelectReason`.
- Core APIs: `Add`, `Remove`, `Clear`, `GoalsOf`, `CurrentGoalOf`, `Recompute`, `Select`, `RegisterGoalType`, `LookupGoalType`, `SetWeightsLookup`, `SetArchetypeDefaultsLookup`.
- Persistence: per-template YAML in `_datafiles/world/dogmud/goals/`, atomic write, sentinel field.
- Concurrency: `cacheMu sync.RWMutex` covers cache; `Select` is lock-free.
- Lazy seeding mechanism: first-access on a fresh mob → archetype defaults → flip sentinel.
- Subpackage: `catalog/` registers concrete goal types.
- Out-of-scope: btree integration (4.4), reactive seeding (4.5), pruning (4.6).

Aim for ~80–150 lines of markdown. Match the voice of existing context.md files (factual, dense, no marketing language).

- [ ] **Step 22.2: Author `internal/goals/catalog/context.md`**

Cover:

- Package purpose: registers the 13-type catalog with the goals substrate.
- File layout: one `<type>.go` per type, plus this doc.
- Registration pattern: each file's `init()` calls `goals.RegisterGoalType`.
- How to add a new type: file naming, schema declaration, Predicate/ContextScore conventions, DedupKey for AllowMultiple types, test file alongside.
- The 13 types as a brief table (name + one-line purpose).
- Adapter pattern: TODO-ADAPT helpers wrap subsystems (factions, opinions, mobs zone scans). When subsystem APIs change, the catalog file is the local-impact point.
- Out-of-scope: planner (4.4), reactive seeding hooks (4.5), satisfaction sweep (4.6).

Aim for ~60–120 lines.

- [ ] **Step 22.3: Commit**

```bash
git add internal/goals/context.md internal/goals/catalog/context.md
git commit -m "docs: context.md for goals package + catalog subpackage (4.3)" -m "Closes the documentation gap from 4.1/4.2 (the goals package never
got a context.md) and authors the catalog subpackage's overview per
project convention (matches relationships/, knowledge/, conversations/).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 23 — Smoke checklist + roadmap + patch notes

Pre-push SOP run, mark the chunk done, draft the patch-notes entry.

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `PATCH_NOTES.md`
- Verify: `_datafiles/config.yaml` (`Logging.LogToFile: false`)

- [ ] **Step 23.1: Wipe instance saves before smoke**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 23.2: Boot the server and confirm clean startup**

Run: `timeout 90 go run . 2>&1 | grep -iE "panic|loadedcount|started|fatal" | head -40`

Watch for:
- `mobs.LoadDataFiles() loadedCount=...` succeeds.
- Archetype YAMLs with the new `default_goals:` blocks parse cleanly (no warnings about unknown types).
- `MainWorker state="Started"` appears.
- No panics.

Stop the server with Ctrl+C.

- [ ] **Step 23.3: Live admin-command smoke test**

Re-boot the server, connect as admin, exercise the lazy-seed path on a known mob:

```
goal current 371      → should show a survival goal (seeded from forager archetype's defaults)
goal scores 371       → should show survival in the table
goal clear 371        → wipe goals
goal current 371      → should report "none — (0 goals on file)"
                         (NOT re-seeded because sentinel preserved)
```

Tail the server log — confirm a `goals.switch` debug line fires when a damaged mob's `survival` goal is selected.

- [ ] **Step 23.4: Confirm disk file shape**

Inspect a seeded mob's file (e.g., `_datafiles/world/dogmud/goals/371-tova.yaml`). Confirm `seeded_from_archetype: true` is present and the seeded goal(s) round-tripped correctly.

- [ ] **Step 23.5: Update `MOB_ALIVENESS_ROADMAP.md`**

Find the 4.3 row in the chunks table. Flip status `Not started` → `Done`. Find the rollup line and update `24 / 42 done` → `25 / 42 done • 0 in progress • 17 not started`. Find the 4.3 detail block (lines starting `### 4.3 Goal types catalog`) and add a `**Shipped:**` paragraph similar to the 4.2 detail block, listing the 13 types + engine deltas + archetype defaults + spec/plan paths.

- [ ] **Step 23.6: Append `PATCH_NOTES.md` entry**

Add a dated entry at the top of the file (above the 4.2 entry):

```
## 2026-05-27 — Mob aliveness chunk 4.3: goal types catalog

13 concrete goal types now register with the strategic-layer
substrate: survival, wealth-gold, wealth-item, craft-item,
revenge-mob, revenge-faction, protection-mob, protection-faction,
befriend, befriend-faction, mastery-skill, mastery-equip,
visit-zone. Each has a Predicate (when satisfied), ContextScore
(relevance multiplier), and — where multi-instance makes sense
— an AllowMultiple flag plus DedupKey func so the same mob can
hold goals against multiple targets without collapsing.

Engine deltas: declarative ParamSchema validation at Add time
(rejects malformed goals); AllowMultiple + DedupKey for
multi-instance types; archetype lazy-seed sentinel so default
goals seed once per mob template on first access and survive
admin Clear.

Sparse archetype defaults ship: every combat-capable archetype
defaults to a survival goal (kicks in when HP drops to ~25%);
thieves and shopkeepers add a generic wealth-gold goal.
Mob-specific param goals (revenge targets, befriend targets)
arrive via 4.5 reactive event hooks.

Substrate-only — chosen goals aren't wired into behavior-tree
execution yet (chunk 4.4). Observable change: `goal current
<mob>` now returns a real current goal for most loaded mobs;
the `goals.switch` debug log fires when survival kicks in during
combat. No player-facing change.

Note: the existing MobIdle gossip system (`buildGossipLine` in
NewRound_HandleIdleMobs.go) is intentionally untouched.
```

- [ ] **Step 23.7: Verify `Logging.LogToFile: false`**

Open `_datafiles/config.yaml`. Confirm `LogToFile: false` under `Logging:`. Set it if not (prod droplet disk-space SOP).

- [ ] **Step 23.8: Full test suite + final build**

Run: `go test ./...`
Expected: PASS across the board.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 23.9: Commit roadmap + patch notes**

```bash
git add MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md _datafiles/config.yaml
git commit -m "chore(roadmap): mark aliveness 4.3 goal types catalog Done (25/42)" -m "- Roadmap: 4.3 status -> Done, rollup 24/42 -> 25/42.
- PATCH_NOTES: chunk 4.3 entry.
- Config: confirm LogToFile=false per pre-push SOP.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:** every section maps to tasks:
- §1 architecture → Tasks 7-19 (catalog subpackage + per-type files).
- §2 API surface → Tasks 1, 2, 3, 4, 5 (engine deltas) + Task 6 (boot wiring).
- §3 engine deltas → Tasks 1 (ParamSchema), 2 (AllowMultiple/DedupKey), 3+4 (lazy seed + sentinel).
- §4 the 13-type catalog → Tasks 7-19 (one task per type).
- §5 archetype defaults → Task 21.
- §6 persistence delta → Task 4 (sentinel field on MobGoals).
- §7 cross-type conflict policy → No-op task (explicit non-decision; documented in spec).
- §8 note on dropped gossip → documentation only, no task needed (covered in spec body).
- §9 edge cases → covered across Predicate/ContextScore tests in Tasks 7-19; engine-edge tests in Tasks 1, 2, 4.
- §10 testing → distributed across all per-type tasks + Task 23 smoke.

**Placeholder scan:** the per-type tasks (especially 10-19) intentionally contain TODO-ADAPT helpers because the exact subsystem APIs (crafting registry, combat memory, opinions, factions accessor signatures) need to be verified via codegraph by the implementer at execution time. Each TODO-ADAPT block has a clear directive (which package to look up, which symbol to call, fallback if unclear) — these are NOT "fill-in-the-blank" placeholders, they are documented adaptation points.

**Type consistency:** signatures across tasks match (`Predicate(g *Goal, mob *mobs.Mob) bool`, `ContextScoreFn(g *Goal, mob *mobs.Mob) float64`, `DedupKey func(g *Goal) string`). `GoalDefault` shape consistent across Task 3 (goals side) and Task 5 (behaviortree side). `ParamSchema` consistent across Task 1 (definition) and Tasks 7-19 (usage).

**Scope:** 23 tasks, single feature branch, comparable to chunk 4.2 (11 tasks for L). The catalog tasks (7-19) are the bulk and each is small (~5 steps).

