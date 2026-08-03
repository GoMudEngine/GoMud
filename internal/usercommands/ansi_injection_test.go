package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/ansitags"
)

// The attack: close the tag the command opens around player text, then open
// one of your own. internal/hooks runs AnsiParse over the assembled message for
// every recipient, so an unescaped tag renders as real colour and the line
// becomes indistinguishable from server output — a credible fake
// "session expired, re-enter your password" prompt.
const ansiInjectionPayload = `</ansi><ansi fg="red">SESSION EXPIRED re-enter password`

// A benign message of no particular shape. The surrounding template emits a
// fixed number of ANSI escapes regardless of what the player typed, so this
// gives us the baseline to compare against.
const ansiBenignPayload = `SESSION EXPIRED re-enter password`

// runCommandAndCollect invokes a command and returns everything it queued back
// to the acting player, parsed the way a real client would see it.
func runCommandAndCollect(t *testing.T, user *users.UserRecord, room *rooms.Room,
	cmd func(string, *users.UserRecord, *rooms.Room, events.EventFlag) (bool, error), rest string) string {

	t.Helper()

	events.DrainQueuedMessagesForTest(user.UserId) // clear anything left over

	if _, err := cmd(rest, user, room, 0); err != nil {
		t.Fatalf("command returned error: %v", err)
	}

	msgs := events.DrainQueuedMessagesForTest(user.UserId)
	if len(msgs) == 0 {
		t.Fatalf("command queued no message back to the player; nothing to assert on")
	}

	return ansitags.Parse(strings.Join(msgs, "\n"))
}

func assertNoInjection(t *testing.T, name, benignOut, attackOut string) {
	t.Helper()

	benignEscapes := strings.Count(benignOut, "\x1b")
	attackEscapes := strings.Count(attackOut, "\x1b")

	// The template's own colouring is identical in both runs, so any extra
	// escape sequence came from the player's text.
	if attackEscapes != benignEscapes {
		t.Fatalf("%s: player text produced %d extra ANSI escape sequence(s) (%d vs baseline %d)\noutput: %q",
			name, attackEscapes-benignEscapes, attackEscapes, benignEscapes, attackOut)
	}

	// The tag must have been rendered as literal characters, not consumed.
	if !strings.Contains(attackOut, `fg="red"`) {
		t.Fatalf("%s: the injected tag was consumed by the parser rather than shown literally\noutput: %q", name, attackOut)
	}

	// And the player's words must still be readable — escaping neutralises,
	// it does not censor.
	if !strings.Contains(attackOut, "SESSION EXPIRED") {
		t.Fatalf("%s: the player's visible text was lost\noutput: %q", name, attackOut)
	}
}

// A player must not be able to emit a working <ansi> tag through `say`.
func TestSayCannotInjectAnsiTags(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	benign := runCommandAndCollect(t, user, room, Say, ansiBenignPayload)
	attack := runCommandAndCollect(t, user, room, Say, ansiInjectionPayload)

	assertNoInjection(t, "say", benign, attack)
}

// Same for `emote`, which uses a different formatter.
func TestEmoteCannotInjectAnsiTags(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	benign := runCommandAndCollect(t, user, room, Emote, ansiBenignPayload)
	attack := runCommandAndCollect(t, user, room, Emote, ansiInjectionPayload)

	assertNoInjection(t, "emote", benign, attack)
}

// `shout` crosses room boundaries, so a forged tag there reaches players who
// never chose to listen.
func TestShoutCannotInjectAnsiTags(t *testing.T) {
	cleanup := seedAllRegistries()
	defer cleanup()

	user, room := getTestUserAndRoom(t)

	benign := runCommandAndCollect(t, user, room, Shout, ansiBenignPayload)
	attack := runCommandAndCollect(t, user, room, Shout, ansiInjectionPayload)

	// Shout uppercases the text, so match on the uppercase forms.
	if strings.Count(attack, "\x1b") != strings.Count(benign, "\x1b") {
		t.Fatalf("shout: player text produced extra ANSI escape sequences\noutput: %q", attack)
	}
	if !strings.Contains(strings.ToLower(attack), `fg="red"`) {
		t.Fatalf("shout: the injected tag was consumed rather than shown literally\noutput: %q", attack)
	}
}

// Control: the payload really is dangerous when it is not escaped. Without
// this, the tests above would still pass if the parser had simply stopped
// emitting ANSI.
func TestAnsiInjectionPayloadIsDangerousUnescaped(t *testing.T) {
	raw := ansitags.Parse(`<ansi fg="saytext">` + ansiInjectionPayload + `</ansi>`)
	safe := ansitags.Parse(`<ansi fg="saytext">` + ansiBenignPayload + `</ansi>`)

	if strings.Count(raw, "\x1b") <= strings.Count(safe, "\x1b") {
		t.Fatalf("control failed: the unescaped payload produced no extra ANSI, so these tests prove nothing")
	}
}
