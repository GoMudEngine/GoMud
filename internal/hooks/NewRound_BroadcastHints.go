package hooks

import (
	"os"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
)

// Broadcasts periodic gameplay hints/tips to players who haven't opted out.
// Fires every hintIntervalRounds rounds (~5 minutes at default 4s rounds).

const (
	hintIntervalRounds = 75 // ~5 minutes at 4-second rounds
	hintConfigKey      = `hints`
)

var (
	hintMessages []string
	hintIndex    int
	hintOnce     sync.Once
)

type hintsFile struct {
	Hints []string `yaml:"hints"`
}

func loadHints() {
	path := string(configs.GetFilePathsConfig().DataFiles) + `/hints.yaml`

	data, err := os.ReadFile(path)
	if err != nil {
		mudlog.Error("BroadcastHints", "error", "failed to load hints.yaml", "path", path, "err", err)
		return
	}

	var hf hintsFile
	if err := yaml.Unmarshal(data, &hf); err != nil {
		mudlog.Error("BroadcastHints", "error", "failed to parse hints.yaml", "err", err)
		return
	}

	hintMessages = hf.Hints
	mudlog.Info("...LoadHints()", "loadedCount", len(hintMessages))
}

func BroadcastHints(e events.Event) events.ListenerReturn {
	evt := e.(events.NewRound)

	if evt.RoundNumber%hintIntervalRounds != 0 {
		return events.Continue
	}

	hintOnce.Do(loadHints)

	if len(hintMessages) == 0 {
		return events.Continue
	}

	// Pick the next hint in rotation
	tip := hintMessages[hintIndex%len(hintMessages)]
	hintIndex++

	prefix := `<ansi fg="cyan-bold">[Tip]</ansi> `
	fullText := prefix + tip

	// Send to each active user who hasn't opted out
	for _, u := range users.GetAllActiveUsers() {
		// Skip deafened players
		if u.Deafened {
			continue
		}

		// Check opt-out: hints default to ON (nil = on)
		hintsOpt := u.GetConfigOption(hintConfigKey)
		if b, ok := hintsOpt.(bool); ok && !b {
			continue
		}

		u.SendText(fullText)
	}

	// Also fire a Communication event for webclient comms window
	events.AddToQueue(events.Communication{
		CommType: "broadcast",
		Name:     "Tip",
		Message:  tip,
	})

	return events.Continue
}
