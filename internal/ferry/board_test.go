package ferry

import (
	"strings"
	"testing"
)

func TestNotDockedQuoteFormat(t *testing.T) {
	got := formatNotDockedQuote("the Test Packet", 4, "PM")
	if !strings.Contains(got, "the Test Packet") || !strings.Contains(got, "4 PM") {
		t.Fatalf("quote missing vessel name or time: %q", got)
	}
}
