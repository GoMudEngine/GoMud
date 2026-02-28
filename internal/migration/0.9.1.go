package migration

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"gopkg.in/yaml.v2"
)

// Description:
// rooms.Room.ZoneConfig was removed when Zone data was migrated to zone-config.yaml in zone folders
// This function loads all of the yaml files in the DATAFILES/world/*/rooms/* and looks for any ZoneConfig data.
// If found, the data is moved to a zone-config.yaml file, and the ZoneConfig data in the Room datafile is removed.
func migrate_RoomZoneConfig() error {

	// This struct is how ZoneConfig looked as of 0.9.1
	// Since we will be upgrading an older version to this format, use a copy of the struct from that period
	// To ensure we aren't using a struct that has changed over time
	type zoneConfig_1_0_0 struct {
		Name         string `yaml:"name,omitempty"`
		RoomId       int    `yaml:"roomid,omitempty"`
		MobAutoScale struct {
			Minimum int `yaml:"minimum,omitempty"` // level scaling minimum
			Maximum int `yaml:"maximum,omitempty"` // level scaling maximum
		} `yaml:"autoscale,omitempty"` // level scaling range if any
		Mutators []struct {
			MutatorId      string `yaml:"mutatorid,omitempty"`      // Short text that will uniquely identify this modifier ("dusty")
			SpawnedRound   uint64 `yaml:"spawnedround,omitempty"`   // Tracks when this mutator was created (useful for decay)
			DespawnedRound uint64 `yaml:"despawnedround,omitempty"` // Track when it decayed to nothing.
		} `yaml:"mutators,omitempty"`
		IdleMessages []string         `yaml:"idlemessages,omitempty"` // list of messages that can be displayed to players in the zone, assuming a room has none defined
		MusicFile    string           `yaml:"musicfile,omitempty"`    // background music to play when in this zone
		DefaultBiome string           `yaml:"defaultbiome,omitempty"` // city, swamp etc. see biomes.go
		RoomIds      map[int]struct{} `yaml:"-"`                      // Does not get written. Built dyanmically when rooms are loaded.
	}

	c := configs.GetConfig()

	worldfilesGlob := filepath.Join(string(c.FilePaths.DataFiles), "rooms", "*", "*.yaml")
	matches, err := filepath.Glob(worldfilesGlob)

	if err != nil {
		return err
	}

	existingZoneFiles := map[string]struct{}{}

	// We only care about room files, so ###.yaml (possible negative)
	re := regexp.MustCompile(`^[\-0-9]+\.yaml$`)
	for _, path := range matches {

		//
		// Must look like a room yaml file:
		// 1.yaml
		// 123.yaml
		// -83.yaml
		// etc.
		//

		if !re.MatchString(filepath.Base(path)) {
			continue
		}

		//
		// strip the filename form the room file and replace with zone-config.yaml
		// to get the path to the zone-config.yaml
		//
		zoneFilePath := filepath.Join(filepath.Dir(path), "zone-config.yaml")

		//
		// The following checks whether the zone config file already exists
		// We will leave the config data in the room data file if the zone-config.yaml is already present.
		// It should be inert if present, since it is not unmarshalled into anything in current code.
		//

		// Check whether zone file already is tracked as existing, if found, skip.
		if _, ok := existingZoneFiles[zoneFilePath]; ok {
			continue
		}

		_, err = os.Stat(zoneFilePath)
		if err == nil {
			// Mark zone file as existing, skip further processing.
			existingZoneFiles[zoneFilePath] = struct{}{}
			continue
		}

		//
		// End check for existing zone-config.yaml
		// After this point, we will unmarshal the yaml file into a generic map structure.
		// This allows us to examine the data in the yaml file, particularly the "zoneconfig" node
		// since the ZoneConfig field has been removed from the rooms.Room struct
		// We can de-populate the field, move it, and re-write the yaml back to the original room template file.
		// The downside to this method is that being a map, the fields will be read/written in a non-deterministic manner,
		// So the room yaml file field orders may be written in a random order.
		// Because of this, and as a final fix, we will finally marshal/unmarshal into the proper room struct from the map data
		// Allowing us to write the data in an expected ordered form.
		//

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		//
		// First do a simple check for the field name in the text file.
		// We know the way the field will appear: "zoneconfig:"
		// This avoids having to unmarshal the struct and search that way, unnecessarily.
		//
		if !strings.Contains(string(data), "zoneconfig:") {
			continue
		}

		//
		// Unmarshal the entire yaml file into a map
		// This will let us further examine the data, modify it, etc.
		//
		filedata := map[string]any{}
		err = yaml.Unmarshal(data, &filedata)
		if err != nil {
			return fmt.Errorf("failed to parse YAML: %w", err)
		}

		// Make sure that the zoneconfig key is present and populated
		if filedata[`zoneconfig`] == nil {
			continue
		}

		mudlog.Info("Migration 0.9.1", "file", path, "message", "migrating zoneconfig from room data file to zone-config.yaml")

		//
		// From here on out, this code migrates zoneconfig data out of room file and into zone-config.yaml
		//
		roomFileInfo, _ := os.Stat(path)

		mudlog.Info("Migration 0.9.1", "file", path, "message", "isolating zoneconfig data")

		//
		// Isolate the zoneconfig and write it to its own zone-config.yaml file
		// We'll marshal just the zoneconfig data, get its bytes, then unmarshal it into
		// the desired target structure.
		// Some fields have changed or are missing due to some slight differences in the new struct
		// so we'll also try and reconcile some of that by pulling from the core room definition
		//
		zoneBytes, err := yaml.Marshal(filedata[`zoneconfig`])
		if err != nil {
			return err
		}

		zoneDataStruct := zoneConfig_1_0_0{}

		if err = yaml.Unmarshal(zoneBytes, &zoneDataStruct); err != nil {
			return err
		}

		if filedata[`zone`] != nil {
			if zoneName, ok := filedata[`zone`].(string); ok {
				zoneDataStruct.Name = zoneName
			} else {
				if title, ok := filedata[`title`].(string); ok {
					zoneDataStruct.Name = title
				}
			}

			if defaultBiome, ok := filedata[`biome`].(string); ok {
				zoneDataStruct.DefaultBiome = defaultBiome
			}
		}

		mudlog.Info("Migration 0.9.1", "file", path, "message", "writing "+zoneFilePath)

		//
		// Write the zone data to the zone-config.yaml path
		// We'll just use whatever permissions were set in the room file for this file.
		//
		zoneFileBytes, err := yaml.Marshal(zoneDataStruct)
		if err != nil {
			return err
		}
		if err := os.WriteFile(zoneFilePath, zoneFileBytes, roomFileInfo.Mode().Perm()); err != nil {
			return err
		}

		// Mark zone file as existing
		existingZoneFiles[zoneFilePath] = struct{}{}

		mudlog.Info("Migration 0.9.1", "file", path, "message", "writing modified room data")

		//
		// Now clear the "zoneconfig" node from the room data.
		// The data will be in a random order if we just write this back to the room yaml file,
		// so we'll take the extract step of marshalling the room data from the map into a string,
		// and then unmarshal it into the actual target rooms.Room{} struct.
		// This way, when writing to a file, it'll be in the typical field order according to the struct
		// field order.
		//
		delete(filedata, `zoneconfig`)

		// First marshal the modified room data into bytes
		modifiedRoomBytes, err := yaml.Marshal(filedata)
		if err != nil {
			return err
		}

		// Unmarshal the bytes into the proper struct
		modifiedRoomStruct := rooms.Room{}
		if err = yaml.Unmarshal(modifiedRoomBytes, &modifiedRoomStruct); err != nil {
			return err
		}

		// Marshal again, this time using the proper struct
		modifiedRoomBytes, err = yaml.Marshal(modifiedRoomStruct)
		if err != nil {
			return err
		}

		// Again, we'll just use the rooms original permissions when writing.
		if err := os.WriteFile(path, modifiedRoomBytes, roomFileInfo.Mode().Perm()); err != nil {
			return err
		}

		mudlog.Info("Migration 0.9.1", "file", path, "message", "successfully updated")

	}

	return nil
}
