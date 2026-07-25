# NPC Greetings (Admin Web-Building Sub-Project 5a) — Design

**Date:** 2026-07-25
**Status:** Approved design, pre-implementation
**Epic:** Admin web-building. Sub-project 5 (dialogue / behavior trees /
quests) was decomposed during brainstorming; this is **5a**, and the dialogue
editor becomes **5b**.
**Predecessor specs:** `2026-07-23-mob-authoring-3-design.md`,
`2026-07-25-zone-lifecycle-config-design.md`,
`2026-07-25-spawn-list-editor-design.md`

## Goal

Make the `greetings:` block that **186 of 302 dialogue files already author**
actually reach players.

This is not a new feature so much as the connection of an existing one. The
content exists, was written deliberately, and has never run.

## Why this comes before the dialogue editor

Sub-project 5 was going to be the dialogue editor. Measuring the data first
turned up something that changes the order: `greetings:` is a top-level key in
186 dialogue files and **there is no such field on `dialogue.DialogueFile`**.
The lenient YAML decode drops it silently.

Building an editor against a struct we already know is wrong would mean either
rendering a field the engine ignores, or hiding authored content from the
person editing it. So the struct gets fixed first, and 5b is built against a
`DialogueFile` that tells the truth.

### How it escaped every existing gate

Verified by strict-unmarshal probe over all 302 files: 186 report `greetings`
as an unknown field.

- The **YAML drift gate** (`TestSmoke_NoNewSilentlyIgnoredYAMLKeys`) only
  probes what `loadAllDataFiles` loads, and **dialogue is lazy-loaded** — it
  never sees a dialogue file.
- **`TestSmoke_AllDialogueFilesParse`** does read every dialogue file, but only
  asserts that `yaml.Unmarshal` returns no error. Unknown keys are not an error
  under lenient decoding.

The two gates were built for different failure modes and this fell between
them. §7 closes the gap so no future dialogue field can do the same.

### The content is real, not redundant

Compared each dead greeting against the tree-root text that NPC currently
serves:

| | count |
|---|---|
| greeting is **distinct prose** | **185** |
| greeting duplicates the root | 1 |
| files with more than one greeting (mood variants) | 54 |

The two registers are consistently different. A greeting is a *welcome*:

> "Welcome to the Golden Bough — best beds in Amber Valley and the only ear in
> it that hears everything. Come in, come in, you look road-worn."

The tree root is a *self-introduction*:

> "Hesper Vane, and this is my house — the Golden Bough. If a thing happens in
> Amber Valley, it gets argued about over my tables within the hour."

The root text is what a player currently receives **every time** they talk to
that NPC, including the fiftieth time. The greetings were written as
first-contact lines — "come in, come in", "mind the shavings, pull up that
stool" — which is why they read as room-entry dialogue rather than answers.

## Scope decisions (user-confirmed)

- **Fires on room entry**, ambient. It does not replace the tree root, which
  keeps serving `talk`.
- **One per NPC per player per boot**, via the existing in-process memory.
- Mood variants selected by the NPC's current mood.
- Suppressed while asleep / in combat / mid-schedule, and for hidden players.
- Delivered as NPC speech so the channel and `deafen` systems apply.
- At most one greeting per room entry.

## 1. Data model

```go
// Greeting is one ambient line an NPC offers when a player arrives.
type Greeting struct {
    Text  string   `yaml:"text"`
    Moods []string `yaml:"moods,omitempty"`
}
```

added to `DialogueFile` as:

```go
    Greetings []Greeting `yaml:"greetings,omitempty"`
```

**No content files change.** The struct is being shaped to the YAML that
already exists, not the other way round. A parse test over all 302 files
confirms the block loads into the new field with no edits.

## 2. Trigger

`internal/usercommands/go.go`, in the player-arrival section that already
holds the chunk-3.6 conversation boost (`go.go:702`). That block is the
established precedent for "an NPC reacts because a player just walked in", and
the greeting hook sits beside it rather than inventing a second seam.

Ordering: the greeting fires **before** the conversation-boost roll, so an
arriving player is welcomed before two NPCs start talking amongst themselves.

## 3. Frequency

Gated on the existing per-`(mobInstanceId, userId)` memory
(`internal/dialogue/memory.go`), which is an in-process map — so "once per
boot" is the natural consequence rather than a rule needing new storage.

