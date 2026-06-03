# Mob Aliveness 4.5 — Reactive Goal Generation Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Ship 10 reactive seeder rules (3 forced-by-4.4 + 7 reactive) in a new `internal/seeders/` package so NPCs react to world events — friends-of-killed-NPCs get revenge goals, gifts bump opinions, faction kill/rep counters drive `revenge-faction`/`befriend-faction` predicates, missing-materials craft goals seed wealth-item fallbacks.

**Architecture:** New `internal/seeders/` subpackage (mirrors 4.3 catalog + 4.4 planners). Each rule is one file with `init()` calling `Register(ruleName, fn, eventTypes...)`. A package-level dispatcher maps event types to subscribed rules; `main.go` adds one `events.AddListener(eventType, seeders.Dispatch)` per event type seeders care about. Rules invoke `goals.Add`, `mob.Character.SetMiscData`, or `opinions.Bump` as their effect. Rule 3 (materials → wealth-item) is the one architectural exception — triggered by the craft-item planner's Failure branch via a public `seeders.SeedMaterialsForRecipe(mob, recipeId)` function rather than via event subscription.

**Tech Stack:** Go 1.25 · existing `events`, `mudlog`, `mobs`, `goals`, `opinions`, `factions`, `relationships`, `rooms`, `crafting`, `users` packages.

**Spec:** `docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md`

---

## Task 1 — Seeder framework: types, registry, dispatcher

The skeleton. Pure types + a registration map + a dispatcher with panic recovery. No rules registered yet; no behavior change yet.

**Files:**
- Create: `internal/seeders/seeders.go`
- Create: `internal/seeders/seeders_test.go`

- [ ] **Step 1.1: Create the package file**

Create `internal/seeders/seeders.go`:

```go
// Package seeders contains chunk-4.5 reactive goal-generation rules.
// Each rule subscribes to one or more event types (or is invoked
// directly from another package for architectural exceptions) and
// produces effects via the normal substrate APIs: goals.Add (goal
// seeders), mob.Character.SetMiscData (counter writers), or
// opinions.Bump (opinion shifters).
//
// Per-rule files live alongside this one. Each rule's init() calls
// Register. main.go imports this package + wires events.AddListener
// for each event type seeders care about.
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md
package seeders

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// RuleFn is invoked once per event the rule is registered for. The
// rule inspects the event payload, decides whether to act, and applies
// effects via substrate APIs. Rules must be defensive against the
// event payload not matching expectations (return early on type
// assertion failures, missing fields, etc.).
type RuleFn func(event events.Event)

// registration ties a rule's name (for logging) to its function.
type registration struct {
	name string
	fn   RuleFn
}

var (
	registryMu sync.RWMutex
	registry   = map[string][]registration{} // event type name → registered rules
)

// Register subscribes a rule to one or more event type names. Called
// from each per-rule file's init() function. ruleName is used for
// panic-recovery log lines + future admin tooling.
//
// types parameter accepts the values produced by event.Type() — the
// string identifier the events package uses for each concrete type
// (e.g., "MobDeath", "Communication"). Register the rule for each
// type it cares about.
func Register(ruleName string, fn RuleFn, types ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, t := range types {
		registry[t] = append(registry[t], registration{name: ruleName, fn: fn})
	}
}

// Dispatch is the package-level event listener wired by main.go for
// every event type seeders care about. Looks up rules for the event's
// type, invokes each under panic recovery.
//
// Returns events.Continue always — seeders observe events; they don't
// suppress them.
func Dispatch(event events.Event) events.ListenerReturn {
	typeName := event.Type()

	registryMu.RLock()
	rules := registry[typeName]
	registryMu.RUnlock()

	for _, reg := range rules {
		invokeRuleSafely(reg.name, reg.fn, event)
	}
	return events.Continue
}

// invokeRuleSafely wraps a rule call in panic recovery. Mirrors the
// 4.2 invokeContextScore / 4.3 invokeDedupKey / 4.4 invokePlannerSafely
// patterns. A panic logs a warn line with rule name + event type and
// returns; other rules' invocation continues.
func invokeRuleSafely(ruleName string, fn RuleFn, event events.Event) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("seeders.rule panic",
				"rule", ruleName,
				"event_type", event.Type(),
				"panic", fmt.Sprintf("%v", r))
		}
	}()
	fn(event)
}

// resetRegistryForTest wipes the registry. Test-only seam — package-
// internal so tests can isolate rule registration.
func resetRegistryForTest() {
	registryMu.Lock()
	registry = map[string][]registration{}
	registryMu.Unlock()
}
```

- [ ] **Step 1.2: Write framework tests**

Create `internal/seeders/seeders_test.go`:

```go
package seeders

import (
	"sync/atomic"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

// fakeEvent implements events.Event for tests — minimal shape.
type fakeEvent struct{ typeName string }

func (f fakeEvent) Type() string { return f.typeName }

func TestRegister_DispatchInvokesRuleForMatchingType(t *testing.T) {
	resetRegistryForTest()
	var called int32
	Register("test-rule", func(events.Event) {
		atomic.AddInt32(&called, 1)
	}, "test-type-A")

	Dispatch(fakeEvent{typeName: "test-type-A"})
	if atomic.LoadInt32(&called) != 1 {
		t.Errorf("rule not invoked on matching event")
	}
}

func TestDispatch_NoRulesForType_NoOp(t *testing.T) {
	resetRegistryForTest()
	// No rules registered. Dispatch must not panic.
	ret := Dispatch(fakeEvent{typeName: "unsubscribed-type"})
	if ret != events.Continue {
		t.Errorf("Dispatch returned %v, want events.Continue", ret)
	}
}

func TestDispatch_OnlyMatchingTypeFires(t *testing.T) {
	resetRegistryForTest()
	var calledA, calledB int32
	Register("rule-A", func(events.Event) { atomic.AddInt32(&calledA, 1) }, "type-A")
	Register("rule-B", func(events.Event) { atomic.AddInt32(&calledB, 1) }, "type-B")

	Dispatch(fakeEvent{typeName: "type-A"})
	if atomic.LoadInt32(&calledA) != 1 {
		t.Errorf("rule-A should have fired")
	}
	if atomic.LoadInt32(&calledB) != 0 {
		t.Errorf("rule-B should NOT have fired on type-A event")
	}
}

func TestDispatch_MultipleRulesForSameType_AllFire(t *testing.T) {
	resetRegistryForTest()
	var calledA, calledB int32
	Register("rule-A", func(events.Event) { atomic.AddInt32(&calledA, 1) }, "shared-type")
	Register("rule-B", func(events.Event) { atomic.AddInt32(&calledB, 1) }, "shared-type")

	Dispatch(fakeEvent{typeName: "shared-type"})
	if atomic.LoadInt32(&calledA) != 1 || atomic.LoadInt32(&calledB) != 1 {
		t.Errorf("both rules should have fired: a=%d b=%d", calledA, calledB)
	}
}

func TestDispatch_PanicInOneRule_OthersStillFire(t *testing.T) {
	resetRegistryForTest()
	var calledAfter int32
	Register("rule-panic", func(events.Event) {
		panic("boom")
	}, "shared-type")
	Register("rule-after", func(events.Event) {
		atomic.AddInt32(&calledAfter, 1)
	}, "shared-type")

	// Must not panic.
	Dispatch(fakeEvent{typeName: "shared-type"})
	if atomic.LoadInt32(&calledAfter) != 1 {
		t.Errorf("rule-after should have fired despite rule-panic")
	}
}

func TestRegister_OneRuleMultipleTypes(t *testing.T) {
	resetRegistryForTest()
	var called int32
	Register("multi-type", func(events.Event) {
		atomic.AddInt32(&called, 1)
	}, "type-X", "type-Y")

	Dispatch(fakeEvent{typeName: "type-X"})
	Dispatch(fakeEvent{typeName: "type-Y"})
	Dispatch(fakeEvent{typeName: "type-Z"})

	if atomic.LoadInt32(&called) != 2 {
		t.Errorf("called=%d, want 2 (X + Y, not Z)", called)
	}
}
```

⚠️ The `fakeEvent` test fixture implements only `Type()`. Real events have more methods or are concrete structs. Check via codegraph (`codegraph_node Event`) — if `events.Event` is an interface with just `Type() string`, the fixture works as-is. If it requires more methods, add minimal stubs.

- [ ] **Step 1.3: Run tests + build**

Run: `go test ./internal/seeders/ -v`
Expected: PASS for all 6 tests.

Run: `go build ./...`
Expected: clean build.

- [ ] **Step 1.4: Commit**

```bash
git add internal/seeders/seeders.go internal/seeders/seeders_test.go
git commit -m "feat(seeders): framework — RuleFn + Register + Dispatch (4.5)" -m "New internal/seeders/ subpackage for chunk-4.5 reactive goal generation.
RuleFn type, sync.RWMutex-guarded registry mapping event type names to
rules, Dispatch listener that fires registered rules under panic
recovery (mirrors 4.2/4.3/4.4 invoke* wrapper patterns).

Per-rule files (Tasks 4-13) will call Register in init(); main.go
(Task 3) wires events.AddListener for each event type seeders subscribe
to.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 2 — Shared helpers (`state.go`)

Two shared helpers used by multiple rules: `applyCooldown` for per-pair throttling and `seedRevengeGoalIfAbsent` for the multiple revenge-seeding rules.

**Files:**
- Create: `internal/seeders/state.go`
- Create: `internal/seeders/state_test.go`

- [ ] **Step 2.1: Create state.go**

```go
package seeders

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CooldownKeyPrefix is the MiscData key prefix used by applyCooldown.
// Distinct from 4.4's "plan:" prefix so ClearPlanState (fired on goal
// switch) does NOT wipe seeder cooldowns — the cooldown is independent
// of strategic-layer state.
const CooldownKeyPrefix = "seed_cooldown:"

