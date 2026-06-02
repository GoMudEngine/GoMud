package actions

import (
	"math"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// UnlimitedSell is the Quantity sentinel meaning "sell every match".
const UnlimitedSell = math.MaxInt

type SellOptions struct {
	ItemName        string // ignored when SellAllSellable
	Quantity        int    // 1, N, or UnlimitedSell
	SellAllSellable bool   // mob inventory-sweep mode (every sellable item)
	MerchantName    string // optional target merchant name; "" = first willing
}

type SellStopReason int

const (
	SellStopSoldAll       SellStopReason = iota // ran out of matching items (normal)
	SellStopNoItem                              // seller never had the item
	SellStopNoMerchant                          // no willing merchant in room
	SellStopMerchantBroke                       // merchant ran out of gold (player path only)
	SellStopRejected                            // merchant declined the item type
)

type SellResult struct {
	Sold         int
	TotalGold    int
	Reason       SellStopReason
	LastItemName string
}

// Sell is the shared seller entry point for players and mobs. The seller is
// abstracted via Actor; the merchant is a shopkeeper mob resolved from the
// seller's room. Player sells draw down shop gold; mob sells credit the seller
// but leave shop gold intact (see internal/shops/context.md "two sell models").
//
// NOTE: forager.SellToVendor is a different path — a free supply handoff, not a
// sale. See internal/forager/vendor_sell.go.
func Sell(seller Actor, opts SellOptions) SellResult {
	// Implemented in a later task.
	return SellResult{Reason: SellStopNoItem}
}

// silence unused import until logic lands
var _ = items.Item{}
