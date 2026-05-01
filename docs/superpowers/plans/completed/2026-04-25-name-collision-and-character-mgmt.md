# Name Collision Prevention + Character Management Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Centralize name validation, surface mob/player name collisions, and add player `rename` + `deletecharacter` commands with cooldowns and confirmations.

**Architecture:** A new `users.ValidateActorName` function consolidates duplicated validation across pet/companion/character creation. A boot-time `mobs.AuditMobNameCollisions` warns on conflicts. Two new top-level commands (`rename`, `deletecharacter`) operate on the active character with opt-out-default confirmations; the existing admin `rename` migrates to `renameitem`.

**Tech Stack:** Go (engine), YAML (config), GoMud command/prompt framework, mudlog for logging.

**Spec:** `docs/superpowers/specs/2026-04-25-name-collision-and-character-mgmt-design.md`

---

## File Structure

**New files:**

- `internal/users/validate_actor_name.go` — single-source-of-truth validator + opts struct.
- `internal/users/validate_actor_name_test.go` — table-driven tests for the validator.
- `internal/usercommands/renameself.go` — player `rename` command handler.
- `internal/usercommands/renameself_test.go` — command flow tests.
- `internal/usercommands/deletecharacter.go` — player `deletecharacter` command handler.
- `internal/usercommands/deletecharacter_test.go` — command flow tests.
- `_datafiles/world/dogmud/templates/help/rename.template` — `help rename` text.
- `_datafiles/world/dogmud/templates/help/deletecharacter.template` — `help deletecharacter` text.

**Modified files:**

- `internal/configs/config.balance.go` — add `CharacterRenameCooldownHours`.
- `_datafiles/config.yaml` — add the new knob with default 168.
- `internal/users/userrecord.go` — add `LastRenameAt time.Time` field.
- `internal/users/users.go` — add `RenameUser`, `RemoveUserAndDisconnect`; refactor `ValidateName`.
- `internal/usercommands/start.go` — replace inline validation with one `ValidateActorName` call.
- `internal/inputhandlers/login_prompt_handler.go` — same migration.
- `internal/usercommands/pet.go` — same migration.
- `internal/usercommands/companion.go` — same migration (fixes missing mob check).
- `internal/usercommands/usercommands.go` — register `rename`, `deletecharacter`; rename admin entry to `renameitem`.
- `internal/mobs/mobs.go` — add `AuditMobNameCollisions`.
- `internal/mobs/mobs_test.go` — add audit tests.
- `main.go` — call audit after `mobs.LoadDataFiles()`.

---

## Task 1: Add `CharacterRenameCooldownHours` config

**Files:**
- Modify: `internal/configs/config.balance.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the field to the Balance struct**

Open `internal/configs/config.balance.go` and find a logical grouping (regen knobs or similar). Add:

```go
CharacterRenameCooldownHours ConfigInt `yaml:"CharacterRenameCooldownHours"` // Hours between player renames (0 disables; default 168 = 7 days)
```

- [ ] **Step 2: Add a default-clamp in the Balance config validator**

In the same file there is a Validate-style method (look for other `if g.X < 0 { g.X = 0 }` patterns). Add:

```go
if b.CharacterRenameCooldownHours < 0 {
    b.CharacterRenameCooldownHours = 0
}
```

If no such method exists, skip — `ConfigInt` zero is a valid disabled state.

- [ ] **Step 3: Add the knob to config.yaml under Balance**

Open `_datafiles/config.yaml` and add inside the `Balance:` block:

```yaml
  # Hours between player rename uses. 0 disables the cooldown (rename
  # anytime). Default 168 (7 real-time days).
  CharacterRenameCooldownHours: 168
```

- [ ] **Step 4: Run the build to verify it compiles**

```bash
go build ./...
```

Expected: builds clean.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat(config): add CharacterRenameCooldownHours balance knob

Default 168 hours (7 days). Foundation for the upcoming player
rename command's cooldown enforcement."
```

---

## Task 2: Add `LastRenameAt` field to UserRecord

**Files:**
- Modify: `internal/users/userrecord.go`

- [ ] **Step 1: Add the field**

In `internal/users/userrecord.go`, locate the `UserRecord` struct field block (around line 31-40 — fields like `Joined time.Time`). Add after `Joined`:

```go
LastRenameAt time.Time `yaml:"lastrenameat,omitempty"` // Last time the player used the rename command; cooldown uses this
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./...
```

Expected: builds clean. The `time` package should already be imported.

- [ ] **Step 3: Commit**

```bash
git add internal/users/userrecord.go
git commit -m "feat(users): add LastRenameAt field to UserRecord

Persisted with omitempty; populated by the upcoming rename command
to enforce the per-user cooldown."
```

---

## Task 3: Write the `ValidateActorName` test fixture (failing)

**Files:**
- Create: `internal/users/validate_actor_name_test.go`

- [ ] **Step 1: Write a table-driven test**

```go
package users

import (
    "strings"
    "testing"
)

func TestValidateActorName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        opts      ValidateActorOpts
        wantErr   bool
        errSubstr string
    }{
        {name: "empty", input: "", wantErr: true, errSubstr: "between"},
        {name: "valid_novel_name", input: "Bobblesworth", wantErr: false},
        {name: "skip_mob_check_passes", input: "Bobblesworth", opts: ValidateActorOpts{SkipMobCheck: true}, wantErr: false},
        {name: "skip_banned_check_passes", input: "Bobblesworth", opts: ValidateActorOpts{SkipBannedCheck: true}, wantErr: false},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateActorName(tt.input, tt.opts)
            if tt.wantErr && err == nil {
                t.Fatalf("expected error, got nil")
            }
            if !tt.wantErr && err != nil {
                t.Fatalf("expected no error, got %v", err)
            }
            if tt.wantErr && tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
                t.Fatalf("expected error to contain %q, got %v", tt.errSubstr, err)
            }
        })
    }
}
```

- [ ] **Step 2: Run the test to verify it fails to compile**

```bash
go test ./internal/users/ -run TestValidateActorName -v
```

Expected: build error (`ValidateActorName` and `ValidateActorOpts` undefined).

---

## Task 4: Implement `ValidateActorName`

**Files:**
- Create: `internal/users/validate_actor_name.go`

- [ ] **Step 1: Implement the validator**

