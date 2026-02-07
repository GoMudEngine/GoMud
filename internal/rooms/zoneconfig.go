package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type ZoneConfig struct {
	Name         string `yaml:"name,omitempty"`
	RoomId       int    `yaml:"roomid,omitempty"`
	MobAutoScale struct {
		Minimum int `yaml:"minimum,omitempty"` // stat pool scaling minimum
		Maximum int `yaml:"maximum,omitempty"` // stat pool scaling maximum
	} `yaml:"autoscale,omitempty"` // stat pool scaling range if any
	Mutators     mutators.MutatorList `yaml:"mutators,omitempty"`     // mutators defined here apply to entire zone
	IdleMessages []string             `yaml:"idlemessages,omitempty"` // list of messages that can be displayed to players in the zone, assuming a room has none defined
	MusicFile    string               `yaml:"musicfile,omitempty"`    // background music to play when in this zone
	DefaultBiome string               `yaml:"defaultbiome,omitempty"` // city, swamp etc. see biomes.go
	RoomIds      map[int]struct{}     `yaml:"-"`                      // Does not get written. Built dyanmically when rooms are loaded.
}

// Generates a random stat pool value between min and max
func (z *ZoneConfig) GenerateRandomStatPool() int {
	return util.Rand(z.MobAutoScale.Maximum-z.MobAutoScale.Minimum) + z.MobAutoScale.Minimum
}

func (z *ZoneConfig) Id() string {
	return z.Name
}

func (z *ZoneConfig) Validate() error {
	if z.MobAutoScale.Minimum < 0 {
		z.MobAutoScale.Minimum = 0
	}

	if z.MobAutoScale.Maximum < 0 {
		z.MobAutoScale.Maximum = 0
	}

	// If either is set, neither can be zero.
	if z.MobAutoScale.Minimum > 0 || z.MobAutoScale.Maximum > 0 {

		if z.MobAutoScale.Maximum < z.MobAutoScale.Minimum {
			z.MobAutoScale.Maximum = z.MobAutoScale.Minimum
		}

		if z.MobAutoScale.Minimum == 0 {
			z.MobAutoScale.Minimum = z.MobAutoScale.Maximum
		}
	}

	if z.RoomIds == nil {
		z.RoomIds = make(map[int]struct{})
	}

	return nil
}

func (z *ZoneConfig) Filename() string {
	return "zone-config.yaml"
}

func (z *ZoneConfig) Filepath() string {
	zone := ZoneNameSanitize(z.Name)
	return util.FilePath(zone, `/`, z.Filename())
}

func NewZoneConfig(zName string) *ZoneConfig {
	return &ZoneConfig{
		Name:    zName,
		RoomIds: map[int]struct{}{},
	}
}
