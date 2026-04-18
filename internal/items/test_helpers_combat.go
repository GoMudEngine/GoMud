package items

// File: test_helpers_combat.go
//
// Test-only helper to seed the attackMessages / defenseMessages maps.
// Production code populates these via LoadDataFiles() at startup; tests
// in other packages (e.g., hooks-package combat-routing tests) need a
// way to inject minimal entries to avoid the fatal infinite recursion
// in GetAttackMessage when the maps are empty.

// SeedAttackMessagesForTest replaces the package-level attackMessages
// map with the supplied data and returns a cleanup function that
// restores the original. Pass nil to install an empty map (rarely
// useful — combat code calls GetAttackMessage which infinite-recurses
// when no entry resolves).
func SeedAttackMessagesForTest(msgs map[ItemSubType]*WeaponAttackMessageGroup) func() {
	orig := attackMessages
	if msgs == nil {
		attackMessages = map[ItemSubType]*WeaponAttackMessageGroup{}
	} else {
		attackMessages = msgs
	}
	return func() {
		attackMessages = orig
	}
}

// SeedDefenseMessagesForTest replaces the package-level defenseMessages
// map and returns a restore cleanup. See SeedAttackMessagesForTest.
func SeedDefenseMessagesForTest(msgs map[DefenseType]*DefenseMessageGroup) func() {
	orig := defenseMessages
	if msgs == nil {
		defenseMessages = map[DefenseType]*DefenseMessageGroup{}
	} else {
		defenseMessages = msgs
	}
	return func() {
		defenseMessages = orig
	}
}

// MinimalCombatMessageFixture returns a minimal attackMessages map keyed
// on subType=Generic that resolves all five Intensity tiers with a
// single placeholder phrase. Sufficient to satisfy GetAttackMessage's
// fallback path during cross-package combat tests.
func MinimalCombatMessageFixture() map[ItemSubType]*WeaponAttackMessageGroup {
	mk := func(s string) MessageOptions {
		return MessageOptions{ItemMessage(s)}
	}
	tieredAll := SkillTieredMessages{
		Beginner: mk("you swing"),
		Expert:   mk("you swing"),
		Master:   mk("you swing"),
	}
	options := AttackOptions{
		Together: TogetherMessages{
			ToAttacker: tieredAll,
			ToDefender: tieredAll,
			ToRoom:     tieredAll,
		},
		Separate: SeparateMessages{
			ToAttacker:     tieredAll,
			ToDefender:     tieredAll,
			ToAttackerRoom: tieredAll,
			ToDefenderRoom: tieredAll,
		},
	}
	return map[ItemSubType]*WeaponAttackMessageGroup{
		Generic: {
			OptionId: Generic,
			Options: AttackTypes{
				Prepare:  options,
				Wait:     options,
				Miss:     options,
				Weak:     options,
				Normal:   options,
				Heavy:    options,
				Critical: options,
				Fumble:   options,
			},
		},
	}
}