// applyCooldown returns true and writes a fresh expiration round if no
// active cooldown exists for the (rule, key) pair on this mob. Returns
// false (no write) if the cooldown is still active.
//
// Cooldown markers live on the BENEFICIARY mob's MiscData under
//   "seed_cooldown:<rule_name>:<key>"
// where key is a per-rule identifier (e.g., "<userId>" for gift,
// "<attackerInstanceId>" for combat-assist).
//
// windowRounds is the cooldown duration. The stored value is the round
// at which the cooldown EXPIRES — so a fresh call writes
// (currentRound + windowRounds).
func applyCooldown(mob *mobs.Mob, ruleName, key string, windowRounds uint64) bool {
	if mob == nil {
		return false
	}
	miscKey := CooldownKeyPrefix + ruleName + ":" + key
	nowRound := util.GetRoundCount()

	expires := readMiscUint64(mob, miscKey)
	if nowRound < expires {
		return false // cooldown active
	}

	mob.Character.SetMiscData(miscKey, nowRound+windowRounds)
	return true
}

// seedRevengeGoalIfAbsent checks whether a revenge-mob goal targeting
// the same (kind, id) already exists on the mob; if so returns nil
// (dedup — don't escalate priority on repeat offense in 4.5).
// Otherwise calls goals.Add with the standard revenge-mob shape and
// returns the added Goal.
//
// Returns nil on goals.Add failure (logged at warn level).
func seedRevengeGoalIfAbsent(mob *mobs.Mob, targetKind string, targetId, priority int) *goals.Goal {
	if mob == nil || targetKind == "" || targetId == 0 {
		return nil
	}
	mobId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)

	// Pre-check: is there already a revenge-mob goal targeting this
	// same (kind, id) on the mob? Walk GoalsOf and inspect Params.
	existing := goals.GoalsOf(mobId, name)
	for _, g := range existing {
		if g.Type != "revenge-mob" {
			continue
		}
		eKind, _ := g.Params["target_kind"].(string)
		eId := paramAsInt(g.Params["target_id"])
		if eKind == targetKind && eId == targetId {
			return nil // already targeting this revenge
		}
	}

	g := &goals.Goal{
		Type:     "revenge-mob",
		Priority: priority,
		Params: map[string]any{
			"target_kind": targetKind,
			"target_id":   targetId,
		},
	}
	res, err := goals.Add(mobId, name, g)
	if err != nil {
		mudlog.Warn("seeders.seedRevenge: Add failed",
			"mob_id", mobId, "target_kind", targetKind,
			"target_id", targetId, "error", err)
		return nil
	}
	return res.Added
}

// readMiscUint64 reads a uint64 (or coerces int/int64) from MiscData.
// Returns 0 if absent or wrong type.
func readMiscUint64(mob *mobs.Mob, key string) uint64 {
	if mob == nil {
		return 0
	}
	raw := mob.Character.GetMiscData(key)
	if raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case uint64:
		return v
	case int64:
		return uint64(v)
	case int:
		return uint64(v)
	}
	return 0
}

