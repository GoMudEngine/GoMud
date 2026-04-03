# Command Unification — Substage 5: Skills + NPC Behavior Foundation

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Unify sneak (mob version is a stub), extract shared skill-check patterns for search/forage, add skill progression to mob craft/forage paths, and lay the foundation for NPC economy behavior.

**Architecture:** Sneak gets a proper shared action with opposed rolls for mobs. Search and forage get shared score calculation helpers. Craft/buy/sell mob capabilities are deferred — they need design work beyond unification.

**Tech Stack:** Go, testify/assert, existing skill/buff systems

---

## Scope Assessment

| Command | Action This Substage | Reason |
|---------|---------------------|--------|
| Sneak | Full unification | Mob version is a stub (auto-success, no rolls) |
| Search | Extract score calc helper | No mob equivalent; helper enables future mob search |
| Forage | Extract score calc helper | No mob equivalent; helper enables future mob forage |
| Craft | Progression only | Mob crafting needs design work; just ensure progression fires |
| Buy/Sell | Defer | No mob equivalent; NPC economy is a major feature |

---

### Task 1: Shared Sneak Action

**Files:**
- Create: `internal/actions/sneak.go`
- Modify: `internal/usercommands/skill.skullduggery.sneak.go`
- Modify: `internal/mobcommands/sneak.go`

The mob sneak is currently a 24-line stub that auto-succeeds with no
rolls. After unification, mobs use the same opposed-roll logic as
players — a mob with low skullduggery might fail to hide.

- [ ] **Step 1: Read both sneak files fully**

Read `internal/usercommands/skill.skullduggery.sneak.go` (153 lines)
and `internal/mobcommands/sneak.go` (24 lines).

Key shared logic to extract:
- Sneak score calculation: `Dexterity + (SkullduggerySkill × 25.0)` with illumination penalty
- Observer score: `Perception + (SearchSkill × 25.0)`
- Opposed roll against each observer in the room
- Buff 9 application on success
- Misc-data `sneaking` flag

Also read `internal/usercommands/go.go` to find `calcSneakScore` — this
function is already used for detection on room entry and should be
reused or moved to the shared package.

- [ ] **Step 2: Create shared sneak action**

```go
type SneakResult struct {
    Success     bool
    SpottedBy   string  // name of who spotted you (empty if success)
    AlreadyHidden bool
    InCombat    bool
    OnCooldown  bool
    SkillTooLow bool
}

func ExecuteSneak(actor Actor) SneakResult
```

ExecuteSneak should:
1. Check already hidden → `{AlreadyHidden: true}`
2. Check in combat (`Aggro != nil`) → `{InCombat: true}`
3. Calculate sneak score using shared `CalcSneakScore`
4. Roll against each player in room (opposed roll)
5. Roll against each mob in room (opposed roll, skip self)
6. If spotted by anyone → `{Success: false, SpottedBy: name}`
7. If success → apply buff 9, set misc-data `sneaking`
8. Return result

Move `calcSneakScore` from `go.go` to `internal/actions/sneak.go` as
an exported `CalcSneakScore` function. Update `go.go` to call the
shared version.

**Design decision for mobs:** Mobs skip the skill gate (`Skullduggery >= 1`)
since mob skills work differently. But they DO get the opposed rolls —
a mob with low Dex and no skullduggery skill will usually fail to hide.
This makes mob stealth feel earned rather than automatic.

- [ ] **Step 3: Update user sneak wrapper**

Refactor to use `actions.ExecuteSneak()`. Keep user-specific:
- Skill gate check (Skullduggery >= 1)
- Failure cooldown (`SneakFailCooldown`)
- Party member exclusion from detection
- Skill progression event
- Quest engine notification
- Player messaging

- [ ] **Step 4: Update mob sneak wrapper**

Refactor to use `actions.ExecuteSneak()`. Much more capable now:
- Mobs actually roll to hide instead of auto-succeeding
- Mob-specific: no skill gate, no cooldown on failure, no messaging
- Skill progression: `mob.Character.OnSkillUse("skullduggery", 0)`

