package usercommands

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func setupCrimesAndFactionsForAdminTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_CRIMES_DIR_OVERRIDE", t.TempDir())
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	body := `faction_id: thornwall_citizens
display_name: "Citizens"
description: "x"
default_rep: 0
allies: []
enemies: []
`
	if err := os.WriteFile(filepath.Join(dir, "thornwall_citizens.yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
	crimes.ClearCache()
}

func TestAdminCrime_ListEmpty(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	if _, err := Crime("list thornwall_citizens", admin, room, 0); err != nil {
		t.Fatalf("crime list: %v", err)
	}
}

func TestAdminCrime_ListAfterRecord(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	victim := &mobs.Mob{MobId: 100, Character: characters.Character{Name: "city beggar"}}
	crimes.Record([]string{"thornwall_citizens"}, crimes.KindMurder,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: target.UserId},
		victim, 250, 467, "Thornwall City", false)

	if _, err := Crime("list thornwall_citizens", admin, room, 0); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCrime_Show(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	target := users.GetByUserId(1)

	victim := &mobs.Mob{MobId: 100, Character: characters.Character{Name: "city beggar"}}
	crimes.Record([]string{"thornwall_citizens"}, crimes.KindAssault,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: target.UserId},
		victim, 250, 467, "Thornwall City", false)

	if _, err := Crime("show "+target.Character.Name, admin, room, 0); err != nil {
		t.Fatal(err)
	}
}

func TestAdminCrime_Resolve(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)

	victim := &mobs.Mob{MobId: 100, Character: characters.Character{Name: "city beggar"}}
	ids := crimes.Record([]string{"thornwall_citizens"}, crimes.KindMurder,
		crimes.Perpetrator{Type: crimes.PerpPlayer, Id: 17},
		victim, 250, 467, "Thornwall City", false)

	cmd := "resolve thornwall_citizens " + strconv.Itoa(ids[0]) + " fine paid"
	if _, err := Crime(cmd, admin, room, 0); err != nil {
		t.Fatal(err)
	}
	rows := crimes.AllForFaction("thornwall_citizens", true)
	if rows[0].ResolvedBy != "fine paid" {
		t.Errorf("expected resolved with reason; got %+v", rows[0])
	}
}

func TestAdminCrime_UnknownFaction(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	if _, err := Crime("list nonexistent", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// No panic, friendly error sent.
}

func TestAdminCrime_PruneStale(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()
	setupCrimesAndFactionsForAdminTest(t)

	admin, room := getTestUserAndRoom(t)
	if _, err := Crime("prune-stale thornwall_citizens", admin, room, 0); err != nil {
		t.Fatal(err)
	}
	// No assertion on count — just confirm no panic.
}
