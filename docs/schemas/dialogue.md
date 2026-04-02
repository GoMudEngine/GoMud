# Dialogue Schema Reference

## 1. Filename & Location

**Path formula:**
```
_datafiles/world/dogmud/dialogue/{zone_folder}/{mobid}.yaml
```

- `{zone_folder}` = `ConvertForFilename(zone display name)` — same conversion used by rooms and mobs.
- `{mobid}` = the integer mob ID. The filename IS the ID (same convention as rooms).

**Worked example:**
- Zone: `Sanctum Basin`, Mob ID: `50`
- Path: `_datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml`

**Relationship to LLMProfile:**
The dialogue YAML is the **fallback** for any mob that has an LLMProfile defined. When Ollama is unreachable or the LLM returns nothing usable, the engine falls back to the dialogue file for responses. For mobs *without* an LLMProfile, the dialogue file is the primary conversation system. Always provide a dialogue file for any NPC players are expected to interact with.

---

## 2. Field Reference

### Top-level Fields

| Field | Type | Required | Notes |
|-------|------|----------|-------|
| `mobid` | int | **yes** | Must match filename and the mob YAML's `mobid`. |
| `zone` | string | **yes** | Display name of the zone. |
| `defaultMood` | string | no | Starting mood: `"friendly"`, `"neutral"`, `"cautious"`, `"hostile"`. Default: `"neutral"`. |
| `greetings` | list | no | Mood-aware greeting messages sent when a player enters the room. |
| `patterns` | list | no | Keyword-matching response patterns for freeform `say` input. |
| `tree` | object | no | Structured conversation tree for guided dialogue. |
| `memory` | object | no | Memory retention settings. |

### Greeting Node

```yaml
greetings:
  - text: "Welcome, traveler."
    moods: ["friendly", "neutral"]   # Only shown in these moods
  - text: "Keep your distance."
    moods: ["hostile", "cautious"]
```

### Pattern Entry

```yaml
patterns:
  - keywords: ["hello", "hi", "greet", "hey"]   # Any of these trigger this pattern
    moods: ["neutral", "friendly"]               # (optional) only match in these moods
    moodChange: friendly                         # (optional) shift mood after responding
    responses:                                   # Randomly chosen from this list
      - "Peace be upon you, seeker."
      - "Welcome. The Chrysalis watches over all."
```

| Pattern Field | Type | Notes |
|---------------|------|-------|
| `keywords` | list | Any keyword match triggers this pattern. Case-insensitive. |
| `responses` | list | One response chosen randomly on each match. |
| `moods` | list | (optional) Only match if mob's current mood is in this list. |
| `moodChange` | string | (optional) New mood after responding. |
| `questRequired` | list | (optional) Quest tokens the player must have (e.g. `["5-start"]`). |
| `questExcluded` | list | (optional) Quest tokens that hide this pattern if present. |
| `grantsQuest` | string | (optional) Quest token granted on match (e.g. `"5-start"`). |
| `requiresItem` | int | (optional) Item ID the player must hold; consumed on match. |
| `givesItem` | int | (optional) Item ID given to the player on match. |
| `questFlagRequired` | map | (optional) Quest flags the player must have (e.g. `{"11-branch": "rhett"}`). |
| `questFlagExcluded` | map | (optional) Quest flags that hide this pattern if matching. |
| `setsQuestFlag` | object | (optional) Flag to set on match: `{key: "11-branch", value: "rhett"}`. |

### Tree Root

```yaml
tree:
  root:
    text: "What brings you here, seeker?"
    hints: "You could ask about the Chrysalis or the Awakening."
    variants:                           # Quest-based greeting variants
      - questRequired: ["5-start"]
        questExcluded: ["5-end"]
        text: "Have you found the ledger yet?"
        hints: "Search the tollhouse for the crossing ledger."
      - questRequired: ["5-end"]
        text: "You did good work. The magistrate has the evidence now."
```

When a player initiates `talk`, the engine checks `variants` first (in order).
The first variant whose `questRequired`/`questExcluded` conditions are met is
used instead of the default `text`. If no variant matches, the default root
text is shown.

Root variants also support `questFlagRequired` and `questFlagExcluded` for
flag-based gating. This is essential for branching quests where the same
quest token is shared but the path differs.

### Tree Nodes

