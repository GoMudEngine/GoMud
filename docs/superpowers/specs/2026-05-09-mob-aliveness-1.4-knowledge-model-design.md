# Mob Aliveness 1.4 — NPC Knowledge Model

> **Phase 1 substrate.** Per (observer NPC) × (subject) record of what
> the NPC knows about that subject. Subject is polymorphic — a player
> or another NPC. v1 ships identity, location, routine-observation,
> and witnessed-crime fact types. Pattern detection (time-of-day,
> periodicity) and richer fact types (statements heard, items seen,
> aliases) are out of scope for v1.

## Goal

Per-NPC × per-subject store of facts the NPC has learned. Lets the
strategic / tactical / dialogue layers ask "what does this NPC
actually know about this player or that mob?" rather than treating
NPCs as amnesiacs whose only memory is the dialogue mood/visit-count
counter.

The chunk's job is **substrate only** — the storage, write API, read
API, decay rules, integration points with existing substrates (1.1
opinions, 1.3 crimes), and a narrow set of v1 auto-write hooks
(forager/caravan observation + crime witnessing). Consumers (4.x
strategic goals, 5.1 town justice, 5.2 bounty hunting, dialogue
content authors) plug in later.

## Architectural musts

The brief in MOB_ALIVENESS_ROADMAP.md lists "name learned, last-seen
room, deeds witnessed, items seen carried" as in-scope. During
brainstorming we identified five architectural choices that are
cheap to bake in now but expensive to retrofit. v1 commits to all
five.

1. **Polymorphic subject.** Every record keys on a tagged
   `(SubjectType, Id)` pair where `SubjectType ∈ {player, mob}`.
   `mob` always refers to the mob TEMPLATE id, not a specific
   instance — knowledge persists across the subject's deaths and
   respawns. This is what makes NPC↔NPC knowledge work without a
   schema change later.

2. **Source attribution on every record.** Each record has a
   `source` field naming how the knowledge was acquired —
   `witnessed`, `told`, `deduced`, `unknown`. v1 only writes
   `witnessed` (auto-write paths are direct observation and crime
   witnessing); other values are placeholder enum members for
   future gossip / inference write paths.

3. **Confidence tier on every record.** `confidence ∈ {high, med,
   low}`. v1 writes `high` for everything (witnessed knowledge).
   Future paths (gossip = med, deduction = low) write the right
   tier without schema churn.

4. **Per-fact-type decay rules, not a uniform curve.** Identity
   facts (has-met, name) never decay v1. Last-seen has implicit
   staleness — the read API exposes the round, callers compare.
   The observation log self-bounds via FIFO ring of the most
   recent K entries (no time-based decay needed). Crimes-witnessed
   inherits its decay from 1.3's resolved/stale state via lazy
   filter on read.

5. **NPC-on-NPC supported, not just NPC-on-player.** Same schema,
   same write API, same read API. The only difference is the
   subject's `type` field. v1 *does* have one NPC-on-NPC auto-write
   path — the forager/caravan trigger, where observers write
   knowledge of those moving NPCs as their subjects. What v1
   intentionally does NOT have is broader "every NPC observes every
   other NPC they share a room with" auto-write — too much hot-path
   cost for no v1 consumer. Manual writes work fine, and the schema
   is forward-compatible with broader auto-write in 4.x.

We also commit to **forget**: an explicit `Forget` API that drops a
record entirely (or a single fact via `ForgetFact`). Useful for
amnesia-flavored spells, debug, quest cleanup. Cheap to add.

## Architecture & storage

`internal/knowledge/` package, parallel to `internal/opinions/`,
`internal/crimes/`, `internal/factions/`. Reuses the same patterns:
lazy-load on access, in-memory cache, mutex-guarded sync-save on
write.

**On disk:** `_datafiles/world/dogmud/knowledge/{observerMobId}-{namesimple}.yaml`,
gitignored. One file per observer mob TEMPLATE — all instances of a
given template share the same knowledge. This matches the opinions
store; instances are ephemeral, templates carry long-term memory.

