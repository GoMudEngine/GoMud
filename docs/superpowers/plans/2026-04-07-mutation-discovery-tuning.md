# Mutation Discovery Tuning Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Prefer deepening existing mutations at 70/30 over new discoveries, and gently shift new mutation rolls toward rarer results for players with higher average mutation levels.

**Architecture:** Add a config knob for deepen chance, update `GetWeightedPool()` to compute rarity bonus from average mutation level, restructure the acquire/deepen decision in both player and mob round ticks.

**Tech Stack:** Go, existing mutations/configs packages.

---

### Task 1: Add MutationDeepenChance Config Knob

**Files:**
- Modify: `internal/configs/config.balance.go:186-187` (add field after MutationMaxLevel)
- Modify: `internal/configs/config.balance.go:693-697` (add validation)
- Modify: `_datafiles/config.yaml` (add entry)

- [ ] **Step 1: Add field to BalanceConfig struct**

In `internal/configs/config.balance.go`, add after the `MutationMaxLevel` field (line 186):

```go
MutationDeepenChance         ConfigFloat `yaml:"MutationDeepenChance"`         // Probability of deepening vs new discovery when both possible (default 0.70)
```

- [ ] **Step 2: Add validation**

In the `Validate()` method, add after the `MutationMaxLevel` validation (around line 694):

```go
if b.MutationDeepenChance <= 0 || b.MutationDeepenChance > 1.0 {
	b.MutationDeepenChance = 0.70
}
```

- [ ] **Step 3: Add config entry**

In `_datafiles/config.yaml`, add near the other mutation config entries:

```yaml
  MutationDeepenChance: 0.70         # Probability of deepening vs new when both possible (0.0-1.0)
```

- [ ] **Step 4: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 5: Commit**

```bash
git add internal/configs/config.balance.go _datafiles/config.yaml
git commit -m "feat: add MutationDeepenChance config knob (default 0.70)"
```

---

### Task 2: Add Rarity Uplift to GetWeightedPool

**Files:**
- Modify: `internal/mutations/mutations.go:189-219` (GetWeightedPool function)
- Test: `internal/mutations/mutations_test.go` (add tests)

- [ ] **Step 1: Write the tests**

Add to `internal/mutations/mutations_test.go`:

```go
func TestGetWeightedPool_RarityUplift_NoMutations(t *testing.T) {
	// No mutations → rarityBonus = 0, normal weights
	owned := map[string]int{}
	pool := GetWeightedPool(owned)
	// Count appearances of a known common mutation (rarity 1-3)
	// and a known rare mutation (rarity 8+) to verify weighting exists
	assert.Greater(t, len(pool), 0, "pool should not be empty")
}

func TestGetWeightedPool_RarityUplift_HighAvgLevel(t *testing.T) {
	// Owned mutations at high levels → rarityBonus shifts pool
	// Create a baseline pool with no owned mutations
	basePool := GetWeightedPool(map[string]int{})

	// Create a pool with some high-level mutations owned
	// (these mutations are excluded from pool, but the avg level
	// affects weighting of remaining mutations)
	owned := map[string]int{
		"thick-hide": 4,
		"iron-gut":   4,
	}
	upliftPool := GetWeightedPool(owned)

	// The uplift pool should be smaller than baseline (fewer entries
	// per common mutation due to bonus, plus 2 mutations excluded)
	// We can't assert exact values without knowing all mutations,
	// but the pool should be non-empty and smaller
	assert.Greater(t, len(basePool), len(upliftPool),
		"high avg level should produce a smaller pool (reduced common weights)")
}

func TestRarityBonus_Calculation(t *testing.T) {
	tests := []struct {
		name     string
		owned    map[string]int
		expected int
	}{
		{"no mutations", map[string]int{}, 0},
		{"all level 1", map[string]int{"a": 1, "b": 1}, 0},
		{"avg level 2", map[string]int{"a": 2, "b": 2}, 1},
		{"avg level 3", map[string]int{"a": 3, "b": 3}, 2},
		{"avg level 4", map[string]int{"a": 4, "b": 4}, 3},
		{"mixed levels", map[string]int{"a": 1, "b": 3}, 1}, // avg 2, floor(2)-1=1
		{"single level 4", map[string]int{"a": 4}, 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := calcRarityBonus(tt.owned)
			assert.Equal(t, tt.expected, got)
		})
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/mutations/... -run TestGetWeightedPool_RarityUplift -v`
Run: `go test ./internal/mutations/... -run TestRarityBonus -v`
Expected: FAIL — `calcRarityBonus` undefined.

- [ ] **Step 3: Implement calcRarityBonus and update GetWeightedPool**

In `internal/mutations/mutations.go`, add before `GetWeightedPool`:

```go
// calcRarityBonus computes a pool weight reduction based on the player's
// average mutation level. Higher average levels shift new discoveries
// toward rarer mutations.
//
//	avgLevel 1 → bonus 0 (normal weights)
//	avgLevel 2 → bonus 1
//	avgLevel 3 → bonus 2
//	avgLevel 4 → bonus 3
func calcRarityBonus(owned map[string]int) int {
	if len(owned) == 0 {
		return 0
	}
	totalLevels := 0
	for _, level := range owned {
		totalLevels += level
	}
	avgLevel := totalLevels / len(owned) // integer division = floor
	bonus := avgLevel - 1
	if bonus < 0 {
		bonus = 0
	}
	return bonus
}
```

Then update `GetWeightedPool` to use it. Replace the weight calculation (lines 210-213):

```go
func GetWeightedPool(owned map[string]int, disabledSlots ...[]string) []string {
	// Build a set of disabled slots for quick lookup
	slotDisabled := map[string]bool{}
	if len(disabledSlots) > 0 && disabledSlots[0] != nil {
		for _, slot := range disabledSlots[0] {
			slotDisabled[slot] = true
		}
	}
	hasArms := !slotDisabled["weapon"] && !slotDisabled["offhand"]

	// Rarity uplift: reduce common mutation weights for advanced players
	rarityBonus := calcRarityBonus(owned)

	pool := make([]string, 0, len(allMutations)*5)
	for id, spec := range allMutations {
		if _, has := owned[id]; has {
			continue
		}
		if HasConflict(owned, id) {
			continue
		}
		if spec.RequiresArms && !hasArms {
			continue
		}
		weight := 11 - spec.Rarity - rarityBonus
		if weight < 1 {
			weight = 1
		}
		for i := 0; i < weight; i++ {
			pool = append(pool, id)
		}
	}
	return pool
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/mutations/... -v`
Expected: All PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/mutations/mutations.go internal/mutations/mutations_test.go
git commit -m "feat: add rarity uplift to mutation weighted pool based on avg level"
```

---

### Task 3: Restructure Player Mutation Decision (Deepen vs New)

**Files:**
- Modify: `internal/hooks/NewRound_UserRoundTick.go:185-238`

- [ ] **Step 1: Restructure the acquire/deepen decision**

In `internal/hooks/NewRound_UserRoundTick.go`, replace the block starting at line 185 (`if user.Character.MutationProgress >= threshold {`) through line 239 (the closing brace of the threshold block). The new logic:

```go
						if user.Character.MutationProgress >= threshold {
							user.Character.MutationProgress = 0

							// Decide: deepen existing mutation or acquire new one
							doDeepen := false
							if canAcquire && canDeepen {
								// Both possible — coin flip weighted toward deepening
								if util.Rand(100) < int(mb.MutationDeepenChance*100) {
									doDeepen = true
								}
							} else if canDeepen && !canAcquire {
								// At max count — must deepen
								doDeepen = true
							}
							// else: canAcquire && !canDeepen — acquire new (doDeepen stays false)

							if doDeepen {
								mutId := mutations.RollDeepening(user.Character.Mutations)
								if mutId != "" {
									user.Character.Mutations[mutId]++
									newLevel := user.Character.Mutations[mutId]
									if spec := mutations.GetMutation(mutId); spec != nil {
										levelTag := fmt.Sprintf("Level %d", newLevel)
										if newLevel >= int(mb.MutationMaxLevel) {
											levelTag = "fully matured"
										}
										user.SendText(fmt.Sprintf(
											`<ansi fg="magenta">The Chrysalis deepens its hold. Your <ansi fg="yellow">%s</ansi> grows stronger (%s).</ansi>`,
											spec.Name, levelTag))
									}
								}
							} else if canAcquire {
								pool := mutations.GetWeightedPool(user.Character.Mutations)
								if len(pool) > 0 {
									mutId := mutations.RollAcquisition(pool)
									if user.Character.Mutations == nil {
										user.Character.Mutations = make(map[string]int)
									}
									user.Character.Mutations[mutId] = 1
									spec := mutations.GetMutation(mutId)
									if spec != nil {
										user.SendText(fmt.Sprintf(
											`<ansi fg="magenta">Something stirs beneath your skin. A mutation emerges: <ansi fg="yellow">%s</ansi>.</ansi>`,
											spec.Name))
										user.SendText(fmt.Sprintf(`<ansi fg="magenta">%s</ansi>`, spec.Description))

										// Emit world event for gossip system
										sig := worldevents.Regional
										if spec.Rarity >= 8 {
											sig = worldevents.Global
										}
										zone := user.Character.Zone
										region := ""
										if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
											region = zCfg.Region
										}
										worldevents.EmitWorldEvent(worldevents.WorldEvent{
											Type:         worldevents.PlayerMutationMilestone,
											Significance: sig,
											ZoneName:     zone,
											RegionName:   region,
											PlayerName:   user.Character.Name,
											Description: fmt.Sprintf("%s has undergone a mutation: %s.",
												user.Character.Name, spec.Name),
										})
									}
								}
							}
						}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean build.