```yaml
tree:
  nodes:
    - id: chrysalis
      triggers: ["chrysalis", "cult", "order", "faith"]   # Keywords that navigate to this node
      text: "The Chrysalis is not a faith of dogma, but of transformation."
      hints: "You could ask about the Awakening or the Priest's role."
      unlocks: ["awakening", "role"]      # Node IDs made available after this one
      requires: []                        # Node IDs that must be visited first
      moodChange: friendly
```

| Tree Node Field | Type | Notes |
|-----------------|------|-------|
| `id` | string | Unique within this dialogue file. |
| `triggers` | list | Keywords that navigate to this node from the parent. |
| `text` | string | The NPC's response when this node is reached. |
| `hints` | string | Hint text shown to the player after the NPC speaks. |
| `unlocks` | list | Node IDs that become reachable after visiting this node. |
| `requires` | list | Node IDs that must have been visited before this one is reachable. |
| `moodChange` | string | Mood shift when this node is reached. |
| `moods` | list | Only reachable when mob's current mood is in this list. |
| `questRequired` | list | (optional) Quest tokens the player must have. |
| `questExcluded` | list | (optional) Quest tokens that hide this node if present. |
| `grantsQuest` | string | (optional) Quest token granted when this node is reached. |
| `requiresItem` | int | (optional) Item ID the player must hold; consumed on activation. |
| `givesItem` | int | (optional) Item ID given to the player when this node is reached. |
| `questFlagRequired` | map | (optional) Quest flags the player must have (e.g. `{"11-branch": "rhett"}`). |
| `questFlagExcluded` | map | (optional) Quest flags that hide this node if matching. |
| `setsQuestFlag` | object | (optional) Flag to set when reached: `{key: "11-branch", value: "rhett"}`. |

### Memory Sub-object

```yaml
memory:
  expiryPeriod: "1h"    # How long conversation context is retained ("1h", "30m", "24h")
```

### Mood Values

| Value | Meaning |
|-------|---------|
| `friendly` | Warm, open, helpful. Likely to share information freely. |
| `neutral` | Default. Professional, measured, neither welcoming nor hostile. |
| `cautious` | Guarded. May withhold information. |
| `hostile` | Aggressive or contemptuous. May refuse to engage. |

---

## 3. Annotated Example

```yaml
# _datafiles/world/dogmud/dialogue/sanctum_basin/50.yaml
mobid: 50
zone: Sanctum Basin
defaultMood: neutral             # Starts neutral; shifts based on conversation

patterns:
  - keywords: ["hello", "hi", "greet", "hey"]
    moods: ["neutral", "friendly", "grateful"]  # Only responds in these moods
    responses:
      - "Peace be upon you, seeker."
      - "Welcome. The Chrysalis watches over all who pass through the Basin."

  - keywords: ["chrysalis", "cult", "faith", "belief"]
    responses:
      - "The Chrysalis is not a faith of dogma, but of transformation."
    # No moodChange — this topic doesn't shift mood

  - keywords: ["awakening", "transform", "change", "become"]
    responses:
      - "The Awakening is what stirs within all creatures..."
    moodChange: friendly         # Talking about the Awakening warms the Priest

  - keywords: [""]               # Catch-all fallback
    responses:
      - "Speak your question clearly. I am listening."

tree:
  root:
    text: "Seeker. You have come to the right place. What weighs on you?"
    hints: "Ask the Priest about the Chrysalis, the Awakening, or the sanctum."

  nodes:
    - id: chrysalis
      triggers: ["chrysalis", "cult", "order", "faith"]
      text: "The Chrysalis does not recruit. Those drawn here arrive because something in them is already changing."
      hints: "Ask about the Awakening, or the Priest's role here."
      unlocks: ["awakening", "role"]

    - id: awakening
      triggers: ["awakening", "transform", "change", "become"]
      text: "The Awakening is the deep current in all living things..."
      moodChange: friendly
      unlocks: ["awakening_manifest"]

    - id: awakening_manifest
      triggers: ["manifest", "how", "example", "witness"]
      requires: ["awakening"]    # Only reachable after visiting 'awakening' node
      text: "I have seen a frightened child become a capable guardian."

memory:
  expiryPeriod: "1h"             # Conversation context resets after 1 real hour
```

---

## 4. Gotchas

