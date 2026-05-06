package usercommands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func setupFactionsForAdminTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	body := `faction_id: warren
display_name: "The Warren"
description: "x"
default_rep: -25
allies: []
enemies: []
`
	if err := os.WriteFile(filepath.Join(dir, "warren.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
}

func TestAdminFaction_SetAndShow(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)
	if target == nil {
		t.Fatal("test user 1 missing")
	}

	cmd := "set warren " + target.Character.Name + " 50"
	if _, err := Faction(cmd, admin, room, 0); err != nil {
		t.Fatalf("faction set: %v", err)
	}
	if got := factions.GetRep("warren", target.UserId); got != 50 {
		t.Errorf("after set, GetRep = %d, want 50", got)
	}
}

func TestAdminFaction_Bump(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	if _, err := Faction("bump warren "+target.Character.Name+" 20", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// default_rep -25, +20 → -5
	if got := factions.GetRep("warren", target.UserId); got != -5 {
		t.Errorf("after bump 20 from default -25, GetRep = %d, want -5", got)
	}
}

func TestAdminFaction_Reset(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	factions.SetRep("warren", target.UserId, 80)
	if _, err := Faction("reset warren "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	if got := factions.GetRep("warren", target.UserId); got != -25 {
		t.Errorf("after reset, GetRep = %d, want -25 (default)", got)
	}
}

func TestAdminFaction_List(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	if _, err := Faction("list", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// We can't easily capture user.SendText output in tests; the
	// behavioral assertion is "no panic, returns true".
}

func TestAdminFaction_UnknownFaction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	// Should produce a friendly error, not a panic.
	if _, err := Faction("set nonexistent "+target.Character.Name+" 50", admin, room, 0); err != nil {
		t.Fatalf("faction set on unknown faction returned error: %v", err)
	}
	if got := factions.GetRep("nonexistent", target.UserId); got != 0 {
		t.Errorf("unknown faction should return 0, got %d", got)
	}
}
