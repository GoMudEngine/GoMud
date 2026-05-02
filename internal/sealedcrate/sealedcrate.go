// Package sealedcrate provides a player-untouchable container
// primitive used for autonomous deliveries between long-running
// system NPCs (e.g. Kessa the Fernway forager and the caravan).
//
// A SealedCrate is bound to a single room, persists across reboots,
// and is mutated only by code paths inside internal/behaviortree
// (forager / caravan tick functions). Player commands (get, look,
// put, picklock, open) recognize a sealed-crate noun and emit
// flavor — they never read or write the crate's items list.
package sealedcrate

import (
	"sync"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// Crate is the single sealed-crate instance for one room.
type Crate struct {
	roomId   int
	capacity int
	mu       sync.Mutex
	items    []items.Item
}

// New constructs an empty crate. Capacity 0 or negative defaults to
// 2000.
func New(roomId, capacity int) *Crate {
	if capacity <= 0 {
		capacity = 2000
	}
	return &Crate{roomId: roomId, capacity: capacity}
}

func (c *Crate) RoomId() int   { return c.roomId }
func (c *Crate) Capacity() int { return c.capacity }

func (c *Crate) Len() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.items)
}

// Add appends an item if there's room. Returns true on success.
func (c *Crate) Add(it items.Item) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.items) >= c.capacity {
		return false
	}
	c.items = append(c.items, it)
	return true
}

// DrainAll empties the crate and returns its contents.
func (c *Crate) DrainAll() []items.Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := c.items
	c.items = nil
	return out
}

// Snapshot returns a copy of the items list (read-only view, used
// for persistence and inspection).
func (c *Crate) Snapshot() []items.Item {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]items.Item, len(c.items))
	copy(out, c.items)
	return out
}

// SetItemsForLoad replaces the items list wholesale. Used only by
// the persistence loader at boot.
func (c *Crate) SetItemsForLoad(its []items.Item) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.items = its
}
