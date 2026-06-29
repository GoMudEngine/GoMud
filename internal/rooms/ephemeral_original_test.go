package rooms

import "testing"

func TestOriginalRoomId_NonEphemeral(t *testing.T) {
	orig, ok := OriginalRoomId(5200)
	if ok {
		t.Fatalf("5200 should not be ephemeral, got ok=true orig=%d", orig)
	}
	if orig != 5200 {
		t.Fatalf("non-ephemeral must return the same id, got %d", orig)
	}
}
