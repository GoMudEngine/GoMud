# Ferry Stage 1 — Vessels + Schedule + Paid Passenger Travel — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Three real, boardable ferry vessels running clock-derived schedules on the reserved water routes (Stillwater↔NP, Stillwater↔Confluence, Confluence↔NP), with agent-paid boarding and free gangplank disembark.

**Architecture:** New `internal/ferry` package. Vessel state is a pure function of the game round (no persistence, restart-safe). A `NewRound` hook reconciles the world to the computed state each round (gangplank temp exits, transition emotes, at-sea ambiance). Boarding is a new behavior-tree action (`board_ferry`) on dock-agent NPCs, modeled on `open_instance_portal` (parse ask text → charge gold → `MoveToRoom`).

**Tech Stack:** Go (GoMud engine), YAML data files, `fileloader` generic loader, `events` listener bus, behavior trees, gametime clock.

**Spec:** `docs/superpowers/specs/2026-07-03-ferry-system-design.md` (Stage 1 scope only — no trade factors, no warehouses).

---

## Verified engine facts (do not re-derive)

- `exit.TemporaryRoomExit{RoomId, Title, UserId, Expires}`; `room.AddTemporaryExit(name string, t exit.TemporaryRoomExit) bool`; `room.RemoveTemporaryExit(t exit.TemporaryRoomExit) bool` (matches on UserId+Title+RoomId); `room.ExitsTemp map[string]exit.TemporaryRoomExit` is exported.
- `rooms.MoveToRoom(userId int, toRoomId int, isSpawn ...bool) error`.
- `events.RegisterListener(events.NewRound{}, Handler)` — registration lives in `internal/hooks/hooks.go` (~line 44, after `IdleMobs`).
- Behavior-tree actions register in `internal/behaviortree/actions.go` (`actionRegistry["open_instance_portal"] = actOpenInstancePortal` at line 57). `actOpenInstancePortal` in `actions_mob.go:335` is the model: `ctx.Event.UserId`, `ctx.Event.Text` (text AFTER "ask <mob> "), `ctx.RoomId`, `mob.Command("say ...")`, gold via `user.Character.Gold`.
- `fileloader.LoadAllFlatFiles[K, T]` requires `Id() K`, `Filepath() string`, `Validate() error`; filesystem path must end in `Filepath()`.
- `gametime.GetDate(forceRound ...uint64) GameDate` → `.Hour` (12h int), `.AmPm`, `.Hour24`. `configs.GetTimingConfig().RoundsPerDay` = **900** in prod config; `RoundSeconds` = 4. So 1 game-hour = 37.5 rounds = 2.5 real minutes.
- Timing chosen for Stage 1 (revises the spec's placeholder 6h/2h — spec explicitly defers exact values to plan): **crossing 2 game-hours (75 rounds ≈ 5 real min), layover 1 game-hour (37 rounds ≈ 2.5 real min), cycle = 224 rounds ≈ 15 real min.**
- Behavior YAMLs auto-attach by filename `behaviors/<zone>/<mobid>-<convertforfilename-name>.yaml` (no field in the mob YAML; see Threshold-Keeper 9553).
- Free IDs (verified via `id_inventory.py` 2026-07-03): rooms global next free **6423**, mobs global next free **9571**.
- Zone-config format (`rooms/river_road/zone-config.yaml`): `name`, `roomid`, `defaultbiome`, `region`.
- `GamePlay` config struct is `internal/configs/config.gameplay.go` (uses `ConfigBool`/`ConfigString` types).

## ID assignments (this plan owns these — do not reallocate)

| ID | What |
|----|------|
| rooms 6423 | Deck of the *Lakewind Packet* (Stillwater↔NP) |
| rooms 6424 | Deck of the *Grey Heron* (Stillwater↔Confluence) |
| rooms 6425 | Deck of the *Broadbeam* (Confluence↔NP) |
| mobs 9571 | The Ferry Agent (Stillwater, dock 4118) |
| mobs 9572 | The Quay Agent (NP Docks, dock 5502) |
| mobs 9573 | The Barge Master (Confluence, dock 6109) |
| mobs 9574 | A Weathered Deckhand (packet deck 6423) |
| mobs 9575 | A Barge Poleman (heron deck 6424) |
| mobs 9576 | A River Bargeman (broadbeam deck 6425) |

Every port serves **two** routes, so agents take a destination word:
`ask agent passage plymouth` / `ask master passage stillwater`, resolved
via a per-agent `routes:` param map (same shape as `open_instance_portal`'s
`zones:` map).

## File structure

```
internal/ferry/route.go         Route struct, YAML loader, registry, world validation
internal/ferry/route_test.go
internal/ferry/state.go         Pure clock→state math (StateAt, NextDockedRound)
internal/ferry/state_test.go
internal/ferry/controller.go    Tick(): transitions, gangplank reconcile, ambiance
internal/ferry/board.go         Board(): fare charge + MoveToRoom + quotes
internal/ferry/board_test.go    Quote formatting tests (pure parts only)
internal/hooks/NewRound_FerryTick.go
internal/hooks/hooks.go                     (modify: register listener)
internal/behaviortree/actions_ferry.go     actBoardFerry
internal/behaviortree/actions_ferry_test.go
internal/behaviortree/actions.go            (modify: register action)
internal/configs/config.gameplay.go         (modify: FerriesEnabled)
_datafiles/config.yaml                      (modify: FerriesEnabled: true)
main.go                                     (modify: ferry.LoadDataFiles())
_datafiles/world/dogmud/ferries/*.yaml      3 route files
_datafiles/world/dogmud/rooms/ferries/      zone-config + 3 deck rooms
_datafiles/world/dogmud/mobs/...            3 agents + 3 deckhands
_datafiles/world/dogmud/behaviors/...       3 agent behavior trees
_datafiles/world/dogmud/rooms/{stillwater/4118,new_plymouth_docks/5502,the_confluence/6109}.yaml  (modify: spawninfo + berth prose)
_datafiles/world/dogmud/dialogue/hartcharn/9182.yaml  (modify: Severin Pell points at the live service)
tools/playtest/goals/ferry-stage1.yaml      harness goals
```

---

### Task 1: Config knob — `GamePlay.FerriesEnabled`

**Files:**
- Modify: `internal/configs/config.gameplay.go`
- Modify: `_datafiles/config.yaml`

- [ ] **Step 1: Add the field.** In `internal/configs/config.gameplay.go`, add to the `GamePlay` struct, directly under the `MapConsistencyEnforce` field (line ~22):

```go
	FerriesEnabled ConfigBool `yaml:"FerriesEnabled"` // Master toggle for the ferry vessel controller + boarding (Stage 1 ferry system)
```

No `Validate()` handling needed — `ConfigBool` zero value is `false`, which is the safe default (ferries off unless configured on).

- [ ] **Step 2: Enable in the world config.** In `_datafiles/config.yaml`, find the `GamePlay:` section (search for `MapConsistencyEnforce`) and add beneath it, matching the file's comment style:

```yaml
  # - FerriesEnabled -
  # Master toggle for the scheduled ferry vessels (Stage 1 ferry system).
  FerriesEnabled: true
```

- [ ] **Step 3: Build.**

Run: `go build ./...`
Expected: clean compile.

- [ ] **Step 4: Commit.**

```bash
git add internal/configs/config.gameplay.go _datafiles/config.yaml
git commit -m "feat(ferry): GamePlay.FerriesEnabled config knob"
```

---

### Task 2: Route type + loader + registry

**Files:**
- Create: `internal/ferry/route.go`
- Test: `internal/ferry/route_test.go`

- [ ] **Step 1: Write the failing tests.**

```go
package ferry

import "testing"

func validRoute() Route {
	return Route{
		RouteId:       "test_route",
		Name:          "The Test Packet",
		DeckRoom:      6423,
		Ports:         []Port{{DockRoom: 4118}, {DockRoom: 5502}},
		CrossingHours: 2,
		LayoverHours:  1,
		Fare:          75,
	}
}

func TestRouteValidate_Valid(t *testing.T) {
	if err := validRoute().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestRouteValidate_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Route)
	}{
		{"missing routeid", func(r *Route) { r.RouteId = "" }},
		{"missing name", func(r *Route) { r.Name = "" }},
		{"no deck room", func(r *Route) { r.DeckRoom = 0 }},
		{"one port", func(r *Route) { r.Ports = r.Ports[:1] }},
		{"three ports", func(r *Route) { r.Ports = append(r.Ports, Port{DockRoom: 9}) }},
		{"same dock twice", func(r *Route) { r.Ports[1].DockRoom = r.Ports[0].DockRoom }},
		{"zero crossing", func(r *Route) { r.CrossingHours = 0 }},
		{"zero layover", func(r *Route) { r.LayoverHours = 0 }},
		{"negative fare", func(r *Route) { r.Fare = -1 }},
	}
	for _, tc := range cases {
		r := validRoute()
		tc.mutate(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
	}
}

func TestRouteFilepath(t *testing.T) {
	if got := validRoute().Filepath(); got != "test_route.yaml" {
		t.Fatalf("Filepath() = %q", got)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/ferry/ -run TestRoute -v`
Expected: FAIL (package doesn't exist yet / undefined: Route).

- [ ] **Step 3: Implement `route.go`.**

```go
// Package ferry implements the scheduled ferry vessels: clock-derived
// vessel state, gangplank reconciliation, and agent-paid boarding.
// Stage 1 of docs/superpowers/specs/2026-07-03-ferry-system-design.md.
package ferry

import (
	"fmt"
	"time"

	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Port is one end of a route.
type Port struct {
	DockRoom int `yaml:"dock_room"`
}

// Route is one ferry line: a vessel (deck room) shuttling between two ports.
type Route struct {
	RouteId           string `yaml:"routeid"`
	Name              string `yaml:"name"` // display name, e.g. "the Lakewind Packet"
	DeckRoom          int    `yaml:"deck_room"`
	Ports             []Port `yaml:"ports"` // exactly two
	CrossingHours     int    `yaml:"crossing_hours"`
	LayoverHours      int    `yaml:"layover_hours"`
	PhaseOffsetRounds int    `yaml:"phase_offset_rounds"`
	Fare              int    `yaml:"fare"`
}

func (r Route) Id() string       { return r.RouteId }
func (r Route) Filepath() string { return r.RouteId + `.yaml` }

// Validate covers intrinsic checks; world checks (rooms exist) happen in
// LoadDataFiles once room data is loaded.
func (r Route) Validate() error {
	if r.RouteId == `` {
		return fmt.Errorf(`ferry route missing routeid`)
	}
	if r.Name == `` {
		return fmt.Errorf(`ferry route %s missing name`, r.RouteId)
	}
	if r.DeckRoom <= 0 {
		return fmt.Errorf(`ferry route %s missing deck_room`, r.RouteId)
	}
	if len(r.Ports) != 2 {
		return fmt.Errorf(`ferry route %s must have exactly 2 ports, has %d`, r.RouteId, len(r.Ports))
	}
	if r.Ports[0].DockRoom == r.Ports[1].DockRoom {
		return fmt.Errorf(`ferry route %s has the same dock_room at both ports`, r.RouteId)
	}
	for i, p := range r.Ports {
		if p.DockRoom <= 0 {
			return fmt.Errorf(`ferry route %s port %d missing dock_room`, r.RouteId, i)
		}
	}
	if r.CrossingHours <= 0 || r.LayoverHours <= 0 {
		return fmt.Errorf(`ferry route %s needs positive crossing_hours and layover_hours`, r.RouteId)
	}
	if r.Fare < 0 {
		return fmt.Errorf(`ferry route %s has negative fare`, r.RouteId)
	}
	return nil
}

var routes = map[string]Route{}

// RouteFor returns a registered route by id.
func RouteFor(routeId string) (Route, bool) {
	r, ok := routes[routeId]
	return r, ok
}

// AllRoutes returns all registered routes.
func AllRoutes() []Route {
	out := make([]Route, 0, len(routes))
	for _, r := range routes {
		out = append(out, r)
	}
	return out
}

// LoadDataFiles loads + validates every route. Panics on world-integrity
// failures (missing rooms, shared deck rooms) — same startup rigor as the
// schedule/patrol validators. Must be called AFTER rooms.LoadDataFiles().
func LoadDataFiles() {
	start := time.Now()

	loaded, err := fileloader.LoadAllFlatFiles[string, Route](`_datafiles/world/dogmud/ferries`)
	if err != nil {
		panic(fmt.Sprintf(`ferry.LoadDataFiles: %v`, err))
	}

	decksSeen := map[int]string{}
	for id, r := range loaded {
		if rooms.LoadRoom(r.DeckRoom) == nil {
			panic(fmt.Sprintf(`ferry route %s: deck_room %d does not exist`, id, r.DeckRoom))
		}
		for i, p := range r.Ports {
			if rooms.LoadRoom(p.DockRoom) == nil {
				panic(fmt.Sprintf(`ferry route %s: port %d dock_room %d does not exist`, id, i, p.DockRoom))
			}
		}
		if other, dup := decksSeen[r.DeckRoom]; dup {
			panic(fmt.Sprintf(`ferry routes %s and %s share deck_room %d`, id, other, r.DeckRoom))
		}
		decksSeen[r.DeckRoom] = id
	}

	routes = loaded
	mudlog.Info(`ferry.LoadDataFiles()`, `loadedCount`, len(routes), `Time Taken`, time.Since(start))
}
```

NOTE for executor: check the exact data-path idiom used at the top of
`internal/mutators/mutators.go` `LoadDataFiles()` (line ~325). If it builds
the path from `configs.GetConfig().FilePaths` rather than a literal
`_datafiles/...` string, use the same idiom here.

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/ferry/ -run TestRoute -v`
Expected: PASS (3 tests).

- [ ] **Step 5: Commit.**

```bash
git add internal/ferry/route.go internal/ferry/route_test.go
git commit -m "feat(ferry): Route type, YAML loader, startup world validation"
```

---

### Task 3: Clock→state math

**Files:**
- Create: `internal/ferry/state.go`
- Test: `internal/ferry/state_test.go`

The cycle: `DOCKED port0 (layover) → AT SEA →port1 (crossing) → DOCKED port1 (layover) → AT SEA →port0 (crossing)`, repeating. All pure functions taking `roundsPerDay` as a parameter so tests never touch global config.

- [ ] **Step 1: Write the failing tests.**

With `roundsPerDay=900`, crossing 2h → 75 rounds, layover 1h → 37 rounds, cycle = 224.

```go
package ferry

import "testing"

func testRoute() Route {
	r := validRoute() // from route_test.go
	return r
}

func TestStateAt_CycleBoundaries(t *testing.T) {
	r := testRoute() // crossing 2h=75r, layover 1h=37r, cycle=224r
	cases := []struct {
		round    uint64
		docked   bool
		portIdx  int
		roundsUntil int
	}{
		{0, true, 0, 37},    // start of layover at port 0
		{36, true, 0, 1},    // last docked round at port 0
		{37, false, 1, 75},  // first at-sea round, heading to port 1
		{111, false, 1, 1},  // last at-sea round
		{112, true, 1, 37},  // docked port 1
		{148, true, 1, 1},   // last docked at port 1
		{149, false, 0, 75}, // heading back to port 0
		{223, false, 0, 1},  // last round of the cycle
		{224, true, 0, 37},  // wraps: docked port 0 again
	}
	for _, tc := range cases {
		s := StateAt(r, tc.round, 900)
		if s.Docked != tc.docked || s.PortIdx != tc.portIdx || s.RoundsUntilTransition != tc.roundsUntil {
			t.Errorf("round %d: got %+v, want docked=%v port=%d until=%d",
				tc.round, s, tc.docked, tc.portIdx, tc.roundsUntil)
		}
	}
}

func TestStateAt_PhaseOffset(t *testing.T) {
	r := testRoute()
	r.PhaseOffsetRounds = 37
	s := StateAt(r, 0, 900)
	// offset 37 puts round 0 at the first at-sea round
	if s.Docked || s.PortIdx != 1 {
		t.Fatalf("with offset 37, round 0 should be at sea toward port 1, got %+v", s)
	}
}

func TestNextDockedRound(t *testing.T) {
	r := testRoute()
	// At round 0 the vessel is already docked at port 0.
	if got := NextDockedRound(r, 0, 0, 900); got != 0 {
		t.Errorf("port 0 from round 0: got %d, want 0", got)
	}
	// Next docking at port 1 begins at round 112.
	if got := NextDockedRound(r, 1, 0, 900); got != 112 {
		t.Errorf("port 1 from round 0: got %d, want 112", got)
	}
	// From round 113 (already docked at 1) it's immediate.
	if got := NextDockedRound(r, 1, 113, 900); got != 113 {
		t.Errorf("port 1 from round 113: got %d, want 113", got)
	}
	// From round 150 (at sea toward 0), port 1 next docks at 224+112=336.
	if got := NextDockedRound(r, 1, 150, 900); got != 336 {
		t.Errorf("port 1 from round 150: got %d, want 336", got)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/ferry/ -run 'TestStateAt|TestNextDockedRound' -v`
Expected: FAIL — undefined: StateAt.

- [ ] **Step 3: Implement `state.go`.**

```go
package ferry

// VesselState is where a vessel is at a given round. When Docked,
// PortIdx is the berth. When at sea, PortIdx is the DESTINATION port.
type VesselState struct {
	Docked                bool
	PortIdx               int
	RoundsUntilTransition int
}

// hoursToRounds converts game-hours to rounds (integer truncation is fine
// — schedule precision of ±1 round is invisible to players).
func hoursToRounds(hours, roundsPerDay int) int {
	return hours * roundsPerDay / 24
}

// StateAt computes vessel state as a pure function of the round. This is
// the restart-safety property: no state is stored anywhere.
func StateAt(r Route, round uint64, roundsPerDay int) VesselState {
	lay := hoursToRounds(r.LayoverHours, roundsPerDay)
	cross := hoursToRounds(r.CrossingHours, roundsPerDay)
	cycle := 2 * (lay + cross)

	p := int((round + uint64(r.PhaseOffsetRounds)) % uint64(cycle))

	switch {
	case p < lay: // docked at port 0
		return VesselState{Docked: true, PortIdx: 0, RoundsUntilTransition: lay - p}
	case p < lay+cross: // at sea toward port 1
		return VesselState{Docked: false, PortIdx: 1, RoundsUntilTransition: lay + cross - p}
	case p < lay+cross+lay: // docked at port 1
		return VesselState{Docked: true, PortIdx: 1, RoundsUntilTransition: lay + cross + lay - p}
	default: // at sea toward port 0
		return VesselState{Docked: false, PortIdx: 0, RoundsUntilTransition: cycle - p}
	}
}

// NextDockedRound returns the earliest round >= fromRound at which the
// vessel is docked at portIdx. Linear scan bounded by one cycle (<= a few
// hundred iterations) — called only on player asks, simplicity wins.
func NextDockedRound(r Route, portIdx int, fromRound uint64, roundsPerDay int) uint64 {
	cycle := uint64(2 * (hoursToRounds(r.LayoverHours, roundsPerDay) + hoursToRounds(r.CrossingHours, roundsPerDay)))
	for i := uint64(0); i <= cycle; i++ {
		s := StateAt(r, fromRound+i, roundsPerDay)
		if s.Docked && s.PortIdx == portIdx {
			return fromRound + i
		}
	}
	return fromRound // unreachable: a 2-port cycle always docks at both
}
```

- [ ] **Step 4: Run tests.**

Run: `go test ./internal/ferry/ -v`
Expected: PASS (all route + state tests).

- [ ] **Step 5: Commit.**

```bash
git add internal/ferry/state.go internal/ferry/state_test.go
git commit -m "feat(ferry): pure clock-derived vessel state math"
```

---

### Task 4: Controller — transitions, gangplank reconcile, ambiance

**Files:**
- Create: `internal/ferry/controller.go`

Room interactions make this thin layer integration-tested (boot + harness), not unit-tested; all decisions it makes come from the Task 3 pure math.

- [ ] **Step 1: Implement `controller.go`.**

```go
package ferry

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const gangplankExitName = `gangplank`

// ambiance lines shown on deck while at sea, rotated deterministically.
var atSeaAmbiance = []string{
	`Water slides past the hull in a long, unhurried hiss.`,
	`The deck lifts and settles. Somewhere below, cargo shifts and goes quiet again.`,
	`A gull keeps pace off the rail for a while, decides you have no food, and banks away.`,
	`Spray flicks over the bow. The shoreline is a low smudge, too far to name.`,
}

// ambianceEveryNRounds paces the at-sea flavor (~every 92s at 4s rounds).
const ambianceEveryNRounds = 23

// Tick reconciles every vessel to its clock-derived state. Called once per
// round from the NewRound hook. Idempotent: transition emotes fire only on
// the round the phase changes; the gangplank ensure/remove runs every round.
func Tick() {
	if !bool(configs.GetGamePlayConfig().FerriesEnabled) {
		return
	}
	rpd := int(configs.GetTimingConfig().RoundsPerDay)
	now := util.GetRoundCount()

	for _, r := range AllRoutes() {
		cur := StateAt(r, now, rpd)
		if now > 0 {
			prev := StateAt(r, now-1, rpd)
			if prev.Docked && !cur.Docked {
				emitDeparture(r, prev.PortIdx)
			}
			if !prev.Docked && cur.Docked {
				emitArrival(r, cur.PortIdx)
			}
		}
		reconcileGangplank(r, cur)
		if !cur.Docked && now%ambianceEveryNRounds == 0 {
			emitAmbiance(r, now)
		}
	}
}

// reconcileGangplank ensures the deck's gangplank temp exit matches the
// vessel state: present and pointing at the berth while docked, absent at
// sea. Self-heals if anything else pruned or altered it.
func reconcileGangplank(r Route, s VesselState) {
	deck := rooms.LoadRoom(r.DeckRoom)
	if deck == nil {
		return
	}
	if !s.Docked {
		if t, ok := deck.ExitsTemp[gangplankExitName]; ok {
			deck.RemoveTemporaryExit(t)
		}
		return
	}
	wantDock := r.Ports[s.PortIdx].DockRoom
	if t, ok := deck.ExitsTemp[gangplankExitName]; ok {
		if t.RoomId == wantDock {
			return // already correct
		}
		deck.RemoveTemporaryExit(t)
	}
	deck.AddTemporaryExit(gangplankExitName, exit.TemporaryRoomExit{
		RoomId:  wantDock,
		Title:   gangplankExitName,
		UserId:  0,          // anyone may disembark
		Expires: `1 day`,    // effectively never; controller removes it explicitly
	})
}

func emitDeparture(r Route, fromPortIdx int) {
	if dock := rooms.LoadRoom(r.Ports[fromPortIdx].DockRoom); dock != nil {
		dock.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`Lines come off the bollards and %s stands out into open water.`, r.Name))
	}
	if deck := rooms.LoadRoom(r.DeckRoom); deck != nil {
		deck.SendText(messaging.CategoryRoomDescription,
			`The gangplank comes up, the lines go over, and the deck heels as she comes about. The shore begins to slide away.`)
	}
}

func emitArrival(r Route, atPortIdx int) {
	if dock := rooms.LoadRoom(r.Ports[atPortIdx].DockRoom); dock != nil {
		dock.SendText(messaging.CategoryRoomDescription,
			fmt.Sprintf(`%s noses up to the berth and makes fast. The gangplank rattles down.`, r.Name))
	}
	if deck := rooms.LoadRoom(r.DeckRoom); deck != nil {
		deck.SendText(messaging.CategoryRoomDescription,
			`Fenders squeal against timber as she comes alongside. The gangplank goes down — you can walk ashore.`)
	}
}

func emitAmbiance(r Route, now uint64) {
	deck := rooms.LoadRoom(r.DeckRoom)
	if deck == nil {
		return
	}
	line := atSeaAmbiance[int(now/ambianceEveryNRounds)%len(atSeaAmbiance)]
	deck.SendText(messaging.CategoryRoomDescription, line)
}
```

NOTE for executor: `r.Name` values in the route YAMLs are written lowercase-
article style ("the Lakewind Packet") so they read correctly mid-sentence.
Sentence-initial uses in emitArrival/emitDeparture: if the rendered text
reads awkwardly in-game ("the Lakewind Packet noses up..."), capitalize via
a small helper — check in the smoke test.

- [ ] **Step 2: Build.**

Run: `go build ./...`
Expected: clean compile. (If `configs.GetGamePlayConfig().FerriesEnabled` doesn't compile because the accessor is named differently, find the accessor used for `MapConsistencyEnforce` callers and match it.)

- [ ] **Step 3: Commit.**

```bash
git add internal/ferry/controller.go
git commit -m "feat(ferry): per-round controller — transitions, gangplank reconcile, ambiance"
```

---

### Task 5: NewRound hook + startup load

**Files:**
- Create: `internal/hooks/NewRound_FerryTick.go`
- Modify: `internal/hooks/hooks.go` (~line 44)
- Modify: `main.go` (after `mobs.LoadDataFiles()` block, near line 1420 where `pets.LoadDataFiles()` / `quests.LoadDataFiles()` live)

- [ ] **Step 1: Create the hook.**

```go
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/ferry"
)

// FerryTick reconciles every ferry vessel to its clock-derived state.
// The FerriesEnabled gate lives inside ferry.Tick().
func FerryTick(e events.Event) events.ListenerReturn {
	ferry.Tick()
	return events.Continue
}
```

- [ ] **Step 2: Register it.** In `internal/hooks/hooks.go`, after the `IdleMobs` registration line:

```go
	events.RegisterListener(events.NewRound{}, FerryTick) // Ferry vessels: schedule reconcile
```

- [ ] **Step 3: Load routes at startup.** In `main.go`, after `quests.LoadDataFiles()` (line ~1421 — rooms and mobs are both loaded by then), add:

```go
	ferry.LoadDataFiles()
```

…and add `"github.com/GoMudEngine/GoMud/internal/ferry"` to main.go's imports.

- [ ] **Step 4: Build + full test suite.**

Run: `go build ./... && go test ./internal/ferry/ ./internal/hooks/ -count=1`
Expected: clean compile, tests pass. (No route YAMLs exist yet — `LoadAllFlatFiles` over an absent/empty dir: if the walk errors on a missing directory, create the empty `_datafiles/world/dogmud/ferries/` dir with the first route file in Task 8 before booting. Do NOT boot-test until Task 10.)

- [ ] **Step 5: Commit.**

```bash
git add internal/hooks/NewRound_FerryTick.go internal/hooks/hooks.go main.go
git commit -m "feat(ferry): NewRound tick hook + startup route loading"
```

---

### Task 6: Boarding — `ferry.Board` + quotes

**Files:**
- Create: `internal/ferry/board.go`
- Test: `internal/ferry/board_test.go`

- [ ] **Step 1: Write the failing test (pure quote formatting).**

```go
package ferry

import (
	"strings"
	"testing"
)

func TestNotDockedQuoteFormat(t *testing.T) {
	got := formatNotDockedQuote("the Test Packet", 4, "PM")
	if !strings.Contains(got, "the Test Packet") || !strings.Contains(got, "4 PM") {
		t.Fatalf("quote missing vessel name or time: %q", got)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/ferry/ -run TestNotDockedQuote -v`
Expected: FAIL — undefined: formatNotDockedQuote.

- [ ] **Step 3: Implement `board.go`.**

```go
package ferry

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// BoardResult reports what Board did (for tests/telemetry; the player
// messaging is handled inside).
type BoardResult int

const (
	BoardOk BoardResult = iota
	BoardNotAtPort
	BoardNotDocked
	BoardNoGold
	BoardError
)

// Board runs the full paid-boarding flow: the asking user, the agent mob
// (speaks refusals/quotes), the room the ask happened in (must be one of
// the route's docks), and the resolved route. Mirrors the Threshold-Keeper
// gold-charge pattern (actOpenInstancePortal).
func Board(user *users.UserRecord, mob *mobs.Mob, roomId int, routeId string) BoardResult {
	r, ok := RouteFor(routeId)
	if !ok {
		return BoardNotAtPort
	}
	portIdx := -1
	for i, p := range r.Ports {
		if p.DockRoom == roomId {
			portIdx = i
		}
	}
	if portIdx < 0 {
		return BoardNotAtPort
	}

	if !bool(configs.GetGamePlayConfig().FerriesEnabled) {
		mob.Command(`say No sailings today. The line is suspended.`)
		return BoardNotDocked
	}

	rpd := int(configs.GetTimingConfig().RoundsPerDay)
	now := util.GetRoundCount()
	s := StateAt(r, now, rpd)

	if !s.Docked || s.PortIdx != portIdx {
		arriveRound := NextDockedRound(r, portIdx, now, rpd)
		gd := gametime.GetDate(arriveRound)
		mob.Command(`say ` + formatNotDockedQuote(r.Name, gd.Hour, gd.AmPm))
		return BoardNotDocked
	}

	if user.Character.Gold < r.Fare {
		mob.Command(fmt.Sprintf(`say Passage on %s is %d gold. Come back when you have it.`, r.Name, r.Fare))
		return BoardNoGold
	}

	user.Character.Gold -= r.Fare

	dockRoom := rooms.LoadRoom(roomId)

	if err := rooms.MoveToRoom(user.UserId, r.DeckRoom); err != nil {
		user.Character.Gold += r.Fare // refund
		mob.Command(`say Trouble at the gangplank. Your coin is returned.`)
		return BoardError
	}

	if dockRoom != nil {
		dockRoom.SendTextVisual(messaging.CategoryRoomDescription,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> pays the fare and crosses the gangplank aboard %s.`, user.Character.Name, r.Name),
			user.UserId)
	}
	user.SendText(messaging.CategoryRoomDescription,
		fmt.Sprintf(`You pay %d gold and cross the gangplank aboard %s.`, r.Fare, r.Name))

	return BoardOk
}

// formatNotDockedQuote is pure so it can be unit tested without the
// gametime/config globals.
func formatNotDockedQuote(vesselName string, hour int, amPm string) string {
	return fmt.Sprintf(`%s is out on the water. She ties up here again around %d %s.`, vesselName, hour, amPm)
}
```

- [ ] **Step 4: Run tests + build.**

Run: `go test ./internal/ferry/ -count=1 && go build ./...`
Expected: PASS + clean compile.

- [ ] **Step 5: Commit.**

```bash
git add internal/ferry/board.go internal/ferry/board_test.go
git commit -m "feat(ferry): Board — fare charge, MoveToRoom, schedule quotes"
```

---

### Task 7: Behavior-tree action `board_ferry`

**Files:**
- Create: `internal/behaviortree/actions_ferry.go`
- Test: `internal/behaviortree/actions_ferry_test.go`
- Modify: `internal/behaviortree/actions.go` (registry, after line 57)

- [ ] **Step 1: Write the failing registration test** (mirror `actions_move_player_test.go`):

```go
package behaviortree

import "testing"

func TestBoardFerry_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["board_ferry"]; !ok {
		t.Fatal(`board_ferry not in actionRegistry`)
	}
}
```

- [ ] **Step 2: Run to verify failure.**

Run: `go test ./internal/behaviortree/ -run TestBoardFerry -v`
Expected: FAIL.

- [ ] **Step 3: Implement `actions_ferry.go`.**

```go
package behaviortree

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/ferry"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actBoardFerry handles "ask <agent> passage <destination>". Params:
//
//	routes:            # destination keyword → routeid
//	  plymouth: stillwater_np_packet
//	  confluence: stillwater_confluence_barge
//
// Resolves the destination word from the ask text, then delegates the
// entire boarding flow (schedule check, fare, move, messaging) to
// ferry.Board. Returns Failure when the text names no known destination
// AND no boarding keyword is present, so other tree branches can try;
// returns Success (with mob dialogue) on every handled outcome.
func actBoardFerry(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}

	// Normalize the routes param (same YAML-map contortions as
	// actOpenInstancePortal's zones param).
	routesMap := map[string]string{}
	if raw, ok := params["routes"]; ok {
		switch m := raw.(type) {
		case map[string]any:
			for k, v := range m {
				if s, ok := v.(string); ok {
					routesMap[strings.ToLower(k)] = s
				}
			}
		case map[any]any:
			for k, v := range m {
				ks, kok := k.(string)
				vs, vok := v.(string)
				if kok && vok {
					routesMap[strings.ToLower(ks)] = vs
				}
			}
		case map[string]string:
			for k, v := range m {
				routesMap[strings.ToLower(k)] = v
			}
		}
	}
	if len(routesMap) == 0 {
		return Failure
	}

	text := strings.ToLower(ctx.Event.Text)
	routeId := ``
	for keyword, id := range routesMap {
		if strings.Contains(text, keyword) {
			routeId = id
			break
		}
	}

	if routeId == `` {
		// The keyword_match condition above this action already
		// established boarding intent — list the destinations.
		dests := make([]string, 0, len(routesMap))
		for k := range routesMap {
			dests = append(dests, k)
		}
		sort.Strings(dests)
		mob.Command(`say Passage where? I book for ` + strings.Join(dests, ` and `) + `.`)
		return Success
	}

	ferry.Board(user, mob, ctx.RoomId, routeId)
	return Success
}
```

- [ ] **Step 4: Register.** In `internal/behaviortree/actions.go`, after the `open_instance_portal` line:

```go
	actionRegistry["board_ferry"] = actBoardFerry
```

- [ ] **Step 5: Run tests + build.**

Run: `go test ./internal/behaviortree/ -run TestBoardFerry -v && go build ./...`
Expected: PASS + clean compile.

- [ ] **Step 6: Commit.**

```bash
git add internal/behaviortree/actions_ferry.go internal/behaviortree/actions_ferry_test.go internal/behaviortree/actions.go
git commit -m "feat(ferry): board_ferry behavior-tree action"
```

---

### Task 8: Route data files

**Files:**
- Create: `_datafiles/world/dogmud/ferries/stillwater_np_packet.yaml`
- Create: `_datafiles/world/dogmud/ferries/stillwater_confluence_barge.yaml`
- Create: `_datafiles/world/dogmud/ferries/confluence_np_barge.yaml`

- [ ] **Step 1: Write the three route files.**

`stillwater_np_packet.yaml`:

```yaml
routeid: stillwater_np_packet
name: the Lakewind Packet
deck_room: 6423
ports:
  - dock_room: 4118   # Stillwater — Boat Rental Pier
  - dock_room: 5502   # New Plymouth — The North Quay
crossing_hours: 2
layover_hours: 1
phase_offset_rounds: 0
fare: 75
```

`stillwater_confluence_barge.yaml`:

```yaml
routeid: stillwater_confluence_barge
name: the Grey Heron
deck_room: 6424
ports:
  - dock_room: 4118   # Stillwater — Boat Rental Pier
  - dock_room: 6109   # The Confluence — The Barge Dock
crossing_hours: 2
layover_hours: 1
phase_offset_rounds: 75
fare: 50
```

`confluence_np_barge.yaml`:

```yaml
routeid: confluence_np_barge
name: the Broadbeam
deck_room: 6425
ports:
  - dock_room: 6109   # The Confluence — The Barge Dock
  - dock_room: 5502   # New Plymouth — The North Quay
crossing_hours: 2
layover_hours: 1
phase_offset_rounds: 150
fare: 50
```

(Offsets 0/75/150 stagger the three vessels across the 224-round cycle so
no two dock-calls at the shared ports coincide constantly.)

- [ ] **Step 2: Commit.** (Boot test comes after the rooms exist — Task 10.)

```bash
git add _datafiles/world/dogmud/ferries/
git commit -m "content(ferry): three route definitions (packet, heron, broadbeam)"
```

---

### Task 9: The `ferries` zone — deck rooms + deckhand mobs

**Files:**
- Create: `_datafiles/world/dogmud/rooms/ferries/zone-config.yaml`
- Create: `_datafiles/world/dogmud/rooms/ferries/6423.yaml`, `6424.yaml`, `6425.yaml`
- Create: `_datafiles/world/dogmud/mobs/ferries/9574-a_weathered_deckhand.yaml`, `9575-a_barge_poleman.yaml`, `9576-a_river_bargeman.yaml`

- [ ] **Step 1: zone-config.**

```yaml
name: Ferries
roomid: 6423
defaultbiome: water
region: Windward Marches
non_cartesian: true
```

- [ ] **Step 2: Deck rooms.** Model: prose hard-wrapped ≤80 cols, `zone: Ferries`, `biome: water`, NO `exits:` block (the only exit is the controller's temporary gangplank), spaced coords, a couple of nouns, idle messages, deckhand spawninfo. `6423.yaml`:

```yaml
roomid: 6423
zone: Ferries
title: The Deck of the Lakewind Packet
description: >
  The packet's deck is scrubbed pale by years of weather and boots. A
  low deckhouse amidships shelters the wheel; coils of line hang from
  the rail pins in tidy loops, and the boards underfoot carry the
  constant small motion of open water. Benches along the gunwale seat
  a handful of passengers out of the spray. Whether the shore is near
  or far depends entirely on when you look.
biome: water
coord:
  x: 0
  y: 0
  z: 0
nouns:
  deckhouse: Low and weatherboarded, just tall enough to stand at the
    wheel inside. The glass is crazed with old salt.
  benches: Plank benches worn smooth, bolted to the deck. Initials
    have been knifed into the undersides where the crew does not
    bother to sand.
idlemessages:
- The rigging creaks as the packet takes a swell at a shallow angle.
- A line snaps taut, thrums, and goes slack again.
spawninfo:
  - mobid: 9574
    respawnrate: "15 real minutes"
```

`6424.yaml` (Grey Heron) and `6425.yaml` (Broadbeam) follow the identical
structure with coords `x: 10` and `x: 20`, barge-flavored prose (flat
cargo barge: sweeps/poles, tarped cargo mounds, a steering oar; the
Broadbeam is the novel-canon NP river barge — plainer, older, work-worn),
titles `The Deck of the Grey Heron` / `The Deck of the Broadbeam`, and
spawninfo mobids 9575 / 9576. Write full original prose per room (same
register as 6423 — working vessel, no numbers, no lore leaks).

- [ ] **Step 3: Deckhand mobs.** Model on the Town Crier (`mobs/the_confluence/9439-a_town_crier.yaml` — verified schema). `9574-a_weathered_deckhand.yaml`:

```yaml
mobid: 9574
zone: Ferries
behavior_archetype: noncombat_passive
non_combatant: true
charm_immune: true
hostile: false
statpool: 30
maxwander: 0
groups:
  - humanoid
idlecommands:
  - 'emote checks a coil of line on its pin, finds it to his liking,
    and moves on down the rail.'
  - ''
  - 'say She rides better with a full hold. Light days she slaps the
    chop like a skipping stone.'
  - ''
  - 'emote squints at the water ahead, then at the sky, and appears
    satisfied with both.'
  - ''
activitylevel: 10
character:
  name: A Weathered Deckhand
  description: |
    A Weathered Deckhand works the packet's lines with the unhurried
    economy of someone who has done every task aboard a thousand
    times. His hands are rope-burned smooth and his squint is
    permanent. He answers questions in the shortest form that is not
    rude, and some that are.
  speciesid: 1
  level: 1
  gold: 2
  stats:
    strength:
      training: 5
    vitality:
      training: 4
```

`9575-a_barge_poleman.yaml` and `9576-a_river_bargeman.yaml`: same
structure, `mobid` 9575/9576, original names/descriptions/idlecommands in
the same register (river bargemen, not lake sailors). **Filename must be
`<mobid>-<ConvertForFilename(character.name)>.yaml`** — e.g. `A Barge
Poleman` → `9575-a_barge_poleman.yaml`.

- [ ] **Step 4: Name-collision check.** Grep the mob roster for the three
new names before finalizing (`grep -ri "deckhand\|poleman\|bargeman"
_datafiles/world/dogmud/mobs/`) — per the no-name-recycling rule, rename if
an existing mob already uses the name.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/rooms/ferries/ _datafiles/world/dogmud/mobs/ferries/
git commit -m "content(ferry): Ferries zone — three vessel decks + deckhands"
```

---

### Task 10: First boot test

- [ ] **Step 1: Instance-wipe SOP** (template edits to existing rooms come next task, but establish the habit now):

```bash
rm -rf _datafiles/world/dogmud/mobs.instances/* _datafiles/world/dogmud/rooms.instances/*
```

- [ ] **Step 2: Boot.**

Run: `go run .` (from repo root — NEVER `go run main.go`)
Expected in the log: `rooms.LoadDataFiles()` count +3 rooms (1342 total if baseline 1339), `mobs.LoadDataFiles()` +3, **`ferry.LoadDataFiles() loadedCount=3`**, `ValidateZoneConsistency` errors=0, no panics. Kill the server after verifying (SOP: kill all GoMud/go processes).

- [ ] **Step 3: Fix anything the boot surfaces, then commit fixes if any.**

```bash
git add -A && git commit -m "fix(ferry): boot-test fixes"   # only if needed
```

---

### Task 11: Dock agents — mobs, behaviors, dock-room edits

**Files:**
- Create: `_datafiles/world/dogmud/mobs/stillwater/9571-the_ferry_agent.yaml`
- Create: `_datafiles/world/dogmud/mobs/new_plymouth_docks/9572-the_quay_agent.yaml`
- Create: `_datafiles/world/dogmud/mobs/the_confluence/9573-the_barge_master.yaml`
- Create: `_datafiles/world/dogmud/behaviors/stillwater/9571-the_ferry_agent.yaml`
- Create: `_datafiles/world/dogmud/behaviors/new_plymouth_docks/9572-the_quay_agent.yaml`
- Create: `_datafiles/world/dogmud/behaviors/the_confluence/9573-the_barge_master.yaml`
- Modify: `_datafiles/world/dogmud/rooms/stillwater/4118.yaml`, `_datafiles/world/dogmud/rooms/new_plymouth_docks/5502.yaml`, `_datafiles/world/dogmud/rooms/the_confluence/6109.yaml`

- [ ] **Step 1: Agent mobs.** Same schema as the deckhands (Task 9 Step 3), `zone:` matching the host zone's display name (`Stillwater` / `New Plymouth Docks` / `The Confluence`), `maxwander: 0`, `statpool: 30`, `non_combatant: true`. Names: **The Ferry Agent** (9571), **The Quay Agent** (9572), **The Barge Master** (9573) — run the same name-collision grep as Task 9 Step 4. Descriptions: original prose — a fare-taker with a strongbox and sailing-times slate, in each port's register. Idlecommands should advertise the service, e.g. for 9571:

```yaml
idlecommands:
  - 'say Passage to Plymouth or the Confluence -- fair rates, dry
    benches, no walking.'
  - ''
  - 'emote chalks a departure time onto a slate, frowns at the light,
    and rubs an hour off it.'
  - ''
```

- [ ] **Step 2: Agent behavior trees.** Modeled on the Threshold-Keeper (verified: selector of `player_ask` sequences; grant/transactional branches FIRST). `behaviors/stillwater/9571-the_ferry_agent.yaml`:

```yaml
# The Ferry Agent — mob 9571 — books passage at the Stillwater pier.
# Serves BOTH Stillwater routes; destination word picks the route.
tree:
  type: selector
  children:

    # ── PURCHASE: boarding intent → board_ferry resolves destination,
    # schedule, fare, and movement. Sits FIRST (grant-node-first SOP). ──
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [passage, board, boarding, ticket, fare, ride, sail, plymouth, confluence]
        - type: action
          do: board_ferry
          routes:
            plymouth: stillwater_np_packet
            confluence: stillwater_confluence_barge

    # ── info / greetings ──
    - type: sequence
      event: player_ask
      children:
        - type: condition
          check: keyword_match
          keywords: [hello, hi, help, who, what, ferry, boat, packet, barge, agent]
        - type: action
          do: say
          text: I book passage on the water. The Lakewind Packet runs to Plymouth and the Grey Heron runs down to the Confluence -- paid at the plank, off again at the far pier.
        - type: action
          do: say
          text: Ask me for passage and name your landing -- Plymouth, or the Confluence. If the boat is out, I will tell you when she ties up next.
        - type: action
          do: send_user_text
          text: '<ansi fg="181">  [Try: ask agent passage plymouth]</ansi>'
```

`9572-the_quay_agent.yaml` (NP): same tree shape with
`routes: {stillwater: stillwater_np_packet, confluence: confluence_np_barge}`,
destination keywords `[..., stillwater, confluence]`, info text naming the
Lakewind Packet (to Stillwater) and the Broadbeam (to the Confluence), hint
`ask agent passage stillwater`.

`9573-the_barge_master.yaml` (Confluence): `routes: {stillwater:
stillwater_confluence_barge, plymouth: confluence_np_barge}`, keywords
`[..., stillwater, plymouth]`, info text naming the Grey Heron and the
Broadbeam, hint `ask master passage plymouth`.

Discoverability SOP: every trigger word above appears in the agent's say
text, the hint line, or the dock-room prose (next step).

- [ ] **Step 3: Dock-room edits.** In each of 4118 / 5502 / 6109: (a) append a berth sentence to `description` making the service visible, e.g. for 4118: `A stubby berth at the pier's end serves the passage boats; a painted board lists sailings, and the <ansi fg="mobname">ferry agent</ansi> keeps a strongbox beside it.`; (b) add a `sailings board` noun describing destinations in prose (no times — they shift); (c) add the agent to `spawninfo`:

```yaml
  - mobid: 9571
    respawnrate: "15 real minutes"
```

(5502 → 9572, 6109 → 9573. Preserve every existing field in these room
files — these are surgical additions, not rewrites.)

- [ ] **Step 4: Boot + smoke.** Instance wipe (Task 10 Step 1 — REQUIRED now: 4118/5502/6109 are template edits that stale instance saves would shadow), then `go run .`; verify `ferry.LoadDataFiles() loadedCount=3`, mobs +6 total, no panics. In a client: `ask agent passage plymouth` at 4118 charges gold and moves you aboard (or quotes the next docking if she's out). Kill server after.

- [ ] **Step 5: Commit.**

```bash
git add _datafiles/world/dogmud/mobs/ _datafiles/world/dogmud/behaviors/ _datafiles/world/dogmud/rooms/stillwater/4118.yaml _datafiles/world/dogmud/rooms/new_plymouth_docks/5502.yaml _datafiles/world/dogmud/rooms/the_confluence/6109.yaml
git commit -m "content(ferry): dock agents, boarding behaviors, berth prose at all three ports"
```

---

### Task 12: Severin Pell points at the live service

**Files:**
- Modify: `_datafiles/world/dogmud/dialogue/hartcharn/9182.yaml`

- [ ] **Step 1:** Read the file. Severin Pell is Hartcharn's Ferry & Coach Agent, written when the packet was only an idea. Update ONLY the lines that describe the Stillwater↔NP packet so they present it as running now (e.g. sells the *idea*: "the packet office in Stillwater books passage — the Lakewind Packet runs the lake to Plymouth"), keeping his voice and every other node untouched. No semicolons in NPC text; first-person NPC voice; hints in narrator voice.

- [ ] **Step 2: Commit.**

```bash
git add _datafiles/world/dogmud/dialogue/hartcharn/9182.yaml
git commit -m "content(ferry): Severin Pell's dialogue references the running packet"
```

---

### Task 13: Full boot verification + cartcheck

- [ ] **Step 1:** Instance wipe, then `go run .`. Verify the complete checklist: rooms 1342, mobs 603 (597+6), `ferry.LoadDataFiles() loadedCount=3`, `ValidateZoneConsistency` errors=0 (mode=panic), zero panics, zero load warnings from the new files.
- [ ] **Step 2:** As admin in a client: `cartcheck ferries` → clean (zone is non_cartesian; expect no findings).
- [ ] **Step 3:** Watch one full transition live: `goto 6423`, wait ≤15 real minutes, confirm departure emote, ambiance lines, arrival emote, gangplank appears in exits (`look`) when docked and is walkable (`gangplank`).
- [ ] **Step 4:** Kill all GoMud/go processes (SOP).
- [ ] **Step 5: Commit any fixes.**

---

### Task 14: Harness feel test

**Files:**
- Create: `tools/playtest/goals/ferry-stage1.yaml`

- [ ] **Step 1: Write the goals file** (match the structure of an existing file in `tools/playtest/goals/`):

Goals to encode:
1. Find the ferry service in Stillwater from room prose alone (discoverability).
2. Ask the agent for passage with no destination → get the destination list.
3. Ask while the vessel is out → get a plausible next-docking time quote.
4. Board with sufficient gold → verify gold decreased and location is the deck.
5. Ride a full crossing → arrival emote fires, gangplank appears, walk ashore at the far port.
6. Attempt to board with insufficient gold → polite refusal, no charge.
7. Sleep on deck through a departure → still aboard on arrival (sleeper
   rides through; no ejection, no error).
8. Feel notes: does the crossing length feel right? Is the deck alive
   enough (deckhand, ambiance)?
9. Web client spot-check (human or screenshot step): the deck room
   renders sanely on the leather map (temp gangplank exit = portal/named
   exit; no fog-of-war weirdness) — closes the spec's mapper open item.

- [ ] **Step 2: Run** `/playtest local feature-tester ferry-stage1.yaml` (fresh boot, instance wipe first). Triage findings: fix mechanical bugs now; log feel/tuning notes for the user's evening playtest per the defer-tuning rule.

- [ ] **Step 3: Commit** goals file + any fixes.

```bash
git add tools/playtest/goals/ferry-stage1.yaml
git commit -m "test(ferry): stage-1 harness feel-test goals + findings fixes"
```

---

## Post-plan notes

- **Spec knob revision:** crossing 2h / layover 1h replaces the spec's
  placeholder 6h/2h (RoundsPerDay=900 makes 6h a 15-real-minute ride; spec
  defers exact values to plan). Amend the spec's Config Knobs line in the
  same commit as Task 8.
- **Pre-push SOP applies** before any prod push (PATCH_NOTES, LogToFile,
  boot test) — not part of this plan.
- Stage 2 (trade factors + cross-region stock) is a separate plan.