`PlayerMemory` gains one field:

```go
    Greeted bool // this player has been greeted by this mob instance
```

Consequences, accepted deliberately:

- A mob that dies and respawns gets a **new instance id**, so it greets that
  player again. For a killed-and-respawned shopkeeper this reads correctly.
- On a long-running server a returning player is not greeted again until a
  reboot. This is tied to server lifecycle, which players cannot perceive.
  The alternative — gating on `MemoryConfig.ExpiryPeriod`, already authored in
  106 files — was considered and **not** chosen; recorded here so the tradeoff
  is visible if the behaviour ever feels wrong in play.

## 4. Mood selection

54 files carry multiple greetings tagged with `moods:`. Selection:

1. First greeting whose `moods` contains the NPC's current mood.
2. Otherwise the first greeting with no `moods` (the unconditional line).
3. Otherwise none — say nothing rather than deliver a line written for a mood
   the NPC is not in.

Mood comes from the same source the pattern matcher already uses, so a mood
change mid-session changes which greeting a later instance offers.

## 5. Suppression

No greeting when:

| condition | why |
|-----------|-----|
| NPC asleep (`buffs.Sleeping`) | a sleeping innkeeper cannot welcome anyone |
| NPC in combat | it is busy, and the line would read absurdly |
| NPC mid-schedule-activity | scheduled NPCs are deliberately occupied |
| player entered **hidden** | a sneaking thief must not be hailed by name — this would otherwise silently defeat stealth |
| NPC has no greeting for its mood | §4 |

The hidden-player case is the one with a mechanical consequence rather than a
cosmetic one, and it must be covered by a test.

## 6. Delivery

Routed through the NPC speech path, not raw room text, so:

- `deafen` suppresses it like other NPC speech,
- it appears on the same channel players already filter,
- it is attributed to the NPC rather than appearing as narration.

At most **one** greeting per room entry. Measured: 160 rooms spawn exactly one
greeting-capable NPC, 9 spawn two, none spawn three or more — so a first-match
cap is sufficient and no arbitration rule is needed.

## 7. Closing the gate gap

`TestSmoke_AllDialogueFilesParse` is extended to also **strict**-unmarshal each
dialogue file and report unknown keys, with a baseline map in the same shape as
`knownSilentlyIgnoredKeys`.

Without this, the next dialogue field authored-but-never-implemented repeats
exactly this incident: content written, never run, invisible to CI, discovered
by accident months later. After this change `greetings` resolves, so the
baseline starts empty and any new unknown key fails immediately.

This is the durable half of the work. The greeting feature is worth having;
the gate is what stops the same class recurring.

## 8. Content fix

Exactly one of the 186 greetings duplicates its NPC's tree-root text. Left
alone, that NPC would greet a player and then repeat itself verbatim on `talk`.
The implementation identifies it and rewrites the greeting to a distinct
welcome line, in the NPC's established voice.

## 9. Testing & verification gate

**Unit** (`internal/dialogue`): greeting selection picks the mood-matching
variant; falls back to the untagged line; returns none when neither exists;
`Greeted` suppresses a second greeting for the same instance/player; a new
instance id greets again.

**Parse**: all 302 dialogue files load with the `greetings` block populated
into the new field, and the count of files with a non-empty `Greetings` is
**186** — the same number the probe found, so the field is not silently
matching a subset.

**Suppression**: table test covering asleep / combat / hidden player, asserting
silence in each.

**Gate**: extended dialogue strict-parse test passes with an empty baseline.

**Boot**: clean under `MapConsistencyEnforce: panic`.

**CONTENT ADVERSARIAL PLAYTEST — REQUIRED.** This changes what 186 NPCs do in
front of players, and CLAUDE.md's content gate applies: boot-clean does not
verify an experience. Run the playtest harness with a critical mandate, walk
into greeting rooms as a confused newcomer, and read every line. Specifically
watch for: greetings that read oddly out of their intended context, a greeting
immediately followed by a near-identical root text on `talk`, and greeting spam
in the 9 two-greeter rooms.

## Out of scope

- The **dialogue editor** — now 5b, built against the corrected struct.
- `MemoryConfig.ExpiryPeriod`-based frequency (§3, considered and rejected).
- Rewriting any greeting prose beyond the single duplicate in §8.
- Quest and behavior-tree editors (5c/5d), still deferred.
