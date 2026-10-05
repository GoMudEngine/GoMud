package inputhandlers

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/stretchr/testify/assert"
)

// newEnvironPayload builds the bytes that appear between `IAC SB NEW-ENVIRON IS`
// and the trailing `IAC SE` for a sequence of VAR/VALUE pairs.
func newEnvironPayload(pairs ...[2]string) []byte {
	out := []byte{}
	for _, p := range pairs {
		out = append(out, term.TELNET_NEWENV_VAR)
		out = append(out, []byte(p[0])...)
		out = append(out, term.TELNET_NEWENV_VALUE)
		out = append(out, []byte(p[1])...)
	}
	return out
}

func TestNewEnvironIsMudlet(t *testing.T) {
	tests := []struct {
		name    string
		payload []byte
		want    bool
	}{
		{
			name:    "mudlet with version",
			payload: newEnvironPayload([2]string{"CLIENT_NAME", "Mudlet"}, [2]string{"CLIENT_VERSION", "4.17.2"}),
			want:    true,
		},
		{
			name:    "mudlet uppercase value",
			payload: newEnvironPayload([2]string{"CLIENT_NAME", "MUDLET"}),
			want:    true,
		},
		{
			name:    "client name not first",
			payload: newEnvironPayload([2]string{"CLIENT_VERSION", "4.17.2"}, [2]string{"CLIENT_NAME", "Mudlet"}),
			want:    true,
		},
		{
			name:    "different client",
			payload: newEnvironPayload([2]string{"CLIENT_NAME", "TinTin++"}),
			want:    false,
		},
		{
			name:    "client name without value",
			payload: append([]byte{term.TELNET_NEWENV_VAR}, []byte("CLIENT_NAME")...),
			want:    false,
		},
		{
			name:    "empty payload",
			payload: []byte{},
			want:    false,
		},
		{
			name:    "uservar segment ignored, var matched",
			payload: append(append([]byte{term.TELNET_NEWENV_USERVAR}, []byte("FOO")...), newEnvironPayload([2]string{"CLIENT_NAME", "Mudlet"})...),
			want:    true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := newEnvironIsMudlet(tc.payload); got != tc.want {
				t.Errorf("newEnvironIsMudlet() = %v, want %v", got, tc.want)
			}
		})
	}
}

// TestNewEnvironResponseMatcher verifies the term matcher extracts the payload
// that newEnvironIsMudlet expects from a full client response frame, including
// the trailing IAC SE terminator that the lenient matcher leaves in the payload.
func TestNewEnvironResponseMatcher(t *testing.T) {
	// IAC SB NEW-ENVIRON IS <vars> IAC SE
	frame := append(
		term.TelnetNewEnvironResponse.BytesWithPayload(newEnvironPayload([2]string{"CLIENT_NAME", "Mudlet"})),
		term.TELNET_IAC, term.TELNET_SE,
	)

	ok, payload := term.Matches(frame, term.TelnetNewEnvironResponse)
	if !ok {
		t.Fatalf("expected frame to match TelnetNewEnvironResponse")
	}
	// The trailing IAC SE remains in payload; the parser must still detect Mudlet.
	if !newEnvironIsMudlet(payload) {
		t.Errorf("expected extracted payload to be detected as Mudlet, payload=%v", payload)
	}
}

// TestTelnetIACHandlerShortScreenSizeReport sends a NAWS report cut short after
// three bytes. It used to index past the payload and panic on the connection
// goroutine, which has no recover, so the whole server exited.
func TestTelnetIACHandlerShortScreenSizeReport(t *testing.T) {
	clientInput := &connections.ClientInput{
		ConnectionId: 1,
		DataIn:       []byte{term.TELNET_IAC, term.TELNET_SB, term.TELNET_OPT_NAWS, 0, 80, 0},
	}

	assert.NotPanics(t, func() {
		TelnetIACHandler(clientInput, map[string]any{})
	})
}
