# Code Cleanup 1.2a: Combat + Spell God-Function Refactor — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Decompose 3 oversized functions on combat's hottest paths —
`handlePlayerVsMob` (286 lines) and `handleMobVsPlayer` (236 lines) in
`internal/hooks/NewRound_DoCombat_helpers.go`, plus `applyMobEffect`
(246 lines) in `internal/hooks/spell_resolution.go` — into focused
helpers. Combat helpers move to a new file
`internal/hooks/NewRound_DoCombat_resolution.go`; spell case helpers stay
inline in `spell_resolution.go`.

**Architecture:** Combat batch first (commits 1–4), spell batch second
(commits 5–7), overview flip last (commit 8). Cross-parent shared helpers
(damage-bonus pipeline, wait-round short-circuit) extracted before the
single-parent decompositions. `handleMobVsPlayer` refactored before
`handlePlayerVsMob` so shared helpers are discovered on the smaller
parent first. One commit per extraction step (independently revertable).

**Tech Stack:** Go 1.25. No new dependencies. Verification via existing
`TestApplyMobEffect_*` tests (spell batch) + manual smoke + 5-min
test-mud AI run (combat batch — the PvM/MvP handlers have no unit tests).

**Spec:** `docs/superpowers/specs/2026-04-17-code-cleanup-1.2a-combat-spell-refactor-design.md`

**Branch:** `feature/stage-1.2a-combat-spell-refactor` off `development`.

---

## Task 0: Create feature branch

**Files:** none.

- [ ] **Step 1: Verify you're on `development` and clean**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git status
git branch --show-current
```

Expected: `development`. Working tree should be clean EXCEPT for the
following known unrelated working-tree noise:

- `.claude/settings.local.json` — dirty
- `internal/usercommands/_datafiles/feedback/bugs.txt`, `suggestions.txt` — dirty
- `"Screenshot 2026-04-17 084513.png"` — untracked

These are **out of scope** for 1.2a. Do NOT stage or commit them at any
point in this plan. If `git status` shows anything else dirty,
investigate before proceeding.

- [ ] **Step 2: Create feature branch**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout -b feature/stage-1.2a-combat-spell-refactor
```

Expected: `Switched to a new branch 'feature/stage-1.2a-combat-spell-refactor'`.

- [ ] **Step 3: Baseline verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
```

Expected: clean build, clean vet, all `internal/hooks` tests pass
(including the 7 `TestApplyMobEffect_*` tests that will serve as our
spell-batch characterization net).

---

## Scope Policy: Bug Fixes AND PvM/MvP Parity Fixes

The spec's scope-creep policy is expanded for 1.2a to cover the PvM/MvP
quadrant-divergence problem explicitly. Design intent is that mobs and
characters behave identically in combat — divergences that aren't
behavior-tree / dialogue / AI-related are parity bugs, not features.

**During any task in this plan:**

1. **Clear bugs** (nil deref, dropped error, obvious typo) → preceding
   `fix:` commit with a one-line summary of what was broken. Refactor
   commit that follows is then semantics-preserving.
2. **Quick parity fixes** (PvM has something MvP doesn't or vice versa,
   and adding the missing half is a small mechanical change — message
   emission, a missing guard clause, a skipped progression call) →
   preceding `fix(combat): parity - <description>` commit. Fix the
   gap, then do the refactor.
   - "Quick" = under ~30 lines of new code, no new dependency, no
     new config knob, no gameplay rebalance.
   - The subagent must run `go build && go vet && go test` after the
     parity fix before the refactor commit.
3. **Non-quick parity gaps** (needs new config, new gameplay decision,
   or the fix ripples through multiple systems) → **DO NOT FIX HERE**.
   Instead, append one line to
   `C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_pvm_mvp_parity_gaps.md`
   in the form:
   `- <task N>: <function pair> — <gap description> — <why not a quick fix>`
   and continue the refactor preserving the existing divergent behavior.
4. **Ambiguous cases** (can't tell if parity is a bug or intentional
   design) → **PAUSE and ask the user**. Do not guess.

The goal: every quadrant-divergence encountered during extraction
either gets fixed in this branch or gets logged for a future dedicated
pass. Nothing silently survives.

Known gaps from plan research (start here, add to the memory file as
more are discovered):

| Location | Gap | Likely disposition |
|----------|-----|-------------------|
| lifesteal message | PvM emits "feeds on the blow" user message; MvP has no symmetric mob-source message | Quick parity fix — add symmetric MvP message |
| Minor Shield | Check whether only MvP applies Minor Shield or whether PvM defender (a mob) should also benefit from its shield-stack | Needs investigation — likely quick fix OR log to memory if design-intent |
| reciprocal-aggro guard | PvM guards on `defUser.Character.Health > 0` (Shadow Realm); verify MvP has symmetric guard on `defMob` health | Quick parity fix if missing |
| crit room re-lookup | MvP re-loads room via `rooms.LoadRoom(mob.RoomId)`; PvM may or may not — verify both sides handle room-changed-during-round identically | Quick parity fix if PvM stale |
| mob-aggro-on-attack | PvM-specific `go <exit>` + `GetAngryCommand` + `attack` chase chain; does MvP have equivalent when a player flees mid-attack? | Likely log to memory — chase logic is meaningful gameplay |
| applyMobEffect buff aggro | Conditionally gated on `Harm*` prefix only — verify `applyPlayerEffect` counterpart uses same gate | Likely log to memory (applyPlayerEffect is out of 1.2a scope) |

---

## Task 1: Extract combat damage-bonus pipeline helpers

Pull the four consecutive damage-bonus stages (Minor Shield reduction,
Conviction Surge, Adrenaline Surge, return damage, lifesteal) into
helpers. Produced by reading `handleMobVsPlayer` first since its version
is cleanest, but applied to both parents in this commit. New file
`NewRound_DoCombat_resolution.go` is created here.

**Files:**
- Create: `internal/hooks/NewRound_DoCombat_resolution.go`
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` — `handlePlayerVsMob` and `handleMobVsPlayer` damage-bonus sections replaced with helper calls.

### Setup

- [ ] **Step 1: Confirm parent function line ranges**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func handlePlayerVsMob\|^func handleMobVsPlayer\|^func handleMobVsMob" internal/hooks/NewRound_DoCombat_helpers.go
```

Expected: `handlePlayerVsMob` at ~1102, `handleMobVsPlayer` at ~1388,
`handleMobVsMob` at ~1624. Note the ranges — if they've drifted, update
the line references in subsequent steps.

- [ ] **Step 2: Confirm the damage-bonus block shapes in both parents**

The damage-bonus pipeline in `handleMobVsPlayer` (lines ~1460–1527):
1. Minor Shield reduction on `defUser.Character.HasCondition(ConditionShield)` (~1460–1466).
2. Conviction Surge on `mob.Character.HasBuffFlag(buffs.DamageBonus)` (~1469–1476).
3. Adrenaline Surge via `mutations.IsAdrenalSurgeActive` (~1479–1490).
4. Return damage via `defUser.Character.StatMod("return_damage")` + species `ReturnDamage`, emits "recoils from striking you!" to defender, "recoils from striking %s" to mob room (~1494–1513).
5. Lifesteal via `mob.Character.StatMod("lifesteal_pct")` — mob heals, NO message to anyone (~1516–1527).

The damage-bonus pipeline in `handlePlayerVsMob` (lines ~1203–1265) is
structurally identical EXCEPT:
- No Minor Shield block (defender is a mob; no `ConditionShield` check).
- Return damage messages flip: `sendVisualRoomText(uRoom, ...)` "recoils from striking %s" + `user.SendText(...)` "You recoil from striking %s!".
- Lifesteal emits a "Your weapon feeds on the blow!" message to the user (line ~1260–1262).

**Gotcha:** The PvM lifesteal variant has a user-visible message; the
MvP variant does not. The helper needs a role parameter or two sibling
helpers. The spec (risk register) explicitly allows sibling helpers if
unification hides semantics — take that route here.

**Gotcha:** Minor Shield reduction is MvP-only (only defending players
have `ConditionShield`). Keep it outside the shared helper OR gate it
with a defender-type check. Cleanest: keep the Minor Shield block
inline in `handleMobVsPlayer`, wrap only stages 2–5 in the shared
helper.

### Extract

- [ ] **Step 3: Create `NewRound_DoCombat_resolution.go` with package + imports**

Create `internal/hooks/NewRound_DoCombat_resolution.go` with:

```go
package hooks

