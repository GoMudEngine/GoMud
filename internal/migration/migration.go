package migration

import (
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/version"
)

// Migration code goes here.
// They should be put in the order of oldest to newest and follow the pattern as below
func doAllMigrations(lastConfigVersion version.Version) error {

	// 0.0.0 -> 0.9.1
	if lastConfigVersion.IsOlderThan(version.New(0, 9, 1)) {

		if err := migrate_RoomZoneConfig(); err != nil {
			return err
		}

	}

	// 0.9.1 -> 0.10.0
	if lastConfigVersion.IsOlderThan(version.New(0, 10, 0)) {

		if err := migrate_UserStatsRename(); err != nil {
			return err
		}

	}

	// 0.10.0 -> 0.11.0
	if lastConfigVersion.IsOlderThan(version.New(0, 11, 0)) {

		if err := migrate_RollCharacterStats(); err != nil {
			return err
		}

	}

	// 0.11.0 -> 0.12.0
	if lastConfigVersion.IsOlderThan(version.New(0, 12, 0)) {

		if err := migrate_RaceToSpecies(); err != nil {
			return err
		}

	}

	// 0.12.0 -> 0.13.0
	if lastConfigVersion.IsOlderThan(version.New(0, 13, 0)) {

		// Migrations run BEFORE main.go's data-load block, so
		// factions.LoadAllDefinitions() has not yet been called.
		// Load now so the rep-seeding can resolve the warren faction.
		if err := factions.LoadAllDefinitions(); err != nil {
			return err
		}

		if err := migrate_SeedWarrenRepFromQuestToken(); err != nil {
			return err
		}

	}

	if lastConfigVersion.IsOlderThan(version.New(0, 14, 0)) {
		// Player mutation migration: reclassify every save onto the cluster
		// graph (wipe retired-41, grant a cluster seed). Datafiles are backed
		// up by Run() before this and restored on error (spec §7 safety).
		if err := migrate_ReclassifyPlayerMutations(false); err != nil {
			return err
		}
	}

	return nil
}

// Entrypoint for migrations.
// This is run on server start-up, after config files are loaded.
// NOTE: This means migrations that modify config files themselves would need special consideration
func Run(lastConfigVersion version.Version, serverVersion version.Version) error {

	//
	// If already up to speed on version, we don't really need to do anything.
	//
	if lastConfigVersion.IsEqualTo(serverVersion) {
		return nil
	}

	//
	// Start by making a backup of all datafiles.
	//
	backupFolder, err := datafilesBackup()
	if err != nil {
		return fmt.Errorf(`could not backup datafiles: %w`, err)
	}
	defer os.RemoveAll(backupFolder)

	//
	// If an error occured, restore backup
	//
	if err := doAllMigrations(lastConfigVersion); err != nil {
		copyDir(backupFolder, string(configs.GetFilePathsConfig().DataFiles))
		return err
	}

	//
	// Finally, since successful, update to the version this migration is for
	//
	configs.SetVal(`Server.CurrentVersion`, serverVersion.String())

	return nil
}
