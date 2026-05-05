package hooks

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// CheckStorageFees runs on every NewRound and checks whether the game
// month has changed. When it has, it charges all online users their
// storage fee. Offline users are charged on next login via
// ChargeStorageFee (called from user validation or login path).
func CheckStorageFees(e events.Event) events.ListenerReturn {

	gdNow := gametime.GetDate()
	currentMonth := gdNow.Year*12 + gdNow.Month

	for _, u := range users.GetAllActiveUsers() {
		ChargeStorageFee(u, currentMonth)
	}

	return events.Continue
}

// ChargeStorageFee processes the monthly storage fee for a single user.
// Safe to call multiple times — uses StorageFeeLastMonth to prevent
// double-charging. Called from the round tick for online users and
// from the login path for returning offline users.
//
// Billing is per-slot (stack), not per-unit. A stack of 50 iron ore
// in one slot costs 1g, not 50g. Forfeiture drops the cheapest whole
// slot(s) (by spec.Value × Count) until the shortfall is covered.
func ChargeStorageFee(u *users.UserRecord, currentMonth int) {
	if u.ItemStorage.SlotCount() == 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	if u.Character.StorageFeeLastMonth >= currentMonth {
		return // Already charged this month
	}

	feePerSlot := int(configs.GetBalanceConfig().StorageFeePerItem)
	if feePerSlot <= 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	slotCount := u.ItemStorage.SlotCount()
	fee := slotCount * feePerSlot

	if u.Character.Bank >= fee {
		// Can pay in full
		u.Character.Bank -= fee
		u.Character.StorageFeeLastMonth = currentMonth

		// Warn if they won't be able to pay next month
		if u.Character.Bank < fee {
			msg := fmt.Sprintf(
				"Thornwall Bank Notice: Your monthly storage fee of "+
					"%dg has been collected (%d slot(s) at %dg/slot). "+
					"You have %dg remaining in your account. Next "+
					"month's fee will be %dg -- please deposit "+
					"additional gold or retrieve items to avoid "+
					"forfeiture.",
				fee, slotCount, feePerSlot, u.Character.Bank, fee)
			u.Inbox.Add(users.Message{
				FromName: "Thornwall Bank",
				Message:  msg,
				DateSent: time.Now(),
			})
		}
		return
	}

	// Cannot pay in full — deduct what they have, forfeit cheapest stacks
	available := u.Character.Bank
	u.Character.Bank = 0
	shortfall := fee - available

	// Sort slots by per-stack value (spec.Value × Count) ascending — cheapest first
	type valuedSlot struct {
		idx   int
		value int
		name  string
	}

	slots := u.ItemStorage.GetSlots()
	valued := make([]valuedSlot, len(slots))
	for i, slot := range slots {
		spec := slot.Item.GetSpec()
		stackValue := spec.Value * slot.Count
		var displayName string
		if slot.Count > 1 {
			displayName = fmt.Sprintf("a stack of %d %s", slot.Count, slot.Item.Name())
		} else {
			displayName = slot.Item.DisplayName()
		}
		valued[i] = valuedSlot{idx: i, value: stackValue, name: displayName}
	}
	sort.Slice(valued, func(a, b int) bool {
		return valued[a].value < valued[b].value
	})

	// Forfeit whole slots until shortfall is covered (no partial-stack peeling)
	forfeited := []string{}
	removeIdxs := map[int]bool{}
	goldCovered := 0

	for _, vs := range valued {
		if goldCovered >= shortfall {
			break
		}
		forfeited = append(forfeited, vs.name)
		removeIdxs[vs.idx] = true
		goldCovered += feePerSlot // each forfeited slot covers 1 fee unit
	}

	// Remove forfeited slots in reverse order to preserve indices
	for i := len(slots) - 1; i >= 0; i-- {
		if removeIdxs[i] {
			u.ItemStorage.RemoveSlot(i)
		}
	}

	// Build the item list string for the inbox notice
	itemList := ""
	for i, name := range forfeited {
		if i > 0 {
			itemList += ", "
		}
		itemList += name
	}

	remaining := u.ItemStorage.SlotCount()
	msg := fmt.Sprintf(
		"Thornwall Bank Notice: Insufficient funds for storage "+
			"fees. The following were forfeited: %s. Your "+
			"remaining %d slot(s) are secure.",
		itemList, remaining)
	u.Inbox.Add(users.Message{
		FromName: "Thornwall Bank",
		Message:  msg,
		DateSent: time.Now(),
	})

	u.Character.StorageFeeLastMonth = currentMonth
}
