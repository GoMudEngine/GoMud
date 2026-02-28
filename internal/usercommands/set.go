package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Set(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(strings.ToLower(rest))

	if len(args) == 0 {
		displaySetStatus(user)
		return true, nil
	}

	setTarget := args[0]
	args = args[1:]

	switch setTarget {
	case `description`:
		return cmdSetDescription(user, rest, setTarget)
	case `auction`:
		return cmdSetToggle(user, `auction`, `Auctions`, true)
	case `shortadjectives`:
		return cmdSetToggle(user, `shortadjectives`, `Short Adjectives`, false)
	case `tinymap`:
		return cmdSetToggle(user, `tinymap`, `Tinymap`, true)
	case `screenreader`:
		return cmdSetScreenReader(user)
	case `prompt`:
		return cmdSetPrompt(user, rest, setTarget, args)
	case `fprompt`:
		return cmdSetFprompt(user, rest, setTarget, args)
	case `wimpy`:
		return cmdSetWimpy(user, rest, setTarget, args)
	default:
		// Are they setting a macro? // setTarget should be "=1" etc
		if len(setTarget) == 2 && setTarget[0] == '=' {
			return cmdSetMacro(user, setTarget, args)
		}
	}

	return true, nil
}

func displaySetStatus(user *users.UserRecord) {
	c := configs.GetTextFormatsConfig()

	user.SendText(`<ansi fg="yellow-bold">description:</ansi>`)
	user.SendText(`<ansi fg="yellow">` + util.SplitStringNL(user.Character.Description, 80) + `</ansi>`)
	user.SendText(``)

	user.SendText(`<ansi fg="yellow-bold">ScreenReader:</ansi> `)
	if user.ScreenReader {
		user.SendText(`<ansi fg="green">ON</ansi>`)
	} else {
		user.SendText(`<ansi fg="red">OFF</ansi>`)
	}
	user.SendText(``)

	displayBoolSetting(user, `auction`, `auction`)
	displayBoolSetting(user, `shortadjectives`, `shortadjectives`)
	displayBoolSetting(user, `tinymap`, `tinymap`)

	currentPrompt := user.GetConfigOption(`prompt`)
	if currentPrompt == nil {
		currentPrompt = c.Prompt.String()
	}
	user.SendText(`<ansi fg="yellow-bold">prompt: </ansi> `)
	user.SendText(currentPrompt.(string))
	user.SendText(``)

	currentPrompt = user.GetConfigOption(`fprompt`)
	if currentPrompt == nil {
		currentPrompt = c.FightPrompt.String()
	}
	user.SendText(`<ansi fg="yellow-bold">fprompt:</ansi> `)
	user.SendText(currentPrompt.(string))
	user.SendText(``)

	currentWimpy := user.GetConfigOption(`wimpy`)
	if currentWimpy == nil {
		currentWimpy = 0
	}
	user.SendText(`<ansi fg="yellow-bold">wimpy:</ansi> `)
	user.SendText(fmt.Sprintf(`%d%%`, currentWimpy.(int)))
	user.SendText(``)

	user.SendText(`See: <ansi fg="command">help set</ansi>`)
}

func displayBoolSetting(user *users.UserRecord, settingName string, label string) {
	on := user.GetConfigOption(settingName)
	onTxt := `<ansi fg="red">OFF</ansi>`
	if on == nil || on.(bool) {
		onTxt = `<ansi fg="green">ON</ansi>`
	}
	user.SendText(`<ansi fg="yellow-bold">` + label + `:</ansi> `)
	user.SendText(onTxt)
	user.SendText(``)
}

func cmdSetDescription(user *users.UserRecord, rest string, setTarget string) (bool, error) {
	rest = strings.TrimSpace(rest[len(setTarget):])
	if len(rest) > 1024 {
		rest = rest[:1024]
	}
	user.Character.Description = rest

	user.SendText("Description set. Look at yourself to confirm.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `description`,
	})

	return true, nil
}

func cmdSetToggle(user *users.UserRecord, settingName string, displayName string, defaultVal bool) (bool, error) {
	on := user.GetConfigOption(settingName)
	if on == nil {
		on = defaultVal
	}
	if !on.(bool) {
		on = true
		user.SendText(displayName + ` toggled <ansi fg="red">ON</ansi>.`)
	} else {
		on = false
		user.SendText(displayName + ` toggled <ansi fg="red">OFF</ansi>.`)
	}

	user.SetConfigOption(settingName, on)

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   settingName,
	})

	return true, nil
}

