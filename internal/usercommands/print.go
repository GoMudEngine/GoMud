package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Prints message to screen
func Print(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	events.AddToQueue(events.Message{
		UserId: user.UserId,
		Text:   rest,
	})

	return true, nil
}

// PrintLine (command `printline`) is just a simple measuring tool/ruler for layout purposes
func PrintLine(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	sectionSize := 20

	lineLength, _ := strconv.Atoi(rest)
	if lineLength > 0 {
		if lineLength > 240 {
			lineLength = 240
		}

		sections := lineLength / sectionSize

		finalOutput := ``

		for i := 0; i < sections; i++ {
			sectionStr := ``

			sectionStr += strings.Repeat(`=`, sectionSize)
			countStr := fmt.Sprintf(` %d |`, (i+1)*sectionSize)
			sectionStr = sectionStr[:sectionSize-len(countStr)] + countStr

			finalOutput += sectionStr
		}

		remaining := lineLength - len(finalOutput)

		if remaining > 0 {
			finalOutput += strings.Repeat(`=`, remaining)
			finalOutput = finalOutput[:len(finalOutput)-1] + `|`
		}

		finalOutput = strings.ReplaceAll(finalOutput, `=`, `<ansi fg="8">=</ansi>`)
		user.SendTextLegacy(finalOutput)
	}

	return true, nil
}
