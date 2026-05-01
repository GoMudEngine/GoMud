# Name Collision Prevention + Player Character Management

**Date:** 2026-04-25
**Status:** Design (approved)
**Logged followups closed:** project_name_collision_prevention.md

## Problem

The MUD's name space is shared across players, alts, mob templates, and
companions. The 2026-04-18 target-resolution refactor added a "players win
over mobs" tiebreak inside `room.ResolveTargetActor` as a workaround for
player↔mob name collisions, but the underlying ambiguity was unresolved.

Concurrently, two player-quality-of-life gaps exist:

- No way for a player to rename their character (e.g., to fix a regretted
  name or work around a collision-prone name).
- No way for a player to delete their account.

Companion-name validation also has a known gap: `validateCompanionName`
does not check against mob template names, while pet naming and character
creation do.

## Goals

1. Centralize the duplicated name-validation logic into a single function
   so future call sites (and existing ones) cannot drift.
2. Surface mob-template/player name collisions to operators at server
   startup via a warn-only audit.
3. Add a player `rename` command (with cooldown, opt-out-default
   confirmation) so any player can self-service a name conflict.
4. Add a player `deletecharacter` command with two-stage confirmation
   (yes/no default-no, then case-sensitive name typing) so accounts can be
   permanently deleted with strong friction.

## Non-Goals

- Reserved-words list (the live mob-template scan covers the common case
  and a denylist would duplicate maintenance burden).
- Actor disambiguation syntax (`2.goblin`) — out of scope; left for a
  future session.
- Hard-blocking startup on mob/player collisions — warn-only by design;
  prod data may already contain collisions and the rename command gives
  affected players a self-service fix.
- Soft-delete with admin-restorable accounts — overkill for current scale.
- Companion-name graveyard — locking historical names would force players
  to keep inventing new ones; cycle is desirable.
- Surfacing the existing alt-character menu — DOGMud has no
  `IsCharacterRoom` in world data, so the alt menu is unreachable; this
  design treats one-character-per-account as the operative model.

## Architecture

```
┌─────────────────────────────────────────────────────────────┐
│  users.ValidateActorName(name, opts)   [NEW, centralized]   │
│   ├─ length / regex / banned-pattern                        │
│   ├─ player-name collision (online + offline + alts)        │
│   ├─ companion-nickname collision                           │
│   ├─ mob-template-name collision                            │
│   └─ opts: SkipMobCheck, SkipBannedCheck, ExcludeUserId     │
└─────────────────────────────────────────────────────────────┘
        ▲                ▲                ▲              ▲
        │                │                │              │
   ┌────┴───┐     ┌──────┴─────┐    ┌─────┴────┐   ┌─────┴─────┐
   │ login  │     │  start     │    │ rename   │   │ companion │
   │ prompt │     │  (tutorial)│    │          │   │ /pet      │
   └────────┘     └────────────┘    └──────────┘   └───────────┘
                                          │
                                          ▼
                              UserRecord.LastRenameAt
                              (time.Time, persisted)

┌──────────────────────────────────────────────────┐
│  mobs.AuditMobNameCollisions    [NEW, warn-only] │
│   For each template name:                        │
│     if matches existing player → mudlog.Warn     │
└──────────────────────────────────────────────────┘

┌──────────────────────────────────────┐
│  deletecharacter command   [NEW]     │
│   yes/no(default no) → type-the-name │
│   → users.RemoveUserAndDisconnect    │
└──────────────────────────────────────┘
```

### Files

**New:**
- `internal/users/validate_actor_name.go`
- `internal/users/validate_actor_name_test.go`
- `internal/usercommands/renameself.go` (registered as `rename`)
- `internal/usercommands/renameself_test.go`
- `internal/usercommands/deletecharacter.go` (registered as `deletecharacter`)
- `internal/usercommands/deletecharacter_test.go`

**Modified:**
- `internal/users/users.go` — add `RenameUser`, `RemoveUserAndDisconnect`,
  refactor `ValidateName` to call `ValidateActorName`.
- `internal/users/userrecord.go` — add `LastRenameAt time.Time` field.
- `internal/usercommands/start.go` — replace inline checks with one
  `ValidateActorName` call.
- `internal/inputhandlers/login_prompt_handler.go` — replace inline checks
  in `ValidateCharacterName` with `ValidateActorName`.
- `internal/usercommands/pet.go` — replace inline checks with
  `ValidateActorName`.
- `internal/usercommands/companion.go` — replace `validateCompanionName`
  body with `ValidateActorName` call (this fixes the missing mob-name
  check as a free byproduct).
- `internal/usercommands/usercommands.go` — register `rename` (player),
  `renameitem` (former admin `rename`), `deletecharacter`.
