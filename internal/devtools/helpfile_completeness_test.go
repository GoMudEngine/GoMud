package devtools

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// dataRoot returns the path to the dogmud data files relative to the repo root.
// Tests run from the package directory, so we walk up to find _datafiles.
func dataRoot(t *testing.T) string {
	t.Helper()
	// Walk up from the test's working directory until we find _datafiles
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "_datafiles")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return filepath.Join(candidate, "world", "dogmud")
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("cannot find _datafiles directory from %s", dir)
		}
		dir = parent
	}
}

// listYAMLBasenames returns all .yaml filenames (without extension) in a directory.
func listYAMLBasenames(t *testing.T, dir string) []string {
	t.Helper()
	var names []string
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".yaml") {
			base := strings.TrimSuffix(info.Name(), ".yaml")
			names = append(names, base)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("cannot walk directory %s: %v", dir, err)
	}
	return names
}

// helpFileExists checks if a help template exists for the given name.
func helpFileExists(root string, name string) bool {
	path := filepath.Join(root, "templates", "help", name+".template")
	_, err := os.Stat(path)
	return err == nil
}

// TestHelpFileCompleteness_Spells ensures every spell YAML has a matching help file.
func TestHelpFileCompleteness_Spells(t *testing.T) {
	root := dataRoot(t)
	spellDir := filepath.Join(root, "spells")

	spells := listYAMLBasenames(t, spellDir)
	if len(spells) == 0 {
		t.Fatal("no spell YAML files found — check path")
	}

	missing := []string{}
	for _, spell := range spells {
		if !helpFileExists(root, spell) {
			missing = append(missing, spell)
		}
	}

	if len(missing) > 0 {
		t.Errorf("spells missing help files (%d):\n  %s\n\nAdd help templates to _datafiles/world/dogmud/templates/help/ for each.",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestHelpFileCompleteness_Recipes ensures every recipe YAML has a matching help file.
func TestHelpFileCompleteness_Recipes(t *testing.T) {
	root := dataRoot(t)
	recipeDir := filepath.Join(root, "recipes")

	recipes := listYAMLBasenames(t, recipeDir)
	if len(recipes) == 0 {
		t.Fatal("no recipe YAML files found — check path")
	}

	missing := []string{}
	for _, recipe := range recipes {
		if !helpFileExists(root, recipe) {
			missing = append(missing, recipe)
		}
	}

	if len(missing) > 0 {
		t.Errorf("recipes missing help files (%d):\n  %s\n\nAdd help templates to _datafiles/world/dogmud/templates/help/ for each.",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestHelpFileCompleteness_Mutations ensures every mutation YAML has a matching help file.
func TestHelpFileCompleteness_Mutations(t *testing.T) {
	root := dataRoot(t)
	mutationDir := filepath.Join(root, "mutations")

	mutations := listYAMLBasenames(t, mutationDir)
	if len(mutations) == 0 {
		t.Fatal("no mutation YAML files found — check path")
	}

	missing := []string{}
	for _, mutation := range mutations {
		if !helpFileExists(root, mutation) {
			missing = append(missing, mutation)
		}
	}

	if len(missing) > 0 {
		t.Errorf("mutations missing help files (%d):\n  %s\n\nAdd help templates to _datafiles/world/dogmud/templates/help/ for each.",
			len(missing), strings.Join(missing, "\n  "))
	}
}

// TestHelpFileCompleteness_Skills ensures core skills have help files.
func TestHelpFileCompleteness_Skills(t *testing.T) {
	root := dataRoot(t)

	// All skills that should have help files
	requiredSkills := []string{
		"weapon-combat",
		"unarmed-combat",
		"ranged-combat",
		"spellcasting",
		"rhetoric",
		"search",
		"skullduggery",
		"map",
		"foraging",
		"blacksmithing",
		"alchemy",
		"tailoring",
		"cooking",
		"jewelcrafting",
		"enchanting",
	}

	missing := []string{}
	for _, skill := range requiredSkills {
		if !helpFileExists(root, skill) {
			missing = append(missing, skill)
		}
	}

	if len(missing) > 0 {
		t.Errorf("skills missing help files (%d):\n  %s\n\nAdd help templates to _datafiles/world/dogmud/templates/help/ for each.",
			len(missing), strings.Join(missing, "\n  "))
	}
}