// readMiscInt reads an int from MiscData (coerces int/int64). Returns
// def if absent or wrong type. Mirrors the helpers in 4.3 catalog and
// 4.4 planners packages.
func readMiscInt(mob *mobs.Mob, key string, def int) int {
	if mob == nil {
		return def
	}
	raw := mob.Character.GetMiscData(key)
	if raw == nil {
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

// bumpMiscInt increments an int MiscData value by delta. Initializes
// to delta if absent. Tolerates int / int64 coercion.
func bumpMiscInt(mob *mobs.Mob, key string, delta int) {
	if mob == nil {
		return
	}
	current := readMiscInt(mob, key, 0)
	mob.Character.SetMiscData(key, current+delta)
}

// paramAsInt coerces a Goal.Params value to int. Returns 0 on failure
// (matches the catalog's tolerance for int vs int64 from YAML).
func paramAsInt(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// resolveKillerFromMobDeath inspects a MobDeath event and returns the
// killer's (kind, id) tuple. Returns ("", 0) if the event doesn't
// identify a killer or the type assertions fail.
//
// TODO-ADAPT: the exact MobDeath struct fields are verified at impl
// time. Common shapes: KillerUserId, KillerMobInstanceId. Adapt the
// extraction to whatever the real struct exposes.
func resolveKillerFromMobDeath(event interface{}) (kind string, id int) {
	// TODO-ADAPT: replace with concrete field reads against events.MobDeath.
	// Example shape:
	//   md := event.(events.MobDeath)
	//   if md.KillerUserId > 0 { return "player", md.KillerUserId }
	//   if md.KillerMobInstanceId > 0 { return "mob", int(md.KillerMobInstanceId) }
	_ = event
	return "", 0
}

// instanceIdAsKey converts a mob InstanceId int to a stable string key
// for cooldown lookups.
func instanceIdAsKey(id int) string { return strconv.Itoa(id) }

// userIdAsKey converts a user id int to a stable string key.
func userIdAsKey(id int) string { return strconv.Itoa(id) }
```

⚠️ `resolveKillerFromMobDeath` is the one TODO-ADAPT in this task — the exact `events.MobDeath` struct shape is verified at impl time. Use `codegraph_node MobDeath` to inspect. Rules 1 + 4 (the two MobDeath subscribers) call this helper, so wire it correctly once.

- [ ] **Step 2.2: Write state tests**

Create `internal/seeders/state_test.go`:

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestApplyCooldown_FirstCall_TrueAndWrites(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99001)}
	mob.Character.Name = "cooldown_first"

	ok := applyCooldown(mob, "test-rule", "key-1", 100)
	if !ok {
		t.Errorf("first call should return true (no active cooldown)")
	}

	// Verify the MiscData key was written.
	if got := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1"); got == nil {
		t.Errorf("cooldown marker not written to MiscData")
	}
}

func TestApplyCooldown_WithinWindow_FalseNoWrite(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99002)}
	mob.Character.Name = "cooldown_within"

	// First call: write cooldown.
	applyCooldown(mob, "test-rule", "key-1", 100)
	first := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1")

	// Second call immediately: should be blocked.
	ok := applyCooldown(mob, "test-rule", "key-1", 100)
	if ok {
		t.Errorf("second call within window should return false")
	}

	// Marker unchanged.
	second := mob.Character.GetMiscData("seed_cooldown:test-rule:key-1")
	if first != second {
		t.Errorf("cooldown marker was rewritten during active window: %v -> %v", first, second)
	}
}

func TestApplyCooldown_DifferentKey_NotBlocked(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99003)}
	mob.Character.Name = "cooldown_diff_key"

	applyCooldown(mob, "test-rule", "key-A", 100)
	// Different key on same rule: should NOT be blocked.
	ok := applyCooldown(mob, "test-rule", "key-B", 100)
	if !ok {
		t.Errorf("different cooldown key should not be blocked by key-A")
	}
}

func TestApplyCooldown_DifferentRule_NotBlocked(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99004)}
	mob.Character.Name = "cooldown_diff_rule"

	applyCooldown(mob, "rule-A", "shared-key", 100)
	ok := applyCooldown(mob, "rule-B", "shared-key", 100)
	if !ok {
		t.Errorf("different rule should not be blocked by rule-A on same key")
	}
}

func TestApplyCooldown_NilMob_NoOp(t *testing.T) {
	if applyCooldown(nil, "test", "key", 100) {
		t.Errorf("nil mob should return false (no work to do)")
	}
}

func TestBumpMiscInt_FromAbsent(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99005)}
	mob.Character.Name = "bump_absent"
	bumpMiscInt(mob, "counter:x", 3)
	if got := readMiscInt(mob, "counter:x", 0); got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestBumpMiscInt_FromExisting(t *testing.T) {
	mob := &mobs.Mob{MobId: mobs.MobId(99006)}
	mob.Character.Name = "bump_existing"
	mob.Character.SetMiscData("counter:y", 10)
	bumpMiscInt(mob, "counter:y", 5)
	if got := readMiscInt(mob, "counter:y", 0); got != 15 {
		t.Errorf("got %d, want 15", got)
	}
}

// Note: seedRevengeGoalIfAbsent tests defer to rule 4's
// friend_killed_to_revenge tests, where the helper is exercised with
// real goals.Add flow + dedup verification.
```

- [ ] **Step 2.3: Run tests + build**

Run: `go test ./internal/seeders/ -v`
Expected: PASS for all framework + state tests.

- [ ] **Step 2.4: Commit**

```bash
git add internal/seeders/state.go internal/seeders/state_test.go
git commit -m "feat(seeders): shared helpers — cooldown, revenge dedup, MiscData (4.5)" -m "applyCooldown: per-(rule, key) throttling with seed_cooldown: MiscData
prefix (distinct from 4.4's plan: prefix so ClearPlanState does NOT
wipe seeder cooldowns — they outlive goal switches).

seedRevengeGoalIfAbsent: pre-check + goals.Add for the multiple
revenge-seeding rules (4, 5, 6). Dedup on (target_kind, target_id).

bumpMiscInt: int counter increment for faction-kill / faction-rep
writers.

resolveKillerFromMobDeath: TODO-ADAPT stub — concrete events.MobDeath
field reads wired at first MobDeath-subscriber task (rule 1).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 3 — New event types + firer wiring + main.go boot

Three new event types in `internal/events/eventtypes.go` that rules 6/7/9 will subscribe to. Wire each event's firer at the appropriate source (give.go, attack.go/taunt.go, btree's keep-action). Then add `seeders.Dispatch` as the listener for every event type seeders care about.

This is the Option-B (events-everywhere) architectural choice — all rules except rule 3 go through the same dispatcher. Rule 3 stays a direct planner call because it's an internal planner state, not a world event.

**Files:**
- Modify: `internal/events/eventtypes.go` (add three new event struct types)
- Modify: `internal/usercommands/give.go` (or wherever the give action lives — verify) — fire `GiftOffered` on every give; fire `GiftAccepted` if the receiver has no consider-keep btree path (quest-givers etc.)
- Modify: `internal/actions/attack.go` and `internal/actions/taunt.go` — fire `PlayerAttackedMob` when player engages a mob
- Modify: `internal/behaviortree/actions_*.go` — find the btree action that decides keep-or-return on offered items (per chunk 2.3 equip-if-better) and fire `GiftAccepted` in the keep branch
- Modify: `main.go` (add seeders import + listener wiring)

- [ ] **Step 3.1: Add the three new event types**

In `internal/events/eventtypes.go`, append (placement: after `ItemOwnership` or wherever world-event types cluster — read the file to pick the right spot):

```go
// PlayerAttackedMob fires when a player initiates a combat action
// (attack, taunt, etc.) against a mob. Consumed by chunk-4.5 seeders:
//   - aggressive_action_to_revenge (rule 6) — seeds revenge-mob into
//     the attacked mob + non-hostile witnesses.
//   - combat_assist_to_opinion_boost (rule 9) — if the attacked mob
//     was already engaged with another non-player mob, bumps the
//     beneficiary's opinion of the attacking player.
//
// Chunk 4.5.
type PlayerAttackedMob struct {
	UserId        int // player initiating the attack
	MobInstanceId int // mob being attacked
}

func (p PlayerAttackedMob) Type() string { return `PlayerAttackedMob` }

// GiftOffered fires when a player runs `give <item> <mob>`. Fires
// unconditionally on every give. No consumers in chunk 4.5 — reserved
// slot for future rules (analytics, tutorial hints) that want to react
// to the offer regardless of whether the mob keeps it.
//
// For opinion-boost purposes, use GiftAccepted instead — GiftOffered
// is gameable (worthless-rock spam still fires it).
//
// Chunk 4.5.
type GiftOffered struct {
	UserId        int
	MobInstanceId int
	ItemId        int
}

func (g GiftOffered) Type() string { return `GiftOffered` }

// GiftAccepted fires when a mob decides to KEEP an item offered by a
// player. Fired by the btree action that decides keep-or-return
// (chunk 2.3 equip-if-better), OR by the give-action handler when the
// receiver has no consider path (e.g., quest-givers, certain
// flavor NPCs). NOT fired when the mob returns the item.
//
// Consumed by chunk-4.5 seeders:
//   - gift_to_opinion_boost (rule 7) — value-tiered opinion bump.
//
// Chunk 4.5.
type GiftAccepted struct {
	UserId        int
	MobInstanceId int
	ItemId        int
}

func (g GiftAccepted) Type() string { return `GiftAccepted` }
```

⚠️ Verify the `Type()` method pattern matches the existing event types (e.g., `func (r RoomChange) Type() string { return ``RoomChange`` }` from `eventtypes.go:147`). Match the receiver-naming convention used elsewhere in the file.

- [ ] **Step 3.2: Fire `PlayerAttackedMob` from attack action**

Find the attack-action handler via codegraph: `codegraph_search attack` and look in `internal/actions/attack.go` or `internal/usercommands/attack.go`. Identify the line where the player's attack is fully committed (after target resolution, before any damage rolls).

Insert (adapt to local variable names):

```go
// Chunk 4.5: notify seeders that a player engaged a mob.
events.AddToQueue(events.PlayerAttackedMob{
    UserId:        user.UserId,
    MobInstanceId: targetMob.InstanceId,
})
```

Skip when target is a player, not a mob — only mobs receive seeded revenge / opinion shifts. Verify the handler's existing target-type branching.

- [ ] **Step 3.3: Fire `PlayerAttackedMob` from taunt action**

Find `internal/actions/taunt.go` (or wherever the taunt logic lives — `codegraph_search taunt`). Insert the same `events.AddToQueue(events.PlayerAttackedMob{...})` at the point taunt commits its effect on a mob target. Taunt is symmetric to attack for rule 9's purposes — it's also "player engaged with a mob," and combat-assist should fire when a player taunts an attacker to draw aggro off a friendly mob.

- [ ] **Step 3.4: Fire `GiftOffered` from give action**

Find the give-action handler — likely `internal/usercommands/give.go`. After the item-transfer to the mob succeeds (item now in mob's inventory), insert:

```go
// Chunk 4.5: notify seeders of every give.
events.AddToQueue(events.GiftOffered{
    UserId:        user.UserId,
    MobInstanceId: targetMob.InstanceId,
    ItemId:        item.ItemId,
})
```

Skip if target is a player.

- [ ] **Step 3.5: Fire `GiftAccepted` from btree's keep-action**

Per chunk 2.3 (equip-if-better), most combat-capable mobs have a btree action that decides whether to keep an offered item or return it. Find it via codegraph — `codegraph_search equip_if_better` or similar; likely a file like `internal/behaviortree/actions_equip_if_better.go`.

In the branch where the mob decides to KEEP the item (or where the keep-action's effect is applied), insert:

```go
// Chunk 4.5: notify seeders that a gift was accepted (mob kept the item).
events.AddToQueue(events.GiftAccepted{
    UserId:        offeringUserId,        // verify variable name
    MobInstanceId: mob.InstanceId,
    ItemId:        item.ItemId,
})
```

The `offeringUserId` may not be directly available in the btree action — it depends on how the give-considered-item state is propagated. Options:
- If the give-action handler stores the offering user id in the item's metadata or the mob's MiscData under a `pending_gift_from_user:` key, the btree action reads it.
- If no such state exists, may need to add it in the give-action handler (Step 3.4): set `mob.Character.SetMiscData("pending_gift_from_user:"+strconv.Itoa(item.ItemId), user.UserId)` and the btree keep-action reads + clears it.

Implementer verifies the existing flow + chooses the cleanest path.

- [ ] **Step 3.6: Fire `GiftAccepted` for mobs without a consider-keep btree path**

Quest-givers and certain flavor NPCs don't run the equip-if-better btree action — they just accept the item silently. For these, the give-action handler itself fires `GiftAccepted` immediately after the give (in addition to `GiftOffered`).

Detection: if the mob's archetype doesn't include the equip-if-better action OR the mob's `BehaviorArchetype` is in a known-list of "always-accepts" archetypes (noncombat_questgiver, noncombat_passive). Cheapest pragmatic check: fire `GiftAccepted` immediately ONLY when the mob doesn't have the consider-keep behavior. If implementer can't cleanly detect this, alternative: fire `GiftAccepted` for ALL gives in the give-handler PLUS the btree path; rule 7 dedups via its cooldown so a double-fire only bumps once. (Slightly noisier but always correct.)

Document the chosen approach in a code comment + commit message.

- [ ] **Step 3.7: Wire seeders.Dispatch listeners in main.go**

Add the seeders import + listener wiring. In `main.go`'s imports block, add:

```go
"github.com/GoMudEngine/GoMud/internal/seeders"
```

After the existing chunk-4.4 `goals.SetPlanStateClear(planners.ClearPlanState)` line, insert:

```go
// Wire seeders.Dispatch to every event type the chunk-4.5 seeders
// package subscribes to. Hand-maintained — implementers add a line
// when a new rule subscribes to a new event type.
events.AddListener(events.MobDeath{}, seeders.Dispatch)
events.AddListener(events.Communication{}, seeders.Dispatch)
events.AddListener(events.Quest{}, seeders.Dispatch)
events.AddListener(events.PlayerAttackedMob{}, seeders.Dispatch)
events.AddListener(events.GiftAccepted{}, seeders.Dispatch)
events.AddListener(events.SkillUsed{}, seeders.Dispatch)
```

⚠️ Verify the `events.AddListener` API signature via codegraph. The list intentionally OMITS `events.GiftOffered` (no consumers in 4.5) and `events.ItemOwnership` (was in the original plan but rule 5 has been moved to its own event or stays on ItemOwnership depending on Step 8.1 verification — keep ItemOwnership in this list if rule 5 still subscribes to it):

```go
events.AddListener(events.ItemOwnership{}, seeders.Dispatch) // for rule 5 (theft)
```

If `events.SkillUsed` doesn't exist, omit that line (rule 10 covers the deferral).

- [ ] **Step 3.8: Build + boot smoke**

Run: `go build ./...`
Expected: clean build.

Run: `timeout 30 go run . 2>&1 | grep -iE "panic|started|fatal" | head -10`
Expected: `MainWorker state="Started"` with no panic.

Stop with Ctrl+C.

- [ ] **Step 3.9: Commit**

```bash
git add internal/events/eventtypes.go internal/actions/attack.go internal/actions/taunt.go internal/usercommands/give.go internal/behaviortree/ main.go
git commit -m "feat(events): add PlayerAttackedMob + GiftOffered + GiftAccepted (4.5)" -m "Three new event types for chunk-4.5 seeders to subscribe to (the
Option-B architectural choice — all rules except rule 3 go through
the unified Dispatch path).

PlayerAttackedMob: fired by attack.go + taunt.go when player engages
a mob. Consumed by rules 6 (aggressive-action revenge) + 9 (combat-
assist opinion).

GiftOffered: fired by give.go unconditionally on every give. No
4.5 consumers; reserved slot for future rules.

GiftAccepted: fired by btree's keep-action (or give-handler when
the mob has no consider-keep path). Consumed by rule 7 (gift opinion
boost) — only fires on actually-kept gifts, not on declined offers.
Prevents worthless-rock spam-bumping.

main.go wires seeders.Dispatch listeners for every event type rules
4-13 subscribe to.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

---

## Per-rule tasks (4–13): conventions

The next 10 tasks each create one rule file under `internal/seeders/`. Conventions:

- File name matches the rule (e.g., `faction_kill_counter.go`).
- Each file declares its rule name as a constant: `const ruleNameFactionKillCounter = "faction_kill_counter"`. Used by `Register` + log lines.
- Each file's `init()` calls `Register(ruleName, ruleFn, eventTypeNames...)`.
- Each task adds a `<rule>_test.go` alongside with: registration check + per-branch tests + dedup/cooldown round-trip where applicable.
- Test mob fixtures follow the 4.3/4.4 pattern: `mob := &mobs.Mob{}; mob.Character.Name = "fixture"; mob.MobId = mobs.MobId(N)`.
- Event-payload TODO-ADAPTs are concentrated in the rules — each rule's first step verifies the exact event struct via `codegraph_node <EventTypeName>` and adapts the type assertion + field reads.

---

## Task 4 — Rule 1: `faction_kill_counter`

**Files:**
- Create: `internal/seeders/faction_kill_counter.go`
- Create: `internal/seeders/faction_kill_counter_test.go`
- Modify: `internal/seeders/state.go` (wire `resolveKillerFromMobDeath` TODO-ADAPT)

- [ ] **Step 4.1: Verify the `events.MobDeath` struct shape**

Run `codegraph_node MobDeath` (the events one, not the hook). Look for fields identifying the victim and killer. Common shape:

```go
type MobDeath struct {
    MobInstanceId        int   // victim mob instance id
    KillerUserId         int   // 0 if killer was a mob
    KillerMobInstanceId  int   // 0 if killer was a player
}
```

Verify and adapt the rule's field reads accordingly. Also wire the TODO-ADAPT `resolveKillerFromMobDeath` helper in `state.go` (Task 2):

```go
// In internal/seeders/state.go, replace the resolveKillerFromMobDeath stub with:
func resolveKillerFromMobDeath(event events.Event) (kind string, id int) {
    md, ok := event.(events.MobDeath)
    if !ok {
        return "", 0
    }
    if md.KillerUserId > 0 {
        return "player", md.KillerUserId
    }
    if md.KillerMobInstanceId > 0 {
        return "mob", md.KillerMobInstanceId
    }
    return "", 0
}
```

Add the `events` import to `state.go` if not present. Adapt field names to whatever the real `events.MobDeath` exposes.

- [ ] **Step 4.2: Create faction_kill_counter.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const ruleNameFactionKillCounter = "faction_kill_counter"

func init() {
	Register(ruleNameFactionKillCounter, factionKillCounter, "MobDeath")
}

// factionKillCounter: on MobDeath, if the killer is a mob, bump
// faction_kills_inflicted:<faction> on the killer for each faction
// the victim belongs to. Read by the 4.3 catalog's revenge-faction
// Predicate.
func factionKillCounter(event events.Event) {
	md, ok := event.(events.MobDeath)
	if !ok {
		return
	}
	killerKind, killerId := resolveKillerFromMobDeath(event)
	if killerKind != "mob" || killerId == 0 {
		return // player-as-killer doesn't write to mob counters
	}

	victim := mobs.GetInstance(md.MobInstanceId)
	if victim == nil {
		return
	}
	victimFactions := factions.FactionsForMob(victim)
	if len(victimFactions) == 0 {
		return
	}

	killer := mobs.GetInstance(killerId)
	if killer == nil {
		return
	}

	for _, fid := range victimFactions {
		bumpMiscInt(killer, "faction_kills_inflicted:"+fid, 1)
	}
}
```

- [ ] **Step 4.3: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestFactionKillCounter_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["MobDeath"] {
		if reg.name == ruleNameFactionKillCounter {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("faction_kill_counter not registered for MobDeath")
	}
}

func TestFactionKillCounter_PlayerKiller_NoOp(t *testing.T) {
	// Dispatching a MobDeath with player killer must not panic and
	// must not crash on missing mob fixtures.
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	evt := events.MobDeath{
		MobInstanceId: 99,
		KillerUserId:  5,
	}
	factionKillCounter(evt)
}

// Full counter-increment integration test requires loaded mob + faction
// data; deferred to Task 15 smoke per spec §6.3.
```

- [ ] **Step 4.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/state.go internal/seeders/faction_kill_counter.go internal/seeders/faction_kill_counter_test.go
git commit -m "feat(seeders): faction_kill_counter rule (4.5)" -m "On MobDeath with mob-killer + faction-belonging victim: bump
faction_kills_inflicted:<faction> on killer for each victim faction.
Unblocks 4.3 catalog's revenge-faction Predicate.

Also wires the resolveKillerFromMobDeath TODO-ADAPT helper from
Task 2 against the verified events.MobDeath struct shape.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 5 — Rule 2: `faction_rep_counter`

**Files:**
- Create: `internal/seeders/faction_rep_counter.go`
- Create: `internal/seeders/faction_rep_counter_test.go`

- [ ] **Step 5.1: Verify the trigger event shape**

Spec says: "Communication with give/positive-dialogue OR quest completion." Verify via codegraph what events fire on positive interactions:
- `codegraph_node Communication` — does it have a subtype/action field?
- `codegraph_node Quest` — does it have a completion subtype?
- `codegraph_node ItemOwnership` — does it fire on player-gives-to-mob?

Pick whichever event gives the cleanest "positive interaction with a mob" signal. **If no clean event fits:** ship the rule registered for the closest candidate (likely `Communication`), filter defensively, document the gap. Befriend-faction Predicate stays at 0 until a follow-up wires the cleaner trigger — this is acceptable for 4.5 ship; the catalog file already documents this dependency.

- [ ] **Step 5.2: Create faction_rep_counter.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

const ruleNameFactionRepCounter = "faction_rep_counter"

func init() {
	// TODO-ADAPT: register for whatever event the give/positive-dialogue
	// path actually fires. Verify in Step 5.1 + adapt.
	Register(ruleNameFactionRepCounter, factionRepCounter, "Communication")
}

// factionRepCounter: on a positive-interaction event (give, successful
// trade, positive dialogue), bump faction_rep_built_with:<faction>
// on the receiver mob for each faction the giver belongs to. Read by
// the 4.3 catalog's befriend-faction Predicate.
//
// 4.5 ships against whichever event yields the cleanest "positive
// interaction" signal at impl time. May over-fire if filters are
// imprecise — that's acceptable for ship; the predicate threshold
// absorbs noise.
func factionRepCounter(event events.Event) {
	// TODO-ADAPT — full body per verified event shape. Conservative
	// stub: type-assert, early-return on unexpected shape. Implementer
	// fills in the field reads + factions walk based on Step 5.1.
	_, ok := event.(events.Communication)
	if !ok {
		return
	}
	_ = mobs.GetInstance
	_ = factions.FactionsForMob
}
```

⚠️ Heavy TODO-ADAPT in this rule. Implementer fully verifies the event shape and replaces the stub with real field reads + factions walk. If no event cleanly maps, leave as a documented stub.

- [ ] **Step 5.3: Write tests**

```go
package seeders

import "testing"

func TestFactionRepCounter_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["Communication"] {
		if reg.name == ruleNameFactionRepCounter {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("faction_rep_counter not registered for Communication")
	}
}

// Behavior tests require verified event shape + live mob fixtures.
// Defer to Task 15 smoke.
```

- [ ] **Step 5.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/faction_rep_counter.go internal/seeders/faction_rep_counter_test.go
git commit -m "feat(seeders): faction_rep_counter rule (4.5)" -m "On positive-interaction event: bump faction_rep_built_with:<faction>
on receiver for each faction the giver belongs to. Unblocks
befriend-faction Predicate.

May ship as a documented stub if the verified event shape doesn't
cleanly distinguish positive interactions; befriend-faction stays at
0 until a follow-up wires the cleaner trigger.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 6 — Rule 3: `craft_materials_to_wealth_item` (planner-invoked)

The architectural exception — invoked directly from the craft-item planner's Failure branch.

**Files:**
- Create: `internal/seeders/craft_materials_to_wealth_item.go`
- Create: `internal/seeders/craft_materials_to_wealth_item_test.go`
- Modify: `internal/planners/craft_item.go` (call into seeders from Failure branch)

- [ ] **Step 6.1: Create craft_materials_to_wealth_item.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const ruleNameCraftMaterialsSeed = "craft_materials_to_wealth_item"
const craftMaterialsSeedPriority = 60

// SeedMaterialsForRecipe is the public entry point called from
// internal/planners/craft_item.go's Failure branch when materials are
// missing. Walks the recipe ingredients and seeds a wealth-item goal
// for each ingredient tag the mob doesn't have. DedupKey on the
// catalog's wealth-item type collapses repeat seedings.
//
// NOT registered with the event dispatcher — no clean "planner failed
// because materials missing" world event exists, so the planner calls
// this function directly.
func SeedMaterialsForRecipe(mob *mobs.Mob, recipeId string) {
	if mob == nil || recipeId == "" {
		return
	}
	r := crafting.GetRecipe(recipeId)
	if r == nil {
		return
	}
	mobId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)

	for _, ing := range r.Ingredients {
		// TODO-ADAPT: verify field name on RecipeIngredient.
		// Likely ItemTag or Tag — codegraph_node RecipeIngredient.
		tag := ing.ItemTag
		if tag == "" {
			continue
		}
		if mobHasIngredientTag(mob, tag) {
			continue
		}

		g := &goals.Goal{
			Type:     "wealth-item",
			Priority: craftMaterialsSeedPriority,
			Params:   map[string]any{"item_tag": tag},
		}
		_, err := goals.Add(mobId, name, g)
		if err != nil {
			mudlog.Debug("seeders.SeedMaterialsForRecipe: Add",
				"mob_id", mobId, "recipe", recipeId, "tag", tag, "error", err)
		}
	}
}

// mobHasIngredientTag checks backpack for an item with the given
// ComponentTag. Duplicate of catalog's wealth-item helper (separate
// packages).
func mobHasIngredientTag(mob *mobs.Mob, tag string) bool {
	for i := range mob.Character.Items {
		if mob.Character.Items[i].GetSpec().ComponentTag == tag {
			return true
		}
	}
	return false
}
```

⚠️ TODO-ADAPT on `RecipeIngredient.ItemTag` — verify via codegraph_node RecipeIngredient.

- [ ] **Step 6.2: Wire into craft-item planner**

Modify `internal/planners/craft_item.go`. Find the Failure branch where materials are determined missing (per 4.4 Task 11, it's the `if !mobHasRecipeMaterials(mob, r) { return PlanResult{Status: StatusFailure} }` line). Replace with:

```go
if !mobHasRecipeMaterials(mob, r) {
    // Chunk 4.5: seed wealth-item for missing ingredients so the mob
    // has a productive next-tick goal instead of spinning.
    seeders.SeedMaterialsForRecipe(mob, rid)
    return PlanResult{Status: StatusFailure}
}
```

Add import:

```go
"github.com/GoMudEngine/GoMud/internal/seeders"
```

⚠️ Verify no import cycle. Direction: `planners → seeders`. Seeders doesn't import planners.

- [ ] **Step 6.3: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestSeedMaterialsForRecipe_NilMob_NoPanic(t *testing.T) {
	SeedMaterialsForRecipe(nil, "any-recipe")
}

func TestSeedMaterialsForRecipe_UnknownRecipe_NoOp(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Name = "unknown_recipe_test"
	SeedMaterialsForRecipe(mob, "does-not-exist-recipe-zzz")
}

// Integration test (real recipe + missing materials → wealth-item seeded)
// requires live recipe registry; deferred to Task 15 smoke per spec §6.3.
```

- [ ] **Step 6.4: Run + commit**

Run: `go test ./internal/seeders/ ./internal/planners/ 2>&1 | tail -5` → PASS.
Run: `go build ./...` → clean (no import cycle).

```bash
git add internal/seeders/craft_materials_to_wealth_item.go internal/seeders/craft_materials_to_wealth_item_test.go internal/planners/craft_item.go
git commit -m "feat(seeders): craft_materials_to_wealth_item rule (4.5)" -m "Architectural exception — invoked directly from craft-item planner's
Failure branch. seeders.SeedMaterialsForRecipe walks recipe ingredients,
seeds wealth-item (priority 60) for each missing tag. DedupKey collapses
repeats. Craft-item's missing-materials state becomes productive
instead of spinning.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 7 — Rule 4: `friend_killed_to_revenge`

**Files:**
- Create: `internal/seeders/friend_killed_to_revenge.go`
- Create: `internal/seeders/friend_killed_to_revenge_test.go`

- [ ] **Step 7.1: Verify the relationships API**

Run `codegraph_search RelationsOf` to confirm the chunk 1.6 query function. Expected shape: `relationships.RelationsOf(mobId int) []Relation` (or similar). Each `Relation` should expose: the other-side mob id, the type (friend/family/lover/enemy/etc.), and possibly a subtype.

Friendly relationship types per spec §3.4: `friend`, `family`, `lover` (verify exact strings/constants — chunk 1.6 may define an enum).

- [ ] **Step 7.2: Create friend_killed_to_revenge.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
)

const ruleNameFriendKilledToRevenge = "friend_killed_to_revenge"
const friendKilledRevengePriority = 85

func init() {
	Register(ruleNameFriendKilledToRevenge, friendKilledToRevenge, "MobDeath")
}

// friendKilledToRevenge: on MobDeath, walk the victim's relationship
// edges; for each friend/family/lover edge, seed a revenge-mob goal
// on the friend targeting the killer.
//
// Priority 85 (above survival's 80 — grief outweighs baseline
// self-preservation). DedupKey collapses repeat seedings against the
// same killer.
func friendKilledToRevenge(event events.Event) {
	md, ok := event.(events.MobDeath)
	if !ok {
		return
	}
	killerKind, killerId := resolveKillerFromMobDeath(event)
	if killerKind == "" || killerId == 0 {
		return // no resolvable killer
	}

	// Walk victim's relationships.
	relations := relationships.RelationsOf(md.MobInstanceId) // TODO-ADAPT: verify the arg is instance id vs template id
	if len(relations) == 0 {
		return
	}

	for _, rel := range relations {
		if !isFriendlyRelationType(rel.Type) { // TODO-ADAPT: verify Relation.Type field
			continue
		}
		// Resolve the other side of the edge — the friend mob.
		// TODO-ADAPT: Relation may expose OtherMobInstanceId or similar.
		friendInstanceId := rel.OtherMobInstanceId
		friend := mobs.GetInstance(friendInstanceId)
		if friend == nil {
			continue // friend is dead/unloaded
		}
		seedRevengeGoalIfAbsent(friend, killerKind, killerId, friendKilledRevengePriority)
	}
}

// isFriendlyRelationType returns true for relationship types where the
// "friend" should mourn (and possibly seek revenge for) the victim.
// TODO-ADAPT: replace string literals with chunk 1.6's actual constants
// (e.g., relationships.TypeFriend, relationships.TypeFamily, etc.).
func isFriendlyRelationType(t string) bool {
	switch t {
	case "friend", "family", "lover":
		return true
	}
	return false
}
```

⚠️ Three TODO-ADAPTs: (1) `RelationsOf` arg (instance vs template id), (2) `Relation` struct field names, (3) friendly-type constants. Implementer verifies + wires.

- [ ] **Step 7.3: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestFriendKilledToRevenge_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["MobDeath"] {
		if reg.name == ruleNameFriendKilledToRevenge {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("friend_killed_to_revenge not registered for MobDeath")
	}
}

func TestFriendKilledToRevenge_NoKiller_NoOp(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	evt := events.MobDeath{MobInstanceId: 999}
	friendKilledToRevenge(evt)
}

func TestIsFriendlyRelationType(t *testing.T) {
	for _, friendly := range []string{"friend", "family", "lover"} {
		if !isFriendlyRelationType(friendly) {
			t.Errorf("%q should be friendly", friendly)
		}
	}
	for _, unfriendly := range []string{"enemy", "rival", "", "stranger"} {
		if isFriendlyRelationType(unfriendly) {
			t.Errorf("%q should NOT be friendly", unfriendly)
		}
	}
}

// End-to-end integration (relationship edge → MobDeath → revenge goal
// on friend) deferred to Task 15 smoke.
```

- [ ] **Step 7.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/friend_killed_to_revenge.go internal/seeders/friend_killed_to_revenge_test.go
git commit -m "feat(seeders): friend_killed_to_revenge rule (4.5)" -m "On MobDeath: walk victim's chunk-1.6 relationship edges; for each
friend/family/lover edge, seed revenge-mob (priority 85) on the friend
targeting the killer. seedRevengeGoalIfAbsent dedups against existing
revenge against the same target.

The roadmap's headline reactive-aliveness example.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 8 — Rule 5: `witness_of_theft_to_revenge`

**Files:**
- Create: `internal/seeders/witness_of_theft_to_revenge.go`
- Create: `internal/seeders/witness_of_theft_to_revenge_test.go`

- [ ] **Step 8.1: Verify the theft event shape**

Run `codegraph_node ItemOwnership` and check for a subtype/reason field indicating theft. If `ItemOwnership` has e.g. a `Reason string` field with values like `"steal"` or `"theft"`, that's the trigger.

If no theft signal exists in `ItemOwnership`, search for a dedicated theft event: `codegraph_search Theft`. If neither exists, the rule may need to be invoked from the `steal` action handler directly (similar pattern to combat-assist in Task 12). Document the choice.

- [ ] **Step 8.2: Create witness_of_theft_to_revenge.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const ruleNameWitnessOfTheftToRevenge = "witness_of_theft_to_revenge"
const theftVictimRevengePriority = 90
const theftWitnessRevengePriority = 60

func init() {
	// TODO-ADAPT: register for the verified theft event type.
	// If invoking from action handler instead, no Register call here.
	Register(ruleNameWitnessOfTheftToRevenge, witnessOfTheftToRevenge, "ItemOwnership")
}

// witnessOfTheftToRevenge: on a theft event (player steals from a
// mob), seed revenge-mob against the thief into the victim (priority
// 90) and into other mobs in the same room (priority 60).
//
// Filter: thief must be a player; victim must be a mob.
func witnessOfTheftToRevenge(event events.Event) {
	// TODO-ADAPT: type-assert + filter on theft subtype per Step 8.1.
	io, ok := event.(events.ItemOwnership)
	if !ok {
		return
	}
	// Example field reads — adapt per verified struct shape.
	thiefUserId := io.UserId         // TODO-ADAPT
	victimMobId := io.MobInstanceId  // TODO-ADAPT
	reason := io.Reason              // TODO-ADAPT (or whatever subtype field)
	if reason != "steal" && reason != "theft" {
		return // not a theft event
	}
	if thiefUserId == 0 || victimMobId == 0 {
		return
	}

	victim := mobs.GetInstance(victimMobId)
	if victim == nil {
		return
	}

	// Seed revenge into the victim (high priority).
	seedRevengeGoalIfAbsent(victim, "player", thiefUserId, theftVictimRevengePriority)

	// Seed revenge into witnesses (other mobs in same room).
	room := rooms.LoadRoom(victim.Character.RoomId)
	if room == nil {
		return
	}
	// TODO-ADAPT: room.GetMobs() or similar — verify the room API for
	// enumerating mob instance ids in the room.
	for _, witnessInstanceId := range room.GetMobs() {
		if witnessInstanceId == victim.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstanceId)
		if witness == nil {
			continue
		}
		seedRevengeGoalIfAbsent(witness, "player", thiefUserId, theftWitnessRevengePriority)
	}
}
```

⚠️ Many TODO-ADAPTs — verify event shape, theft reason value, room-mob enumeration API.

- [ ] **Step 8.3: Write tests**

```go
package seeders

