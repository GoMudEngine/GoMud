package textutil

// SendTextConfig holds the messaging functions for a text send operation.
// This avoids import cycles — callers pass in their send functions.
type SendTextConfig struct {
	UserSendFunc func(msg string)              // sends to the acting user/mob owner
	RoomSendFunc func(msg string, skip ...int) // sends to the room, excluding user
	ExcludeId    int                           // user ID to exclude from room messages
}

// SendPhaseText substitutes tokens and sends user/room text for a spell
// or buff phase. Empty text = no message.
//
// The colorName parameter is retained for caller compatibility but is
// now ignored: pre-chunk-7 callers wrapped the message in an inline
// stretched color pattern (e.g., "pink" / "cyan") which clobbered the
// Category-based color the messaging pipeline applies via its color
// stage. Coloring now comes exclusively from the cfg.UserSendFunc /
// RoomSendFunc closure's Category. Drop the colorName arg in a future
// cleanup pass.
func SendPhaseText(userText, roomText string, ctx TokenContext, colorName string, cfg SendTextConfig) {
	_ = colorName // intentionally unused — see doc comment
	if userText != "" && cfg.UserSendFunc != nil {
		cfg.UserSendFunc(SubstituteTokens(userText, ctx))
	}
	if roomText != "" && cfg.RoomSendFunc != nil {
		cfg.RoomSendFunc(SubstituteTokens(roomText, ctx), cfg.ExcludeId)
	}
}
