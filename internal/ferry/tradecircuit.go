package ferry

import (
	"fmt"
	"slices"

	"github.com/GoMudEngine/GoMud/internal/economy"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// TradeCircuit describes one duplex trade factor riding one ferry route:
// at each port it loads that port's export manifest, rides to the other
// port, walks that port's vendor stops delivering by bucket, returns to
// the dock, and waits for the boat home.
type TradeCircuit struct {
	RouteId             string
	FactorMobId         int
	HomePortIdx         int         // where the factor spawns/recovers (0 or 1)
	PortExports         [2][]int    // manifest loaded AT port i
	PortStops           [2][]int    // vendor rooms walked AT port i
	PortDeliveryBuckets [2][]string // buckets deliverable AT port i (the OTHER port's export buckets)
	LoadCap             int
	NewSlotMaxStock     int
}

func (c TradeCircuit) Validate() error {
	if c.RouteId == `` {
		return fmt.Errorf(`trade circuit missing RouteId`)
	}
	if c.FactorMobId <= 0 {
		return fmt.Errorf(`trade circuit %s missing FactorMobId`, c.RouteId)
	}
	if c.HomePortIdx < 0 || c.HomePortIdx > 1 {
		return fmt.Errorf(`trade circuit %s HomePortIdx must be 0 or 1`, c.RouteId)
	}
	for i := 0; i < 2; i++ {
		if len(c.PortExports[i]) == 0 {
			return fmt.Errorf(`trade circuit %s port %d has no exports`, c.RouteId, i)
		}
		if len(c.PortStops[i]) == 0 {
			return fmt.Errorf(`trade circuit %s port %d has no stops`, c.RouteId, i)
		}
	}
	if c.LoadCap <= 0 {
		return fmt.Errorf(`trade circuit %s needs positive LoadCap`, c.RouteId)
	}
	if c.NewSlotMaxStock <= 0 {
		return fmt.Errorf(`trade circuit %s needs positive NewSlotMaxStock`, c.RouteId)
	}
	return nil
}

// tradeCircuits is the registry, keyed by route id. Registered in Go (the
// importCircuits precedent in internal/caravan/import_circuits.go).
var tradeCircuits = map[string]TradeCircuit{
	"stillwater_np_packet": {
		RouteId:     "stillwater_np_packet",
		FactorMobId: 9577, // A Lakeway Factor
		HomePortIdx: 0,    // Stillwater pier 4118
		PortExports: [2][]int{
			{40057, 40058, 40059, 40056}, // Stillwater exports (stillwater bucket)
			{40018, 40019, 40006, 40021}, // NP exports (thornwall+base)
		},
		PortStops: [2][]int{
			{4105, 4106, 4125}, // Stillwater: Wulf, Brindle, Ilsa
			{5505, 5508},       // NP: Dunmar Wells, Chandler Voss
		},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"}, // NP goods arriving in Stillwater
			{"stillwater"},        // lake goods arriving in NP
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
	"stillwater_confluence_barge": {
		RouteId:     "stillwater_confluence_barge",
		FactorMobId: 9578, // A Riverway Factor
		HomePortIdx: 0,    // Stillwater pier 4118
		PortExports: [2][]int{
			{40057, 40058, 40059, 40056}, // Stillwater exports
			{40123, 40124, 40125, 40126}, // Confluence exports (confluence bucket)
		},
		PortStops: [2][]int{
			{4105, 4106, 4125}, // Stillwater
			{6110, 6128},       // Confluence: Pella, Varro
		},
		PortDeliveryBuckets: [2][]string{
			{"confluence"},
			{"stillwater"},
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
	"confluence_np_barge": {
		RouteId:     "confluence_np_barge",
		FactorMobId: 9579, // A Broadwater Factor
		HomePortIdx: 0,    // Confluence Barge Dock 6109
		PortExports: [2][]int{
			{40123, 40124, 40125, 40126}, // Confluence exports
			{40018, 40019, 40006, 40021}, // NP exports
		},
		PortStops: [2][]int{
			{6110, 6128}, // Confluence
			{5505, 5508}, // NP
		},
		PortDeliveryBuckets: [2][]string{
			{"thornwall", "base"},
			{"confluence"},
		},
		LoadCap:         12,
		NewSlotMaxStock: 6,
	},
}

// TradeCircuitFor returns the circuit for a route id, if registered.
func TradeCircuitFor(routeId string) (TradeCircuit, bool) {
	c, ok := tradeCircuits[routeId]
	return c, ok
}

// IsFactorMobId reports whether a mob TEMPLATE id belongs to a registered
// trade circuit. Used by the idle hook to exempt factors from the
// displaced-home recovery pull — the ferry controller owns factor
// movement, and a factor standing on an exitless vessel deck is not
// lost; pathto-home from there flags it stuck and the stuck-mob
// cleanup kills it (2026-07-03 playtest, BUG-1).
func IsFactorMobId(mobId int) bool {
	for _, c := range tradeCircuits {
		if c.FactorMobId == mobId {
			return true
		}
	}
	return false
}

// validateTradeCircuits runs boot-time integrity checks. Called from
// LoadDataFiles AFTER routes are loaded. Panics on failures (startup
// validator doctrine). Circuits referencing unloaded routes are an error;
// routes without circuits are fine (passenger-only lines).
func validateTradeCircuits() {
	for id, c := range tradeCircuits {
		if err := c.Validate(); err != nil {
			panic(fmt.Sprintf(`ferry trade circuit %s: %v`, id, err))
		}
		_, ok := RouteFor(c.RouteId)
		if !ok {
			// A circuit for a route with no YAML: tolerate ONLY if no routes
			// loaded at all (the data-optional skip path); otherwise panic.
			if len(routes) == 0 {
				continue
			}
			panic(fmt.Sprintf(`ferry trade circuit %s references unknown route`, id))
		}
		if mobs.GetMobSpec(mobs.MobId(c.FactorMobId)) == nil {
			panic(fmt.Sprintf(`ferry trade circuit %s: factor mob %d does not exist`, id, c.FactorMobId))
		}
		for i := 0; i < 2; i++ {
			for _, itemId := range c.PortExports[i] {
				if economy.BucketFor(itemId) == `` {
					panic(fmt.Sprintf(`ferry trade circuit %s: export item %d is unbucketed — delivery would silently skip it`, id, itemId))
				}
			}
			for _, stop := range c.PortStops[i] {
				if rooms.LoadRoom(stop) == nil {
					panic(fmt.Sprintf(`ferry trade circuit %s: stop room %d does not exist`, id, stop))
				}
			}
			// Cross-check: everything exported FROM the other port must be
			// deliverable AT this port, i.e. its bucket must appear in
			// PortDeliveryBuckets[i]. A transposed bucket cell would boot
			// clean and then silently no-op every delivery (VisitVendors
			// skips off-bucket items) — catch it here instead.
			for _, itemId := range c.PortExports[1-i] {
				bucket := economy.BucketFor(itemId)
				if !slices.Contains(c.PortDeliveryBuckets[i], bucket) {
					panic(fmt.Sprintf(
						`ferry trade circuit %s: port %d cannot deliver export item %d (bucket %q not in PortDeliveryBuckets[%d] %v) — deliveries would silently no-op`,
						id, i, itemId, bucket, i, c.PortDeliveryBuckets[i]))
				}
			}
		}
	}
}