**Dialogue file is the fallback, not the primary, for LLM mobs.**
If a mob has an `llmprofile`, the engine tries Ollama first. The dialogue YAML only activates when Ollama is unreachable or returns nothing. Both systems should be populated for any important NPC.

**Zone folder must match `ConvertForFilename(zone)` — same as rooms and mobs.**
`sanctum-basin/` panics. Use `sanctum_basin/`.

**`requires:` checks visited nodes, not mood.**
A node with `requires: ["chrysalis"]` is only reachable if the player has visited the `chrysalis` node in this session (subject to `memory.expiryPeriod`).

**`unlocks:` must reference node IDs that exist.**
A typo in an `unlocks` list silently makes those nodes unreachable. Double-check all node ID references.

**The catch-all pattern `keywords: [""]`.**
An empty-string keyword acts as a catch-all fallback. Always include one at the end of the patterns list to handle unexpected input gracefully.

**`defaultMood` vs `moodChange`.**
`defaultMood` is the mob's starting mood when it first spawns. `moodChange` in a pattern or node shifts the current mood. Mood persists across conversation turns but resets on mob respawn.

**mobid in dialogue must match the mob YAML.**
If mob 50 is named "Chrysalis Priest" in its mob YAML, the dialogue file must be `50.yaml` in the appropriate zone folder. The engine links them by mob ID only.

**Quest tokens use the format `{questId}-{stepId}`.**
For example, `"5-start"` means quest 5, step "start". The engine checks
`HasQuest()` which returns true if the player is on that step or any later
step. Use `questExcluded` to hide options after a quest advances past a
certain point.

**`requiresItem` consumes the item.**
When a node or pattern has `requiresItem: 21`, the engine removes item 21
from the player's backpack on activation. The item check and removal happen
atomically with the dialogue response.

**`grantsQuest` fires the quest event handler.**
The quest event handler processes all rewards (gold, items, buffs) defined
in the quest YAML. You do not need a separate reward mechanism — just point
`grantsQuest` at the right quest token.

**`grantsQuest` nodes must exclude the quest's end token.**
Every node or pattern with `grantsQuest` must include the quest's end token
(e.g., `{questid}-end`) in `questExcluded`, not just the token being granted.
Without this, a player who has completed the quest can be re-offered it.
Example: a node granting `"10-start"` must have
`questExcluded: ["10-start", "10-end"]`. The dialogue loader logs a warning
at runtime if this exclusion is missing.

**Greeting `variants` are checked in order.**
The first variant whose conditions match wins. Put more specific conditions
(e.g. quest completed) before less specific ones (e.g. quest started).

**Quest-granting nodes MUST include `"quest"` and `"task"` triggers.**
Every tree node with a `grantsQuest` field must include `"quest"` and `"task"`
in its `triggers` list. This ensures players can always type
`ask <npcname> quest` or `ask <npcname> task` to discover available quests.
The same applies to `patterns` entries that introduce quests — include
`"quest"` and `"task"` in the `keywords` list. This is an SOP for all quest
NPCs.

