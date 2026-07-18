package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/util"
)

// EmoteResult contains the result of an Emote alias lookup.
type EmoteResult struct {
	IsAlias   bool
	AliasText string
}

// Emote checks whether rest matches a known emote alias. If it does,
// IsAlias is true and AliasText holds the pre-written description.
// The caller is responsible for all room display and player feedback.
func Emote(rest string) EmoteResult {
	if aliasText, ok := EmoteAliases[rest]; ok {
		return EmoteResult{IsAlias: true, AliasText: aliasText}
	}
	return EmoteResult{IsAlias: false}
}

// FormatEmoteText formats an emote action for room display.
// nameColor is "username" for players, "mobname" for mobs.
func FormatEmoteText(name string, emoteText string, nameColor string) string {
	// Emote body is a warm tan (256-color 137, #af875f) rather than the old
	// dark navy (256-color 20, #0000d7). Blue-on-black is barely legible,
	// especially on a laptop in a bright room (2026-07-18 accessibility report),
	// and 137 is the palette's intended `mob-emote` tone that this formatter had
	// never actually used.
	msg := fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="137">%s</ansi>`, nameColor, name, emoteText)
	return util.SplitStringNL(msg, 80)
}
