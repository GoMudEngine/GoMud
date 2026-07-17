# Newcomer Tutorial Redesign — Guide-Led Handholding (Route 1)

**Date:** 2026-07-17
**Status:** Design approved; ready for implementation plan
**Scope:** Rebuild the route-1 "complete noob" onboarding (the instanced Newcomer
Antechamber) as a strict, guide-led, thoroughly-explained tutorial. Routes 2
("New to DOGMud") and 3 ("Veteran") are **out of scope** and untouched.

---

## 1. Problem

The current route-1 antechamber (rooms 6258–6262) is a downgrade from even the
base GoMud onboarding:

- Lessons are narrated by a **disembodied "calm voice" baked into room
  `description:` prose**, not spoken by a character.
- Commands are set off with **plain double-spaces** (`type  look `) instead of
  the `<ansi fg="command">` cyan highlighting used everywhere else in the game
  (Hadwen, quest hints, help templates). This is a jarring inconsistency and
  makes commands hard to spot.
- The Guide NPC (mob 9491) only appears in the **final** room, as a doorman — it
  never teaches.
- Several true fundamentals are never taught as their own lessons: **movement**,
  **combat**, the **special-move cooldown**, **fleeing**, and the game's
  defining **use-based / no-levels / no-class progression** model.

For a player who selected "New to text MUDs," this is exactly the audience that
needs the *most* structured handholding and gets the least.

## 2. Goals & Non-Goals

**Goals**
- A single, warm, named mentor NPC (**Dewey**) present throughout, speaking every
  lesson, with consistent command highlighting.
- **Strict gating:** each sub-lesson must be *performed* before the tutorial
  advances. A complete novice cannot get lost or skip ahead.
- **Thorough and deliberately slow.** Start dead-simple, then open into prompts
  that reward reading, attention, and curiosity. Re-teaching concepts that
  Pothole Coulee also covers is **intended reinforcement, not redundancy**.
- Teach the DOGMud-specific systems a newcomer will otherwise never discover in
  guided form: the special-move cooldown and use-based progression.
- Fix the command-highlighting inconsistency.

**Non-Goals**
- No changes to routes 2 or 3, `start.go` routing, quests 21/30/31, Cleric
  Hadwen, or Pothole Coulee content.
- The mentor does **not** reappear later in the game (explicitly cut; user is
  indifferent, so we avoid half-building it — YAGNI).
- Not a full mechanics course. The tutorial *mentions and demonstrates*; the
  `help` system and Pothole Coulee own depth.

## 3. Form (approved)

- **Interaction model:** Strict gated tutor (option A). The guide will not open a
  room's exit until the current sub-lesson is done; premature-exit attempts are
  intercepted with a nudge.
- **Spatial model:** Themed multi-step rooms (option B). ~6 themed rooms, each
  running several gated sub-steps in place, with the guide traveling alongside.
  Movement between rooms is itself the movement lesson.
- **Guide identity:** A named, warm **fellow-Opened mentor** (option B) who
  remembers being new in this same grey place. Plainspoken, patient, a little
  wry — *not* mystical or cold. First contact should read as a real person
  choosing to help.

## 4. The Guide — Dewey

**Persona.** Dewey is a prior Opened who lingers in the between-place to steady
newcomers. Copy is written **pronoun-light** so gender can be set later (or left
they/them). Tone: encouraging, concrete, occasionally dry humor. Never
condescending.

**Presence mechanic (no follow/companion plumbing).** One guide mob template,
spawned into **all six** themed rooms via each room's spawn list. Same name and
appearance = the player perceives one continuous companion. At each exit Dewey
says an "I'll be right through there with you" line and is simply already present
in the next room. This reuses the existing instanced-room + behavior-tree pattern
— **no new follow code**.

**Two layers on the guide:**
- **Room behavior trees (per room)** drive each *gated lesson*: detect the taught
  command, praise, advance per-instance sub-step state, and intercept premature
  exits. Every teaching line is **attributed to Dewey by name** (`Dewey says,
  "..."`) with `<ansi fg="command">` highlighting — replacing the disembodied
  room-prose voice.
- **Dialogue tree (on the mob)** handles optional `ask dewey about <topic>` for
  curious players — the world/Gaius, the pools, the Opened, "no levels,"
  mutations. This is the "open up and let them explore" valve. The strict path
  never *requires* it; curiosity is rewarded, not gated.

