package moderation

import (
	"os"
	"strings"
	"testing"
)

// Petitions filed before write-side ANSI escaping shipped can hold raw
// <ansi> payloads. loadPetitions must scrub them (idempotently) so the
// stored payload is neutralized on the next save.
func TestLoadPetitions_ScrubsStoredAnsiPayloads(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()

	if err := os.MkdirAll(moderationDir(), 0755); err != nil {
		t.Fatal(err)
	}
	raw := `- id: 1
  reporter: someplayer
  room_id: 5
  zone: Stillwater
  message: '</ansi><ansi fg="red">FAKE ADMIN NOTICE'
  status: open
  note: '<ansi bg="white">resolved note</ansi>'
`
	if err := os.WriteFile(petitionsPath(), []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}

	if n := loadPetitions(); n != 1 {
		t.Fatalf("loaded %d petitions, want 1", n)
	}

	mu.Lock()
	p := petitions[0]
	mu.Unlock()
	for field, val := range map[string]string{"message": p.Message, "note": p.Note} {
		if strings.Contains(val, "<ansi") || strings.Contains(val, "</ansi>") {
			t.Errorf("%s still holds a live ansi tag after load: %q", field, val)
		}
	}
}
