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
  guided form: the special-move cooldown, **conditions** (buffs/debuffs), and
  use-based progression.
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
IDs in forward order: **room 1 = 6258, room 2 = 6259, room 3 = 6260, room 4 = 6261,
room 5 = 6262, room 6 = 6467** (the new room — 6263 is already taken by
`east_road_to_greenford`; 6467 is the next globally-free room ID per
`id_inventory.py`). Every command word below is rendered with `<ansi fg="command">`
cyan highlighting in-game.

| # | Room (working title) | Gated sub-lessons, in order |
|---|----------------------|------------------------------|
| 1 | **The Threshold** | `look` → `look <thing>` / examine a room noun → walk `north` (movement + "the world is a grid of rooms") |
| 2 | **Knowing Yourself** | `status` (the six stats + three pools: health / stamina / conviction) → progression **teaser** ("these grow — but not the way you'd expect; I'll show you soon") |
| 3 | **What You Carry** | `get <item>` (the grey token, existing `itemid: 2`) → `inventory` / `inv` → `help` (the "you'll forget, the game remembers" meta-lesson) |
| 4 | **The World Speaks** | `say <text>` → `ask dewey <topic>` (NPCs answer; doubles as the curiosity/exploration hook) |
| 5 | **The Proving** | *Pre-combat:* `warcry` (steel yourself — *voice*, a self-buff) → `conditions` (see the buff → learn conditions) → `cooldowns` (the shout spent your focus → learn the shared well). *In combat:* `attack effigy` (normal swings) → `cast spike` (*belief*) → recast `cast spike` (blocked — the shared well is spent; real recover message) → `trip` after the well refills (lands, knocks the effigy prone = an enemy condition) → **forced progression tick (banner once)** → **progression primer** → `flee` (carries player to room 6) |
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
`warcried` → `saw_conditions` → `checked_cooldowns` → `attacked` → `cast_spike` →
`recast_blocked` → `tripped` → `progression_shown` → `primer_heard` →
(exit via `flee`).

> **Authoring constraint (verified):** room behavior trees have **no condition to
> read a player's cooldown or buffs** (only quest/item/gold/flag/spell/misc-data
> player conditions exist). So the "blocked move" beat cannot branch on cooldown
> state. We make it deterministic instead: the player **recasts `cast spike`
> immediately**, which is reliably blocked (the first cast just spent the shared
> well), then `trip` is prompted only after the well refills so it lands cleanly.
> The tree drives purely off command detection + per-instance `set_state` flags.

**Why this shape (conditions & cooldown taught pre-combat, then a belief+body
sampler).** `bash` is excluded — it needs a shield a fresh character (carrying only
the grey token) lacks. The room makes two points: (1) DOGMud's core promise that
**every playstyle is valid; no class locks you in**, and (2) the existence of
**conditions** (temporary effects) and the **shared cooldown**. It does this in two
phases:

- **Pre-combat.** The player opens with **`warcry`** — a gear-free rhetoric shout
  (*voice*) that is a **self-buff**: it lands a real *condition* on the player (buff
  79 "Warcry", a damage bonus, ~25 rounds) and, because it's a special move, starts
  the shared cooldown. One shout teaches two systems: **`conditions`** shows the buff
  it applied; **`cooldowns`** shows the focus it spent. (`rally` — a defensive
  self-buff — is a one-word swap if you prefer the panic-button framing.)
