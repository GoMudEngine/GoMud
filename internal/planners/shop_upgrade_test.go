package planners

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/itemvalue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func TestScanZoneUpgrades_NilMob(t *testing.T) {
	if _, ok := scanZoneUpgrades(nil, itemvalue.WeightProfile{}, 100, true, 1.0); ok {
		t.Errorf("expected ok=false for nil mob")
	}
}

func TestScanZoneUpgrades_NoShopsLoaded(t *testing.T) {
	mob := &mobs.Mob{}
	mob.Character.Zone = "stillwater"
	if _, ok := scanZoneUpgrades(mob, itemvalue.WeightProfile{}, 1000, true, 1.0); ok {
		t.Errorf("expected ok=false when no shops are loaded")
	}
}