func cmdSetScreenReader(user *users.UserRecord) (bool, error) {
	if user.ScreenReader {
		user.SendText(`ScreenReader mode toggled <ansi fg="red">OFF</ansi>.`)
	} else {
		user.SendText(`ScreenReader mode toggled <ansi fg="red">ON</ansi>.`)
	}
	user.ScreenReader = !user.ScreenReader

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `screenreader`,
	})

	return true, nil
}

func cmdSetPrompt(user *users.UserRecord, rest string, setTarget string, args []string) (bool, error) {
	c := configs.GetTextFormatsConfig()

	if len(args) < 1 {
		currentPrompt := user.GetConfigOption(`prompt`)
		if currentPrompt == nil {
			currentPrompt = c.Prompt.String()
		}
		user.SendText("Your current prompt:\n")
		user.SendText(currentPrompt.(string))
		user.SendText("\n" + `Type <ansi fg="command">help set-prompt</ansi> for more info on customizing prompts.` + "\n")
		return true, nil
	}

	promptStr := rest[len(setTarget)+1:]

	if promptStr == `default` {
		user.SetConfigOption(`prompt`, nil)
		user.SetConfigOption(`prompt-compiled`, nil)
		user.SetConfigOption(`prompt`, c.Prompt.String())
		user.SetConfigOption(`prompt-compiled`, util.ConvertColorShortTags(c.Prompt.String()))
	} else if promptStr == `none` {
		user.SetConfigOption(`prompt`, ``)
		user.SetConfigOption(`prompt-compiled`, ``)
	} else {
		user.SetConfigOption(`prompt`, promptStr)
		user.SetConfigOption(`prompt-compiled`, util.ConvertColorShortTags(promptStr))
	}

	user.SendText("Prompt set.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `prompt`,
	})

	return true, nil
}

func cmdSetFprompt(user *users.UserRecord, rest string, setTarget string, args []string) (bool, error) {

	if len(args) < 1 {
		customPrompt := user.GetConfigOption(`fprompt`)
		user.SendText(`<ansi fg="yellow-bold">Fight Prompt:</ansi>`)
		if customPrompt == nil {
			user.SendText(`  <ansi fg="8">[system default - toggle-driven]</ansi>`)
		} else {
			user.SendText(`  ` + customPrompt.(string))
		}
		user.SendText(``)
		user.SendText(`<ansi fg="yellow">Toggle Settings</ansi> <ansi fg="8">(set fprompt tog <name> to change):</ansi>`)
		toggleList := []struct{ name, desc string }{
			{"bars", "HP/SP/CP vital bars"},
			{"pos", "Your combat position"},
			{"target", "Target name"},
			{"targethealth", "Target health description"},
			{"targetpos", "Target position"},
			{"tank", "Party tank info"},
		}
		for _, t := range toggleList {
			val := user.GetConfigOption(`fprompt-tog-` + t.name)
			on := val == nil || val.(bool)
			state := `<ansi fg="green">[ON] </ansi>`
			if !on {
				state = `<ansi fg="red">[OFF]</ansi>`
			}
			user.SendText(fmt.Sprintf(`  <ansi fg="magenta">%-14s</ansi> %s %s`, t.name, state, t.desc))
		}
		user.SendText(``)
		user.SendText(`Type <ansi fg="command">help set-prompt</ansi> for more info on customizing prompts.`)
		return true, nil
	}

	promptStr := rest[len(setTarget)+1:]

	// Toggle subcommand: set fprompt tog <element>
	if strings.HasPrefix(promptStr, `tog `) {
		element := strings.TrimSpace(promptStr[4:])
		validToggles := map[string]string{
			"bars":         "HP/SP/CP vital bars",
			"pos":          "Your combat position",
			"target":       "Target name",
			"targethealth": "Target health description",
			"targetpos":    "Target position",
			"tank":         "Party tank info",
		}
		desc, valid := validToggles[element]
		if !valid {
			user.SendText(`Unknown toggle. Valid options: bars, pos, target, targethealth, targetpos, tank`)
			return true, nil
		}
		key := `fprompt-tog-` + element
		currentVal := user.GetConfigOption(key)
		current := true // default on
		if currentVal != nil {
			if b, ok := currentVal.(bool); ok {
				current = b
			}
		}
		user.SetConfigOption(key, !current)
		// Rebuild and cache the toggle-driven template immediately
		user.SetConfigOption(`fprompt-default-compiled`, util.ConvertColorShortTags(user.BuildFightPromptTemplate()))
		if !current {
			user.SendText(fmt.Sprintf(`Fight prompt: %s <ansi fg="green">[ON]</ansi>`, desc))
		} else {
			user.SendText(fmt.Sprintf(`Fight prompt: %s <ansi fg="red">[OFF]</ansi>`, desc))
		}
		return true, nil
	}

	if promptStr == `default` {
		user.SetConfigOption(`fprompt`, nil)
		user.SetConfigOption(`fprompt-compiled`, nil)
		user.SetConfigOption(`fprompt-default-compiled`, nil)
		for _, key := range []string{`bars`, `pos`, `target`, `targethealth`, `targetpos`, `tank`} {
			user.SetConfigOption(`fprompt-tog-`+key, nil)
		}
	} else if promptStr == `none` {
		user.SetConfigOption(`fprompt`, ``)
		user.SetConfigOption(`fprompt-compiled`, ``)
	} else {
		user.SetConfigOption(`fprompt`, promptStr)
		user.SetConfigOption(`fprompt-compiled`, util.ConvertColorShortTags(promptStr))
	}

	user.SendText("fprompt set.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `fprompt`,
	})

	return true, nil
}

