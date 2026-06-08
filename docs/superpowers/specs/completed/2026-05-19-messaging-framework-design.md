# Centralized Messaging Framework (Design)

**Status:** Draft 2026-05-19 — awaiting user review before writing-plans handoff
**Branch:** `feature/mob-aliveness-1.3-crimes` (continuation)
**Predecessor:** chunk 6 (Perception FSM shipped DORMANT, 2026-05-19)
**Successor:** mob aliveness substrate work (paused since chunk 0; resumes after this chunk)
**Visual brainstorm artifacts:** `.superpowers/brainstorm/79731-1779233531/content/*.html`

---

## 1. Problem Statement

Every player-facing line of text in DOGMud is composed and emitted in
isolation at the call site. Room broadcasts, combat narration, mob
idles, dialogue, system messages, status panels, prompts — each one
builds its own ANSI-tagged string, picks (or forgets) a color, decides
(or doesn't) whether to wrap, and ships through one of three
broadcast helpers (`room.SendText`, `room.SendTextVisual`,
`user.SendText`). The result is:

- **228 broadcast call sites** scattered across `combat/`, `hooks/`,
  `mobcommands/`, `usercommands/`, `rooms/`, `actions/`. Each one
  classifies its own content as visual vs audio (and often gets it
  wrong — the master spec's headline "companion-name leak" bug is
  one of those misclassifications).
- **`canSeeInRoom` and `sendRoomTextDarknessAware` duplicated** across
  `combat/combat.go` and `hooks/NewRound_DoCombat_helpers.go` and
  `actions/actor_mob.go`. Each duplicate computes the same predicate
  with subtle drift.
- **Color application is ad-hoc.** Combat narration is mostly plain
  white. Some spells use `pink`. Buffs use `cyan`. Surprise attacks
  prefix with `magenta-bold`. There is no enumerated taxonomy; each
  call site picks colors by feel. The chunk 4f review (memory
  `[[combat-text-color-coding]]`) explicitly flagged this.
- **Line wrapping is naive.** A single `wrapText` in
  `internal/usercommands/motd.go` counts byte length (ANSI escapes
  inflate the count, breaking the wrap). All other broadcast text
  ships unwrapped — players with wide terminals see one-line walls,
  narrow terminals get ugly soft wraps.
- **Perception FSM (chunk 6) ships dormant.** Sight/blindness state
  is observable but no consumer reads it; the framework that should
  consume it doesn't exist yet.
- **Style defects recur**: "negligible damage damage" (duplicate
  word), "a aggressive posture" (article disagreement), "in
  control, you press forward" (sentence-start lowercase), "remaining
  human scatter" (plural mismatch). Each site fixes these as
  one-offs; the underlying pattern keeps regenerating new bugs.

This chunk centralizes all player-facing text into a single pipeline:
compose → normalize → anonymize → color → wrap → deliver. Every
output path consults the same set of helpers. The Perception FSM gets
wired into the sight gate; infrared observers get an anonymized "red
shapes" render; combat narration gets categorized and colored;
broadcasts get per-recipient line-width-aware wrapping; recurring
style defects are caught at render time.

---

## 2. Design Goals

1. **One pipeline owns all player-facing text.** Every broadcast
   helper routes through the same compose → normalize → anonymize →
   color → wrap → deliver chain. No call site bypasses any stage.
2. **Channel separation is enforced.** Visual content (room
   descriptions, combat narration, names) goes through the
   sight-gated path. Audio / tactile / speech / smell goes through
   the unfiltered path. The 228-site audit assigns each call to the
   correct channel.
3. **Sight gate consumes the chunk-6 Perception FSM.** Combined
   with room lighting and observer vision modes (NightVision,
   InfraredVision), it produces three outcomes per recipient:
   CanSeeClearly (full text), CanSeeShapes (infrared in dark —
   anonymized "red shapes"), or no visual (audio still delivered).
4. **Categories are an enum, color mapping is data-driven.** A
   single `messaging.Category` enum names every recognized text
   category. Each maps to an existing `ansi-aliases.yaml` entry —
   no hard-coded color literals in code; the YAML is the tunable
   source of truth.
5. **Style normalization stages catch recurring defects.** ANSI
   tag canonicalization for names, sentence-start capitalization,
   article a/an agreement, duplicate-word collapse, sentence-end
   punctuation auto-append — all five run as part of the pipeline.
6. **Wrapping is ANSI-aware and per-recipient.** The wrapper counts
   display-width characters, not bytes; respects open ANSI tags
   across line breaks; reads a per-`UserRecord` `LineWidth`
   preference (default 80).
7. **The headline companion-name-leak bug is fixed.** The audit
   identifies the leaking call site; the centralized helper makes
   the right thing happen by default.

---

## 3. Package Layout

New package: `internal/messaging/`.

```
internal/messaging/
├── messaging.go          Category enum, Channel constants, public API
├── pipeline.go           The compose → normalize → anonymize → color → wrap → deliver pipeline
├── predicates.go         CanSeeClearly / CanSeeShapes (consume Perception + lighting + vision flags)
├── anonymize.go          Regex ANSI-name-tag stripping for infrared "red shapes" rendering
├── color.go              Category → ANSI alias resolver
├── normalize.go          Sentence-start cap, a/an, dup-word, sentence-end punct, ANSI canon
├── wrap.go               ANSI-aware line wrapping at configurable width
├── progression.go        Banner format helper (SKILL ADVANCEMENT / STATISTIC INCREASED)
├── messaging_test.go     Behavior Matrix unit tests
├── pipeline_test.go      End-to-end pipeline tests
└── context.md            Package documentation
```

---

## 4. The Pipeline

Every call to `room.SendText(cat, text, ...)`, `room.SendTextVisual(cat, text, ...)`,
`user.SendText(cat, text)`, `user.SendTextVisual(cat, text)` flows through:

```
                     ┌──────────────────────────────┐
                     │ 1. Compose                   │
                     │    caller produces text +   │
                     │    category tag              │
                     └─────────────┬────────────────┘
                                   │
                     ┌─────────────▼────────────────┐
                     │ 2. Style normalize            │
                     │    capitalization, a/an,     │
                     │    dup-word, end-punct,      │
                     │    ANSI tag canonicalization │
                     └─────────────┬────────────────┘
                                   │
            ┌──────────────────────┴──────────────────────┐
            │   For each recipient                         │
            │                                              │
            │   3. Sight gate (visual channel only)        │
            │      • CanSeeClearly?  → full text          │
            │      • CanSeeShapes?   → anonymize stage     │
            │      • Neither?        → skip (audio ok)    │
            │                                              │
            │   4. Anonymize (infrared-only path)          │
            │      strip <ansi fg="username">/</ansi> +    │
            │      <ansi fg="mobname">/</ansi> name tags   │
            │      replace with "a figure" / "something"   │
            │                                              │
            │   5. Apply category color tag                │
            │      <ansi fg="category-X">…</ansi>          │
            │                                              │
            │   6. Wrap at recipient's LineWidth (80 def)  │
            │      ANSI-aware: counts display width,       │
            │      carries open tags across line breaks    │
            │                                              │
            │   7. Deliver to recipient connection         │
            └──────────────────────────────────────────────┘
```

Audio channel (`SendText`) skips stages 3 + 4 + 5 (still gets color
if the Category has a non-default mapping). Tactile/speech/smell
content uses appropriate Category constants but doesn't get sight-
gated.

---

## 5. Helper API

### 5.1 Extended signatures

```go
// internal/rooms/rooms.go — existing helpers gain a Category leading parameter
func (r *Room) SendText(cat messaging.Category, txt string, excludeUserIds ...int)
func (r *Room) SendTextVisual(cat messaging.Category, txt string, excludeUserIds ...int)

// internal/users/userrecord.go — per-user variants
func (u *UserRecord) SendText(cat messaging.Category, txt string)
func (u *UserRecord) SendTextVisual(cat messaging.Category, txt string)
```

- **`SendText`** = unfiltered audio/tactile/speech/smell. All recipients
  in the room receive it (modulo `excludeUserIds`). Audio still
  reaches Blinded observers.
- **`SendTextVisual`** = sight-gated. Per-recipient pipeline (see §4).

`messaging.CategoryDefault` (= 0, zero value) keeps prose unchanged for
sites where coloring isn't appropriate (system messages, room
descriptions). The Category param is required at every call site after
the migration — no default-arg variant, so the audit forces every
caller to think about what category their text is.

