package users

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// syncTestDir writes count realistic user files and returns a synced index
// for them.
func syncTestDir(t testing.TB, count int) (string, *UserIndex) {
	t.Helper()

	dir := t.TempDir()
	for i := 1; i <= count; i++ {
		writeScanTestUser(t, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 1)
	}

	idx := newScanTestIndex(dir)
	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("initial sync failed: %v", err)
	}
	if changed != count {
		t.Fatalf("expected initial sync to parse %d files, got %d", count, changed)
	}
	return dir, idx
}

// TestSyncNoChangesSkipsWrite verifies a second sync over an unchanged
// directory parses nothing and leaves the index file untouched.
func TestSyncNoChangesSkipsWrite(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir, idx := syncTestDir(t, 10)

	before, err := os.Stat(idx.Filename)
	if err != nil {
		t.Fatal(err)
	}

	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("second sync failed: %v", err)
	}
	if changed != 0 {
		t.Errorf("expected 0 changed on unchanged directory, got %d", changed)
	}

	after, err := os.Stat(idx.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if !after.ModTime().Equal(before.ModTime()) || after.Size() != before.Size() {
		t.Error("index file was rewritten despite no changes")
	}
}

// TestSyncDetectsChangedFile verifies a modified user file is re-parsed and
// its index entry updated, without disturbing the others.
func TestSyncDetectsChangedFile(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir, idx := syncTestDir(t, 10)

	// Rename the user inside file 3. The content length changes, so the
	// size comparison alone must catch it even with coarse mtimes.
	writeScanTestUser(t, dir, 3, `Renamed_User_Three`, `Hero_3_Reborn`, 2)

	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if changed != 1 {
		t.Errorf("expected 1 changed record, got %d", changed)
	}

	if userId, found := idx.FindByUsername(`renamed_user_three`); !found || userId != 3 {
		t.Errorf("expected renamed user findable with userId 3, got %d, found=%v", userId, found)
	}
	if _, found := idx.FindByUsername(`user_3`); found {
		t.Error("old username should be gone after re-parse")
	}
	if userId, found := idx.FindByUsername(`user_7`); !found || userId != 7 {
		t.Errorf("untouched user_7 should remain, got %d, found=%v", userId, found)
	}
}

// TestSyncDetectsNewAndDeleted verifies new files are indexed and records
// for deleted files are dropped.
func TestSyncDetectsNewAndDeleted(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir, idx := syncTestDir(t, 10)

	writeScanTestUser(t, dir, 11, `User_11`, `Hero_11`, 1)
	if err := os.Remove(filepath.Join(dir, `2.yaml`)); err != nil {
		t.Fatal(err)
	}

	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if changed != 2 {
		t.Errorf("expected 2 changed records (one new, one dropped), got %d", changed)
	}

	if userId, found := idx.FindByUsername(`user_11`); !found || userId != 11 {
		t.Errorf("expected new user_11 indexed, got %d, found=%v", userId, found)
	}
	if _, found := idx.FindByUserId(2); found {
		t.Error("deleted user 2 should be gone from the index")
	}
	if highest := idx.GetHighestUserId(); highest != 11 {
		t.Errorf("expected highest userId 11, got %d", highest)
	}
}

// TestSyncUpgradesOldFormatIndex verifies an index in the previous record
// format triggers a full rebuild into the current format.
func TestSyncUpgradesOldFormatIndex(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir := t.TempDir()
	for i := 1; i <= 5; i++ {
		writeScanTestUser(t, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 1)
	}

	// Hand-craft a version 2 index: 89-byte records, no character name.
	header := `VERSION=2,RECORDCOUNT=1,RECORDSIZE=89,CHECKSUM=12345`
	header = header + strings.Repeat(` `, FixedHeaderTotalLength-1-len(header)) + "\n"
	record := make([]byte, 89)
	copy(record[:80], `staleuser`)
	record[80] = 42 // userid 42, little-endian low byte
	record[88] = IndexLineTerminatorV1
	idxPath := filepath.Join(dir, `users.idx`)
	if err := os.WriteFile(idxPath, append([]byte(header), record...), 0644); err != nil {
		t.Fatal(err)
	}

	idx := newScanTestIndex(dir)
	idx.metaData = idx.getMetaDataFromFile()
	idx.loadRecords()

	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if changed != 5 {
		t.Errorf("expected full rebuild of 5 users after format upgrade, got %d", changed)
	}

	if _, found := idx.FindByUsername(`staleuser`); found {
		t.Error("record from the old-format index should not survive the upgrade")
	}
	if meta := idx.GetMetaData(); meta.IndexVersion != IndexVersion || meta.RecordSize != IndexRecordSizeV3 {
		t.Errorf("expected index upgraded to version %d with record size %d, got %+v", IndexVersion, IndexRecordSizeV3, meta)
	}
}