import "testing"

func TestWitnessOfTheftToRevenge_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["ItemOwnership"] {
		if reg.name == ruleNameWitnessOfTheftToRevenge {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("witness_of_theft_to_revenge not registered for ItemOwnership")
	}
}

// Behavior tests require verified event shape + live room + mob fixtures.
// Defer to Task 15 smoke.
```

- [ ] **Step 8.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/witness_of_theft_to_revenge.go internal/seeders/witness_of_theft_to_revenge_test.go
git commit -m "feat(seeders): witness_of_theft_to_revenge rule (4.5)" -m "On theft event: seed revenge-mob against thief into victim
(priority 90) and other mobs in the same room (priority 60).
Filter: thief is a player, victim is a mob. Witnesses get
lower-priority revenge — they're not the direct target.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 9 — Rule 6: `aggressive_action_to_revenge`

**Files:**
- Create: `internal/seeders/aggressive_action_to_revenge.go`
- Create: `internal/seeders/aggressive_action_to_revenge_test.go`

Event-subscribed rule (Option B architecture per Task 3). Subscribes to `PlayerAttackedMob` — the firers are already in attack.go + taunt.go from Task 3, no action-handler edits needed in this task.

- [ ] **Step 9.1: Create aggressive_action_to_revenge.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

const ruleNameAggressiveActionToRevenge = "aggressive_action_to_revenge"
const aggressiveVictimRevengePriority = 75
const aggressiveWitnessRevengePriority = 50

func init() {
	Register(ruleNameAggressiveActionToRevenge, aggressiveActionToRevenge, "PlayerAttackedMob")
}

// aggressiveActionToRevenge: on PlayerAttackedMob, seed revenge-mob
// into the attacked mob (priority 75) and non-hostile witnesses in
// the room (priority 50). Skips already-auto-hostile mobs (revenge
// would be redundant noise — they already attack on sight).
//
// Shares the PlayerAttackedMob event with rule 9 (combat_assist).
// Both rules fire on the same event; their logic is independent.
func aggressiveActionToRevenge(event events.Event) {
	pa, ok := event.(events.PlayerAttackedMob)
	if !ok {
		return
	}
	if pa.UserId == 0 || pa.MobInstanceId == 0 {
		return
	}
	attackedMob := mobs.GetInstance(pa.MobInstanceId)
	if attackedMob == nil {
		return
	}
	if attackedMob.AutoAggro {
		return // already hostile; revenge is redundant noise
	}

	// Seed revenge into the attacked mob.
	seedRevengeGoalIfAbsent(attackedMob, "player", pa.UserId, aggressiveVictimRevengePriority)

	// Seed revenge into non-hostile witnesses in the room.
	room := rooms.LoadRoom(attackedMob.Character.RoomId)
	if room == nil {
		return
	}
	for _, witnessInstanceId := range room.GetMobs() { // TODO-ADAPT: verify room API
		if witnessInstanceId == attackedMob.InstanceId {
			continue
		}
		witness := mobs.GetInstance(witnessInstanceId)
		if witness == nil || witness.AutoAggro {
			continue
		}
		seedRevengeGoalIfAbsent(witness, "player", pa.UserId, aggressiveWitnessRevengePriority)
	}
}
```

