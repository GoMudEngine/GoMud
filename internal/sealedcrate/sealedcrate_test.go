package sealedcrate

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestCrate_AddRespectsCapacity(t *testing.T) {
	c := New(4038, 3) // capacity 3
	if !c.Add(items.New(40021)) {
		t.Fatalf("first Add returned false")
	}
	if !c.Add(items.New(40028)) {
		t.Fatalf("second Add returned false")
	}
	if !c.Add(items.New(40023)) {
		t.Fatalf("third Add returned false")
	}
	if c.Add(items.New(40020)) {
		t.Fatalf("fourth Add returned true; expected false (over capacity)")
	}
	if got := c.Len(); got != 3 {
		t.Errorf("Len = %d, want 3", got)
	}
}

func TestCrate_DrainAll(t *testing.T) {
	c := New(4038, 100)
	c.Add(items.New(40021))
	c.Add(items.New(40028))
	drained := c.DrainAll()
	if len(drained) != 2 {
		t.Errorf("DrainAll returned %d items, want 2", len(drained))
	}
	if c.Len() != 0 {
		t.Errorf("Len after DrainAll = %d, want 0", c.Len())
	}
}

func TestCrate_RoomIdAndCapacity(t *testing.T) {
	c := New(4038, 2000)
	if c.RoomId() != 4038 {
		t.Errorf("RoomId = %d, want 4038", c.RoomId())
	}
	if c.Capacity() != 2000 {
		t.Errorf("Capacity = %d, want 2000", c.Capacity())
	}
}

func TestCrate_SnapshotIsIndependent(t *testing.T) {
	c := New(4038, 100)
	c.Add(items.New(40021))
	c.Add(items.New(40028))

	snap := c.Snapshot()
	if len(snap) != 2 {
		t.Fatalf("Snapshot len = %d, want 2", len(snap))
	}
	// Mutating the returned slice must not affect the crate.
	snap[0] = items.Item{}
	if c.Len() != 2 {
		t.Errorf("crate Len after snap mutation = %d, want 2", c.Len())
	}
	if c.Snapshot()[0].ItemId != 40021 {
		t.Errorf("crate item 0 was mutated via snap")
	}
}

func TestCrate_SetItemsForLoad(t *testing.T) {
	c := New(4038, 100)
	c.Add(items.New(40021)) // pre-load state, will be replaced
	c.SetItemsForLoad([]items.Item{
		items.New(40050),
		items.New(40051),
		items.New(40053),
	})
	if c.Len() != 3 {
		t.Errorf("Len after SetItemsForLoad = %d, want 3", c.Len())
	}
	drained := c.DrainAll()
	if len(drained) != 3 || drained[0].ItemId != 40050 {
		t.Errorf("DrainAll after SetItemsForLoad mismatch: %+v", drained)
	}
}
