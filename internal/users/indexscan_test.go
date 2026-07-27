package users

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// writeScanTestUser writes a user file shaped like a real record: the index
// fields up top and an embedded character block padded out so the scan pays
// a realistic lexing cost. Production user files run 10-20KB; padKB controls
// how close a test file gets to that.
func writeScanTestUser(t testing.TB, dir string, userId int, username string, charName string, padKB int) {
	t.Helper()

	var sb strings.Builder
	fmt.Fprintf(&sb, "userid: %d\n", userId)
	fmt.Fprintf(&sb, "username: %s\n", username)
	sb.WriteString("password: 52c69134f0185daafe43fa511b6d1db16e59404a7992aa0d9bfa0bea05d592a9\n")
	sb.WriteString("joined: 2026-01-07T13:52:18.427407+02:00\n")
	if charName != `` {
		fmt.Fprintf(&sb, "character:\n    name: %s\n    roomid: 1\n    level: 5\n    experience: 1234\n", charName)
		for i := 0; sb.Len() < padKB*1024; i++ {
			fmt.Fprintf(&sb, "    itemfiller%d: some padding value that stands in for inventory and buffs %d\n", i, i)
		}
	}

	if err := os.WriteFile(filepath.Join(dir, fmt.Sprintf(`%d.yaml`, userId)), []byte(sb.String()), 0644); err != nil {
		t.Fatal(err)
	}
}

func newScanTestIndex(dir string) *UserIndex {
	return &UserIndex{
		Filename:   filepath.Join(dir, `users.idx`),
		byUsername: make(map[string]int64),
		byUserId:   make(map[int64]string),
	}
}