### 5.2 Companion helpers

```go
// messaging.SendProgression — builds the SKILL ADVANCEMENT / STATISTIC INCREASED
// banner (see §11). Replaces the existing one-liner skill-up text.
func SendProgression(user *users.UserRecord, kind ProgressionKind, name string, tier *TierChange)

type ProgressionKind int
const (
    ProgSkill ProgressionKind = iota
    ProgStat
)

type TierChange struct {
    From string // e.g. "apprentice", "keen"
    To   string // e.g. "journeyman", "exceptional"
}
```

---

## 6. The Category Enum

```go
package messaging

type Category int

const (
    CategoryDefault Category = iota

    // === Combat ===
    CategoryHitMelee        // weapon swings, slashing/cleaving/stabbing band
    CategoryHitBlunt        // bludgeoning/slam/fist/gore
    CategoryHitNaturalSharp // claws/bite/sting
    CategoryHitRanged       // shooting/whipping
    CategoryHitCaster       // wand/sceptre/staff used as melee
    CategoryHitUnarmed      // explicit unarmed strikes

    CategoryDodge           // dodge defense
    CategoryParry           // parry defense
    CategoryBlock           // block defense

    CategoryGrappleFlow     // grapple round narration (warm taupe)
    CategoryGrappleHigh     // grapple position-change outcome (brighter taupe)

    CategorySubmission      // chunk 4d submission attempts (dusty rose)
    CategoryDeath           // death messages (faded grey)

    // === Special moves (per actual code colors) ===
    CategorySurpriseAttack  // *[SURPRISE ATTACK]* prefix, magenta-bold (existing)
    CategoryKick            // verb-only yellow-bold (existing)
    CategoryTrip            // amber (between kick + grapple)
    CategoryBash            // copper (between trip + grapple)
    CategoryRally           // cyan-bold whole line (existing)
    CategoryWarcry          // red-bold whole line (existing)
    CategoryTauntSuccess    // dusty rose-pink
    CategoryTauntResist     // green (existing)
    CategoryTauntFailure    // yellow (existing)

    // === Spell phases ===
    CategorySpellFold       // cast-begin + wait-tick (pale steel-cyan, all schools share)
    CategorySpellDisruption // concentration-shatters (warning amber)
    CategorySpellElemental  // resolve text — fire/ice/lightning (warm red-orange)
    CategorySpellEnhancement// resolve text — buffs/shields/enchantments (warm gold)
    CategorySpellMental     // resolve text — psionics/illusion (dusty plum)
    CategorySpellVital      // resolve text — heal/cure (sage mint)
    CategorySpellManifestation // resolve text — summon/raise (dusty rose-pink)

    // === Social ===
    CategorySpeech          // "<player> says, ..." — light-blue quoted text
    CategoryWhisper         // dusty lavender
    CategoryShout           // amber bold
    CategoryOOC             // desaturated blue-grey
    CategoryNPCDialogue     // NPC quest replies — parchment tone
    CategoryDialogueHint    // "[You could ask...]" — sage italic
    CategoryEmote           // /emote command output
    CategoryMobIdle         // idle mob ambient lines — warm grey
    CategoryMobEmote        // directed mob emotes — slightly warmer

    // === System / meta ===
    CategoryBroadcast       // server-wide announcements — sky blue
    CategoryTip             // [Tip] nudges — sage
    CategorySystem          // engine messages, "you sit down and meditate" — dim grey
    CategoryError           // "You can't do that here" — warm red
    CategoryWarning         // "You are about to be kicked" — amber
    CategorySkillProgress   // *** sharpening *** lines (used by SendProgression)
    CategoryLogin           // "X has logged in" — sage
    CategoryLogout          // "X has gone offline" — dim grey

    // === Environment ===
    CategoryRoomDescription // unchanged today; gets wrapping only
    CategoryRoomEntry       // "X arrives from the south"
    CategoryRoomExit        // "X leaves to the north"
    CategoryWeather         // sun rises/sets, rain banners
    CategoryTimeOfDay       // dawn/dusk callouts

    // === Other ===
    CategoryLoot            // "You pick up X" — system grey
    CategoryEquipment       // "You wear X" — system grey
    CategoryBuffApply       // existing cyan treatment for buff start
    CategoryBuffExpire      // existing cyan for buff end
    CategoryMutation        // mutation activation/deactivation
    CategoryToxin           // toxin/drunk/poisoned narration
)
```

