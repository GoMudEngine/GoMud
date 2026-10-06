package inputhandlers

import (
	"bytes"
	"net"
	"testing"
	"time"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

// captureConn is a minimal net.Conn that records every byte written to it, so
// tests can assert on exactly what the terminal would receive.
type captureConn struct {
	buf bytes.Buffer
}

func (c *captureConn) Read(p []byte) (int, error)         { return 0, net.ErrClosed }
func (c *captureConn) Write(p []byte) (int, error)        { return c.buf.Write(p) }
func (c *captureConn) Close() error                       { return nil }
func (c *captureConn) LocalAddr() net.Addr                { return dummyAddr{} }
func (c *captureConn) RemoteAddr() net.Addr               { return dummyAddr{} }
func (c *captureConn) SetDeadline(time.Time) error        { return nil }
func (c *captureConn) SetReadDeadline(time.Time) error    { return nil }
func (c *captureConn) SetWriteDeadline(time.Time) error   { return nil }

type dummyAddr struct{}

func (dummyAddr) Network() string { return "test" }
func (dummyAddr) String() string  { return "test" }

// TestCleanserEmitsWidthAwareBackspace is the integration test for the user's
// core complaint: backspacing a wide (CJK) character must erase the full number
// of columns it occupies, not just one.
func TestCleanserEmitsWidthAwareBackspace(t *testing.T) {
	cc := &captureConn{}
	cd := connections.Add(cc, nil, connections.ConnHuman)
	t.Cleanup(func() { connections.Remove(cd.ConnectionId()) })

	ci := &connections.ClientInput{ConnectionId: cd.ConnectionId(), Buffer: []byte{}, Cursor: 0}
	state := map[string]any{}

	// Type a CJK character (2 display columns).
	ci.DataIn = []byte("你")
	CleanserInputHandler(ci, state)

	if string(ci.Buffer) != "你" {
		t.Fatalf("buffer=%q want 你", string(ci.Buffer))
	}
	if ci.Cursor != 3 {
		t.Fatalf("cursor=%d want 3", ci.Cursor)
	}
	if !bytes.Contains(cc.buf.Bytes(), []byte("你")) {
		t.Errorf("echo should contain 你, got %q", cc.buf.String())
	}

	// Backspace it.
	cc.buf.Reset()
	ci.DataIn = []byte{term.ASCII_BACKSPACE}
	ci.BSPressed = false
	CleanserInputHandler(ci, state)

	if string(ci.Buffer) != "" {
		t.Errorf("buffer=%q want empty after backspace", string(ci.Buffer))
	}
	if ci.Cursor != 0 {
		t.Errorf("cursor=%d want 0 after backspace", ci.Cursor)
	}

	out := cc.buf.Bytes()
	// The glyph is 2 columns wide, so the erase must move back 2 columns...
	if !bytes.Contains(out, cursorBackwardN(2)) {
		t.Errorf("expected a 2-column cursor-back (ESC[2D) in %q", cc.buf.String())
	}
	// ...and clear to the end of the line so no half-glyph remains.
	if !bytes.Contains(out, []byte(term.AnsiEraseLineForward.String())) {
		t.Errorf("expected erase-to-end-of-line (ESC[0K) in %q", cc.buf.String())
	}
	// The old single-column "\b \b" sequence must NOT be used for wide chars.
	if bytes.Contains(out, term.BACKSPACE_SEQUENCE) {
		t.Errorf("wide-char backspace must not use the 1-col BACKSPACE_SEQUENCE, got %q", cc.buf.String())
	}
}

// TestCleanserMidLineInsertEcho checks that inserting a character in the middle
// of the line redraws the tail and repositions the cursor (not just overwrites).
func TestCleanserMidLineInsertEcho(t *testing.T) {
	cc := &captureConn{}
	cd := connections.Add(cc, nil, connections.ConnHuman)
	t.Cleanup(func() { connections.Remove(cd.ConnectionId()) })

	// Existing buffer "llo" with cursor at the front (as if moved left from end).
	ci := &connections.ClientInput{
		ConnectionId: cd.ConnectionId(),
		Buffer:       []byte("llo"),
		Cursor:       0,
	}
	ci.DataIn = []byte("X")
	CleanserInputHandler(ci, map[string]any{})

	if string(ci.Buffer) != "Xllo" {
		t.Errorf("buffer=%q want Xllo", string(ci.Buffer))
	}
	if ci.Cursor != 1 {
		t.Errorf("cursor=%d want 1", ci.Cursor)
	}
	// The redraw must emit the inserted char + the tail so "llo" is preserved.
	if !bytes.Contains(cc.buf.Bytes(), []byte("Xllo")) {
		t.Errorf("echo should contain the redrawn tail Xllo, got %q", cc.buf.String())
	}
}

// TestAnsiHandlerLeftArrowEmitsWidthAwareMove checks that the Left arrow moves
// the terminal cursor back by the display width of the character it crosses.
func TestAnsiHandlerLeftArrowEmitsWidthAwareMove(t *testing.T) {
	cc := &captureConn{}
	cd := connections.Add(cc, nil, connections.ConnHuman)
	t.Cleanup(func() { connections.Remove(cd.ConnectionId()) })

	// Buffer "x你" with cursor at the end (3 bytes).
	ci := &connections.ClientInput{
		ConnectionId: cd.ConnectionId(),
		Buffer:       []byte("x你"),
		Cursor:       4, // len("x你")
		DataIn:       []byte{term.ANSI_ESC, '[', 'D'}, // Left arrow
	}
	AnsiHandler(ci, map[string]any{})

	if ci.Cursor != 1 { // crossed back over 你 (3 bytes) -> lands after 'x'
		t.Errorf("cursor=%d want 1", ci.Cursor)
	}
	// 你 is 2 columns wide, so the terminal cursor must move back 2 columns.
	if !bytes.Contains(cc.buf.Bytes(), cursorBackwardN(2)) {
		t.Errorf("expected ESC[2D for crossing a wide char, got %q", cc.buf.String())
	}
}