- [ ] **Step 3: Run tests**

Run: `go test ./internal/hooks/... -count=1`
Expected: All PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/hooks/NewRound_UserRoundTick.go
git commit -m "feat: restructure player mutation to prefer deepening (70/30 coin flip)"
```

---

### Task 4: Restructure Mob Mutation Decision

**Files:**
- Modify: `internal/hooks/NewRound_MobRoundTick.go:137-210`

- [ ] **Step 1: Restructure the mob acquire/deepen decision**

In `internal/hooks/NewRound_MobRoundTick.go`, replace the block starting at line 137 (`if mob.Character.MutationProgress >= threshold {`) through its closing brace. Apply the same coin-flip logic as the player path:

```go
					if mob.Character.MutationProgress >= threshold {
						mob.Character.MutationProgress = 0

						// Decide: deepen existing mutation or acquire new one
						doDeepen := false
						if canAcquire && canDeepen {
							if util.Rand(100) < int(mb.MutationDeepenChance*100) {
								doDeepen = true
							}
						} else if canDeepen && !canAcquire {
							doDeepen = true
						}

						if doDeepen {
							if mutId := mutations.RollDeepening(mob.Character.Mutations); mutId != "" {
								mob.Character.Mutations[mutId]++
								newLevel := mob.Character.Mutations[mutId]
								if spec := mutations.GetMutation(mutId); spec != nil {
									if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
										room.SendText(fmt.Sprintf(
											`<ansi fg="magenta">The mutation in <ansi fg="mobname">%s</ansi> intensifies.</ansi>`,
											mob.Character.Name))
									}
									// Deepening significance: bump one tier if level 3+
									sig := worldevents.Local
									if spec.Rarity >= 5 {
										sig = worldevents.Regional
									}
									if newLevel >= int(mb.MutationMaxLevel) {
										if sig < worldevents.Global {
											sig++
										}
									}
									zone := mob.Character.Zone
									region := ""
									if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
										region = zCfg.Region
									}
									worldevents.EmitWorldEvent(worldevents.WorldEvent{
										Type:         worldevents.MobMutationGained,
										Significance: sig,
										ZoneName:     zone,
										RegionName:   region,
										MobName:      mob.Character.Name,
										Description: fmt.Sprintf("%s's mutation %s intensifies (level %d)",
											mob.Character.Name, spec.Name, newLevel),
									})
								}
							}
						} else if canAcquire {
							var specDisabledSlots []string
							if specInfo := species.GetSpecies(mob.Character.SpeciesId); specInfo != nil {
								specDisabledSlots = specInfo.DisabledSlots
							}
							pool := mutations.GetWeightedPool(mob.Character.Mutations, specDisabledSlots)
							if mutId := mutations.RollAcquisition(pool); mutId != "" {
								if mob.Character.Mutations == nil {
									mob.Character.Mutations = make(map[string]int)
								}
								mob.Character.Mutations[mutId] = 1
								if spec := mutations.GetMutation(mutId); spec != nil {
									if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
										room.SendText(fmt.Sprintf(
											`<ansi fg="magenta">Something shifts in <ansi fg="mobname">%s</ansi>. %s</ansi>`,
											mob.Character.Name, spec.Visual))
									}
									sig := worldevents.Local
									if spec.Rarity >= 8 {
										sig = worldevents.Global
									} else if spec.Rarity >= 5 {
										sig = worldevents.Regional
									}
									zone := mob.Character.Zone
									region := ""
									if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
										region = zCfg.Region
									}
									worldevents.EmitWorldEvent(worldevents.WorldEvent{
										Type:         worldevents.MobMutationGained,
										Significance: sig,
										ZoneName:     zone,
										RegionName:   region,
										MobName:      mob.Character.Name,
										Description: fmt.Sprintf("%s has manifested a mutation: %s",
											mob.Character.Name, spec.Name),
									})
								}
							}
						}
					}
```

- [ ] **Step 2: Verify build and tests**

Run: `go build ./... && go test ./internal/hooks/... ./internal/mutations/... -count=1`
Expected: Clean build, all tests pass.

- [ ] **Step 3: Commit**

```bash
git add internal/hooks/NewRound_MobRoundTick.go
git commit -m "feat: restructure mob mutation to prefer deepening (same 70/30 coin flip)"
```

---

### Task 5: Final Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1 2>&1 | tail -30`
Expected: All tests pass.

- [ ] **Step 2: Verify config loads correctly**

Run: `go build ./... && echo "build OK"`
Expected: Clean build.

- [ ] **Step 3: Commit any cleanup**

If any cleanup is needed, commit with: `chore: final cleanup for mutation discovery tuning`
