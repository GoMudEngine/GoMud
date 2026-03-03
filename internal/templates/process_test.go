package templates

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// emptyFS is a ReadFileFS that always returns "not found".
// Used when we want to bypass the registered-FS path and force OS fallback.
type emptyFS struct{}

func (emptyFS) Open(name string) (fs.File, error)          { return nil, fmt.Errorf("not found: %s", name) }
func (emptyFS) ReadFile(name string) ([]byte, error)        { return nil, fmt.Errorf("not found: %s", name) }

// dataFilesRoot is the relative path from this package dir to the default data files.
const dataFilesRoot = `../../_datafiles/world/default`

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)

	// Point config at the repo's data files so Process() can find templates.
	configs.AddOverlayOverrides(map[string]any{
		"FilePaths.DataFiles": dataFilesRoot,
	})

	// Register the data files directory as a readable filesystem.
	// Without this, readFile() returns (nil, nil) when fileSystems is empty,
	// causing every template lookup to match the .md variant with empty content.
	RegisterFS(os.DirFS(dataFilesRoot).(fs.ReadFileFS))

	code := m.Run()

	// Clean up registered filesystems.
	fileSystems = nil

	os.Exit(code)
}

// ─── Tier 2: Process() happy-path tests ───────────────────────────────────────

func TestProcess_FileNotFound(t *testing.T) {
	result, err := Process("nonexistent/template/that/does/not/exist", nil)
	assert.Error(t, err)
	assert.Contains(t, result, "[TEMPLATE READ ERROR")
}

func TestProcess_StaticTemplates(t *testing.T) {
	// These templates use no (or only builtin) data — nil works.
	static := []string{
		"help/about",
		"help/attack",
		"help/alias",
		"help/drop",
		"help/emote",
		"help/get",
		"help/look",
		"help/say",
		// login prompts use language.T which requires translation setup — skip.
		"admincommands/help/command.buff",
		"admincommands/help/command.spawn",
		"admincommands/shutdown-final",
	}

	for _, name := range static {
		t.Run(name, func(t *testing.T) {
			if !templateFileExists(name) {
				t.Skipf("template %q not found on disk — skipping", name)
			}
			result, err := Process(name, nil)
			require.NoError(t, err, "Process(%q) returned error", name)
			assert.NotContains(t, result, "[TEMPLATE ERROR]")
			assert.NotEmpty(t, result)
		})
	}
}

func TestProcess_GenericTemplates(t *testing.T) {
	// sunrise/sunset need .Day, .Month, .Year
	type dayData struct {
		Day   int
		Month int
		Year  int
	}
	data := dayData{Day: 15, Month: 3, Year: 42}

	generics := []string{"generic/sunrise", "generic/sunset"}
	for _, name := range generics {
		t.Run(name, func(t *testing.T) {
			if !templateFileExists(name) {
				t.Skipf("template %q not found — skipping", name)
			}
			result, err := Process(name, data)
			require.NoError(t, err, "Process(%q) error", name)
			assert.NotContains(t, result, "[TEMPLATE ERROR]")
			// The rendered output should contain the day number.
			assert.Contains(t, result, "15")
		})
	}
}

func TestProcess_TableTemplate(t *testing.T) {
	table := GetTable(
		"Test Table",
		[]string{"Name", "Value"},
		[][]string{
			{"alpha", "100"},
			{"beta", "200"},
		},
	)

	result, err := Process("tables/generic", table)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.Contains(t, result, "Test Table")
	assert.Contains(t, result, "alpha")
	assert.Contains(t, result, "200")
}

func TestProcess_DescriptionTemplates(t *testing.T) {
	// Stub struct matching the fields used in descriptions/room.template.
	// We use a local struct to avoid importing rooms (circular dependency).
	type roomDesc struct {
		Description    string
		IsNight        bool
		IsDark         bool
		RoomAlerts     []string
		TrackingString string
	}

	data := roomDesc{
		Description:    "A dusty stone corridor stretches into darkness.",
		IsNight:        false,
		IsDark:         false,
		RoomAlerts:     []string{},
		TrackingString: "",
	}

	result, err := Process("descriptions/room", data)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.Contains(t, result, "dusty stone corridor")

	// Test dark variant
	data.IsDark = true
	result2, err := Process("descriptions/room", data)
	require.NoError(t, err)
	assert.NotContains(t, result2, "[TEMPLATE ERROR]")
	assert.Contains(t, result2, "room-description-dark")
}

