package caravan

import (
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// TransferCargoToRunner moves all items from the wagon's inventory
// whose bucket (per economy.BucketFor) is in the outboundBuckets list
// to the runner's inventory. Returns the number of items moved.
// Used at depot arrival in the caravan arrival listener.
//
// Items the runner can't store (carry cap exceeded) stop the transfer
// early — leftover items stay on the wagon and the caller can re-run
// later or accept the partial transfer. Chunk 3.8.
func TransferCargoToRunner(wagon, runner *mobs.Mob, outboundBuckets []string) int {
	if wagon == nil || runner == nil || len(outboundBuckets) == 0 {
		return 0
	}
	moved := 0
	// Iterate in reverse so RemoveItem is index-safe.
	for i := len(wagon.Character.Items) - 1; i >= 0; i-- {
		item := wagon.Character.Items[i]
		bucket := economy.BucketFor(item.ItemId)
		if bucket == "" || !slices.Contains(outboundBuckets, bucket) {
			continue
		}
		if !runner.Character.StoreItem(item) {
			break // runner at carry cap; stop transferring
		}
		wagon.Character.RemoveItem(item)
		moved++
	}
	return moved
}

// TransferAllCargoBack moves every item from the runner's inventory
// back to the wagon. No bucket filtering — what didn't sell goes home.
// Called on PatrolCompleted by the runner-completion listener. Returns
// the number of items moved. Chunk 3.8.
func TransferAllCargoBack(runner, wagon *mobs.Mob) int {
	if runner == nil || wagon == nil {
		return 0
	}
	moved := 0
	for i := len(runner.Character.Items) - 1; i >= 0; i-- {
		item := runner.Character.Items[i]
		if !wagon.Character.StoreItem(item) {
			break // wagon at carry cap; stop transferring
		}
		runner.Character.RemoveItem(item)
		moved++
	}
	return moved
}
