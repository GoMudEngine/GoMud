package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// buildPlayerState's GiveItem backs dialogue `givesItem:`. It discarded
// StoreItem's return, so an overloaded player was told "You receive a X" for
// an item that was never stored AND got an ItemOwnership{Gained:true} event —
// which ItemOwnership_CheckItemQuests uses to fire item_gain quest triggers
// and auto-grant QuestToken items. Quest state advanced on a phantom item, and
// the granting node was then excluded and could not fire again.

const (
	tgiItemId = 91201
	tgiUserId = 9121
)

func seedTalkGiveItemFixture(t *testing.T) (*users.UserRecord, func()) {
	t.Helper()

	cleanItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		tgiItemId: {
			ItemId:  tgiItemId,
			Name:    "sealed letter",
			Type:    items.Object,
			Subtype: items.Mundane,
			Weight:  5.0,
		},
	})

	u := users.NewTestUser(tgiUserId, "talkgive", "Talkgive", uint64(tgiUserId))
	cleanUsers := users.SeedUsersForTest(map[int]*users.UserRecord{tgiUserId: u})

	events.DrainQueuedMessagesForTest(tgiUserId)
	events.DrainQueuedItemOwnershipForTest(tgiUserId)

	return u, func() {
		events.DrainQueuedMessagesForTest(tgiUserId)
		events.DrainQueuedItemOwnershipForTest(tgiUserId)
		cleanUsers()
		cleanItems()
	}
}

func TestDialogueGiveItem_OverloadedPlayer_NoPhantomDeliveryOrEvent(t *testing.T) {
	u, cleanup := seedTalkGiveItemFixture(t)
	defer cleanup()

	// Carry capacity is Strength x CarryCapacityMultiplier; StoreItem refuses
	// beyond 2x that. Zero Strength makes any weighted item unstorable.
	u.Character.Stats.Strength.ValueAdj = 0

	buildPlayerState(u).GiveItem(tgiItemId)

	if got := len(u.Character.Items); got != 0 {
		t.Errorf("player holds %d items; want 0 (nothing was actually stored)", got)
	}

	if evts := events.DrainQueuedItemOwnershipForTest(tgiUserId); len(evts) != 0 {
		t.Errorf("queued %d ItemOwnership events for an undelivered item; want 0 "+
			"(these fire item_gain quest triggers and auto-grant quest tokens)", len(evts))
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(tgiUserId), "\n")
	if strings.Contains(msgs, "You receive") {
		t.Errorf("player was told they received an item they do not have:\n%s", msgs)
	}
	if !strings.Contains(msgs, "carrying too much") {
		t.Errorf("player got no usable explanation for the failure:\n%s", msgs)
	}
}

func TestDialogueGiveItem_HappyPath_StillDeliversAndFiresEvent(t *testing.T) {
	u, cleanup := seedTalkGiveItemFixture(t)
	defer cleanup()

	buildPlayerState(u).GiveItem(tgiItemId)

	if got := len(u.Character.Items); got != 1 {
		t.Fatalf("player holds %d items; want 1", got)
	}

	evts := events.DrainQueuedItemOwnershipForTest(tgiUserId)
	if len(evts) != 1 || !evts[0].Gained {
		t.Errorf("want exactly one ItemOwnership{Gained:true}; got %+v", evts)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(tgiUserId), "\n")
	if !strings.Contains(msgs, "You receive") {
		t.Errorf("successful delivery sent no receipt message:\n%s", msgs)
	}
}