Mapping table — see `_datafiles/world/dogmud/ansi-aliases.yaml`
additions in §8. The enum is extensible — adding a new Category is a
2-line change (enum constant + YAML alias) without touching the
pipeline.

---

## 7. Composite Predicates (§4 stage 3)

```go
// predicates.go

// CanSeeClearly returns true if the observer can read a normal-text
// visual broadcast in this room. Composes Perception state + room
// lighting + NightVision. Blinded observers (any source) and dark
// rooms with no NightVision both return false.
func CanSeeClearly(observer *characters.Character, room *rooms.Room) bool {
    if observer == nil || observer.Perception == nil {
        return true // defensive — pre-init characters default to seeing
    }
    if observer.Perception.State() == perception.Blinded {
        return false
    }
    if room != nil && room.GetVisibility() >= 1 {
        return true
    }
    return observer.HasFlagFromAnySource(buffs.NightVision)
}

// CanSeeShapes returns true if the observer can detect SOMETHING is
// happening — either full clarity (CanSeeClearly subsumes) OR infrared
// in the dark. Blindness gates this too — broken eyes don't see
// infrared.
func CanSeeShapes(observer *characters.Character, room *rooms.Room) bool {
    if CanSeeClearly(observer, room) {
        return true
    }
    if observer == nil || observer.Perception == nil {
        return true
    }
    if observer.Perception.State() == perception.Blinded {
        return false
    }
    return observer.HasFlagFromAnySource(buffs.InfraredVision)
}
```

