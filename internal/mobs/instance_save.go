package mobs

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// MobInstanceData holds only the progression fields that need to persist
// across server restarts. Combat state (health, stamina, aggro, etc.) is
// intentionally excluded — mobs respawn at full resources.
type MobInstanceData struct {
	StrengthTraining   int            `yaml:"strength_training,omitempty"`
	DexterityTraining  int            `yaml:"dexterity_training,omitempty"`
	PerceptionTraining int            `yaml:"perception_training,omitempty"`
	VitalityTraining   int            `yaml:"vitality_training,omitempty"`
	WillpowerTraining  int            `yaml:"willpower_training,omitempty"`
	CharismaTraining   int            `yaml:"charisma_training,omitempty"`
	Skills             map[string]int `yaml:"skills,omitempty"`
	SkillUseCount      map[string]int `yaml:"skill_use_count,omitempty"`
	StatUseCount       map[string]int `yaml:"stat_use_count,omitempty"`
	Mutations          map[string]int `yaml:"mutations,omitempty"`
	MutationProgress   float64        `yaml:"mutation_progress,omitempty"`
}

// instanceFilename returns the base filename for a mob instance save.
func instanceFilename(mobId MobId, mobName string, homeRoomId int) string {
	return fmt.Sprintf("%d-%s-room%d.yaml", int(mobId), util.ConvertForFilename(mobName), homeRoomId)
}

// instancePath returns the full filesystem path for a mob instance save.
func instancePath(mobId MobId, zone string, mobName string, homeRoomId int) string {
	zonePath := ZoneNameSanitize(zone)
	filename := instanceFilename(mobId, mobName, homeRoomId)
	return util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs.instances`, `/`, zonePath, `/`, filename,
	)
}

// SaveMobInstance writes a mob's progression data to disk so it survives
// server restarts. Only called for mobs with progression enabled.
//
// Charmed mobs are skipped — their progression is the owner's
// responsibility (CompanionInfo on the owner's user YAML). The file
// layer would otherwise be a redundant, room-keyed second persistence
// that leaks across player-summon cycles. See
// docs/superpowers/specs/2026-04-21-summons-dont-persist-design.md.
func SaveMobInstance(mob *Mob) error {
	// Companions live on CompanionInfo, not in mobs.instances/.
	if mob.Character.IsCharmed() {
		return nil
	}

	b := configs.GetBalanceConfig()
	if !bool(b.MobProgressionEnabled) {
		return nil
	}

	// Only save if the mob has gained any training beyond zero
	if !hasProgression(mob) {
		return nil
	}

	data := MobInstanceData{
		StrengthTraining:   mob.Character.Stats.Strength.Training,
		DexterityTraining:  mob.Character.Stats.Dexterity.Training,
		PerceptionTraining: mob.Character.Stats.Perception.Training,
		VitalityTraining:   mob.Character.Stats.Vitality.Training,
		WillpowerTraining:  mob.Character.Stats.Willpower.Training,
		CharismaTraining:   mob.Character.Stats.Charisma.Training,
		MutationProgress:   mob.Character.MutationProgress,
	}

	if len(mob.Character.Skills) > 0 {
		data.Skills = mob.Character.Skills
	}
	if len(mob.Character.SkillUseCount) > 0 {
		data.SkillUseCount = mob.Character.SkillUseCount
	}
	if len(mob.Character.StatUseCount) > 0 {
		data.StatUseCount = mob.Character.StatUseCount
	}
	if len(mob.Character.Mutations) > 0 {
		data.Mutations = mob.Character.Mutations
	}

	savePath := instancePath(mob.MobId, mob.Zone, mob.Character.Name, mob.HomeRoomId)

	// Ensure zone subdirectory exists
	dir := filepath.Dir(savePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("mobs.SaveMobInstance: mkdir %s: %w", dir, err)
	}

	bytes, err := yaml.Marshal(&data)
	if err != nil {
		return fmt.Errorf("mobs.SaveMobInstance: marshal: %w", err)
	}

	if err := os.WriteFile(savePath, bytes, 0644); err != nil {
		return fmt.Errorf("mobs.SaveMobInstance: write %s: %w", savePath, err)
	}

	return nil
}

// LoadMobInstance reads a saved mob instance from disk. Returns nil if no
// saved instance exists (indicating a fresh spawn from template).
func LoadMobInstance(mobId MobId, zone string, mobName string, homeRoomId int) *MobInstanceData {
	savePath := instancePath(mobId, zone, mobName, homeRoomId)

	bytes, err := os.ReadFile(savePath)
	if err != nil {
		return nil // File doesn't exist = fresh spawn
	}

	var data MobInstanceData
	if err := yaml.Unmarshal(bytes, &data); err != nil {
		mudlog.Error("mobs.LoadMobInstance", "path", savePath, "error", err)
		return nil
	}

	// Skill rename migration: stealth -> skullduggery
	if data.Skills != nil {
		if v, ok := data.Skills["stealth"]; ok {
			data.Skills["skullduggery"] = v
			delete(data.Skills, "stealth")
		}
	}

	return &data
}

// DeleteMobInstance removes a mob's saved instance file from disk.
// Called on mob death so respawns start fresh from template.
func DeleteMobInstance(mobId MobId, zone string, mobName string, homeRoomId int) {
	savePath := instancePath(mobId, zone, mobName, homeRoomId)
	os.Remove(savePath) // Ignore errors (file may not exist)
}

// PruneStaleInstances walks the mobs.instances directory and removes any
// instance files older than maxAgeDays. Called at startup.
func PruneStaleInstances(maxAgeDays int) {
	baseDir := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs.instances`,
	)

	cutoff := time.Now().AddDate(0, 0, -maxAgeDays)
	pruned := 0

	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if info.IsDir() {
			return nil
		}
		if info.ModTime().Before(cutoff) {
			if removeErr := os.Remove(path); removeErr == nil {
				pruned++
			}
		}
		return nil
	})

	if pruned > 0 {
		mudlog.Info("mobs.PruneStaleInstances", "pruned", pruned, "maxAgeDays", maxAgeDays)
	}
}

