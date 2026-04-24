# Sanctum Basin Mob Audit — Design

## Goal

Migrate all 17 mobs in the Sanctum Basin tutorial zone to the behavior
archetype system, and use the audit as a vehicle to deliver prod-safe
(LLM-free) tutorial content that introduces new players to recently-
added gameplay systems (salvage, bandolier/grenades, rally/warcry,
mutations, companions, spell discovery, etc.).

Sanctum Basin is the first area a new player experiences. Nailing the
player-facing tutorial experience here sets the tone for the whole game.

This work establishes the template for auditing other zones.

## Scope

**In scope:**
- 4 new behavior archetypes with generic role-shaped handlers
- 17 mob YAML updates (assign `behavior_archetype`, add
  `attack_reject_emote`, fix `non_combatant` flags where needed)
- 5 new dialogue YAML files + 4 existing dialogue YAML files updated
  with tutorial content patterns
- New btree event `player_attack_rejected` fired from attack.go and
  cast.go (HarmSingle only)
- New `Mob.LastAttackRejectedRound` field + round-based dedupe
- Audit template (implicit — the work pattern becomes reusable for
  subsequent zones)

**Out of scope:**
- LLM systemprompt updates (LLM is dev-only, not running in prod)
- Other zones (thornwall_city, marches_spur_road, etc. — separate
  future audits)
- HarmMulti non-combatant filtering (not an observed bug; follow-up if
  surfaced)
- New gear-breaking mechanic (salvage stays material-recovery-only)
- Engine changes to make archetype + per-mob btree composable (acceptable
  duplication for now; revisit if painful)

## Context

### Existing dialogue system (prod-safe, LLM-independent)

`internal/usercommands/ask.go` dispatches player asks in this order:
1. BTree `player_ask` event — short-circuits if handled
2. LLM path (dev only; disabled in prod)
3. Dialogue YAML fallback — `deliverDialogue` at ask.go:23 runs when
   LLM path doesn't fire or times out

Dialogue YAML at `_datafiles/world/dogmud/dialogue/<zone>/<mobid>.yaml`
supports:
- `patterns` — keyword-matched responses with `mood_change`
- `tree` — root intro text + quest-aware nodes with triggers,
  requirements, quest grants

**Key insight:** the dialogue YAML is the deterministic prod delivery
mechanism. Keyword patterns are how we deliver tutorial content
reliably.

### Existing archetype system (post-pack-tactics-revamp)

Mob YAML field `behavior_archetype: <name>` resolves to
`_datafiles/world/dogmud/behaviors/archetypes/<name>.yaml`. Per-mob
files at `behaviors/<zone>/<mobid>-<name>.yaml` REPLACE (not extend)
the archetype.

Seven archetypes exist: `generic_fighter`, `tank_taunter`,
`melee_self_buff`, `pure_caster`, `lookout`, `support_caster`,
`leader`. All are combat-oriented.

## Design

### Architecture layers

| Layer | What it does | Where it lives |
|-------|--------------|----------------|
| Archetype btree | Generic role-shaped handlers (enter, give, attack-rejected) | `behaviors/archetypes/*.yaml` |
| Per-NPC dialogue | Deterministic keyword → canned response | `dialogue/sanctum_basin/*.yaml` |
| Mob YAML | Archetype assignment + per-NPC attack-rejection emote | `mobs/sanctum_basin/*.yaml` |
| Engine | `player_attack_rejected` btree event + dedupe | `internal/usercommands/attack.go`, `internal/actions/cast.go`, `internal/mobs/mobs.go` |

### Layer 1: Four new archetypes

#### `noncombat_questgiver`

Used by non-combat NPCs with dialogue/tutorial roles. Currently sanctum_basin:
chrysalis_priest, combat_trainer, wilderness_guide_fen, elder_saris,
basin_warden, basin_scholar.

