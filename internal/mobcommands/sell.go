package mobcommands

import (
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// Sell is the mob-side entry into actions.Sell. Mobs don't read text, so an
// empty request is a silent no-op. "sell all" (no item name) maps to the
// inventory-sweep mode; the planners (wealth-gold, upgrade-gear) rely on this.
func Sell(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {
	if rest == "" {
		return true, nil
	}
	actor := &actions.MobActor{Mob: mob, Room: room}

	lower := strings.ToLower(strings.TrimSpace(rest))
	if lower == "all" {
		actions.Sell(actor, actions.SellOptions{SellAllSellable: true})
		return true, nil
	}

	// "sell N <item>" / "sell all <item>" / "sell <item>"
	parts := strings.SplitN(strings.TrimSpace(rest), " ", 2)
	opts := actions.SellOptions{Quantity: 1, ItemName: rest}
	if len(parts) == 2 {
		if strings.ToLower(parts[0]) == "all" {
			opts.Quantity = actions.UnlimitedSell
			opts.ItemName = parts[1]
		} else if n, err := strconv.Atoi(parts[0]); err == nil && n >= 1 {
			opts.Quantity = n
			opts.ItemName = parts[1]
		}
	}
	actions.Sell(actor, opts)
	return true, nil
}
