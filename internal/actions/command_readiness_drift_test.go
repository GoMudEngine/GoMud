package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/stretchr/testify/assert"
)

// driftCase describes one point in the (command × gate) matrix.
type driftCase struct {
	name       string
	cmd        string
	mutate     func(*mobs.Mob)
	wantReady  bool
	wantReason string // Execute*-side flag when not ready; ignored when wantReady=true
}

// TestCommandReadinessDrift asserts CommandIsReady and each Execute*
// agree on readiness for a shared actor state. When they diverge, the
// btree's command_best_of can issue a command the command itself
// silently rejects. Surfaced once in T7 smoke testing of the tank_taunter
// archetype (taunt unregistered on mob side); this test is the guard.
//
// To add a new command or gate: add a row below. The helper at the
// bottom of this file dispatches to the right Execute* and reads the
// named result field.
//
// SCOPE LIMIT: This test checks boolean agreement ("is this command
// ready?"), not which specific reason flag Execute* returns. For the
// not-ready path, gate-ordering differences between CommandIsReady
// and Execute* can cause a case to fail for a cryptic reason-mismatch
// even when the boolean agrees; see the bash_cooldown note below for
// a concrete example.
//
// SYNC POINT: when adding a new gate to CommandIsReady or an
// Execute*, add the corresponding drift row here.
func TestCommandReadinessDrift(t *testing.T) {
	cleanup := seedBuffsForTest()
	defer cleanup()

	cases := []driftCase{
		// ─── taunt ────────────────────────────────────────────────
		{"taunt_ready", "taunt",
			func(m *mobs.Mob) { /* default has aggro */ },
			true, ""},
		{"taunt_crafting", "taunt",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"taunt_cooldown", "taunt",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"taunt_no_aggro", "taunt",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── rally ────────────────────────────────────────────────
		{"rally_ready", "rally",
			nil,
			true, ""},
		{"rally_crafting", "rally",
			func(m *mobs.Mob) { setCraftingForTest(m) },
			false, "Crafting"},
		{"rally_cooldown", "rally",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"rally_already_active", "rally",
			func(m *mobs.Mob) { m.Character.AddBuff(80, false) },
			false, "AlreadyActive"},

		// ─── warcry ───────────────────────────────────────────────
		{"warcry_ready", "warcry", nil, true, ""},
		{"warcry_crafting", "warcry",
			func(m *mobs.Mob) { setCraftingForTest(m) },
			false, "Crafting"},
		{"warcry_cooldown", "warcry",
			func(m *mobs.Mob) { m.Character.Cooldowns = characters.Cooldowns{"special-move": 3} },
			false, "OnCooldown"},
		{"warcry_already_active", "warcry",
			func(m *mobs.Mob) { m.Character.AddBuff(79, false) },
			false, "AlreadyActive"},

		// ─── trip ─────────────────────────────────────────────────
		{"trip_ready", "trip",
			func(m *mobs.Mob) {
				// Target must be standing (not prone). newTestMob
				// sets default aggro to user 1, but for trip we need
				// a real target mob that's not prone.
				targetMob := &mobs.Mob{InstanceId: 200}
				targetMob.Character.Name = "Target"
				targetMob.Character.CombatPosition = characters.PositionStanding
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			true, ""},
		{"trip_crafting", "trip",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
				targetMob := &mobs.Mob{InstanceId: 201}
				targetMob.Character.Name = "Target"
				targetMob.Character.CombatPosition = characters.PositionStanding
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "Crafting"},
		{"trip_cooldown", "trip",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
				targetMob := &mobs.Mob{InstanceId: 202}
				targetMob.Character.Name = "Target"
				targetMob.Character.CombatPosition = characters.PositionStanding
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "OnCooldown"},
		{"trip_no_aggro", "trip",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── bash ─────────────────────────────────────────────────
		// NOTE: bash_cooldown is intentionally omitted. CommandIsReady
		// rejects on the universal cooldown gate first, but ExecuteBash
		// rejects on NoShield first (default test mobs have no shield
		// and no naturalbash). Both agree on the readiness bool, but
		// the reason flag would differ. This test is a readiness-bool
		// agreement test, not a reason-flag agreement test. If
		// ExecuteBash's gate ordering ever changes to put cooldown
		// before NoShield, add the bash_cooldown row back.
		{"bash_crafting", "bash",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"bash_no_shield", "bash",
			func(m *mobs.Mob) {
				m.Character.SpeciesId = 1 // human, no naturalbash, no shield
			},
			false, "NoShield"},

		// ─── grapple ──────────────────────────────────────────────
		{"grapple_crafting", "grapple",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
				targetMob := &mobs.Mob{InstanceId: 204}
				targetMob.Character.Name = "Target"
				targetMob.Character.CombatPosition = characters.PositionStanding
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "Crafting"},
		{"grapple_cooldown", "grapple",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
				targetMob := &mobs.Mob{InstanceId: 205}
				targetMob.Character.Name = "Target"
				targetMob.Character.CombatPosition = characters.PositionStanding
				mobs.SetInstanceForTest(targetMob.InstanceId, targetMob)
				m.Character.SetAggro(0, targetMob.InstanceId, characters.DefaultAttack)
			},
			false, "OnCooldown"},
		{"grapple_no_aggro", "grapple",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},

		// ─── kick ─────────────────────────────────────────────────
		{"kick_ready", "kick",
			func(m *mobs.Mob) { /* default has aggro */ },
			true, ""},
		{"kick_crafting", "kick",
			func(m *mobs.Mob) {
				setCraftingForTest(m)
			},
			false, "Crafting"},
		{"kick_cooldown", "kick",
			func(m *mobs.Mob) {
				m.Character.Cooldowns = characters.Cooldowns{"special-move": 3}
			},
			false, "OnCooldown"},
		{"kick_no_aggro", "kick",
			func(m *mobs.Mob) { m.Character.EndAggro() },
			false, "NoTarget"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Clean up test mobs after each case to avoid pollution.
			// The mutate function may set up target mobs via
			// mobs.SetInstanceForTest; we clean them all up here.
			defer func() {
				for id := 200; id <= 205; id++ {
					mobs.SetInstanceForTest(id, nil)
				}
			}()

			mob := newTestMob(t, tc.mutate)
			actor := &MobActor{Mob: mob, Room: nil}

			gotReady := CommandIsReady(actor, tc.cmd)
			assert.Equal(t, tc.wantReady, gotReady,
				"CommandIsReady(%s) for case %q", tc.cmd, tc.name)

			if tc.wantReady {
				// Don't run Execute* for happy-path — some have target-
				// resolution or other side effects that require more
				// setup. The !wantReady path is where drift matters.
				return
			}

			gotFlag := runExecuteAndReadFlag(tc.cmd, actor, tc.wantReason)
			assert.True(t, gotFlag,
				"Execute%s for case %q did not return %s=true", tc.cmd, tc.name, tc.wantReason)
		})
	}
}

// runExecuteAndReadFlag dispatches to the Execute* matching cmd and
// returns whether the named result field is true.
func runExecuteAndReadFlag(cmd string, actor Actor, flag string) bool {
	switch cmd {
	case "taunt":
		r := ExecuteTaunt(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "rally":
		r := ExecuteRally(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "warcry":
		r := ExecuteWarcry(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "AlreadyActive":
			return r.AlreadyActive
		}
	case "trip":
		r := ExecuteTrip(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "bash":
		r := ExecuteBash(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoShield":
			return r.NoShield
		}
	case "grapple":
		r := ExecuteGrapple(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	case "kick":
		r := ExecuteKick(actor)
		switch flag {
		case "Crafting":
			return r.Crafting
		case "OnCooldown":
			return r.OnCooldown
		case "NoTarget":
			return r.NoTarget
		}
	}
	return false
}

// setCraftingForTest puts a mob into crafting state.
func setCraftingForTest(m *mobs.Mob) {
	m.Character.CraftingState = &characters.CraftingState{
		RecipeId:    "test-recipe",
		RoundsTotal: 5,
	}
}