```yaml
# behaviors/archetypes/noncombat_questgiver.yaml
tree:
  type: selector
  children:
    - type: action
      event: player_enter
      do: emote
      text: "{mob_name} glances up as you enter."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "{mob_name} raises an eyebrow but says nothing."

    - type: sequence
      event: player_give
      children:
        - type: action
          do: emote
          text: "{mob_name} declines politely and hands it back."
        - type: action
          do: return_item
```

**No `player_ask` handler** — ask goes through btree FIRST but returns
Failure here, falls through to dialogue YAML which handles the keyword
patterns. This is deliberate.

#### `noncombat_shopkeeper`

Used by merchants with shops. Sanctum Basin: korvath, alchemist_yenna,
merchant_adela.

Identical structure to `noncombat_questgiver` with shopkeeper-flavored
default emotes (e.g. "{mob_name} nods in greeting from behind the counter.").
The functional behavior is the same — classification value is future-
proofing (someday shopkeepers might get shop-related events).

#### `noncombat_passive`

Used by ambient non-interactive creatures that cannot be attacked.
Sanctum Basin: chrysalis_echo, meadow_lizard (needs `non_combatant: true`
fixup).

```yaml
# behaviors/archetypes/noncombat_passive.yaml
tree:
  type: selector
  children:
    # occasional ambient emote on player enter — engine caller controls frequency
    - type: action
      event: player_enter
      do: emote
      text: "{mob_name} pays you little mind."

    - type: action
      event: player_attack_rejected
      do: emote
      text: "{mob_name} slips just out of reach."
```

No give, no ask.

#### `combat_passive`

Used by things that can be hit but don't have a special-move tactical
tree. They still fight back with basic attacks via the default combat
loop — they just never bash/trip/grapple/kick/taunt/cast specials.

Sanctum Basin: training_dummy (mob 65) — has no per-mob btree file
today; gets just the `combat_passive` archetype. An unrelated training
dummy (mob 58, in `behaviors/tutorial/`) has a `mob_die` handler; that
file belongs to a different mob and is not touched by this spec.

```yaml
# behaviors/archetypes/combat_passive.yaml
# Intentionally empty tree. Mobs with this archetype fall through the
# combat loop to basic attacks without special-move selection. Value is
# documentation + classification.
tree:
  type: selector
  children: []
```

### Layer 2: Dialogue YAML updates

#### Per-NPC system-mention assignments

Each tutorial NPC owns a set of newer systems they mention naturally
through their keyword patterns. Reasoning: match the system to the
NPC's in-world role so mentions feel organic.

| NPC (id) | Primary role | Systems to cover |
|----------|--------------|------------------|
| Korvath (52) | Blacksmith + shop | Enchanting slot-targeting, 2H weapons, weapon maker's marks, salvage after failed craft |
| Alchemist Yenna (53) | Alchemy master + shop | Bandolier, grenades (flashbang/firebomb/toxic), potion aging + toxicity, salvage after failed craft |
| Merchant Adela (63) | General shop | Bartering/haggling, tavern gossip, encumbrance/carry capacity |
| Combat Trainer (51) | Melee training | Rally/warcry shouts, companions (summon/charm/conjure/raise), mutations from combat, position (prone/grapple) |
| Wilderness Guide Fen (54) | Tracking/foraging | Foraging for ingredients, pack tactics awareness, scent/track, fleeing |
| Elder Saris (55) | Senior spellcaster | **Spell discovery mechanic** (Per + skill reduce decay), spellcasting vs manifestation schools, mutations from repeated magic use |
| Basin Warden (56) | Patrol/guard | Pack routines, aggro + respawn grace, dungeon pacing |
| Basin Scholar (79) | Lore/history | Mutations (chrysalis lore), the Chrysalis Priest's role, zone navigation |
| Chrysalis Priest (50) | Religious/lore | Mutations as spiritual change, faith/conviction, spell discovery |

#### Dialogue YAML pattern structure (template)

For each NPC, add `patterns` entries for their system-mention keywords.
Example for Yenna (file `dialogue/sanctum_basin/53.yaml`, new):

