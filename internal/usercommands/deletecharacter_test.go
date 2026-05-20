package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// defaultBiome returns a minimal lit biome suitable for test rooms.
var defaultBiome = &rooms.BiomeInfo{
	BiomeId: "default",
	Name:    "Default",
	LitArea: true,
}

// seedDefaultBiome seeds the rooms biome registry with a single default entry
// and returns a cleanup function. Required by any test that calls into
// room.SendTextVisual (which calls GetVisibility → GetBiome).
func seedDefaultBiome(t *testing.T) func() {
	t.Helper()
	return rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"default": defaultBiome,
	})
}

// newDeleteCharacterUser creates a UserRecord seeded into the global
// userManager and returns it along with a stub room.
// The seeded user is automatically cleaned up when the test ends.
func newDeleteCharacterUser(t *testing.T, charName string) (*users.UserRecord, *rooms.Room) {
	t.Helper()
	const testUserId = 701
	u := users.NewTestUser(testUserId, strings.ToLower(charName), charName, 0)
	cleanup := users.SeedUsersForTest(map[int]*users.UserRecord{testUserId: u})
	t.Cleanup(cleanup)
	return u, &rooms.Room{RoomId: 700, Zone: "TestZone"}
}

// answerPromptQuestion sets the response on the Nth question of the user's
// active prompt, simulating the player typing an answer.
func answerPromptQuestion(u *users.UserRecord, idx int, response string) {
	p := u.GetPrompt()
	if p == nil || idx >= len(p.Questions) {
		return
	}
	p.Questions[idx].Response = response
	p.Questions[idx].Done = true
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestDeleteCharacter_FirstCallAsksFirstGate verifies that invoking the
// handler with no prompt state returns handled=true without mutating the user
// record (Q1 is pending, so the handler exits early).
func TestDeleteCharacter_FirstCallAsksFirstGate(t *testing.T) {
	user, room := newDeleteCharacterUser(t, "Alice")

	handled, err := DeleteCharacter("", user, room, events.EventFlag(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	// Prompt should now be active and Q1 should be pending.
	p := user.GetPrompt()
	if p == nil {
		t.Fatal("expected active prompt after first call")
	}
	if len(p.Questions) == 0 {
		t.Fatal("expected at least one question in prompt")
	}
	if p.Questions[0].Done {
		t.Error("Q1 should not be Done yet after first call")
	}
	// User should still be in the manager.
	if users.GetByUserId(user.UserId) == nil {
		t.Error("user should still exist after early-return from first gate")
	}
}

// TestDeleteCharacter_FirstGateNo verifies that answering "no" to the first
// gate sends an aborted message and leaves the user intact.
func TestDeleteCharacter_FirstGateNo(t *testing.T) {
	user, room := newDeleteCharacterUser(t, "Alice")

	// First call — creates Q1.
	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck

	// Answer "no" to Q1.
	answerPromptQuestion(user, 0, "no")

	// Second call — Q1 is done with response="no".
	handled, err := DeleteCharacter("", user, room, events.EventFlag(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	sent := drainSentText(user.UserId)
	if !strings.Contains(strings.ToLower(sent), "aborted") {
		t.Errorf("expected 'Aborted' message for no-answer; got %q", sent)
	}
	// User should still exist.
	if users.GetByUserId(user.UserId) == nil {
		t.Error("user should still exist after answering no")
	}
	// Prompt should be cleared.
	if user.GetPrompt() != nil {
		t.Error("prompt should be cleared after abort")
	}
}

// TestDeleteCharacter_WrongNameRejected verifies that confirming "yes" but
// then typing the wrong name sends a "doesn't match" message and leaves the
// user intact.
func TestDeleteCharacter_WrongNameRejected(t *testing.T) {
	user, room := newDeleteCharacterUser(t, "Alice")

	// First call — creates Q1.
	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck
	// Answer "yes" to Q1.
	answerPromptQuestion(user, 0, "yes")

	// Second call — Q1 done, creates Q2.
	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck

	// Answer with wrong name to Q2.
	answerPromptQuestion(user, 1, "alice") // lowercase — wrong case

	// Third call — Q1+Q2 done, wrong name.
	handled, err := DeleteCharacter("", user, room, events.EventFlag(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	sent := drainSentText(user.UserId)
	if !strings.Contains(strings.ToLower(sent), "doesn't match") &&
		!strings.Contains(strings.ToLower(sent), "does not match") {
		t.Errorf("expected 'doesn't match' message; got %q", sent)
	}
	// User must still exist.
	if users.GetByUserId(user.UserId) == nil {
		t.Error("user should still exist after wrong-name rejection")
	}
}

// TestDeleteCharacter_WrongCase_Rejected verifies that the second gate is
// case-sensitive: "alice" is rejected when the name is "Alice".
func TestDeleteCharacter_WrongCase_Rejected(t *testing.T) {
	user, room := newDeleteCharacterUser(t, "Alice")

	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck
	answerPromptQuestion(user, 0, "yes")
	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck
	answerPromptQuestion(user, 1, "ALICE")               // wrong case

	handled, err := DeleteCharacter("", user, room, events.EventFlag(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	sent := drainSentText(user.UserId)
	if !strings.Contains(strings.ToLower(sent), "doesn't match") &&
		!strings.Contains(strings.ToLower(sent), "does not match") {
		t.Errorf("expected rejection for wrong-case name; got %q", sent)
	}
	if users.GetByUserId(user.UserId) == nil {
		t.Error("user should still exist after wrong-case rejection")
	}
}

// TestDeleteCharacter_CorrectName_DeletesUser verifies that the happy path
// (yes + exact name) removes the user from the in-memory registry.
func TestDeleteCharacter_CorrectName_DeletesUser(t *testing.T) {
	cleanupBiomes := seedDefaultBiome(t)
	defer cleanupBiomes()

	user, room := newDeleteCharacterUser(t, "Alice")

	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck
	answerPromptQuestion(user, 0, "yes")
	DeleteCharacter("", user, room, events.EventFlag(0)) //nolint:errcheck
	answerPromptQuestion(user, 1, "Alice")               // exact case-sensitive match

	handled, err := DeleteCharacter("", user, room, events.EventFlag(0))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	// The user should have been removed from the in-memory registry.
	if users.GetByUserId(user.UserId) != nil {
		t.Error("user should be removed from registry after confirmed deletion")
	}
}