- **In-combat.** With blood up, the player fights: **`attack`** (normal swings,
  never tire), then **`cast spike`** — *belief*, the starter `conviction-spike`
  `harmsingle` spell (`internal/characters/spells.go` `StarterSpells`), surfacing the
  magic system the tutorial otherwise never touches. Spike draws the well down, so an
  immediate **`trip`** (*body*) is genuinely **blocked** (real "You need a moment to
  recover…" message) before it lands — teaching the shared cooldown across
  *different* families. Trip then knocks the effigy **prone** — a *condition on the
  enemy* — so conditions land from both the buff and the debuff side.

Across the room the player uses voice, belief, and body — all from the one well —
mapping onto the progression primer that follows (Charisma / Willpower / Dexterity).
Deeper martial and rhetoric drill still happen with Drillmaster Vorn in Pothole
Coulee. **Build-time checks:** (1) `warcry` and `trip` have no weapon/skill
prerequisite for a fresh character; (2) the pool affords `warcry` + a cast of `spike`
(`cost: 50`) — top the pool in-room if tight; (3) the tree gates each "now do X"
prompt where the move must *succeed* on `GetCooldown("special-move")==0`, but
deliberately prompts the `trip` *before* readiness returns to show the block message.
**Pacing note:** the effigy is harmless, so Dewey narrates each short wait ("keep
swinging — you'll feel the readiness return").

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

> **Dewey:** "Before we spar — a fighter readies more than fists. Steel yourself
> first: type `warcry` and let the sound harden your nerve."
>
> *(player warcries → gains the Warcry condition)*
>
> **Dewey:** "Feel that lift? You just put an *effect* on yourself — a good one. We
> call those *conditions*. Some raise you up like this; others drag you down — a
> poison, a fear, a leg swept out from under you. Type `conditions` to see what's
> riding on you, and how long it lasts."
>
> *(player checks conditions → the Warcry condition is listed with its countdown)*
>
> **Dewey:** "There's your war cry, ticking down. But shouting like that spent
> something too: your focus. Your biggest moves — shouts, spells, special strikes —
> all draw from one well, and it takes a few rounds to refill. Type `cooldowns` to
> watch it come back."
>
> *(player checks cooldowns → the special-move cooldown is ticking down)*
>
> **Dewey:** "Good. Now, with your blood up — let's fight. Type `attack effigy`.
> Your body swings on its own each round, and those normal swings never tire, well
> or no well."
>
> *(player attacks; auto-swings begin)*
>
> **Dewey:** "Once your focus returns, spend it on *belief* — type `cast spike` to
> drive a spike of raw conviction into it. No robes, no order, no permission: in
> Gaius, belief is a weapon."
>
> *(cooldown clear; player casts → the effigy takes the hit)*
>
> **Dewey:** "Now try `cast spike` again, straightaway — go on."
>
> *(player recasts immediately → real engine message: "You need a moment before you
> can do that." — the shared well is spent)*
>
> **Dewey:** "See? Same well you felt with the war cry. Belief just drew it down, so
> your next big move has to wait a breath. Let it fill — then take its legs the
> body's way: type `trip`."
>
> *(cooldown clears; player trips → the effigy topples)*
>
> **Dewey:** "Down it goes — and now *it* wears a condition: knocked flat, easy
> pickings. Conditions cut both ways. And look what you've done — voice, belief, and
> body, all three, all from the same well. Every one of them is *yours*; no class
> picks one and locks the rest away. You lean where you like, and grow toward it.
> Which brings me to the last thing worth understanding..."

Then the **forced progression tick** fires — the player sees the *genuine* boxed
`SKILL ADVANCEMENT` banner for `spellcasting` (the standard `banner.Format(banner.
Skill, ...)` output they'll see on real progression; **not** the separate
critical-success "*** … technique improves! ***" string, which is a different code
path), and Dewey anchors the primer to it:

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
- **Conditions are real, inspectable state.** `warcry`/`rally` are **self-buffs**
  that apply a combat *condition* to the actor with **no target or combat required**
  (buff 79 "Warcry" — a damage bonus; buff 80 "Rally" — mitigation; ~25 rounds;
  `internal/actions/combat_warcry.go` / `combat_rally.go`). The `conditions` command
  (`internal/usercommands/conditions.go`) lists the player's active buffs + combat
  conditions with **name, description, and rounds left**. `trip` applies a **prone**
  condition to its *target* — the enemy-side example the copy uses. So "warcry →
  `conditions` before combat" and "trip → an enemy condition" are both accurate.

### 6.2 Forced progression tick — implementation note

A short tutorial cannot *rely* on a natural progression roll (~12% per stat check;
~1 skill rank per 25 uses). **No existing behavior-tree action grants a
progression tick** (verified — nothing in `internal/behaviortree` or
`internal/mobcommands` calls `progression.go`), so add a small new room action
**`grant_progression`** (param `skill`). Cleanest guaranteed implementation:
resolve the triggering player via `users.GetByUserId(ctx.Event.UserId)` and call
`char.CheckSkillProgression(skill, userId, 1000.0)` — the chance clamps to 1.0
(`progression.go:108-110`) so a large multiplier makes it certain, and that real
path does the genuine `IncreaseSkill` + tier bookkeeping and emits the standard
`banner.Format(banner.Skill, …)` banner via `events.AddToQueue`. Guard it to fire
**once** with a room `set_state`/`state_equals` flag. This is honest — the player
*did* use the skill — and it teaches recognition of the banner for later. (Gated
on `UseSkillProgression`, which is `true` in prod.)

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

**Create (new IDs — verified free via `id_inventory.py`):**
- **Room 6467** `rooms/newcomer_antechamber/6467.yaml` (the 6th themed room, "The
  Landing") + its behavior tree `behaviors/rooms/newcomer_antechamber/6467.yaml`.
  (6263 is taken by `east_road_to_greenford`; 6467 is the next globally-free room.)
- **Straw effigy** practice-dummy mob **9614** (next globally-free mob) + goals —
  attackable but harmless (`behavior_archetype: combat_passive`, `hostile: false`,
  `maxwander: 0`, high vitality so it survives the lesson), spawned in room 5
  (6262). NOT `non_combatant` (that would make it unattackable). No engine
  invulnerability flag exists, so non-lethality = high HP + zero base damage.

**Reuse (no change):**
- Grey token item `itemid: 2` (the existing antechamber token pickup; relocated
  to the room-3 "What You Carry" room, ID 6260)

**Edit:**
- `_datafiles/config.yaml` — add `6467` to `TutorialRooms`; fix the stale
  "Sanctum Basin replaces the old tutorial zone; TutorialRooms is intentionally
  empty" comment (it is contradicted by the populated array and the live code).

**Code (one small task, flagged for the plan):**
1. **`grant_progression` behavior action** (§6.2) — new room action calling
   `CheckSkillProgression(skill, userId, 1000.0)`; guaranteed one real progression
   banner.

**Instancing note (drove a plan correction):** the antechamber runs as a private
**ephemeral instance**, so in-instance room-to-room movement must use a **real
`north` exit** (which the instance runtime remaps to the player's copy), never
`move_player(templateId)` (that would pull the player into the shared template
room). So room 5 (6262) has a real `north` exit to room 6 (6467); the `flee` lesson
simply runs the real `flee`, which moves the player out that single forward exit.
Only the final hop from room 6 to room 5200 (a real, non-instanced room) uses
`move_player`, exactly as the old terminus did. The effigy's floor stats make flee
reliably succeed; movement/`flee` are blocked only until the combat lesson is done.

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
  room 5200 with quest 30 active. In room 5 specifically: `warcry` applies the
  **Warcry condition** and it appears in the `conditions` list with a countdown;
  `cooldowns` shows the special-move timer after the shout; `cast spike` lands; the
  **immediate recast** of `cast spike` is **blocked** with the real recover message;
  `trip` then lands after the well refills and leaves the effigy **prone**; and a
  fresh character's pool affords `warcry` + the casts. Also confirm **every authored
  room noun is examinable** (`look <noun>` returns its lore prose) — the examine
  lesson in room 1 depends on it.
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