New buff flag `buffs.InfraredVision = "infrared-vision"` added to
`internal/buffs/buffspec.go` — matches the YAML mutation tag that
`steppe_wolf` / `deep_gnawer` already carry but which currently has
no code-side handling.

---

## 8. Color Rendering Path

DOGMud renders colors via `<ansi fg="alias">text</ansi>` tags resolved
through `_datafiles/world/dogmud/ansi-aliases.yaml`. The framework
adds new aliases for each Category. Example additions:

```yaml
# Combat — actor names (retuned from current bright-yellow / bright-cyan)
username: 153       # cool light blue (was 11 / bright yellow)
mobname:  180       # warm tan (was 51 / bright cyan)
petname:  108       # teal-cyan (was 215 / orange — companion is now teal in the palette)

# Combat — damage bands (256-color approximations of the mock hexes)
combat-hit-melee:        173   # dusty rose
combat-hit-blunt:        137   # warm brown
combat-hit-natural-sharp: 95   # dark red-brown
combat-hit-ranged:       137   # warm coffee (close to blunt; tune in T1)
combat-hit-caster:       146   # lavender-tinged
combat-hit-unarmed:      173   # dusty orange — current value

# Combat — defense
combat-dodge:            108   # leaf green
combat-parry:            143   # lime
combat-block:            71    # forest

# Combat — special moves (preserve existing where possible)
combat-surprise:         9     # bright red (was magenta-bold; user shifted toward red)
combat-kick:             11    # yellow-bold (existing)
combat-trip:             214   # amber
combat-bash:             172   # copper
combat-rally:            14    # cyan-bold (existing)
combat-warcry:           9     # red-bold (existing)
combat-taunt-success:    169   # dusty rose-pink
combat-taunt-resist:     2     # green (existing)
combat-taunt-failure:    11    # yellow (existing)

combat-grapple-flow:     94    # warm taupe
combat-grapple-high:     130   # brighter taupe
combat-submission:       169   # dusty rose-pink
combat-death:            8     # faded grey (existing convention)

# Spells
spell-fold:              152   # pale steel-cyan
spell-disruption:        214   # warning amber
spell-elemental:         173   # warm red-orange
spell-enhancement:       179   # warm gold
spell-mental:            146   # dusty plum
spell-vital:             108   # sage mint
spell-manifestation:     169   # dusty rose-pink

# Social
speech-text:             111   # light-blue quoted text
whisper:                 139   # dusty lavender
shout:                   215   # amber bold
ooc:                     67    # desaturated blue-grey
npc-dialogue:            180   # parchment
dialogue-hint:           65    # sage italic
mob-idle:                144   # warm grey
mob-emote:               137   # slightly warmer grey

# System
broadcast:               75    # sky blue
tip:                     108   # sage
system:                  8     # dim grey
error:                   9     # warm red
warning:                 214   # amber
skill-progress:          179   # warm gold (matches enhancement)
login:                   108   # sage
logout:                  8     # dim grey
```