```go
package users

import (
    "errors"
    "fmt"
    "regexp"
    "strings"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/mobs"
)

// ValidateActorOpts tunes which checks ValidateActorName performs.
type ValidateActorOpts struct {
    SkipMobCheck    bool // skip mob-template-name collision check
    SkipBannedCheck bool // skip banned-pattern check
    ExcludeUserId   int  // ignore collisions on this user (self-rename)
}

// ValidateActorName is the single source of truth for any name a player
// or companion will be referred to by in-world. Pet/companion/character
// creation and self-rename all go through this. Returns nil on success
// or a player-displayable error explaining the rejection.
func ValidateActorName(name string, opts ValidateActorOpts) error {
    validation := configs.GetValidationConfig()

    if len(name) < int(validation.NameSizeMin) || len(name) > int(validation.NameSizeMax) {
        return fmt.Errorf("name must be between %d and %d characters long",
            validation.NameSizeMin, validation.NameSizeMax)
    }

    if validation.NameRejectRegex != `` {
        if !regexp.MustCompile(validation.NameRejectRegex.String()).MatchString(name) {
            return errors.New(validation.NameRejectReason.String())
        }
    }

    if !opts.SkipBannedCheck {
        if bannedPattern, ok := configs.GetConfig().IsBannedName(name); ok {
            return fmt.Errorf(`that name matched the prohibited pattern: %q`, bannedPattern)
        }
    }

    if !opts.SkipMobCheck {
        for _, mobName := range mobs.GetAllMobNames() {
            if strings.EqualFold(mobName, name) {
                return errors.New("that name is in use")
            }
        }
    }

    if foundUserId, _ := CharacterNameSearch(name); foundUserId > 0 && foundUserId != opts.ExcludeUserId {
        return errors.New("that name is already in use")
    }

    userManagerMu.RLock()
    existingId, exists := userManager.Usernames[strings.ToLower(name)]
    userManagerMu.RUnlock()
    if exists && existingId != opts.ExcludeUserId {
        return errors.New("that name is already in use")
    }

    if CompanionNameExists(name) {
        return errors.New("that name is in use by a companion")
    }

    return nil
}
```

- [ ] **Step 2: Verify the username map mutex name**

The validator references `userManagerMu`. Confirm the lock name in `internal/users/users.go`:

```bash
grep -n "userManagerMu" internal/users/users.go | head -5
```

If the actual name differs (e.g., `userMu`, `userIndexMu`), update Step 1's code to match.

- [ ] **Step 3: Run the test to verify the basic cases pass**

```bash
go test ./internal/users/ -run TestValidateActorName -v
```

Expected: all four cases pass (`empty` errors, `valid_novel_name` passes, two skip cases pass).

- [ ] **Step 4: Commit**

```bash
git add internal/users/validate_actor_name.go internal/users/validate_actor_name_test.go
git commit -m "feat(users): centralize actor-name validation

Single source of truth for player/character/companion name validation.
Replaces five duplicated validation sites in subsequent commits."
```

---

## Task 5: Add collision-coverage tests for `ValidateActorName`

**Files:**
- Modify: `internal/users/validate_actor_name_test.go`

- [ ] **Step 1: Look at the existing test helpers in the package**

```bash
grep -n "func TestMain\|test_helpers\|fixtureUser\|seedUser" internal/users/*.go | head -10
```

The test will need a way to seed an in-memory user (or a mob) for collision cases. Use whatever helper the existing `users_test.go` uses for similar setup. Inspect:

```bash
sed -n '1,80p' internal/users/users_test.go
```

- [ ] **Step 2: Add collision tests**

Add to `internal/users/validate_actor_name_test.go`:

```go
func TestValidateActorName_PlayerCollision(t *testing.T) {
    // Seed an offline user named "Calabean" — adapt to whatever helper
    // users_test.go uses (e.g., writeFixtureUser, or direct file write
    // into a t.TempDir-backed users folder).
    seedOfflineUser(t, 42, "Calabean")

    if err := ValidateActorName("Calabean", ValidateActorOpts{}); err == nil {
        t.Fatal("expected collision error, got nil")
    }

    // ExcludeUserId means self-rename of UserId=42 to "Calabean" is allowed.
    if err := ValidateActorName("Calabean", ValidateActorOpts{ExcludeUserId: 42}); err != nil {
        t.Fatalf("expected ExcludeUserId to bypass own-name collision, got %v", err)
    }
}

func TestValidateActorName_MobCollision(t *testing.T) {
    seedMobName(t, "Goblin") // adapt to mobs package test helper

    if err := ValidateActorName("Goblin", ValidateActorOpts{}); err == nil {
        t.Fatal("expected mob collision error, got nil")
    }

    if err := ValidateActorName("Goblin", ValidateActorOpts{SkipMobCheck: true}); err != nil {
        t.Fatalf("SkipMobCheck should bypass mob collision, got %v", err)
    }
}
```

If no `seedOfflineUser` / `seedMobName` helper exists, write thin local helpers in this file using the same approach the existing tests use (commonly: temp directory + write fixture YAML, then call init).

- [ ] **Step 3: Run the tests**

```bash
go test ./internal/users/ -run TestValidateActorName -v
```

Expected: all sub-tests pass.

- [ ] **Step 4: Commit**

```bash
git add internal/users/validate_actor_name_test.go
git commit -m "test(users): collision coverage for ValidateActorName

Verifies player-name, mob-name, and ExcludeUserId/SkipMobCheck
opt behaviors."
```

---

## Task 6: Migrate `users.ValidateName` to delegate to `ValidateActorName`

**Files:**
- Modify: `internal/users/users.go`

- [ ] **Step 1: Replace the body of `ValidateName`**

Locate `func ValidateName(name string) error` (around line 516). Replace its body with:

```go
func ValidateName(name string) error {
    return ValidateActorName(name, ValidateActorOpts{})
}
```

Remove the now-unused inline regex/banned/mob/Exists/CompanionNameExists checks within the function (keep the function signature for backward compat with all existing callers).

- [ ] **Step 2: Verify imports — remove `regexp` if no longer used elsewhere in users.go**

```bash
go build ./internal/users/
```

If `regexp` import becomes unused, remove it.

- [ ] **Step 3: Run the existing users tests**

```bash
go test ./internal/users/ -v
```

Expected: all existing tests pass (the contract of `ValidateName` is unchanged).

- [ ] **Step 4: Commit**