func TestProcess_CharacterSkillsTemplate(t *testing.T) {
	data := map[string]any{
		"SkillList": map[string]int{
			"melee":   2,
			"ranged":  0,
			"stealth": 4,
		},
		"SkillCooldowns": map[string]int{
			"melee":   0,
			"ranged":  0,
			"stealth": 0,
		},
		"TrainingPoints": 3,
	}

	result, err := Process("character/skills", data)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.Contains(t, result, "Skills")
	assert.Contains(t, result, "melee")
	// DOG skills soft-cap at 50; level 4 = apprentice, not MAXIMUM
	assert.Contains(t, result, "apprentice")
}

func TestProcess_CharacterConditionsTemplate(t *testing.T) {
	type condition struct {
		Name        string
		Description string
		RoundsLeft  int
		PermaBuff   bool
	}

	data := []condition{
		{Name: "Blessed", Description: "A warm glow surrounds you.", RoundsLeft: 10, PermaBuff: false},
		{Name: "NightVision", Description: "You can see in the dark.", RoundsLeft: 0, PermaBuff: true},
	}

	result, err := Process("character/conditions", data)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.Contains(t, result, "Conditions")
	assert.Contains(t, result, "Blessed")
	assert.Contains(t, result, "NightVision")
}

// ─── Screen Reader Path ───────────────────────────────────────────────────────

func TestProcess_ScreenReaderPath(t *testing.T) {
	// ForceScreenReaderUserId = -1 triggers the screen-reader fallback branch.
	// Most templates don't have a .screenreader.template variant, so it falls
	// through to the normal .template — but the screen-reader strip logic runs.
	if !templateFileExists("help/about") {
		t.Skip("help/about template not found")
	}

	result, err := Process("help/about", nil, ForceScreenReaderUserId)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.NotEmpty(t, result)
}

func TestProcess_ScreenReaderWithDedicatedTemplate(t *testing.T) {
	// sunrise has a .screenreader.template variant — test that it's preferred.
	if !templateFileExists("generic/sunrise") {
		t.Skip("generic/sunrise template not found")
	}

	type dayData struct {
		Day   int
		Month int
		Year  int
	}
	data := dayData{Day: 1, Month: 1, Year: 1}

	result, err := Process("generic/sunrise", data, ForceScreenReaderUserId)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.NotEmpty(t, result)
}

// ─── WithValidUserId (user cache miss path) ──────────────────────────────────

func TestProcess_WithValidUserId(t *testing.T) {
	// Seed a user so GetByUserId can find them.
	testUser := users.NewTestUser(999, "testplayer", "TestHero", 1)
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{999: testUser})
	defer cleanup()

	// Clear any cached template config for this userId.
	ClearTemplateConfigCache(999)

	if !templateFileExists("help/about") {
		t.Skip("help/about template not found")
	}

	result, err := Process("help/about", nil, 999)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.NotEmpty(t, result)
}

// ─── OS Filesystem Path ──────────────────────────────────────────────────────

func TestProcess_OSFilesystemPath(t *testing.T) {
	// Replace registered filesystems with an always-failing FS to force OS path.
	// Using nil would cause readFile to return (nil,nil) — a latent bug.
	origFS := fileSystems
	fileSystems = []fs.ReadFileFS{emptyFS{}}
	defer func() { fileSystems = origFS }()

	if !templateFileExists("help/about") {
		t.Skip("help/about template not found")
	}

	result, err := Process("help/about", nil)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.NotEmpty(t, result)
}

func TestProcess_OSFilesystemScreenReader(t *testing.T) {
	// Test the screen-reader strip branch in the OS filesystem path (lines 241-248).
	origFS := fileSystems
	fileSystems = []fs.ReadFileFS{emptyFS{}}
	defer func() { fileSystems = origFS }()

	if !templateFileExists("help/about") {
		t.Skip("help/about template not found")
	}

	result, err := Process("help/about", nil, ForceScreenReaderUserId)
	require.NoError(t, err)
	assert.NotContains(t, result, "[TEMPLATE ERROR]")
	assert.NotEmpty(t, result)
}