// TestScanUserFilesInDir verifies the scan picks up every valid user file,
// including old-format username.yaml files, while skipping alt files,
// malformed yaml, and files without a userid.
func TestScanUserFilesInDir(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir := t.TempDir()
	for i := 1; i <= 5; i++ {
		writeScanTestUser(t, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 1)
	}

	// A user file in the old username.yaml naming, still valid by content.
	if err := os.WriteFile(filepath.Join(dir, `legacy.yaml`), []byte("userid: 77\nusername: legacy\ncharacter:\n    name: Oldtimer\n"), 0644); err != nil {
		t.Fatal(err)
	}

	junkFiles := map[string]string{
		`3.alts.yaml`: "alts:\n- name: x\n",
		`broken.yaml`: "userid: [not closed\n",
		`50.yaml`:     "username: idless\n",
		`notes.txt`:   `hello`,
	}
	for name, content := range junkFiles {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	scan := scanUserFilesInDir(dir)

	if len(scan) != 6 {
		t.Fatalf("expected 6 scanned users, got %d", len(scan))
	}

	byId := make(map[int]UserFileScan, len(scan))
	for _, s := range scan {
		byId[s.UserId] = s
	}

	if s, ok := byId[3]; !ok || s.Username != `User_3` || s.CharacterName != `Hero_3` {
		t.Errorf("expected userId 3 with username 'User_3' and character 'Hero_3', got %+v", s)
	}
	if s, ok := byId[77]; !ok || s.Username != `legacy` || s.CharacterName != `Oldtimer` {
		t.Errorf("expected old-format file to scan as userId 77 'legacy'/'Oldtimer', got %+v", s)
	}
}

// TestRebuildFromScanRoundTrip verifies applyScan builds correct lookup
// state, persists it atomically, and that a fresh UserIndex reads back the
// identical records and checksum.
func TestRebuildFromScanRoundTrip(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir := t.TempDir()
	for i := 1; i <= 10; i++ {
		writeScanTestUser(t, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 1)
	}

	checksum, err := computeDirChecksum(dir)
	if err != nil {
		t.Fatal(err)
	}

	idx := newScanTestIndex(dir)
	if err := idx.applyScan(scanUserFilesInDir(dir), checksum); err != nil {
		t.Fatalf("applyScan failed: %v", err)
	}

	if userId, found := idx.FindByUsername(`user_7`); !found || userId != 7 {
		t.Errorf("expected to find 'user_7' with userId 7, got %d, found=%v", userId, found)
	}
	if highest := idx.GetHighestUserId(); highest != 10 {
		t.Errorf("expected highest userId 10, got %d", highest)
	}

	reloaded := newScanTestIndex(dir)
	reloaded.metaData = reloaded.getMetaDataFromFile()
	reloaded.loadRecords()

	if reloaded.metaData.RecordCount != 10 {
		t.Fatalf("expected 10 records after reload, got %d", reloaded.metaData.RecordCount)
	}
	if reloaded.metaData.Checksum != checksum {
		t.Errorf("expected persisted checksum %d, got %d", checksum, reloaded.metaData.Checksum)
	}
	for i := 1; i <= 10; i++ {
		username, found := reloaded.FindByUserId(i)
		if !found || username != fmt.Sprintf(`user_%d`, i) {
			t.Errorf("expected 'user_%d' for userId %d, got '%s', found=%v", i, i, username, found)
		}
	}
}

// TestRebuildFromScanReplacesStaleIndex verifies index entries with no
// matching user file do not survive a rebuild.
func TestRebuildFromScanReplacesStaleIndex(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	dir := t.TempDir()
	for i := 1; i <= 5; i++ {
		writeScanTestUser(t, dir, i, fmt.Sprintf(`User_%d`, i), ``, 0)
	}

	idx := newScanTestIndex(dir)
	if err := idx.Create(); err != nil {
		t.Fatal(err)
	}
	if err := idx.AddUser(999, `phantom`); err != nil {
		t.Fatal(err)
	}

	checksum, err := computeDirChecksum(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := idx.applyScan(scanUserFilesInDir(dir), checksum); err != nil {
		t.Fatalf("applyScan failed: %v", err)
	}

	if _, found := idx.FindByUsername(`phantom`); found {
		t.Error("phantom user survived rebuild")
	}
	if highest := idx.GetHighestUserId(); highest != 5 {
		t.Errorf("expected highest userId 5 after rebuild, got %d", highest)
	}
}

// TestIsUpToDateRejectsTruncatedIndex verifies the header/records
// consistency guard: an index whose records section was cut short must
// never report up to date, even when the directory checksum still matches.
func TestIsUpToDateRejectsTruncatedIndex(t *testing.T) {
	mudlog.SetupLogger(nil, "", "", false)

	usersDir := t.TempDir()

	for i := 1; i <= 5; i++ {
		writeScanTestUser(t, usersDir, i, fmt.Sprintf(`User_%d`, i), ``, 0)
	}

	checksum, err := computeDirChecksum(usersDir)
	if err != nil {
		t.Fatal(err)
	}

	idx := newScanTestIndex(usersDir)
	if err := idx.applyScan(scanUserFilesInDir(usersDir), checksum); err != nil {
		t.Fatalf("applyScan failed: %v", err)
	}
	if !idx.isUpToDateForDir(usersDir) {
		t.Fatal("expected freshly rebuilt index to be up to date")
	}

	// Cut the records section short, simulating a crash mid-write, and
	// reload the way startup does.
	fileBytes, err := os.ReadFile(idx.Filename)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(idx.Filename, fileBytes[:len(fileBytes)-20], 0644); err != nil {
		t.Fatal(err)
	}

	truncated := newScanTestIndex(usersDir)
	truncated.metaData = truncated.getMetaDataFromFile()
	truncated.loadRecords()

	if truncated.isUpToDateForDir(usersDir) {
		t.Fatal("truncated index must not report up to date")
	}
}

// BenchmarkRebuildFromScan measures scan plus rebuild over synthetic user
// directories with realistically sized files (embedded character block).
func BenchmarkRebuildFromScan(b *testing.B) {
	mudlog.SetupLogger(nil, "", "", false)

	for _, userCt := range []int{100, 1000, 10000} {
		b.Run(fmt.Sprintf(`%d_users`, userCt), func(b *testing.B) {
			dir := b.TempDir()
			for i := 1; i <= userCt; i++ {
				writeScanTestUser(b, dir, i, fmt.Sprintf(`User_%d`, i), fmt.Sprintf(`Hero_%d`, i), 8)
			}
			idx := newScanTestIndex(dir)

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				checksum, err := computeDirChecksum(dir)
				if err != nil {
					b.Fatal(err)
				}
				if err := idx.applyScan(scanUserFilesInDir(dir), checksum); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkScanVsFullUnmarshal isolates the parse cost the scan avoids:
// unmarshaling a real user file (the stock admin record with an embedded
// character) into the minimal scan struct versus a full UserRecord.
func BenchmarkScanVsFullUnmarshal(b *testing.B) {
	mudlog.SetupLogger(nil, "", "", false)

	fileBytes, err := os.ReadFile(`../../_datafiles/world/default/users/1.yaml`)
	if err != nil {
		b.Skip(`stock user file not available`)
	}

	b.Run(`minimal_scan`, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var scanned userFileScanFields
			if err := yaml.Unmarshal(fileBytes, &scanned); err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run(`full_userrecord`, func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			var u UserRecord
			if err := yaml.Unmarshal(fileBytes, &u); err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkScanRealUsers scans a real users directory named by the
// BENCH_USERS_DIR environment variable. The directory is only read - the
// rebuilt index is written to the benchmark temp dir - so it is safe to
// point at a live server's users directory. Skipped when the variable is
// unset.
func BenchmarkScanRealUsers(b *testing.B) {
	mudlog.SetupLogger(nil, "", "", false)

	usersDir := os.Getenv(`BENCH_USERS_DIR`)
	if usersDir == `` {
		b.Skip(`set BENCH_USERS_DIR to a users directory to run this benchmark`)
	}

	idx := newScanTestIndex(b.TempDir())

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		checksum, err := computeDirChecksum(usersDir)
		if err != nil {
			b.Fatal(err)
		}
		if err := idx.applyScan(scanUserFilesInDir(usersDir), checksum); err != nil {
			b.Fatal(err)
		}
	}
	b.StopTimer()

	b.ReportMetric(float64(idx.GetMetaData().RecordCount), `users`)
}