We didn't use a per-target file (one per player listing every NPC
who knows them) because that puts us at one-write-per-observation
across many files. Per-NPC keeps writes about one target colocated
in one file, matching the read pattern (NPCs query their own
knowledge, rarely "who-knows-me").

## Schema

```yaml
observer_mob_id: 99
observer_name: records_clerk_pell
records:
  - subject:
      type: player           # "player" | "mob"
      id: 17
    has_met: true
    name_learned: "Bob"
    source: witnessed        # default for all facts on this record
    confidence: high
    last_seen_room: 462
    last_seen_round: 2065719
    observations:            # bounded — last KnowledgeObservationLogMax entries
      - {room: 462, round: 2065600}
      - {room: 463, round: 2065650}
      - {room: 462, round: 2065719}
    crimes_witnessed: [3, 5]
    learned_round: 2065600
    last_updated_round: 2065719
```

Go types (sketch — implementer may adjust naming):

```go
type SubjectType string
const (
    SubjectPlayer SubjectType = "player"
    SubjectMob    SubjectType = "mob"
)

type Subject struct {
    Type SubjectType
    Id   int
}

type Source string
const (
    SourceWitnessed Source = "witnessed"
    SourceTold      Source = "told"
    SourceDeduced   Source = "deduced"
    SourceUnknown   Source = "unknown"
)

type Confidence string
const (
    ConfidenceHigh Confidence = "high"
    ConfidenceMed  Confidence = "med"
    ConfidenceLow  Confidence = "low"
)

type Observation struct {
    Room  int    `yaml:"room"`
    Round uint64 `yaml:"round"`
}

type Record struct {
    Subject          Subject       `yaml:"subject"`
    HasMet           bool          `yaml:"has_met"`
    NameLearned      string        `yaml:"name_learned,omitempty"`
    Source           Source        `yaml:"source"`
    Confidence       Confidence    `yaml:"confidence"`
    LastSeenRoom     int           `yaml:"last_seen_room"`
    LastSeenRound    uint64        `yaml:"last_seen_round"`
    Observations     []Observation `yaml:"observations,omitempty"`
    CrimesWitnessed  []int         `yaml:"crimes_witnessed,omitempty"`
    LearnedRound     uint64        `yaml:"learned_round"`
    LastUpdatedRound uint64        `yaml:"last_updated_round"`
}

type ObserverFile struct {
    ObserverMobId int       `yaml:"observer_mob_id"`
    ObserverName  string    `yaml:"observer_name"`
    Records       []*Record `yaml:"records"`
}
```

All write paths write `Source = SourceWitnessed`, `Confidence =
ConfidenceHigh` in v1.

## Configuration knob

`Balance.KnowledgeObservationLogMax` (default 32) — cap on the
per-record observation log. Older entries fall off FIFO. Sized so
the frequented-rooms top-K query has enough signal to identify a
dominant room without files growing unbounded under long-lived
observers.

## Public API

Subject construction helpers (avoid leaking enum strings into call
sites):

```go
func PlayerSubject(userId int) Subject
func MobSubject(mobId int) Subject
```

**Writes** (all sync-persist):

```go
// Idempotent. Sets HasMet=true, updates LastSeen, fires
// LearnedRound on first call only. Writes Source/Confidence on the
// record (defaults to high if record is fresh).
func RecordMet(observerMobId int, subject Subject, room int, source Source)

// Append to observation log (bounded). Updates LastSeen.
// No-op if (room, round) already exists at the tail (same-round dedup).
func RecordObservation(observerMobId int, subject Subject, room int)

// Set name. If name already known and equal, no-op. If different,
// overwrite (v1 has no aliases or conflict resolution — first write
// wins in practice because writes are idempotent at the source).
func RecordName(observerMobId int, subject Subject, name string, source Source)

// Append crime ID; deduplicated.
func RecordCrimeWitnessed(observerMobId int, subject Subject, crimeId int)

// Forget operations.
func Forget(observerMobId int, subject Subject)
func ForgetFact(observerMobId int, subject Subject, fact string)
// fact ∈ {"name", "observations", "crimes"}; unknown fact is a no-op
```

