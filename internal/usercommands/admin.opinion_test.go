package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// drainSendText is a stub. user.SendText enqueues to an event queue
// rather than writing to an in-test buffer, so we cannot capture output
// without standing up the full event pipeline. The behavioral asserts
// (state changes via opinions.Get) carry the test load.
func drainSendText(u *users.UserRecord) string {
	return ""
}

func TestAdminOpinionSetAndShow(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1) // same user for simplicity
	if target == nil {
		t.Fatal("test user 1 missing")
	}

	// Use mobId=1 (Skeleton, seeded in usercommands_test.go).
	cmd := "set 1 " + target.Character.Name + " -60"
	if _, err := Opinion(cmd, admin, room, 0); err != nil {
		t.Fatalf("opinion set: %v", err)
	}
	if got := opinions.Get(1, target.UserId); got != -60 {
		t.Errorf("after set, Get = %d, want -60", got)
	}

	// Drain the user's text buffer before show (no-op; see drainSendText).
	_ = drainSendText(admin)
	if _, err := Opinion("show "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatalf("opinion show: %v", err)
	}
	// Note: output content assertions are skipped because user.SendText
	// enqueues to the event queue rather than an in-test capture buffer.
	// The show command exercising without error is sufficient here.
}

// TestAdminOpinionMultiWordMob is the Stage 2 fix target: "opinion set bank
// clerk <player> <score>" must resolve the multi-word mob template. Before the
// fix, positional args + a space-vs-underscore ident comparison made it fail
// (mob unresolved / arg count wrong). Observable via opinions.Get.
func TestAdminOpinionMultiWordMob(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	cleanMobs := mobs.SeedMobsForTest(
		map[int]*mobs.Mob{
			1: {MobId: 1, Zone: "TestZone", Character: characters.Character{Name: "Skeleton"}},
			3: {MobId: 3, Zone: "TestZone", Character: characters.Character{Name: "Bank Clerk"}},
		},
		map[int]*mobs.Mob{},
	)
	defer cleanMobs()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	if _, err := Opinion("set bank clerk "+target.Character.Name+" -60", admin, room, 0); err != nil {
		t.Fatalf("opinion set multi-word mob: %v", err)
	}
	if got := opinions.Get(3, target.UserId); got != -60 {
		t.Errorf("after set on multi-word mob, Get(3) = %d, want -60", got)
	}
}

func TestAdminOpinionBump(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	if _, err := Opinion("bump 1 "+target.Character.Name+" -10", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if got := opinions.Get(1, target.UserId); got != -10 {
		t.Errorf("after bump -10, Get = %d, want -10", got)
	}
	if _, err := Opinion("bump 1 "+target.Character.Name+" -10", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if got := opinions.Get(1, target.UserId); got != -20 {
		t.Errorf("after second bump, Get = %d, want -20", got)
	}
}

func TestAdminOpinionReset(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	if _, err := Opinion("set 1 "+target.Character.Name+" -90", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if _, err := Opinion("reset 1 "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// Mob 1 (Skeleton) has no DefaultDisposition set in the seed, so Go's
	// zero value (0) applies. After reset, Get should return 0.
	if got := opinions.Get(1, target.UserId); got != 0 {
		t.Errorf("after reset, Get = %d, want 0", got)
	}
}

func TestAdminOpinionUnknownSubcommand(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)

	// Unknown subcommand should return true (handled) and no error.
	handled, err := Opinion("frobnicate", admin, room, 0)
	if !handled {
		t.Error("expected handled=true for unknown subcommand")
	}
	if err != nil {
		t.Errorf("expected no error for unknown subcommand, got %v", err)
	}
}

func TestAdminOpinionNoArgs(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)

	handled, err := Opinion("", admin, room, 0)
	if !handled {
		t.Error("expected handled=true for empty args")
	}
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestAdminOpinionShowOne(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	dir := t.TempDir()
	t.Setenv("DOGMUD_OPINIONS_DIR_OVERRIDE", dir)
	opinions.ClearCache()

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	// Seed a score first.
	opinions.Set(1, target.UserId, 42)

	// show <mobId> <playerName>
	if _, err := Opinion("show 1 "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatalf("opinion show one: %v", err)
	}
	// Behavioral: score still 42 after a read-only show.
	if got := opinions.Get(1, target.UserId); got != 42 {
		t.Errorf("score changed after show: got %d, want 42", got)
	}
}
