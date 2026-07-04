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

func TestItemSpecBoundsValidation(t *testing.T) {
	valid := func(mut func(*ItemSpec)) *ItemSpec {
		spec := &ItemSpec{ItemId: 999910, Name: "bounds test", Type: Weapon}
		mut(spec)
		return spec
	}

	tests := []struct {
		name    string
		mut     func(*ItemSpec)
		wantErr bool
	}{
		// Proc chance boundaries
		{"chance 1 valid", func(s *ItemSpec) {
			s.Procs = []ItemProc{{Trigger: "on_hit", Chance: 1, Effect: "lifesteal"}}
		}, false},
		{"chance 100 valid", func(s *ItemSpec) {
			s.Procs = []ItemProc{{Trigger: "on_hit", Chance: 100, Effect: "lifesteal"}}
		}, false},
		{"chance 0 invalid", func(s *ItemSpec) {
			s.Procs = []ItemProc{{Trigger: "on_hit", Chance: 0, Effect: "lifesteal"}}
		}, true},
		{"chance 101 invalid", func(s *ItemSpec) {
			s.Procs = []ItemProc{{Trigger: "on_hit", Chance: 101, Effect: "lifesteal"}}
		}, true},
		// Reserve pct boundaries
		{"reserve 0 valid", func(s *ItemSpec) { s.ReserveHealthPct = 0 }, false},
		{"reserve 0.99 valid", func(s *ItemSpec) { s.ReserveHealthPct = 0.99 }, false},
		{"reserve exactly 1.0 invalid", func(s *ItemSpec) { s.ReserveHealthPct = 1.0 }, true},
		{"reserve negative invalid", func(s *ItemSpec) { s.ReserveHealthPct = -0.1 }, true},
		// Hunger fields
		{"hunger rounds -1 invalid", func(s *ItemSpec) { s.HungerRounds = -1 }, true},
		{"hunger rounds positive valid", func(s *ItemSpec) { s.HungerRounds = 50 }, false},
		{"hunger drain pct 1.0 invalid", func(s *ItemSpec) { s.HungerDrainPct = 1.0 }, true},
		{"hunger drain pct negative invalid", func(s *ItemSpec) { s.HungerDrainPct = -0.05 }, true},
		{"hunger drain pct 0.02 valid", func(s *ItemSpec) { s.HungerDrainPct = 0.02 }, false},
		// Mutation tick fields
		{"mutation interval -1 invalid", func(s *ItemSpec) { s.MutationTickInterval = -1 }, true},
		{"mutation chance negative invalid (interval 0)", func(s *ItemSpec) { s.MutationTickChance = -1 }, true},
		{"mutation chance 101 invalid (interval 0)", func(s *ItemSpec) { s.MutationTickChance = 101 }, true},
		{"mutation interval set, chance 0 invalid", func(s *ItemSpec) {
			s.MutationTickInterval = 10
			s.MutationTickChance = 0
		}, true},
		{"mutation interval set, chance 1 valid", func(s *ItemSpec) {
			s.MutationTickInterval = 10
			s.MutationTickChance = 1
		}, false},
		{"mutation rarity floor 10 valid", func(s *ItemSpec) { s.MutationRarityFloor = 10 }, false},
		{"mutation rarity floor 11 invalid", func(s *ItemSpec) { s.MutationRarityFloor = 11 }, true},
		{"mutation rarity floor negative invalid", func(s *ItemSpec) { s.MutationRarityFloor = -1 }, true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := valid(tc.mut).Validate()
			if tc.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tc.wantErr && err != nil {
				t.Fatalf("expected valid, got: %v", err)
			}
		})
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
