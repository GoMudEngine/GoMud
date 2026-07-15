package hooks

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
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

	// Cannot pay in full — deduct what they have, then seize cheapest stacks to
	// cover the shortfall. High-value stacks are auctioned (emitted for the
	// auctions module); sub-floor stacks are disposed outright (as before).
	available := u.Character.Bank
	u.Character.Bank = 0
	shortfall := fee - available

	minValue := int(configs.GetBalanceConfig().StorageSeizureMinValue)

	slots := u.ItemStorage.GetSlots()
	seize, dispose := planStorageForfeiture(slots, shortfall, feePerSlot, minValue)

	// Emit a seizure event per auctioned slot; collect names + slots to remove.
	removeIdxs := map[int]bool{}
	seizedNames := make([]string, 0, len(seize))
	for _, s := range seize {
		seizedNames = append(seizedNames, s.name)
		removeIdxs[s.slotIdx] = true
		events.AddToQueue(events.StorageItemSeized{
			UserId: u.UserId,
			Item:   s.item,
			Count:  s.count,
			Owed:   s.owed,
		})
	}
	disposedNames := make([]string, 0, len(dispose))
	for _, d := range dispose {
		disposedNames = append(disposedNames, d.name)
		removeIdxs[d.slotIdx] = true
	}

	// Remove selected slots in reverse index order to preserve indices.
	for i := len(slots) - 1; i >= 0; i-- {
		if removeIdxs[i] {
			u.ItemStorage.RemoveSlot(i)
		}
	}

	// Build the notice — show only the non-empty outcomes.
	notice := "Thornwall Bank Notice: Insufficient funds for storage fees."
	if len(seizedNames) > 0 {
		notice += " Seized and sent to auction to cover the debt (anything they" +
			" fetch above what you owe returns to you): " + joinNames(seizedNames) + "."
	}
	if len(disposedNames) > 0 {
		notice += " Too little value to auction and disposed: " + joinNames(disposedNames) + "."
	}
	remaining := u.ItemStorage.SlotCount()
	notice += fmt.Sprintf(" Your remaining %d slot(s) are secure.", remaining)

	u.Inbox.Add(users.Message{
		FromName: "Thornwall Bank",
		Message:  notice,
		DateSent: time.Now(),
	})

	u.Character.StorageFeeLastMonth = currentMonth
}

// plannedSeizure is one slot selected for auction (its aggregate value cleared
// the floor). plannedDisposal is one selected slot too cheap to auction.
type plannedSeizure struct {
	slotIdx int
	item    items.Item
	count   int
	owed    int
	name    string
}

type plannedDisposal struct {
	slotIdx int
	name    string
}

// planStorageForfeiture selects the cheapest slots (by aggregate value) needed
// to cover the shortfall (feePerSlot per slot), then classifies each: a slot
// whose aggregate value (spec.Value * Count) clears minValue is auctioned
// (seized), the rest are disposed. Pure — no config or event side effects — so
// the selection/floor logic is unit-testable without the global config.
func planStorageForfeiture(slots []users.StorageSlot, shortfall, feePerSlot, minValue int) (seize []plannedSeizure, dispose []plannedDisposal) {
	type valuedSlot struct {
		idx        int
		stackValue int
		name       string
	}
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
		valued[i] = valuedSlot{idx: i, stackValue: stackValue, name: displayName}
	}
	sort.Slice(valued, func(a, b int) bool {
		return valued[a].stackValue < valued[b].stackValue
	})

	goldCovered := 0
	for _, vs := range valued {
		if goldCovered >= shortfall {
			break
		}
		goldCovered += feePerSlot // each selected slot covers one fee unit
		slot := slots[vs.idx]
		if vs.stackValue < minValue {
			dispose = append(dispose, plannedDisposal{slotIdx: vs.idx, name: vs.name})
			continue
		}
		seize = append(seize, plannedSeizure{
			slotIdx: vs.idx,
			item:    slot.Item,
			count:   slot.Count,
			owed:    feePerSlot,
			name:    vs.name,
		})
	}
	return seize, dispose
}

// joinNames renders a comma-separated list of item names for a bank notice.
func joinNames(names []string) string {
	out := ""
	for i, n := range names {
		if i > 0 {
			out += ", "
		}
		out += n
	}
	return out
}