(Exact 256-color codes are first-pass approximations; final values
tuned during T1 implementation against real terminal renders.)

The framework calls `<ansi fg="combat-hit-melee">...</ansi>` at
the color stage. The existing template-resolution pipeline does the
rest unchanged.

---

## 9. Anonymizer

`anonymize.go` implements a regex-based pass that runs ONLY for
infrared observers (CanSeeShapes && !CanSeeClearly). It strips two
patterns and replaces them with anonymized stand-ins:

```go
// Input:  <ansi fg="mobname">Thornwall Thug</ansi> strikes
//         <ansi fg="username">Calabe</ansi> with a steel longsword!
//
// Output: <ansi fg="combat-anon">a figure</ansi> strikes
//         <ansi fg="combat-anon">a figure</ansi> with something!

var nameTagPattern = regexp.MustCompile(
    `<ansi fg="(username|mobname|petname)">[^<]+</ansi>`)

func Anonymize(text string) string {
    return nameTagPattern.ReplaceAllString(text,
        `<ansi fg="combat-anon">a figure</ansi>`)
}
```

`combat-anon` alias resolves to a red shade (`#e63946` ≈ code `196`).

Bare-name occurrences (names embedded in prose without ANSI tags) leak
through — that's the v1 quality bar. A future polish pass can refine
to a content-aware anonymizer or per-broadcast variants. The 228-site
audit ensures most names are properly tagged, which keeps the leak
rate low.

---

## 10. Wrapping

`wrap.go` provides ANSI-aware line wrapping at a configurable width.

```go
// WrapAnsi wraps text at maxWidth display columns. ANSI escape
// sequences (`<ansi ...>...</ansi>` tags) don't count toward width.
// Open tags carry across line breaks (each new line gets a fresh
// `<ansi fg="...">` re-opener if the previous line ended mid-tag).
func WrapAnsi(text string, maxWidth int) string
```

Per-recipient width comes from `UserRecord.LineWidth` (new config
option, default 80). Players can `set linewidth 100` to widen.

The existing `internal/usercommands/motd.go:wrapText` is sunset and
replaced. Sites that pre-wrapped manually drop their local wrap calls
during the audit.

---

## 11. Style Normalization

`normalize.go` runs FIRST in the pipeline (before sight gating, before
color). Five stages, each independent and configurable per-Category:

