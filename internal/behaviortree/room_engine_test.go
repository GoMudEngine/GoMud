package behaviortree

import (
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// clearEngineRoomState removes room IDs from roomTrees and noRoomTree under
// the engine mutex. Defer this in every test that writes to those maps.
func clearEngineRoomState(t *testing.T, roomIds ...int) {
	t.Helper()
	e := GetEngine()
	e.mu.Lock()
	defer e.mu.Unlock()
	for _, id := range roomIds {
		delete(e.roomTrees, id)
		delete(e.noRoomTree, id)
	}
}

// clearRoomStates removes room IDs from the roomStates map under roomStateMu.
// Defer this in EnsureRoomBTreeState tests.
func clearRoomStates(t *testing.T, roomIds ...int) {
	t.Helper()
	roomStateMu.Lock()
	defer roomStateMu.Unlock()
	for _, id := range roomIds {
		delete(roomStates, id)
	}
}

// writeTempYAML writes YAML content to a temp dir file and returns the path.
func writeTempYAML(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("writeTempYAML: %v", err)
	}
	return path
}

// TestTryRoomBehavior_NoRoom_ReturnsFalse verifies that when LoadRoom returns
// nil (unknown room id), TryRoomBehavior returns false and does NOT set the
// negative cache for that id (it never reached the tree-loading path).
func TestTryRoomBehavior_NoRoom_ReturnsFalse(t *testing.T) {
	const roomId = 99901

	defer clearEngineRoomState(t, roomId)

	result := TryRoomBehavior(roomId, EventContext{EventType: "room_enter"})
	if result {
		t.Errorf("expected false for unknown room, got true")
	}
	if GetEngine().HasNoRoomTree(roomId) {
		t.Errorf("noRoomTree[%d] must NOT be set when room itself is missing", roomId)
	}
}

// TestTryRoomBehavior_NoTreeFile_NegativeCache verifies that a room with no
// YAML on disk: first call returns false AND sets noRoomTree; second call
// short-circuits (negative cache) without re-reading disk.
func TestTryRoomBehavior_NoTreeFile_NegativeCache(t *testing.T) {
	// Use a zone name that will never exist on disk in the test data dir.
	const roomId = 99902
	const zone = "TestZone_1_6_DoesNotExist"

	cleanRoom := seedTestRoom(t, roomId, zone)
	defer cleanRoom()
	defer clearEngineRoomState(t, roomId)

	event := EventContext{EventType: "room_enter"}

	// First call: file not on disk → false, negative cache set.
	result1 := TryRoomBehavior(roomId, event)
	if result1 {
		t.Errorf("first call: expected false, got true")
	}
	if !GetEngine().HasNoRoomTree(roomId) {
		t.Errorf("first call: noRoomTree[%d] should be set after missing-file", roomId)
	}

	// Second call: should short-circuit via negative cache.
	result2 := TryRoomBehavior(roomId, event)
	if result2 {
		t.Errorf("second call: expected false (negative cache), got true")
	}
}

// TestTryRoomBehavior_LoadParseError_FailureNotCached verifies that a
// malformed YAML tree: returns false AND does NOT set the negative cache
// (parse errors are not cached for rooms — retry is allowed next tick).
func TestTryRoomBehavior_LoadParseError_FailureNotCached(t *testing.T) {
	const roomId = 99903
	const zone = "TestZone_1_6_DoesNotExist"

	tmpDir := t.TempDir()
	malformedYAML := "tree: [\nnot: valid: yaml: :::\n"
	yamlPath := writeTempYAML(t, tmpDir, "99903.yaml", malformedYAML)

	cleanRoom := seedTestRoom(t, roomId, zone)
	defer cleanRoom()
	defer clearEngineRoomState(t, roomId)

	// Pre-load via LoadRoomTree to inject the parse-error path by directly
	// testing LoadTreeFromFile with bad YAML (simulated via LoadRoomTree).
	// Since TryRoomBehavior uses GetRoomBehaviorPath which won't match tmpDir,
	// we test LoadRoomTree directly for the error-not-cached contract.
	err := GetEngine().LoadRoomTree(roomId, yamlPath)
	if err == nil {
		t.Fatal("expected parse error from malformed YAML, got nil")
	}
	// After a failed LoadRoomTree, the negative cache must NOT be set
	// (LoadRoomTree only sets the tree on success; it doesn't set noRoomTree).
	if GetEngine().HasNoRoomTree(roomId) {
		t.Errorf("HasNoRoomTree[%d] must be false after parse error — parse errors are not cached", roomId)
	}
}

// TestTryRoomBehavior_HappyPath_Success verifies that when a valid tree is
// pre-loaded via LoadRoomTree, TryRoomBehavior returns true.
func TestTryRoomBehavior_HappyPath_Success(t *testing.T) {
	const roomId = 99904
	const zone = "TestZone_1_6_DoesNotExist"

	// A minimal always-success tree using the set_state action.
	// Params are inlined at the node level (no nested "params:" key).
	happyYAML := `tree:
  type: action
  do: set_state
  key: t
  value: ok
`
	tmpDir := t.TempDir()
	yamlPath := writeTempYAML(t, tmpDir, "99904.yaml", happyYAML)

	cleanRoom := seedTestRoom(t, roomId, zone)
	defer cleanRoom()
	defer clearEngineRoomState(t, roomId)
	defer clearRoomStates(t, roomId)

	if err := GetEngine().LoadRoomTree(roomId, yamlPath); err != nil {
		t.Fatalf("LoadRoomTree failed: %v", err)
	}

	event := EventContext{EventType: "room_enter"}
	result := TryRoomBehavior(roomId, event)
	if !result {
		t.Errorf("expected true (Success tree), got false")
	}
}