**Safety flags:** non-combatant, charm-immune, non-fleeing (same posture as the
current guide mob 9491). Dewey re-themes the existing 9491 mob (renamed file per
`ConvertForFilename`).

## 5. Room-by-Room Curriculum

Six themed rooms, one-way corridor (each room's only open exit is forward, which
also guarantees `flee` in room 5 can only carry the player onward). Rooms map to
IDs in forward order: **room 1 = 6258, room 2 = 6259, … room 5 = 6262, room 6 =
6263** (the new room). Every command word below is rendered with
`<ansi fg="command">` cyan highlighting in-game.

| # | Room (working title) | Gated sub-lessons, in order |
|---|----------------------|------------------------------|
| 1 | **The Threshold** | `look` → `look <thing>` / examine a room noun → walk `north` (movement + "the world is a grid of rooms") |
| 2 | **Knowing Yourself** | `status` (the six stats + three pools: health / stamina / conviction) → progression **teaser** ("these grow — but not the way you'd expect; I'll show you soon") |
| 3 | **What You Carry** | `get <item>` (the grey token, existing `itemid: 2`) → `inventory` / `inv` → `help` (the "you'll forget, the game remembers" meta-lesson) |
| 4 | **The World Speaks** | `say <text>` → `ask dewey <topic>` (NPCs answer; doubles as the curiosity/exploration hook) |
| 5 | **The Proving** | `attack effigy` → `cast spike` → *feel the cooldown* (retry → recover message) → `cooldowns` → `trip` (body) → `warcry` (voice) [the "all playstyles valid" sampler; shared cooldown spaces them] → **forced real progression tick (banner shown once)** → **progression primer** → `flee` (carries player to room 6) |
| 6 | **The Landing** | Dewey's handoff → vortex to the Awakening Pool (room 5200); quest 30 (the Awakening) begins with Cleric Hadwen |

**Ordering rationale.** Progression is taught in room 5 — *after* the player has
done something and seen a real change — not lectured in the abstract in room 2
(room 2 only plants the teaser). Flee is the final combat beat and doubles as the
transition into the handoff room.

### 5.1 Room Atmosphere & Nouns

The between-place is *not* a featureless grey void — each themed room has a
distinct visual identity within the pre-Awakening dream logic, and each carries
**2–4 examinable `nouns:`** that reward the examine lesson and seed worldbuilding
(Gaius, the Opened, the pools, the Ring of Change). Room 1 especially must have
rich nouns, because it *teaches* `look <thing>` — the first thing a player
examines has to pay off, or the lesson lands flat. Nouns are authored
un-hyphenated with spaces per the parser convention; noun prose obeys the 80-col
wrap and the no-hard-numbers rule.

Design direction per room (build authors the final prose):

