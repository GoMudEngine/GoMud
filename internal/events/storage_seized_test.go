package events

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestStorageItemSeized_Type(t *testing.T) {
	var e Event = StorageItemSeized{
		UserId: 7,
		Item:   items.Item{ItemId: 42},
		Count:  3,
		Owed:   1,
	}
	if e.Type() != "StorageItemSeized" {
		t.Errorf("Type() = %q, want StorageItemSeized", e.Type())
	}
}