1. **Sentence-start capitalization.** Auto-capitalize the first
   non-ANSI-tag character of each line, and after `. `/`! `/`? `.
   Skips lines that begin with a known proper-noun ANSI tag
   (`<ansi fg="username">`, `<ansi fg="mobname">`, etc.) — those are
   already proper-cased.

2. **Article a/an agreement.** Pattern `\b a ([aeiouAEIOU])` →
   `\b an $1`. Skip the inverse case (don't change `an house` to
   `a house` — that's domain-dependent on h-silence). Bonus pass
   handles `\b A ([aeiouAEIOU])` → `\b An $1`.

3. **Duplicate-word collapse.** Pattern `\b(\w+) \1\b` → `$1`.
   Catches "damage damage", "the the", etc. ANSI-aware: doesn't
   collapse across tag boundaries.

4. **Sentence-end punctuation auto-append.** If a line doesn't end
   in `.`/`!`/`?`/`,`/`)`/quotes/whitespace-trimmed, append `.`.
   Skip lines that look like banners (start with `━` or `*`),
   exclamation-only fragments, or pure-ANSI-tag wrappers.

5. **ANSI tag canonicalization for names.** Ensure player names are
   wrapped in `<ansi fg="username">…</ansi>` and mob names in
   `<ansi fg="mobname">…</ansi>`. The framework scans for known
   player/mob names from the room context and wraps any bare
   occurrences. Best-effort; doesn't catch first-letter-capitalized
   common nouns ("The thug") but does catch proper-name occurrences
   ("Calabe", "Thornwall Thug").

Each stage can be disabled per-Category via a small lookup table for
content where the transform would be wrong (e.g., room descriptions
that already manage their own capitalization).

**Out of scope**: singular/plural verb agreement. Requires template
keys to express subject-verb relationships; would inflate this chunk
beyond reason. A future template-system chunk picks it up.

---

## 12. Progression Banner Format

`progression.go` implements the SKILL ADVANCEMENT / STATISTIC INCREASED
banner from the brainstorm mocks.

```go
// SendProgression emits a two-line (or three-line on tier crossing)
// banner via the user's connection. Uses CategorySkillProgress.
func SendProgression(user *users.UserRecord, kind ProgressionKind, name string, tier *TierChange)
```

Banner layout (rendered after pipeline):

```
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
                       SKILL ADVANCEMENT
                        Unarmed Combat
                  apprentice → journeyman             ← only on tier crossing
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
```

Tier names come from the existing helpers:
`skills.GetSkillRankDescription` (novice → grandmaster) and
`templates.statQuality` (feeble → godlike).

The legacy `*** You feel your X skills sharpening! ***` and
`<ansi fg="magenta">***</ansi> ... <ansi fg="yellow">%s</ansi> ...`
patterns in `internal/characters/progression.go` are sunset and
replaced by `SendProgression` calls.

---

## 13. Migration Strategy

Per-package audit, sequenced across the 228 broadcast call sites and
the non-broadcast renderers (status, inventory, help, MOTD).

**Phase 1 — Framework primitives (no behavior change):**

- T1: New `internal/messaging/` package. Category enum, helper stubs.
- T2: Pipeline implementation (compose → normalize → anonymize → color → wrap → deliver).
- T3: `CanSeeClearly` + `CanSeeShapes` predicates. `InfraredVision` buff flag.
- T4: `ansi-aliases.yaml` color additions (the full Category palette).
- T5: `WrapAnsi` ANSI-aware wrapping; `UserRecord.LineWidth` config option.
- T6: `Anonymize` regex helper.
- T7: `SendProgression` banner helper.
- T8: Style-normalization stages with per-Category skip table.
- T9: Helper API extensions — `SendText`/`SendTextVisual` gain Category leading param. Existing callers (228 of them) still compile via a brief shim: a `SendTextLegacy` for old signatures that maps to `CategoryDefault` and emits a deprecation log. Shim is removed at end of audit.

**Phase 2 — Per-package audit + cutover:**

Each task audits one package's broadcast call sites and migrates them to the new API with proper Category tags. The shim's deprecation logs are zeroed in each package as it lands.

- T10: `internal/combat/` (~30 sites)
- T11: `internal/hooks/` (~60 sites)
- T12: `internal/mobcommands/` (~25 sites)
- T13: `internal/usercommands/` (~70 sites)
- T14: `internal/rooms/`, `internal/actions/`, `internal/behaviortree/` (~30 sites)
- T15: Renderer migrations (status, inventory, online, who, help, MOTD) — these don't go through `SendText` but consume `UserRecord.LineWidth` for wrapping.

**Phase 3 — Cleanup + close-out:**

- T16: Sunset `canSeeInRoom` duplicates + `sendRoomTextDarknessAware` + `wrapText` in motd.go + legacy progression messaging.
- T17: Companion-name-leak smoke fix (the audit IDs the leaking call; T17 validates the fix lands cleanly).
- T18: Context.md sweep + roadmap update + patch notes + AI smoke pass.

Estimated 18 tasks. Comparable to chunk 5 in size; biggest sub-task is T13 (`usercommands/` cutover) at ~70 sites.

---

## 14. Sunset List

**Helpers deleted at end of chunk:**

- `internal/combat/combat.go:canSeeInRoom` (duplicate)
- `internal/hooks/NewRound_DoCombat_helpers.go:canSeeInRoom` (duplicate)
- `internal/actions/actor_mob.go:sendRoomTextDarknessAware`
- `internal/usercommands/motd.go:wrapText` (naive byte-count wrapper)
- Existing `*** You feel your X skills sharpening! ***` literal in `internal/characters/progression.go`
- The inline dark-room loop inside `internal/rooms/rooms.go:SendTextVisual` (replaced by pipeline)

**Color aliases retuned (not deleted):**

- `username` (11 / bright yellow → 153 / cool blue) — matches new palette
- `mobname` (51 / bright cyan → 180 / warm tan)
- `petname` (215 / orange → 108 / teal-cyan)

**Compat preserved:**

- All existing `<ansi fg="alias">…</ansi>` patterns continue to work — the framework adds new aliases, doesn't remove any. The user-facing color shift is the deliberate change.
- Existing `room.SendText` / `room.SendTextVisual` / `user.SendText` signatures get a leading `Category` param. The migration uses a temporary `SendTextLegacy` shim that maps to `CategoryDefault`; shim deleted at end of audit (T16).

---

## 15. Risks / Open Questions

- **Color-rendering performance.** Pipeline adds 5-7 stages per recipient per broadcast. For high-fan-out events (room with 30+ recipients during combat), this is ~30 × 5 stages per round. Each stage is pure-string regex/transform — should be ~microseconds per call. Profile during T1; if a hot spot emerges, cache by category+text pair.
- **Wrapping under load.** ANSI-aware wrap must handle malformed input (orphan tags, nested ansi). Strict parsing risks panics in production. T5 uses a defensive parser that falls back to byte-count + visible-character estimation on parse errors.
- **256-color codes are first-pass approximations.** The exact codes in §8 are educated guesses; final values get tuned in T4 against real terminal renders. Several may need adjustment after seeing them in the dark MUD terminal context.
- **Tabular displays (status / inventory / online) staying self-contained.** They don't route through `SendText`, but they DO consume `UserRecord.LineWidth`. Need to verify each tabular renderer's existing wrap behavior gets updated to read this config. Risk: missing one leaves it stuck at the old hard-coded width.
- **Audit drift between Phase 1 and Phase 2.** Phase 2 spans many tasks; Phase 1's API could shift mid-cutover. Lock the API signatures at T9 and treat any change as a re-spec event.
- **Style normalization over-correction.** Auto-capitalization could miscapitalize a deliberately-lowercase poetic line in a room description. Per-Category skip table mitigates; specific cases get added to the skip list as smoke surfaces them.
- **Companion teal vs dodge green proximity.** From the mock follow-up: companion `#7fc4b5` and dodge `#8fbf94` are close enough that color-blind players might conflate. Worth a follow-up after smoke.
- **Per-user `LineWidth` config**. Default 80; needs a `set linewidth N` command for players to override. Add in T5 alongside the config option.

---

## 16. Success Criteria

1. `internal/messaging/` package exists with the full Category enum + 5-stage pipeline + helpers.
2. `SendTextVisual` is sight-gated using the chunk-6 Perception FSM + lighting + NightVision/InfraredVision.
3. Infrared observers in dark rooms receive anonymized "red shapes" text via the regex stripper.
4. All 228 broadcast call sites migrated to the new API with appropriate Category tags.
5. Tabular renderers (status / inventory / online / who) consume `UserRecord.LineWidth` for their internal wrapping.
6. `canSeeInRoom` duplicates removed; only one implementation remains (in `internal/messaging/predicates.go`).
7. `sendRoomTextDarknessAware` removed; callers route through `room.SendTextVisual` instead.
8. Naive `wrapText` in `motd.go` removed; centralized `messaging.WrapAnsi` used everywhere.
9. Progression banners render in the new SKILL ADVANCEMENT / STATISTIC INCREASED format, including tier-crossing third line.
10. Combat narration colored by category — defense (green family), damage (weapon-band hues), special moves (existing kick/rally/warcry/taunt colors), grapple (warm taupe), spell (5-school palette + fold/disruption split), surprise (`*[SURPRISE ATTACK]*` magenta-bold).
11. AI feature-tester smoke: no panics, no missing-color debug strings, no double-anonymization bugs, the companion-name leak no longer reproduces in a dark room.
12. `go build ./...` clean, `go test ./...` green, server boots cleanly past data-file loading.

---

## 17. Implementation Order (Preview for writing-plans)

1. New `internal/messaging/` package skeleton + Category enum + Behavior Matrix unit tests (RED → GREEN).
2. Pipeline core (compose → normalize → anonymize → color → wrap).
3. CanSeeClearly + CanSeeShapes predicates; InfraredVision buff flag.
4. `ansi-aliases.yaml` color additions.
5. WrapAnsi + UserRecord.LineWidth config; `set linewidth` command.
6. Anonymize regex helper.
7. SendProgression banner helper.
8. Style normalization stages with per-Category skip table.
9. SendText / SendTextVisual API extensions + temporary legacy shim.
10. Audit + migrate `internal/combat/`.
11. Audit + migrate `internal/hooks/`.
12. Audit + migrate `internal/mobcommands/`.
13. Audit + migrate `internal/usercommands/`.
14. Audit + migrate `internal/rooms/` + `internal/actions/` + `internal/behaviortree/`.
15. Tabular-renderer migration (status / inventory / online / who / help / MOTD wrapping integration).
16. Sunset list — remove duplicates + shim + naive wrapText + legacy progression.
17. Companion-name-leak smoke fix validation.
18. Context.md sweep + roadmap update + patch notes + AI feature-tester smoke pass.

Estimated 18 tasks. Largest non-framework task: T13 (`usercommands/` audit, ~70 sites). No smoke until T18 — the framework is dormant in Phase 1, exercised by per-package audits in Phase 2, and validated end-to-end at the close.

---

## 18. Visual Brainstorm Artifacts

Mockups from the design conversation live at:
`.superpowers/brainstorm/79731-1779233531/content/`

Key files:
- `category-landscape.html` — initial scope inventory of all text categories
- `combat-colors.html` / `combat-colors-v2.html` — palette A (saturated) vs B (muted, chosen)
- `weapon-hues.html` — damage subtype band palette
- `specials-and-surprise.html` — surprise / kick / rally / warcry / taunt
- `trip-bash-others.html` — knockdown gradient + speech + dialogue + mob idles + system
- `progression-v3.html` — final banner format (skill + stat, with tier crossing)
- `spell-schools.html` — 5-school spell palette
- `spell-folds.html` — fold (steel-cyan) + resolve (school) split

These are reference for the visual treatment; the spec is authoritative for behavior.
