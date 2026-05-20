package usercommands

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
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
	case `hints`:
		return cmdSetToggle(user, `hints`, `Hints`, true)
	case `screenreader`:
		return cmdSetScreenReader(user)
	case `charset`:
		return cmdSetCharset(user)
	case `prompt`:
		return cmdSetPrompt(user, rest, setTarget, args)
	case `fprompt`:
		return cmdSetFprompt(user, rest, setTarget, args)
	case `wimpy`:
		return cmdSetWimpy(user, rest, setTarget, args)
	case `submission`:
		return cmdSetSubmission(user, args)
	case `surrender`:
		return cmdSetSurrender(user, args)
	case `linewidth`:
		return cmdSetLineWidth(user, args)
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

	user.SendTextLegacy(`<ansi fg="yellow-bold">description:</ansi>`)
	user.SendTextLegacy(`<ansi fg="yellow">` + util.SplitStringNL(user.Character.Description, 80) + `</ansi>`)
	user.SendTextLegacy(``)

	user.SendTextLegacy(`<ansi fg="yellow-bold">ScreenReader:</ansi> `)
	if user.ScreenReader {
		user.SendTextLegacy(`<ansi fg="green">ON</ansi>`)
	} else {
		user.SendTextLegacy(`<ansi fg="red">OFF</ansi>`)
	}
	user.SendTextLegacy(``)

	user.SendTextLegacy(`<ansi fg="yellow-bold">charset:</ansi> `)
	if user.AsciiMode {
		user.SendTextLegacy(`<ansi fg="red">ASCII</ansi> (legacy client mode)`)
	} else {
		user.SendTextLegacy(`<ansi fg="green">UTF-8</ansi>`)
	}
	user.SendTextLegacy(``)

	displayBoolSetting(user, `auction`, `auction`)
	displayBoolSetting(user, `shortadjectives`, `shortadjectives`)
	displayBoolSetting(user, `tinymap`, `tinymap`)
	displayBoolSetting(user, `hints`, `hints`)

	currentPrompt := user.GetConfigOption(`prompt`)
	if currentPrompt == nil {
		currentPrompt = c.Prompt.String()
	}
	user.SendTextLegacy(`<ansi fg="yellow-bold">prompt: </ansi> `)
	user.SendTextLegacy(currentPrompt.(string))
	user.SendTextLegacy(``)

	currentPrompt = user.GetConfigOption(`fprompt`)
	if currentPrompt == nil {
		currentPrompt = c.FightPrompt.String()
	}
	user.SendTextLegacy(`<ansi fg="yellow-bold">fprompt:</ansi> `)
	user.SendTextLegacy(currentPrompt.(string))
	user.SendTextLegacy(``)

	currentWimpy := user.GetConfigOption(`wimpy`)
	if currentWimpy == nil {
		currentWimpy = 0
	}
	user.SendTextLegacy(`<ansi fg="yellow-bold">wimpy:</ansi> `)
	user.SendTextLegacy(fmt.Sprintf(`%d%%`, currentWimpy.(int)))
	user.SendTextLegacy(``)

	user.SendTextLegacy(`<ansi fg="yellow-bold">submission:</ansi> `)
	user.SendTextLegacy(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, user.Character.SubmissionPolicy))
	user.SendTextLegacy(``)

	user.SendTextLegacy(`<ansi fg="yellow-bold">surrender:</ansi> `)
	user.SendTextLegacy(fmt.Sprintf(`<ansi fg="green">%s</ansi>`, user.Character.SurrenderPolicy))
	user.SendTextLegacy(``)

	user.SendTextLegacy(`See: <ansi fg="command">help set</ansi>`)
}

func displayBoolSetting(user *users.UserRecord, settingName string, label string) {
	on := user.GetConfigOption(settingName)
	onTxt := `<ansi fg="red">OFF</ansi>`
	if on == nil || on.(bool) {
		onTxt = `<ansi fg="green">ON</ansi>`
	}
	user.SendTextLegacy(`<ansi fg="yellow-bold">` + label + `:</ansi> `)
	user.SendTextLegacy(onTxt)
	user.SendTextLegacy(``)
}

func cmdSetDescription(user *users.UserRecord, rest string, setTarget string) (bool, error) {
	rest = strings.TrimSpace(rest[len(setTarget):])
	if len(rest) > 1024 {
		rest = rest[:1024]
	}
	user.Character.Description = rest

	user.SendTextLegacy("Description set. Look at yourself to confirm.")

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
		user.SendTextLegacy(displayName + ` toggled <ansi fg="red">ON</ansi>.`)
	} else {
		on = false
		user.SendTextLegacy(displayName + ` toggled <ansi fg="red">OFF</ansi>.`)
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
		user.SendTextLegacy(`ScreenReader mode toggled <ansi fg="red">OFF</ansi>.`)
	} else {
		user.SendTextLegacy(`ScreenReader mode toggled <ansi fg="red">ON</ansi>.`)
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
		user.SendTextLegacy("Your current prompt:\n")
		user.SendTextLegacy(currentPrompt.(string))
		user.SendTextLegacy("\n" + `Type <ansi fg="command">help set-prompt</ansi> for more info on customizing prompts.` + "\n")
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

	user.SendTextLegacy("Prompt set.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `prompt`,
	})

	return true, nil
}

