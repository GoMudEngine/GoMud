package hooks

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// helper: write a faction definition into the test override dir.
func writeFactionDef(t *testing.T, slug, body string) {
	t.Helper()
	dir := os.Getenv("DOGMUD_FACTIONS_DIR_OVERRIDE")
	if err := os.WriteFile(filepath.Join(dir, slug+".yaml"), []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
}

func setupFactionsForHookTest(t *testing.T) {
	t.Helper()
	t.Setenv("DOGMUD_FACTIONS_DIR_OVERRIDE", t.TempDir())
	t.Setenv("DOGMUD_FACTIONS_REP_DIR_OVERRIDE", t.TempDir())
	writeFactionDef(t, "warren", `
faction_id: warren
display_name: "The Warren"
description: "x"
default_rep: 0
allies: []
enemies: []
`)
	if err := factions.LoadAllDefinitions(); err != nil {
		t.Fatal(err)
	}
	factions.ClearCache()
}

func TestMobDeathFactionRep_BumpsForDamager(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      72,
		InstanceId: 200,
		Groups:     []string{"warren"},
		Character: characters.Character{
			Name:   "warren scout",
			RoomId: 301,
			Buffs:  buffs.New(),
		},
	}
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{72: mob}, map[int]*mobs.Mob{200: mob})
	defer cleanup()

	evt := events.MobDeath{
		MobId:        72,
		InstanceId:   200,
		RoomId:       301,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	want := int(configs.GetBalanceConfig().FactionMemberKillRep)
	if got := factions.GetRep("warren", 17); got != want {
		t.Errorf("after kill, warren rep for user 17 = %d, want %d", got, want)
	}
}

func TestMobDeathFactionRep_NoFactionsNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      999,
		InstanceId: 201,
		Groups:     []string{"humanoid"}, // no defined faction
		Character:  characters.Character{Name: "x", RoomId: 301, Buffs: buffs.New()},
	}
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{999: mob}, map[int]*mobs.Mob{201: mob})
	defer cleanup()

	evt := events.MobDeath{
		MobId:        999,
		InstanceId:   201,
		RoomId:       301,
		PlayerDamage: map[int]int{17: 50},
	}
	MobDeathFactionRep(evt)

	if got := factions.GetRep("warren", 17); got != 0 {
		t.Errorf("warren rep changed for non-faction kill: %d", got)
	}
}

func TestMobDeathFactionRep_NoPlayersNoChange(t *testing.T) {
	setupFactionsForHookTest(t)

	mob := &mobs.Mob{
		MobId:      72,
		InstanceId: 202,
		Groups:     []string{"warren"},
		Character:  characters.Character{Name: "scout", RoomId: 301, Buffs: buffs.New()},
	}
	cleanup := mobs.SeedMobsForTest(map[int]*mobs.Mob{72: mob}, map[int]*mobs.Mob{202: mob})
	defer cleanup()

	evt := events.MobDeath{
		MobId:        72,
		InstanceId:   202,
		RoomId:       301,
		PlayerDamage: map[int]int{}, // env damage / no contributors
	}
	MobDeathFactionRep(evt)

	if got := factions.GetRep("warren", 17); got != 0 {
		t.Errorf("warren rep changed despite no damager: %d", got)
	}
}