// TestRebuildFromIndexReadsNoFiles verifies the character index can be
// rebuilt purely from index records: after deleting every user file, the
// names must still resolve.
func TestRebuildFromIndexReadsNoFiles(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir, idx := syncTestDir(t, 5)

	files, err := filepath.Glob(filepath.Join(dir, `*.yaml`))
	if err != nil {
		t.Fatal(err)
	}
	for _, f := range files {
		if err := os.Remove(f); err != nil {
			t.Fatal(err)
		}
	}

	ci := freshCharacterIndex()
	orig := characterIndex
	characterIndex = ci
	defer func() { characterIndex = orig }()

	ci.RebuildFromIndex(idx)

	if ci.Len() != 5 {
		t.Fatalf("expected 5 characters from index records, got %d", ci.Len())
	}
	if userId, found := ci.Find(`Hero_4`); !found || userId != 4 {
		t.Errorf("expected Hero_4 to resolve to userId 4, got %d, found=%v", userId, found)
	}
}

// TestSyncCompletesAddUserStub verifies the runtime AddUser record (zero
// mtime and size) is re-parsed and completed by the next sync.
func TestSyncCompletesAddUserStub(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir, idx := syncTestDir(t, 3)

	// Simulate a runtime registration: index entry first, file second.
	if err := idx.AddUser(4, `Newcomer`); err != nil {
		t.Fatal(err)
	}
	writeScanTestUser(t, dir, 4, `Newcomer`, `Fresh_Hero`, 1)

	changed, err := idx.syncWithDir(dir)
	if err != nil {
		t.Fatalf("sync failed: %v", err)
	}
	if changed != 1 {
		t.Errorf("expected exactly the stub record to be re-parsed, got %d changed", changed)
	}

	ci := freshCharacterIndex()
	ci.RebuildFromIndex(idx)
	if userId, found := ci.Find(`Fresh_Hero`); !found || userId != 4 {
		t.Errorf("expected completed record to carry character name, got %d, found=%v", userId, found)
	}
}

// BenchmarkSyncNoChanges measures the steady-state startup cost: a sync
// over a directory where nothing changed since the last index write.
func BenchmarkSyncNoChanges(b *testing.B) {
	mudlog.SetupLogger(nil, "", "", false)

	for _, userCt := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf(`%d_users`, userCt), func(b *testing.B) {
			dir := b.TempDir()
			for i := 1; i <= userCt; i++ {
				writeScanTestUser(b, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 8)
			}
			idx := newScanTestIndex(dir)
			if _, err := idx.syncWithDir(dir); err != nil {
				b.Fatal(err)
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				changed, err := idx.syncWithDir(dir)
				if err != nil {
					b.Fatal(err)
				}
				if changed != 0 {
					b.Fatalf("expected steady state, got %d changed", changed)
				}
			}
		})
	}
}

// BenchmarkSyncWithChurn measures a sync where 50 of 1000 user files
// changed since the last index write - the realistic restart shape.
func BenchmarkSyncWithChurn(b *testing.B) {
	mudlog.SetupLogger(nil, "", "", false)

	dir := b.TempDir()
	for i := 1; i <= 1000; i++ {
		writeScanTestUser(b, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 8)
	}
	idx := newScanTestIndex(dir)
	if _, err := idx.syncWithDir(dir); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		for u := 1; u <= 50; u++ {
			writeScanTestUser(b, dir, u, fmt.Sprintf(`User_%d`, u), fmt.Sprintf(`Hero_%d_v%d`, u, i), 8)
		}
		b.StartTimer()

		changed, err := idx.syncWithDir(dir)
		if err != nil {
			b.Fatal(err)
		}
		if changed != 50 {
			b.Fatalf("expected 50 changed, got %d", changed)
		}
	}
}
