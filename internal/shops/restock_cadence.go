// Package shops — restock_cadence.go
package shops

import "github.com/GoMudEngine/GoMud/internal/configs"

// RestockCadenceHours returns the configured restock period for the
// given rarity tier, in game-time hours. Returns 0 for unrecognized
// tiers — callers treat 0 as "no scheduled restock".
func RestockCadenceHours(b *configs.Balance, rarityTier int) int {
	switch rarityTier {
	case 50:
		return int(b.RestockCadenceTier50Hours)
	case 40:
		return int(b.RestockCadenceTier40Hours)
	case 30:
		return int(b.RestockCadenceTier30Hours)
	case 20:
		return int(b.RestockCadenceTier20Hours)
	case 10:
		return int(b.RestockCadenceTier10Days) * 24
	default:
		return 0
	}
}
