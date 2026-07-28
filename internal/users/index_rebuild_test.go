package users

import (
	"os"
	"path/filepath"
	"testing"
)

func writeUserFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

func newTestIndex(t *testing.T, dir string) *UserIndex {
	t.Helper()
	idx := &UserIndex{Filename: filepath.Join(dir, `users.idx`)}
	if err := idx.Create(); err != nil {
		t.Fatalf("creating index: %v", err)
	}
	return idx
}

// rebuildFromDir must index every user with ONE minimal-decode scan and a
// single atomic write (upstream #638 port).
func TestRebuildFromDir_IndexesAllUsers(t *testing.T) {
	dir := t.TempDir()
	writeUserFile(t, dir, "2.yaml", "userid: 2\nusername: alice\ncharacter:\n  name: Aliceia\n")
	writeUserFile(t, dir, "3.yaml", "userid: 3\nusername: bob\ncharacter:\n  name: Bobrick\n")
	writeUserFile(t, dir, "10.yaml", "userid: 10\nusername: carol\ncharacter:\n  name: Carolix\n")

	idx := newTestIndex(t, dir)
	if err := idx.rebuildFromDir(dir); err != nil {
		t.Fatalf("rebuildFromDir: %v", err)
	}

	if got := idx.GetMetaData().RecordCount; got != 3 {
		t.Errorf("RecordCount = %d, want 3", got)
	}
	if got := idx.GetHighestUserId(); got != 10 {
		t.Errorf("GetHighestUserId = %d, want 10", got)
	}
	for name, want := range map[string]int64{"alice": 2, "bob": 3, "carol": 10} {
		id, found := idx.FindByUsername(name)
		if !found || id != want {
			t.Errorf("FindByUsername(%q) = (%d, %v), want (%d, true)", name, id, found, want)
		}
	}
}

// A malformed user file must be skipped with a warning — NOT abort the walk.
// The old SearchOfflineUsers-based rebuild returned the error from the walk
// callback, silently dropping every user after the bad file from the index
// (those players could no longer log in).
func TestRebuildFromDir_MalformedFileDoesNotAbort(t *testing.T) {
	dir := t.TempDir()
	// Lexically first so the old behavior would have dropped everyone after it.
	writeUserFile(t, dir, "1.yaml", "userid: [1, this is not\n\tvalid yaml at all: {{{{\n")
	writeUserFile(t, dir, "2.yaml", "userid: 2\nusername: alice\n")
	writeUserFile(t, dir, "3.yaml", "userid: 3\nusername: bob\n")

	idx := newTestIndex(t, dir)
	if err := idx.rebuildFromDir(dir); err != nil {
		t.Fatalf("rebuildFromDir: %v", err)
	}

	if got := idx.GetMetaData().RecordCount; got != 2 {
		t.Errorf("RecordCount = %d, want 2 (malformed file skipped)", got)
	}
	if _, found := idx.FindByUsername("alice"); !found {
		t.Error("alice missing — users after a malformed file must still index")
	}
	if _, found := idx.FindByUsername("bob"); !found {
		t.Error("bob missing — users after a malformed file must still index")
	}
}

// .alts.yaml companions, records with no userid/username, and duplicates are
// skipped (first record wins on duplicate).
func TestRebuildFromDir_SkipsAltsEmptyAndDuplicates(t *testing.T) {
	dir := t.TempDir()
	writeUserFile(t, dir, "2.yaml", "userid: 2\nusername: alice\n")
	writeUserFile(t, dir, "2.alts.yaml", "userid: 99\nusername: notauser\n")
	writeUserFile(t, dir, "4.yaml", "character:\n  name: NoIdentity\n") // no userid/username
	writeUserFile(t, dir, "5.yaml", "userid: 5\nusername: alice\n")     // duplicate username
	writeUserFile(t, dir, "6.yaml", "userid: 2\nusername: eve\n")       // duplicate userid

	idx := newTestIndex(t, dir)
	if err := idx.rebuildFromDir(dir); err != nil {
		t.Fatalf("rebuildFromDir: %v", err)
	}

	if got := idx.GetMetaData().RecordCount; got != 1 {
		t.Errorf("RecordCount = %d, want 1 (alts/empty/duplicates skipped)", got)
	}
	id, found := idx.FindByUsername("alice")
	if !found || id != 2 {
		t.Errorf("FindByUsername(alice) = (%d, %v), want (2, true)", id, found)
	}
	if _, found := idx.FindByUsername("notauser"); found {
		t.Error(".alts.yaml file must not be indexed")
	}
	if _, found := idx.FindByUsername("eve"); found {
		t.Error("duplicate userid must not be indexed (first record wins)")
	}
}

// The rebuild must not leave a temp file behind, and the written index must be
// readable by a fresh UserIndex instance (atomic replace, not in-place writes).
func TestRebuildFromDir_AtomicWriteReadableByFreshIndex(t *testing.T) {
	dir := t.TempDir()
	writeUserFile(t, dir, "2.yaml", "userid: 2\nusername: alice\n")

	idx := newTestIndex(t, dir)
	if err := idx.rebuildFromDir(dir); err != nil {
		t.Fatalf("rebuildFromDir: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Name() != "users.idx" && e.Name() != "2.yaml" {
			t.Errorf("unexpected leftover file after rebuild: %s", e.Name())
		}
	}

	fresh := &UserIndex{Filename: idx.Filename}
	fresh.metaData = fresh.getMetaDataFromFile()
	fresh.calculateHighestUserId()
	id, found := fresh.FindByUsername("alice")
	if !found || id != 2 {
		t.Errorf("fresh index FindByUsername(alice) = (%d, %v), want (2, true)", id, found)
	}
	if fresh.GetHighestUserId() != 2 {
		t.Errorf("fresh index highest userId = %d, want 2", fresh.GetHighestUserId())
	}
}
