package inputhandlers

import (
	"testing"
	"unicode/utf8"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/term"
)

func TestCleanserInputHandler_UTF8Backspace(t *testing.T) {
	tests := []struct {
		name           string
		initialBuffer  string
		expectedBuffer string
		description    string
	}{
		{
			name:           "ASCII backspace",
			initialBuffer:  "hello",
			expectedBuffer: "hell",
			description:    "Should remove single ASCII character",
		},
		{
			name:           "UTF8 2-byte character",
			initialBuffer:  "hellö", // ö is 2 bytes in UTF-8
			expectedBuffer: "hell",
			description:    "Should remove complete 2-byte UTF-8 character",
		},
		{
			name:           "UTF8 3-byte character",
			initialBuffer:  "hello€", // € is 3 bytes in UTF-8
			expectedBuffer: "hello",
			description:    "Should remove complete 3-byte UTF-8 character",
		},
		{
			name:           "UTF8 4-byte character",
			initialBuffer:  "hello🚀", // 🚀 is 4 bytes in UTF-8
			expectedBuffer: "hello",
			description:    "Should remove complete 4-byte UTF-8 character",
		},
		{
			name:           "Mixed UTF8 characters",
			initialBuffer:  "héllö🚀",
			expectedBuffer: "héllö",
			description:    "Should remove last character from mixed UTF-8 string",
		},
		{
			name:           "Empty buffer",
			initialBuffer:  "",
			expectedBuffer: "",
			description:    "Should handle empty buffer gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			clientInput := &connections.ClientInput{
				ConnectionId: 1,
				DataIn:       []byte{term.ASCII_BACKSPACE}, // Simulate backspace input
				Buffer:       []byte(tt.initialBuffer),
				EnterPressed: false,
			}
			sharedState := make(map[string]any)

			// Verify initial buffer is valid UTF-8
			if !utf8.Valid(clientInput.Buffer) {
				t.Fatalf("Initial buffer is not valid UTF-8: %q", tt.initialBuffer)
			}

			// Call the handler
			CleanserInputHandler(clientInput, sharedState)

			// Verify result
			result := string(clientInput.Buffer)
			if result != tt.expectedBuffer {
				t.Errorf("Expected buffer %q, got %q", tt.expectedBuffer, result)
			}

			// Verify the resulting buffer is still valid UTF-8
			if !utf8.Valid(clientInput.Buffer) {
				t.Errorf("Resulting buffer is not valid UTF-8: %q (bytes: %v)", result, clientInput.Buffer)
			}

			// Verify BSPressed flag was set
			if !clientInput.BSPressed {
				t.Error("Expected BSPressed to be true")
			}
		})
	}
}

func TestCleanserInputHandler_NoBackspace(t *testing.T) {
	// Test that normal input is handled correctly
	clientInput := &connections.ClientInput{
		ConnectionId: 1,
		DataIn:       []byte("hello🚀"), // Multi-byte UTF-8 input
		Buffer:       []byte("existing"),
		EnterPressed: false,
	}
	sharedState := make(map[string]any)

	CleanserInputHandler(clientInput, sharedState)

	// Should append the new input to existing buffer
	expected := "existinghello🚀"
	result := string(clientInput.Buffer)
	if result != expected {
		t.Errorf("Expected buffer %q, got %q", expected, result)
	}

	// Verify the resulting buffer is valid UTF-8
	if !utf8.Valid(clientInput.Buffer) {
		t.Errorf("Resulting buffer is not valid UTF-8: %q", result)
	}

	// Verify BSPressed flag was not set
	if clientInput.BSPressed {
		t.Error("Expected BSPressed to be false")
	}
}