// TestTryRoomBehavior_RoomCommand_ReturnsIntercepted verifies the intercept
// contract: a tree with `do: intercept` returns true for room_command events;
// a tree without intercept returns false for the same event type.
func TestTryRoomBehavior_RoomCommand_ReturnsIntercepted(t *testing.T) {
	const interceptRoomId = 99905
	const noInterceptRoomId = 99906
	const zone = "TestZone_1_6_DoesNotExist"

	interceptYAML := `tree:
  type: action
  do: intercept
`
	noInterceptYAML := `tree:
  type: action
  do: set_state
  key: t
  value: ok
`
	tmpDir := t.TempDir()
	interceptPath := writeTempYAML(t, tmpDir, "99905.yaml", interceptYAML)
	noInterceptPath := writeTempYAML(t, tmpDir, "99906.yaml", noInterceptYAML)

	// Both rooms must be seeded in a single SeedRoomsForTest call because each
	// call replaces the global roomManager and would evict the previous room.
	cleanRooms := rooms.SeedRoomsForTest(map[int]*rooms.Room{
		interceptRoomId: {RoomId: interceptRoomId, Zone: zone, Title: "Test Intercept Room", Exits: map[string]exit.RoomExit{}},
		noInterceptRoomId: {RoomId: noInterceptRoomId, Zone: zone, Title: "Test No-Intercept Room", Exits: map[string]exit.RoomExit{}},
	}, map[string]*rooms.ZoneConfig{})
	defer cleanRooms()
	defer clearEngineRoomState(t, interceptRoomId, noInterceptRoomId)
	defer clearRoomStates(t, interceptRoomId, noInterceptRoomId)

	if err := GetEngine().LoadRoomTree(interceptRoomId, interceptPath); err != nil {
		t.Fatalf("LoadRoomTree (intercept) failed: %v", err)
	}
	if err := GetEngine().LoadRoomTree(noInterceptRoomId, noInterceptPath); err != nil {
		t.Fatalf("LoadRoomTree (no-intercept) failed: %v", err)
	}

	event := EventContext{EventType: "room_command", Text: "go north"}

	// Tree with intercept action → ctx.Intercepted = true → returns true.
	if !TryRoomBehavior(interceptRoomId, event) {
		t.Errorf("intercept tree: expected true for room_command, got false")
	}

	// Tree without intercept action → ctx.Intercepted stays false → returns false.
	if TryRoomBehavior(noInterceptRoomId, event) {
		t.Errorf("no-intercept tree: expected false for room_command, got true")
	}
}

// TestEnsureRoomBTreeState_PersistsAcrossCalls verifies that:
//  1. The same pointer is returned on repeated calls for the same roomId.
//  2. Concurrent goroutines all receive the same pointer.
//  3. State written in one call is visible in subsequent calls.
func TestEnsureRoomBTreeState_PersistsAcrossCalls(t *testing.T) {
	const roomId = 99907

	defer clearRoomStates(t, roomId)

	// Call 1 — establishes state.
	state1 := EnsureRoomBTreeState(roomId)
	if state1 == nil {
		t.Fatal("EnsureRoomBTreeState returned nil")
	}

	// Call 2 — must return exact same pointer.
	state2 := EnsureRoomBTreeState(roomId)
	if state1 != state2 {
		t.Errorf("expected same pointer on second call: got %p vs %p", state1, state2)
	}

	// Write a value and verify it persists.
	state1.Set("ping", "pong")
	state3 := EnsureRoomBTreeState(roomId)
	val := state3.Get("ping")
	if val == nil {
		t.Error("state value 'ping' not found after Set")
	} else if val != "pong" {
		t.Errorf("expected 'pong', got %v", val)
	}

	// Concurrent goroutines must all receive the same pointer.
	const goroutines = 20
	results := make([]*BehaviorState, goroutines)
	var wg sync.WaitGroup
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		i := i
		go func() {
			defer wg.Done()
			results[i] = EnsureRoomBTreeState(roomId)
		}()
	}
	wg.Wait()

	for i, s := range results {
		if s != state1 {
			t.Errorf("goroutine %d: expected pointer %p, got %p", i, state1, s)
		}
	}
}

// TestQueueDelayed_RecoversFromPanic verifies that a panicking delayed-action
// closure does not crash DrainQueue and does not abort subsequent ready
// closures in the same DrainQueue call. The engine-level safeExecuteDelayed
// wrapper recovers the panic and logs at mudlog.Error.
func TestQueueDelayed_RecoversFromPanic(t *testing.T) {
	e := GetEngine()

	firstRan := false
	secondRan := false

	e.QueueDelayed(0, func() {
		firstRan = true
		panic("intentional test panic")
	})
	e.QueueDelayed(0, func() {
		secondRan = true
	})

	// Must not propagate the panic — DrainQueue returns normally.
	e.DrainQueue()

	if !firstRan {
		t.Errorf("first closure did not run before panicking")
	}
	if !secondRan {
		t.Errorf("second closure did not run after first panicked — wrapper failed to contain the panic")
	}
}

// TestQueueDelayed_RunsSucceededClosure verifies the happy-path contract: a
// normal closure submitted via QueueDelayed runs exactly once when DrainQueue
// is called. Locks the contract that the wrapper does not swallow non-panic
// returns or run a closure twice.
func TestQueueDelayed_RunsSucceededClosure(t *testing.T) {
	e := GetEngine()

	var mu sync.Mutex
	ran := 0

	e.QueueDelayed(0, func() {
		mu.Lock()
		ran++
		mu.Unlock()
	})

	e.DrainQueue()

	mu.Lock()
	defer mu.Unlock()
	if ran != 1 {
		t.Errorf("expected closure to run exactly once, got %d", ran)
	}
}
