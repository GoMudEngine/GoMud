package weather

import "testing"

func TestBuildConfig(t *testing.T) {
	vals := map[string]any{
		"Enabled":            true,
		"IncludeSecretExits": true,
		"RebuildGraphOnBoot": false,
	}
	c := buildConfig(func(k string) any { return vals[k] })

	if !c.Enabled {
		t.Error("Enabled should be true")
	}
	if !c.IncludeSecretExits {
		t.Error("IncludeSecretExits should be true")
	}
	if c.RebuildGraphOnBoot {
		t.Error("RebuildGraphOnBoot should be false")
	}
}

func TestBuildConfigDefaults(t *testing.T) {
	// A getter with no values: every knob falls back to its shipped default.
	cfg := buildConfig(func(string) any { return nil })
	if cfg.Enabled {
		t.Error("Enabled must default false when config is absent (overlay normally supplies true)")
	}
	if cfg.TickEveryGameHours != 1 || cfg.MaxActiveFronts != 8 || cfg.SpawnRateScale != 1.0 {
		t.Errorf("sim defaults wrong: %+v", cfg)
	}
	if cfg.EmoteMode != "module" || cfg.EmoteEveryRounds != 20 {
		t.Errorf("emote defaults wrong: %+v", cfg)
	}
	if !cfg.BuffsEnabled || !cfg.Persist || cfg.Seed != 0 {
		t.Errorf("buff/persist/seed defaults wrong: %+v", cfg)
	}
	if !cfg.IncludeSecretExits {
		t.Error("IncludeSecretExits must default true")
	}
}

func TestBuildConfigCoercionAndClamps(t *testing.T) {
	vals := map[string]any{
		"Enabled":            true,
		"Seed":               7,          // yaml int
		"TickEveryGameHours": 0,          // clamps to 1
		"MaxActiveFronts":    "12",       // string coercion
		"SpawnRateScale":     2.5,        // float
		"EmoteMode":          "TAG-ONLY", // case-insensitive
		"EmoteEveryRounds":   2,          // clamps to 5
		"BuffsEnabled":       false,
		"Persist":            false,
	}
	cfg := buildConfig(func(k string) any { return vals[k] })
	if cfg.Seed != 7 || cfg.TickEveryGameHours != 1 || cfg.MaxActiveFronts != 12 {
		t.Errorf("coercion wrong: %+v", cfg)
	}
	if cfg.SpawnRateScale != 2.5 || cfg.EmoteMode != "tag-only" || cfg.EmoteEveryRounds != 5 {
		t.Errorf("clamps wrong: %+v", cfg)
	}
	if cfg.BuffsEnabled || cfg.Persist {
		t.Errorf("bool overrides ignored: %+v", cfg)
	}
}

func TestBuildConfigBadEmoteModeFallsBack(t *testing.T) {
	cfg := buildConfig(func(k string) any {
		if k == "EmoteMode" {
			return "shouty"
		}
		return nil
	})
	if cfg.EmoteMode != "module" {
		t.Errorf("invalid EmoteMode should fall back to module: %q", cfg.EmoteMode)
	}
}

func TestSimConfig(t *testing.T) {
	cfg := buildConfig(func(string) any { return nil })
	cfg.MaxActiveFronts = 3
	cfg.SpawnRateScale = 0
	sc := cfg.simConfig()
	if sc.MaxActiveFronts != 3 {
		t.Errorf("front budget not applied: %+v", sc)
	}
	if sc.SpawnChance != 0 {
		t.Errorf("scale 0 should zero the spawn chance: %v", sc.SpawnChance)
	}

	cfg.MaxActiveFronts = 0 // bypasses buildConfig's clamp; simConfig must still pass it through
	cfg.SpawnRateScale = 10
	sc = cfg.simConfig()
	if sc.SpawnChance != 1 {
		t.Errorf("spawn chance must cap at 1: %v", sc.SpawnChance)
	}
}

func TestBuildConfigClampsFrontBudgetAndSeed(t *testing.T) {
	cfg := buildConfig(func(k string) any {
		return map[string]any{"MaxActiveFronts": 0, "Seed": -1}[k]
	})
	if cfg.MaxActiveFronts != 1 {
		t.Errorf("MaxActiveFronts 0 must clamp to 1: %d", cfg.MaxActiveFronts)
	}
	if cfg.Seed != 0 {
		t.Errorf("negative Seed must clamp to 0 (derive-from-world): %d", cfg.Seed)
	}
}
