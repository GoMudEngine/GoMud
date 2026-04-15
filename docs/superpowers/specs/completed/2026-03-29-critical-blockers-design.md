# Critical Blockers — Design Spec (2026-03-29)

Three critical bugs from the prod feedback scrape, plus related fixes.

---

## 1. Death Loop in Shadow Realm

### Problem

A player died, was moved to the Shadow Realm (room 75, "Waiting room"),
but remained stuck in combat state. Both `go` (portal) and `quit` check
`Aggro != nil` and refuse if in combat. Server reboot did not fix it
(though Aggro is not persisted — the character was likely still loaded
in memory).

### Root Cause

In `DoCombat()` (`NewRound_DoCombat.go:26-31`), the execution order is:

1. `handlePlayerCombat()` — players attack mobs
2. `handleMobCombat()` — mobs attack players
3. `handleAffected()` — check for deaths, call `suicide`

Inside `handleMobVsPlayer()` (line 1165), reciprocal aggro is set on
the defending player with **no health check**:

```go
if defUser.Character.Aggro == nil {
    defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
}
```

A mob can re-aggro a player who hit 0 HP earlier in the same round,
before `handleAffected()` runs `suicide` to clear it. The player ends
up in the Shadow Realm with stale aggro.

### Fix — Three Layers

**Layer 1 — Root fix** (`NewRound_DoCombat_helpers.go:1165`):

Add a health guard to the reciprocal aggro assignment:

```go
if defUser.Character.Health > 0 && defUser.Character.Aggro == nil {
    defUser.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
}
```

**Layer 2 — Safety net** (`suicide.go`, after line 200):

Re-clear aggro after the room move, in case any other code path assigns
aggro to a dead/dying player:

```go
user.Character.Aggro = nil
```

**Layer 3 — Escape hatch** (`go.go:41-44`):

If the player is in the death recovery room, skip the combat check
entirely. A player in the Shadow Realm should always be allowed to leave:

```go
deathRoom := configs.GetSpecialRoomsConfig().DeathRecoveryRoom
if user.Character.Aggro != nil && user.Character.RoomId != int(deathRoom) {
    user.SendText("You can't do that! You are in combat!")
    return true, nil
}
```

### Files Changed

- `internal/hooks/NewRound_DoCombat_helpers.go` — health guard on line 1165
- `internal/usercommands/suicide.go` — post-move aggro clear
- `internal/usercommands/go.go` — death room escape hatch

---

## 2. Spell Script Integration (onMagic / onCast / onWait)

### Problem

Spell JS scripts define `onCast()`, `onWait()`, and `onMagic()` callbacks,
but these are never invoked during the player spell casting pipeline.
`resolveSpell()` dispatches purely by `EffectType` — spells with custom
script logic (fold-anchor, chrysalis-aid, summon spells) do nothing when
cast by players.

The scripting infrastructure (`TrySpellScriptEvent()` in
`internal/scripting/spell.go`) exists and works, but is only called from
mob cast paths (`internal/mobcommands/aid.go`, `internal/mobcommands/cast.go`).

### Affected Spells

- `fold-anchor` — set/recall anchor (no effect)
- `chrysalis-aid` — heal downed player (no effect)
- `summon-steppe-spirit` — summon wolf companion (no effect)
- `summon-hive-swarm` — summon swarm companion (no effect)
- Any future spell relying on script callbacks

### Fix — Wire Scripts into Player Cast Pipeline

Three integration points in `NewRound_DoCombat_helpers.go`:

**`onCast`** — When `CastingState` is first created (cast initiation),
call `TrySpellScriptEvent("onCast", ...)`. If the script returns `false`,
cancel the cast.

**`onWait`** — During fold accumulation (each round the cast ticks),
call `TrySpellScriptEvent("onWait", ...)`.

**`onMagic`** — At the end of `resolveSpell()` in `spell_resolution.go`,
after all engine effect-type processing, call
`TrySpellScriptEvent("onMagic", ...)`. This lets script spells layer on
top of engine effects or serve as the sole effect.

### Fold Anchor Split — Two Spells

The current fold-anchor toggles between set and recall based on state.
Split into two explicit spells for clarity:

**`fold-anchor`** (set mode):
- `onMagic()` stores current room ID in `fold-anchor-room` via
  `SetMiscCharacterData`
- Flavor text about weaving a Chrysalis anchor

**`fold-recall`** (recall mode):
- `onMagic()` reads `fold-anchor-room` and calls `MoveRoom()`
- If no anchor is set, fails gracefully with a message
- Same base_folds (6), school (enhancement), stat (willpower)

**New files:**
- `_datafiles/world/dogmud/spells/fold-recall.yaml`
- `_datafiles/world/dogmud/spells/fold-recall.js`
- `_datafiles/world/dogmud/templates/help/fold-recall.template`

**Updated files:**
- `_datafiles/world/dogmud/spells/fold-anchor.yaml` — update description
- `_datafiles/world/dogmud/spells/fold-anchor.js` — set-only logic
- `_datafiles/world/dogmud/templates/help/fold-anchor.template` — update