import (
    "fmt"
    "math"

    "github.com/GoMudEngine/GoMud/internal/buffs"
    "github.com/GoMudEngine/GoMud/internal/characters"
    "github.com/GoMudEngine/GoMud/internal/combat"
    "github.com/GoMudEngine/GoMud/internal/mobs"
    "github.com/GoMudEngine/GoMud/internal/mutations"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/species"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// File: NewRound_DoCombat_resolution.go
//
// Phase helpers for the per-combatant round handlers (handlePlayerVsMob
// and handleMobVsPlayer). Extracted during 1.2a god-function refactor.
// Same package as NewRound_DoCombat_helpers.go and combat_shared_helpers.go
// — symbols resolve without imports. If a helper is only called by one
// parent, keep it private to this file (lowercase, no export).
```

Do NOT add helpers yet in this step; just the skeleton + package
comment. Build check:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
```

Expected: builds clean (file is valid even with no helpers).

- [ ] **Step 4: Add the MvP damage-bonus helper**

In `NewRound_DoCombat_resolution.go`, add:

```go
// applyCombatDamageBonuses_MvP applies the Conviction Surge / Adrenaline
// Surge / return-damage / lifesteal pipeline for a mob attacker against a
// player defender. Mutates roundResult.DamageToTarget, attacker.Health,
// defender.Health. Emits return-damage messages to defUser + mob room.
//
// NOTE: Minor Shield reduction is NOT inside this helper — it's checked
// on ConditionShield which only player defenders have, and lives inline
// in handleMobVsPlayer before this helper is called.
func applyCombatDamageBonuses_MvP(
    roundResult *combat.AttackResult,
    mob *mobs.Mob,
    defUser *users.UserRecord,
    mobRoom *rooms.Room,
) {
    // body: stages 2–5 from handleMobVsPlayer lines ~1469–1527, verbatim,
    // with roundResult accessed as *roundResult.
}
```

Shape: 4 sequential `if roundResult.Hit && roundResult.DamageToTarget > 0 { ... }` blocks, one per stage. Preserve mutation order: bonus damage is added BEFORE return damage is computed (return damage uses the post-bonus `DamageToTarget`). This ordering is load-bearing.

- [ ] **Step 5: Add the PvM damage-bonus helper**

```go
// applyCombatDamageBonuses_PvM applies the Conviction Surge / Adrenaline
// Surge / return-damage / lifesteal pipeline for a player attacker
// against a mob defender. Differs from the MvP variant in two messages:
// return damage emits to uRoom + user, lifesteal emits a "feeds on the
// blow" message to the user.
func applyCombatDamageBonuses_PvM(
    roundResult *combat.AttackResult,
    user *users.UserRecord,
    defMob *mobs.Mob,
    uRoom *rooms.Room,
    defRoom *rooms.Room,
) {
    // body: stages from handlePlayerVsMob lines ~1203–1265, verbatim,
    // with roundResult accessed as *roundResult.
}
```

Shape: 4 sequential blocks mirroring MvP but with PvM-specific
messages. Note `mobDisplayName(defMob, defRoom, user.UserId)` is used
for the return-damage message.

- [ ] **Step 6: Replace MvP inline block with helper call**

In `handleMobVsPlayer`, lines ~1469–1527, replace the four damage-bonus
blocks (Conviction Surge through lifesteal) with:

```go
applyCombatDamageBonuses_MvP(&roundResult, mob, defUser, mobRoom)
```

Keep the Minor Shield block (~1460–1466) as-is, inline, immediately
before the helper call.

- [ ] **Step 7: Replace PvM inline block with helper call**

In `handlePlayerVsMob`, lines ~1203–1265, replace the four damage-bonus
blocks with:

```go
applyCombatDamageBonuses_PvM(&roundResult, user, defMob, uRoom, defRoom)
```

### Verify

- [ ] **Step 8: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
```

Expected: clean. `handleMobVsMob` still has its own inline copy — do
NOT touch it (out of scope per spec).

### Commit

- [ ] **Step 9: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_DoCombat_resolution.go internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract combat damage-bonus pipeline helpers

Pull the Conviction Surge / Adrenaline Surge / return-damage / lifesteal
stages out of handlePlayerVsMob and handleMobVsPlayer into sibling
helpers applyCombatDamageBonuses_PvM / _MvP in the new file
NewRound_DoCombat_resolution.go. Minor Shield stays inline in the MvP
parent because ConditionShield is player-defender-only.

PvM and MvP kept as siblings (not unified) because return-damage and
lifesteal emit role-specific messages.

Zero behavior change. handleMobVsMob left with its own inline copy
(out of scope per 1.2a spec).

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Expected: commit succeeds. Verify with `git log --oneline -1`.

---

## Task 2: Extract combat wait-round short-circuit helper

The `RoundsWaiting > 0` block appears symmetrically in `handlePlayerVsMob`
(~lines 1164–1184) and `handleMobVsPlayer` (~lines 1430–1450) with
side-specific `combat.SourceTarget` tags. Extract one helper with role
params replacing both.

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_resolution.go` — add `handleCombatWaitRound`.
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` — replace both wait-round blocks.

### Setup

- [ ] **Step 1: Verify the two wait-round blocks match structurally**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "RoundsWaiting > 0" internal/hooks/NewRound_DoCombat_helpers.go
```

Expected: 3 matches — `handlePlayerVsMob` (~1164), `handleMobVsPlayer`
(~1430), `handleMobVsMob` (~1647). We only consolidate the first two.

**Gotcha:** `handlePlayerVsMob` emits `MessagesToSource` (attacker-is-user);
`handleMobVsPlayer` emits `MessagesToTarget` (defender-is-user). Same
semantic, different message slot. The helper must dispatch to the user
regardless of role — branch on role param.

**Gotcha:** `handlePlayerVsMob` uses `combat.User, combat.Mob` as the
`GetWaitMessages` role args; `handleMobVsPlayer` uses `combat.Mob,
combat.User`. Helper takes both role tags and a `viewerUserId` param.

**Gotcha:** Both blocks contain a `return` after emission — the helper
signals with a bool return so the parent can `return` itself. Do NOT
try to move the `return` into the helper; the parents do more than just
wait-round logic.

### Extract

- [ ] **Step 2: Add `handleCombatWaitRound` helper**

In `NewRound_DoCombat_resolution.go`, add:

```go
// handleCombatWaitRound handles the RoundsWaiting > 0 short-circuit.
// Returns true if wait-round messages were emitted (caller should
// return immediately). Returns false if not waiting (caller continues).
//
// roleSource/roleTarget are the combat.SourceTarget constants the
// parent passes through to combat.GetWaitMessages (e.g. combat.User,
// combat.Mob for a player attacker).
//
// attackerUser/defenderUser may be nil — if nil, messages of that
// slot are skipped. viewerUserId is the dark-room fallback viewer.
func handleCombatWaitRound(
    attackerChar *characters.Character,
    defenderChar *characters.Character,
    roleSource combat.SourceTarget,
    roleTarget combat.SourceTarget,
    attackerUser *users.UserRecord,
    defenderUser *users.UserRecord,
    attackerRoom *rooms.Room,
    defenderRoom *rooms.Room,
    viewerUserId int,
) bool {
    // body:
    // 1. if attackerChar.Aggro.RoundsWaiting <= 0, return false
    // 2. mudlog.Debug like the originals
    // 3. attackerChar.Aggro.RoundsWaiting--
    // 4. roundResult := combat.GetWaitMessages(items.Wait, attackerChar, defenderChar, roleSource, roleTarget)
    // 5. for _, msg := range roundResult.MessagesToSource { if attackerUser != nil { attackerUser.SendText(msg) } }
    // 6. for _, msg := range roundResult.MessagesToTarget { if defenderUser != nil { defenderUser.SendText(msg) } }
    // 7. for MessagesToSourceRoom → sendVisualRoomText(attackerRoom, msg, viewerUserId)
    // 8. for MessagesToTargetRoom → sendVisualRoomText(defenderRoom, msg, viewerUserId)
    // 9. sendDarkRoomCombatFallback(attackerRoom, viewerUserId)
    // 10. if defenderRoom != attackerRoom { sendDarkRoomCombatFallback(defenderRoom, viewerUserId) }
    // 11. return true
}
```

- [ ] **Step 3: Replace MvP wait-round block**

In `handleMobVsPlayer` (~lines 1430–1450), replace the `if mob.Character.Aggro.RoundsWaiting > 0 { ... return }` block with:

```go
if handleCombatWaitRound(
    &mob.Character, defUser.Character,
    combat.Mob, combat.User,
    nil, defUser,
    mobRoom, defRoom,
    defUser.UserId,
) {
    return
}
```

- [ ] **Step 4: Replace PvM wait-round block**

In `handlePlayerVsMob` (~lines 1164–1184), replace the analogous block with:

```go
if handleCombatWaitRound(
    user.Character, &defMob.Character,
    combat.User, combat.Mob,
    user, nil,
    uRoom, defRoom,
    user.UserId,
) {
    return
}
```

### Verify

- [ ] **Step 5: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
```

Expected: clean.

### Commit

- [ ] **Step 6: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_DoCombat_resolution.go internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract combat wait-round short-circuit helper

The RoundsWaiting > 0 block appeared symmetrically in handlePlayerVsMob
and handleMobVsPlayer. Extract handleCombatWaitRound into
NewRound_DoCombat_resolution.go, taking role tags + nullable user
slots to dispatch source/target messages correctly for either role.

handleMobVsMob's copy left in place (out of scope per spec).

Zero behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Extract `handleMobVsPlayer` phase helpers

Break remaining `handleMobVsPlayer` body into target-validation,
crit-and-messaging, progression, and resolution helpers. Parent shrinks
to ~60 lines.

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_resolution.go` — add 4 helpers.
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` — `handleMobVsPlayer` body replaced.

### Setup

- [ ] **Step 1: Map remaining phases in `handleMobVsPlayer`**

After Tasks 1 and 2, `handleMobVsPlayer` still contains these phases
(line numbers approximate, post-Task-1/2):

1. Target resolution + early-exit: defUser lookup, room match, defRoom load, CancelBuffsWithFlag, downed grace, hidden check, affectedPlayerIds append, reciprocal aggro, party auto-attack, grapple progression, target switch, weapon pickup (lines ~1388–1428).
2. Wait-round short-circuit (now a single helper call from Task 2).
3. Attack roll + moon mod + Minor Shield + damage bonuses (Task-1 helper call) (~1452–1527).
4. Combat analytics + crit effects + charmed mob assist (~1529–1548).
5. Darkness replacement + buff apply + message emission (~1550–1572).
6. Concentration + offhand break + crit-received (~1574–1580).
7. Mob progression (stat gain messaging + weapon skill + unarmed + defender progression) (~1582–1609).
8. Round resolution (health ≤ 0 → EndAggro / RetargetOrEnd / SetAggro) (~1611–1620).

Phase 1 is already heavily delegated to existing helpers
(`handleMobDownedGrace`, `handlePartyAutoAttack`,
`processGrappleProgression`, `handleMobTargetSwitch`,
`handleMobWeaponPickup`). Keep the orchestration inline in the parent;
don't wrap it further.

- [ ] **Step 2: Confirm helper signatures for referenced code**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func applyCritEffects\|^func handleCharmedMobAssist\|^func handlePlayerConcentrationBreak\|^func handleOffhandBreakUserDef\|^func mobDisplayName\|^func processDefenderProgression\|^func RetargetOrEnd" internal/hooks/*.go
```

Verify signatures so helper extraction passes the right types. All of
these are already in-package.

### Extract

- [ ] **Step 3: Add `handleMvPCritAndMessaging` helper**

In `NewRound_DoCombat_resolution.go`:

```go
// handleMvPCritAndMessaging runs the crit-effect + charmed-mob-assist +
// darkness-replacement + buff + message-emission block for mob-vs-player.
// Must be called AFTER combat.RecordAttack and AFTER the damage-bonus
// helper; mutates defUser/mob buffs and emits all per-round messages.
func handleMvPCritAndMessaging(
    roundResult *combat.AttackResult,
    mob *mobs.Mob,
    defUser *users.UserRecord,
    mobRoom *rooms.Room,
    defRoom *rooms.Room,
) {
    // body: phases 4 + 5 from the map above, verbatim:
    //
    // critResult := applyCritEffects(&mob.Character, defUser.Character, *roundResult, mobRoom)
    // if critResult.DefenderMsg != "" { defUser.SendText(critResult.DefenderMsg) }
    // if critResult.RoomMsg != ""     { mobRoom.SendText(critResult.RoomMsg, defUser.UserId) }
    //
    // room := rooms.LoadRoom(mob.Character.RoomId) // NOTE: fresh lookup, not mobRoom reuse
    // handleCharmedMobAssist(room, defUser.UserId, fmt.Sprintf("#%d", mob.InstanceId))
    //
    // tgtCanSee := canSeeInRoom(defUser.Character, defRoom)
    // replaceDarknessMessages(roundResult, true, tgtCanSee)
    //
    // for _, buffId := range roundResult.BuffSource { mob.AddBuff(buffId, "combat") }
    // for _, buffId := range roundResult.BuffTarget { defUser.AddBuff(buffId, "combat") }
    // for _, msg := range roundResult.MessagesToTarget      { defUser.SendText(msg) }
    // for _, msg := range roundResult.MessagesToSourceRoom  { sendVisualRoomText(mobRoom, msg, defUser.UserId) }
    // for _, msg := range roundResult.MessagesToTargetRoom  { sendVisualRoomText(defRoom, msg, defUser.UserId) }
    // sendDarkRoomCombatFallback(mobRoom, defUser.UserId)
    // if defRoom != mobRoom { sendDarkRoomCombatFallback(defRoom, defUser.UserId) }
}
```

**Gotcha:** The `rooms.LoadRoom(mob.Character.RoomId)` re-lookup at
line ~1547 exists because the mob might have moved. Preserve the fresh
lookup — don't substitute `mobRoom`.

- [ ] **Step 4: Add `handleMvPProgression` helper**

```go
// handleMvPProgression emits mob stat-gain messages, tracks mob weapon
// skill use, and runs defender progression + crit-received hook for a
// player defender.
func handleMvPProgression(
    roundResult *combat.AttackResult,
    mob *mobs.Mob,
    defUser *users.UserRecord,
    mobRoom *rooms.Room,
) {
    // body: phases 6 + 7 from the map, verbatim:
    //
    // handlePlayerConcentrationBreak(defUser, *roundResult, defUser-defRoom)
    //   NOTE: defRoom is used in handlePlayerConcentrationBreak — needs
    //   param. See signature adjustment below.
    // handleOffhandBreakUserDef(*roundResult, defUser, defUser-defRoom)
    // if roundResult.Hit && roundResult.Crit { defUser.Character.OnCritReceived("physical", defUser.UserId) }
    // statMobName := mobDisplayName(mob, mobRoom, 0)
    // if gained := mob.Character.OnStatUse("strength", 0); gained { emit MobStatGainMessages["strength"] to mobRoom }
    // ... same for dexterity ...
    // for wh range roundResult.WeaponHits { ... OnSkillUse / OnCriticalSuccess / OnCriticalFailure }
    // if no WeaponHits && Hit { OnSkillUse(UnarmedCombat, 0) }
    // processDefenderProgression(defUser.Character, defUser.UserId, *roundResult)
}
```

Adjust signature to include `defRoom` since
`handlePlayerConcentrationBreak` and `handleOffhandBreakUserDef` both
need it:

```go
func handleMvPProgression(
    roundResult *combat.AttackResult,
    mob *mobs.Mob,
    defUser *users.UserRecord,
    mobRoom *rooms.Room,
    defRoom *rooms.Room,
)
```

- [ ] **Step 5: Add `handleMvPRoundResolution` helper**

```go
// handleMvPRoundResolution closes out the round: sets aggro if both
// combatants live, otherwise calls EndAggro / RetargetOrEnd as
// appropriate. Mirrors handlePvMRoundResolution but for a mob attacker.
func handleMvPRoundResolution(mob *mobs.Mob, defUser *users.UserRecord, mobRoom *rooms.Room) {
    // body: phase 8, verbatim:
    //
    // if mob.Character.Health <= 0 || defUser.Character.Health <= 0 {
    //     defUser.Character.EndAggro()
    //     if mob.Character.Health > 0 {
    //         RetargetOrEnd(&mob.Character, mobRoom, 0, mob.InstanceId)
    //     } else {
    //         mob.Character.EndAggro()
    //     }
    // } else {
    //     mob.Character.SetAggro(defUser.UserId, 0, characters.DefaultAttack)
    // }
}
```

- [ ] **Step 6: Rewrite `handleMobVsPlayer` body to orchestrate helpers**

Replace the post-Task-2 body with (sketch):

```go
func handleMobVsPlayer(mob *mobs.Mob, mobRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int) {
    defUser := users.GetByUserId(mob.Character.Aggro.UserId)
    if defUser == nil || mob.Character.RoomId != defUser.Character.RoomId {
        mob.Character.EndAggro()
        return
    }
    defRoom := rooms.LoadRoom(defUser.Character.RoomId)
    if defRoom == nil {
        mob.Character.EndAggro()
        return
    }
    defUser.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
    if handleMobDownedGrace(mob, defUser, defRoom, affectedPlayerIds) {
        return
    }
    if defUser.Character.HasBuffFlag(buffs.Hidden) {
        return
    }
    *affectedPlayerIds = append(*affectedPlayerIds, mob.Character.Aggro.UserId)
    if defUser.Character.Health > 0 && defUser.Character.Aggro == nil {
        defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
    }
    handlePartyAutoAttack(mob, defUser)
    processGrappleProgression(&mob.Character, defUser.Character, mob.Character.Name, defUser.Character.Name, mobRoom, 0, defUser.UserId)
    if handleMobTargetSwitch(mob, mobRoom) {
        return
    }
    handleMobWeaponPickup(mob)

    if handleCombatWaitRound(&mob.Character, defUser.Character, combat.Mob, combat.User, nil, defUser, mobRoom, defRoom, defUser.UserId) {
        return
    }

    restore := applyMoonMods(&mob.Character, moonMod)
    roundResult := combat.AttackMobVsPlayer(mob, defUser)
    restore()

    // Stage 11.4: Minor Shield — MvP-specific, inline.
    if roundResult.Hit && defUser.Character.HasCondition(characters.ConditionShield) {
        reduction := int(defUser.Character.GetConditionMagnitude(characters.ConditionShield)) / 2
        if roundResult.DamageToTarget > reduction+1 {
            roundResult.DamageToTarget -= reduction
            roundResult.DamageToTargetReduction += reduction
        }
    }

    applyCombatDamageBonuses_MvP(&roundResult, mob, defUser, mobRoom)

    // Analytics
    atkType := "unarmed"
    if mob.Character.Equipment.Weapon.ItemId > 0 {
        atkType = "weapon"
    }
    combat.RecordAttack(roundResult, combat.Mob, combat.User, atkType, &mob.Character, defUser.Character, evt.RoundNumber)

    handleMvPCritAndMessaging(&roundResult, mob, defUser, mobRoom, defRoom)
    handleMvPProgression(&roundResult, mob, defUser, mobRoom, defRoom)
    handleMvPRoundResolution(mob, defUser, mobRoom)
}
```

**Gotcha:** The reciprocal-aggro guard `defUser.Character.Health > 0`
must stay — Shadow Realm regression otherwise. The spec risk register
does not flag this but it's easy to miss during extraction.

### Verify

- [ ] **Step 7: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/...
```

Expected: clean. Helpers should each be under 80 lines; parent under 60.

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_DoCombat_resolution.go internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract handleMobVsPlayer phase helpers

Break the remaining handleMobVsPlayer body into three helpers in
NewRound_DoCombat_resolution.go:
- handleMvPCritAndMessaging: crit effects + charmed mob assist +
  darkness replacement + buff + per-round messages
- handleMvPProgression: concentration, offhand break, mob stat-gain
  messaging, weapon/unarmed skill tracking, defender progression
- handleMvPRoundResolution: health-check endgame (EndAggro /
  RetargetOrEnd / SetAggro)

Parent handleMobVsPlayer shrinks from 236 → ~60 lines. Minor Shield
stays inline because it's MvP-specific (ConditionShield only on player
defenders).

Zero behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Extract `handlePlayerVsMob` phase helpers

Apply the same decomposition to `handlePlayerVsMob`, reusing helpers from
Tasks 1–3 where symmetric. Parent shrinks from 286 → ~80 lines. This is
the largest extraction and the last in the combat batch; it closes with
a full combat smoke test.

**Files:**
- Modify: `internal/hooks/NewRound_DoCombat_resolution.go` — add 4 PvM-specific helpers.
- Modify: `internal/hooks/NewRound_DoCombat_helpers.go` — `handlePlayerVsMob` body replaced.

### Setup

- [ ] **Step 1: Map remaining phases in `handlePlayerVsMob`**

After Tasks 1 and 2, `handlePlayerVsMob` phases (line numbers
approximate, post-Task-1/2):

1. Target resolution with PvM-specific multi-room exit check + `combat_start` emission + CancelBuffsWithFlag + Health < 1 gate (~1102–1162).
2. Wait-round short-circuit (Task 2 helper).
3. Hidden-target short-circuit with "You can't seem to find your target" message (~1186–1189).
4. affectedPlayerIds append + grapple progression (~1191–1193).
5. Attack + moon mod + damage bonuses (Task 1 helper) (~1195–1265).
6. RecordAttack + applyCritEffects + dispatchCritEffectsPvM (~1267–1275).
7. Darkness replacement + buff apply + message emission (~1277–1299).
8. Mob concentration break + behaviortree mob_hurt (~1301–1317).
9. Player progression (stat use + weapon skill + unarmed + defender progression) (~1319–1338).
10. Hostility group marking + mob aggro-on-attack (go <exit> / GetAngryCommand / attack) + companion owner assist (~1340–1365).
11. Round resolution with "You turn your attention to…" retarget message (~1367–1382).

**Gotcha:** Phase 1's `combat_start` emission for first-round defender
mobs sits intentionally BEFORE the attack roll (inline comment flags
this — see spec risk register). Helper extraction must preserve that
ordering. Flagged again: see Step 3 below.

**Gotcha:** Phase 1's target resolution returns early with
`user.Character.EndAggro()` in THREE places (uRoom nil, !targetFound,
defRoom nil, Health < 1). All four must stay intact.

**Gotcha:** Phase 10's "mob aggro on attack" logic includes a
room-walk to `go <exit>` back to the user's room if the mob is now in
a different room. This multi-step side effect must survive extraction
byte-for-byte — it's a real test-mud-observable behavior.

**Gotcha:** The trailing `_ = roomId` on line ~1384 is a compile
silencer for an unused variable declared at line ~1104. The refactor
can drop the `roomId` declaration entirely if nothing else uses it;
verify with `go vet`.

- [ ] **Step 2: Verify helper signatures used**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func applyCritEffects\|^func dispatchCritEffectsPvM\|^func handleCompanionOwnerAssist\|^func processGrappleProgression\|^func checkConcentrationBreak\|^func recordConcentrationFailure\|^func RetargetOrEnd" internal/hooks/*.go internal/characters/*.go
```

### Extract

- [ ] **Step 3: Add `resolvePlayerVsMobTarget` helper**

In `NewRound_DoCombat_resolution.go`:

```go
// resolvePlayerVsMobTarget validates the user's current combat target
// is reachable, loads its room, emits combat_start for first-round
// defender mobs (BEFORE the attack roll so round-1 dies still fire
// reactive AI), cancels combat-cancel buffs, and guards Health < 1.
//
// Returns (defMob, defRoom, ok). On ok == false the helper has already
// called user.Character.EndAggro() and/or sent the "can't be found"
// message; caller should return immediately.
//
// NOTE: combat_start emission timing is intentional — keep it in this
// helper at the current position, not moved later.
func resolvePlayerVsMobTarget(
    user *users.UserRecord,
    uRoom *rooms.Room,
    evt events.NewRound,
) (defMob *mobs.Mob, defRoom *rooms.Room, ok bool) {
    // body: phase 1, verbatim:
    //
    // defMob = mobs.GetInstance(user.Character.Aggro.MobInstanceId)
    // targetFound logic (nil / room mismatch with optional exit check)
    // if !targetFound { user.SendText("Your target can't be found."); user.Character.EndAggro(); return nil, nil, false }
    // defRoom = rooms.LoadRoom(defMob.Character.RoomId)
    // if defRoom == nil { user.Character.EndAggro(); return nil, nil, false }
    //
    // if defMob.CombatMemory == nil && defMob.Character.Aggro != nil {
    //     mudlog.Debug(...)
    //     defMob.CombatMemory = mobai.SetMemory(...)
    //     events.AddToQueue(events.MobAISignal{...})
    // }
    //
    // defMob.Character.CancelBuffsWithFlag(buffs.CancelIfCombat)
    // if defMob.Character.Health < 1 { user.Character.EndAggro(); return nil, nil, false }
    //
    // return defMob, defRoom, true
}
```

Note: this helper needs `mobai` + `events` imports. Add them to
`NewRound_DoCombat_resolution.go`.

- [ ] **Step 4: Add `handlePvMCritAndMessaging` helper**

```go
// handlePvMCritAndMessaging runs analytics + crit effects + darkness
// replacement + buff + per-round message emission for player-vs-mob.
// Mirrors handleMvPCritAndMessaging but for a player attacker.
func handlePvMCritAndMessaging(
    roundResult *combat.AttackResult,
    user *users.UserRecord,
    defMob *mobs.Mob,
    uRoom *rooms.Room,
    defRoom *rooms.Room,
    roundNumber uint64,
) {
    // body: phases 6 + 7 combined:
    //
    // atkType := "unarmed"; if user.Character.Equipment.Weapon.ItemId > 0 { atkType = "weapon" }
    // combat.RecordAttack(*roundResult, combat.User, combat.Mob, atkType, user.Character, &defMob.Character, roundNumber)
    //
    // critResult := applyCritEffects(user.Character, &defMob.Character, *roundResult, uRoom)
    // dispatchCritEffectsPvM(critResult, user, defMob, uRoom)
    //
    // srcCanSee := canSeeInRoom(user.Character, uRoom)
    // replaceDarknessMessages(roundResult, srcCanSee, true)
    //
    // for _, buffId := range roundResult.BuffSource { user.AddBuff(buffId, "combat") }
    // for _, buffId := range roundResult.BuffTarget { defMob.AddBuff(buffId, "combat") }
    // for _, msg := range roundResult.MessagesToSource     { user.SendText(msg) }
    // for _, msg := range roundResult.MessagesToSourceRoom { sendVisualRoomText(uRoom, msg, user.UserId) }
    // for _, msg := range roundResult.MessagesToTargetRoom { sendVisualRoomText(defRoom, msg, user.UserId) }
    // sendDarkRoomCombatFallback(uRoom, user.UserId)
    // if defRoom != uRoom { sendDarkRoomCombatFallback(defRoom, user.UserId) }
}
```

- [ ] **Step 5: Add `handlePvMProgressionAndAggro` helper**

```go
// handlePvMProgressionAndAggro runs mob-concentration break, mob_hurt
// behavior tree, player attacker progression, defender progression,
// hostility group marking, mob-aggro-on-attack (with exit-walk to
// user's room if mob moved), and companion owner assist.
func handlePvMProgressionAndAggro(
    roundResult *combat.AttackResult,
    user *users.UserRecord,
    defMob *mobs.Mob,
    uRoom *rooms.Room,
    cfg *configs.Config,
) {
    // body: phases 8, 9, 10:
    //
    // if checkConcentrationBreak(&defMob.Character, roundResult.DamageToTarget) {
    //     recordConcentrationFailure(combat.Mob, combat.User, &defMob.Character, castingTargetChar(defMob.Character.CastingState))
    //     defMob.Character.CastingState = nil
    //     uRoom.SendText(fmt.Sprintf("<ansi fg=\"mobname\">%s</ansi>'s concentration breaks.", defMob.Character.Name))
    // }
    // if roundResult.Hit {
    //     behaviortree.TryMobBehavior(defMob.InstanceId, behaviortree.EventContext{EventType: "mob_hurt", UserId: user.UserId, RoomId: defMob.Character.RoomId})
    // }
    //
    // user.Character.OnStatUse("strength", user.UserId)
    // user.Character.OnStatUse("dexterity", user.UserId)
    // for wh range roundResult.WeaponHits { ... OnSkillUse / OnCriticalSuccess / OnCriticalFailure }
    // if no WeaponHits && Hit { OnSkillUse(UnarmedCombat, user.UserId) }
    // processDefenderProgression(&defMob.Character, 0, *roundResult)
    //
    // for _, groupName := range defMob.Groups {
    //     mobs.MakeHostile(groupName, user.UserId, cfg.Timing.MinutesToRounds(2)-user.Character.Stats.Charisma.ValueAdj)
    // }
    //
    // if defMob.Character.Aggro == nil {
    //     defMob.PreventIdle = true
    //     if user.Character.RoomId != defMob.Character.RoomId {
    //         if mobRoom := rooms.LoadRoom(defMob.Character.RoomId); mobRoom != nil {
    //             for exitName, exitInfo := range mobRoom.Exits {
    //                 if exitInfo.RoomId == user.Character.RoomId {
    //                     defMob.Command(fmt.Sprintf("go %s", exitName))
    //                     if actionStr := defMob.GetAngryCommand(); actionStr != "" { defMob.Command(actionStr) }
    //                     break
    //                 }
    //             }
    //         }
    //     }
    //     defMob.Command(fmt.Sprintf("attack @%d", user.UserId))
    // }
    // handleCompanionOwnerAssist(defMob, fmt.Sprintf("@%d", user.UserId))
}
```

- [ ] **Step 6: Add `handlePvMRoundResolution` helper**

```go
// handlePvMRoundResolution closes out the round. When the user survives
// and RetargetOrEnd picks a new target, emit the PvM-specific "You turn
// your attention to…" message to the user.
func handlePvMRoundResolution(user *users.UserRecord, defMob *mobs.Mob, uRoom *rooms.Room) {
    // body: phase 11, verbatim:
    //
    // if user.Character.Health <= 0 || defMob.Character.Health <= 0 {
    //     defMob.Character.EndAggro()
    //     if user.Character.Health > 0 {
    //         if RetargetOrEnd(user.Character, uRoom, user.UserId, 0) {
    //             if mob := mobs.GetInstance(user.Character.Aggro.MobInstanceId); mob != nil {
    //                 user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"mobname\">%s</ansi>!", mob.Character.Name))
    //             } else if newDef := users.GetByUserId(user.Character.Aggro.UserId); newDef != nil {
    //                 user.SendText(fmt.Sprintf("You turn your attention to <ansi fg=\"username\">%s</ansi>!", newDef.Character.Name))
    //             }
    //         }
    //     } else {
    //         user.Character.EndAggro()
    //     }
    // } else {
    //     user.Character.SetAggro(0, defMob.InstanceId, characters.DefaultAttack)
    // }
}
```

- [ ] **Step 7: Rewrite `handlePlayerVsMob` body**

Replace the entire post-Task-1/2 body with (sketch):

```go
func handlePlayerVsMob(user *users.UserRecord, uRoom *rooms.Room, evt events.NewRound, moonMod float64, affectedPlayerIds *[]int, affectedMobInstanceIds *[]int) {
    c := configs.GetConfig()

    *affectedMobInstanceIds = append(*affectedMobInstanceIds, user.Character.Aggro.MobInstanceId)

    defMob, defRoom, ok := resolvePlayerVsMobTarget(user, uRoom, evt)
    if !ok {
        return
    }

    if handleCombatWaitRound(user.Character, &defMob.Character, combat.User, combat.Mob, user, nil, uRoom, defRoom, user.UserId) {
        return
    }

    if defMob.Character.HasBuffFlag(buffs.Hidden) {
        user.SendText("You can't seem to find your target.")
        return
    }

    *affectedPlayerIds = append(*affectedPlayerIds, user.Character.Aggro.UserId)
    processGrappleProgression(user.Character, &defMob.Character, user.Character.Name, defMob.Character.Name, uRoom, user.UserId, 0)

    restore := applyMoonMods(user.Character, moonMod)
    roundResult := combat.AttackPlayerVsMob(user, defMob)
    restore()

    applyCombatDamageBonuses_PvM(&roundResult, user, defMob, uRoom, defRoom)

    handlePvMCritAndMessaging(&roundResult, user, defMob, uRoom, defRoom, evt.RoundNumber)
    handlePvMProgressionAndAggro(&roundResult, user, defMob, uRoom, c)
    handlePvMRoundResolution(user, defMob, uRoom)
}
```

Drop the `roomId := user.Character.RoomId` + trailing `_ = roomId`
since nothing else uses it. `go vet` will confirm.

### Verify

- [ ] **Step 8: Build + vet + test (full suite — batch boundary)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean. This is the combat-batch boundary; full `./...` test
pass required.

- [ ] **Step 9: Manual combat smoke test**

Start local server. Run:

1. **PvM happy path** — spawn a hostile mob, `attack` it with a 1H weapon, verify hit/miss/crit messages appear correctly, mob and player Health/Stamina move as expected, defeat it.
2. **MvP retaliation** — low-level mob auto-retaliates on your attack; confirm it hits back, defender messages appear, Minor Shield reduction works (use a spell-shield buff).
3. **Wait-round** — type `rest` adjacent-round to trigger a wait-round on the attacker side; verify wait messages emit once then combat resumes.
4. **2H weapon** — swap to a 2H weapon, attack, verify PvM path.
5. **Dual-wield** — equip two 1H weapons, verify both WeaponHits emit correctly.
6. **Shield** — equip a shield, take a hit, confirm offhand-break path fires if weapon durability runs out.
7. **Moon Phase** — if accessible, enable a moon mod and confirm the attack roll incorporates it (no visible change beyond stat tuning).
8. **Retarget message** — engage two mobs at once, kill one; verify "You turn your attention to…" message appears for the new target.
9. **Companion assist** — charm a mob, attack a hostile mob, verify the charmed companion auto-assists.

If anything looks different from pre-refactor behavior, stop and
diagnose before proceeding.

- [ ] **Step 10: Test-mud AI autonomous combat run (5 minutes)**

Use the `test-mud` skill (or equivalent runner) to drive 5 minutes of
autonomous combat against the running server. Review logs for any
error or unexpected flow. If a divergence surfaces:
- If it's a known pre-existing bug → document and proceed.
- If it's introduced by this batch → revert the offending commit(s) and
  retry, or fix in a follow-up commit before committing Task 4.

### Commit

- [ ] **Step 11: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/NewRound_DoCombat_resolution.go internal/hooks/NewRound_DoCombat_helpers.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract handlePlayerVsMob phase helpers

Break handlePlayerVsMob into four helpers in
NewRound_DoCombat_resolution.go:
- resolvePlayerVsMobTarget: target validity (with multi-room exit
  check) + combat_start emission + CancelBuffsWithFlag + Health<1 gate
- handlePvMCritAndMessaging: analytics + crit + darkness + per-round
  messages
- handlePvMProgressionAndAggro: concentration break, mob_hurt behavior,
  attacker/defender progression, hostility groups, mob aggro with
  exit-walk, companion assist
- handlePvMRoundResolution: health-check endgame with "You turn your
  attention to..." retarget message

Parent handlePlayerVsMob shrinks from 286 → ~80 lines. combat_start
emission timing preserved (before attack roll, inside target-resolution
helper). Shares applyCombatDamageBonuses_PvM and handleCombatWaitRound
with Task 1 and Task 2.

Zero behavior change — verified by manual combat smoke (PvM hit/miss/
crit, MvP retaliation, wait-rounds, 2H/dual-wield/shield, Moon Phase,
retarget, companion assist) plus 5-min test-mud AI run.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Extract `applyMobEffect` damage + dot + knockdown cases

Pull the three damage-dealing `EffectType` branches into per-case
helpers. Parent's switch cases become one-liners for those three.

**Files:**
- Modify: `internal/hooks/spell_resolution.go` — add helpers inline, replace switch cases.

### Setup

- [ ] **Step 1: Confirm `applyMobEffect` boundaries**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "^func applyMobEffect\|^func resolveAgainstPlayer" internal/hooks/spell_resolution.go
```

Expected: `applyMobEffect` at ~244, `resolveAgainstPlayer` at ~490.
`applyMobEffect` ends at line 487.

- [ ] **Step 2: Confirm shared pre-case setup and aggro blocks**

Current pre-case setup (lines 244–257):
- `dmgDealt := 0`
- `critTag := ""` set from `isCrit`
- `viewerId := 0` from `user`
- `mName := mobDisplayName(mob, room, viewerId)`

These shared variables need to be passed into each case helper as
params. Signature template per spec:

```go
func applyMobEffect_<case>(
    user *users.UserRecord,
    casterChar *characters.Character,
    mob *mobs.Mob,
    room *rooms.Room,
    spellData *spells.SpellData,
    magnitude int,
    isCrit bool,
    critTag string,
    mName string,
) (dmgDealt int)
```

Not every helper needs every param — `applyMobEffect_dot` doesn't use
`casterChar` for damage calc (only needs it for skill lookup via
`user`), and `applyMobEffect_tame` uses only user/mob/room/mName.
Drop unused params per-helper.

**Gotcha:** The "set aggro on both sides immediately" block at
lines 281–289, 328–336, 365–373 is triplicated across damage/dot/
knockdown. Task 7 will consolidate it. For Tasks 5 and 6, each
extracted helper keeps its own inline copy — do NOT pre-extract the
aggro helper here.

**Gotcha:** `applyMobEffect_damage` signature takes `casterChar` because
`combat.TrySpellDeflection` requires it. Don't drop it even though the
switch dispatcher already has it.

### Extract

- [ ] **Step 3: Add `applyMobEffect_damage`**

Insert `applyMobEffect_damage` in `spell_resolution.go` immediately
before the parent `applyMobEffect`. Body mirrors the current `"damage":`
case (lines 260–313) verbatim:

```go
func applyMobEffect_damage(
    user *users.UserRecord,
    casterChar *characters.Character,
    mob *mobs.Mob,
    room *rooms.Room,
    spellData *spells.SpellData,
    magnitude int,
    isCrit bool,
    critTag string,
    mName string,
) int {
    // body = verbatim "damage" case from current applyMobEffect:
    // 1. dmg := calcSpellDamageForCharacter(...)
    // 2. TrySpellDeflection → deflected/critDeflect flags + dmg adjustment
    // 3. dmgDealt := dmg; mob.Character.Health -= dmg
    // 4. aggro block (triplicated — consolidated in Task 7):
    //    if mob.Character.Aggro == nil { mob.PreventIdle = true; if user != nil { mob.Character.SetAggro(user.UserId, 0, DefaultAttack) } }
    //    if user != nil && user.Character.Aggro == nil { user.Character.SetAggro(0, mob.InstanceId, DefaultAttack) }
    // 5. three-branch messaging (critDeflect / partial deflect / normal)
    // 6. return dmgDealt
}
```

- [ ] **Step 4: Add `applyMobEffect_dot`**

```go
func applyMobEffect_dot(
    user *users.UserRecord,
    mob *mobs.Mob,
    room *rooms.Room,
    spellData *spells.SpellData,
    magnitude int,
    critTag string,
    mName string,
) int {
    // body = verbatim "dot" case (lines 315–344):
    // 1. casterSkill / casterWil lookup (uses user; defaults if user == nil)
    // 2. dotDuration := calcSpellDuration(...) / 3; if < 3 { = 3 }
    // 3. mob.Character.AddCondition(ConditionPoisoned, dotDuration, float64(magnitude), "spell")
    // 4. aggro block (as in Task 5 Step 3)
    // 5. messaging
    // 6. return 0 (dot doesn't deal immediate damage)
}
```

- [ ] **Step 5: Add `applyMobEffect_knockdown`**

```go
func applyMobEffect_knockdown(
    user *users.UserRecord,
    casterChar *characters.Character,
    mob *mobs.Mob,
    room *rooms.Room,
    spellData *spells.SpellData,
    magnitude int,
    isCrit bool,
    critTag string,
    mName string,
) int {
    // body = verbatim "knockdown" case (lines 346–387):
    // 1. dmg := calcSpellDamageForCharacter(...)
    // 2. TrySpellDeflection (knockdown still applies even if damage deflected)
    // 3. dmgDealt := dmg; mob.Character.Health -= dmg
    // 4. mob.Character.CombatPosition = PositionProne; PositionRoundsMin = 1
    // 5. aggro block (as in Task 5 Step 3)
    // 6. two-branch messaging (deflected / normal) + RoomMsg
    // 7. return dmgDealt
}
```

- [ ] **Step 6: Rewrite those three switch cases as one-liners**

In `applyMobEffect`, replace the three cases:

```go
case "damage":
    dmgDealt = applyMobEffect_damage(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
case "dot":
    dmgDealt = applyMobEffect_dot(user, mob, room, spellData, magnitude, critTag, mName)
case "knockdown":
    dmgDealt = applyMobEffect_knockdown(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
```

Leave the `"buff"`, `"tame"`, `default` cases intact for Task 6.

### Verify

- [ ] **Step 7: Build + vet + test (with focus on `TestApplyMobEffect_*`)**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/... -run "TestApplyMobEffect" -v
go test ./internal/hooks/...
```

Expected: all 7 `TestApplyMobEffect_*` tests PASS unchanged:
- `TestApplyMobEffect_Damage`
- `TestApplyMobEffect_DamageWithCrit`
- `TestApplyMobEffect_DotEffect`
- `TestApplyMobEffect_NilUser`
- `TestApplyMobEffect_Knockdown`
- `TestApplyMobEffect_Buff`
- `TestApplyMobEffect_Tame_NotAnimal`
- `TestApplyMobEffect_DefaultEffect`

Any failure = stop, diagnose, fix before committing.

### Commit

- [ ] **Step 8: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract applyMobEffect damage + dot + knockdown cases

Pull the three damage-dealing EffectType branches out of applyMobEffect
into per-case helpers applyMobEffect_damage, applyMobEffect_dot,
applyMobEffect_knockdown. Parent switch cases become one-liners;
shared pre-case setup (critTag, viewerId, mName) continues to live in
the parent and is passed down.

Triplicated aggro block still inline in each helper — consolidated in
a later commit per plan.

Zero behavior change. Verified by all 7 existing TestApplyMobEffect_*
tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Extract `applyMobEffect` buff + tame + default cases

Extract the remaining three branches. Parent becomes a ~50-line
dispatcher + shared pre-case setup.

**Files:**
- Modify: `internal/hooks/spell_resolution.go` — add 3 helpers, shrink switch.

### Setup

- [ ] **Step 1: Confirm remaining cases**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n 'case "buff":\|case "tame":\|default:' internal/hooks/spell_resolution.go
```

Expected: one `buff` case, one `tame` case, one `default:` inside
`applyMobEffect` (~lines 389–484). Ignore any matches in other
functions.

**Gotcha:** `applyMobEffect_buff` has a CONDITIONAL aggro block — only
fires for `Harm*` spell types (lines 418–428). Preserve the
`spellData.Type == HarmSingle || HarmArea || HarmMulti` gate. Task 7
will NOT consolidate this aggro call into the shared helper because it's
gated; document the separation when adding the helper.

**Gotcha:** `applyMobEffect_buff` has an inner loop over `spellData.BuffIds`
with a `buffs.GetBuffSpec` + `buffs.ComputeTickAmount` computation that
references `user.Character.Equipment.Weapon` and `items.GetItemSpec`.
Moving this block needs the `items` and `buffs` imports already present
in `spell_resolution.go`.

**Gotcha:** `applyMobEffect_tame` has an anti-recursion cleanup loop
over `mob.Character.GetCharmIds()` that calls `mobs.DestroyInstance`.
Preserve the ordering: strip sub-charms → clear CharmedMobs → Charm(user) →
EndAggro → TrackCharmed → messaging. Do NOT rearrange.

### Extract

- [ ] **Step 2: Add `applyMobEffect_buff`**

```go
func applyMobEffect_buff(
    user *users.UserRecord,
    mob *mobs.Mob,
    room *rooms.Room,
    spellData *spells.SpellData,
    critTag string,
    mName string,
) int {
    // body = verbatim "buff" case (lines 389–436):
    // 1. for buffId range spellData.BuffIds: mob.AddBuff + tick-pool snapshot (HealthMax / StaminaMax / ConvictionMax) when user != nil
    // 2. conditional aggro for Harm* spell types (UNIQUE to buff — keep inline even after Task 7)
    // 3. messaging to user + room
    // 4. return 0
}
```

- [ ] **Step 3: Add `applyMobEffect_tame`**

```go
func applyMobEffect_tame(
    user *users.UserRecord,
    mob *mobs.Mob,
    room *rooms.Room,
    mName string,
) int {
    // body = verbatim "tame" case (lines 438–477):
    // 1. isAnimal check; if not, emit "cannot be tamed" and return 0
    // 2. if user != nil { anti-recursion cleanup; CharmedMobs = nil; Charm; EndAggro; TrackCharmed; messaging }
    // 3. return 0
}
```

- [ ] **Step 4: Add `applyMobEffect_default`**

```go
func applyMobEffect_default(
    user *users.UserRecord,
    spellData *spells.SpellData,
    mName string,
) int {
    // body = verbatim default case (lines 479–484):
    // if user != nil { user.SendText("...takes effect on %s.", spellData.Name, mName) }
    // return 0
}
```

- [ ] **Step 5: Collapse the switch**

Rewrite `applyMobEffect` body to match the spec's target shape (~50 lines):

```go
func applyMobEffect(user *users.UserRecord, casterChar *characters.Character, mob *mobs.Mob, room *rooms.Room, spellData *spells.SpellData, magnitude int, isCrit bool) int {
    critTag := ""
    if isCrit {
        critTag = ` <ansi fg="yellow">[CRIT!]</ansi>`
    }
    viewerId := 0
    if user != nil {
        viewerId = user.UserId
    }
    mName := mobDisplayName(mob, room, viewerId)

    switch spellData.EffectType {
    case "damage":
        return applyMobEffect_damage(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
    case "dot":
        return applyMobEffect_dot(user, mob, room, spellData, magnitude, critTag, mName)
    case "knockdown":
        return applyMobEffect_knockdown(user, casterChar, mob, room, spellData, magnitude, isCrit, critTag, mName)
    case "buff":
        return applyMobEffect_buff(user, mob, room, spellData, critTag, mName)
    case "tame":
        return applyMobEffect_tame(user, mob, room, mName)
    default:
        return applyMobEffect_default(user, spellData, mName)
    }
}
```

### Verify

- [ ] **Step 6: Build + vet + test**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/... -run "TestApplyMobEffect" -v
go test ./internal/hooks/...
```

Expected: all 7 `TestApplyMobEffect_*` tests pass.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
refactor(hooks): extract applyMobEffect buff + tame + default cases

Extract applyMobEffect_buff, applyMobEffect_tame, applyMobEffect_default.
Parent applyMobEffect becomes a ~50-line dispatcher: shared pre-case
setup (critTag, viewerId, mName) + switch → helper.

The buff-case conditional aggro (Harm* spell types only) stays inline
in applyMobEffect_buff because it's gated — not consolidated in Task 7.

Zero behavior change. Verified by all 7 existing TestApplyMobEffect_*
tests.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Consolidate shared `applyMobEffect` aggro helper

The "set aggro on both sides immediately" block appears three times
inside `applyMobEffect_damage`, `_dot`, `_knockdown`. Collapse into one
helper `setMobSpellAggro`. `applyMobEffect_buff`'s aggro call stays
inline because it's conditionally gated on `Harm*` types.

**Files:**
- Modify: `internal/hooks/spell_resolution.go` — add `setMobSpellAggro`, replace 3 inline blocks.

### Setup

- [ ] **Step 1: Confirm the three duplicated blocks**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
grep -n "Set aggro on both sides immediately" internal/hooks/spell_resolution.go
```

Expected: 3 matches — one each in `applyMobEffect_damage`, `_dot`,
`_knockdown`. If not exactly 3, stop and re-read.

- [ ] **Step 2: Verify shape of each block**

All three should be byte-identical:

```go
if mob.Character.Aggro == nil {
    mob.PreventIdle = true
    if user != nil {
        mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
    }
}
if user != nil && user.Character.Aggro == nil {
    user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
}
```

**Gotcha:** If any of them differs (for example, a missing
`PreventIdle` or a different `SetAggro` type), STOP — do not force
unification. The spec risk register explicitly warns about subtle
per-case differences. Document and revert to plan-level: skip the
divergent case from the helper (stays inline), or leave Task 7
as-shipped for the remaining identical pair.

### Extract

- [ ] **Step 3: Add `setMobSpellAggro` helper**

Insert above `applyMobEffect_damage`:

```go
// setMobSpellAggro sets reciprocal aggro between the caster and the
// mob target immediately after a hostile spell lands. Safe to call
// with user == nil (no-op on the user side, still sets mob aggro
// toward nobody-in-particular by leaving the branch off).
//
// Note: the buff case (applyMobEffect_buff) does NOT call this helper
// because its aggro is gated on spell Type being Harm*. Keep that
// block inline.
func setMobSpellAggro(user *users.UserRecord, mob *mobs.Mob) {
    if mob.Character.Aggro == nil {
        mob.PreventIdle = true
        if user != nil {
            mob.Character.SetAggro(user.UserId, 0, characters.DefaultAttack)
        }
    }
    if user != nil && user.Character.Aggro == nil {
        user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
    }
}
```

- [ ] **Step 4: Replace the three inline blocks with helper calls**

In `applyMobEffect_damage`, `applyMobEffect_dot`, and
`applyMobEffect_knockdown`, replace the 7-line inline aggro block with:

```go
setMobSpellAggro(user, mob)
```

Double-check placement — the replacement must be in the exact same
position in each helper (before the case-specific messaging).

### Verify

- [ ] **Step 5: Build + vet + spell batch tests**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./internal/hooks/... -run "TestApplyMobEffect" -v
go test ./...
```

Expected: all 7 `TestApplyMobEffect_*` pass AND full `./...` suite
clean. This is the spell-batch boundary; full suite required.

- [ ] **Step 6: Manual spell smoke test**

Start local server. Cast one spell per `EffectType` branch at a mob
and verify behavior:

1. **`damage`** — cast `sparks` (or equivalent single-target damage spell) at a hostile mob. Verify damage number displays, mob takes the hit, aggro engages, crit messaging works by repeated casting to force a crit.
2. **`dot`** — cast `poison` at a mob. Verify `ConditionPoisoned` message emits, aggro engages, damage ticks over subsequent rounds.
3. **`knockdown`** — cast `knockback` (or equivalent). Verify prone + position lockout, mob can't immediately swing.
4. **`buff` (harmful)** — cast `weaken` at a mob. Verify buff applies, aggro engages (because it's a Harm* type), tick-pool snapshot computed.
5. **`tame`** — cast `tame` at an animal-group mob. Verify charm applies, mob follows you, "becomes your companion" message.
6. **`default`** — pick a spell with an unhandled EffectType if any exists (or verify fallback by intentionally casting an edge-case spell). Verify "takes effect on %s" fallback message.

Also cross-verify **player-side bonus-damage still works** (from Task
1): attack a mob with low HP to trigger Adrenaline Surge and confirm
the bonus damage applies; attack with a lifesteal-enchanted weapon and
verify healing + "feeds on the blow" message.

Any deviation = stop, diagnose before committing.

### Commit

- [ ] **Step 7: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add internal/hooks/spell_resolution.go
git commit -m "$(cat <<'EOF'
refactor(hooks): consolidate shared applyMobEffect aggro helper

The "set aggro on both sides immediately" block appeared three times
inside applyMobEffect_damage / _dot / _knockdown (byte-identical).
Collapse into setMobSpellAggro, called from all three.

applyMobEffect_buff's aggro block stays inline because it's gated on
Harm* spell types — different semantics.

Zero behavior change. Verified by all 7 existing TestApplyMobEffect_*
tests plus spell smoke across damage / dot / knockdown / buff / tame.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Mark 1.2a complete in overview

**Files:**
- Modify: `docs/superpowers/code_cleanup_stage_1_overview.md`

- [ ] **Step 1: Update status row**

Edit `docs/superpowers/code_cleanup_stage_1_overview.md`. Find the 1.2a
row (likely `| 1.2a | God-Function Refactor — Combat + Spell | 5h | Med-High | Planning |`).
Change `Planning` to `Complete`.

- [ ] **Step 2: Verify overview body still matches what shipped**

Read the overview's 1.2a section. If it references lines of code to
decompose or function names, make sure they still match reality
post-refactor. No PATCH_NOTES entry — this is an internal-only
refactor with zero player impact.

- [ ] **Step 3: Commit**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git add docs/superpowers/code_cleanup_stage_1_overview.md
git commit -m "$(cat <<'EOF'
docs: mark code cleanup 1.2a complete

3 combat/spell god-functions decomposed:
- handlePlayerVsMob 286 → ~80 lines
- handleMobVsPlayer 236 → ~60 lines
- applyMobEffect 246 → ~50 lines dispatcher

7 combat phase helpers in new NewRound_DoCombat_resolution.go,
6 per-EffectType helpers inline in spell_resolution.go, plus a
consolidated setMobSpellAggro helper.

Existing TestApplyMobEffect_* suite (7 tests) passes unchanged on
refactored code. Combat path verified via manual smoke + test-mud AI
autonomous run.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Final verification

- [ ] **Step 1: Confirm branch commit count and order**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git log --oneline feature/stage-1.2a-combat-spell-refactor ^development
```

Expected: exactly 8 commits, in this order (newest first):

1. `docs: mark code cleanup 1.2a complete`
2. `refactor(hooks): consolidate shared applyMobEffect aggro helper`
3. `refactor(hooks): extract applyMobEffect buff + tame + default cases`
4. `refactor(hooks): extract applyMobEffect damage + dot + knockdown cases`
5. `refactor(hooks): extract handlePlayerVsMob phase helpers`
6. `refactor(hooks): extract handleMobVsPlayer phase helpers`
7. `refactor(hooks): extract combat wait-round short-circuit helper`
8. `refactor(hooks): extract combat damage-bonus pipeline helpers`

- [ ] **Step 2: Verify branch diff shape**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git diff --stat development...feature/stage-1.2a-combat-spell-refactor
```

Expected files changed:
- `internal/hooks/NewRound_DoCombat_helpers.go` — net reduction of ~400 lines (parents shrank, helpers moved out).
- `internal/hooks/NewRound_DoCombat_resolution.go` — new file, ~300 lines of helpers.
- `internal/hooks/spell_resolution.go` — net +20 to +40 lines (helpers added, parent shrank but dispatch and signatures add a bit back).
- `docs/superpowers/code_cleanup_stage_1_overview.md` — 1-line status flip.

No other files should appear. If .claude/settings.local.json,
feedback/*.txt, or Screenshot*.png show up: STOP, `git restore --staged`
them, and investigate.

- [ ] **Step 3: Final whole-project verification**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
go build ./...
go vet ./...
go test ./...
```

Expected: clean. No test regressions anywhere, including
`TestApplyMobEffect_*` (7 tests unchanged).

- [ ] **Step 4: Confirm parent function line counts meet spec targets**

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
awk '/^func handlePlayerVsMob/,/^}$/' internal/hooks/NewRound_DoCombat_helpers.go | wc -l
awk '/^func handleMobVsPlayer/,/^}$/' internal/hooks/NewRound_DoCombat_helpers.go | wc -l
awk '/^func applyMobEffect\(/,/^}$/' internal/hooks/spell_resolution.go | wc -l
```

Expected (from spec success criteria):
- `handlePlayerVsMob` ≤ 80 lines.
- `handleMobVsPlayer` ≤ 60 lines.
- `applyMobEffect` ~50 lines.

If any parent exceeds target, review the corresponding task for an
extraction opportunity missed. If helpers exceed 80 lines each, ditto.

---

## Merge to development (after user review)

Do NOT merge until the user has reviewed the branch. Once approved:

```bash
cd "/c/Users/Calabe Davis/workspace/DOGMud"
git checkout development
git merge --no-ff feature/stage-1.2a-combat-spell-refactor -m "$(cat <<'EOF'
merge: stage 1.2a combat + spell god-function refactor

Decomposes handlePlayerVsMob (286→~80), handleMobVsPlayer (236→~60),
applyMobEffect (246→~50 dispatcher). New file
internal/hooks/NewRound_DoCombat_resolution.go for combat phase helpers;
spell case helpers stay inline in spell_resolution.go.

All 7 TestApplyMobEffect_* tests pass unchanged. Combat path verified
by manual smoke + test-mud AI autonomous run. Zero player-visible
behavior change.

Co-Authored-By: Claude Opus 4.7 (1M context) <noreply@anthropic.com>
EOF
)"
```

Do NOT push to origin. Combat changes deserve a soak — let 1.2a bake on
`development` for a week of test-mud exposure before rolling to `master`.
