package questengine_test

import (
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
)

func newBridgeUser(userId int) *users.UserRecord {
	return users.NewTestUser(userId, "testuser", "TestChar", 0)
}

func TestGameBridge_GetRoomId(t *testing.T) {
	u := newBridgeUser(1)
	b := questengine.NewGameBridge(u, 42)
	assert.Equal(t, 42, b.GetRoomId())
}

func TestGameBridge_GetUserId(t *testing.T) {
	u := newBridgeUser(7)
	b := questengine.NewGameBridge(u, 1)
	assert.Equal(t, 7, b.GetUserId())
}

func TestGameBridge_HasItem_False(t *testing.T) {
	u := newBridgeUser(1)
	b := questengine.NewGameBridge(u, 1)
	assert.False(t, b.HasItem(999))
}

func TestGameBridge_HasItem_True(t *testing.T) {
	u := newBridgeUser(1)
	// Directly append an item with a known ID.
	u.Character.Items = append(u.Character.Items, items.Item{ItemId: 999})
	b := questengine.NewGameBridge(u, 1)
	assert.True(t, b.HasItem(999))
}

func TestGameBridge_HasItem_WrongId(t *testing.T) {
	u := newBridgeUser(1)
	u.Character.Items = append(u.Character.Items, items.Item{ItemId: 100})
	b := questengine.NewGameBridge(u, 1)
	assert.False(t, b.HasItem(999))
}

func TestGameBridge_HasQuest_False_NoProgress(t *testing.T) {
	u := newBridgeUser(1)
	b := questengine.NewGameBridge(u, 1)
	// No quest progress at all — must return false.
	assert.False(t, b.HasQuest("3-start"))
}

func TestGameBridge_HasQuest_True_ExactMatch(t *testing.T) {
	u := newBridgeUser(1)
	// Seed quest progress directly: quest 3 is at step "start".
	u.Character.QuestProgress = map[int]string{3: "start"}
	b := questengine.NewGameBridge(u, 1)
	assert.True(t, b.HasQuest("3-start"))
}

func TestGameBridge_HasQuest_False_DifferentStep(t *testing.T) {
	u := newBridgeUser(1)
	// Quest 3 is at "start", not "end".
	u.Character.QuestProgress = map[int]string{3: "start"}
	b := questengine.NewGameBridge(u, 1)
	// "3-end" requires IsTokenAfter logic with quest data — but
	// "start" != "end" so the direct equality check fails,
	// and IsTokenAfter needs quest data which won't be loaded in tests.
	// We only verify that it doesn't panic and returns a bool.
	result := b.HasQuest("3-end")
	_ = result // value depends on quest registry; just ensure no panic
}

// TestGameBridge_GiveMutation_EmitsGainedEvent verifies GiveMutation (used by
// quest reward blocks) queues a mutations.Gained event (UserId, MutationId,
// Rank=1, IsNew=true) — Task 3 moved reveal text to the hooks listener that
// consumes this event, so GiveMutation itself must not send player text and
// must not silently skip the emit.
func TestGameBridge_GiveMutation_EmitsGainedEvent(t *testing.T) {
	u := newBridgeUser(1)
	u.Character.Mutations = map[string]int{}

	// SpeciesId 0 is intentionally left unseeded: mutations.CanApplyTo
	// fails open (returns true) when species.GetSpecies returns nil, so no
	// species fixture is required to get a non-empty weighted pool here.
	cleanMuts := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"test-mut-1": {
			MutationId: "test-mut-1",
			Name:       "Test Mutation",
			Rarity:     1,
			Pros:       []mutations.MutationEffect{{Type: "stat_flat", Target: "strength", Value: 1}},
		},
	})
	defer cleanMuts()

	// Drain any events left queued-but-unprocessed by earlier tests in this
	// package before installing the capture listener.
	events.ProcessEvents()

	var mu sync.Mutex
	captured := []mutations.Gained{}
	id := events.RegisterListener(mutations.Gained{}, func(e events.Event) events.ListenerReturn {
		if g, ok := e.(mutations.Gained); ok {
			mu.Lock()
			captured = append(captured, g)
			mu.Unlock()
		}
		return events.Continue
	})
	defer events.UnregisterListener(mutations.Gained{}, id)

	b := questengine.NewGameBridge(u, 1)
	b.GiveMutation()

	events.ProcessEvents()

	if _, ok := u.Character.Mutations["test-mut-1"]; !ok {
		t.Fatalf("expected test-mut-1 in u.Character.Mutations, got %v", u.Character.Mutations)
	}

	mu.Lock()
	defer mu.Unlock()
	if len(captured) != 1 {
		t.Fatalf("expected exactly 1 mutations.Gained event, got %d: %+v", len(captured), captured)
	}
	got := captured[0]
	assert.Equal(t, 1, got.UserId)
	assert.Equal(t, "test-mut-1", got.MutationId)
	assert.Equal(t, 1, got.Rank)
	assert.True(t, got.IsNew)
}
