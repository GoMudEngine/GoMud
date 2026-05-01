package health

import (
	"testing"

	"gopkg.in/yaml.v2"
)

func TestSnapshot_YAMLRoundTrip(t *testing.T) {
	in := Snapshot{
		Timestamp:   "2026-05-01T13:00:00Z",
		UnixTs:      1746104400,
		Round:       12345,
		Manual:      true,
		ManualLabel: "pre stage-3.4",
		Shops: []ShopSnapshot{
			{
				Zone: "stillwater", MobId: 341, RoomId: 4105,
				Name: "Storekeeper Wulf", CraftSupport: "general",
				Gold: 487, StartingGold: 500, LastRestockRound: 12000,
				Stock: []StockSnapshot{
					{ItemId: 40001, Bucket: "base", Current: 8, Max: 20, RestockQty: 5},
				},
			},
		},
		Caravans: []CaravanSnapshot{
			{
				InstId: 42, Name: "Caravan Master Borric",
				State: "outbound_transit", StateEnteredRound: 12100, RoomId: 1500,
				CargoCount: 14, CargoCapacity: 30,
				CargoByBucket: map[string]int{"base": 5, "stillwater": 9},
			},
		},
		Foragers: []ForagerSnapshot{
			{
				InstId: 88, Name: "Storekeeper Wulf",
				Territory: "stillwater_marsh",
				State: "foraging", StateEnteredRound: 12200, RoomId: 4520,
				CargoCount: 6, CargoCapacity: 12,
				CargoByBucket: map[string]int{"stillwater": 6},
			},
		},
	}

	bytes, err := yaml.Marshal(in)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var out Snapshot
	if err := yaml.Unmarshal(bytes, &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if out.UnixTs != in.UnixTs {
		t.Errorf("UnixTs: got %d, want %d", out.UnixTs, in.UnixTs)
	}
	if len(out.Shops) != 1 || out.Shops[0].MobId != 341 {
		t.Errorf("Shops round-trip mismatch: got %+v", out.Shops)
	}
	if len(out.Caravans) != 1 || out.Caravans[0].State != "outbound_transit" {
		t.Errorf("Caravans round-trip mismatch: got %+v", out.Caravans)
	}
	if got := out.Foragers[0].CargoByBucket["stillwater"]; got != 6 {
		t.Errorf("Foragers cargo bucket: got %d, want 6", got)
	}
}