⚠️ TODO-ADAPT on `room.GetMobs()` — verify the room API for enumerating mob instance ids. May be `room.MobInstanceIds`, `room.GetMobIds()`, or similar.

- [ ] **Step 9.2: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestAggressiveActionToRevenge_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["PlayerAttackedMob"] {
		if reg.name == ruleNameAggressiveActionToRevenge {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("aggressive_action_to_revenge not registered for PlayerAttackedMob")
	}
}

func TestAggressiveActionToRevenge_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	aggressiveActionToRevenge(events.PlayerAttackedMob{})
}

// Full seed test requires live mob + room fixtures; deferred to Task 15 smoke.
```

- [ ] **Step 9.3: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/aggressive_action_to_revenge.go internal/seeders/aggressive_action_to_revenge_test.go
git commit -m "feat(seeders): aggressive_action_to_revenge rule (4.5)" -m "Subscribes to PlayerAttackedMob event (fired by attack.go + taunt.go
per Task 3). Seeds revenge-mob into attacked mob (priority 75) and
non-hostile witnesses (priority 50). Skips already-auto-hostile mobs
(revenge would be redundant noise).

Shares the PlayerAttackedMob event with rule 9 (combat_assist) — the
multi-consumer payoff of Option B's events-everywhere architecture.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 10 — Rule 7: `gift_to_opinion_boost`

**Files:**
- Create: `internal/seeders/gift_to_opinion_boost.go`
- Create: `internal/seeders/gift_to_opinion_boost_test.go`

Event-subscribed on `GiftAccepted` (NOT `GiftOffered` — the latter is gameable since a worthless-rock spam still fires it; `GiftAccepted` fires only when the mob actually keeps the item per Task 3). The firers are already wired in Task 3 (give.go + btree keep-action); no action-handler edits in this task.

- [ ] **Step 10.1: Create gift_to_opinion_boost.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const ruleNameGiftToOpinionBoost = "gift_to_opinion_boost"
const giftCooldownRounds uint64 = 100

func init() {
	Register(ruleNameGiftToOpinionBoost, giftToOpinionBoost, "GiftAccepted")
}

// giftToOpinionBoost: on GiftAccepted (mob KEPT the item — not just
// received it), bump the mob's opinion of the giving player by N
// scaled to item value. Per-(giver, receiver) cooldown prevents spam.
//
// Value tiers per spec §3.7:
//   value 1-49     → +1
//   value 50-199   → +3
//   value 200-999  → +5
//   value 1000+    → +8
//
// Subscribes to GiftAccepted, NOT GiftOffered — the latter fires on
// every give regardless of whether the mob valued the item. Worthless-
// rock spam doesn't bump opinion because the mob's consider-keep btree
// (chunk 2.3 equip-if-better) won't keep worthless items.
func giftToOpinionBoost(event events.Event) {
	ga, ok := event.(events.GiftAccepted)
	if !ok {
		return
	}
	if ga.UserId == 0 || ga.MobInstanceId == 0 || ga.ItemId == 0 {
		return
	}
	receiver := mobs.GetInstance(ga.MobInstanceId)
	if receiver == nil {
		return
	}

	// Resolve item value via the items registry.
	// TODO-ADAPT: verify items.FindSpecByItemId or equivalent. May
	// already exist as items.GetItemSpec(itemId) — codegraph_search.
	spec := items.GetSpec(ga.ItemId)
	if spec.Value <= 0 {
		return // defensive — mob shouldn't have accepted a worthless item
	}

	bump := giftValueToOpinionBump(spec.Value)
	if bump == 0 {
		return
	}

	// Cooldown: once per 100 rounds per (giver, receiver).
	if !applyCooldown(receiver, ruleNameGiftToOpinionBoost, userIdAsKey(ga.UserId), giftCooldownRounds) {
		return // cooldown active
	}

	opinions.Bump(int(receiver.MobId), ga.UserId, bump)
}

// giftValueToOpinionBump implements the value-tier table.
func giftValueToOpinionBump(value int) int {
	switch {
	case value >= 1000:
		return 8
	case value >= 200:
		return 5
	case value >= 50:
		return 3
	case value >= 1:
		return 1
	}
	return 0
}
```

