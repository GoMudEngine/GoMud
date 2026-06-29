# Character-Creation Experience Poll & Tailored Onboarding

**Date:** 2026-06-29
**Status:** Design (approved, pre-plan)
**Origin:** Malia's newbie-area playtest (item C1 in
`docs/playtest-feedback-2026-06-29-newbie.md`) + the 2026-06-29 triple smoke
test, which confirmed (a) total beginners need more basic-mechanics hand-holding,
(b) the MOTD's "veterans arrive already Awakened" is unimplemented (S3), and
(c) there is no veteran fast-track out of the tutorial (S9).

## 1. Overview

At character creation, ask the player one question about their experience level
and tailor the opening of the game to their answer. Three tiers:

| Tier | Who | Opening |
|------|-----|---------|
| **A — New to MUDs** | Never played a text MUD | A private, instanced **tutorial antechamber** that drills the basic verbs (look / move / inventory / status / talk) one at a time, then funnels into Pothole Coulee for the Awakening Rite. |
| **B — New to DOGMud** | Plays MUDs, new to this game | **Pothole Coulee, unchanged** — exactly today's experience (the Rite, then the seven trails). |
| **C — Veteran** | Wants to skip onboarding | Confirm, then arrive in **Thornwall already Awakened** (granted `30-end` + a starting mutation), with a back-door to reach the coulee if they mis-picked. |

### Goals
- A true beginner is taught how to play before being asked to play.
- A returning/experienced player is not forced through content they don't need.
- The MOTD's veteran promise becomes true.

### Non-goals
- No change to the Pothole Coulee zone, the Rite, or the seven trails (Tier B is
  a strict no-op / regression baseline).
