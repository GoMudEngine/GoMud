package term

import (
	"bytes"
	"testing"
)

func TestEncodeMSSPPayload_SingleField(t *testing.T) {
	got := EncodeMSSPPayload([]MSSPField{{Name: "PLAYERS", Values: []string{"3"}}})
	want := []byte{MSSP_VAR}
	want = append(want, []byte("PLAYERS")...)
	want = append(want, MSSP_VAL)
	want = append(want, []byte("3")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("single field mismatch\n got=%v\nwant=%v", got, want)
	}
}

func TestEncodeMSSPPayload_MultiValue(t *testing.T) {
	got := EncodeMSSPPayload([]MSSPField{{Name: "GAMEPLAY", Values: []string{"Adventure", "Roleplaying"}}})
	want := []byte{MSSP_VAR}
	want = append(want, []byte("GAMEPLAY")...)
	want = append(want, MSSP_VAL)
	want = append(want, []byte("Adventure")...)
	want = append(want, MSSP_VAL)
	want = append(want, []byte("Roleplaying")...)
	if !bytes.Equal(got, want) {
		t.Fatalf("multi-value mismatch\n got=%v\nwant=%v", got, want)
	}
}

func TestEncodeMSSPPayload_EscapesIAC(t *testing.T) {
	// A raw IAC (0xFF) byte inside a value must be doubled so it can't corrupt
	// the sub-negotiation stream.
	got := EncodeMSSPPayload([]MSSPField{{Name: "X", Values: []string{string([]byte{TELNET_IAC})}}})
	want := []byte{MSSP_VAR, 'X', MSSP_VAL, TELNET_IAC, TELNET_IAC}
	if !bytes.Equal(got, want) {
		t.Fatalf("IAC escape mismatch\n got=%v\nwant=%v", got, want)
	}
}

func TestIsMSSPCommand(t *testing.T) {
	if !IsMSSPCommand([]byte{TELNET_IAC, TELNET_DO, MSSP}) {
		t.Fatal("IAC DO MSSP should be recognized as an MSSP command")
	}
	if IsMSSPCommand([]byte{TELNET_IAC, TELNET_DO, MSP}) {
		t.Fatal("IAC DO MSP must not be recognized as MSSP")
	}
}
