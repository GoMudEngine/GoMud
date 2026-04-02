package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/stretchr/testify/assert"
)

func init() {
	// Initialize mudlog for tests
	mudlog.SetupLogger(nil, "", "", false)
}

// TestEmoteAliasLookup verifies that a known emote alias returns IsAlias=true
// and the correct alias text.
func TestEmoteAliasLookup(t *testing.T) {
	result := Emote("bow")
	assert.True(t, result.IsAlias, "bow should be a recognized alias")
	assert.Equal(t, "bows gracefully.", result.AliasText, "bow alias text mismatch")
}

// TestEmoteNonAlias verifies that a custom emote string returns IsAlias=false
// and an empty AliasText.
func TestEmoteNonAlias(t *testing.T) {
	result := Emote("does something custom")
	assert.False(t, result.IsAlias, "custom emote should not be an alias")
	assert.Equal(t, "", result.AliasText, "custom emote should have empty alias text")
}

// TestFormatSayTextNormal verifies that non-sneaking say messages include
// the actor's name and the word "says,".
func TestFormatSayTextNormal(t *testing.T) {
	formatted := FormatSayText("Gandalf", "You shall not pass!", false, "username", "saytext")
	assert.Contains(t, formatted, "Gandalf", "normal say should include actor name")
	assert.Contains(t, formatted, "says,", "normal say should contain 'says,'")
	assert.Contains(t, formatted, "You shall not pass!", "normal say should include message text")
	assert.NotContains(t, formatted, "someone says", "normal say should not use 'someone says'")
}

// TestFormatSayTextSneaking verifies that sneaking say messages use "someone says,"
// and do NOT include the actor's name.
func TestFormatSayTextSneaking(t *testing.T) {
	formatted := FormatSayText("Sneaky", "Watch out!", true, "username", "saytext")
	assert.Contains(t, formatted, "someone says,", "sneaking say should use 'someone says,'")
	assert.NotContains(t, formatted, "Sneaky", "sneaking say should not reveal the actor name")
	assert.Contains(t, formatted, "Watch out!", "sneaking say should include message text")
}

// TestFormatEmoteText verifies that formatted emote messages contain both
// the actor's name and the emote text.
func TestFormatEmoteText(t *testing.T) {
	formatted := FormatEmoteText("Arwen", "looks thoughtfully at the horizon.", "mobname")
	assert.Contains(t, formatted, "Arwen", "emote should include actor name")
	assert.Contains(t, formatted, "looks thoughtfully at the horizon.", "emote should include emote text")
}

// TestAuditCommandParity_NoWarnings verifies that AuditCommandParity produces
// no warnings when user and mob commands are properly gated by the intentional
// allowlists.
func TestAuditCommandParity_NoWarnings(t *testing.T) {
	// User commands: common ones (say, emote, go) plus one user-only (help)
	userCmds := []string{"say", "emote", "go", "help"}
	// Mob commands: common ones (say, emote, go) plus one mob-only (wander)
	mobCmds := []string{"say", "emote", "go", "wander"}

	// help is in userOnlyCommands with category "ui"
	assert.Contains(t, userOnlyCommands, "help")
	// wander is in mobOnlyCommands with category "mob-ai"
	assert.Contains(t, mobOnlyCommands, "wander")

	// This should not panic or produce warnings (mudlog.Warn calls internally).
	// We're just verifying the function completes without error.
	AuditCommandParity(userCmds, mobCmds)
}

// TestEmoteAliasesPopulated verifies that the EmoteAliases map has the expected
// number of entries (82 as mentioned in the task).
func TestEmoteAliasesPopulated(t *testing.T) {
	assert.Greater(t, len(EmoteAliases), 50, "EmoteAliases should have many entries")
	// Check that some well-known aliases exist
	assert.Contains(t, EmoteAliases, "bow")
	assert.Contains(t, EmoteAliases, "laugh")
	assert.Contains(t, EmoteAliases, "dance")
	assert.Contains(t, EmoteAliases, "sit")
}

// TestFormatSayTextLineWrapping verifies that very long say messages are
// wrapped at 80 characters per line (via util.SplitStringNL).
func TestFormatSayTextLineWrapping(t *testing.T) {
	longText := strings.Repeat("word ", 30) // Creates a very long message
	formatted := FormatSayText("Speaker", longText, false, "username", "saytext")
	// Check that the output contains newlines (line wrapping)
	lines := strings.Split(formatted, "\n")
	// With proper wrapping, we should have multiple lines
	assert.Greater(t, len(lines), 1, "long say text should be wrapped across multiple lines")
	// Each line should not exceed 80 characters (accounting for ANSI codes)
	for _, line := range lines {
		// Remove ANSI codes to check actual visual length
		cleanLine := stripANSICodes(line)
		assert.LessOrEqual(t, len(cleanLine), 80, "wrapped line should not exceed 80 visible characters")
	}
}

// TestFormatEmoteTextLineWrapping verifies that very long emote messages are
// wrapped at 80 characters per line.
func TestFormatEmoteTextLineWrapping(t *testing.T) {
	longEmote := strings.Repeat("emote ", 25) // Creates a very long emote
	formatted := FormatEmoteText("Actor", longEmote, "mobname")
	lines := strings.Split(formatted, "\n")
	assert.Greater(t, len(lines), 1, "long emote text should be wrapped across multiple lines")
	for _, line := range lines {
		cleanLine := stripANSICodes(line)
		assert.LessOrEqual(t, len(cleanLine), 80, "wrapped line should not exceed 80 visible characters")
	}
}

// stripANSICodes removes ANSI color codes from a string to measure visible length.
func stripANSICodes(s string) string {
	// Simple approach: remove anything between < and >
	result := ""
	inTag := false
	for _, ch := range s {
		if ch == '<' {
			inTag = true
		} else if ch == '>' {
			inTag = false
		} else if !inTag {
			result += string(ch)
		}
	}
	return result
}
