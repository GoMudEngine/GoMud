// Package health captures and scores point-in-time snapshots of the
// NPC economy (shops, caravans, foragers). Snapshots persist as YAML
// and feed the /admin/economy/ dashboard.
package health

// Snapshot is the single payload written hourly to disk. Both yaml
// (disk) and json (web API) tags are present — the dashboard JS
// expects lowercase JSON field names.
type Snapshot struct {
	Timestamp   string `yaml:"timestamp"                 json:"timestamp"`
	UnixTs      int64  `yaml:"unix_ts"                   json:"unix_ts"`
	Round       uint64 `yaml:"round"                     json:"round"`
	Manual      bool   `yaml:"manual"                    json:"manual"`
	ManualLabel string `yaml:"manual_label,omitempty"    json:"manual_label,omitempty"`

	Shops    []ShopSnapshot    `yaml:"shops"    json:"shops"`
	Caravans []CaravanSnapshot `yaml:"caravans" json:"caravans"`
	Foragers []ForagerSnapshot `yaml:"foragers" json:"foragers"`
}

// ShopSnapshot captures one merchant's economic state.
type ShopSnapshot struct {
	Zone             string          `yaml:"zone"               json:"zone"`
	MobId            int             `yaml:"mob_id"             json:"mob_id"`
	RoomId           int             `yaml:"room_id"            json:"room_id"`
	Name             string          `yaml:"name"               json:"name"`
	CraftSupport     string          `yaml:"craft_support"      json:"craft_support"`
	Gold             int             `yaml:"gold"               json:"gold"`
	StartingGold     int             `yaml:"starting_gold"      json:"starting_gold"`
	LastRestockRound uint64          `yaml:"last_restock_round" json:"last_restock_round"`
	Stock            []StockSnapshot `yaml:"stock"              json:"stock"`
}

// StockSnapshot is a per-item entry. Bucket comes from economy.BucketFor().
type StockSnapshot struct {
	ItemId     int    `yaml:"item_id"     json:"item_id"`
	Bucket     string `yaml:"bucket"      json:"bucket"`
	Current    int    `yaml:"current"     json:"current"`
	Max        int    `yaml:"max"         json:"max"`
	RestockQty int    `yaml:"restock_qty" json:"restock_qty"`
}

// CaravanSnapshot captures one caravan-leader instance + its
// co-located wagon's cargo. CargoWeight and CargoCapacity are both
// in pounds — carry weight is what actually limits the wagon, so the
// dashboard's "is the wagon filling up?" question reads honestly as
// a weight ratio. CargoByBucket sums per-item weights by bucket.
type CaravanSnapshot struct {
	InstId            int            `yaml:"inst_id"             json:"inst_id"`
	Name              string         `yaml:"name"                json:"name"`
	State             string         `yaml:"state"               json:"state"`
	StateEnteredRound uint64         `yaml:"state_entered_round" json:"state_entered_round"`
	RoomId            int            `yaml:"room_id"             json:"room_id"`
	CargoWeight       int            `yaml:"cargo_weight"        json:"cargo_weight"`   // pounds
	CargoCapacity     int            `yaml:"cargo_capacity"      json:"cargo_capacity"` // pounds
	CargoByBucket     map[string]int `yaml:"cargo_by_bucket"     json:"cargo_by_bucket"`
}

// ForagerSnapshot captures one forager NPC's state + backpack
// composition. CargoWeight and CargoCapacity are pounds (same
// convention as CaravanSnapshot).
type ForagerSnapshot struct {
	InstId            int            `yaml:"inst_id"             json:"inst_id"`
	Name              string         `yaml:"name"                json:"name"`
	Territory         string         `yaml:"territory"           json:"territory"`
	State             string         `yaml:"state"               json:"state"`
	StateEnteredRound uint64         `yaml:"state_entered_round" json:"state_entered_round"`
	RoomId            int            `yaml:"room_id"             json:"room_id"`
	CargoWeight       int            `yaml:"cargo_weight"        json:"cargo_weight"`   // pounds
	CargoCapacity     int            `yaml:"cargo_capacity"      json:"cargo_capacity"` // pounds
	CargoByBucket     map[string]int `yaml:"cargo_by_bucket"     json:"cargo_by_bucket"`
}