```yaml
mobid: 53
zone: Sanctum Basin
defaultMood: neutral

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    responses:
      - "Three drops, not four. Yes? Yes. Welcome. Mind the table — some of the compounds stain."

  # System mention: bandolier
  - keywords: ["bandolier", "carry", "pouch", "belt"]
    responses:
      - "A potion bandolier keeps vials in reach without rooting through a pack. The belt routes any potion you pick up straight to a slot. Worth it if you drink mid-fight."

  # System mention: grenades
  - keywords: ["grenade", "flashbang", "firebomb", "toxic"]
    responses:
      - "Dangerous work. I brew flashbangs, firebombs, and toxic flasks for those who need them. They age like potions — do not hoard them."

  # System mention: potion aging + toxicity
  - keywords: ["potion", "aging", "toxicity", "spoiled", "old"]
    responses:
      - "Every potion has a peak. Fresh is weaker, peak is best, decay is already losing you. And drink too many in a day — your body will tell you. Toxicity stacks faster than most think."

  # System mention: salvage after failed craft
  - keywords: ["salvage", "wasted", "failed", "ruined"]
    responses:
      - "Not wasted. If you have the knack, you can pull reagents back from a bad batch. Ask the smiths; they do the same with bent metal."

  # Catch-all / fallback
  - keywords: [""]
    responses:
      - "Mm. Precision, that is the thing."
      - "Trail off. Snap back. Yes?"

tree:
  root:
    text: "I am Yenna. Alchemy, potions, reagents. Fen sent you for lessons? Good. Three drops, not four."
    hints: "You could ask about potions, grenades, bandolier, aging, or salvage."

  nodes:
    # Existing tutorial-progression nodes go here if/when quest engine wires them
```

**Template rules for all 9 NPC dialogue files:**
- Keep all spoken `responses` in first person (NPC speaks)
- Keep `hints` in second person (narrator → player: "You could ask about…")
- **Every system-mention keyword list must also appear in a hint** so players can discover it (feedback memory: every trigger word must be discoverable)
- Wrap prose at 80 chars
- No hard numbers in responses — describe feel/effect, not mechanics
- Include a catch-all `keywords: [""]` fallback so unmatched questions have a reply

#### Existing files to update (add patterns, don't remove existing content)

- `dialogue/sanctum_basin/50.yaml` (chrysalis_priest) — mutations + discovery + faith
- `dialogue/sanctum_basin/52.yaml` (korvath) — salvage + enchanting + 2H
- `dialogue/sanctum_basin/55.yaml` (elder_saris) — spell discovery + manifestation + mutations
- `dialogue/sanctum_basin/79.yaml` (basin_scholar) — mutations + chrysalis lore

#### New files to create

- `dialogue/sanctum_basin/51.yaml` (combat_trainer)
- `dialogue/sanctum_basin/53.yaml` (alchemist_yenna)
- `dialogue/sanctum_basin/54.yaml` (wilderness_guide_fen)
- `dialogue/sanctum_basin/56.yaml` (basin_warden)
- `dialogue/sanctum_basin/63.yaml` (merchant_adela)

### Layer 3: Attack-rejection btree event

#### New event: `player_attack_rejected`

Fired on the targeted mob when a player's attack/harm is rejected
because the target is flagged `non_combatant: true`.

Payload:
```go
EventContext{
    EventType: "player_attack_rejected",
    UserId:    user.UserId,
    MobId:     0,  // player was the actor, not a mob
}
```

#### Dedupe: round-based per-mob

Mirror the `LastSuicideRound` pattern from the respawn aggro cleanup fix
(2026-04-21 prod push).

Add to `internal/mobs/mobs.go` `Mob` struct:
```go
LastAttackRejectedRound uint64 // last round a player_attack_rejected event fired
```

In the firing call sites (attack.go, cast.go), check
`util.GetRoundCount() > mob.LastAttackRejectedRound` before firing;
update to current round when firing.