**All dialogue must use correct narrative voice — never 3rd-person self-reference.**
NPC `text` fields are spoken by the NPC — they must use first person ("I",
"my", "me"). NPC `hints` fields are narrator text displayed to the player in
bracketed narrator style (e.g. `[You could ask about the Awakening.]`) — they
must describe what the player sees or can do, never refer to the NPC in first
person. **Never write hints like "Ask about why she left" where "she" is the
NPC speaking.** Instead: "You could ask why she left" (narrator describing the
player's option) or "You could ask about her past." Preferred pattern for
hints: describe the player's options from the player's perspective — e.g.,
"You could ask about the marriage, or about her father."

**Triggers must match what hints suggest.**
If a hint tells the player to "calm her down" or "tell her you mean no harm",
the triggers list MUST include words from those phrases ("calm", "harm",
"peace"). If a player reads the hint and types the obvious keyword, it must
work. Undiscoverable triggers are broken triggers.

**Prefer `questRequired` over `requires` for quest-gated nodes.**
`requires` depends on per-player memory of visited nodes, which can expire
via `expiryPeriod` and brick quests. If a node should only appear after a
quest step, use `questRequired` — quest tokens are permanent and reliable.
Only use `requires` for non-quest conversational branching within a single
sitting (e.g., "ask about X before Y becomes available").

**`expiryPeriod` should almost never be set.**
Memory expiry causes `requires`-gated nodes to become unreachable,
bricking quests mid-progress. The ONLY valid use case for `expiryPeriod`
is a quest where urgency is the explicit design intent — e.g., "deliver
this letter before the trolls attack the city!" where failing the timer
is a designed outcome. For all other NPCs, leave `expiryPeriod` empty
or omit the `memory` section entirely.

---

## 5. Branching Quests with Flags

Quest flags enable branching quests where players choose between mutually
exclusive paths. Unlike quest tokens (which track linear step progress),
flags record **which choice** the player made.

### Flag Declaration (Quest YAML)

Flags must be declared in the quest's YAML file. Undeclared flag
references cause a server panic at startup.

```yaml
# _datafiles/world/dogmud/quests/11-the_windwardens_dilemma.yaml
questid: 11
name: The Windwarden's Dilemma
flags:
  - key: branch
    values: [sylara, rhett]
    description: "Which NPC the player sided with"
steps:
  - id: start
  - id: end
```

The fully-qualified flag key is `"{questid}-{key}"` — e.g., `"11-branch"`.

### Setting Flags (Dialogue Nodes)

Use `setsQuestFlag` on the dialogue node where the player makes their
choice:

```yaml
    - id: her_cause
      triggers: ["circle", "conflict", "quest", "task", "help"]
      questExcluded: ["11-start"]
      grantsQuest: "11-start"
      setsQuestFlag:
        key: "11-branch"
        value: "sylara"
      text: "The Windwarden explains the conflict..."
```

Both NPCs grant the same step (`11-start`) but set different flag values.
The flag records which path was taken.

### Gating on Flags (Dialogue Nodes)

Use `questFlagRequired` and `questFlagExcluded` to show different content
based on the player's choice:

```yaml
    # Only Sylara-path players see this
    - id: offer_covenant
      triggers: ["covenant", "quest", "task"]
      questRequired: ["11-end"]
      questFlagRequired:
        "11-branch": "sylara"
      grantsQuest: "12-start"
      text: "I will teach you the covenant..."

    # Rhett-path players get dismissed
    - id: dismiss_rhett_path
      triggers: ["quest", "task", "help", "covenant"]
      questRequired: ["11-start"]
      questFlagRequired:
        "11-branch": "rhett"
      text: "The covenant is not for those who side with the one
        who would destroy the circle."
```

### Gating Root Variants

Root variants (shown on `talk`) also support flag gating:

```yaml
    variants:
      - questRequired: ["11-end"]
        questExcluded: ["12-start"]
        questFlagRequired:
          "11-branch": "sylara"
        text: "The Windwarden offers to teach you the covenant."
        hints: "You could ask about the covenant."
      - questRequired: ["11-end"]
        questExcluded: ["13-start"]
        questFlagRequired:
          "11-branch": "rhett"
        text: "The Windwarden turns away coldly."
```

### Dismissal Nodes (Critical)

**Every NPC that offers a branching quest MUST have a dismissal node
for wrong-path players.** Without it, the NPC's keyword patterns fire
instead, making it look like there's a quest to discover. The player
will keep trying keywords, getting frustrated.

Place dismissal nodes at the TOP of the nodes list (first-match wins):

```yaml
  nodes:
    - id: dismiss_wrong_path
      triggers: ["quest", "task", "help", "work"]
      questRequired: ["11-start"]
      questFlagRequired:
        "11-branch": "sylara"
      text: "I cannot work with someone whose loyalties lie
        elsewhere. The crystal will wait."

    - id: his_cause
      # ... normal quest-granting node
```

### Checklist for Branching Quests

- [ ] Quest YAML declares flags with allowed values
- [ ] Each branch NPC sets the flag via `setsQuestFlag`
- [ ] Followup quest nodes use `questFlagRequired` to gate
- [ ] Dismissal nodes exist for wrong-path players on BOTH NPCs
- [ ] Dismissal nodes are at the TOP of the nodes list
- [ ] Root variants use `questFlagRequired` for path-specific greetings
- [ ] Mid-quest root variants exist (player has started the OTHER
      quest but not finished — `questRequired: ["12-start"]`,
      `questExcluded: ["12-end"]`)
- [ ] Test both paths end-to-end with a fresh character
- [ ] Test visiting the wrong NPC at every quest stage