**Reads:**

```go
func Get(observerMobId int, subject Subject) *Record   // nil if no record
func HasMet(observerMobId int, subject Subject) bool
func NameOf(observerMobId int, subject Subject) (name string, ok bool)
func LastSeen(observerMobId int, subject Subject) (room int, round uint64, ok bool)

// Top-K rooms by observation count from the bounded log.
type RoomCount struct { Room int; Count int }
func FrequentedRooms(observerMobId int, subject Subject, topK int) []RoomCount

// Joined view: walks crimes_witnessed against the 1.3 substrate.
// Caller filters resolved if they want — this is the lazy-filter-
// on-read policy from the design discussion.
type WitnessedCrime struct {
    CrimeId       int
    Kind          crimes.Kind
    ResolvedRound uint64  // 0 if unresolved
}
func WitnessedCrimes(observerMobId int, subject Subject) []WitnessedCrime
```

**Bulk listing** (admin-side, best-effort):

```go
func AllForObserver(observerMobId int) []*Record
func AllObserversOfPlayer(userId int) []int  // walks in-memory cache only
```

## Auto-write hooks

Two trigger sources in v1. Each writes to knowledge from a specific
call site. We intentionally do NOT introduce a generic
"every mob movement triggers a knowledge write" hook — the cost
would dominate hot paths in busy rooms. If 4.x strategic consumers
later want broader observation coverage, generalizing then is
straightforward.

### Trigger 1 — forager/caravan movement

When a forager or caravan enters a room, every other NPC mob already
in that room records an observation of the moving entity:

```
forager/caravan F moves into room R
  for each NPC mob N in R where N.InstanceId != F.InstanceId:
      knowledge.RecordObservation(N.MobId, MobSubject(F.MobId), R)
      knowledge.RecordMet(N.MobId, MobSubject(F.MobId), R, SourceWitnessed)
```

Implementation: add a hook at the forager and caravan packages'
movement seams (call `knowledge.RecordObservation` after each move
completes). The forager/caravan package each gain ~5-10 lines.

The trigger is asymmetric on purpose: foragers/caravans are
*observed by* others, but they don't auto-write knowledge of *what
they see*. Their own knowledge stays empty until a v2 consumer
needs it. Same for non-forager/non-caravan mobs in the room — their
movements don't fire knowledge writes either; they're the recipients
of the trigger, not the source.

### Trigger 2 — crime witnessing (1.3 integration)

When 1.3's three call sites record a crime, each witness gets a
knowledge write for the perpetrator. Three sites:

- `internal/usercommands/attack.go` — `recordAssaultCrime`
- `internal/hooks/MobDeath_FactionRep.go` — murder recording / upgrade
- `internal/usercommands/skill.skullduggery.steal.go` — failed-steal theft

At each site (sketch):

```go
crimeIds := crimes.Record(...)  // returns one ID per faction
for _, witnessInstId := range witnesses {
    if w := mobs.GetInstance(witnessInstId); w != nil {
        for _, crimeId := range crimeIds {
            knowledge.RecordCrimeWitnessed(int(w.MobId),
                knowledge.PlayerSubject(user.UserId), crimeId)
        }
        knowledge.RecordMet(int(w.MobId),
            knowledge.PlayerSubject(user.UserId), room.RoomId, SourceWitnessed)
    }
}
```

The integration point is the call site, not inside `crimes.Record()`,
so the crime substrate's responsibility stays narrow (record a crime
row, period) and we don't add `internal/knowledge` as an import
dependency of the crimes package. Three sites is small and
discoverable; tests will catch any miss.