// NukeSummonsInstances removes every file under
// _datafiles/.../mobs.instances/summons/ at boot. Companion persistence
// lives on CompanionInfo on the owner's user YAML; any file in this
// directory is stale and would poison the next summon of the same
// template in the same room. No-op if the directory doesn't exist.
// Returns the count of files removed (for logging).
func NukeSummonsInstances() int {
	baseDir := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `mobs.instances`, `/`, `summons`,
	)

	pruned := 0
	filepath.Walk(baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors (e.g., directory doesn't exist)
		}
		if info.IsDir() {
			return nil
		}
		if removeErr := os.Remove(path); removeErr == nil {
			pruned++
		}
		return nil
	})

	if pruned > 0 {
		mudlog.Info("mobs.NukeSummonsInstances", "pruned", pruned)
	}

	return pruned
}

// hasProgression returns true if the mob has any non-zero progression data
// worth persisting (any stat training, skills, use counts, or mutations).
func hasProgression(mob *Mob) bool {
	s := &mob.Character.Stats
	if s.Strength.Training != 0 || s.Dexterity.Training != 0 ||
		s.Perception.Training != 0 || s.Vitality.Training != 0 ||
		s.Willpower.Training != 0 || s.Charisma.Training != 0 {
		return true
	}
	if len(mob.Character.Skills) > 0 {
		return true
	}
	if len(mob.Character.SkillUseCount) > 0 {
		return true
	}
	if len(mob.Character.StatUseCount) > 0 {
		return true
	}
	if len(mob.Character.Mutations) > 0 {
		return true
	}
	if mob.Character.MutationProgress > 0 {
		return true
	}
	return false
}
