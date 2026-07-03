// Package warehouse implements the Stage 3 city warehouse buffer pools:
// persistent, backend-only inventory that soaks up end-of-circuit carrier
// surplus and a slow ambient accrual, to be spent by Stage 4 drawdown.
// Spec: docs/superpowers/specs/2026-07-03-ferry-system-design.md.
// Player access: NONE, forever. Only caravans/runners/factors touch this.
package warehouse

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy"
)

// City is a warehouse city's static config.
type City struct {
	Zone         string // display zone name (rooms' Zone field)
	AccrualItems []int  // ambient-accrual seed list (all must be bucketed)
}

// cities is the registry (Go-map precedent: caravan importCircuits).
var cities = map[string]City{
	"New Plymouth Docks": {
		Zone: "New Plymouth Docks",
		// base/thornwall staples the capital "produces" off-screen.
		AccrualItems: []int{40001, 40003, 40006, 40012, 40018, 40019, 40021},
	},
	"The Confluence": {
		Zone: "The Confluence",
		// river goods (confluence bucket, Stage 2).
		AccrualItems: []int{40123, 40124, 40125, 40126},
	},
}

// Entry is one stocked item.
type Entry struct {
	ItemId  int `yaml:"item_id"`
	Current int `yaml:"current"`
}

// Warehouse is one city's persistent pool + counters.
type Warehouse struct {
	Zone          string  `yaml:"zone"`
	Stock         []Entry `yaml:"stock,omitempty"`
	CapturedCount int     `yaml:"captured_count,omitempty"` // via overflow capture
	AccruedCount  int     `yaml:"accrued_count,omitempty"`  // via ambient accrual
	DrawnCount    int     `yaml:"drawn_count,omitempty"`    // Stage 4 (0 until then)
}

var (
	mu         sync.Mutex
	warehouses = map[string]*Warehouse{}
	dirty      = map[string]bool{}

	// itemCapFn resolves the per-item cap; swapped in tests so unit tests
	// never read live config.
	itemCapFn = func() int { return int(configs.GetBalanceConfig().WarehouseItemCap) }
)

// CityFor returns the city config for a zone, if it hosts a warehouse.
func CityFor(zone string) (City, bool) {
	c, ok := cities[zone]
	return c, ok
}

// WarehouseFor returns the live pool for a zone (creating a zero-value one
// for registered cities), or nil for non-warehouse zones.
func WarehouseFor(zone string) *Warehouse {
	if _, ok := cities[zone]; !ok {
		return nil
	}
	mu.Lock()
	defer mu.Unlock()
	return getOrCreateLocked(zone)
}

func getOrCreateLocked(zone string) *Warehouse {
	if w, ok := warehouses[zone]; ok {
		return w
	}
	w := &Warehouse{Zone: zone}
	warehouses[zone] = w
	return w
}

// StockOf returns the current count of an item.
func (w *Warehouse) StockOf(itemId int) int {
	for i := range w.Stock {
		if w.Stock[i].ItemId == itemId {
			return w.Stock[i].Current
		}
	}
	return 0
}

// Deposit banks qty of an item into a city's warehouse via CAPTURE.
// Returns false (nothing banked) for non-warehouse zones or unbucketed
// items. Accepts partially when the cap truncates. Counter: CapturedCount.
func Deposit(zone string, itemId int, qty int) bool {
	return deposit(zone, itemId, qty, false)
}

// accrue is the ambient-accrual variant (counter: AccruedCount).
func accrue(zone string, itemId int, qty int) bool {
	return deposit(zone, itemId, qty, true)
}

func deposit(zone string, itemId int, qty int, isAccrual bool) bool {
	if qty <= 0 {
		return false
	}
	if _, ok := cities[zone]; !ok {
		return false
	}
	if economy.BucketFor(itemId) == `` {
		return false
	}
	mu.Lock()
	defer mu.Unlock()
	w := getOrCreateLocked(zone)

	itemCap := itemCapFn()
	cur := 0
	idx := -1
	for i := range w.Stock {
		if w.Stock[i].ItemId == itemId {
			cur, idx = w.Stock[i].Current, i
			break
		}
	}
	room := itemCap - cur
	if room <= 0 {
		return false
	}
	add := qty
	if add > room {
		add = room
	}
	if idx < 0 {
		w.Stock = append(w.Stock, Entry{ItemId: itemId, Current: add})
	} else {
		w.Stock[idx].Current += add
	}
	if isAccrual {
		w.AccruedCount += add
	} else {
		w.CapturedCount += add
	}
	dirty[zone] = true
	return true
}

// AllWarehouses returns the live pools for every registered city (creating
// zero-value ones as needed). Order is not guaranteed.
func AllWarehouses() []*Warehouse {
	mu.Lock()
	defer mu.Unlock()
	out := make([]*Warehouse, 0, len(cities))
	for zone := range cities {
		out = append(out, getOrCreateLocked(zone))
	}
	return out
}

// ResetForTest clears all state and pins the item cap to 400 so unit
// tests never read live config.
func ResetForTest() {
	mu.Lock()
	defer mu.Unlock()
	warehouses = map[string]*Warehouse{}
	dirty = map[string]bool{}
	itemCapFn = func() int { return 400 }
}
