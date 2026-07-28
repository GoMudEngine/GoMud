package inputhandlers

import (
	"strings"
	"testing"
)

// The plaintext MSSP variant: a crawler sends "mssp-request" at the login
// prompt and expects an MSSP-REPLY-START / KEY<TAB>VALUE / MSSP-REPLY-END
// block (this is exactly what Grapevine's checker parses when the telnet
// option produces nothing: \r stripped, lines split on \n, fields on \t).
func TestBuildMSSPTextReply_Format(t *testing.T) {
	reply := string(buildMSSPTextReply(buildMSSPFields(baseInputs())))

	// Parse the way Grapevine's telnet-elixir MSSP.parse_text does.
	lines := strings.Split(strings.ReplaceAll(reply, "\r", ""), "\n")
	inBlock := false
	got := map[string]string{}
	sawEnd := false
	for _, ln := range lines {
		switch {
		case ln == "MSSP-REPLY-START":
			inBlock = true
		case ln == "MSSP-REPLY-END":
			sawEnd = true
			inBlock = false
		case inBlock && ln != "":
			parts := strings.SplitN(ln, "\t", 2)
			if len(parts) != 2 {
				t.Errorf("field line %q lacks a tab separator", ln)
				continue
			}
			got[parts[0]] = parts[1]
		}
	}

	if !sawEnd {
		t.Fatal("reply missing MSSP-REPLY-END")
	}
	if got["NAME"] != "Delusions of Grandeur" {
		t.Errorf("NAME = %q", got["NAME"])
	}
	if got["PLAYERS"] != "3" {
		t.Errorf("PLAYERS = %q", got["PLAYERS"])
	}
	// Multi-value fields join with tabs (Grapevine splits [name | values...]).
	if got["GAMEPLAY"] != "Adventure\tRoleplaying" {
		t.Errorf("GAMEPLAY = %q", got["GAMEPLAY"])
	}
}

func TestBuildMSSPTextReply_NilWhenDisabled(t *testing.T) {
	in := baseInputs()
	in.Enabled = false
	if reply := buildMSSPTextReply(buildMSSPFields(in)); reply != nil {
		t.Errorf("disabled MSSP must yield a nil text reply, got %q", string(reply))
	}
}
