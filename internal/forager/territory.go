package forager

// ForagerKind identifies one of the three Stage 3.1 foragers.
type ForagerKind int

const (
	KindMarsh ForagerKind = iota
	KindSteppe
	KindFernway
)

// ForagerProfile holds the static data for one forager — territory,
// prey whitelist, vendor visit list, supply buckets, anchor room.
//
// Data is keyed by mob ID so the btree can resolve at runtime via
// the leader's MobId.
type ForagerProfile struct {
	Kind           ForagerKind
	MobId          int
	Name           string
	SanctuaryRoom  int    // fold-recall anchor
	TerritoryRooms []int  // wandering range
	PreyWhitelist  []int  // mobIds of legal engagement targets
	VendorRooms    []int  // delivery route (Marsh + Steppe). Empty for Fernway.
	MeetingRoom    int    // Fernway only: where caravan handoff happens. 0 for Marsh + Steppe.
	Buckets        []string // supply buckets this forager fills
}

var profiles = map[int]*ForagerProfile{
	371: { // Marsh Forager (Vella)
		Kind:           KindMarsh,
		MobId:          371,
		Name:           "Vella",
		SanctuaryRoom:  4123,
		TerritoryRooms: []int{4177, 4178, 4179, 4180, 4181, 4182, 4183, 4184, 4185, 4186, 4187, 4188, 4189, 4190, 4191, 4192, 4193, 4194, 4195, 4196},
		PreyWhitelist:  []int{367 /*marsh rat*/, 368 /*dragonfly swarm*/},
		VendorRooms:    []int{4102, 4103, 4105, 4106, 4125, 4126, 4135, 4143},
		Buckets:        []string{"stillwater", "base", "overlap"},
	},
	243: { // Steppe Forager (Halix)
		Kind:           KindSteppe,
		MobId:          243,
		Name:           "Halix",
		SanctuaryRoom:  468,
		TerritoryRooms: []int{3000, 3001, 3002, 3003, 3005, 3006, 3015, 3019, 3020, 3021, 3022, 3023, 3024, 3025, 3026, 3027, 3028, 3029},
		PreyWhitelist:  []int{200 /*steppe rat*/, 201 /*dust crow*/, 213 /*dust hare*/, 234 /*ground squirrel*/, 214 /*sage grouse*/, 231 /*tumble beetle*/},
		VendorRooms:    []int{464, 470, 471, 475, 480, 481, 482, 483, 507},
		Buckets:        []string{"base", "overlap"},
	},
	366: { // Fernway Forager (Kessa)
		Kind:           KindFernway,
		MobId:          366,
		Name:           "Kessa",
		SanctuaryRoom:  4197, // Forager's Camp (created in Task 14)
		TerritoryRooms: []int{4157, 4158, 4159, 4160, 4161, 4162, 4163, 4164, 4165, 4166, 4167, 4168, 4169, 4170, 4171, 4172, 4173, 4174, 4175, 4176},
		PreyWhitelist:  []int{360 /*wild hare*/, 362 /*honey bees*/},
		VendorRooms:    nil,
		MeetingRoom:    4038,
		Buckets:        []string{"fernway"},
	},
}

// ProfileFor returns the static profile for a forager by mob ID, or
// nil if the mob ID is not a registered forager.
func ProfileFor(mobId int) *ForagerProfile {
	return profiles[mobId]
}

// AllProfiles returns every registered forager. Stable order by Kind
// (Marsh, Steppe, Fernway) so callers that need determinism (tests,
// CLI) get consistent output.
func AllProfiles() []*ForagerProfile {
	out := make([]*ForagerProfile, 0, len(profiles))
	for _, k := range []int{371, 243, 366} {
		if p := profiles[k]; p != nil {
			out = append(out, p)
		}
	}
	return out
}