func cmdSetWimpy(user *users.UserRecord, rest string, setTarget string, args []string) (bool, error) {

	if len(args) < 1 {
		currentWimpy := user.GetConfigOption(`wimpy`)
		if currentWimpy == nil {
			currentWimpy = 0
		}
		user.SendText("Your current wimpy:\n")
		user.SendText(fmt.Sprintf(`%d%%`, currentWimpy.(int)))
		user.SendText("\n" + `Type <ansi fg="command">help wimpy</ansi> to learn about the wimpy setting.` + "\n")
		return true, nil
	}

	wimpyStr := rest[len(setTarget)+1:]
	wimipyInt, _ := strconv.Atoi(wimpyStr)

	if wimipyInt == 0 {
		user.SetConfigOption(`wimpy`, nil)
	} else {
		user.SetConfigOption(`wimpy`, wimipyInt)
	}

	user.SendText("wimpy set.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `wimpy`,
	})

	return true, nil
}

func cmdSetMacro(user *users.UserRecord, setTarget string, args []string) (bool, error) {

	setVal := strings.Join(args, ` `)

	macroNum, _ := strconv.Atoi(string(setTarget[1]))
	if macroNum == 0 {
		user.SendText("Invalid macro number supplied.")
		return true, nil
	}

	if user.Macros == nil {
		user.Macros = make(map[string]string)
	}

	// Keep macros small enough.
	if len(setVal) > 128 {
		setVal = setVal[:128]
	}

	if len(setVal) == 0 {
		delete(user.Macros, setTarget)
		user.SendText(fmt.Sprintf(`Macro <ansi fg="command">=%d</ansi> deleted.`, macroNum))
	} else {

		allComands := strings.Split(setVal, ";")
		if len(allComands) > 10 {
			user.SendText(`Macros are limited to 10 commands.`)
			return true, nil
		}

		finalMacroCommands := []string{}

		for _, cmd := range allComands {

			if len(cmd) > 0 {
				if cmd[0] == '=' {
					user.SendText(`You cannot reference macros inside of a macro`)
					return true, nil
				}
				finalMacroCommands = append(finalMacroCommands, cmd)
			}

		}

		if len(finalMacroCommands) < 1 {
			user.SendText(`There was a problem setting your macro.`)
			return true, nil
		}

		user.Macros[setTarget] = strings.Join(finalMacroCommands, `;`)

		user.SendText(fmt.Sprintf(`Macro set. Type <ansi fg="command">=%d</ansi> or (if your terminal supports it) press <ansi fg="command">F%d</ansi> to use it.`, macroNum, macroNum))
	}

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `macro`,
	})

	return true, nil
}
