package questengine

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Character.StoreItem returns false and stores nothing once carried weight
// would exceed twice carry capacity — the game's own "crushed" encumbrance
// tier, which players do reach. GameBridge.GiveItem discarded that boolean and
// told the player "You receive a X" regardless, so a quest reward item could
// vanish while the trigger's remaining actions (typically the quest-advance
// grant) ran anyway.

const (
	gitItemId = 91101
	gitUserId = 9111
)

func seedGiveItemFixture(t *testing.T) (*users.UserRecord, func()) {
	t.Helper()

	cleanItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		gitItemId: {
			ItemId:  gitItemId,
			Name:    "quest ledger",
			Type:    items.Object,
			Subtype: items.Mundane,
			Weight:  5.0,
		},
	})

	u := users.NewTestUser(gitUserId, "givetest", "Givetest", uint64(gitUserId))
	cleanUsers := users.SeedUsersForTest(map[int]*users.UserRecord{gitUserId: u})

	// Drain anything the fixture itself queued so assertions below only see
	// what the grant path produced.
	events.DrainQueuedMessagesForTest(gitUserId)
	events.DrainQueuedItemOwnershipForTest(gitUserId)

	return u, func() {
		events.DrainQueuedMessagesForTest(gitUserId)
		events.DrainQueuedItemOwnershipForTest(gitUserId)
		cleanUsers()
		cleanItems()
	}
}

// crushPlayer drives the character past its carry limit. Carry capacity is
// Strength x CarryCapacityMultiplier and StoreItem refuses beyond 2x that, so
// zeroing Strength makes any weighted item unstorable.
func crushPlayer(u *users.UserRecord) {
	u.Character.Stats.Strength.ValueAdj = 0
}

func joined(msgs []string) string { return strings.Join(msgs, "\n") }

func TestGameBridge_GiveItem_OverloadedPlayer_NoPhantomDelivery(t *testing.T) {
	u, cleanup := seedGiveItemFixture(t)
	defer cleanup()

	crushPlayer(u)

	bridge := NewGameBridge(u, 1)
	err := bridge.GiveItem(gitItemId)

	if err == nil {
		t.Error("GiveItem returned nil error for an overloaded player; the trigger's remaining actions would still run")
	}
	if got := len(u.Character.Items); got != 0 {
		t.Errorf("player holds %d items; want 0 (nothing was actually stored)", got)
	}

	msgs := joined(events.DrainQueuedMessagesForTest(gitUserId))
	if strings.Contains(msgs, "You receive") {
		t.Errorf("player was told they received an item they do not have:\n%s", msgs)
	}
	if !strings.Contains(msgs, "carrying too much") {
		t.Errorf("player got no usable explanation for the failure:\n%s", msgs)
	}
}

func TestGameBridge_GiveItem_HappyPath_StillDelivers(t *testing.T) {
	u, cleanup := seedGiveItemFixture(t)
	defer cleanup()

	bridge := NewGameBridge(u, 1)
	if err := bridge.GiveItem(gitItemId); err != nil {
		t.Fatalf("GiveItem on an unencumbered player returned %v; want nil", err)
	}

	if got := len(u.Character.Items); got != 1 {
		t.Fatalf("player holds %d items; want 1", got)
	}

	msgs := joined(events.DrainQueuedMessagesForTest(gitUserId))
	if !strings.Contains(msgs, "You receive") {
		t.Errorf("successful delivery sent no receipt message:\n%s", msgs)
	}
}

// ExecuteAction must surface the failure so executeActions abandons the rest
// of the trigger instead of granting the quest token on a phantom item.
func TestExecuteAction_GiveItem_PropagatesFailure(t *testing.T) {
	u, cleanup := seedGiveItemFixture(t)
	defer cleanup()

	crushPlayer(u)

	bridge := NewGameBridge(u, 1)
	if err := ExecuteAction(ActionDef{GiveItem: gitItemId}, bridge); err == nil {
		t.Error("ExecuteAction swallowed a failed give_item; want an error so the trigger aborts")
	}
}
