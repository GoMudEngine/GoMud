package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// planStorageForfeiture holds the selection + floor decision. Testing it
// directly keeps these cases deterministic and free of the global config
// (StorageFeePerItem is 0 in the test binary, which short-circuits the caller).

func TestPlanStorageForfeiture_SeizesHighValue_DisposesLowValue(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5001, Name: "gilded blade", Type: items.Weapon, Value: 1000})
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5002, Name: "twig", Type: items.Object, IsComponent: true, Value: 10})

	slots := []users.StorageSlot{
		{Item: items.Item{ItemId: 5001}, Count: 1}, // 1000 >= 250 -> seize
		{Item: items.Item{ItemId: 5002}, Count: 1}, // 10   < 250 -> dispose
	}
	// shortfall 2, feePerSlot 1 -> both cheapest slots selected.
	seize, dispose := planStorageForfeiture(slots, 2, 1, 250)

	require.Len(t, seize, 1, "only the >=250 slot is auctioned")
	assert.Equal(t, 5001, seize[0].item.ItemId)
	assert.Equal(t, 1, seize[0].count)
	assert.Equal(t, 1, seize[0].owed, "lien == feePerSlot")
	require.Len(t, dispose, 1)
	assert.Equal(t, 5002, slots[dispose[0].slotIdx].Item.ItemId)
}

func TestPlanStorageForfeiture_SeizesAggregatePile(t *testing.T) {
	// Each unit 50g (below floor) but a stack of 6 aggregates to 300g (>= floor).
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5003, Name: "silk bolt", Type: items.Object, IsComponent: true, Value: 50})

	slots := []users.StorageSlot{
		{Item: items.Item{ItemId: 5003}, Count: 6}, // 300 >= 250 -> seize whole stack
	}
	seize, dispose := planStorageForfeiture(slots, 1, 1, 250)

	require.Len(t, seize, 1, "the aggregate 300g pile is auctioned as one lot")
	assert.Equal(t, 5003, seize[0].item.ItemId)
	assert.Equal(t, 6, seize[0].count, "whole stack carried")
	assert.Len(t, dispose, 0)
}

func TestPlanStorageForfeiture_FloorDisposesLowAggregate(t *testing.T) {
	// Many cheap units whose aggregate is still below the floor -> disposed.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 5004, Name: "pebble", Type: items.Object, IsComponent: true, Value: 5})

	slots := []users.StorageSlot{
		{Item: items.Item{ItemId: 5004}, Count: 10}, // 50 < 250
	}
	seize, dispose := planStorageForfeiture(slots, 1, 1, 250)

	assert.Len(t, seize, 0, "sub-floor aggregate is never auctioned")
	require.Len(t, dispose, 1)
}
