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
| 5 | **The Proving** | `attack effigy` → `bash` → *feel the cooldown* (retry `bash` → real recover message) → `cooldowns` → **forced real progression tick (banner shown once)** → **progression primer** → `flee` (carries player to room 6) |
| 6 | **The Landing** | Dewey's handoff → vortex to the Awakening Pool (room 5200); quest 30 (the Awakening) begins with Cleric Hadwen |

**Ordering rationale.** Progression is taught in room 5 — *after* the player has
done something and seen a real change — not lectured in the abstract in room 2
(room 2 only plants the teaser). Flee is the final combat beat and doubles as the
transition into the handoff room.

## 6. Room 5 "The Proving" — detailed script & copy

The pedagogically load-bearing room. Gated sub-step chain (per-instance state):
`attacked` → `bashed` → `saw_cooldown` → `checked_cooldowns` → `progression_shown`
→ `primer_heard` → (exit via `flee`).

**The practice effigy (new mob).** The existing dummies
(`pothole_coulee/9109-training_dummy`, `9163-practice_butt`, `9146-practice_mote`)
are built for open-ended practice and are too tanky / wrong-purpose for a tight
scripted micro-lesson. **Create a new, purpose-built "straw effigy"** for the
antechamber:
- Attackable (so `attack`/`bash` and combat entry work).
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
> tire. But you've got harder blows in you. Try one: type `bash` to throw your
> shoulder into it."
>
> *(player bashes)*
>
> **Dewey:** "Ha! Felt that. Now — try to `bash` again, right away."
>
> *(player retries → real engine message: "You need a moment to recover before
> attempting another special move.")*
>
> **Dewey:** "There it is. Your strongest moves — special strikes like that,
> spells, battle-shouts, the powers a mutation will one day grant you — all draw
> from the same well of focus. Spend it and you need a few rounds before the next
> one. Your normal swings never wait, though, so you're never helpless. Want to
> see the timer? Type `cooldowns`."
>
> *(player checks cooldowns)*

Then the **forced progression tick** fires — the player sees the *genuine* banner
(e.g. `*** A moment of brilliance! Your unarmed-combat technique improves! ***`),
and Dewey anchors the primer to it:

> **Dewey:** "See that? You just got *better* — not from any tally of kills or a
> level you climbed. There are no levels here, and no class boxing you in. In
> Gaius you grow by *doing*. Swing blades and your `Dexterity` sharpens; push
> yourself to exhaustion and your `Strength` answers; bend others with words and
> `Charisma` rises; loose arrows with a keen eye and `Perception` wakes. And over
> a longer road, the way you fight quietly pulls change through you — mutations,
> drawn from the Ring toward whatever you keep becoming. You never pick it. You
> *earn* it. When you're curious, `help skills` and `help mutations` hold the
> long version."

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
  room 5200 with quest 30 active.
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