```bash
git add internal/users/users.go
git commit -m "refactor(users): ValidateName delegates to ValidateActorName

No behavior change — eliminates duplicated checks. Subsequent commits
migrate the other four call sites."
```

---

## Task 7: Migrate `start.go` to `ValidateActorName`

**Files:**
- Modify: `internal/usercommands/start.go`

- [ ] **Step 1: Read the current name-validation block**

The block lives roughly at lines 45-90 and includes: username-equality guard, `users.ValidateName`, `IsBannedName`, `CharacterNameSearch`, and the explicit `mobs.GetAllMobNames` loop.

```bash
sed -n '45,95p' internal/usercommands/start.go
```

- [ ] **Step 2: Replace the inline checks with one call**

After the username-equality guard (line 52-56 area), keep that guard for now (Task 16 investigates it) and replace the four inline checks (`ValidateName`, `IsBannedName`, `CharacterNameSearch`, `GetAllMobNames` loop) with a single:

```go
if err := users.ValidateActorName(question.Response, users.ValidateActorOpts{}); err != nil {
    user.SendText(`That name won't work: ` + err.Error())
    return true, errors.New(err.Error())
}
```

Match the surrounding error-handling pattern (re-ask, return, etc.) — adapt control flow to whatever start.go currently does for rejected names.

- [ ] **Step 3: Drop now-unused imports**

`mobs` and `configs` may become unused inside this file. Run `go build` and remove unused imports.

```bash
go build ./internal/usercommands/
```

- [ ] **Step 4: Run usercommands tests**

```bash
go test ./internal/usercommands/ -v
```

Expected: all pass.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/start.go
git commit -m "refactor(start): use ValidateActorName for tutorial name picking"
```

---

## Task 8: Migrate `login_prompt_handler.go` to `ValidateActorName`

**Files:**
- Modify: `internal/inputhandlers/login_prompt_handler.go`

- [ ] **Step 1: Replace the body of `ValidateCharacterName`**

Locate `func ValidateCharacterName` (around line 158). Replace the four inline checks (`users.ValidateName`, `CharacterNameSearch`, `mobs.GetAllMobNames` loop, `IsBannedName`) with:

```go
if err := users.ValidateActorName(input, users.ValidateActorOpts{}); err != nil {
    return "", err
}
```

Keep the username-equality guard at the top (`strings.EqualFold(input, results["username-new"])`) — that remains a meaningful check during signup.

- [ ] **Step 2: Drop unused imports**

```bash
go build ./internal/inputhandlers/
```

`mobs` and `configs.GetConfig()` likely become unused — remove.

- [ ] **Step 3: Run inputhandlers tests**

```bash
go test ./internal/inputhandlers/ -v
```

Expected: all pass.

- [ ] **Step 4: Commit**

```bash
git add internal/inputhandlers/login_prompt_handler.go
git commit -m "refactor(login): ValidateCharacterName uses ValidateActorName"
```

---

## Task 9: Migrate `pet.go` to `ValidateActorName`

**Files:**
- Modify: `internal/usercommands/pet.go`

- [ ] **Step 1: Replace the four inline checks**

In `internal/usercommands/pet.go` around lines 38-58, replace the block:

```go
if err := users.ValidateName(newName); err != nil { ... }
if bannedPattern, ok := configs.GetConfig().IsBannedName(newName); ok { ... }
for _, name := range mobs.GetAllMobNames() { ... }
if foundUserId, _ := users.CharacterNameSearch(newName); foundUserId > 0 { ... }
```

With:

```go
if err := users.ValidateActorName(newName, users.ValidateActorOpts{}); err != nil {
    user.SendText(`That name won't work: ` + err.Error())
    return true, nil
}
```

- [ ] **Step 2: Drop unused imports**

```bash
go build ./internal/usercommands/
```

`mobs` likely becomes unused if no other reference; remove.

- [ ] **Step 3: Run pet-related tests**

```bash
go test ./internal/usercommands/ -run Pet -v
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/pet.go
git commit -m "refactor(pet): use ValidateActorName for pet naming"
```

---

## Task 10: Migrate `companion.go` `validateCompanionName` (fixes mob-check gap)

**Files:**
- Modify: `internal/usercommands/companion.go`

- [ ] **Step 1: Replace the body of `validateCompanionName`**

Locate `func validateCompanionName(name string) error` (around line 286). Replace the body with:

```go
func validateCompanionName(name string) error {
    // Companion names use a stricter character set than general validation
    // (letters only, 2-20 chars). Run that gate first.
    if len(name) < 2 || len(name) > 20 {
        return fmt.Errorf("Companion names must be between 2 and 20 characters.")
    }
    for _, ch := range name {
        if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') {
            return fmt.Errorf("Companion names may only contain letters.")
        }
    }

    // Standard collision checks (banned, mob, player, companion).
    if err := users.ValidateActorName(name, users.ValidateActorOpts{}); err != nil {
        return err
    }
    return nil
}
```

This delegates the standard checks while keeping companion-specific length/charset rules first (so error messages are companion-flavored where it matters).

- [ ] **Step 2: Verify the build**

```bash
go build ./internal/usercommands/
```

- [ ] **Step 3: Test it — write a tiny regression for the mob-check gap**

Add to `internal/usercommands/companion_test.go` (create the file if missing):

```go
package usercommands

import (
    "strings"
    "testing"
)

func TestValidateCompanionName_BlocksMobNames(t *testing.T) {
    // Seed a mob template named "Goblin" — adapt to whatever helper
    // existing tests in this package use (search for "GetAllMobNames" /
    // "addMobTemplate" / "fixtureMob").
    seedMobName(t, "Goblin")

    err := validateCompanionName("Goblin")
    if err == nil {
        t.Fatal("expected mob-name collision rejection, got nil")
    }
    if !strings.Contains(strings.ToLower(err.Error()), "in use") {
        t.Fatalf("expected error to mention 'in use', got %v", err)
    }
}
```

If no test-package mob helper exists, copy the pattern used by `pet_test.go` or skip this test and rely on the validator tests in Task 5 to cover the path.

- [ ] **Step 4: Run companion tests**

```bash
go test ./internal/usercommands/ -run Companion -v
```

Expected: pass (or skipped if no helper available).

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/companion.go internal/usercommands/companion_test.go
git commit -m "fix(companion): block mob-name collisions in companion naming

validateCompanionName previously skipped the mob-template check that
pet/character creation already enforces. Centralizing through
ValidateActorName closes the gap."
```