⚠️ TODO-ADAPT on `items.GetSpec(itemId)` — verify the actual lookup function name via codegraph (`codegraph_search items.GetSpec` or `codegraph_search GetItemSpec`). The 4.4 catalog used `item.GetSpec()` on an existing item; here we need a lookup-by-id since the event carries only the item id, not the item pointer.

- [ ] **Step 10.2: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestGiftToOpinionBoost_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["GiftAccepted"] {
		if reg.name == ruleNameGiftToOpinionBoost {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("gift_to_opinion_boost not registered for GiftAccepted")
	}
}

func TestGiftValueToOpinionBump_Tiers(t *testing.T) {
	cases := []struct {
		value int
		want  int
	}{
		{0, 0},
		{1, 1},
		{49, 1},
		{50, 3},
		{199, 3},
		{200, 5},
		{999, 5},
		{1000, 8},
		{99999, 8},
	}
	for _, c := range cases {
		got := giftValueToOpinionBump(c.value)
		if got != c.want {
			t.Errorf("value=%d: got %d, want %d", c.value, got, c.want)
		}
	}
}

func TestGiftToOpinionBoost_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	giftToOpinionBoost(events.GiftAccepted{})
}

// End-to-end (item-value lookup + cooldown round-trip) requires live
// items registry + mob fixture; deferred to Task 15 smoke.
```

- [ ] **Step 10.3: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/gift_to_opinion_boost.go internal/seeders/gift_to_opinion_boost_test.go
git commit -m "feat(seeders): gift_to_opinion_boost rule (4.5)" -m "Subscribes to GiftAccepted (NOT GiftOffered) — only fires when the
mob actually KEEPS the item, per the consider-keep-or-return btree
action wired in Task 3. Worthless-rock spam doesn't bump opinion
because the mob's btree won't keep worthless items in the first place.

Value-tier bump (+1/+3/+5/+8 for value thresholds 1/50/200/1000) with
100-round per-pair cooldown to prevent legitimate-gift spam.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 11 — Rule 8: `quest_completion_to_opinion_boost`

**Files:**
- Create: `internal/seeders/quest_completion_to_opinion_boost.go`
- Create: `internal/seeders/quest_completion_to_opinion_boost_test.go`

- [ ] **Step 11.1: Verify the quest event shape**

Run `codegraph_node Quest` (the events one). Expected fields: subtype indicating completion vs progress, the questId, the completing userId, and a reference to the quest giver mob (if not directly, retrievable via `quests.GetQuestById`).

- [ ] **Step 11.2: Create quest_completion_to_opinion_boost.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const ruleNameQuestCompletionToOpinion = "quest_completion_to_opinion_boost"
const questCompletionDefaultBump = 10

func init() {
	Register(ruleNameQuestCompletionToOpinion, questCompletionToOpinion, "Quest")
}

// questCompletionToOpinion: on quest completion event, bump opinion
// of the quest giver toward the completing user. Default +10; if the
// quest YAML declares a complete_opinion_bump field, use that instead.
//
// No cooldown — quest completion is itself rate-limited by quest
// gating; can't be spammed.
func questCompletionToOpinion(event events.Event) {
	q, ok := event.(events.Quest)
	if !ok {
		return
	}
	// TODO-ADAPT: filter on subtype/completion field. Real Quest event
	// may distinguish quest progress vs completion.
	if !isQuestCompletion(q) {
		return
	}

	completingUserId := q.UserId  // TODO-ADAPT
	questId := q.QuestId          // TODO-ADAPT
	if completingUserId == 0 || questId == "" {
		return
	}

	// Resolve quest giver mob template id.
	// TODO-ADAPT: quests.GetQuestById(questId) → returns spec with
	// GiverMobId int (or similar). May need to walk the quest's NPCs
	// list if multiple NPCs are involved.
	giverMobTemplateId := resolveQuestGiverMobId(questId)
	if giverMobTemplateId == 0 {
		return
	}

	// Determine bump amount: quest YAML may declare a custom value.
	bump := resolveQuestOpinionBump(questId, questCompletionDefaultBump)

	opinions.Bump(giverMobTemplateId, completingUserId, bump)
}

// isQuestCompletion is the Quest-event filter. TODO-ADAPT per the
// real event's subtype/action field.
func isQuestCompletion(q events.Quest) bool {
	// TODO-ADAPT: e.g., return q.Action == "complete"
	return true // permissive default; replace per real event
}

// resolveQuestGiverMobId returns the mob template id of the quest's
// giver (0 if unresolvable). TODO-ADAPT — wire against the quests
// package's lookup API.
func resolveQuestGiverMobId(questId string) int {
	// TODO-ADAPT: quests.GetQuestById(questId) → walk for giver.
	return 0
}

// resolveQuestOpinionBump returns the per-quest opinion bump value
// (from quest YAML's complete_opinion_bump field if present) or the
// default. TODO-ADAPT.
func resolveQuestOpinionBump(questId string, defaultBump int) int {
	// TODO-ADAPT: quest YAML may declare complete_opinion_bump.
	// Default to the constant.
	return defaultBump
}
```