| # | Atmosphere | Example nouns (each seeds a little lore) |
|---|-----------|------------------------------------------|
| 1 **The Threshold** | Formless grey light slowly resolving into the first solid things; the player is becoming real. | **your hands** (half-there, firming as you wake); **the grey** (the between-place — "not death, not yet life"); **the threshold stone** underfoot (worn by every soul that ever woke here) |
| 2 **Knowing Yourself** | A still, dark, mirror-smooth surface that shows the player their own shape — a foreshadow of the Awakening Pool. | **the still water** (shows *you*, and something waiting beneath); **the sixfold pattern** faint in the depths (the six stats motif); **three embers** — pale blue-green, steady (the health/stamina/conviction pools) |
| 3 **What You Carry** | A quiet alcove of small kept things; the grey token rests here. | **the grey token** (the pickup, `itemid: 2`); **the alcove/shelf** (things the Opened leave behind for the next newcomer); **drifting motes** of the same pool-light |
| 4 **The World Speaks** | Sound seeps into the silence — distant voices, wind, the first hint of the world beyond. | **the far voices** (echoes of Gaius, of the living world you're bound for); **a doorway of light** showing glimpsed rooftops/coulee walls; **the wind** (carries the smell of real air) |
| 5 **The Proving** | A worn practice ring scuffed into the grey; the straw effigy stands ready. | **the straw effigy** (the target); **the scuffed ring** (countless newcomers have squared off here); **scattered straw** |
| 6 **The Landing** | The edge of the between-place; ahead, the pool-light of the true Awakening Pool and the way through. | **the pool-light** ahead (the real world, the Awakening waiting); **the archway/vortex** (the way through); **Dewey** (a warm last look before you go) |

## 6. Room 5 "The Proving" — detailed script & copy

The pedagogically load-bearing room. Gated sub-step chain (per-instance state):
`attacked` → `cast_spike` → `saw_cooldown` → `checked_cooldowns` → `tripped` →
`warcried` → `progression_shown` → `primer_heard` → (exit via `flee`).

**Why a three-move sampler (spell + strike + shout), not `bash`.** `bash` requires
a shield a fresh character (carrying only the grey token) lacks — excluded. But a
single demo move over-indexes on one playstyle. To drive home DOGMud's core
promise — **every playstyle is valid; no class locks you in** — room 5 samples one
gear-free move from each major family, all sharing the *one* cooldown:

- **`cast spike`** — *belief* (magic / Willpower). Every new character starts with
  `conviction-spike` (alias `spike`, a `harmsingle` damage spell), `chrysalis-glow`,
  and `identify` (`internal/characters/spells.go` `StarterSpells`). Surfaces the
  magic system the tutorial otherwise never touches.
- **`trip`** — *body* (martial / Dexterity). Bare-handed knockdown; vivid feedback.
- **`warcry`** — *voice* (rhetoric / Charisma). Gear-free shout; visibly affects the
  effigy. (`rally` — the self-steady panic-button — is a one-word swap if preferred.)

This is also a *stronger* cooldown lesson: the player feels the same recovery gate
whether they cast, strike, or shout, so "one well of focus for all your big moves"
lands concretely. `attack` still comes first for the "normal swings never tire"
contrast. The three families map cleanly onto the progression primer that follows
(Willpower / Dexterity / Charisma). Deeper martial and rhetoric drill still happen
with Drillmaster Vorn in Pothole Coulee. **Build-time checks:** (1) confirm `trip`
and `warcry` have no weapon/skill prerequisite for a fresh character; (2) confirm
the pool affords two casts of `spike` (`cost: 50`) plus the shout — top the pool
in-room if tight. **Pacing note:** the shared cooldown (4 rounds) spaces the three
moves; the effigy is harmless, so Dewey fills each wait ("keep swinging — you'll
feel the readiness return") and the tree gates the next prompt on cooldown-ready.

**The practice effigy (new mob).** The existing dummies
(`pothole_coulee/9109-training_dummy`, `9163-practice_butt`, `9146-practice_mote`)
are built for open-ended practice and are too tanky / wrong-purpose for a tight
scripted micro-lesson. **Create a new, purpose-built "straw effigy"** for the
antechamber:
- Attackable and a valid target for spells, martial specials, and shouts (so
  `attack`, `cast spike`, `trip`, and `warcry` all work and combat entry triggers).
- Deals **zero** damage; never retaliates.
- **Never flees** (no flee/courage behavior) — avoids the known Vorn-dummy-flee
  bug.
- Enough HP that it **cannot die** during the scripted lesson (or death is
  suppressed / it is flavor-"righted" by Dewey). The player must never have to
  grind it down or accidentally kill it mid-lesson.
- Stationary; spawns only in room 5; new mob ID allocated at build time.

**Sample copy** (backticked words = `<ansi fg="command">` cyan highlight):

> **Dewey:** "This straw fellow won't mind. Squaring off is simple — type
> `attack effigy` and your body takes over, swinging on its own each round."
>
> *(player attacks; auto-swings begin)*
>
> **Dewey:** "Good. Those steady swings are your *normal* attacks — they never
> tire. But you woke with more than fists, and here's what's worth knowing early:
> *no one way is the right way.* Let me show you three. First — belief. You already
> carry a spell or two. Type `cast spike` to drive a spike of raw conviction into
> it."
>
> *(player casts → the effigy takes the hit)*
>
> **Dewey:** "You felt that leave you. No robes, no order, no permission needed —
> in Gaius, belief *is* a weapon. Now: try to `cast spike` again, right away."
>
> *(player retries → real engine recovery message. Cast has two possible lines
> here — the shared-cooldown "You need a moment before you can do that." or the
> cast-init "Your mind is still recovering from the effort." Either demonstrates
> the wait; build confirms which fires for a back-to-back recast.)*
>
> **Dewey:** "There it is — a moment's recovery. Your big moves all draw from one
> well of focus; spend it and you wait a few rounds. Keep swinging normally
> meanwhile — those never wait. Type `cooldowns` to watch it tick down."
>
> *(player checks cooldowns)*
>
> **Dewey:** "Now watch — that same wait covers *everything*, not just spells.
> When it clears, try the *body's* way: get in close and take its legs. Type
> `trip`."
>
> *(cooldown clears; player trips → the effigy topples)*
>
> **Dewey:** "Ha — down it goes. Same well, different hand. One more kind, and it's
> the one people forget: your *voice*. When the well fills again, loose a `warcry`
> and put fear into it."
>
> *(cooldown clears; player warcries → the effigy flinches)*
>
> **Dewey:** "Spell, strike, and shout — belief, body, and voice. All three drew
> from the same well, and all three are *yours*. No class picks one and locks the
> rest away. You'll lean where you like — and grow toward it. Which brings me to
> the last thing worth understanding..."

Then the **forced progression tick** fires — the player sees the *genuine* banner
(e.g. `*** A moment of brilliance! Your spellcasting technique improves! ***`,
naming one of the skills they just exercised), and Dewey anchors the primer to it:

> **Dewey:** "See that? You just got *better* — not from any tally of kills or a
> level you climbed. There are no levels here, and no class boxing you in. In
> Gaius you grow by *doing*. Channel belief into a spell and your `Willpower`
> deepens; close in with fist or blade and your `Dexterity` sharpens; push
> yourself to exhaustion and your `Strength` answers; bend others with your voice
> and `Charisma` rises; loose arrows with a keen eye and `Perception` wakes. And
> over a longer road, the way you fight quietly pulls change through you —
> mutations, drawn from the Ring toward whatever you keep becoming. You never pick
> it. You *earn* it. When you're curious, `help skills` and `help mutations` hold
> the long version."

Then the flee beat, which transitions to room 6:

> **Dewey:** "Last thing — and maybe the most important. Not every fight is worth
> finishing. When one turns against you, don't die proving a point — get out.
> Type `flee` to break away and run."
>
> *(player flees → combat ends → carried through to room 6, The Landing)*
>
> **Dewey:** *(already there)* "See? Still breathing. Running isn't losing — it's
> how you live long enough to win the next one."

### 6.1 Accuracy constraints (verified against code)

The progression copy must stay truthful to the engine:

- **Melee trains Dexterity, not Strength.** `weapon-combat` and `unarmed-combat`
  govern **Dexterity** (`internal/skills/skills.go` skill→stat map;
  auto-stat-on-skill-use in `progression.go`). Strength grows from **exertion** —
  fighting at low stamina and regenerating (`OnRegenTick`, `progression.go`). The
  CLAUDE.md shorthand "swing a sword → Strength" is imprecise; the tutorial says
  melee sharpens **Dexterity** and Strength comes from pushing yourself.
- **Mutations are drift-you-earn, never chosen.** Matches `help mutations` ("the
  skills you use pull you toward the clusters they suit"). Progress accrues in
  combat; ~70% deepen an existing mutation vs 30% acquire new.
- **The special-move cooldown is a shared 4-round timer** (`SpecialMoveCooldown:
  4`, `config.yaml`; tag `"special-move"` in `internal/characters/cooldowns.go`).
  It gates bash, trip, kick/stomp/knee, grapple, taunt, rally, warcry, cast,
  throw, reload, and mutation active moves. **Normal auto-attacks and `shoot`/
  `fire` are NOT on it** — the copy's "normal swings never wait" is accurate. The
  real blocked message is *"You need a moment to recover before attempting
  another special move."* and the live status command is `cooldowns`.

### 6.2 Forced progression tick — implementation note

A short tutorial cannot *rely* on a natural progression roll (~12% per stat check;
~1 skill rank per 25 uses). To guarantee the player sees the real banner exactly
once, the room 5 behavior tree must **force one genuine progression event** (a
real skill/stat increment that emits the standard `banner.Format(banner.Skill,
...)` / `*** ... technique improves! ***` output). If an existing behavior-tree
action can grant a progression tick, reuse it; otherwise add a small new action
(e.g. `grant_progression <skill>`). This is honest — the player *did* use the
skill — and it teaches recognition of the message for when it fires naturally
later.

## 7. Command Highlighting

Every taught command across all six rooms uses `<ansi fg="command">...</ansi>`
(color alias `command: 6`, cyan; defined in `ansi-aliases.yaml`). This replaces
the antechamber's current plain-double-space convention and brings route 1 in
line with the rest of the game. Applies to Dewey's behavior-tree lines, the
dialogue tree, and any residual room prose.

## 8. File & ID Inventory

IDs allocated via `python tools/id_inventory.py` at build time. The antechamber
zone currently owns rooms 6258–6262 and guide mob 9491.

**Rewrite (existing files — gut the calm-voice prose, drive from Dewey, add
highlighting):**
- `_datafiles/world/dogmud/rooms/newcomer_antechamber/6258–6262.yaml` (5 rooms)
- `_datafiles/world/dogmud/behaviors/rooms/newcomer_antechamber/6258–6262.yaml`
  (5 gating trees)
- `_datafiles/world/dogmud/mobs/newcomer_antechamber/9491-*.yaml` → re-theme to
  **Dewey** (new filename to match the `name:` field per `ConvertForFilename`)
- `_datafiles/world/dogmud/dialogue/newcomer_antechamber/9491.yaml` (dialogue
  file is keyed by mob ID → stays `9491.yaml`)
- `_datafiles/world/dogmud/goals/9491-*.yaml`

**Create (new IDs):**
- **Room 6263** `rooms/newcomer_antechamber/6263.yaml` (the 6th themed room) +
  its behavior tree `behaviors/rooms/newcomer_antechamber/6263.yaml`
- **Straw effigy** practice-dummy mob (new mob ID) + minimal goals/behavior
  (stationary, zero-damage, non-fleeing, non-lethal), spawned in room 5

**Reuse (no change):**
- Grey token item `itemid: 2` (the existing antechamber token pickup; relocated
  to the room-3 "What You Carry" room, ID 6260)

**Edit:**
- `_datafiles/config.yaml` — add `6263` to `TutorialRooms`; fix the stale
  "Sanctum Basin replaces the old tutorial zone; TutorialRooms is intentionally
  empty" comment (it is contradicted by the populated array and the live code).

**Code (two small tasks, flagged for the plan):**
1. **Force-progression behavior action** (§6.2) — reuse or add a behavior-tree
   action that grants one real progression tick with the standard banner.
2. **Guaranteed tutorial flee** — room 5's `flee` must always succeed and can
   only exit forward (single forward exit in room data; effigy does not contest
   the flee). No failed-flee dead-end.

**Untouched:** `start.go` routing, quests 21/30/31, Cleric Hadwen, Pothole
Coulee, routes 2 & 3.

## 9. Testing / Verification

- **Boot test (pre-push SOP):** wipe instance saves
  (`mobs.instances/*`, `rooms.instances/*`), restart, confirm the antechamber
  zone + new mob load without panic (name/filename match, valid trigger events,
  no ID collision). Run `python tools/id_inventory.py` after allocating new IDs.
- **Full walk-through:** create a fresh character, pick route 1, and complete all
  six rooms end-to-end. Verify each gate blocks premature exit, every command is
  highlighted, the effigy cannot be killed and never flees, the cooldown message
  and `cooldowns` output appear, the forced progression banner shows exactly
  once, `flee` succeeds and lands in room 6, and the handoff drops the player at
  room 5200 with quest 30 active. Also confirm the full **playstyle sampler**
  works on the effigy — `cast spike`, `trip`, and `warcry` each land and each is
  gated by the shared cooldown — that a fresh character's pool affords the casts,
  and that **every authored room noun is examinable** (`look <noun>` returns its
  lore prose) — the examine lesson in room 1 depends on it.
- **Playtest:** a naive-newbie `/playtest local feel-tester` pass focused on the
  antechamber, watching for confusion, dead-ends, or unhighlighted commands.

## 10. Open Questions / Risks

- **Effigy non-lethality mechanism:** decide between very-high-HP-that-can't-be-
  depleted-in-the-lesson vs. explicit death-suppression. Prefer whichever is
  simplest with existing mob flags; settle during planning.
- **Force-progression action:** whether an existing behavior action suffices or a
  new one is needed — resolve early in the plan (small, but it's real code).
- **Room 5 flee edge cases:** confirm flee-while-only-one-exit behavior and that
  a zero-damage effigy still permits a normal `flee` resolution.
