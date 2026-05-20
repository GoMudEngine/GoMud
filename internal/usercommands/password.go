package usercommands

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Password(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// Get if already exists, otherwise create new
	cmdPrompt, _ := user.StartPrompt(`password`, rest)

	question := cmdPrompt.Ask(`What is your current password?`, []string{})
	if !question.Done {
		return true, nil
	}

	if !user.PasswordMatches(question.Response) {
		user.SendText(messaging.CategorySystem, `<ansi fg="alert-5">Sorry, your password was incorrect.</ansi>`)
		user.ClearPrompt()
		return true, nil
	}

	question = cmdPrompt.Ask(`What new password would you like?`, []string{})
	if !question.Done {
		return true, nil
	}

	newPW := question.Response

	question = cmdPrompt.Ask(`Confirm the change by entered the new password one more time.`, []string{})
	if !question.Done {
		return true, nil
	}

	newPWConfirm := question.Response

	if newPW != newPWConfirm {
		user.SendText(messaging.CategorySystem, `<ansi fg="alert-5">Sorry, your new password and the confirmation password did not match.</ansi>`)
		user.ClearPrompt()
		return true, nil
	}

	if err := user.SetPassword(newPW); err != nil {
		user.SendText(messaging.CategorySystem, `<ansi fg="alert-5">` + err.Error() + `</ansi>`)
		user.ClearPrompt()
		return true, nil
	}

	users.SaveUser(*user)

	user.SendText(messaging.CategorySystem, `<ansi fg="alert-1">Your password has been changed!</ansi>`)

	return true, nil
}
