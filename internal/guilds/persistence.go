package guilds

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

var dataDirOverride string // test hook; empty = use config

func guildsDir() string {
	if dataDirOverride != "" {
		return dataDirOverride
	}
	return filepath.FromSlash(configs.GetFilePathsConfig().DataFiles.String() + `/guilds`)
}

// SetDataDirForTest points persistence at dir and returns a restore func.
func SetDataDirForTest(dir string) func() {
	prev := dataDirOverride
	dataDirOverride = dir
	return func() { dataDirOverride = prev }
}

func guildPath(tag string) string {
	return filepath.Join(guildsDir(), strings.ToLower(tag)+".yaml")
}

// Save writes a guild to disk.
func Save(g *Guild) error {
	if err := os.MkdirAll(guildsDir(), 0755); err != nil {
		return fmt.Errorf("guilds.Save: mkdir: %w", err)
	}
	b, err := yaml.Marshal(g)
	if err != nil {
		return fmt.Errorf("guilds.Save: marshal: %w", err)
	}
	return os.WriteFile(guildPath(g.Tag), b, 0644)
}

// Delete removes a guild from the registry and disk.
func Delete(tag string) {
	registryMu.Lock()
	if g, ok := byTag[strings.ToUpper(tag)]; ok {
		for _, m := range g.Members {
			delete(byUser, m.UserId)
		}
		delete(byTag, strings.ToUpper(tag))
	}
	registryMu.Unlock()
	_ = os.Remove(guildPath(tag))
}

// LoadDataFiles loads all guild files into the registry at boot. A malformed
// file is logged + skipped (runtime-generated state must not crash the server).
func LoadDataFiles() {
	start := time.Now()
	resetRegistry()
	dir := guildsDir()
	entries, err := os.ReadDir(dir)
	if err != nil {
		mudlog.Info("guilds.LoadDataFiles()", "loadedCount", 0, "note", "no guilds dir")
		return
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		b, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
		if rerr != nil {
			mudlog.Error("guilds.LoadDataFiles", "file", e.Name(), "error", rerr.Error())
			continue
		}
		var g Guild
		if uerr := yaml.Unmarshal(b, &g); uerr != nil {
			mudlog.Error("guilds.LoadDataFiles", "file", e.Name(), "error", uerr.Error())
			continue
		}
		// Retroactive ANSI hygiene (2026-08-03): guild files written before
		// write-side escaping shipped can still hold raw <ansi> payloads in
		// their player-authored fields. Escaping is idempotent, so scrubbing
		// on every load is safe, and the next save persists the clean form.
		g.Name = util.EscapeAnsiTags(g.Name)
		g.Motd = util.EscapeAnsiTags(g.Motd)
		for rank, title := range g.RankTitles {
			g.RankTitles[rank] = util.EscapeAnsiTags(title)
		}
		indexGuild(&g)
		n++
	}
	mudlog.Info("guilds.LoadDataFiles()", "loadedCount", n, "Time Taken", time.Since(start))
}
