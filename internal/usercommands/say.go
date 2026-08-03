package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Say(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>. You can only send <ansi fg="command">whisper</ansi>'s to Admins and Moderators.`)
		return true, nil
	}

	if user.Character.HasBuffFlag(buffs.Drunk) {
		rest = drunkify(rest)
	}

	// Neutralise <ansi> markup before the text is interpolated into the room
	// and self templates below. Escaped last, after drunkify, so no later
	// transform can reassemble a tag. See util.EscapeAnsiTags.
	rest = util.EscapeAnsiTags(rest)

	actor := &actions.UserActor{User: user, Room: room}
	result := actions.Say(actor, rest)

	roomMsg := actions.FormatSayText(user.Character.Name, result.Text, result.IsSneaking, "username", "saytext")
	room.SendTextCommunication(roomMsg, user.UserId)

	selfMsg := fmt.Sprintf(`You say, "<ansi fg="saytext">%s</ansi>"`, result.Text)
	user.SendText(messaging.CategorySpeech, util.SplitStringNL(selfMsg, 80))

	return true, nil
}

func drunkify(sentence string) string {

	var drunkSentence strings.Builder
	isStartOfWord := true
	sentenceLength := len(sentence)
	insertedHiccup := false

	for i, char := range sentence {
		// Randomly decide whether to modify the character
		if util.Rand(10) < 3 || (!insertedHiccup || i == sentenceLength-1) {
			switch char {
			case 's':
				if isStartOfWord {
					drunkSentence.WriteString("sss")
				} else {
					drunkSentence.WriteString("sh")
				}
			case 'S':
				drunkSentence.WriteString("Sh")
			default:
				drunkSentence.WriteRune(char)
			}

			// Insert a hiccup in the middle of the sentence
			if !insertedHiccup && i >= sentenceLength/2 {
				drunkSentence.WriteString(" *hiccup* ")
				insertedHiccup = true
			}
		} else {
			drunkSentence.WriteRune(char)
		}

		// Update isStartOfWord based on spaces and punctuation
		if char == ' ' || char == '.' || char == '!' || char == '?' || char == ',' {
			isStartOfWord = true
		} else {
			isStartOfWord = false
		}
	}

	return drunkSentence.String()
}
