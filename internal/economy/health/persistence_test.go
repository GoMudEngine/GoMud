package health_test

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/economy/health"
)

func TestPersistence_WriteThenLoad(t *testing.T) {
	dir := t.TempDir()
	in := health.Snapshot{UnixTs: 1746104400, Round: 12345, Manual: true, ManualLabel: "x"}

	if err := health.WriteSnapshotTo(dir, in); err != nil {
		t.Fatalf("WriteSnapshotTo: %v", err)
	}

	out, err := health.LoadSnapshotFrom(dir, in.UnixTs)
	if err != nil {
		t.Fatalf("LoadSnapshotFrom: %v", err)
	}
	if out.UnixTs != in.UnixTs || out.ManualLabel != "x" {
		t.Errorf("round-trip mismatch: %+v vs %+v", in, out)
	}
}

func TestPersistence_List(t *testing.T) {
	dir := t.TempDir()
	for _, ts := range []int64{1000, 2000, 3000} {
		health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: ts})
	}
	metas := health.ListSnapshotsFrom(dir)
	if len(metas) != 3 {
		t.Fatalf("got %d, want 3 metas", len(metas))
	}
	if metas[0].UnixTs != 3000 || metas[2].UnixTs != 1000 {
		t.Errorf("not sorted desc: %v", metas)
	}
}

func TestPersistence_Prune_KeepsManual(t *testing.T) {
	dir := t.TempDir()
	old := time.Now().Add(-100 * 24 * time.Hour).Unix()
	recent := time.Now().Unix()

	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: old, Manual: false})
	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: old + 1, Manual: true})
	health.WriteSnapshotTo(dir, health.Snapshot{UnixTs: recent, Manual: false})

	pruned, err := health.PruneSnapshotsIn(dir, 30) // 30-day retention
	if err != nil {
		t.Fatalf("Prune: %v", err)
	}
	if pruned != 1 {
		t.Errorf("pruned: got %d, want 1 (only the old non-manual)", pruned)
	}

	left, _ := os.ReadDir(dir)
	if len(left) != 2 {
		t.Errorf("files left: got %d, want 2 (manual + recent)", len(left))
	}
	// Confirm manual file survived.
	if _, err := os.Stat(filepath.Join(dir, health.SnapshotFilename(old+1))); err != nil {
		t.Errorf("manual snapshot was pruned: %v", err)
	}
}