### Paired Spell Learning

When a player learns either `fold-anchor` or `fold-recall`, both are
granted automatically. Implemented via a hardcoded map in `character.go`:

```go
var pairedSpells = map[string]string{
    "fold-anchor": "fold-recall",
    "fold-recall": "fold-anchor",
}
```

In `LearnSpell()`, after successfully learning a spell, check the map
and learn the paired spell if not already known.

### One-Time Migration

On character load, check if the player knows `fold-anchor` but not
`fold-recall` (or vice versa). If so, grant the missing spell. Gate
with a `MiscData` flag (`migration-fold-pair-done`) so it runs once
per character.

### Files Changed

- `internal/hooks/spell_resolution.go` — call `TrySpellScriptEvent("onMagic")`
- `internal/hooks/NewRound_DoCombat_helpers.go` — call `onCast` and `onWait`
- `internal/characters/character.go` — paired spell map + LearnSpell update
- `internal/characters/character.go` (or load path) — migration logic
- Fold-anchor/recall YAML, JS, and help template files (new + updated)

---

## 3. Quest Spell Rewards + Fetish Gating

### Problem A — No Spell Rewards

The `QuestReward` struct supports gold, items, buffs, skills, and room
moves — but not spells. Quest 12 (The Warden's Covenant) should award
`summon-steppe-spirit` on completion but has no mechanism to do so.

### Problem B — Infinite Fetishes

In `241-windwarden_sylara.js`, after quest completion, every `ask`
about fetishes gives a free Spirit Fetish (item 40031) with no limit.

### Fix A — Add SpellId to QuestReward

**`internal/quests/quests.go`** — Add field to `QuestReward`:

```go
type QuestReward struct {
    QuestId       string
    Gold          int
    ItemId        int
    BuffId        int
    SkillInfo     string
    SpellId       string // spell to teach on quest completion
    PlayerMessage string
    RoomMessage   string
    RoomId        int
}
```

**`internal/hooks/Quest_HandleQuestUpdate.go`** — After the skill reward
block (~line 151), add spell reward handling:

```go
if questInfo.Rewards.SpellId != "" {
    if questUser.Character.LearnSpell(questInfo.Rewards.SpellId) {
        if spellData := spells.GetSpell(questInfo.Rewards.SpellId); spellData != nil {
            questUser.SendText(fmt.Sprintf(
                `<ansi fg="magenta-bold">You have learned the spell: `+
                `<ansi fg="cyan-bold">%s</ansi></ansi>`,
                spellData.Name))
        }
    }
}
```

**Quest 12 YAML** — Add spell reward:

```yaml
rewards:
  spellid: summon-steppe-spirit
  gold: 15
  itemid: 40031
```

### Fix B — Fetish Inventory Gating

In `241-windwarden_sylara.js`, the "subsequent asks" branch: before
giving a fetish, check if the player already has item 40031 in their
backpack OR equipped. If they do, Sylara refuses:

```javascript
// Check backpack and equipped items for existing fetish
if (user.HasItem(FETISH_ITEM_ID) || user.HasEquipped(FETISH_ITEM_ID)) {
    mob.Command('say You already carry a spirit fetish. Use it wisely.');
    return true;
}
```

(Exact API may need adjustment based on available scripting functions —
verify `HasItem` / inventory check methods available to JS scripts.)

### Files Changed

- `internal/quests/quests.go` — add `SpellId` field
- `internal/hooks/Quest_HandleQuestUpdate.go` — handle spell rewards
- `_datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml` — add spellid
- `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js` — fetish gating

---

## Summary of All Files Changed

| File | Change |
|------|--------|
| `internal/hooks/NewRound_DoCombat_helpers.go` | Health guard on reciprocal aggro; onCast/onWait script calls |
| `internal/usercommands/suicide.go` | Post-move aggro clear |
| `internal/usercommands/go.go` | Death room escape hatch |
| `internal/hooks/spell_resolution.go` | Call onMagic via TrySpellScriptEvent |
| `internal/characters/character.go` | Paired spell map, LearnSpell update, migration |
| `internal/quests/quests.go` | SpellId field on QuestReward |
| `internal/hooks/Quest_HandleQuestUpdate.go` | Spell reward handling |
| `_datafiles/world/dogmud/spells/fold-anchor.yaml` | Updated description |
| `_datafiles/world/dogmud/spells/fold-anchor.js` | Set-only logic |
| `_datafiles/world/dogmud/spells/fold-recall.yaml` | New spell |
| `_datafiles/world/dogmud/spells/fold-recall.js` | New script |
| `_datafiles/world/dogmud/templates/help/fold-anchor.template` | Updated |
| `_datafiles/world/dogmud/templates/help/fold-recall.template` | New |
| `_datafiles/world/dogmud/quests/12-the_wardens_covenant.yaml` | Add spellid reward |
| `_datafiles/world/dogmud/mobs/ironwind_steppe/scripts/241-windwarden_sylara.js` | Fetish inventory gating |