**Behavior**: at most one `player_attack_rejected` per mob per round.
Player still sees their individual rejection message ("You can't attack
X") every time they try. Only the *mob's* emote reaction is rate-limited.

#### Firing sites

**1. `internal/usercommands/attack.go:161-164`** (in scope)

```go
if m.IsNonCombatant() {
    user.SendText(fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, m.Character.Name))
    if util.GetRoundCount() > m.LastAttackRejectedRound {
        m.LastAttackRejectedRound = util.GetRoundCount()
        behaviortree.TryMobBehavior(m.InstanceId, behaviortree.EventContext{
            EventType: "player_attack_rejected",
            UserId:    user.UserId,
        })
    }
    return true, nil
}
```

**2. `internal/actions/cast.go:104-107`** (in scope)

Same pattern for HarmSingle rejection. Note: `actor` is the `Actor`
interface; fetch `user.UserId` via `actor.GetUserId()` (guarded by the
existing `actor.IsPlayer()` check at line 98).

**HarmArea (spell_resolution.go:75)**: no change. The silent filter is
the correct behavior — AoE harm shouldn't broadcast rejection emotes to
bystanders who happen to be non-combatants in the room. If they're
silently filtered, they never experience the "attack" in the first place.

**HarmMulti (cast.go:153-171)**: no change for this spec. HarmMulti
targets are resolved via named target or aggro fallback; a non-combatant
is an unlikely target and not a known bug. Follow-up memory can capture
this gap if it surfaces in playtesting.

### Layer 4: Mob YAML updates

#### `attack_reject_emote` field — NOT needed

Originally proposed a per-mob YAML field for the rejection emote. With
the btree event approach, this is replaced by the archetype's
`player_attack_rejected` handler.

For the pilot, all non-combat NPCs share their archetype's generic
`player_attack_rejected` emote:
- questgivers: "{mob_name} raises an eyebrow but says nothing."
- shopkeepers: same (via the shared pattern)
- passive ambient: "{mob_name} slips just out of reach."

This is acceptable baseline behavior. Per-NPC customization is NOT in
scope for this pilot. If playtesting reveals that specific NPCs need
personalized rejection reactions, a followup can add per-mob btree
files — note that per-mob files REPLACE (not extend) archetype, so a
custom file must duplicate the other archetype handlers (enter, give)
it wants to keep.

#### Mob YAML changes per-NPC

| Mob ID | Name | Changes |
|--------|------|---------|
| 50 | chrysalis_priest | `behavior_archetype: noncombat_questgiver` |
| 51 | combat_trainer | `behavior_archetype: noncombat_questgiver` |
| 52 | korvath | `behavior_archetype: noncombat_shopkeeper` |
| 53 | alchemist_yenna | `behavior_archetype: noncombat_shopkeeper` |
| 54 | wilderness_guide_fen | `behavior_archetype: noncombat_questgiver` |
| 55 | elder_saris | `behavior_archetype: noncombat_questgiver` |
| 56 | basin_warden | `behavior_archetype: noncombat_questgiver` |
| 63 | merchant_adela | `behavior_archetype: noncombat_shopkeeper` |
| 79 | basin_scholar | `behavior_archetype: noncombat_questgiver` |
| 65 | training_dummy | `behavior_archetype: combat_passive` |
| 66 | valley_rat | `behavior_archetype: generic_fighter` |
| 67 | cave_bat | `behavior_archetype: generic_fighter` |
| 68 | cave_goblin_guard | `behavior_archetype: generic_fighter` |
| 69 | aberrant_chrysalis | `behavior_archetype: generic_fighter` |
| 70 | cave_troll | `behavior_archetype: generic_fighter` |
| 112 | chrysalis_echo | `behavior_archetype: noncombat_passive` + add `non_combatant: true` (currently missing) |
| 71 | meadow_lizard | `behavior_archetype: noncombat_passive` + add `non_combatant: true` (currently missing) |

### Template emote text

Archetype emotes use `{mob_name}` placeholder — the existing emote
action substitutes the mob's displayed name at render time.

If `{mob_name}` substitution isn't already supported in the emote
action, this is a small engine addition. Verify during implementation;
if not supported, use generic phrasing ("The merchant raises an
eyebrow…") that reads naturally regardless of mob.

## Testing

### Unit tests
- `internal/mobs/mobs_test.go` — `LastAttackRejectedRound` round-based
  dedupe: fire twice in same round, second one is suppressed; fire in
  next round, fires again
- `internal/behaviortree/archetype_noncombat_questgiver_test.go` — load
  archetype + fire `player_enter`, `player_attack_rejected`, `player_give`
  → verify corresponding emotes/actions trigger
- Same test pattern for the other three new archetypes
- `internal/usercommands/attack_test.go` — attacking a non-combatant
  fires the btree event once and rejects player; second attempt same
  round does NOT re-fire btree event

### Integration / smoke (in-game)
- Log in as fresh character in Sanctum Basin
- Walk past each NPC, verify `player_enter` emote fires (or doesn't, if
  the engine's enter-event gating suppresses repeats)
- `attack yenna` → verify rejection message + Yenna's archetype emote
- `ask yenna about grenade` → verify system-mention response from
  dialogue pattern
- `ask korvath about salvage` → verify system-mention response
- `ask saris about discovery` → verify system-mention response
- `give yenna iron_ingot` → verify polite decline + item returned
- Attack each hostile combat mob → verify normal combat behaviors
  (generic_fighter archetype drives this — was already working for
  north_road/steppe mobs after pack-tactics revamp)
- Training dummy: attack, kill, verify `mob_die` emote still fires
  (existing per-mob btree preserved)

### Acceptance criteria
- All 17 sanctum_basin mobs have a `behavior_archetype` value
- Every non-combat NPC has a dialogue YAML with keyword patterns
  covering their assigned system mentions
- Every system-mention keyword is discoverable via a `hints` line
- `player_attack_rejected` fires at most once per mob per round
- No regression in existing combat behavior (5 hostile mobs still fight
  normally; training dummy still dies properly)
- LLM-off smoke test matches LLM-on behavior for the deterministic
  paths (we can temporarily disable LLM config to verify)

## Non-Goals (stated again, clearly)

- **LLM systemprompt updates**: not part of this spec. LLM is disabled
  in prod. llmprofiles stay on NPCs for dev testing only; their content
  does not drive the tutorial experience.
- **Per-mob btree files** (other than training_dummy's existing
  `mob_die` handler): archetype handlers cover the pilot; per-mob
  overrides deferred until a clear need emerges.
- **Gear breaking mechanic**: salvage is material recovery only.
  Mentions framed around "recover reagents from failed crafts" not
  "gear can break."
- **Other zones**: sanctum_basin pilot only. Template extracted after
  pilot ships; other zones iterate from there.
- **HarmMulti non-combatant filter**: not a known bug. Noted for
  followup if playtesting surfaces it.
- **`attack_reject_emote` per-mob YAML field**: covered by archetype +
  per-mob btree override (if customization needed later).
- **Engine work to make archetype + per-mob btrees composable**:
  acceptable duplication for now.

## Risk / Rollback

- **Dialogue YAML additions are additive** — new files don't affect
  existing NPCs; updated files add keywords without removing existing
  ones. Zero regression risk on existing player asks.
- **Archetype assignment in mob YAML** is a single-line addition.
  Rollback = delete the line; mob falls back to no-archetype behavior
  (current state).
- **`player_attack_rejected` event + dedupe field** is a new code path.
  Worst case: event fails to fire → no archetype emote, but player still
  sees the rejection message. Not a user-facing regression.
- **`non_combatant: true` added to meadow_lizard and chrysalis_echo**
  — if either was previously attackable in some playtest scenario, this
  change makes them un-attackable. Verify in smoke test that this is
  the intended behavior for these creatures.
