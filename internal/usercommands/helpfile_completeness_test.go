package usercommands

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// helpDataRoot walks up from the test working directory to find the
// _datafiles folder and returns the dogmud world root. It validates that
// the found _datafiles contains world/dogmud to avoid false positives.
func helpDataRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("cannot get working directory: %v", err)
	}
	for {
		candidate := filepath.Join(dir, "_datafiles")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			// Verify this is the correct _datafiles by checking for world/dogmud
			worldDogmud := filepath.Join(candidate, "world", "dogmud")
			if info, err := os.Stat(worldDogmud); err == nil && info.IsDir() {
				return worldDogmud
			}
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("cannot find _datafiles/world/dogmud directory from %s", dir)
		}
		dir = parent
	}
}

// helpFileExistsAt checks whether a file exists at one of the given paths.
// Returns true if any of them exists.
func helpFileExistsAt(paths ...string) bool {
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// commandHelpAliases maps a command name to another command name whose
// help file it shares. Used when two commands map to the same handler
// function and a single help template covers both.
var commandHelpAliases = map[string]string{
	"companions": "companion",
	"stomp":      "kick",
	"knee":       "kick",
	"tailsweep":  "trip",
	"rep":        "report",
}

// commandHelpSkip lists internal/debug commands that don't need
// player-facing help files. These are not typed directly by players
// (e.g. context dispatchers, no-ops, account-creation flow commands).
var commandHelpSkip = map[string]bool{
	"default":   true, // context-sensitive dispatcher
	"noop":      true, // does nothing
	"start":     true, // account creation only
	"zombieact": true, // internal zombie AI
	"print":     true, // debug echo
	"printline": true, // debug echo
}

// TestHelpFileCompleteness_Commands ensures every registered user command
// has a matching help template. Regular commands live at
// help/<name>.template. Admin commands live at
// admincommands/help/command.<name>.template (or .md).
func TestHelpFileCompleteness_Commands(t *testing.T) {
	root := helpDataRoot(t)
	userHelpDir := filepath.Join(root, "templates", "help")
	adminHelpDir := filepath.Join(root, "templates", "admincommands", "help")

	var missing []string
	for name, info := range userCommands {
		if commandHelpSkip[name] {
			continue
		}
		target := name
		if alias, ok := commandHelpAliases[name]; ok {
			target = alias
		}

		if info.AdminOnly {
			tmpl := filepath.Join(adminHelpDir, "command."+target+".template")
			md := filepath.Join(adminHelpDir, "command."+target+".md")
			if !helpFileExistsAt(tmpl, md) {
				missing = append(missing, name+" (admin) — expected "+
					filepath.Join("admincommands", "help", "command."+target+".template"))
			}
		} else {
			tmpl := filepath.Join(userHelpDir, target+".template")
			md := filepath.Join(userHelpDir, target+".md")
			if !helpFileExistsAt(tmpl, md) {
				missing = append(missing, name+" (user) — expected "+
					filepath.Join("help", target+".template"))
			}
		}
	}

	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("commands missing help files (%d):\n  %s\n\n"+
			"Add a template (a short stub is fine) or add the command to commandHelpSkip "+
			"if it's truly internal, or commandHelpAliases if it shares a help file with "+
			"another command.",
			len(missing), strings.Join(missing, "\n  "))
	}
}
