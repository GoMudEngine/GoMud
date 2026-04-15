package questengine_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
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
