package crafting

import (
	"github.com/GoMudEngine/GoMud/internal/items"
)

// corpseSalvageEntry pairs a mob group key with the salvage returns
// players recover when they salvage a corpse from that group.
type corpseSalvageEntry struct {
	Group   string
	Returns []items.SalvageReturn
}

// corpseSalvageTable is the static lookup table for corpse salvage.
// Order matters: LookupCorpseSalvage returns the first matching entry.
// Future expansion (bird, insect, chrysalis, etc.) just appends here.
var corpseSalvageTable = []corpseSalvageEntry{
	{
		Group: "animal",
		Returns: []items.SalvageReturn{
			{ItemTag: "leather-strip", Quantity: 2},
			{ItemTag: "sinew", Quantity: 1},
		},
	},
	{
		Group: "humanoid",
		Returns: []items.SalvageReturn{
			{ItemTag: "cloth-strip", Quantity: 2},
			{ItemTag: "leather-strip", Quantity: 1},
		},
	},
}

// LookupCorpseSalvage returns the salvage returns for the first matching
// group in the table, or nil if no group matches. The mob's full groups
// slice is passed in; iteration order is the table's declaration order.
func LookupCorpseSalvage(groups []string) []items.SalvageReturn {
	for _, entry := range corpseSalvageTable {
		for _, g := range groups {
			if g == entry.Group {
				return entry.Returns
			}
		}
	}
	return nil
}