---

## Task 11: Add `users.RenameUser` helper + tests

**Files:**
- Modify: `internal/users/users.go`
- Modify: `internal/users/users_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/users/users_test.go`:

```go
func TestRenameUser_Success(t *testing.T) {
    // Bring the user manager up via whatever Init/setup the existing
    // users tests use — match the pattern from TestSaveUser / similar.
    u := seedActiveUser(t, 100, "Alice")

    if err := RenameUser(u, "Bobbi"); err != nil {
        t.Fatalf("expected nil err, got %v", err)
    }
    if u.Username != "Bobbi" {
        t.Errorf("Username = %q, want Bobbi", u.Username)
    }
    if u.Character.Name != "Bobbi" {
        t.Errorf("Character.Name = %q, want Bobbi", u.Character.Name)
    }
    // Old name freed:
    if _, _ := CharacterNameSearch("Alice"); _ != 0 {
        t.Error("expected Alice to be freed from the index")
    }
    // New name owned by 100:
    if id, _ := CharacterNameSearch("Bobbi"); id != 100 {
        t.Errorf("Bobbi should resolve to userId 100, got %d", id)
    }
}

func TestRenameUser_NameAlreadyClaimed(t *testing.T) {
    seedActiveUser(t, 200, "Alice")
    other := seedActiveUser(t, 201, "Charlie")

    if err := RenameUser(other, "Alice"); err == nil {
        t.Fatal("expected error renaming to claimed name, got nil")
    }
    if other.Username != "Charlie" {
        t.Errorf("Username should be untouched, got %q", other.Username)
    }
}
```

- [ ] **Step 2: Run the test to verify it fails**

```bash
go test ./internal/users/ -run TestRenameUser -v
```

Expected: `RenameUser undefined`.

- [ ] **Step 3: Implement `RenameUser`**

Append to `internal/users/users.go`:

```go
// RenameUser atomically updates Username + Character.Name + the username
// index. Caller is responsible for calling SaveUser (which writes to the
// existing UserId-keyed file) and setting LastRenameAt.
func RenameUser(u *UserRecord, newName string) error {
    userManagerMu.Lock()
    defer userManagerMu.Unlock()

    oldName := u.Username
    if _, exists := userManager.Usernames[strings.ToLower(newName)]; exists {
        return errors.New("name was just claimed")
    }
    delete(userManager.Usernames, strings.ToLower(oldName))
    userManager.Usernames[strings.ToLower(newName)] = u.UserId
    u.Username = newName
    if u.Character != nil {
        u.Character.Name = newName
    }

    idx.RemoveUser(oldName)
    idx.AddUser(u.UserId, newName)

    return nil
}
```

If the actual mutex name in the file is not `userManagerMu`, adapt. If `idx` is not the package-level index variable name, look up the correct symbol via `grep -n "idx\." internal/users/users.go`.

- [ ] **Step 4: Run the tests to verify they pass**

```bash
go test ./internal/users/ -run TestRenameUser -v
```

Expected: both subtests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/users/users.go internal/users/users_test.go
git commit -m "feat(users): add RenameUser helper

Atomically updates Username, Character.Name, and the username/userid
indexes under the manager lock. SaveUser writes to the same
UserId-keyed file, so no disk rename is needed."
```

---

## Task 12: Add `users.RemoveUserAndDisconnect` helper + tests

**Files:**
- Modify: `internal/users/users.go`
- Modify: `internal/users/users_test.go`

- [ ] **Step 1: Write the failing test**

```go
func TestRemoveUserAndDisconnect_FreesName(t *testing.T) {
    u := seedActiveUser(t, 300, "Doomed")
    _ = u

    if err := RemoveUserAndDisconnect(300); err != nil {
        t.Fatalf("expected nil err, got %v", err)
    }
    // Username index must be freed:
    if id, _ := CharacterNameSearch("Doomed"); id != 0 {
        t.Errorf("expected Doomed to be unclaimable, still resolves to %d", id)
    }
    // File must be gone:
    path := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "users", "300.yaml")
    if _, err := os.Stat(path); !os.IsNotExist(err) {
        t.Errorf("expected user file to be deleted, stat err=%v", err)
    }
}
```

Add the imports `os`, `path/filepath`, `github.com/GoMudEngine/GoMud/internal/configs` if not already present in the test file.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/users/ -run TestRemoveUserAndDisconnect -v
```

Expected: `RemoveUserAndDisconnect undefined`.

- [ ] **Step 3: Implement `RemoveUserAndDisconnect`**

Append to `internal/users/users.go`:

```go
// RemoveUserAndDisconnect logs the user out, deletes their save file,
// frees username + character name from in-memory indexes, and closes
// their connection. Charmed mobs are uncharmed before logout so they
// don't dangle.
func RemoveUserAndDisconnect(userId int) error {
    u := GetByUserId(userId)
    if u == nil {
        return errors.New("user not found")
    }

    if u.Character != nil {
        for _, mobInstanceId := range u.Character.GetCharmIds() {
            if m := mobs.GetInstance(mobInstanceId); m != nil {
                m.Character.RemoveCharm()
            }
        }
    }

    LogOutUserByConnectionId(u.ConnectionId)

    userPath := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), `/`, `users`, `/`, strconv.Itoa(u.UserId)+`.yaml`)
    if err := os.Remove(userPath); err != nil && !os.IsNotExist(err) {
        return fmt.Errorf("failed to delete user file: %w", err)
    }

    userManagerMu.Lock()
    delete(userManager.Usernames, strings.ToLower(u.Username))
    userManagerMu.Unlock()
    idx.RemoveUser(u.Username)

    connections.Remove(u.ConnectionId)
    return nil
}
```

Add imports as needed: `os`, `strconv`, `github.com/GoMudEngine/GoMud/internal/util`, `github.com/GoMudEngine/GoMud/internal/connections`. Most likely already imported.

- [ ] **Step 4: Run tests**

```bash
go test ./internal/users/ -run TestRemoveUserAndDisconnect -v
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/users/users.go internal/users/users_test.go
git commit -m "feat(users): add RemoveUserAndDisconnect helper

Tears down a user account: uncharms mobs, logs out, deletes
{datafiles}/users/{UserId}.yaml, frees indexes, closes the
connection."
```

---

## Task 13: Migrate admin `rename` registration to `renameitem`

