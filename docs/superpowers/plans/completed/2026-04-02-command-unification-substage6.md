# Command Unification — Substage 6: Tests + Guardrails

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Comprehensive parity test suite, verify registry audit catches drift, document intentional divergences, final cleanup.

**Architecture:** Tests verify the registry audit works. Documentation captures the design decisions. Cleanup squashes any remaining issues.

**Tech Stack:** Go, testify/assert

---

### Task 1: Registry Audit Integration Test

**Files:**
- Modify: `internal/actions/actions_test.go`

Test that the audit function correctly identifies gaps:

- [ ] **Step 1: Test audit catches unlisted user command**

Add a test that passes a user command list containing a command NOT in
the mob list and NOT in the allowlist. Verify the audit logs a warning
(capture via test log or verify no panic).

- [ ] **Step 2: Test audit catches unlisted mob command**

Same but for mob-only.

- [ ] **Step 3: Test audit passes with current registries**

Call `AuditCommandParity` with the actual `GetAllUserCommands()` and
`GetAllMobCommands()` outputs. This is the real integration test — if
any new command was added to either side without updating the allowlist,
this test fails.

- [ ] **Step 4: Commit**

```bash
git commit -m "test: registry audit integration tests"
```

---

### Task 2: Document Intentional Divergences

**Files:**
- Create: `docs/INTENTIONAL_DIVERGENCES.md`

Document all intentional differences between user and mob command
systems with rationale.

- [ ] **Step 1: Write the document**

Sections:
1. **Admin commands** — user-only, no mob equivalent needed
2. **UI commands** — display/config, no game action
3. **Player mechanics** — party, PvP, whisper, etc.
4. **Mob AI** — lookfortrouble, wander, pathto, etc.
5. **Pending consolidation** — howl/taunt, backstab, roar, throw, alchemy
6. **Behavioral asymmetries** — flee (state vs instant), sneak initiation
   roll (players have skill gate, mobs don't), spell initiation
   (players roll, mobs skip)
7. **Progression** — mobs now progress stats/skills/spells same as players

- [ ] **Step 2: Commit**

```bash
git commit -m "docs: INTENTIONAL_DIVERGENCES.md for command unification"
```

---

### Task 3: Final Full Test Run + Cleanup

- [ ] **Step 1: Run ALL tests**

```bash
go test ./... -count=1 -timeout 300s
```

Fix any failures.

- [ ] **Step 2: Verify build**

```bash
go build ./...
go vet ./...
```

- [ ] **Step 3: Review git log**

Check all commits make sense. No WIP or broken intermediate states.

- [ ] **Step 4: Update PATCH_NOTES.md**

Add a section for the command unification work.

- [ ] **Step 5: Final commit**

```bash
git commit -m "chore: substage 6 — final tests, docs, cleanup"
```
