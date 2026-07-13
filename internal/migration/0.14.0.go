package migration

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// Description:
// Player mutation migration onto the Chrysalis cluster graph. For each user:
// classify their play history into a cluster (classify.go), WIPE the retired-41
// mutations, and grant a modest, regrowable cluster seed (grant.go). Idempotent
// via a `mutation_migrated` marker under character. The framework (migration.go
// Run) backs up all datafiles before this runs and restores on error.
//
// Spec: docs/superpowers/specs/2026-07-11-mutation-migration-design.md.

// retiredMutations is the retired-41 set removed from every player's save
// (mirrors the data-side nuke). Any id here is stripped on migration.
var retiredMutations = map[string]bool{
	"adrenaline-surge": true, "bioluminescence": true, "blinding-flash": true,
	"blinding-spit": true, "brazen-resolve": true, "camo-skin": true, "clawed-hands": true,
	"cold-blooded": true, "elongated-limbs": true, "extra-legs": true, "fast-reflexes": true,
	"hasted": true, "healing-gel": true, "heightened-senses": true, "incorporeal": true,
	"infrared-vision": true, "iron-constitution": true, "keen-eyes": true, "large": true,
	"magical-resistance": true, "night-vision": true, "pacifism-aura": true,
	"pheromone-glands": true, "photosynthetic-skin": true, "psychic-resistance": true,
	"rapid-metabolism": true, "regenerative-tissue": true, "sixth-sense": true,
	"skilled": true, "small": true, "sonic-shout": true, "talented": true,
	"tough-skin": true, "toxic-bite": true,
}

var combatClusterOf = map[string]string{
	"weapon-combat": "colossus", "unarmed-combat": "ravener",
	"ranged-combat": "stalker", "skullduggery": "stalker", "rhetoric": "zealot",
}

var craftSkills = []string{"alchemy", "blacksmithing", "tailoring", "cooking", "jewelcrafting", "enchanting", "salvage"}

// migrate_ReclassifyPlayerMutations reclassifies every user save onto the cluster
// graph. When dryRun is true it logs the intended classification+grant per user
// and writes nothing — used to eyeball the result against spec §4 before applying.
func migrate_ReclassifyPlayerMutations(dryRun bool) error {
	c := configs.GetConfig()
	return reclassifyUsersInDir(filepath.Join(string(c.FilePaths.DataFiles), "users"), dryRun)
}

// reclassifyUsersInDir is the testable core — operates on any users directory so
// tests can run it against a disposable temp copy of representative accounts.
func reclassifyUsersInDir(usersDir string, dryRun bool) error {
	matches, err := filepath.Glob(filepath.Join(usersDir, "*.yaml"))
	if err != nil {
		return err
	}
	mode := "APPLY"
	if dryRun {
		mode = "DRY-RUN"
	}
	mudlog.Info("Migration 0.14.0", "message", "Reclassifying player mutations onto the cluster graph", "mode", mode)

	for _, path := range matches {
		if filepath.Base(path) == "users.idx" {
			continue
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", path, err)
		}
		var userMap map[string]interface{}
		if err := yaml.Unmarshal(raw, &userMap); err != nil {
			mudlog.Warn("Migration 0.14.0", "file", filepath.Base(path), "error", err)
			continue
		}
		charData, ok := userMap["character"].(map[interface{}]interface{})
		if !ok {
			continue
		}
		if migrated, _ := charData["mutation_migrated"].(bool); migrated {
			continue // idempotency
		}

		cluster := ClassifyPlayer(extractSignals(userMap))
		seed := SeedForCluster(cluster)

		// Wipe the retired-41 entries but KEEP any surviving mutations the player
		// already held (the ~10 Center enablers that were not retired — e.g.
		// hollow-bones, tail), then merge the cluster seed on top (spec §7).
		newMuts := make(map[string]int)
		for id, rank := range yamlIntMap(charData["mutations"]) {
			if !retiredMutations[id] {
				newMuts[id] = rank
			}
		}
		for id, rank := range seed {
			newMuts[id] = rank
		}

		mudlog.Info("Migration 0.14.0", "file", filepath.Base(path), "class", cluster, "mutations", len(newMuts))
		if dryRun {
			continue
		}

		charData["mutations"] = newMuts
		charData["mutationprogress"] = 0.0
		charData["mutation_migrated"] = true

		out, err := yaml.Marshal(userMap)
		if err != nil {
			return fmt.Errorf("failed to marshal %s: %w", path, err)
		}
		if err := os.WriteFile(path, out, 0644); err != nil {
			return fmt.Errorf("failed to write %s: %w", path, err)
		}
	}
	return nil
}

// extractSignals reads classification inputs. role is a TOP-LEVEL UserRecord
// field; the rest live under character.
func extractSignals(userMap map[string]interface{}) PlayerSignals {
	s := PlayerSignals{}
	if r, ok := userMap["role"].(string); ok {
		s.Role = r
	}
	charData, ok := userMap["character"].(map[interface{}]interface{})
	if !ok {
		return s
	}
	skills := yamlIntMap(charData["skills"])
	s.Manifestation = skills["manifestation"]
	for skill, cluster := range combatClusterOf {
		if lv := skills[skill]; lv > s.TopCombat {
			s.TopCombat, s.TopCombatCluster = lv, cluster
		}
	}
	for _, cs := range craftSkills {
		s.CraftSum += skills[cs]
	}
	s.SpellbookDepth = len(yamlIntMap(charData["spellbook"]))
	if comps, ok := charData["companions"].([]interface{}); ok {
		s.Companions = len(comps)
	}
	if stats, ok := charData["stats"].(map[interface{}]interface{}); ok {
		if wil, ok := stats["willpower"].(map[interface{}]interface{}); ok {
			if t, ok := wil["training"].(int); ok {
				s.WillpowerTrain = t
			}
		}
	}
	return s
}

// yamlIntMap coerces a yaml.v2 map[interface{}]interface{} of {string: int}.
func yamlIntMap(v interface{}) map[string]int {
	out := map[string]int{}
	m, ok := v.(map[interface{}]interface{})
	if !ok {
		return out
	}
	for k, val := range m {
		ks, _ := k.(string)
		if iv, ok := val.(int); ok {
			out[ks] = iv
		}
	}
	return out
}