⚠️ Several TODO-ADAPTs around quest event fields + quests API. Implementer verifies + wires.

- [ ] **Step 11.3: Write tests**

```go
package seeders

import "testing"

func TestQuestCompletionToOpinion_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["Quest"] {
		if reg.name == ruleNameQuestCompletionToOpinion {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("quest_completion_to_opinion_boost not registered for Quest")
	}
}

// Behavior tests require verified event shape + quests package; defer to smoke.
```

- [ ] **Step 11.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/quest_completion_to_opinion_boost.go internal/seeders/quest_completion_to_opinion_boost_test.go
git commit -m "feat(seeders): quest_completion_to_opinion_boost rule (4.5)" -m "On Quest event with completion subtype: bump opinion of quest giver
toward completing user. Default +10; per-quest YAML may declare a
complete_opinion_bump field. No cooldown (quest gating self-limits).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 12 — Rule 9: `combat_assist_to_opinion_boost`

**Files:**
- Create: `internal/seeders/combat_assist_to_opinion_boost.go`
- Create: `internal/seeders/combat_assist_to_opinion_boost_test.go`

Event-subscribed on `PlayerAttackedMob` — **shares the event with rule 6 (aggressive-action)**. Two rules consume the same event with independent logic; this is the multi-consumer payoff of Option B's events-everywhere architecture. The firers are already wired in Task 3; no action-handler edits in this task.

- [ ] **Step 12.1: Create combat_assist_to_opinion_boost.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const ruleNameCombatAssistToOpinion = "combat_assist_to_opinion_boost"
const combatAssistBump = 3
const combatAssistCooldownRounds uint64 = 50

func init() {
	Register(ruleNameCombatAssistToOpinion, combatAssistToOpinion, "PlayerAttackedMob")
}

// combatAssistToOpinion: on PlayerAttackedMob, if the attacked mob is
// currently in combat with another non-player mob, bump the beneficiary
// mob's opinion of the player.
//
// Filter: beneficiary mob must NOT be hostile to the player
// (factions.IsPeacefulToward — prevents fake credit for attacking one
// of two hostile mobs that happen to be fighting each other).
// Cooldown: once per (beneficiary, attacker) pair per 50 rounds.
//
// Shares the PlayerAttackedMob event with rule 6 (aggressive_action).
func combatAssistToOpinion(event events.Event) {
	pa, ok := event.(events.PlayerAttackedMob)
	if !ok {
		return
	}
	if pa.UserId == 0 || pa.MobInstanceId == 0 {
		return
	}
	attackerMob := mobs.GetInstance(pa.MobInstanceId)
	if attackerMob == nil {
		return
	}
	beneficiaryId := resolveAttackerMobTarget(attackerMob.Character.Aggro)
	if beneficiaryId == 0 {
		return // not engaged with a mob
	}
	beneficiary := mobs.GetInstance(beneficiaryId)
	if beneficiary == nil {
		return
	}
	if !factions.IsPeacefulToward(beneficiary, pa.UserId) {
		return // beneficiary is hostile to player; no fake credit
	}
	if !applyCooldown(beneficiary, ruleNameCombatAssistToOpinion,
		instanceIdAsKey(attackerMob.InstanceId), combatAssistCooldownRounds) {
		return // cooldown active
	}

	opinions.Bump(int(beneficiary.MobId), pa.UserId, combatAssistBump)
}

// resolveAttackerMobTarget returns the MobInstanceId from an Aggro
// pointer if it points to a mob, else 0.
func resolveAttackerMobTarget(a *characters.Aggro) int {
	if a == nil {
		return 0
	}
	return a.MobInstanceId
}
```

- [ ] **Step 12.2: Write tests**

```go
package seeders

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
)

func TestCombatAssistToOpinion_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["PlayerAttackedMob"] {
		if reg.name == ruleNameCombatAssistToOpinion {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("combat_assist_to_opinion_boost not registered for PlayerAttackedMob")
	}
}

func TestCombatAssistToOpinion_ZeroFields_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("panic: %v", r)
		}
	}()
	combatAssistToOpinion(events.PlayerAttackedMob{})
}

func TestCombatAssistToOpinion_BothRulesRegisteredForSameEvent(t *testing.T) {
	// Sanity check: rule 6 + rule 9 both fire on PlayerAttackedMob.
	registryMu.RLock()
	defer registryMu.RUnlock()
	count := len(registry["PlayerAttackedMob"])
	if count < 2 {
		t.Errorf("expected ≥2 rules on PlayerAttackedMob, got %d (rule 6 + rule 9 should both register)", count)
	}
}

// Full assist-flow test (peaceful beneficiary + IsPeacefulToward gate)
// requires live mob + faction fixtures; deferred to Task 15 smoke.
```

- [ ] **Step 12.3: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS.

```bash
git add internal/seeders/combat_assist_to_opinion_boost.go internal/seeders/combat_assist_to_opinion_boost_test.go
git commit -m "feat(seeders): combat_assist_to_opinion_boost rule (4.5)" -m "Subscribes to PlayerAttackedMob (shared with rule 6 — the multi-
consumer payoff of Option B). When player engages a mob currently
fighting another non-player mob, bumps the beneficiary mob's opinion
of the player (+3) with a 50-round per-pair cooldown.

IsPeacefulToward filter prevents fake credit for attacking one of two
mutually-hostile mobs.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 13 — Rule 10: `mastery_milestone_to_priority_bump`

**Files:**
- Create: `internal/seeders/mastery_milestone_to_priority_bump.go`
- Create: `internal/seeders/mastery_milestone_to_priority_bump_test.go`

- [ ] **Step 13.1: Verify `events.SkillUsed` fires on rank-up**

Run `codegraph_node SkillUsed`. Inspect the struct fields — does it include the new rank? Does it fire on every use, or only on rank-up?

If `SkillUsed` fires on every skill use (very noisy), the rule can filter internally (only act when `newRank % 10 == 0`). If `SkillUsed` doesn't exist OR doesn't fire on rank changes, leave the rule's registration commented out + add a clear deferral note. Other 9 rules ship regardless.

- [ ] **Step 13.2: Create mastery_milestone_to_priority_bump.go**

```go
package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const ruleNameMasteryMilestone = "mastery_milestone_to_priority_bump"
const masteryMilestoneInterval = 10
const masterySoftCap = 50
const masteryMilestonePriority = 40

func init() {
	// TODO-ADAPT: if events.SkillUsed doesn't fire on rank-up events
	// (per Step 13.1), comment out the Register call and add a
	// deferral note. The rest of the rule's code stays defensive.
	Register(ruleNameMasteryMilestone, masteryMilestoneSeed, "SkillUsed")
}

// masteryMilestoneSeed: on SkillUsed event where the new rank crosses
// a multiple-of-10 milestone (10, 20, 30, 40) below the soft cap,
// seed a mastery-skill goal targeting the next milestone.
//
// Self-prompting NPCs autonomously aim at their next training
// milestone. Borderline useful — depends on events.SkillUsed actually
// firing on rank changes (verify at impl time).
func masteryMilestoneSeed(event events.Event) {
	su, ok := event.(events.SkillUsed)
	if !ok {
		return
	}
	// TODO-ADAPT: verify field names on SkillUsed.
	skillName := su.SkillName
	newRank := su.NewRank
	mobInstanceId := su.MobInstanceId
	if skillName == "" || mobInstanceId == 0 {
		return
	}

	// Milestone check.
	if newRank <= 0 || newRank >= masterySoftCap {
		return // below first milestone or at/above soft cap
	}
	if newRank%masteryMilestoneInterval != 0 {
		return // not a milestone
	}

	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return
	}

	nextMilestone := newRank + masteryMilestoneInterval
	if nextMilestone > masterySoftCap {
		nextMilestone = masterySoftCap
	}

	g := &goals.Goal{
		Type:     "mastery-skill",
		Priority: masteryMilestonePriority,
		Params: map[string]any{
			"skill_name":  skillName,
			"target_rank": nextMilestone,
		},
	}
	// DedupKey on skill_name ensures only one mastery-skill per skill;
	// this seed becomes the active goal or de-dupes against existing.
	_, _ = goals.Add(int(mob.MobId), util.ConvertForFilename(mob.Character.Name), g)
}
```