- No persistent "difficulty" or per-account profile beyond what each tier does at
  creation time. The tier is a routing decision, not stored long-term (the only
  durable effects are the veteran's real `30-end` token + mutation).
- No new combat/quest systems; this is onboarding flow + a small content zone.

## 2. Background — how onboarding works today (verified)

- **Spawn:** `GamePlay.SpecialRooms.StartRoom = 5200` (The Awakening Pool, Pothole
  Coulee). `TutorialRooms: []` (the legacy ephemeral GoMud tutorial is disabled).
  `MoveToRoom(user, StartRoomIdAlias)` resolves `StartRoomIdAlias (0)` →
  `cfg.StartRoom` (`internal/rooms/roommanager.go`).
- **Character creation is two phases:**
  1. **Account/login wizard** — `internal/inputhandlers/login_prompt_handler.go`
     (a generic, extensible multi-step `PromptStep` framework: username, password,
     email, screen-reader, etc.), wired in `internal/inputhandlers/login.go`.
  2. **Character `start`** — `internal/usercommands/start.go`, run while the
     character is in the void (`RoomId == -1`). Sets species (human), asks the
     character **name**, then **already contains an age-gated "Skip tutorial?"
     prompt** (only shown if account age > 1h) that vortexes the player to
     `StartRoomIdAlias`; otherwise it falls through to the (empty) `TutorialRooms`
     ephemeral path.
- **The Rite (quest 30, "The Awakening"):** `30-start` auto-granted on
  `room_enter` 5200 (HandleJoin fires `room_enter` on spawn). `30-end` + the
  first mutation are granted by **Cleric Hadwen's behavior tree**
  (`behaviors/pothole_coulee/9100-cleric_hadwen.yaml`: `do: grant_mutation` then
  `do: grant_quest 30-end`). Movement out of 5200 is hard-blocked until `30-end`
  (`behaviors/rooms/pothole_coulee/5200.yaml`).
- **The mutation grant** (`actGrantMutation`,
  `internal/behaviortree/actions_quest.go`): `mutations.GetWeightedPool(char.Mutations, species)` → `mutations.RollAcquisition(pool)` → `char.Mutations[id] = 1`.
- **The exit:** Warden Esk's arch (Threshold House 5215) requires `30-end` and
  teleports to Thornwall (468). One-way out (no walking route back to the coulee).

## 3. The experience poll

**Location:** `internal/usercommands/start.go`, asked once, **after** the
character name is confirmed (≈ after line 94) and **replacing** the existing
age-gated "Skip tutorial?" block (lines ~100–137). The poll is the explicit,
always-shown successor to that vestigial prompt.

**Mechanism:** uses the existing `cmdPrompt.Ask(question, options, default)`
flow already used for the name/confirm prompts in `start.go` (resumable across
input cycles via `user.StartPrompt`).

**Wording (draft):**
```
How much of Gaius do you already know?
  1) New to text MUDs       — I'll teach you the basics first.
  2) New to DOGMud          — I know MUDs; show me what's different here.
  3) Veteran                — skip all tutorials; drop me into the city.
```
Default: `2` (the safe middle — the current experience).

**Routing by answer:**
- `1` → ephemeral tutorial antechamber (Section 4).
- `2` → `MoveToRoom(user, StartRoomIdAlias)` → 5200 (today's default path).
- `3` → confirm (Section 6), then the veteran path.

The legacy `duration.Hours() > 1` gate is removed — experience is now an explicit
choice for everyone, not an inference from account age.

## 4. Tier A — total newbie → instanced tutorial antechamber

**Tech:** reuse the engine's ephemeral mechanism. Populate
`GamePlay.SpecialRooms.TutorialRooms` with the antechamber template room IDs and
route Tier A through the existing `CreateEphemeralRoomIds(...)` path already coded
in `start.go` (lines ~141–166). Each newbie gets a private instance; the rooms
auto-clean.

**Rooms & lessons (~4–5 ephemeral templates):** each room teaches exactly one
verb and will not open its exit until the player performs it.

1. **Waking dark** — teach `look`. Exit opens after the player looks.
2. **A first step** — teach movement (`north`/compass). Exit gated on a move
   attempt.
3. **What you carry** — teach `inventory`/`inv` (seed one trivial item to look
   at). Exit gated on `inv`.
4. **Knowing yourself** — teach `status` (stats, gold, health). Exit gated on
   `status`.
5. **A voice ahead** — teach `talk`/`ask <npc>` against a friendly tutorial NPC,
   who then sends them on. The final exit vortexes to the Awakening Pool (5200);
   from there the normal Rite + Coulee proceed unchanged.

(Exact room count/splits are a plan-time detail; the gating pattern is the
contract. New room template IDs to be allocated via `python tools/id_inventory.py
--alloc rooms N`. The antechamber NPC needs a mob ID likewise.)

**Content rules:** every lesson states the exact command to type; copy is
beginner-first (no jargon — no "coulee", no "the Opened"); 80-col wrapped. The
antechamber is explicitly a teaching space ("a quiet place between sleeping and
waking"), so it can break the fourth wall lightly about commands.

## 5. Antechamber gating mechanism

**Recommended: room behaviors** on the ephemeral antechamber rooms, mirroring the
existing movement-gate pattern (`behaviors/rooms/pothole_coulee/5200.yaml` already
gates exits behaviorally). Each antechamber room's behavior:
- starts with its exit(s) locked,
- listens for the lesson's command event from the player,
- on success: emits a short "well done" + unlocks the exit (and may auto-prompt
  the next hint).

**Rejected alternative:** a throwaway tutorial *quest* (questengine `command`
triggers + `lock_exits`/`unlock_exits`). It works but writes persistent quest
tokens for a one-time instanced tutorial, and couples a teaching flow to the
quest log. Room behaviors keep the tutorial self-contained and stateless beyond
the instance.

**Plan-time check:** confirm room behaviors can observe player command events
(`look`, `go`, `inventory`, `status`, `talk`) and lock/unlock exits on an
ephemeral room instance. If a needed command isn't observable by a room behavior,
fall back to the questengine `command`-trigger approach for that step only.

## 6. Tier C — veteran → confirm, auto-Awaken, skip, back-door

**Confirm:** choosing `3` asks a one-line `cmdPrompt.Ask("Skip everything and
start in the city, already Awakened? (y/n)", ...)`. `n` returns to the poll.

**Auto-Awaken (must match the Rite's two effects):**
1. Grant `30-end` — via `events.Quest{UserId, QuestToken: "30-end"}` (the same
   path `HandleQuestUpdate` processes; also fires quest-30 completion handling).
   This clears the 5200 movement block and satisfies Esk's gate.
2. Grant a **starting mutation** — call the shared mutation-grant helper (Section
   7). Without this, a "veteran" arrives Opened but mutation-less, which
   contradicts the lore ("already Awakened") and the `mutations` display.

**Skip:** vortex to Thornwall (468) — reuse the vortex copy + `MoveToRoom` +
secret `Look` already in `start.go`'s skip block.

**Welcome + back-door:** print a short veteran welcome on arrival that includes:
> "New to Gaius after all? Type `tutorial` to be taken to the newcomers' pool."

Add a lightweight **`tutorial` command** (`internal/usercommands/tutorial.go`):
for a low-progress character (heuristic: no quest progress beyond `30-end`, or a
small allowlist of early tokens), teleport to the Awakening Pool (5200) and let
the normal Coulee flow take over. Guard against abuse/mid-game use (e.g., refuse
if the character has meaningful progress). This is the mis-pick safety net.

## 7. Shared mutation-grant helper (small refactor)

Extract the roll-and-grant logic currently inline in `actGrantMutation`
(`internal/behaviortree/actions_quest.go`) into a reusable function, e.g.
`mutations.GrantRandom(char *characters.Character) (mutId string)` or a
`characters.Character` method, and call it from BOTH `actGrantMutation` and the
veteran path in `start.go`. Keeps the two Awakening paths (Rite vs veteran skip)
from diverging. (Watch the import direction: `mutations` already imports neither
`characters` cyclically in `actGrantMutation` — confirm a clean home for the
helper during the plan; a `characters.Character.GrantRandomMutation()` may be the
cleaner seam.)

## 8. MOTD fix (S3)

Update `Server.Motd` (in `_datafiles/config.yaml`) "A NEW BEGINNING" blurb:
replace the implication that veterans are *automatically* Awakened with language
that they **choose** their path at character creation (new player / new to DOGMud
/ veteran). Makes the MOTD accurate and advertises the poll.

## 9. Files to create / change

**New:**
- `internal/usercommands/tutorial.go` — the back-door command (+ registry entry).
- `_datafiles/world/dogmud/rooms/<tutorial>/<ids>.yaml` — ~4–5 antechamber room
  templates (IDs via `id_inventory.py`).
- `_datafiles/world/dogmud/behaviors/rooms/<tutorial>/<ids>.yaml` — per-room
  gating behaviors.
- `_datafiles/world/dogmud/mobs/<tutorial>/<id>.yaml` (+ dialogue) — the
  antechamber's friendly "talk" NPC.
- Tests: `start.go` poll-routing test; `tutorial` command test; mutation-helper
  test.

**Changed:**
- `internal/usercommands/start.go` — replace age-gated skip with the 3-way poll +
  routing + veteran auto-Awaken/skip/welcome.
- `internal/behaviortree/actions_quest.go` — use the shared mutation helper.
- `internal/mutations/` or `internal/characters/` — the shared helper.
- `_datafiles/config.yaml` — `TutorialRooms` populated; `Server.Motd` reworded.
- `internal/usercommands/usercommands.go` (or wherever the command registry is) —
  register `tutorial`.

## 10. Testing strategy

- **Unit:** `start.go` poll routes each answer to the right destination (mock the
  prompt + `MoveToRoom`); veteran path grants `30-end` + a mutation; `tutorial`
  command teleports a low-progress char and refuses a high-progress one; the
  shared mutation helper grants from the weighted pool.
- **Boot:** new room/behavior/mob/dialogue YAML loads clean (no panics; schedule/
  zone validators pass).
- **In-game (harness):** re-run the three smoke personas — they map 1:1 to the
  tiers:
  - total-newbie → antechamber teaches each verb and gates correctly, then lands
    at the pool;
  - mud-vet → **regression**: identical to today's Coulee;
  - veteran → confirm → Thornwall, `mutations` shows one, movement is free, and
    `tutorial` returns them to the pool.

## 11. Risks / open questions

- **Gating observability (Section 5):** whether room behaviors can observe every
  needed command on an ephemeral instance — the one technical unknown; plan must
  verify, with the questengine `command`-trigger fallback per step.
- **`tutorial` back-door scope:** the low-progress heuristic must not let
  established characters yank themselves to the pool. Keep the guard strict.
- **Mutation parity:** if `grant_mutation`'s pool/rules change later, the veteran
  path must stay in sync — the shared helper (Section 7) is the mitigation.
- **MUD-vet regression:** the highest-value guarantee — Tier B must be byte-for-
  byte today's flow. The poll default is `2` precisely so a fall-through is safe.

## 12. Build order (single spec, phased plan)

1. **Framework:** poll in `start.go` + routing + Tier B (no-op) + Tier C
   (veteran auto-Awaken/skip/welcome) + `tutorial` command + shared mutation
   helper + MOTD. This is shippable on its own (newbies temporarily route like
   mud-vets until phase 2).
2. **Antechamber content:** the ephemeral room templates + gating behaviors +
   the talk NPC, wired into `TutorialRooms` and the Tier A route.

Both land under this one spec; the plan sequences them so phase 1 is independently
verifiable.