**Files:**
- Modify: `internal/usercommands/usercommands.go`
- Modify: `_datafiles/world/dogmud/templates/admincommands/help/command.rename` (rename file)

- [ ] **Step 1: Update the registration**

In `internal/usercommands/usercommands.go` around line 150, change:

```go
`rename`:      {Rename, false, true, true},    // Admin only
```

to:

```go
`renameitem`:  {Rename, false, true, true},    // Admin only — renames an item in inventory
```

- [ ] **Step 2: Rename the help template file**

```bash
git mv _datafiles/world/dogmud/templates/admincommands/help/command.rename _datafiles/world/dogmud/templates/admincommands/help/command.renameitem
```

If the file extension differs (`.template`), adjust accordingly:

```bash
ls _datafiles/world/dogmud/templates/admincommands/help/ | grep -i rename
```

- [ ] **Step 3: Update the template path inside admin.rename.go**

Open `internal/usercommands/admin.rename.go` and find the `templates.Process("admincommands/help/command.rename", ...)` call (around line 24). Change `command.rename` to `command.renameitem`.

- [ ] **Step 4: Build and run the existing rename test if any**

```bash
go build ./...
go test ./internal/usercommands/ -run Rename -v
```

Expected: builds clean. If a test references the command name `rename`, update it to `renameitem`.

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/usercommands.go internal/usercommands/admin.rename.go _datafiles/world/dogmud/templates/admincommands/help/
git commit -m "refactor(rename): migrate admin rename to renameitem

Frees the rename verb for the upcoming player rename command.
Admins now use renameitem to rename inventory items."
```

---

## Task 14: Add player `Rename` command

**Files:**
- Create: `internal/usercommands/renameself.go`
- Create: `_datafiles/world/dogmud/templates/help/rename.template`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Write the help template**

Create `_datafiles/world/dogmud/templates/help/rename.template`:

```
<ansi fg="cyan-bold">rename &lt;newname&gt;</ansi>

Renames your character. The chosen name must be unique across players,
companions, and mob templates. After a successful rename, you cannot
rename again for the configured cooldown (default 7 days).

Usage:
  rename Bobblesworth

You will be asked to confirm. Type <ansi fg="51">yes</ansi> to apply,
anything else to abort.
```

Match the formatting/wrap style of nearby help templates (check `_datafiles/world/dogmud/templates/help/about.template` for tone).

- [ ] **Step 2: Implement the command handler**

Create `internal/usercommands/renameself.go`:

```go
package usercommands