- `internal/usercommands/admin.rename.go` — no functional changes; the
  function is now registered under `renameitem` instead of `rename`.
- `internal/mobs/mobs.go` — add `AuditMobNameCollisions`.
- `main.go` (or wherever the boot sequence orchestrates mob+user load) —
  call `mobs.AuditMobNameCollisions` after both are loaded.
- `_datafiles/config.yaml` — add
  `Balance.CharacterRenameCooldownHours: 168`.
- `internal/configs/config.balance.go` — add
  `CharacterRenameCooldownHours ConfigInt`.

## Component: Centralized Name Validator

```go
// internal/users/validate_actor_name.go

type ValidateActorOpts struct {
    SkipMobCheck    bool // true for mob loaders that need raw checks
    SkipBannedCheck bool // future-proofing; unused today
    ExcludeUserId   int  // ignore collisions on this user (self-rename)
}

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
    if userId, ok := userManager.Usernames[strings.ToLower(name)]; ok && userId != opts.ExcludeUserId {
        return errors.New("that name is already in use")
    }

    if CompanionNameExists(name) {
        return errors.New("that name is in use by a companion")
    }

    return nil
}
```

### Migration of Existing Call Sites

| Call site | Before | After |
|-----------|--------|-------|
| `users.ValidateName` | inline checks | `ValidateActorName(name, {})` |
| `start.go` | five inline checks | one call |
| `login_prompt_handler.go` `ValidateCharacterName` | three inline checks | one call (keeps the username-equality guard, which becomes a no-op once namespaces are merged) |
| `pet.go` | three inline checks | one call |
| `companion.go` `validateCompanionName` | missing mob check | one call (fixes the gap) |

## Component: Mob-Side Startup Audit

