package inputhandlers

import (
	"testing"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

func TestBufferDisplayWidth(t *testing.T) {
	cases := []struct {
		in   string
		want int
	}{
		{"", 0},
		{"abc", 3},
		{"你好", 4},   // two CJK ideographs, 2 columns each
		{"a你b", 4},  // 1 + 2 + 1
		{"你好世界", 8}, // four CJK ideographs
		{"🚀", 2},    // emoji, wide
	}
	for _, c := range cases {
		if got := bufferDisplayWidth([]byte(c.in)); got != c.want {
			t.Errorf("bufferDisplayWidth(%q) = %d, want %d", c.in, got, c.want)
		}
	}
}

func TestBufferDisplayWidthSkipsInvalidUTF8(t *testing.T) {
	// A lone continuation byte is invalid UTF-8; it must be skipped, not counted.
	bad := []byte{'a', 0xFF, 'b'}
	if got := bufferDisplayWidth(bad); got != 2 {
		t.Errorf("bufferDisplayWidth(invalid) = %d, want 2", got)
	}
}

func TestRuneDisplayWidth(t *testing.T) {
	cases := []struct {
		r    rune
		want int
	}{
		{'a', 1},
		{'你', 2},
		{'€', 2}, // East-Asian wide per runewidth
		{0x00, 1},
	}
	for _, c := range cases {
		if got := runeDisplayWidth(c.r); got != c.want {
			t.Errorf("runeDisplayWidth(%q) = %d, want %d", c.r, got, c.want)
		}
	}
}

func TestKeyMatchers(t *testing.T) {
	// Left / Right (CSI and application-mode variants)
	if !matchesKey([]byte{term.ANSI_ESC, '[', 'D'}, term.AnsiMoveCursorBackward) {
		t.Error("ESC[D should match left arrow")
	}
	if !matchesKey([]byte{term.ANSI_ESC, 'O', 'D'}, term.AnsiKeyLeftApp) {
		t.Error("ESC O D should match app-mode left arrow")
	}
	if !matchesKey([]byte{term.ANSI_ESC, '[', 'C'}, term.AnsiMoveCursorForward) {
		t.Error("ESC[C should match right arrow")
	}

	// Home / End variants across terminals
	homeSeqs := [][]byte{
		{term.ANSI_ESC, '[', 'H'},                     // ESC[H
		{term.ANSI_ESC, '[', '1', '~'},                // ESC[1~
		{term.ANSI_ESC, 'O', 'H'},                     // ESC OH
	}
	for _, s := range homeSeqs {
		if !isHomeKey(s) {
			t.Errorf("expected Home key match for %v", s)
		}
	}
	endSeqs := [][]byte{
		{term.ANSI_ESC, '[', 'F'},
		{term.ANSI_ESC, '[', '4', '~'},
		{term.ANSI_ESC, 'O', 'F'},
	}
	for _, s := range endSeqs {
		if !isEndKey(s) {
			t.Errorf("expected End key match for %v", s)
		}
	}

	// Delete
	if !matchesKey([]byte{term.ANSI_ESC, '[', '3', '~'}, term.AnsiKeyDelete) {
		t.Error("ESC[3~ should match Delete")
	}
}

func TestClampCursor(t *testing.T) {
	ci := &connections.ClientInput{Buffer: []byte("hello")}
	ci.Cursor = 99
	clampCursor(ci)
	if ci.Cursor != 5 {
		t.Errorf("expected cursor clamped to 5, got %d", ci.Cursor)
	}
	ci.Cursor = -3
	clampCursor(ci)
	if ci.Cursor != 0 {
		t.Errorf("expected cursor clamped to 0, got %d", ci.Cursor)
	}
}

// TestCleanserInsertAtCursor verifies that typed text is inserted at the logical
// cursor rather than always appended, enabling mid-line edits.
func TestCleanserInsertAtCursor(t *testing.T) {
	// Buffer "hello", cursor in the middle (after "he"). Typing "X" should
	// produce "heXllo" with the cursor after the X.
	buf := []byte("hello")
	clientInput := &connections.ClientInput{
		ConnectionId: 1,
		DataIn:       []byte("X"),
		Buffer:       buf,
		Cursor:       2,
	}
	CleanserInputHandler(clientInput, make(map[string]any))

	if got := string(clientInput.Buffer); got != "heXllo" {
		t.Errorf("buffer = %q, want %q", got, "heXllo")
	}
	if clientInput.Cursor != 3 {
		t.Errorf("cursor = %d, want 3", clientInput.Cursor)
	}
}

// TestCleanserBackspaceRemovesFullCJKRune checks that a single backspace
// deletes an entire multi-byte CJK character (3 bytes) from the buffer.
func TestCleanserBackspaceRemovesFullCJKRune(t *testing.T) {
	buf := []byte("你好")
	clientInput := &connections.ClientInput{
		ConnectionId: 1,
		DataIn:       []byte{term.ASCII_BACKSPACE},
		Buffer:       buf,
		Cursor:       len(buf),
	}
	CleanserInputHandler(clientInput, make(map[string]any))

	if got := string(clientInput.Buffer); got != "你" {
		t.Errorf("buffer = %q, want %q", got, "你")
	}
	if clientInput.Cursor != 3 { // "你" is 3 bytes
		t.Errorf("cursor = %d, want 3", clientInput.Cursor)
	}
	if !utf8.Valid(clientInput.Buffer) {
		t.Errorf("buffer is not valid UTF-8: %v", clientInput.Buffer)
	}
}

// TestCleanserBackspaceMidBuffer checks that backspace works when the cursor is
// not at the end of the buffer (removes the rune before the cursor).
func TestCleanserBackspaceMidBuffer(t *testing.T) {
	// "heXllo", cursor at index 3 (after X). Backspace removes X -> "hello".
	clientInput := &connections.ClientInput{
		ConnectionId: 1,
		DataIn:       []byte{term.ASCII_BACKSPACE},
		Buffer:       []byte("heXllo"),
		Cursor:       3,
	}
	CleanserInputHandler(clientInput, make(map[string]any))

	if got := string(clientInput.Buffer); got != "hello" {
		t.Errorf("buffer = %q, want %q", got, "hello")
	}
	if clientInput.Cursor != 2 {
		t.Errorf("cursor = %d, want 2", clientInput.Cursor)
	}
}
