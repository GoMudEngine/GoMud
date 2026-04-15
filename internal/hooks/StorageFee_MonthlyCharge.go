package hooks

import (
	"fmt"
	"math"
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
func ChargeStorageFee(u *users.UserRecord, currentMonth int) {
	if u.ItemStorage.Items == nil || len(u.ItemStorage.Items) == 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	if u.Character.StorageFeeLastMonth >= currentMonth {
		return // Already charged this month
	}

	feePerItem := int(configs.GetBalanceConfig().StorageFeePerItem)
	if feePerItem <= 0 {
		u.Character.StorageFeeLastMonth = currentMonth
		return
	}

	itemCount := len(u.ItemStorage.Items)
	fee := itemCount * feePerItem

	if u.Character.Bank >= fee {
		// Can pay in full
		u.Character.Bank -= fee
		u.Character.StorageFeeLastMonth = currentMonth

		// Warn if they won't be able to pay next month
		if u.Character.Bank < fee {
			msg := fmt.Sprintf(
				"Thornwall Bank Notice: Your monthly storage fee of "+
					"%dg has been collected. You have %dg remaining "+
					"in your account. Next month's fee will be %dg "+
					"-- please deposit additional gold or retrieve "+
					"items to avoid forfeiture.",
				fee, u.Character.Bank, fee)
			u.Inbox.Add(users.Message{
				FromName: "Thornwall Bank",
				Message:  msg,
				DateSent: time.Now(),
			})
		}
		return
	}

	// Cannot pay in full — deduct what they have, forfeit cheapest items
	available := u.Character.Bank
	u.Character.Bank = 0
	shortfall := fee - available

	// How many items must be forfeited to cover the shortfall
	itemsToRemove := int(math.Ceil(float64(shortfall) / float64(feePerItem)))
	if itemsToRemove > len(u.ItemStorage.Items) {
		itemsToRemove = len(u.ItemStorage.Items)
	}

	// Sort by gold value ascending (cheapest first)
	type valuedItem struct {
		idx   int
		value int
		item  items.Item
	}
	valued := make([]valuedItem, len(u.ItemStorage.Items))
	for i, itm := range u.ItemStorage.Items {
		spec := itm.GetSpec()
		valued[i] = valuedItem{idx: i, value: spec.Value, item: itm}
	}
	sort.Slice(valued, func(a, b int) bool {
		return valued[a].value < valued[b].value
	})

	// Forfeit the cheapest items
	forfeited := make([]string, 0, itemsToRemove)
	removeSet := make(map[int]bool, itemsToRemove)
	for i := 0; i < itemsToRemove && i < len(valued); i++ {
		forfeited = append(forfeited, valued[i].item.DisplayName())
		removeSet[valued[i].idx] = true
	}

	// Rebuild storage without forfeited items
	kept := make([]items.Item, 0, len(u.ItemStorage.Items)-len(removeSet))
	for i, itm := range u.ItemStorage.Items {
		if !removeSet[i] {
			kept = append(kept, itm)
		}
	}
	u.ItemStorage.Items = kept

	// Send forfeiture notice
	itemList := ""
	for i, name := range forfeited {
		if i > 0 {
			itemList += ", "
		}
		itemList += name
	}
	remaining := len(u.ItemStorage.Items)
	msg := fmt.Sprintf(
		"Thornwall Bank Notice: Insufficient funds for storage "+
			"fees. The following items were forfeited: %s. Your "+
			"remaining %d items are secure.",
		itemList, remaining)
	u.Inbox.Add(users.Message{
		FromName: "Thornwall Bank",
		Message:  msg,
		DateSent: time.Now(),
	})

	u.Character.StorageFeeLastMonth = currentMonth
}
