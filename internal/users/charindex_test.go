package users

import (
	"fmt"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func freshCharacterIndex() *CharacterIndex {
	return &CharacterIndex{byName: make(map[string]int)}
}

func TestCharacterIndex_AddFind(t *testing.T) {
	ci := freshCharacterIndex()

	ci.Add("Aldric", 10)

	userId, found := ci.Find("Aldric")
	if !found || userId != 10 {
		t.Fatalf("expected (10, true), got (%d, %v)", userId, found)
	}
}

func TestCharacterIndex_FindCaseInsensitive(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)

	for _, variant := range []string{"aldric", "ALDRIC", "aLdRiC"} {
		userId, found := ci.Find(variant)
		if !found || userId != 10 {
			t.Errorf("Find(%q): expected (10, true), got (%d, %v)", variant, userId, found)
		}
	}
}

func TestCharacterIndex_AddOverwrite(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)
	ci.Add("Aldric", 20)

	userId, found := ci.Find("Aldric")
	if !found || userId != 20 {
		t.Fatalf("expected overwritten userId 20, got (%d, %v)", userId, found)
	}
}

func TestCharacterIndex_AddIdempotent(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)
	ci.Add("Aldric", 10)

	userId, found := ci.Find("Aldric")
	if !found || userId != 10 {
		t.Fatalf("expected (10, true) after idempotent add, got (%d, %v)", userId, found)
	}
}

func TestCharacterIndex_Remove(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)
	ci.Remove("Aldric")

	_, found := ci.Find("Aldric")
	if found {
		t.Fatal("expected name to be absent after Remove")
	}
}

func TestCharacterIndex_RemoveCaseInsensitive(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)
	ci.Remove("ALDRIC")

	_, found := ci.Find("Aldric")
	if found {
		t.Fatal("expected name to be absent after case-insensitive Remove")
	}
}

func TestCharacterIndex_RemoveNoop(t *testing.T) {
	ci := freshCharacterIndex()
	// Should not panic on a missing name.
	ci.Remove("nobody")
}

func TestCharacterIndex_FindMiss(t *testing.T) {
	ci := freshCharacterIndex()

	userId, found := ci.Find("nobody")
	if found || userId != 0 {
		t.Fatalf("expected (0, false), got (%d, %v)", userId, found)
	}
}

func TestCharacterIndex_MultipleUsersMultipleNames(t *testing.T) {
	ci := freshCharacterIndex()
	ci.Add("Aldric", 10)
	ci.Add("Brynn", 10)
	ci.Add("Cael", 20)

	cases := []struct {
		name   string
		userId int
	}{
		{"Aldric", 10},
		{"Brynn", 10},
		{"Cael", 20},
	}

	for _, tc := range cases {
		userId, found := ci.Find(tc.name)
		if !found || userId != tc.userId {
			t.Errorf("Find(%q): expected (%d, true), got (%d, %v)", tc.name, tc.userId, userId, found)
		}
	}
}

func TestCharacterIndex_Rebuild(t *testing.T) {
	// The scan warns about the missing users directory in a test env, so the
	// logger must be initialized.
	mudlog.SetupLogger(nil, "", "", false)

	// Swap in a fresh singleton so Rebuild exercises the real code path
	// without touching disk (ScanUserFiles finds nothing in a test env).
	orig := characterIndex
	defer func() { characterIndex = orig }()

	ci := freshCharacterIndex()
	characterIndex = ci

	// Pre-populate with stale data that Rebuild should clear.
	ci.Add("stale", 99)

	// Rebuild will call ScanUserFiles (returns nothing in test env) and
	// GetAllActiveUsers (returns nothing since userManager is empty). The stale
	// entry must be gone.
	ci.Rebuild()

	_, found := ci.Find("stale")
	if found {
		t.Fatal("expected stale entry to be cleared after Rebuild")
	}
}

func TestCharacterIndex_Concurrency(t *testing.T) {
	ci := freshCharacterIndex()

	const goroutines = 50
	const namesPerGoroutine = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Writers
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for n := 0; n < namesPerGoroutine; n++ {
				ci.Add(fmt.Sprintf("char_%d_%d", g, n), g+1)
			}
		}()
	}

	// Readers
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for n := 0; n < namesPerGoroutine; n++ {
				ci.Find(fmt.Sprintf("char_%d_%d", g, n))
			}
		}()
	}

	// Removers
	for g := 0; g < goroutines; g++ {
		g := g
		go func() {
			defer wg.Done()
			for n := 0; n < namesPerGoroutine; n++ {
				ci.Remove(fmt.Sprintf("char_%d_%d", g, n))
			}
		}()
	}

	wg.Wait()
}
