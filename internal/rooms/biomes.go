package rooms

import (
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/fileloader"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/pkg/errors"
)

type BiomeInfo struct {
	BiomeId        string  `yaml:"biomeid"`
	Name           string  `yaml:"name"`
	Symbol         string  `yaml:"symbol"`
	Description    string  `yaml:"description"`
	DarkArea       bool    `yaml:"darkarea"`
	LitArea        bool    `yaml:"litarea"`
	RequiredItemId int     `yaml:"requireditemid"`
	UsesItem       bool    `yaml:"usesitem"`
	Burns          bool    `yaml:"burns"`
	MovementCost   float64 `yaml:"movementcost"` // Terrain difficulty multiplier for stamina cost (1.0 = normal, 2.0 = rough)

	// Private fields for runtime use
	symbolRune rune
	filepath   string
}

func (bi *BiomeInfo) GetSymbol() rune {
	if bi.symbolRune == 0 && len(bi.Symbol) > 0 {
		for _, r := range bi.Symbol {
			bi.symbolRune = r
			break
		}
	}
	return bi.symbolRune
}

func (bi *BiomeInfo) SymbolString() string {
	return bi.Symbol
}

func (bi *BiomeInfo) IsLit() bool {
	return bi.LitArea && !bi.DarkArea
}

func (bi *BiomeInfo) IsDark() bool {
	return !bi.LitArea && bi.DarkArea
}

// GetMovementCost returns the terrain difficulty multiplier for stamina cost.
// Returns 1.0 (normal terrain) if not set.
func (bi *BiomeInfo) GetMovementCost() float64 {
	if bi.MovementCost <= 0 {
		return 1.0
	}
	return bi.MovementCost
}

// Implement Loadable interface
func (bi *BiomeInfo) Id() string {
	return strings.ToLower(bi.BiomeId)
}

func (bi *BiomeInfo) Validate() error {
	if bi.BiomeId == "" {
		return fmt.Errorf("biomeid cannot be empty")
	}
	if bi.Name == "" {
		return fmt.Errorf("biome name cannot be empty")
	}
	if bi.Symbol == "" || bi.Symbol == "?" {
		return fmt.Errorf("biome '%s' has invalid or missing symbol", bi.BiomeId)
	}
	if bi.DarkArea && bi.LitArea {
		return fmt.Errorf("biome '%s' cannot be both dark and lit", bi.BiomeId)
	}
	return nil
}

func (bi *BiomeInfo) Filepath() string {
	if bi.filepath == "" {
		bi.filepath = fmt.Sprintf("%s.yaml", bi.BiomeId)
	}
	return bi.filepath
}

var (
	biomes = map[string]*BiomeInfo{}
)

func LoadBiomeDataFiles() {

	start := time.Now()

	dataPath := configs.GetFilePathsConfig().DataFiles.String() + `/biomes`
	tmpBiomes, err := fileloader.LoadAllFlatFiles[string, *BiomeInfo](dataPath)
	if err != nil {
		panic(errors.Wrap(err, `filepath: `+dataPath))
	}

	biomes = tmpBiomes

	if len(biomes) == 0 {
		mudlog.Warn("No biomes loaded from files, using default fallback biome")
		// Create a single default fallback biome
		biomes[`default`] = &BiomeInfo{
			BiomeId:      `default`,
			Name:         `Default`,
			Symbol:       `•`,
			LitArea:      true,
			Description:  `A default biome used when no other biome is specified.`,
			MovementCost: 1.0,
		}
	} else {
		// Always ensure a default biome exists as fallback
		if _, ok := biomes[`default`]; !ok {
			biomes[`default`] = &BiomeInfo{
				BiomeId:      `default`,
				Name:         `Default`,
				Symbol:       `•`,
				LitArea:      true,
				Description:  `A default biome used when no other biome is specified.`,
				MovementCost: 1.0,
			}
		}
	}

	mudlog.Info("biomes.LoadBiomeDataFiles()", "loadedCount", len(biomes), "Time Taken", time.Since(start))
}

func GetBiome(name string) (*BiomeInfo, bool) {
	if name == `` {
		name = `default`
	}
	b, ok := biomes[strings.ToLower(name)]
	return b, ok
}

func GetAllBiomes() []BiomeInfo {
	ret := []BiomeInfo{}
	for _, b := range biomes {
		ret = append(ret, *b)
	}
	return ret
}
