package shops

import "sync"

// enchantReserve is the global pool of enchanting mats produced by alchemy-
// vendor potion decay (the analog of the aggregated forager chests, but
// virtual). In-memory v1 — it refills from ongoing decay, so a restart just
// re-accumulates. Enchanters draw from it neediest-gap-first on their idle tick.
var (
	enchantReserveMu sync.Mutex
	enchantReserve   = map[int]int{}
)

// AddToReserve adds qty of matItemId to the global reserve. No-op for zero id/qty.
func AddToReserve(matItemId, qty int) {
	if matItemId <= 0 || qty <= 0 {
		return
	}
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	enchantReserve[matItemId] += qty
}

// ReservePool returns a copy of the current reserve (safe to mutate / iterate).
func ReservePool() map[int]int {
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	out := make(map[int]int, len(enchantReserve))
	for id, n := range enchantReserve {
		out[id] = n
	}
	return out
}

// DrainReserve removes up to qty of matItemId. Safe if qty exceeds the balance.
func DrainReserve(matItemId, qty int) {
	if matItemId <= 0 || qty <= 0 {
		return
	}
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	if enchantReserve[matItemId] <= qty {
		delete(enchantReserve, matItemId)
	} else {
		enchantReserve[matItemId] -= qty
	}
}

// ResetEnchantReserveForTest clears the reserve (test helper).
func ResetEnchantReserveForTest() {
	enchantReserveMu.Lock()
	defer enchantReserveMu.Unlock()
	enchantReserve = map[int]int{}
}