**Edge cases:**

- Witness mob template id derives from instance via
  `mobs.GetInstance(instanceId).MobId`. If the instance has been
  destroyed by the time we look it up, skip silently — no panic.
- Lone-perp crimes (perp=unknown) skip the knowledge write because
  the subject is not a player. `PlayerSubject(user.UserId)` only
  makes sense when there's a player perpetrator.
- Duplicate writes (same observer, same crime ID) are deduped
  inside `RecordCrimeWitnessed`.

## Decay

| Fact type            | v1 policy                                          |
|----------------------|----------------------------------------------------|
| Identity (has-met)   | Never decays.                                      |
| Identity (name)      | Never decays.                                      |
| Last-seen room/round | No explicit decay; staleness queryable.            |
| Observation log      | Bounded FIFO (cap = `KnowledgeObservationLogMax`). |
| Crimes-witnessed     | Lazy filter on read: walks 1.3 for resolved/stale. |

## Forget

`Forget(observerMobId, subject)` drops the entire record from the
observer's file and persists. `ForgetFact(observerMobId, subject,
fact)` drops one named fact (`"name"`, `"observations"`, `"crimes"`)
and leaves others.

**No cascade in v1.** A `Forget` here does not reset opinion (1.1)
or rep (1.2/1.3) for the same (observer, subject) pair. Each
substrate is its own truth source. Documented decision; revisit when
amnesia-flavored spells land — that's the natural moment to add
cascading forget across substrates.

## Substrate intersections

| Where                                        | v1 policy                                                   |
|----------------------------------------------|-------------------------------------------------------------|
| Crime IDs vs. 1.3 PruneStale / Resolve       | Lazy filter on read (`WitnessedCrimes` joins).              |
| Opinion decay (1.1) vs. knowledge identity   | Independent. Opinions decay; knowledge identity persists.   |
| Faction rep (1.2) vs. knowledge              | No coupling. Rep is faction-scoped; knowledge is per-NPC.   |
| Forget vs. opinion/rep                       | Independent v1; revisit when amnesia spells land.           |

The general v1 rule: **no retroactive feedback loops between
substrates**. Each event hook fires its effects (opinion bump, rep
delta, crime row, knowledge write) at the moment of the event.
Subsequent decay or forget on one substrate does not ripple to the
others. Keeps each substrate independently testable and avoids
ordering races.

## Admin command

`internal/usercommands/admin.knowledge.go`, registered with admin
role permission. Subcommands:

```
knowledge show <mobId>                          — list all records
knowledge show <mobId> <playerName>             — show one player record
knowledge show <mobId> mob <targetId>           — show one NPC record
knowledge frequented <mobId> <playerName> [topK]
knowledge forget <mobId> <playerName>           — drop record
knowledge forget <mobId> <playerName> <fact>    — drop one fact
```

`<mobId>` accepts numeric template id OR a name (using
`mobs.GetMobIdByName` or equivalent). `<playerName>` goes through
`users.GetByCharacterNameOrLoad` so admins can inspect knowledge of
offline players.

Help template at `_datafiles/templates/admincommands/help/command.knowledge`.

## Testing

**Unit tests** in `internal/knowledge/` (analogous to
`crimes_test.go` / `opinions_test.go`):

- Persistence round-trip (write, evict, reload).
- Polymorphic subject keying — same template id used by both player
  and mob subjects shouldn't collide.
- Observation log bounding — write `cap+5` entries, confirm only
  the most recent `cap` survive in FIFO order.
- Same-round dedup — duplicate `RecordObservation(room=R, round=T)`
  doesn't append a duplicate.
- Frequented-rooms top-K math — including ties and `topK >
  len(unique rooms)`.
- Forget / ForgetFact — precise, doesn't touch other records.
- Lazy-filter join with crimes substrate — record knowledge of two
  crimes, resolve one, confirm `WitnessedCrimes` returns both with
  correct `ResolvedRound`.
- Concurrent-write race on lazy-load — same double-check-lock test
  pattern chunk 1.3's T7 caught a real bug with.
- Idempotence of `RecordMet`, `RecordCrimeWitnessed` (no duplicate
  appends).

**Hook integration tests** in `internal/hooks/` (or alongside
forager/caravan packages):

- Forager moves into a populated room → all other NPC mobs in the
  room get a knowledge observation row written.
- Caravan move → same.
- 1.3 crime record at each of the three call sites → witness mobs
  get crime ID added to their knowledge of the perp player.
- Lone-perp crime (perp=unknown) → no knowledge write fires
  (correctly skipped because no player subject).
- Duplicate crime witnessing is idempotent.

**Smoke test goal file** —
`tools/testing/goals/knowledge-thornwall-smoke.yaml` (gitignored,
local-only):

- Player walks the city while foragers / caravans pass through.
  Admin runs `knowledge show <citizen> <player>` and `knowledge
  frequented <citizen> <forager>` to confirm last-seen and
  frequented-rooms tracking.
- Player commits a witnessed assault. Admin verifies the witness's
  `crimes_witnessed` for that player includes the new crime ID.
- Player commits a lone (perp=unknown) murder. Admin confirms no
  knowledge write fired for the perp side.
- Admin runs `knowledge forget <citizen> <player>` and verifies the
  record is gone, AND that opinion / rep are unchanged (no
  cascade).

## Performance

- One file per observer NPC TEMPLATE (not per-instance). Mob roster
  size is in the low hundreds; every-write sync-save fits the budget.
- Bounded observation log caps per-record growth.
- Auto-write triggers are narrow (foragers / caravans + crime
  sites), not "every mob move." Hot path stays cool.
- Lazy-load on first read; no startup mass-load. The
  `AllObserversOfPlayer` admin query walks only the in-memory cache
  (touched files) — explicitly best-effort. A "warm everything"
  admin sweep is a future affordance if the gap matters.

## Out of scope (v1)

| Item                            | Why deferred                                              |
|---------------------------------|-----------------------------------------------------------|
| Aliases / multi-name model      | Single string is fine until consumers care.               |
| Statements heard / dialogue log | Existing dialogue mood/visit-count covers v1 needs.       |
| Items seen carried              | No v1 consumer pull.                                      |
| Schedule pattern detection      | Time-of-day binning needs 3.1 game-time; defer to 3.x.    |
| Periodicity inference           | Algorithmic; better with real observation data.           |
| Disguise / mistaken identity    | Defer to 2.7 mob skullduggery.                            |
| Higher-order meta-knowledge     | Exponential complexity; skip.                             |
| Self-knowledge                  | Separate concern (inventory awareness lives elsewhere).   |
| Per-fact source override        | Record-level source is enough until gossip writes land.   |
| Generic deed log (non-crime)    | Crime IDs cover the only known v1 consumer (5.1).         |

## Open questions / deferred decisions

- **Cascade on forget.** When an amnesia-flavored spell or quest-
  reset lands, should `knowledge.Forget` cascade to opinion reset?
  Likely yes; revisit when the consumer materializes.
- **Broader NPC-on-NPC auto-write.** v1's only NPC-on-NPC auto-
  write is the forager/caravan trigger. Generic "guards observe
  everyone in their patrol" is a 4.x concern; generalize the
  forager/caravan trigger when strategic consumers need it.
- **Per-fact source granularity.** v1 record-level source is fine.
  When gossip / deduction write paths land, evaluate whether to
  promote to per-fact.
- **Best-effort `AllObserversOfPlayer`.** If admin tooling needs a
  comprehensive offline answer, add a "warm everything" sweep. v1
  in-memory-only is enough.

## Migration

None. New package, new files. Existing systems are untouched except
for the three crime call sites (5-line additions each) and the
forager/caravan movement seams.
