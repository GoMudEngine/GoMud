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
	msg := fmt.Sprintf(`<ansi fg="%s">%s</ansi> <ansi fg="20">%s</ansi>`, nameColor, name, emoteText)
	return util.SplitStringNL(msg, 80)
}