func cmdSetFprompt(user *users.UserRecord, rest string, setTarget string, args []string) (bool, error) {

	if len(args) < 1 {
		customPrompt := user.GetConfigOption(`fprompt`)
		user.SendTextLegacy(`<ansi fg="yellow-bold">Fight Prompt:</ansi>`)
		if customPrompt == nil {
			user.SendTextLegacy(`  <ansi fg="8">[system default - toggle-driven]</ansi>`)
		} else {
			user.SendTextLegacy(`  ` + customPrompt.(string))
		}
		user.SendTextLegacy(``)
		user.SendTextLegacy(`<ansi fg="yellow">Toggle Settings</ansi> <ansi fg="8">(set fprompt tog <name> to change):</ansi>`)
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
			user.SendTextLegacy(fmt.Sprintf(`  <ansi fg="magenta">%-14s</ansi> %s %s`, t.name, state, t.desc))
		}
		user.SendTextLegacy(``)
		user.SendTextLegacy(`Type <ansi fg="command">help set-prompt</ansi> for more info on customizing prompts.`)
		return true, nil
	}

	promptStr := rest[len(setTarget)+1:]

	// Toggle subcommand: set fprompt tog <element>  (or just: set fprompt <element>)
	validToggles := map[string]string{
		"bars":         "HP/SP/CP vital bars",
		"pos":          "Your combat position",
		"target":       "Target name",
		"targethealth": "Target health description",
		"targetpos":    "Target position",
		"tank":         "Party tank info",
	}
	var element string
	if strings.HasPrefix(promptStr, `tog `) {
		element = strings.TrimSpace(promptStr[4:])
	} else if _, ok := validToggles[promptStr]; ok {
		element = promptStr
	}
	if element != `` {
		desc, valid := validToggles[element]
		if !valid {
			user.SendTextLegacy(`Unknown toggle. Valid options: bars, pos, target, targethealth, targetpos, tank`)
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
			user.SendTextLegacy(fmt.Sprintf(`Fight prompt: %s <ansi fg="green">[ON]</ansi>`, desc))
		} else {
			user.SendTextLegacy(fmt.Sprintf(`Fight prompt: %s <ansi fg="red">[OFF]</ansi>`, desc))
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

	user.SendTextLegacy("fprompt set.")

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
		user.SendTextLegacy("Your current wimpy:\n")
		user.SendTextLegacy(fmt.Sprintf(`%d%%`, currentWimpy.(int)))
		user.SendTextLegacy("\n" + `Type <ansi fg="command">help wimpy</ansi> to learn about the wimpy setting.` + "\n")
		return true, nil
	}

	wimpyStr := rest[len(setTarget)+1:]
	wimpyInt, _ := strconv.Atoi(wimpyStr)

	if wimpyInt == 0 {
		user.SetConfigOption(`wimpy`, nil)
	} else {
		user.SetConfigOption(`wimpy`, wimpyInt)
	}

	user.SendTextLegacy("wimpy set.")

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `wimpy`,
	})

	return true, nil
}

func cmdSetCharset(user *users.UserRecord) (bool, error) {
	user.AsciiMode = !user.AsciiMode

	// Apply to active connection immediately
	cs := connections.GetClientSettings(user.ConnectionId())
	cs.AsciiMode = user.AsciiMode
	connections.OverwriteClientSettings(user.ConnectionId(), cs)

	if user.AsciiMode {
		user.SendTextLegacy("Charset mode set to <ansi fg=\"red\">ASCII</ansi>.")
		user.SendTextLegacy("Box-drawing characters will be converted to ASCII equivalents.")
		user.SendTextLegacy("Use <ansi fg=\"command\">set charset</ansi> again to switch back to UTF-8.")
	} else {
		user.SendTextLegacy("Charset mode set to <ansi fg=\"green\">UTF-8</ansi>.")
		user.SendTextLegacy("Full Unicode box-drawing characters will be displayed.")
	}

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   "charset",
	})

	return true, nil
}

