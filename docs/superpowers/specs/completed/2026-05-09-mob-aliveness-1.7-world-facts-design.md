# Mob Aliveness 1.7 — World-Model Facts

> **Phase 1 substrate (final).** Standing-fact registry with
> per-NPC awareness tracking. Unifies the existing
> `recentGossipEvents` TempData (event awareness) and the new
> standing-fact awareness into one persistent per-NPC store.
> Worldevents stays as the ring buffer of occurrences;
> `internal/facts/` becomes the "what NPCs know about the world"
> view.

## Goal

NPCs gain awareness of standing facts about the world ("the king
is dead," "the bridge collapsed," "the bandit camp moved east")
distinct from the ephemeral event chatter that the existing
worldevents ring buffer already supports. The chunk also unifies
the per-NPC awareness store: tracking both "I've gossiped about
event X" (replacing the in-memory `recentGossipEvents` TempData)
and "I know fact Y" in a single persistent file per NPC.

The chunk's job is **substrate plus migration** — the storage,
declaration / withdraw / expire API, per-NPC awareness layer,
auto-withdraw on mob respawn, the small additive change to
worldevents (stable event IDs), and the rewrite of
`buildGossipLine` to consume the unified awareness store.

## Architectural musts

The brief lists "fact schema, declaration API, NPC awareness-of-
fact tracking, propagation rules (gossip)" as in-scope. Brain-
storming locked in:

1. **Unified per-NPC awareness store.** One YAML per observer NPC
   template, with two sections — `heard_events` (bounded FIFO of
   recently-gossiped event IDs, replacing the
   `recentGossipEvents` TempData) and `known_facts` (persistent
   awareness records keyed by fact id). Same per-NPC convention
   as chunks 1.4 / 1.6.

2. **Single committed fact registry.** Authored standing facts
   live in one YAML committed to git. Runtime additions / state
   changes (status, claimed-by mob respawn, etc.) layer on top.
   No per-zone fact files v1 — many facts are world-level and
   don't fit a zone shard.

3. **Worldevents stays the ring buffer.** Single additive change:
   add `Id uint64` to `WorldEvent`, set by `EmitWorldEvent` via an
   atomic counter. All existing emitters (combat, suicide,
   mutation, etc.) continue to work; the new field is filled in
   automatically. Awareness records reference event IDs rather
   than composite keys.

4. **Three withdraw signals.** Manual `Withdraw(factId)` (admin
   or quest); time-based via `expiry_round` reached
   (PruneExpired sweep); auto-withdraw via
   `withdraw_on_respawn_of: <mobTemplateId>` field on the fact
   that triggers when that template's instance enters a room.

5. **Lazy filter on read.** Awareness records keep their
   `fact_id` references even when the fact is withdrawn or
   expired; `KnownFactsOf(mobId)` joins against the active
   registry and skips non-active fact ids. Same pattern as
   chunk 1.4's `WitnessedCrimes` joining against 1.3 crimes.

6. **No retroactive feedback loops between substrates.** Continued
   from chunks 1.4 / 1.5 / 1.6. Facts substrate is its own truth
   source; integrates with worldevents (read-only) and with
   `buildGossipLine` (rewrite) but doesn't ripple into
   knowledge / opinion / rep.

We also commit to **migrating the gossip pipeline** as part of v1
— `buildGossipLine` in `internal/hooks/MobIdle_HandleIdleMobs.go`
switches from the `recentGossipEvents` TempData to
`facts.HeardEvent` / `facts.RecordHeardEvent`, and extends the
gossip-candidate pool to also include known facts via
`facts.KnownFactsOf`.

## Architecture & storage

`internal/facts/` package, parallel to chunks 1.1–1.6. Same
patterns: lazy-load + cache + sync-save with mutex; marshal-
under-RLock (the chunk-1.4 review lesson, baked in v1).

**On disk:**
- `_datafiles/world/dogmud/facts.yaml` — committed registry of
  authored standing facts.
- `_datafiles/world/dogmud/facts.awareness/{mobId}-{namesimple}.yaml`
  — gitignored runtime per-NPC awareness state.

**Subject helpers:** none — facts are keyed by string id.

**Reuses from existing code:**
- `worldevents.Significance` enum (Local / Regional / Global) for
  fact significance, ensuring locality math is consistent across
  events and facts.
- The `Source` enum semantically (witnessed / told / deduced /
  unknown) — either imported from chunk 1.4 knowledge or defined
  in parallel; implementer picks whichever avoids cross-package
  coupling.

## Schema

**Fact registry** (`facts.yaml`, committed):
```yaml
facts:
  - id: king-dead
    description: "The king of the Concord is dead."
    significance: global       # Local | Regional | Global
    zone: ""                   # optional locality scope
    region: ""                 # optional region scope
    declared_round: 0          # 0 = engine fills in current round at first load
    expiry_round: 0            # 0 = never expires
    tags: [politics, death]    # free-form, used by gossip templating
    withdraw_on_respawn_of: 0  # 0 = no respawn binding; otherwise mob template id
  - id: bridge-collapsed
    description: "The Sanctum-Basin road bridge has collapsed."
    significance: regional
    region: north_road
    tags: [travel, hazard]
  - id: bandit-camp-moved-east
    description: "The Fernway bandits have moved their camp east."
    significance: local
    region: fernway
    expiry_round: 2000000
    tags: [bandits, intel]
```

**Per-NPC awareness** (`facts.awareness/{mobId}-{namesimple}.yaml`,
gitignored):
```yaml
observer_mob_id: 114
observer_name: old fen
heard_events: [42, 56, 78, 91]   # bounded FIFO
known_facts:
  - fact_id: king-dead
    source: witnessed            # witnessed | told | deduced | unknown
    learned_round: 2065600
  - fact_id: bandit-camp-moved-east
    source: told
    learned_round: 2071200
last_updated_round: 2071200
```

**Per-mob authoring** (mob YAML extension):
```yaml
mobid: 114
name: old fen
# ... existing fields ...
knows_facts:
  - bandit-camp-moved-east
  - bridge-collapsed
```

After `mobs.LoadDataFiles` completes, the facts package walks the
loaded mob templates and seeds per-NPC awareness records for
each declared fact id. Same authoring pattern as chunk 1.6's
`relationships:` field.

**Go types (sketch — implementer may adjust naming):**
```go
type Status string
const (
    StatusActive    Status = "active"
    StatusWithdrawn Status = "withdrawn"
    StatusExpired   Status = "expired"
)

type Fact struct {
    Id                  string                   `yaml:"id"`
    Description         string                   `yaml:"description"`
    Significance        worldevents.Significance `yaml:"significance"`
    Zone                string                   `yaml:"zone,omitempty"`
    Region              string                   `yaml:"region,omitempty"`
    DeclaredRound       uint64                   `yaml:"declared_round"`
    ExpiryRound         uint64                   `yaml:"expiry_round"`
    Tags                []string                 `yaml:"tags,omitempty"`
    WithdrawOnRespawnOf int                      `yaml:"withdraw_on_respawn_of,omitempty"`
    Status              Status                   `yaml:"status"`        // populated runtime; defaults active
}

type Source string
const (
    SourceWitnessed Source = "witnessed"
    SourceTold      Source = "told"
    SourceDeduced   Source = "deduced"
    SourceUnknown   Source = "unknown"
)

type FactKnowledge struct {
    FactId       string `yaml:"fact_id"`
    Source       Source `yaml:"source"`
    LearnedRound uint64 `yaml:"learned_round"`
}

type Awareness struct {
    ObserverMobId    int             `yaml:"observer_mob_id"`
    ObserverName     string          `yaml:"observer_name"`
    HeardEvents      []uint64        `yaml:"heard_events,omitempty"`
    KnownFacts       []FactKnowledge `yaml:"known_facts,omitempty"`
    LastUpdatedRound uint64          `yaml:"last_updated_round"`
}

type Registry struct {
    Facts []*Fact `yaml:"facts"`
}
```

## Configuration knobs

`Balance.FactsHeardEventsMax` (default **32**) — bounded FIFO cap
on the per-NPC `heard_events` list. Mirrors chunk 1.4's
`KnowledgeObservationLogMax`.

No new gold/rep knobs. Facts have no economy footprint.

## Worldevents integration (additive change)

`internal/worldevents/worldevents.go`:
- Add `Id uint64` field to `WorldEvent` struct.
- Add a package-level `var nextEventId atomic.Uint64`.
- In `EmitWorldEvent`, set `evt.Id = nextEventId.Add(1)` before
  appending to the ring buffer.
- All existing emitters (combat, suicide, mutation, idle, etc.)
  remain unchanged — they don't touch the `Id` field; the package
  fills it in.
- `GetRecentWorldEvents` returns events with the new ID.
- Existing `WorldEventFilter` is unchanged.

This is a one-line behavioral change wrapped in a one-field schema
addition. Tests in `worldevents/` get one new assertion (IDs are
monotonically increasing) but no behavioral changes elsewhere.

## Public API

**Fact registry (writes — sync-persist):**
```go
type DeclareOpts struct {
    Description         string
    Significance        worldevents.Significance
    Zone                string
    Region              string
    ExpiryRound         uint64                  // 0 = never
    Tags                []string
    WithdrawOnRespawnOf int                     // 0 = no respawn binding
}

func Declare(factId string, opts DeclareOpts) error
func Withdraw(factId string)
func Expire(factId string)
func PruneExpired() int
func WithdrawAllBoundTo(mobTemplateId int) int  // used by the auto-withdraw hook
```

**Fact registry (reads):**
```go
func GetFact(factId string) *Fact
func AllActiveFacts() []*Fact
func AllFactsByTag(tag string) []*Fact
func AllRows() []*Fact   // every status; admin/debug
```

**Per-NPC awareness (writes — sync-persist):**
```go
func RecordHeardEvent(observerMobId int, eventId uint64)
func RecordKnowsFact(observerMobId int, factId string, source Source)
func ForgetFact(observerMobId int, factId string)
func ForgetAll(observerMobId int)
```

**Per-NPC awareness (reads):**
```go
func HeardEvent(observerMobId int, eventId uint64) bool
func KnowsFact(observerMobId int, factId string) bool

type KnownFact struct {
    Fact         *Fact
    Source       Source
    LearnedRound uint64
}

// Joined view: walks awareness against the active fact registry.
// Lazy-filter-on-read — withdrawn / expired facts skipped. Same
// pattern as chunk 1.4's WitnessedCrimes against 1.3 crimes.
func KnownFactsOf(observerMobId int) []KnownFact

func AllForObserver(observerMobId int) *Awareness  // raw record; admin
```

## buildGossipLine migration

`internal/hooks/MobIdle_HandleIdleMobs.go` `buildGossipLine` is
rewritten to use the facts package's awareness store rather than
the in-memory `recentGossipEvents` TempData.

**Migration steps inside `buildGossipLine`:**

1. Replace
   ```go
   var recentEventKeys []string
   if v := mob.GetTempData("recentGossipEvents"); v != nil {
       recentEventKeys, _ = v.([]string)
   }
   ```
   with a per-event filter:
   ```go
   var candidates []worldevents.WorldEvent
   for _, e := range evts {
       if facts.HeardEvent(int(mob.MobId), e.Id) {
           continue
       }
       candidates = append(candidates, e)
   }
   ```

2. Replace
   ```go
   mob.SetTempData("recentGossipEvents", recentEventKeys)
   ```
   with a record at gossip-emit time:
   ```go
   facts.RecordHeardEvent(int(mob.MobId), pickedEvent.Id)
   ```

3. **New** — pull known facts as additional gossip candidates:
   ```go
   factCandidates := facts.KnownFactsOf(int(mob.MobId))
   ```
   The candidate-selection logic merges both pools. Gossip
   templates get a new key family — `fact-default` for fallback,
   `fact-{tag}` per-tag, `fact-{factId}` per-fact-override —
   resolved with the existing template-resolution chain.

**Templates extension:** update `_datafiles/.../gossip_templates.yaml`
(or wherever gossip templates live; existing file maintained by
the worldevents-consumer):
- Existing per-event-type keys (`MobKilledByPlayer-local`, etc.)
  remain.
- Add a `fact-default: [...]` family at minimum, with one or two
  generic phrasings that take a `{description}` substitution
  (e.g., `"I heard {description}"`).
- Specific fact-id or tag-keyed templates can be added by content
  authors as desired.

**Backwards-compat note:** if no fact-keyed templates are
registered AND no facts are known to the NPC (the v1 case for
most mobs that haven't been authored with `knows_facts:`), the
migration is a no-op — only the underlying dedup mechanism
changed; gossip output is identical.

## Hook for auto-withdraw

`internal/hooks/MobRoomChange_FactsAutoWithdraw.go`:
```go
func MobRoomChangeFactsAutoWithdraw(e events.Event) events.ListenerReturn {
    evt, ok := e.(events.RoomChange)
    if !ok || evt.MobInstanceId == 0 {
        return events.Continue
    }
    mob := mobs.GetInstance(evt.MobInstanceId)
    if mob == nil {
        return events.Continue
    }
    facts.WithdrawAllBoundTo(int(mob.MobId))
    return events.Continue
}
```

Registered alongside the chunk-1.4 `MobRoomChange_KnowledgeObservers`
listener.

## Substrate intersections

| Intersection                            | v1 policy                                                |
|-----------------------------------------|----------------------------------------------------------|
| Facts ↔ worldevents                     | Facts package reads worldevents (Id field) for awareness referencing. Worldevents has no fact dependency. |
| Facts ↔ chunk 1.4 knowledge             | Independent stores. Different shapes (per-subject in 1.4 vs per-fact-id here). May reuse Source enum semantically. |
| Facts ↔ buildGossipLine                 | buildGossipLine migrates to facts. recentGossipEvents TempData removed. |
| Facts ↔ MobDeath                        | No automatic fact emission v1. Authors / quests declare facts on notable deaths via Declare. |
| Facts ↔ mob respawn                     | Auto-withdraw via `withdraw_on_respawn_of` field; new RoomChange hook. |
| Facts ↔ player awareness                | Out of scope v1. Facts are NPC-awareness only. |

The general rule continues from 1.4 / 1.5 / 1.6: **no retroactive
feedback loops between substrates.**

## Gossip surface caveat

The v1 gossip surface that visibly fires fact awareness is the
existing `gossip` group filter in `MobIdle_HandleIdleMobs.go` —
only mobs tagged with the `gossip` group surface awareness in
idle text. As of 2026-05-09, that's the three tavern old men in
Thornwall (Fen 114, Gobb 115, Wrex 116) plus whichever other
mobs content authors tagged. Other NPCs can be taught facts and
will hold the awareness, but won't surface it in idle until they
get the gossip tag OR a future consumer (3.6 idle conversation,
4.5 reactive goals, town-crier feature) reads the awareness.

This is a content concern, not a substrate concern — the facts
package stores awareness for any mob template that gets a
`RecordKnowsFact` call.

## Admin command

`internal/usercommands/admin.fact.go`, admin role only.

```
fact list [--all]                                       — open / all rows
fact show <factId>                                      — full detail
fact declare <factId> [--zone Z] [--region R]
                      [--significance S] [--expiry-rounds N]
                      [--tags a,b,c]
                      [--withdraw-on-respawn-of MOBID]
                      -- <description>
fact withdraw <factId>
fact expire <factId>
fact prune-expired

fact awareness <mobId>                                  — show NPC's heard events + known facts
fact teach <mobId> <factId> [--source S]                — RecordKnowsFact via admin
fact forget <mobId> <factId>                            — drop one fact awareness
fact forget-all <mobId>                                 — drop all awareness for the NPC
```

The `--` separator on `fact declare` lets the description hold
spaces without quoting (everything after `--` is consumed). Tags
are comma-separated.

`<mobId>` accepts numeric template id only (per the unified-
parser-helper memory entry).

`<source>` ∈ `witnessed | told | deduced | unknown`.

**Output formatting** (table form mirrors chunks 1.5 / 1.6).

**Help template:**
`_datafiles/world/dogmud/templates/admincommands/help/command.fact.template`,
mirrors the format used by `command.knowledge` /
`command.bounty` / `command.relationship`.

## Documentation: context.md

Per the aliveness-roadmap maintenance rule introduced in chunk
1.6: every chunk that creates a new package SHIPS a `context.md`.
Chunk 1.7 ships `internal/facts/context.md` covering the package's
overview, file map, key functions, global state, schema, decay
table, integration notes, testing notes. Length target ~250–350
lines, in the voice of chunks 1.4–1.6 backfills.

## Decay

| Field                  | v1 policy                                                         |
|------------------------|-------------------------------------------------------------------|
| Fact status            | Active until withdraw / expire / auto-respawn-withdraw.           |
| Fact `expiry_round`    | 0 = never expires. PruneExpired flips past-expiry to `expired`.   |
| Fact `withdraw_on_respawn_of` | Optional auto-withdraw signal on bound mob template's instance entering a room. |
| Awareness `heard_events` | Bounded FIFO (cap = `FactsHeardEventsMax`).                     |
| Awareness `known_facts` | No bound; lazy-filter on read drops references to non-active facts. |

## Testing

**Unit tests** in `internal/facts/`:
- Persistence round-trip for both registry and awareness files.
- Lazy-load + double-check-lock on both stores.
- Marshal-under-RLock pattern (chunk 1.4 review lesson).
- `Declare` with various opts; collision rejection; default fields.
- `Withdraw` / `Expire` / `PruneExpired` lifecycle transitions; idempotence.
- `WithdrawAllBoundTo` flips matching facts only.
- `RecordHeardEvent` bounded FIFO cap respected; oldest falls off.
- `RecordKnowsFact` idempotent (no duplicate awareness entry on re-record).
- `KnownFactsOf` lazy-filter join: withdrawn / expired skipped.
- `ForgetFact` / `ForgetAll` precise cleanup.

**Hook integration tests** in `internal/hooks/`:
- `MobRoomChange_FactsAutoWithdraw` — fact bound to mob 500
  transitions to withdrawn when mob 500's instance enters a room.
- `buildGossipLine` regression — no behavioral change for event-
  only inputs (existing test fixtures).
- `buildGossipLine` extension — known facts surface as gossip
  candidates given a `fact-default` template.

**Worldevents change test:**
- `EmitWorldEvent` assigns monotonically increasing IDs.
- Existing emitter tests still pass.

**Smoke test goal file**
(`tools/testing/goals/facts-thornwall-smoke.yaml`):

1. Admin runs:
   ```
   fact declare test-fact --significance regional -- The Thornwall mayor has resigned.
   fact teach 114 test-fact --source witnessed
   ```
2. Admin runs `fact awareness 114` — confirms fact on Fen's
   known list.
3. Player travels to The Back Corner (Thornwall City room 484
   — where Fen / Gobb / Wrex hang out) and idles for several
   rounds. Gossip lines should eventually include the test-fact
   (per the `fact-default` template).
4. Admin runs `fact withdraw test-fact`. Re-runs `fact awareness
   114` — fact still on disk but lazy-filter excludes it from
   `KnownFactsOf` (verify via subsequent gossip absence over a
   handful of rounds).
5. Admin runs:
   ```
   fact declare king-dead --withdraw-on-respawn-of 500 -- The king is dead.
   ```
   Then forces a respawn of mob 500 (or waits for natural
   respawn) — fact auto-transitions to withdrawn.

## Performance

- Fact registry: single committed YAML, expected v1 volume in
  the dozens. Read-heavy; in-memory cache lookup is O(1) by id.
- Per-NPC awareness: one file per touched NPC, bounded FIFO on
  `heard_events`, unbounded but lazy-filtered `known_facts`. NPC
  roster size in low hundreds → easily fits memory.
- Auto-withdraw RoomChange hook: filters on `MobInstanceId != 0`
  + cheap lookup against active-fact registry by
  `WithdrawOnRespawnOf` field. Hot path stays cool.
- `buildGossipLine` migration: same O(events × dedup-checks) as
  before; one small additional walk over `KnownFactsOf` per
  gossip emission. Acceptable.

## Out of scope (v1)

| Item                                  | Why deferred                                              |
|---------------------------------------|-----------------------------------------------------------|
| Player awareness of facts             | Substrate is NPC-side v1. Player awareness arrives with future quest content / dialogue surfacing. |
| Auto-emit facts from worldevents      | v1 doesn't auto-create a fact when a notable event fires. Authors / quests declare facts manually. Future hook can listen for event types and emit. |
| Cross-NPC propagation (real gossip transmission) | NPCs hold their own awareness; v1 doesn't model "Fen tells Gobb" transmission. Future chunk wires `RecordKnowsFact` from a propagation hook. |
| Confidence tier on fact awareness     | v1 awareness has source but not confidence. Add when consumers care. |
| Fact subjects (linking facts to entities) | "The king is dead" doesn't formally link to mob 500 except via `withdraw_on_respawn_of`. Future v2: optional `subject:` field referencing a `knowledge.Subject` for richer queries. |
| Per-fact gossip dedup (don't repeat same fact every round) | v1: same fact may surface multiple times in gossip until other candidates exist. Add `last_gossiped_round` per-known-fact when noise warrants. |
| Town crier feature                    | Player-facing news surface. Substrate ready; UX comes with content pass. |
| Player command surface                | None v1. Admin-only. |

## Open questions / deferred decisions

- **Templates organization.** Existing gossip templates live in
  one file (loaded by `loadGossipTemplates` in
  MobIdle_HandleIdleMobs.go). v1 extends that file with
  `fact-default` keys. A future split (fact templates separate
  from event templates) ships when authoring volume justifies.
- **Awareness-record decay.** v1 doesn't decay `known_facts`;
  every learned fact stays in the awareness file until the fact
  itself is withdrawn / expired (which lazy-filter handles). If
  awareness files grow large with stale references, add an
  admin sweep to prune resolved-status references — but YAGNI
  v1.
- **Subject linkage.** Facts that ARE about specific entities
  (the king, the bandit chief) currently only express that link
  via `withdraw_on_respawn_of` for the auto-withdraw mechanic. A
  future enhancement could make the linkage explicit (`subject:
  {type: mob, id: 500}`) so consumers can ask "what facts do I
  know about mob 500?" — defer until 4.5 / 5.x consumers want it.

## Migration

**Required behavioral migration:** `buildGossipLine` in
`internal/hooks/MobIdle_HandleIdleMobs.go` switches from the
in-memory `recentGossipEvents` TempData to
`facts.HeardEvent` / `facts.RecordHeardEvent`. Existing gossip
output is preserved for the event-only case. Existing tests are
expected to pass with the migration; if any rely on the
TempData key directly, they're updated to assert against the
facts substrate instead.

**Mob YAML field:** new optional `knows_facts: [factId, ...]`
field on the `Mob` struct. Existing mobs without the field load
unchanged.

**Worldevents:** new `Id uint64` field auto-filled at emit. No
emitter-side migration required.

**No data migration:** runtime state is gitignored; on first
boot post-deploy, awareness files are created lazily as NPCs
gossip and learn. Existing `recentGossipEvents` TempData is
ephemeral (per-server-run); once the migration ships, no
historical data needs porting.
