# Character-Creation Experience Poll & Tailored Onboarding

**Date:** 2026-06-29
**Status:** Design (approved, pre-plan)
**Origin:** Malia's newbie-area playtest (item C1 in
`docs/playtest-feedback-2026-06-29-newbie.md`) + the 2026-06-29 triple smoke
test, which confirmed (a) total beginners need real hand-holding on DOGMud's
systems, (b) the MOTD's "veterans arrive already Awakened" is unimplemented (S3),
and (c) there is no veteran fast-track out of the tutorial (S9).

## 1. Overview

At character creation, ask the player one question about their experience level
and tailor the opening of the game to their answer. Three tiers:

| Tier | Who | Opening |
|------|-----|---------|
| **A — New to MUDs** | Never played a text MUD | A **three-stage learning arc**: a private instanced **tutorial antechamber** (the performable verbs), then **post-Rite teaching** of mutations + progression from Cleric Hadwen, then a **town beat** introducing player interaction — each lesson landing where it becomes real. |
| **B — New to DOGMud** | Plays MUDs, new to this game | **Pothole Coulee, unchanged** — exactly today's experience (the Rite, then the seven trails). Strict regression baseline. |
| **C — Veteran** | Wants to skip onboarding | Confirm, then arrive in **Thornwall already Awakened** (granted `30-end` + a starting mutation), with a `tutorial` back-door to reach the coulee if mis-picked. |

### Goals
- A true beginner learns DOGMud's actual systems — basic verbs, **stats/status**,
  **mutations**, **skills & use-based progression**, **reading helpfiles**,
  **interacting with NPCs and with other players** — each taught at the point it
  is demonstrable, not dumped in a sterile room.
- An experienced player isn't forced through content they don't need.
- The MOTD's veteran promise becomes true.

### Non-goals
- No change to the Pothole Coulee zone layout, the Rite, or the seven trails
  beyond **newbie-gated additive teaching beats** (Tier B sees none of them and
  is a strict no-op baseline).
- No persistent "difficulty" setting. The only durable state is (a) a small
  per-character **onboarding-tier marker** used to gate the Tier-A teaching beats
  (cleared when the arc completes), and (b) the veteran's real `30-end` + mutation.
- No new combat/quest systems; this is onboarding flow + a small content zone +
  conditional teaching beats.

## 2. Background — how onboarding works today (verified)

- **Spawn:** `GamePlay.SpecialRooms.StartRoom = 5200` (The Awakening Pool, Pothole
  Coulee). `TutorialRooms: []` (the legacy ephemeral GoMud tutorial is disabled).
  `MoveToRoom(user, StartRoomIdAlias)` resolves `StartRoomIdAlias (0)` →
  `cfg.StartRoom` (`internal/rooms/roommanager.go`).
- **Character creation is two phases:**
  1. **Account/login wizard** — `internal/inputhandlers/login_prompt_handler.go`
     (a generic, extensible multi-step `PromptStep` framework).
  2. **Character `start`** — `internal/usercommands/start.go`, run in the void
     (`RoomId == -1`). Sets species (human), asks the **name**, then **already
     contains an age-gated "Skip tutorial?" prompt** (shown only if account age >
     1h) that vortexes to `StartRoomIdAlias`; otherwise falls through to the
     (empty) `TutorialRooms` ephemeral path.
- **The Rite (quest 30, "The Awakening"):** `30-start` auto-granted on
  `room_enter` 5200 (HandleJoin fires `room_enter` on spawn). `30-end` + the first
  mutation are granted by **Cleric Hadwen's behavior tree**
  (`behaviors/pothole_coulee/9100-cleric_hadwen.yaml`: `do: grant_mutation` then
  `do: grant_quest 30-end`). Movement out of 5200 is hard-blocked until `30-end`
  (`behaviors/rooms/pothole_coulee/5200.yaml`).