func cmdSetSubmission(user *users.UserRecord, args []string) (bool, error) {
	if len(args) < 1 {
		user.SendTextLegacy(fmt.Sprintf("Your submission policy: <ansi fg=\"command\">%s</ansi>",
			user.Character.SubmissionPolicy))
		user.SendTextLegacy("Set with: <ansi fg=\"command\">set submission &lt;mercy|subdue|cripple|lethal&gt;</ansi>")
		user.SendTextLegacy("See <ansi fg=\"command\">help submission</ansi> for details.")
		return true, nil
	}

	newPolicy, ok := characters.ParseSubmissionPolicy(args[0])
	if !ok {
		user.SendTextLegacy(fmt.Sprintf("Unknown submission policy: <ansi fg=\"red\">%s</ansi>", args[0]))
		user.SendTextLegacy("Valid policies: mercy, subdue, cripple, lethal.")
		return true, nil
	}

	// First-time lethal confirmation: require the command twice before accepting.
	if newPolicy == characters.PolicyLethal && user.Character.SubmissionPolicy != characters.PolicyLethal {
		confirmed, _ := user.Character.GetMiscData("submission_lethal_confirmed").(string)
		if confirmed != "1" {
			pending, _ := user.Character.GetMiscData("submission_lethal_pending").(string)
			if pending != "1" {
				user.SendTextLegacy(`<ansi fg="yellow-bold">WARNING:</ansi> Setting your submission policy to` +
					` <ansi fg="red-bold">lethal</ansi> means your successful submissions will KILL` +
					` opponents — players included, with full deprogression and corpse loot.`)
				user.SendTextLegacy("If you're sure, run the command again to confirm.")
				user.Character.SetMiscData("submission_lethal_pending", "1")
				return true, nil
			}
			user.Character.SetMiscData("submission_lethal_confirmed", "1")
			user.Character.SetMiscData("submission_lethal_pending", nil)
		}
	}

	user.Character.SubmissionPolicy = newPolicy
	user.SendTextLegacy(fmt.Sprintf("Submission policy set to <ansi fg=\"command\">%s</ansi>.", newPolicy))
	return true, nil
}

func cmdSetSurrender(user *users.UserRecord, args []string) (bool, error) {
	if len(args) < 1 {
		user.SendTextLegacy(fmt.Sprintf("Your surrender policy: <ansi fg=\"command\">%s</ansi>",
			user.Character.SurrenderPolicy))
		user.SendTextLegacy("Set with: <ansi fg=\"command\">set surrender &lt;never|always|auto-tap-below &lt;N&gt;&gt;</ansi>")
		user.SendTextLegacy("See <ansi fg=\"command\">help surrender</ansi> for details.")
		return true, nil
	}

	// Join remaining args so "auto-tap-below 25" works as two tokens.
	policyStr := strings.Join(args, " ")
	newPolicy, ok := characters.ParseSurrenderPolicy(policyStr)
	if !ok {
		user.SendTextLegacy(fmt.Sprintf("Unknown surrender policy: <ansi fg=\"red\">%s</ansi>", policyStr))
		user.SendTextLegacy("Valid policies: never, always, auto-tap-below &lt;1-100&gt;.")
		return true, nil
	}

	user.Character.SurrenderPolicy = newPolicy
	user.SendTextLegacy(fmt.Sprintf("Surrender policy set to <ansi fg=\"command\">%s</ansi>.", newPolicy))
	return true, nil
}

func cmdSetLineWidth(user *users.UserRecord, args []string) (bool, error) {
	if len(args) < 1 {
		user.SendTextLegacy(fmt.Sprintf("Line width is currently %d.", user.GetLineWidth()))
		return true, nil
	}
	n, err := strconv.Atoi(args[0])
	if err != nil || n < 40 || n > 240 {
		user.SendTextLegacy("Line width must be a number between 40 and 240.")
		return true, nil
	}
	user.LineWidth = n
	user.SendTextLegacy(fmt.Sprintf("Line width set to %d.", n))

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `linewidth`,
	})

	return true, nil
}

func cmdSetMacro(user *users.UserRecord, setTarget string, args []string) (bool, error) {

	setVal := strings.Join(args, ` `)

	macroNum, _ := strconv.Atoi(string(setTarget[1]))
	if macroNum == 0 {
		user.SendTextLegacy("Invalid macro number supplied.")
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
		user.SendTextLegacy(fmt.Sprintf(`Macro <ansi fg="command">=%d</ansi> deleted.`, macroNum))
	} else {

		allComands := strings.Split(setVal, ";")
		if len(allComands) > 10 {
			user.SendTextLegacy(`Macros are limited to 10 commands.`)
			return true, nil
		}

		finalMacroCommands := []string{}

		for _, cmd := range allComands {

			if len(cmd) > 0 {
				if cmd[0] == '=' {
					user.SendTextLegacy(`You cannot reference macros inside of a macro`)
					return true, nil
				}
				finalMacroCommands = append(finalMacroCommands, cmd)
			}

		}

		if len(finalMacroCommands) < 1 {
			user.SendTextLegacy(`There was a problem setting your macro.`)
			return true, nil
		}

		user.Macros[setTarget] = strings.Join(finalMacroCommands, `;`)

		user.SendTextLegacy(fmt.Sprintf(`Macro set. Type <ansi fg="command">=%d</ansi> or (if your terminal supports it) press <ansi fg="command">F%d</ansi> to use it.`, macroNum, macroNum))
	}

	events.AddToQueue(events.UserSettingChanged{
		UserId: user.UserId,
		Name:   `macro`,
	})

	return true, nil
}
