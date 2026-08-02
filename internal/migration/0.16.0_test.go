package migration

import (
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v2"
)

// writeVitalityUser writes usersDir/<filename>.yaml with a minimal save that
// carries username + character.stats.{vitality,willpower}.{base,training}.
func writeVitalityUser(t *testing.T, usersDir, filename, username string, vitBase, vitTraining, wilBase, wilTraining int) string {
	t.Helper()
	if err := os.MkdirAll(usersDir, 0755); err != nil {
		t.Fatal(err)
	}
	body := "userid: 1\n" +
		"role: user\n" +
		"username: " + username + "\n" +
		"character:\n" +
		"  name: " + username + "\n" +
		"  stats:\n" +
		"    strength:\n" +
		"      training: 10\n" +
		"      base: 100\n" +
		"    vitality:\n" +
		"      training: " + itoa16(vitTraining) + "\n" +
		"      base: " + itoa16(vitBase) + "\n" +
		"    willpower:\n" +
		"      training: " + itoa16(wilTraining) + "\n" +
		"      base: " + itoa16(wilBase) + "\n"
	path := filepath.Join(usersDir, filename)
	if err := os.WriteFile(path, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func itoa16(i int) string {
	if i == 0 {
		return "0"
	}
	neg := i < 0
	if neg {
		i = -i
	}
	var b []byte
	for i > 0 {
		b = append([]byte{byte('0' + i%10)}, b...)
		i /= 10
	}
	if neg {
		b = append([]byte{'-'}, b...)
	}
	return string(b)
}

type vitalityStats struct {
	Vitality struct {
		Base     int `yaml:"base"`
		Training int `yaml:"training"`
	} `yaml:"vitality"`
	Willpower struct {
		Base     int `yaml:"base"`
		Training int `yaml:"training"`
	} `yaml:"willpower"`
}

func readVitalityStats(t *testing.T, path string) vitalityStats {
	t.Helper()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Character struct {
			Stats vitalityStats `yaml:"stats"`
		} `yaml:"character"`
	}
	if err := yaml.Unmarshal(raw, &doc); err != nil {
		t.Fatal(err)
	}
	return doc.Character.Stats
}

// TestFreezeExploitedVitality_AppliesToFyttyn: base 85 / training 326 (total
// 411) must be frozen to a total of 280 by reducing training only.
func TestFreezeExploitedVitality_AppliesToFyttyn(t *testing.T) {
	dir := t.TempDir()
	path := writeVitalityUser(t, dir, "15.yaml", "fyttyn", 85, 326, 115, 128)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}

	got := readVitalityStats(t, path)
	if got.Vitality.Base != 85 || got.Vitality.Training != 195 {
		t.Errorf("vitality = base %d training %d, want base 85 training 195", got.Vitality.Base, got.Vitality.Training)
	}
	if got.Vitality.Base+got.Vitality.Training != 280 {
		t.Errorf("vitality total = %d, want 280", got.Vitality.Base+got.Vitality.Training)
	}
}

// TestFreezeExploitedVitality_Idempotent: running the migration twice must
// not change an already-frozen save (exercises the total<=280 guard).
func TestFreezeExploitedVitality_Idempotent(t *testing.T) {
	dir := t.TempDir()
	path := writeVitalityUser(t, dir, "15.yaml", "fyttyn", 85, 326, 115, 128)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("first run: %v", err)
	}
	first := readVitalityStats(t, path)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("second run: %v", err)
	}
	second := readVitalityStats(t, path)

	if first.Vitality.Training != 195 {
		t.Fatalf("precondition failed: first-run training = %d, want 195", first.Vitality.Training)
	}
	if second.Vitality.Training != first.Vitality.Training || second.Vitality.Base != first.Vitality.Base {
		t.Errorf("second run changed vitality: %+v -> %+v", first.Vitality, second.Vitality)
	}
}

// TestFreezeExploitedVitality_OtherUsersUntouched: a different, legitimately
// high vitality (e.g. Duard-style, total 195 which is already <=280 anyway,
// but critically NOT named fyttyn) must never be touched even if it were
// above the target, since the migration only matches on username.
func TestFreezeExploitedVitality_OtherUsersUntouched(t *testing.T) {
	dir := t.TempDir()
	// Deliberately above the frozen target to prove the username gate, not the
	// total<=target guard, is what protects other players.
	otherPath := writeVitalityUser(t, dir, "17.yaml", "duard", 100, 400, 90, 40)
	fyttynPath := writeVitalityUser(t, dir, "15.yaml", "fyttyn", 85, 326, 115, 128)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}

	other := readVitalityStats(t, otherPath)
	if other.Vitality.Base != 100 || other.Vitality.Training != 400 {
		t.Errorf("other user's vitality changed: base %d training %d, want base 100 training 400", other.Vitality.Base, other.Vitality.Training)
	}

	fy := readVitalityStats(t, fyttynPath)
	if fy.Vitality.Base+fy.Vitality.Training != 280 {
		t.Errorf("fyttyn total = %d, want 280", fy.Vitality.Base+fy.Vitality.Training)
	}
}

// TestFreezeExploitedVitality_OtherStatsUntouched: willpower (and other
// stats) on fyttyn's own save must be left exactly as they were.
func TestFreezeExploitedVitality_OtherStatsUntouched(t *testing.T) {
	dir := t.TempDir()
	path := writeVitalityUser(t, dir, "15.yaml", "fyttyn", 85, 326, 115, 128)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}

	got := readVitalityStats(t, path)
	if got.Willpower.Base != 115 || got.Willpower.Training != 128 {
		t.Errorf("willpower changed: base %d training %d, want base 115 training 128", got.Willpower.Base, got.Willpower.Training)
	}
}

// TestFreezeExploitedVitality_CaseInsensitive: "Fyttyn" (any case) must still
// match.
func TestFreezeExploitedVitality_CaseInsensitive(t *testing.T) {
	dir := t.TempDir()
	path := writeVitalityUser(t, dir, "15.yaml", "Fyttyn", 85, 326, 115, 128)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("migration: %v", err)
	}

	got := readVitalityStats(t, path)
	if got.Vitality.Base+got.Vitality.Training != 280 {
		t.Errorf("total = %d, want 280", got.Vitality.Base+got.Vitality.Training)
	}
}

// TestFreezeExploitedVitality_AbsentAccountIsNotError: a users dir with no
// fyttyn save at all must return nil, not an error (dev trees have no such
// account).
func TestFreezeExploitedVitality_AbsentAccountIsNotError(t *testing.T) {
	dir := t.TempDir()
	writeVitalityUser(t, dir, "1.yaml", "someoneelse", 50, 20, 50, 20)

	if err := freezeExploitedVitalityInDir(dir, false); err != nil {
		t.Fatalf("expected nil error for a tree with no fyttyn account, got: %v", err)
	}
}