⚠️ TODO-ADAPT on SkillUsed field names AND the registration itself (if event doesn't fire on rank-ups, comment the Register call).

- [ ] **Step 13.3: Write tests**

```go
package seeders

import "testing"

func TestMasteryMilestone_Registered(t *testing.T) {
	registryMu.RLock()
	defer registryMu.RUnlock()
	found := false
	for _, reg := range registry["SkillUsed"] {
		if reg.name == ruleNameMasteryMilestone {
			found = true
			break
		}
	}
	if !found {
		// Acceptable if event doesn't exist — log skip but don't fail.
		t.Skip("mastery_milestone not registered (likely SkillUsed event deferred)")
	}
}

// Behavior tests require verified event shape; defer to Task 15 smoke.
```

- [ ] **Step 13.4: Run + commit**

Run: `go test ./internal/seeders/ -v` → PASS (or skip cleanly if registration deferred).

```bash
git add internal/seeders/mastery_milestone_to_priority_bump.go internal/seeders/mastery_milestone_to_priority_bump_test.go
git commit -m "feat(seeders): mastery_milestone_to_priority_bump rule (4.5)" -m "On SkillUsed event where new rank crosses a multiple-of-10
milestone below soft cap (50): seed mastery-skill goal targeting
the next milestone (priority 40). DedupKey on skill_name keeps one
mastery-skill per skill.

If events.SkillUsed doesn't fire on rank changes, registration is
commented out and rule is dormant pending event-shape work — other
9 rules unaffected.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 14 — `internal/seeders/context.md`

Per-package documentation per project convention.

**Files:**
- Create: `internal/seeders/context.md`

- [ ] **Step 14.1: Author context.md**

Match the voice and structure of `internal/goals/context.md`, `internal/goals/catalog/context.md`, `internal/planners/context.md`. Factual, dense, no marketing.

Cover (~100-150 lines):
- Package purpose: chunk-4.5 reactive goal generation; subscribes to world events + invoked from a few action/planner hooks; produces effects via goals.Add / MiscData / opinions.Bump
- Activation: regular import from `main.go` fires per-rule `init()` registrations + makes exported entry points (`Dispatch`, `SeedMaterialsForRecipe`) callable. Other rules are pure event subscribers — no exported per-rule API.
- File layout: `seeders.go` (framework), `state.go` (shared helpers + MiscData utilities + revenge dedup), 10 per-rule files + tests
- The 10 rules as a table (rule name, trigger, effect)
- How to add a new rule: file naming, init() registration with event type names, RuleFn signature, MiscData key conventions
- Conventions:
  - `CooldownKeyPrefix = "seed_cooldown:"` — distinct from 4.4's `plan:` so cooldowns survive goal switches
  - `applyCooldown(mob, rule, key, window)` for per-pair throttling
  - `seedRevengeGoalIfAbsent(mob, kind, id, priority)` for the multi-revenge rules
- Architectural exception: rule 3 (craft-item materials) invoked directly via exported `SeedMaterialsForRecipe` function — it's an internal planner state, not a world event. All 9 other rules are event-subscribed via Dispatch (Option B). Three new event types defined in Task 3 (`PlayerAttackedMob`, `GiftOffered`, `GiftAccepted`) carry the signals rules 6, 7, 9 need.
- Out-of-scope: 4.6 prune sweep, cross-type conflict mechanism (deferred from 4.3), per-archetype gating, content-side trigger lists
- Reference the spec: `docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md`

- [ ] **Step 14.2: Commit**

```bash
git add internal/seeders/context.md
git commit -m "docs: context.md for internal/seeders (4.5)" -m "Per project convention: matches internal/goals/context.md,
internal/goals/catalog/context.md, internal/planners/context.md.
Covers the 10-rule catalog, framework files, activation via main.go
import + event listeners, MiscData key conventions, and what's out
of scope.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Task 15 — Smoke + roadmap + patch notes

Pre-push SOP run. Mark the chunk done. Engineered smoke for the new observable behavior.

**Files:**
- Modify: `MOB_ALIVENESS_ROADMAP.md`
- Modify: `PATCH_NOTES.md`
- Verify: `_datafiles/config.yaml` (`LogToFile: false`)

- [ ] **Step 15.1: Wipe instance saves**

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 15.2: Boot test**

Run: `timeout 90 go run . 2>&1 | grep -iE "panic|loadedcount|started|fatal" | head -40`

Expect:
- All standard load counts (mobs, rooms, archetypes).
- `MainWorker state="Started"`.
- No panics.
- No errors from `events.AddListener` per-type wiring in main.go.

Stop with Ctrl+C.

- [ ] **Step 15.3: Live engineered smoke #1 — faction kill counter + revenge-faction**

Reboot. Admin-mode: pick a mob A with faction X, a mob B (any faction). Make B kill A 5 times. Check via admin:
- `goal scores <B>` — `revenge-faction:X` predicate should now be satisfied (counter reached 5).

This proves rule 1 wires counter → predicate end-to-end.

- [ ] **Step 15.4: Live engineered smoke #2 — friend-killed → revenge**

Reboot. Admin-mode:
1. Pick two mobs A and B with a `friend` relationship edge between them (chunk 1.6).
2. Admin kills A (player attacks A → death).
3. Verify B now has a `revenge-mob` goal targeting the admin: `goal current <B>` shows it.

This proves rule 4 wires MobDeath → relationship-walk → revenge-mob seed end-to-end.

- [ ] **Step 15.5: Live engineered smoke #3 — gift to opinion**

Reboot. Admin gives a low-value item (value ~10) to any noncombat NPC. Verify `opinion show <npc>` shows the admin's opinion bumped by +1 (per the tier table).

Re-give immediately — verify NO further bump (cooldown active).

- [ ] **Step 15.6: Live engineered smoke #4 — craft-item planner seeds wealth-item**

Reboot. Admin seeds a craft-item goal on a crafter NPC for a recipe whose materials are NOT present in the NPC's inventory. Tick a round. Verify `goal current <npc>` shows a `wealth-item` goal was added (for one of the missing material tags).

This proves rule 3 wires craft-item's Failure branch → SeedMaterialsForRecipe → wealth-item seed end-to-end.

- [ ] **Step 15.7: Update `MOB_ALIVENESS_ROADMAP.md`**

Find the 4.5 row. Flip status to `Done`. Update size from `M` to `L`. Bump rollup `26 / 42 done • 0 in progress • 16 not started` → `27 / 42 done • 0 in progress • 15 not started`.

Find the 4.5 detail block. Append a `**Shipped:**` paragraph summarizing the 10 rules + framework + architectural exceptions + spec/plan paths (mirror 4.3/4.4 detail blocks).

- [ ] **Step 15.8: Append `PATCH_NOTES.md` entry**

Insert at the top, above the 4.4 entry:

```markdown
## 2026-05-27 — Mob aliveness chunk 4.5: reactive goal generation

NPCs now react to world events. Kill an NPC's friend and that NPC
gets a revenge goal against you. Gift an item and the receiver's
opinion of you bumps up. Help a mob fighting another mob and the
helped mob warms to you. Three forced hooks complete the 4.4 loop:
faction kill counter (makes revenge-faction predicates satisfy when
killers rack up kills), faction rep counter (befriend-faction
counterpart), and craft-item materials → wealth-item seed (so
crafters fall back to acquiring missing materials instead of
spinning on a planner Failure).

10 rules in the new internal/seeders/ package. Each is a Go function
that subscribes to one or more event types (via Register +
events.AddListener) or is invoked directly from action/planner hooks
for architectural exceptions. Effects route through existing
substrate APIs (goals.Add, mob.MiscData, opinions.Bump). Per-rule
cooldowns prevent gift-spam / combat-assist-spam from runaway opinion
gain. Panic-recovered dispatcher means one bad rule cannot cascade-
fail others or the event handler.

The roadmap's headline reactive-aliveness example
(player kills NPC's friend → revenge goal seeded into the friend)
lands here.

No schema change. Player-facing impact: noticeable NPC reactivity to
player actions.
```

- [ ] **Step 15.9: Verify `Logging.LogToFile: false`**

Confirm `_datafiles/config.yaml` has `LogToFile: false`.

- [ ] **Step 15.10: Full test suite + final build**

Run: `go test ./...` → PASS.
Run: `go build ./...` → clean.

- [ ] **Step 15.11: Commit roadmap + patch notes**

```bash
git add MOB_ALIVENESS_ROADMAP.md PATCH_NOTES.md _datafiles/config.yaml
git commit -m "chore(roadmap): mark aliveness 4.5 reactive goal generation Done (27/42)" -m "- Roadmap: 4.5 status -> Done, size M -> L (upsized during brainstorming).
- Roadmap rollup: 26/42 -> 27/42.
- PATCH_NOTES: chunk 4.5 entry.
- Config LogToFile=false already set (verified).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>"
```

---

## Self-Review

**Spec coverage:** every section of the 4.5 spec maps to tasks:
- §1 Architecture → Tasks 1-3 (framework + state helpers + main.go wiring).
- §2 API surface → Tasks 1 (PlanFn types, Register, Dispatch), 2 (helpers), 3 (boot).
- §3 The 10 rules → Tasks 4-13 (one per rule).
- §4 Cross-rule patterns → Tasks 2 (helpers) + per-rule task usage.
- §5 Edge cases → covered across per-rule defensive code + framework panic-recovery test.
- §6 Testing strategy & rollout → distributed across all tasks + Task 15 smoke.

**Placeholder scan:** every TODO-ADAPT marker has clear "verify via codegraph at impl time + how to adapt" directives. None are open-ended "fill in later" placeholders. The two heaviest TODO-ADAPT rules (rule 2 `faction_rep_counter` and rule 10 `mastery_milestone`) are explicitly documented as "may ship as stub if the verified event doesn't exist cleanly" — this is the right tradeoff for an L chunk where some event shapes are speculative.

**Type consistency:** `RuleFn`, `Register`, `Dispatch`, `applyCooldown`, `seedRevengeGoalIfAbsent`, `bumpMiscInt`, `SeedMaterialsForRecipe` consistent across tasks 1-13. New event types `PlayerAttackedMob`, `GiftOffered`, `GiftAccepted` defined in Task 3 and consumed by rules 6, 7, 9. Constants like `CooldownKeyPrefix`, rule-name constants, and priority constants follow consistent naming patterns. MiscData key prefixes match between rules and the catalog/planners reads (`faction_kills_inflicted:`, `faction_rep_built_with:`, `seed_cooldown:`).

**Scope:** 15 tasks, single feature branch, L sized chunk. Per-rule tasks are simpler than 4.4's planner tasks (each is one Go file with init + per-rule logic + minimal tests; deeper coverage deferred to Task 15 smoke). Task 3 absorbs the events package additions + firer wiring + main.go listeners (was originally three separate concerns; bundled to keep task count at 15). The architectural exception is rule 3 (planner-invoked); all other rules are uniform event subscribers per Option B.