import (
    "errors"
    "fmt"
    "strings"
    "time"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/events"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/templates"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// Rename lets a player change their character name (and account username,
// since they share a namespace) once per CharacterRenameCooldownHours.
func Rename(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    newName := strings.TrimSpace(rest)
    if newName == `` {
        infoOutput, _ := templates.Process("help/rename", nil, user.UserId)
        user.SendText(infoOutput)
        return true, nil
    }

    if strings.EqualFold(newName, user.Character.Name) {
        user.SendText(`That's already your name.`)
        return true, nil
    }

    cooldownHours := int(configs.GetBalanceConfig().CharacterRenameCooldownHours)
    if cooldownHours > 0 && !user.LastRenameAt.IsZero() {
        elapsed := time.Since(user.LastRenameAt)
        cooldown := time.Duration(cooldownHours) * time.Hour
        if elapsed < cooldown {
            nextAt := user.LastRenameAt.Add(cooldown)
            user.SendText(fmt.Sprintf(
                `You renamed yourself recently. You can rename again on %s.`,
                nextAt.Format(`2006-01-02 at 15:04`)))
            return true, nil
        }
    }

    if err := users.ValidateActorName(newName, users.ValidateActorOpts{ExcludeUserId: user.UserId}); err != nil {
        user.SendText(fmt.Sprintf(`That name won't work: %s`, err.Error()))
        return true, nil
    }

    cmdPrompt, _ := user.StartPrompt(`rename`, rest)
    days := cooldownHours / 24
    confirmText := fmt.Sprintf(
        `Rename yourself from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>? You won't be able to rename again for %d days.`,
        user.Character.Name, newName, days)
    if cooldownHours == 0 {
        confirmText = fmt.Sprintf(
            `Rename yourself from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>?`,
            user.Character.Name, newName)
    }
    q := cmdPrompt.Ask(confirmText, []string{`yes`, `no`}, `no`)
    if !q.Done {
        return true, nil
    }
    if q.Response != `yes` {
        user.SendText(`Aborted.`)
        user.ClearPrompt()
        return true, nil
    }

    oldName := user.Character.Name
    if err := users.RenameUser(user, newName); err != nil {
        user.SendText(fmt.Sprintf(`Rename failed: %s`, err.Error()))
        user.ClearPrompt()
        return true, errors.New(err.Error())
    }
    user.LastRenameAt = time.Now()
    users.SaveUser(*user)

    user.EventLog.Add(`char`, fmt.Sprintf(`Renamed from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>`, oldName, newName))
    user.SendText(`The world ripples briefly — you are now known as <ansi fg="username">` + newName + `</ansi>.`)
    room.SendTextVisual(
        fmt.Sprintf(`<ansi fg="username">%s</ansi> shimmers and is now known as <ansi fg="username">%s</ansi>.`, oldName, newName),
        user.UserId)

    user.ClearPrompt()
    return true, nil
}
```

- [ ] **Step 3: Register the command**

In `internal/usercommands/usercommands.go`, add (alphabetical placement near `remove`):

```go
`rename`:      {Rename, false, true, false}, // Player rename — anywhere, but cooldown-gated
```

- [ ] **Step 4: Verify it compiles**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/renameself.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/help/rename.template
git commit -m "feat(rename): add player rename command

Cooldown-gated (CharacterRenameCooldownHours, default 168) with a
yes/no default-no confirmation. Updates Username + Character.Name +
indexes via users.RenameUser; SaveUser persists to the same
UserId-keyed file."
```

---

## Task 15: Add tests for the player `Rename` command

**Files:**
- Create: `internal/usercommands/renameself_test.go`

- [ ] **Step 1: Look at how existing usercommand tests stub the prompt + user**

```bash
grep -n "StartPrompt\|cmdPrompt" internal/usercommands/usercommands_test.go | head -10
ls internal/usercommands/*_test.go | head -5
```

Pick a similar test file that exercises a multi-step prompt (e.g., `character` or any file with `StartPrompt` mocked) and mirror its setup pattern.

- [ ] **Step 2: Write the tests**

```go
package usercommands

import (
    "strings"
    "testing"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/events"
)

func TestRename_NoArgs_SendsHelp(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")

    handled, err := Rename("", user, room, events.EventFlag(0))
    if err != nil {
        t.Fatalf("err = %v", err)
    }
    if !handled {
        t.Fatal("expected handled=true")
    }
    if !strings.Contains(user.LastSentText(), "rename") {
        t.Errorf("expected help text to mention rename, got %q", user.LastSentText())
    }
}

func TestRename_SameName_NoOp(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")

    Rename("Alice", user, room, events.EventFlag(0))
    if !strings.Contains(user.LastSentText(), "already your name") {
        t.Errorf("expected 'already your name' message, got %q", user.LastSentText())
    }
}

func TestRename_WithinCooldown_Refused(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    setBalanceCooldownHours(t, 168)
    user.LastRenameAt = time.Now().Add(-1 * time.Hour) // 1h ago, well inside 168h

    Rename("Bobbi", user, room, events.EventFlag(0))
    if !strings.Contains(strings.ToLower(user.LastSentText()), "renamed yourself recently") {
        t.Errorf("expected cooldown message, got %q", user.LastSentText())
    }
}

func TestRename_CooldownDisabled_AllowsImmediate(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    setBalanceCooldownHours(t, 0)
    user.LastRenameAt = time.Now().Add(-1 * time.Minute)

    // First call should advance to the confirmation prompt; the test
    // helper for this varies, but at minimum it must NOT short-circuit
    // with the "renamed recently" message.
    Rename("Bobbi", user, room, events.EventFlag(0))
    if strings.Contains(strings.ToLower(user.LastSentText()), "renamed yourself recently") {
        t.Error("cooldown=0 should not gate; got cooldown message")
    }
}
```

If `newTestUserAndRoom` / `LastSentText` / `setBalanceCooldownHours` helpers don't exist in this package, write small ones at the top of the test file using whatever scaffolding the existing usercommand tests use.

- [ ] **Step 3: Run the tests**

```bash
go test ./internal/usercommands/ -run TestRename -v
```

Expected: pass.

- [ ] **Step 4: Commit**

```bash
git add internal/usercommands/renameself_test.go
git commit -m "test(rename): cover help, no-op, cooldown gating"
```

---

## Task 16: Add player `DeleteCharacter` command

**Files:**
- Create: `internal/usercommands/deletecharacter.go`
- Create: `_datafiles/world/dogmud/templates/help/deletecharacter.template`
- Modify: `internal/usercommands/usercommands.go`

- [ ] **Step 1: Write the help template**

Create `_datafiles/world/dogmud/templates/help/deletecharacter.template`:

```
<ansi fg="red-bold">deletecharacter</ansi>

Permanently deletes your account. Your character, gear, gold, bank,
companions, and quest progress are all lost. Your name will become
available for someone else to claim.

This cannot be undone.

You will be asked to confirm twice:
1. <ansi fg="51">yes</ansi> / <ansi fg="51">no</ansi> (defaults to no)
2. Type your character's name exactly to proceed.

Anything else aborts the deletion.
```

- [ ] **Step 2: Implement the command handler**

Create `internal/usercommands/deletecharacter.go`:

```go
package usercommands

import (
    "fmt"

    "github.com/GoMudEngine/GoMud/internal/events"
    "github.com/GoMudEngine/GoMud/internal/mudlog"
    "github.com/GoMudEngine/GoMud/internal/rooms"
    "github.com/GoMudEngine/GoMud/internal/users"
)

// DeleteCharacter permanently deletes the player's account. Two-stage
// confirmation: yes/no(default no) followed by case-sensitive
// type-the-name match.
func DeleteCharacter(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    cmdPrompt, _ := user.StartPrompt(`deletecharacter`, rest)

    q1 := cmdPrompt.Ask(
        `<ansi fg="red">This will permanently delete your account and free your name for someone else to claim. This cannot be undone.</ansi> Continue?`,
        []string{`yes`, `no`}, `no`)
    if !q1.Done {
        return true, nil
    }
    if q1.Response != `yes` {
        user.SendText(`Aborted.`)
        user.ClearPrompt()
        return true, nil
    }

    q2 := cmdPrompt.Ask(
        fmt.Sprintf(`To confirm, type your character's name exactly: <ansi fg="username">%s</ansi>`, user.Character.Name),
        []string{})
    if !q2.Done {
        return true, nil
    }
    if q2.Response != user.Character.Name { // case-sensitive
        user.SendText(`That doesn't match. Aborted.`)
        user.ClearPrompt()
        return true, nil
    }

    oldName := user.Character.Name
    user.EventLog.Add(`char`, `Account deleted by user.`)

    room.SendTextVisual(
        fmt.Sprintf(`<ansi fg="username">%s</ansi>'s form dissolves into shimmering dust.`, oldName),
        user.UserId)
    user.SendText(fmt.Sprintf(`Your form dissolves into shimmering dust. Farewell, <ansi fg="username">%s</ansi>.`, oldName))

    if err := users.RemoveUserAndDisconnect(user.UserId); err != nil {
        mudlog.Error(`deletecharacter`, `error`, err, `userId`, user.UserId)
    }

    return true, nil
}
```

- [ ] **Step 3: Register the command**

In `internal/usercommands/usercommands.go`, add near the `d` block (alphabetical):

```go
`deletecharacter`: {DeleteCharacter, false, true, false}, // Player account deletion
```

- [ ] **Step 4: Verify build**

```bash
go build ./...
```

- [ ] **Step 5: Commit**

```bash
git add internal/usercommands/deletecharacter.go internal/usercommands/usercommands.go _datafiles/world/dogmud/templates/help/deletecharacter.template
git commit -m "feat(deletecharacter): add player account-delete command

Two-stage confirmation: yes/no default-no, then case-sensitive
type-the-name. Calls users.RemoveUserAndDisconnect on success."
```

---

## Task 17: Add tests for `DeleteCharacter`

**Files:**
- Create: `internal/usercommands/deletecharacter_test.go`

- [ ] **Step 1: Write the tests**

```go
package usercommands

import (
    "os"
    "path/filepath"
    "strconv"
    "strings"
    "testing"

    "github.com/GoMudEngine/GoMud/internal/configs"
    "github.com/GoMudEngine/GoMud/internal/events"
)

