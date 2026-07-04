package items

import "testing"

func TestItemProcValidation(t *testing.T) {
	spec := &ItemSpec{
		ItemId: 999901, Name: "test proc item", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_hit", Chance: 25, Effect: "lifesteal", Params: map[string]float64{"ratio": 0.25}}},
	}
	if err := spec.Validate(); err != nil {
		t.Fatalf("valid proc rejected: %v", err)
	}

	bad := &ItemSpec{
		ItemId: 999902, Name: "bad trigger", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_sneeze", Chance: 25, Effect: "lifesteal"}},
	}
	if err := bad.Validate(); err == nil {
		t.Fatal("invalid trigger accepted")
	}

	badEffect := &ItemSpec{
		ItemId: 999903, Name: "bad effect", Type: Weapon,
		Procs: []ItemProc{{Trigger: "on_hit", Chance: 25, Effect: "explode"}},
	}
	if err := badEffect.Validate(); err == nil {
		t.Fatal("invalid effect accepted")
	}

	badReserve := &ItemSpec{ItemId: 999904, Name: "bad reserve", Type: Weapon, ReserveHealthPct: 1.5}
	if err := badReserve.Validate(); err == nil {
		t.Fatal("reserve pct > 1 accepted")
	}
}

func TestProcsFor(t *testing.T) {
	spec := &ItemSpec{Procs: []ItemProc{
		{Trigger: "on_hit", Chance: 100, Effect: "lifesteal"},
		{Trigger: "on_block", Chance: 10, Effect: "aoe_stun"},
	}}
	if got := spec.ProcsFor("on_hit"); len(got) != 1 || got[0].Effect != "lifesteal" {
		t.Fatalf("ProcsFor(on_hit) = %+v", got)
	}
	if got := spec.ProcsFor("on_kill"); len(got) != 0 {
		t.Fatalf("expected empty, got %+v", got)
	}
}