- **Mutation grant** (`actGrantMutation`, `internal/behaviortree/actions_quest.go`):
  `mutations.GetWeightedPool(char.Mutations, species)` → `mutations.RollAcquisition(pool)` → `char.Mutations[id] = 1`.
- **The exit:** Warden Esk's arch (Threshold House 5215) requires `30-end` and
  teleports to Thornwall (468). One-way out.
- **Town footing:** after the Rite, Crier Toke (Hub Square 5203) runs the "Find
  Your Footing" quest (31) walking the player through town — the natural host for
  the Tier-A town teaching beat.

## 3. The experience poll

**Location:** `internal/usercommands/start.go`, asked once **after** the character
name is confirmed (≈ after line 94) and **replacing** the existing age-gated "Skip
tutorial?" block. The poll is the explicit, always-shown successor to that
vestigial prompt; the `duration.Hours() > 1` gate is removed.

**Mechanism:** the existing `cmdPrompt.Ask(question, options, default)` flow
already used for the name/confirm prompts (resumable via `user.StartPrompt`).

**Wording (draft):**
```
How much of Gaius do you already know?
  1) New to text MUDs       — I'll teach you the basics first.
  2) New to DOGMud          — I know MUDs; show me what's different here.
  3) Veteran                — skip all tutorials; drop me into the city.
```
Default: `2` (the safe middle — today's experience).

**Routing by answer:**
- `1` → set the onboarding-tier marker = `newbie` (Section 5), then the ephemeral
  tutorial antechamber (Section 4.1).
- `2` → `MoveToRoom(user, StartRoomIdAlias)` → 5200 (today's default path).
- `3` → confirm, then the veteran path (Section 6).

## 4. Tier A — the total-newbie learning arc

The arc spans three stages so every concept is taught where it's real. The
onboarding-tier marker (Section 5) gates stages 2 and 3 so only Tier-A players see
them.

### 4.1 Stage 1 — instanced tutorial antechamber (performable verbs)

**Tech:** reuse the engine's ephemeral mechanism — populate
`GamePlay.SpecialRooms.TutorialRooms` with the antechamber template IDs and route
Tier A through the existing `CreateEphemeralRoomIds(...)` path in `start.go` (lines
~141–166). Private per-player instance; auto-cleaned.

**Lessons (one room per verb, exit gated until performed):**
1. **`look`** — orient; read the room.
2. **movement** — compass (`north`/etc.); gated on a move.
3. **`status` & stats** — teach reading the stat sheet: the six stats centered at
   100, the three pools (health/stamina/conviction), gold. (This is a key DOGMud
   literacy beat — beginners must learn to read `status`.)
4. **`help`** — teach the help system: `help` index and `help <topic>`; require
   the player to open at least one helpfile (e.g. `help look`). ("How to read
   helpfiles" — explicitly requested.)
5. **`inventory`/`inv`** — seed one trivial item; teach inv + examining an item.
6. **`talk`/`ask <npc>`** — a friendly guide NPC; teach NPC interaction
   (`talk <npc>`, `ask <npc> <topic>`). The guide then sends them on; the final
   exit vortexes to the Awakening Pool (5200).

(Exact room splits are a plan detail; the gating contract + lesson set is fixed.
Room/mob/dialogue IDs via `python tools/id_inventory.py --alloc ...`.)

**Gating mechanism:** room behaviors on the ephemeral rooms (DOGMud already gates
movement behaviorally — see `behaviors/rooms/pothole_coulee/5200.yaml`). Each room
starts with its exit locked, listens for the lesson command, then emits praise +
unlocks. **Plan-time check:** confirm room behaviors can observe each command
(`look`, `go`, `status`, `help`, `inventory`, `talk`/`ask`) on an ephemeral
instance; per-step fallback is a questengine `command` trigger + `unlock_exits`.

**Copy rules:** beginner-first (no "coulee"/"the Opened" jargon); every lesson
shows the exact command; 80-col wrapped; the antechamber is openly a teaching
space ("between sleeping and waking") so light fourth-wall command talk is fine.

### 4.2 Stage 2 — post-Rite: mutations & progression (Cleric Hadwen)

After the Rite grants the player's first mutation, Hadwen gives a **newbie-gated**
teaching beat (additive dialogue node / behavior beat, gated on the onboarding
marker so Tier B never sees it):
- **Mutations:** you now carry one — type `mutations` to read it; mutations are
  how you change and grow; more come with time and deeds.
- **Use-based progression:** DOGMud has **no levels and no XP**. You grow by
  *using* what you have — a stat or skill improves from use. Type `skills` to see
  your skills, `status` to see your stats; both climb as you act.
This lands exactly when the mutation is real and stats/skills exist to inspect.

### 4.3 Stage 3 — town: interacting with other players (Crier Toke / footing)

When the Tier-A player reaches the populated town (folded into Crier Toke's "Find
Your Footing" flow, newbie-gated), introduce **player interaction**: `say` (talk
to the room), `who` (who's online), `tell`/`whisper` (private), and emotes. This
is the natural place — real players are around. Completing this beat (or the
footing quest) **clears the onboarding-tier marker**: the arc is done.

## 5. The onboarding-tier marker

A small per-character marker records the chosen tier so the additive Tier-A beats
(4.2, 4.3) fire only for newbies. Implementation: a `Character` MiscData key
(e.g. `onboarding.tier = "newbie"`), set in `start.go` when Tier A is chosen,
read by Hadwen's stage-2 beat and Toke's stage-3 beat, and cleared when the arc
completes (end of stage 3 / footing). MiscData is already used for similar
per-character bookkeeping (quest cooldowns, conversation counters). Tier B/C do
not set it.

## 6. Tier C — veteran → confirm, auto-Awaken, skip, back-door

**Confirm:** choosing `3` asks `cmdPrompt.Ask("Skip everything and start in the
city, already Awakened? (y/n)", ...)`. `n` returns to the poll.

**Auto-Awaken (must match the Rite's two effects):**
1. Grant `30-end` — via `events.Quest{UserId, QuestToken: "30-end"}` (same path
   `HandleQuestUpdate` processes). Clears the 5200 block and satisfies Esk's gate.
2. Grant a **starting mutation** — via the shared helper (Section 7). Without it a
   "veteran" arrives Opened but mutation-less, contradicting the lore and the
   `mutations` display.

**Skip:** vortex to Thornwall (468) — reuse the vortex copy + `MoveToRoom` +
secret `Look` already in `start.go`'s skip block.

**Welcome + back-door:** on arrival print a short veteran welcome including:
> "New to Gaius after all? Type `tutorial` to be taken to the newcomers' pool."

Add a lightweight **`tutorial` command** (`internal/usercommands/tutorial.go`):
for a low-progress character (strict guard — e.g. no quest progress beyond
`30-end`/a small early-token allowlist) teleport to the Awakening Pool (5200) and
let the normal Coulee flow take over. Refuse for established characters. This is
the mis-pick safety net.

## 7. Shared mutation-grant helper (small refactor)

Extract the roll-and-grant logic inline in `actGrantMutation`
(`internal/behaviortree/actions_quest.go`) into a reusable function — likely a
`characters.Character.GrantRandomMutation() (mutId string)` method (clean import
seam; `mutations` is already a dependency there) — and call it from BOTH
`actGrantMutation` and the veteran path in `start.go`, so the Rite and the veteran
skip never diverge.

## 8. MOTD fix (S3)

Reword `Server.Motd` ("A NEW BEGINNING") in `_datafiles/config.yaml`: replace the
implication that veterans are *automatically* Awakened with language that the
player **chooses** their path at character creation (new to MUDs / new to DOGMud /
veteran). Makes the MOTD accurate and advertises the poll.

## 9. Files to create / change

**New:**
- `internal/usercommands/tutorial.go` (+ registry entry) — the back-door command.
- `_datafiles/world/dogmud/rooms/<tutorial>/<ids>.yaml` — antechamber room
  templates (IDs via `id_inventory.py`).
- `_datafiles/world/dogmud/behaviors/rooms/<tutorial>/<ids>.yaml` — per-room
  gating behaviors.
- `_datafiles/world/dogmud/mobs/<tutorial>/<id>.yaml` (+ dialogue) — the guide NPC.
- Tests: `start.go` poll-routing + veteran-Awaken; `tutorial` command; mutation
  helper; the onboarding-marker set/clear; newbie-gated dialogue beats.

**Changed:**
- `internal/usercommands/start.go` — replace age-gated skip with the 3-way poll +
  routing + onboarding marker + veteran auto-Awaken/skip/welcome.
- `internal/behaviortree/actions_quest.go` — use the shared mutation helper.
- `internal/characters/` (+ `internal/mutations/` as needed) — the helper +
  MiscData marker accessors if not present.
- `_datafiles/world/dogmud/dialogue/pothole_coulee/9100.yaml` (and/or Hadwen's
  behavior) — stage-2 newbie-gated mutations/progression beat.
- `_datafiles/world/dogmud/dialogue/pothole_coulee/9106.yaml` — stage-3
  newbie-gated player-interaction beat in Toke's footing; clear the marker.
- `_datafiles/config.yaml` — `TutorialRooms` populated; `Server.Motd` reworded.
- command registry — register `tutorial`.

## 10. Testing strategy

- **Unit:** poll routes each answer correctly; onboarding marker is set for Tier A
  only and cleared at arc end; veteran path grants `30-end` + a mutation;
  `tutorial` teleports a low-progress char and refuses a high-progress one; the
  shared mutation helper grants from the weighted pool; the stage-2/3 beats fire
  only when the marker is set.
- **Boot:** new room/behavior/mob/dialogue YAML loads clean (no panics; validators
  pass).
- **In-game (harness):** re-run the three smoke personas (they map 1:1 to the
  tiers):
  - total-newbie → antechamber teaches+gates each verb → Rite → Hadwen teaches
    mutations/progression → town teaches player interaction → marker cleared;
  - mud-vet → **regression**: identical to today, sees none of the extra beats;
  - veteran → confirm → Thornwall, `mutations` shows one, movement free, and
    `tutorial` returns them to the pool.

## 11. Risks / open questions

- **Gating observability (4.1):** the one technical unknown — room behaviors
  observing every needed command on an ephemeral instance; questengine
  `command`-trigger fallback per step.
- **Marker lifecycle:** must reliably set at creation and clear at arc end; a
  stuck marker would make a Tier-A player keep getting beats. Keep set/clear
  points explicit and tested.
- **`tutorial` back-door scope:** the low-progress guard must not let established
  characters yank to the pool. Strict.
- **Mutation parity:** the shared helper (Section 7) keeps Rite vs veteran in sync.
- **MUD-vet regression:** highest-value guarantee — Tier B is byte-for-byte
  today's flow; the additive beats are all marker-gated; poll default is `2`.

## 12. Build order (single spec, phased plan)

1. **Framework:** poll in `start.go` + routing + onboarding marker plumbing +
   Tier B (no-op) + Tier C (veteran auto-Awaken/skip/welcome) + `tutorial`
   command + shared mutation helper + MOTD. Shippable alone (newbies temporarily
   route like mud-vets until phase 2).
2. **Antechamber content (stage 1):** ephemeral room templates + gating behaviors
   + the guide NPC, wired into `TutorialRooms` and the Tier-A route.
3. **Downstream teaching beats (stages 2–3):** Hadwen's post-Rite
   mutations/progression beat + Toke's town player-interaction beat, both
   marker-gated, with the marker cleared at the end.

All under this one spec; the plan sequences them so each phase is independently
verifiable.