// ─── ProcessText ──────────────────────────────────────────────────────────────

func TestProcessText_Basic(t *testing.T) {
	// Simple template with variable substitution.
	result, err := ProcessText("Hello, {{ .Name }}!", map[string]any{"Name": "World"})
	require.NoError(t, err)
	assert.Equal(t, "Hello, World!", result)

	// With funcMap functions available (pad, repeat, etc.).
	result2, err := ProcessText(`{{ repeat "*" 5 }}`, nil)
	require.NoError(t, err)
	assert.Equal(t, "*****", result2)
}

func TestProcessText_Errors(t *testing.T) {
	// Parse error: invalid template syntax.
	result, err := ProcessText("{{ .Bad", nil)
	assert.Error(t, err)
	assert.Equal(t, "{{ .Bad", result) // returns raw text on parse error

	// Execute error: call a method on a nil value.
	result2, err := ProcessText("{{ call .Fn }}", map[string]any{"Fn": (func() string)(nil)})
	assert.Error(t, err)
	assert.Equal(t, "[TEMPLATE TEXT ERROR]", result2)
}

func TestProcessText_AnsiFlags(t *testing.T) {
	input := `<ansi fg="red">hello</ansi>`

	// AnsiTagsParse should process the ansi tags.
	result, err := ProcessText(input, nil, AnsiTagsParse)
	require.NoError(t, err)
	// After parsing, ANSI escape codes should be present (not the raw tags).
	assert.NotContains(t, result, `<ansi`)
}

// ─── GetTable ─────────────────────────────────────────────────────────────────

func TestGetTable(t *testing.T) {
	table := GetTable(
		"Inventory",
		[]string{"Item", "Qty"},
		[][]string{
			{"Sword", "1"},
			{"Shield", "2"},
			{"Health Potion", "10"},
		},
	)

	assert.Equal(t, "Inventory", table.Title)
	assert.Equal(t, 2, table.ColumnCount)
	assert.Len(t, table.Rows, 3)

	// Column widths should accommodate the longest entry.
	// "Health Potion" is 13 chars — the widest in column 0.
	assert.GreaterOrEqual(t, table.ColumnWidths[0], 13)

	// GetHeaderCell and GetCell should not panic.
	hdr := table.GetHeaderCell(0)
	assert.Contains(t, hdr, "Item")

	cell := table.GetCell(2, 0)
	assert.Contains(t, cell, "Health Potion")
}

// ─── Tier 1: Parse ALL template files ─────────────────────────────────────────

func TestTier1_ParseAllTemplates(t *testing.T) {
	templateRoot := filepath.Join(dataFilesRoot, "templates")

	var parsed int
	var errors []string

	err := filepath.Walk(templateRoot, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		if ext != ".template" && ext != ".md" {
			return nil
		}

		content, readErr := os.ReadFile(path)
		if readErr != nil {
			errors = append(errors, path+": "+readErr.Error())
			return nil
		}

		_, parseErr := template.New(path).Funcs(funcMap).Parse(string(content))
		if parseErr != nil {
			errors = append(errors, path+": "+parseErr.Error())
			return nil
		}

		parsed++
		return nil
	})

	require.NoError(t, err, "filepath.Walk failed")

	// We expect 200+ template files.
	assert.GreaterOrEqual(t, parsed, 200,
		"expected at least 200 templates to parse, got %d", parsed)

	// Every template should parse without error.
	assert.Empty(t, errors,
		"template parse errors:\n  %s", strings.Join(errors, "\n  "))

	t.Logf("Successfully parsed %d template files with zero errors", parsed)
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

// templateFileExists checks whether a template file is present on disk.
func templateFileExists(name string) bool {
	for _, ext := range []string{".template", ".md"} {
		path := filepath.Join(dataFilesRoot, "templates", filepath.FromSlash(name)+ext)
		if _, err := os.Stat(path); err == nil {
			return true
		}
	}
	return false
}