func TestDeleteCharacter_FirstGateNo_Aborts(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    answerPrompt(t, user, "no") // first gate

    DeleteCharacter("", user, room, events.EventFlag(0))
    if !strings.Contains(strings.ToLower(user.LastSentText()), "abort") {
        t.Errorf("expected abort message, got %q", user.LastSentText())
    }
    if !userFileExists(t, user.UserId) {
        t.Error("user file should still exist after first-gate abort")
    }
}

func TestDeleteCharacter_SecondGateWrongName_Aborts(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    answerPrompts(t, user, []string{"yes", "Bob"}) // first yes, second wrong

    DeleteCharacter("", user, room, events.EventFlag(0))
    if !strings.Contains(strings.ToLower(user.LastSentText()), "doesn't match") {
        t.Errorf("expected mismatch message, got %q", user.LastSentText())
    }
    if !userFileExists(t, user.UserId) {
        t.Error("user file should still exist after wrong-name abort")
    }
}

func TestDeleteCharacter_BothGatesPass_DeletesFile(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    answerPrompts(t, user, []string{"yes", "Alice"})

    DeleteCharacter("", user, room, events.EventFlag(0))
    if userFileExists(t, user.UserId) {
        t.Error("user file should be deleted")
    }
}

func TestDeleteCharacter_CaseSensitiveNameMatch(t *testing.T) {
    user, room := newTestUserAndRoom(t, "Alice")
    answerPrompts(t, user, []string{"yes", "alice"}) // wrong case

    DeleteCharacter("", user, room, events.EventFlag(0))
    if !userFileExists(t, user.UserId) {
        t.Error("user file should still exist when case mismatches")
    }
}

func userFileExists(t *testing.T, userId int) bool {
    t.Helper()
    path := filepath.Join(string(configs.GetFilePathsConfig().DataFiles), "users", strconv.Itoa(userId)+".yaml")
    _, err := os.Stat(path)
    return err == nil
}
```

The `answerPrompt` / `answerPrompts` helpers must be implemented to drive the prompt framework. Mirror the approach in any existing multi-step prompt test (search the package for `cmdPrompt.Ask` test patterns).

- [ ] **Step 2: Run the tests**

```bash
go test ./internal/usercommands/ -run TestDeleteCharacter -v
```

Expected: pass.

- [ ] **Step 3: Commit**

```bash
git add internal/usercommands/deletecharacter_test.go
git commit -m "test(deletecharacter): cover both gates + case sensitivity"
```

---

## Task 18: Add `mobs.AuditMobNameCollisions` + tests

**Files:**
- Modify: `internal/mobs/mobs.go`
- Modify: `internal/mobs/mobs_test.go`

- [ ] **Step 1: Write the failing test**

Append to `internal/mobs/mobs_test.go`:

```go
func TestAuditMobNameCollisions_NoCollisions(t *testing.T) {
    seedMobName(t, "Goblin")

    callCount := 0
    AuditMobNameCollisions(func(name string) (int, bool) {
        callCount++
        return 0, false
    })
    if callCount == 0 {
        t.Error("expected lookup to be invoked at least once")
    }
}

func TestAuditMobNameCollisions_OneCollision(t *testing.T) {
    seedMobName(t, "Goblin")

    found := false
    AuditMobNameCollisions(func(name string) (int, bool) {
        if name == "Goblin" {
            found = true
            return 42, true
        }
        return 0, false
    })
    if !found {
        t.Error("expected lookup to be called with 'Goblin'")
    }
    // We don't assert on log output (mudlog has no test sink built in);
    // the lookup-call assertion is enough to prove the audit walked the
    // mob-name list.
}
```

If `seedMobName` already exists from Task 5, reuse it.

- [ ] **Step 2: Run to verify failure**

```bash
go test ./internal/mobs/ -run TestAuditMobNameCollisions -v
```

Expected: undefined.

- [ ] **Step 3: Implement the audit**

Append to `internal/mobs/mobs.go`:

```go
// AuditMobNameCollisions scans loaded mob template names against the
// supplied playerNameLookup and warns on each collision. Warn-only —
// never blocks startup. Called once after LoadDataFiles completes and
// the user index is reachable. Dependency injection keeps this package
// free of a users-package import.
func AuditMobNameCollisions(playerNameLookup func(name string) (userId int, ok bool)) {
    mobsMu.RLock()
    names := make([]string, len(allMobNames))
    copy(names, allMobNames)
    mobsMu.RUnlock()

    collisions := 0
    for _, mobName := range names {
        if userId, ok := playerNameLookup(mobName); ok {
            mudlog.Warn("mob/player name collision",
                "mobName", mobName,
                "playerUserId", userId,
                "advice", "rename mob template or notify player to use rename command")
            collisions++
        }
    }
    if collisions > 0 {
        mudlog.Warn("mob name collision audit complete", "collisions", collisions)
    }
}
```

Confirm `mudlog` is already imported. Confirm `mobsMu` and `allMobNames` are the actual package-level identifiers (they were referenced earlier in the file).

- [ ] **Step 4: Run tests**

```bash
go test ./internal/mobs/ -run TestAuditMobNameCollisions -v
```

Expected: pass.

- [ ] **Step 5: Commit**

```bash
git add internal/mobs/mobs.go internal/mobs/mobs_test.go
git commit -m "feat(mobs): warn-only audit for mob/player name collisions

Walks the loaded mob template name list and invokes a caller-supplied
lookup to detect player collisions. Logs one warn per hit plus a
summary. Dependency injection avoids importing users from this
package (which already imports mobs)."
```

---

## Task 19: Wire the audit into the boot sequence

**Files:**
- Modify: `main.go`

- [ ] **Step 1: Locate the loader function**

The mob load happens in a function that runs `mobs.LoadDataFiles()` (around line 1095). Confirm:

```bash
grep -n "mobs.LoadDataFiles" main.go
sed -n '1080,1110p' main.go
```

- [ ] **Step 2: Add the audit call after `mobs.LoadDataFiles()`**

Insert after line ~1095:

```go
mobs.LoadDataFiles()
mobs.AuditMobNameCollisions(func(name string) (int, bool) {
    if userId, _ := users.CharacterNameSearch(name); userId > 0 {
        return userId, true
    }
    return 0, false
})
```

Confirm `users` is already imported in `main.go`. If not, add it.

- [ ] **Step 3: Build**

```bash
go build ./...
```

Expected: clean.

- [ ] **Step 4: Smoke test — start the server briefly**

```bash
./gomud --help
# or whatever the run command is; the goal is to confirm no startup panics
```

If you can run a quick local boot, confirm the audit warn lines appear (or don't, depending on whether collisions exist in your test data).

- [ ] **Step 5: Commit**

```bash
git add main.go
git commit -m "feat(boot): wire mob/player name collision audit