- [ ] **Step 5: Verify build + commit**

```bash
git commit -m "refactor: shared Sneak action — mobs now roll to hide"
```

---

### Task 2: Shared Skill Score Helpers

**Files:**
- Create: `internal/actions/skill_helpers.go`
- Modify: `internal/usercommands/go.go` (update calcSneakScore reference)

Extract the common score calculation patterns used by search, forage,
and sneak into shared helpers.

- [ ] **Step 1: Create skill score helpers**

```go
// CalcSneakScore returns the stealth score for an actor.
// Score = Dexterity + (Skullduggery × SkillMultiplier × 25)
// Penalized by illumination (lit rooms are harder to hide in).
func CalcSneakScore(char *characters.Character, roomVisibility int) float64

// CalcSearchScore returns the observation score for an actor.
// Score = Perception + (Search × SkillMultiplier × 25)
func CalcSearchScore(char *characters.Character) float64
```

Read `calcSneakScore` in `go.go` and the score calculations in
`skill.search.go` and `skill.forage.go` to extract the exact formulas.

- [ ] **Step 2: Update references**

Update `go.go` to use `actions.CalcSneakScore` instead of the local
`calcSneakScore`. Update sneak.go and search.go if they have inline
score calculations.

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "refactor: extract shared skill score helpers"
```

---

### Task 3: Mob Craft/Forage Skill Progression

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go` (or wherever crafting completion fires)
- Possibly: `internal/mobcommands/` craft-related files

Ensure that when mobs craft or forage (via existing systems like
crafter NPCs), the appropriate skill progression fires.

- [ ] **Step 1: Audit crafting completion**

Search for where crafting completes (CraftingState → finished) in the
round tick hooks. Does it fire OnSkillUse for the crafting skill? Does
it handle both users and mobs?

- [ ] **Step 2: Audit mob crafter system**

Read `internal/mobcommands/alchemy.go` — this is the mob crafting
command. Does it fire skill progression?

- [ ] **Step 3: Add missing progression**

Add `OnSkillUse` calls to mob crafting completion paths if missing.

- [ ] **Step 4: Verify build + commit**

```bash
git commit -m "feat: mob craft/forage skill progression"
```

---

### Task 4: Registry Audit Update

**Files:**
- Modify: `internal/actions/divergences.go`

Update the intentional divergence allowlists based on substages 1-5.
Some commands that were user-only now have shared actions. Review and
update both allowlists.

- [ ] **Step 1: Review current allowlists against actual state**

- [ ] **Step 2: Update divergences.go**

- [ ] **Step 3: Verify build + commit**

```bash
git commit -m "chore: update command parity allowlists for substages 1-5"
```

---

### Task 5: Tests

**Files:**
- Create: `internal/actions/sneak_test.go`

- [ ] **Step 1: Write sneak tests**

1. `TestCalcSneakScore_Baseline` — Dex 100, skill 0 → baseline score
2. `TestCalcSneakScore_HighSkill` — higher skill → higher score
3. `TestCalcSearchScore_Baseline` — Per 100, skill 0 → baseline
4. `TestExecuteSneak_AlreadyHidden` → AlreadyHidden=true
5. `TestExecuteSneak_InCombat` → InCombat=true

- [ ] **Step 2: Run tests + commit**

```bash
git commit -m "test: sneak + skill score helper tests"
```

---

### Task 6: Final Verification

- [ ] **Step 1: Full build + tests**

- [ ] **Step 2: Manual smoke test**

Test:
- Player sneak works as before
- Mob sneak now sometimes fails (check a low-stat mob)
- Hidden mob detection on room entry still works
- Surprise attacks still work
- Crafting progression fires for crafter NPCs

**REMINDER:** Watch melee skill progression speed — the 0.20
multiplier may be too fast now that auto-attack progression works.

- [ ] **Step 3: Commit any fixups**
