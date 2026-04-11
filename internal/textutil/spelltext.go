package textutil

import (
	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
)

// SendTextConfig holds the messaging functions for a text send operation.
// This avoids import cycles — callers pass in their send functions.
type SendTextConfig struct {
	UserSendFunc func(msg string)              // sends to the acting user/mob owner
	RoomSendFunc func(msg string, skip ...int) // sends to the room, excluding user
	ExcludeId    int                           // user ID to exclude from room messages
}

// SendPhaseText substitutes tokens, applies color wrapping, and sends
// user/room text for a spell or buff phase. Empty text = no message.
func SendPhaseText(userText, roomText string, ctx TokenContext, colorName string, cfg SendTextConfig) {
	if userText != "" && cfg.UserSendFunc != nil {
		msg := SubstituteTokens(userText, ctx)
		if colorName != "" {
			msg = colorpatterns.ApplyColorPattern(msg, colorName, colorpatterns.Stretch)
		}
		cfg.UserSendFunc(msg)
	}
	if roomText != "" && cfg.RoomSendFunc != nil {
		msg := SubstituteTokens(roomText, ctx)
		if colorName != "" {
			msg = colorpatterns.ApplyColorPattern(msg, colorName, colorpatterns.Stretch)
		}
		cfg.RoomSendFunc(msg, cfg.ExcludeId)
	}
}
