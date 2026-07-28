package users

import (
	"os"
	"path/filepath"
	"testing"
)

func writeIdxFile(t *testing.T, dir, name, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644); err != nil {
		t.Fatalf("writing %s: %v", name, err)
	}
}

// characterNameIndexFromDir must map every character name (mains + alts,
// case-folded) to its owning userId in ONE minimal-decode pass — replacing the
// per-lookup full directory scan that made boot O(mobs x users).
func TestCharacterNameIndexFromDir_MainsAndAlts(t *testing.T) {
	dir := t.TempDir()
	writeIdxFile(t, dir, "2.yaml", "userid: 2\nusername: alice\ncharacter:\n  name: Aliceia\n")
	writeIdxFile(t, dir, "3.yaml", "userid: 3\nusername: bob\ncharacter:\n  name: Bobrick\n")
	writeIdxFile(t, dir, "3.alts.yaml", "- name: Shadowbob\n- name: Thirdguy\n")

	idx := characterNameIndexFromDir(dir)

	for name, wantUserId := range map[string]int{
		"aliceia":   2,
		"ALICEIA":   2, // case-insensitive lookup key is the caller's job — index stores folded
		"bobrick":   3,
		"shadowbob": 3,
		"thirdguy":  3,
	} {
		got, ok := idx[foldCharacterName(name)]
		if !ok || got != wantUserId {
			t.Errorf("index[%q] = (%d, %v), want (%d, true)", name, got, ok, wantUserId)
		}
	}
	if _, ok := idx[foldCharacterName("nobody")]; ok {
		t.Error("unexpected entry for unknown name")
	}
}

// Malformed files skip with a warning — never abort the scan.
func TestCharacterNameIndexFromDir_MalformedSkipped(t *testing.T) {
	dir := t.TempDir()
	writeIdxFile(t, dir, "1.yaml", "character: [not\nvalid: {{{\n")
	writeIdxFile(t, dir, "2.yaml", "userid: 2\nusername: alice\ncharacter:\n  name: Aliceia\n")
	writeIdxFile(t, dir, "2.alts.yaml", "also: not: a: list\n")

	idx := characterNameIndexFromDir(dir)

	if got, ok := idx[foldCharacterName("Aliceia")]; !ok || got != 2 {
		t.Errorf("index[Aliceia] = (%d, %v), want (2, true)", got, ok)
	}
}
