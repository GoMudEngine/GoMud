package usercommands

import (
	"strings"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// newRenameSelfUser creates a minimal UserRecord and a stub room for
// RenameSelf tests. The caller is responsible for draining the event
// queue after each call to collect sent text.
func newRenameSelfUser(t *testing.T, charName string) (*users.UserRecord, *rooms.Room) {
	t.Helper()
	u := users.NewTestUser(500, "testplayer", charName, 9900)
	return u, &rooms.Room{RoomId: 500, Zone: "TestZone"}
}

// drainSentText drains all Message events for the given userId from the
// global event queue and returns them concatenated.
func drainSentText(userId int) string {
	msgs := events.DrainQueuedMessagesForTest(userId)
	return strings.Join(msgs, "")
}

// setRenameCooldownHours overrides CharacterRenameCooldownHours for the
// duration of a test. The overlay is cumulative and not currently
// reversible via the public API, so tests that call this must not share
// package-level state that depends on the prior value.
func setRenameCooldownHours(t *testing.T, hours int) {
	t.Helper()
	if err := configs.AddOverlayOverrides(map[string]any{
		"Balance.CharacterRenameCooldownHours": hours,
		"Validation.NameSizeMin":               1,
		"Validation.NameSizeMax":               50,
	}); err != nil {
		t.Fatalf("setRenameCooldownHours: AddOverlayOverrides: %v", err)
	}
}

// ─── Tests ────────────────────────────────────────────────────────────────────

// TestRenameSelf_NoArgs verifies that calling RenameSelf with an empty
// argument returns handled=true and no error (the command consumed the
// input even though it just showed help).
func TestRenameSelf_NoArgs_HandledNoError(t *testing.T) {
	user, room := newRenameSelfUser(t, "Alice")

	handled, err := RenameSelf("", user, room, 0)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}
	// Help text is template-based and may be empty in the test
	// environment; we only assert the command returned cleanly.
}

// TestRenameSelf_SameName_RejectsWithMessage verifies that passing the
// player's current name sends the "already your name" message and still
// returns handled=true.
func TestRenameSelf_SameName_RejectsWithMessage(t *testing.T) {
	user, room := newRenameSelfUser(t, "Alice")

	handled, err := RenameSelf("Alice", user, room, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	sent := drainSentText(user.UserId)
	if !strings.Contains(sent, "already your name") {
		t.Errorf("expected 'already your name' message; got %q", sent)
	}
}

// TestRenameSelf_SameName_CaseInsensitive verifies that the same-name
// check is case-insensitive (Alice == alice).
func TestRenameSelf_SameName_CaseInsensitive(t *testing.T) {
	user, room := newRenameSelfUser(t, "Alice")

	RenameSelf("alice", user, room, 0)

	sent := drainSentText(user.UserId)
	if !strings.Contains(sent, "already your name") {
		t.Errorf("expected 'already your name' for case-insensitive match; got %q", sent)
	}
}

// TestRenameSelf_WithinCooldown_Refused verifies that when a player
// renamed recently (within the configured cooldown window) the command
// sends a cooldown message and refuses to proceed.
func TestRenameSelf_WithinCooldown_Refused(t *testing.T) {
	setRenameCooldownHours(t, 168) // 7 days
	user, room := newRenameSelfUser(t, "Alice")
	user.LastRenameAt = time.Now().Add(-1 * time.Hour) // 1h ago, inside 168h window

	handled, err := RenameSelf("Bobbi", user, room, 0)

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !handled {
		t.Fatal("expected handled=true")
	}

	sent := strings.ToLower(drainSentText(user.UserId))
	if !strings.Contains(sent, "renamed yourself recently") {
		t.Errorf("expected cooldown message; got %q", sent)
	}
}

// TestRenameSelf_CooldownDisabled_PastsCooldownGate verifies that when
// CharacterRenameCooldownHours is 0 the cooldown check is skipped even if
// LastRenameAt is very recent.
func TestRenameSelf_CooldownDisabled_PastsCooldownGate(t *testing.T) {
	setRenameCooldownHours(t, 0)
	user, room := newRenameSelfUser(t, "Alice")
	user.LastRenameAt = time.Now().Add(-1 * time.Minute) // very recent

	// The call will advance past the cooldown gate and hit ValidateActorName
	// (which may fail due to test config) or StartPrompt. We don't care about
	// the final outcome — only that the cooldown message was NOT sent.
	RenameSelf("Bobbi", user, room, 0) //nolint:errcheck

	sent := strings.ToLower(drainSentText(user.UserId))
	if strings.Contains(sent, "renamed yourself recently") {
		t.Errorf("cooldown=0 should skip the cooldown gate; got %q", sent)
	}
}