Runs after mobs.LoadDataFiles. Uses users.CharacterNameSearch
(which scans offline users on disk, no init required) as the
lookup."
```

---

## Task 20: Investigate and clean up the `start.go` username-equality guard

**Files:**
- Modify: `internal/usercommands/start.go` (potentially)

- [ ] **Step 1: Read the guard**

```bash
sed -n '40,60p' internal/usercommands/start.go
```

The guard rejects character names that match the user's `Username`. With Username == Character.Name being the operative model in DOGMud, this guard either:
(a) is dead code — `Username` is set first to the same value the player picks, so the guard always fires only on a true conflict (which can't happen in this flow), or
(b) actively breaks character creation when Username is pre-populated to the chosen name.

- [ ] **Step 2: Check what sets Username before start.go runs**

```bash
grep -n "u.Username = \|user.Username = \|.Username =" internal/inputhandlers/login_prompt_handler.go internal/inputhandlers/login.go | head -10
```

Determine: when start.go runs, does `user.Username` equal a different value (placeholder/temp), or has the player already locked in their final name during signup?

- [ ] **Step 3: Decide**

- If signup picks a different value (temp username, then start.go picks the real one), the guard is meaningful — keep it. Document the model in a 1-line comment.
- If signup already picks the final name and start.go is a one-time confirmation, the guard is a footgun — remove it. Add a comment in the commit message explaining.

- [ ] **Step 4: Apply the chosen action**

- Keep + comment, OR
- Remove the `EqualFold(question.Response, user.Username)` check.

- [ ] **Step 5: Build + run any start.go-touching tests**

```bash
go build ./...
go test ./internal/usercommands/ -run Start -v
```

- [ ] **Step 6: Commit**

```bash
git add internal/usercommands/start.go
git commit -m "chore(start): clarify username-equality guard

[Either: 'remove vestigial guard — Username is locked in by signup
before this code runs' OR 'document why the guard remains needed —
signup picks a temp name and start.go locks in the final one']"
```

---

## Task 21: Manual smoke test on local server

**Files:** none (manual verification)

- [ ] **Step 1: Build a fresh local binary**

```bash
go build -o gomud-local
```

- [ ] **Step 2: Run through the smoke checklist from the spec**

1. Try to create a character named "Goblin" — fails (mob collision; pre-existing protection).
2. Create char "Calabean", `rename Bobblesworth`, verify room broadcast + `who` reflects the new name.
3. Try a second `rename` immediately — refused with cooldown message.
4. Edit `_datafiles/config.yaml`, set `CharacterRenameCooldownHours: 0`, restart, rename twice — both succeed.
5. Have one character name a companion "Bob"; try to create a second char named "Bob" — refused.
6. Name a companion "Goblin" — refused (the bug fix from Task 10).
7. `deletecharacter`: bail at gate 1, bail at gate 2, succeed at gate 2; verify `_datafiles/users/{UserId}.yaml` is gone and the username is re-registerable.
8. Add a mob template whose name matches an existing player; restart; grep server logs for `mob/player name collision`.

- [ ] **Step 3: If all pass, commit any config-tuning notes**

If the smoke test reveals a default cooldown that feels wrong (too long for early access, too short for live), update `_datafiles/config.yaml` and commit:

```bash
git add _datafiles/config.yaml
git commit -m "chore(config): tune CharacterRenameCooldownHours after smoke test"
```

---

## Task 22: Update PATCH_NOTES.md and close the memory entry

**Files:**
- Modify: `PATCH_NOTES.md`
- Modify: `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/MEMORY.md`
- Delete: `~/.claude/projects/C--Users-Calabe-Davis-workspace-DOGMud/memory/project_name_collision_prevention.md`

- [ ] **Step 1: Add a PATCH_NOTES entry**

Append a dated block to `PATCH_NOTES.md` summarizing:
- New player commands `rename` (cooldown-gated) and `deletecharacter` (two-gate confirm).
- Centralized name validation; companion naming now blocks mob-name collisions.
- Boot-time warn for mob/player name collisions.
- Admin `rename` is now `renameitem`.

- [ ] **Step 2: Remove the row from MEMORY.md**

Open `MEMORY.md` and remove the row referencing
`project_name_collision_prevention.md` from the Features & Content table.

- [ ] **Step 3: Delete the backing memory file**

```bash
rm "C:\Users\Calabe Davis\.claude\projects\C--Users-Calabe-Davis-workspace-DOGMud\memory\project_name_collision_prevention.md"
```

- [ ] **Step 4: Commit**

```bash
git add PATCH_NOTES.md
git commit -m "docs: patch notes — name collision + rename/delete commands"
```

(MEMORY.md and the deleted memory file live outside the repo, so no git step for those.)

---

## Self-Review Notes

- **Spec coverage:** every component from the spec maps to a task —
  validator (T3-5), migrations (T6-10), `RenameUser` (T11),
  `RemoveUserAndDisconnect` (T12), admin migration (T13), `Rename`
  command (T14-15), `DeleteCharacter` command (T16-17), audit (T18),
  boot wiring (T19), guard cleanup (T20), smoke (T21), housekeeping
  (T22).
- **Type consistency:** `ValidateActorOpts` fields `SkipMobCheck`,
  `SkipBannedCheck`, `ExcludeUserId` are used identically across all
  task code blocks. `RenameUser`, `RemoveUserAndDisconnect`,
  `AuditMobNameCollisions` signatures match between definition and
  call sites.
- **Test helpers:** several tasks reference helpers
  (`seedOfflineUser`, `seedMobName`, `seedActiveUser`,
  `newTestUserAndRoom`, `answerPrompts`) that may not exist verbatim.
  Each task instructs the implementer to look at existing test
  patterns in the same package and write thin local helpers if no
  matching pattern exists. This is deliberate — the GoMud test scaffolding
  varies per package, and prescribing a single shape would be wrong.