```go
// internal/mobs/mobs.go

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

Wired into the boot sequence after both mob templates and the user index
are loaded:

```go
mobs.AuditMobNameCollisions(func(name string) (int, bool) {
    if userId, found := users.CharacterNameSearch(name); found > 0 {
        return userId, true
    }
    return 0, false
})
```

Dependency injection keeps `mobs` free of a `users` import (avoids cycle
risk; `users` already imports `mobs`).

## Component: `rename` Command

**Surface:** top-level `rename <newname>`. The existing admin `rename`
moves to `renameitem`.

```go
func Rename(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
    newName := strings.TrimSpace(rest)
    if newName == `` {
        infoOutput, _ := templates.Process("usercommands/help/command.rename", nil, user.UserId)
        user.SendText(infoOutput)
        return true, nil
    }

    if strings.EqualFold(newName, user.Character.Name) {
        user.SendText(`That's already your name.`)
        return true, nil
    }

    cooldownHours := configs.GetBalanceConfig().CharacterRenameCooldownHours
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
    q := cmdPrompt.Ask(
        fmt.Sprintf(`Rename yourself from <ansi fg="username">%s</ansi> to <ansi fg="username">%s</ansi>? You won't be able to rename again for %d days.`,
            user.Character.Name, newName, cooldownHours/24),
        []string{`yes`, `no`}, `no`)
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
        return true, nil
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

### `users.RenameUser`

User save files on disk are keyed by `UserId` (int), not Username
(`{datafiles}/users/{UserId}.yaml`), so no disk-file rename is required.
The caller's subsequent `SaveUser` writes the new Username+Character.Name
into the same file.

```go
// RenameUser atomically updates Username + Character.Name + the username
// index. Caller is responsible for saving and setting LastRenameAt.
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
    u.Character.Name = newName

    idx.RemoveUser(oldName)
    idx.AddUser(u.UserId, newName)

    return nil
}
```

### Config Addition

```yaml
# _datafiles/config.yaml
Balance:
  CharacterRenameCooldownHours: 168  # 7 days; 0 disables cooldown
```

## Component: `deletecharacter` Command

```go
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

### `users.RemoveUserAndDisconnect`

```go
func RemoveUserAndDisconnect(userId int) error {
    u := GetByUserId(userId)
    if u == nil {
        return errors.New("user not found")
    }

    for _, mobInstanceId := range u.Character.GetCharmIds() {
        if m := mobs.GetInstance(mobInstanceId); m != nil {
            m.Character.RemoveCharm()
        }
    }

    LogOutUserByConnectionId(u.ConnectionId)

    // User files are keyed by UserId on disk, matching SaveUser's path.
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

### Edge Cases

| Case | Behavior |
|------|----------|
| User in combat | Combat ends naturally on logout. |
| User has companions | `Character.Companions` is part of the file; gone with deletion. |
| User has charmed mobs | Uncharmed before logout; mobs roam free. |
| User in shop/storage prompt | `LogOutUserByConnectionId` clears prompts. |
| User has gold/items | All gone — that is the point. |
| Inbox messages to deleted user | Existing offline-user delivery silently no-ops. |
| Concurrent rename to same name | `RenameUser` holds `userManagerMu`; second call sees populated index and returns "name was just claimed". |

## Testing

### Unit

**`internal/users/validate_actor_name_test.go`** (new)

| Case | Expectation |
|------|-------------|
| Empty / too-short / too-long | error mentions length range |
| Banned-pattern match | error mentions pattern |
| Banned-pattern + `SkipBannedCheck: true` | passes |
| Mob-name collision | error |
| Mob-name collision + `SkipMobCheck: true` | passes |
| Player-name collision (offline char) | error |
| Player-name collision but `ExcludeUserId` matches | passes (self-rename) |
| Companion-nickname collision | error |
| Valid novel name | nil error |

**`internal/users/users_test.go`** (additions)

| Case | Expectation |
|------|-------------|
| `RenameUser` updates Usernames + idx + Character.Name + Username | all coherent |
| `RenameUser` to claimed name | error, no mutation |
| `RenameUser` followed by `SaveUser` | new Username persisted to existing `{UserId}.yaml` |
| `RemoveUserAndDisconnect` removes file + frees indexes | name re-claimable |
| `RemoveUserAndDisconnect` with charmed mobs | mobs uncharmed, not destroyed |

**`internal/mobs/mobs_test.go`** (additions)

| Case | Expectation |
|------|-------------|
| `AuditMobNameCollisions` no collisions | no warn logs |
| `AuditMobNameCollisions` one collision | one per-collision warn + summary warn |

**`internal/usercommands/renameself_test.go`** (new)

| Case | Expectation |
|------|-------------|
| No args | help template sent |
| Newname == current name | "That's already your name." |
| Within cooldown | error with next-rename time |
| Past cooldown but mob collision | rejection from validator |
| Valid rename, confirmation = no | aborted, no mutation |
| Valid rename, confirmation = yes | name updated, `LastRenameAt` set, room broadcast sent |
| Cooldown disabled (0) | no cooldown gate |

**`internal/usercommands/deletecharacter_test.go`** (new)

| Case | Expectation |
|------|-------------|
| Gate 1 = no | aborted, user still exists |
| Gate 1 = yes, gate 2 = wrong name | aborted, user still exists |
| Gate 1 = yes, gate 2 = correct | file deleted, username freed, connection closed |
| Gate 2 = correct name with different case | rejected (case-sensitive) |

### Integration / Smoke

Run on local server before merge:

1. Create char "Goblin" — fail (mob collision).
2. Create char "Calabean", rename to "Bobblesworth" — verify room broadcast + `who` output reflects new name.
3. Try second rename within cooldown — refused.
4. Set `CharacterRenameCooldownHours: 0`, restart, rename twice — both succeed.
5. Companion-name collision: name companion "Bob"; second player tries to create char "Bob" — refused.
6. Companion mob check (the bug fix): name companion "Goblin" — refused (was previously allowed).
7. Delete account: bail at gate 1, bail at gate 2, succeed at gate 2; verify file removed and username re-registerable.
8. Mob-side warning: add a mob template named after an existing player, restart, grep logs for the warning.

### Out of Scope for Tests

- High-volume concurrent renames — `userManagerMu` serializes the
  index swap.

## Open Questions Resolved During Brainstorm

- **Active-only vs. alts** → active only; DOGMud has no `IsCharacterRoom` exposed.
- **Char-only vs. account delete** → full account; merged namespace requires it to free names.
- **Confirmation strength** → yes/no(default no) for both, plus typed-name confirmation for delete.
- **Rename economics** → free + cooldown, configurable, default 168 hours.
- **Surface name `delete`** → `deletecharacter` (explicit, foot-gun-resistant).
- **Surface name `rename`** → migrate admin `rename` to `renameitem`; players get `rename`.

## Implementation Investigation Notes

These resolve during plan-writing or first implementation step:

- **`start.go` username-equality guard.** The tutorial currently rejects a
  character name that matches the user's `Username`. With merged
  namespaces (Username == Character.Name post-signup), this guard is at
  best vestigial and at worst will brick character creation. Plan-writer
  should locate the current signup→tutorial flow, confirm whether the
  guard still has a real purpose, and remove it if not.
- **Boot sequence wiring point.** The exact location to call
  `mobs.AuditMobNameCollisions` after both mob templates and the user
  index are loaded — find the orchestrating function (likely in `main.go`
  or `internal/world/`).

## Memory Cleanup

After this work merges, the entry
`project_name_collision_prevention.md` is closed. Update `MEMORY.md` to
remove the row from the "Features & Content" table and delete the
backing memory file.
