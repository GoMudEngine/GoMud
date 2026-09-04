package connections

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// A websocket text frame must be valid UTF-8, so a payload that splits a rune
// closes the connection with 1007 rather than showing the player anything.
func TestValidUTF8Payload(t *testing.T) {
	bar := strings.Repeat("░", 25)

	tests := []struct {
		name     string
		in       []byte
		replaced bool
	}{
		{name: "ascii", in: []byte("You see a rusty sword."), replaced: false},
		{name: "multi-byte runes", in: []byte(bar), replaced: false},
		{name: "empty", in: []byte{}, replaced: false},
		// The tail of a progress bar cut after the first byte of a 3-byte rune.
		{name: "rune split at the end", in: []byte(bar)[:len(bar)-2], replaced: true},
		// Orphaned continuation bytes, i.e. the other half of that cut.
		{name: "orphan continuation bytes", in: []byte{0x96, 0x91, 'h', 'i'}, replaced: true},
		{name: "lone 0xff", in: []byte{'a', 0xff, 'b'}, replaced: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			payload, replaced := validUTF8Payload(test.in)

			if replaced != test.replaced {
				t.Fatalf("replaced = %v, want %v", replaced, test.replaced)
			}
			if !utf8.Valid(payload) {
				t.Fatalf("payload is still not valid UTF-8: %q", payload)
			}
			if !test.replaced && string(payload) != string(test.in) {
				t.Fatalf("valid payload was altered: got %q, want %q", payload, test.in)
			}
		})
	}
}
