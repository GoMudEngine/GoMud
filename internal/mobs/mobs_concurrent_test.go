package mobs

import (
	"sync"
	"testing"
)

// TestConcurrentInstanceAccess hammers the instance map from multiple
// goroutines to exercise the RWMutex. Must be run with `go test -race`.
func TestConcurrentInstanceAccess(t *testing.T) {
	const writers = 4
	const readers = 8
	const iters = 500

	var wg sync.WaitGroup

	// Writers create and destroy instances.
	for w := 0; w < writers; w++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				m := &Mob{InstanceId: id*1000 + i}
				m.Character.Name = "racer"
				storeInstanceForTest(m)
				removeInstanceForTest(m.InstanceId)
			}
		}(w)
	}

	// Readers iterate and read.
	for r := 0; r < readers; r++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iters; i++ {
				_ = GetAllMobInstanceIds()
				_ = GetInstance(i)
			}
		}()
	}

	wg.Wait()
}

// storeInstanceForTest / removeInstanceForTest expose the locked mutation
// paths without going through the full NewMobById / DestroyInstance flow,
// which pulls in too many dependencies for a focused race test.
func storeInstanceForTest(m *Mob) {
	mobInstancesMu.Lock()
	mobInstances[m.InstanceId] = m
	mobInstancesMu.Unlock()
}

func removeInstanceForTest(id int) {
	mobInstancesMu.Lock()
	delete(mobInstances, id)
	mobInstancesMu.Unlock()
}
