package guilds

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// Guild files written before write-side ANSI escaping shipped can hold raw
// <ansi> payloads in player-authored fields. LoadDataFiles must scrub them
// (idempotently) so the stored payload is neutralized on the next save.
func TestLoadDataFiles_ScrubsStoredAnsiPayloads(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)
	defer SetDataDirForTest(t.TempDir())()

	dir := guildsDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := `tag: xyz
name: '<ansi fg="red">Evil</ansi> Guild'
leaderuserid: 1
motd: '</ansi><ansi bg="black">payload MOTD'
ranktitles:
  leader: '<ansi fg="magenta">Overlord</ansi>'
`
	if err := os.WriteFile(filepath.Join(dir, "xyz.yaml"), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	LoadDataFiles()

	g, ok := Get("xyz")
	if !ok || g == nil {
		t.Fatal("guild xyz did not load")
	}
	for field, val := range map[string]string{
		"name":              g.Name,
		"motd":              g.Motd,
		"ranktitles.leader": g.RankTitles[RankLeader],
	} {
		if strings.Contains(val, "<ansi") || strings.Contains(val, "</ansi>") {
			t.Errorf("%s still holds a live ansi tag after load: %q", field, val)
		}
	}
}
