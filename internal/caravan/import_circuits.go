package caravan

// ImportCircuit describes a self-perpetuating, looping supply runner whose
// cargo originates from a stationary import source (sea import → warehouse),
// with NO traveling inter-zone wagon. Distinct from the legacy
// wagon-triggered oneshot circuits in runner_completion_listener.go.
type ImportCircuit struct {
	PatrolId        string   // the runner's looping (strict) patrol id
	RunnerMobId     int      // who runs it (Dobb = 9304)
	DepotEvent      string   // ArrivalEvent tag at the load stop
	VendorEvent     string   // ArrivalEvent tag at delivery stops
	DeliveryBuckets []string // economy buckets delivered to vendors
	ImportItems     []int    // the sea-import manifest (item IDs loaded at the depot)
	LoadCap         int      // max items carried per loop (bounds inventory growth)
}

// importCircuits is the registry of looping import circuits, keyed by patrol id.
var importCircuits = map[string]ImportCircuit{
	"np_docks_runner_circuit": {
		PatrolId:    "np_docks_runner_circuit",
		RunnerMobId: 9304, // Dobb
		DepotEvent:  "np_runner_depot",
		VendorEvent: "np_runner_vendor",
		// Must cover EVERY bucket the Crafting vendors sell from — in a
		// CaravanServedZone the ticker restock is suppressed, so any vendor item
		// whose bucket is not delivered here would starve. Steel ingot (40018) is
		// classified "thornwall"; the tag is internal/invisible and the runner
		// only ever touches its own Crafting circuit, so including it is safe.
		DeliveryBuckets: []string{"base", "overlap", "thornwall"},
		// The full sea-import manifest = the union of every Crafting vendor's
		// shop items, so the runner can refill all of them:
		ImportItems: []int{
			40001, // iron ingot (Halvard)
			40018, // steel ingot (Halvard)
			40019, // chain link (Halvard)
			40020, // coal dust (Halvard)
			40006, // glass vial (Vesna/Edda/Orin)
			40004, // healer's root (Vesna)
			40005, // bitter thistle (Vesna)
			40012, // thread spool (Nessa)
			40013, // bone needle (Nessa)
			40007, // cloth strip (Nessa)
			40002, // leather strip (Corwin)
		},
		LoadCap: 24,
	},
}

// ImportCircuitFor returns the import circuit for a patrol id, if registered.
func ImportCircuitFor(patrolId string) (ImportCircuit, bool) {
	c, ok := importCircuits[patrolId]
	return c, ok
}
