package term

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// A NAWS report is four bytes: width then height, each high byte first. A
// client can send fewer, and that must come back as an error rather than a
// read past the end of the payload.
func TestTelnetParseScreenSizePayload(t *testing.T) {
	tests := []struct {
		name       string
		payload    []byte
		wantWidth  int
		wantHeight int
		wantErr    bool
	}{
		{name: "80x24", payload: []byte{0, 80, 0, 24}, wantWidth: 80, wantHeight: 24},
		{name: "high bytes", payload: []byte{1, 0, 0, 50}, wantWidth: 256, wantHeight: 50},
		{name: "trailing IAC SE", payload: []byte{0, 120, 0, 40, TELNET_IAC, TELNET_SE}, wantWidth: 120, wantHeight: 40},
		{name: "three bytes", payload: []byte{0, 80, 0}, wantErr: true},
		{name: "two bytes", payload: []byte{0, 80}, wantErr: true},
		{name: "empty", payload: []byte{}, wantErr: true},
		{name: "nil", payload: nil, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			width, height, err := TelnetParseScreenSizePayload(tt.payload)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantWidth, width)
			assert.Equal(t, tt.wantHeight, height)
		})
	}
}
