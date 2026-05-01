package health

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"gopkg.in/yaml.v2"
)

// SnapshotFilename returns the canonical filename for a snapshot at
// the given unix-ts. Exposed for test parity.
func SnapshotFilename(unixTs int64) string {
	return fmt.Sprintf("%d.yaml", unixTs)
}

// snapshotDir returns the production snapshot directory. Tests use
// the *To variants below to redirect storage to t.TempDir().
func snapshotDir() string {
	return filepath.Join(configs.GetFilePathsConfig().DataFiles.String(), "economy", "snapshots")
}

// WriteSnapshot writes to the production directory.
func WriteSnapshot(s Snapshot) error { return WriteSnapshotTo(snapshotDir(), s) }

// LoadSnapshot reads from the production directory.
func LoadSnapshot(unixTs int64) (*Snapshot, error) { return LoadSnapshotFrom(snapshotDir(), unixTs) }

// ListSnapshots lists from the production directory.
func ListSnapshots() []SnapshotMeta { return ListSnapshotsFrom(snapshotDir()) }

// PruneSnapshots prunes from the production directory.
func PruneSnapshots(retentionDays int) (int, error) {
	return PruneSnapshotsIn(snapshotDir(), retentionDays)
}

// WriteSnapshotTo writes a snapshot YAML to the given directory.
// Creates the directory if missing. Filename is "{unix_ts}.yaml".
func WriteSnapshotTo(dir string, s Snapshot) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %q: %w", dir, err)
	}
	data, err := yaml.Marshal(s)
	if err != nil {
		return fmt.Errorf("marshal: %w", err)
	}
	path := filepath.Join(dir, SnapshotFilename(s.UnixTs))
	return os.WriteFile(path, data, 0o644)
}

// LoadSnapshotFrom reads "{unix_ts}.yaml" from dir.
func LoadSnapshotFrom(dir string, unixTs int64) (*Snapshot, error) {
	path := filepath.Join(dir, SnapshotFilename(unixTs))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var s Snapshot
	if err := yaml.Unmarshal(data, &s); err != nil {
		return nil, fmt.Errorf("unmarshal %q: %w", path, err)
	}
	return &s, nil
}

// SnapshotMeta is a lightweight directory entry. Used for fast listing
// without parsing every snapshot.
type SnapshotMeta struct {
	UnixTs      int64
	Manual      bool
	ManualLabel string
}

// manualPeek is the minimal struct needed to read the Manual and
// ManualLabel fields from a snapshot YAML without deserializing the
// entire payload (shops + caravans + foragers arrays). Replaces the
// previous LoadSnapshotFrom call in ListSnapshotsFrom, which caused
// ~36MB of allocations + 720 full unmarshals per dashboard fetch at
// 30-day retention.
type manualPeek struct {
	Manual      bool   `yaml:"manual"`
	ManualLabel string `yaml:"manual_label,omitempty"`
}

// ListSnapshotsFrom returns metas sorted by timestamp descending. The
// Manual + ManualLabel fields are obtained via a lightweight peek into
// each YAML (manualPeek), reading only those two keys rather than
// deserializing the full Snapshot.
func ListSnapshotsFrom(dir string) []SnapshotMeta {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	out := make([]SnapshotMeta, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".yaml") {
			continue
		}
		base := strings.TrimSuffix(e.Name(), ".yaml")
		ts, err := strconv.ParseInt(base, 10, 64)
		if err != nil {
			continue
		}
		meta := SnapshotMeta{UnixTs: ts}
		// Peek for manual flag — reads only manual + manual_label keys.
		if data, err := os.ReadFile(filepath.Join(dir, e.Name())); err == nil {
			var peek manualPeek
			if err := yaml.Unmarshal(data, &peek); err == nil {
				meta.Manual = peek.Manual
				meta.ManualLabel = peek.ManualLabel
			}
		}
		out = append(out, meta)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].UnixTs > out[j].UnixTs })
	return out
}

// PruneSnapshotsIn deletes auto-snapshots in dir older than
// retentionDays. Manual snapshots are never pruned. Returns the number
// of files deleted.
func PruneSnapshotsIn(dir string, retentionDays int) (int, error) {
	if retentionDays <= 0 {
		return 0, nil
	}
	cutoff := time.Now().Add(-time.Duration(retentionDays) * 24 * time.Hour).Unix()
	deleted := 0
	for _, meta := range ListSnapshotsFrom(dir) {
		if meta.Manual {
			continue
		}
		if meta.UnixTs >= cutoff {
			continue
		}
		path := filepath.Join(dir, SnapshotFilename(meta.UnixTs))
		if err := os.Remove(path); err != nil {
			return deleted, fmt.Errorf("remove %q: %w", path, err)
		}
		deleted++
	}
	return deleted, nil
}
