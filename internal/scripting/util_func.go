package scripting

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/dop251/goja"
)

var (
	EventFlags = map[string]events.EventFlag{
		`CmdNone`:                    events.CmdNone,
		`CmdSkipScripts`:             events.CmdSkipScripts,
		`CmdSecretly`:                events.CmdSecretly,
		`CmdIsRequeue`:               events.CmdIsRequeue,
		`CmdBlockInput`:              events.CmdBlockInput,
		`CmdUnBlockInput`:            events.CmdUnBlockInput,
		`CmdBlockInputUntilComplete`: events.CmdBlockInputUntilComplete, // Combines both of the above two
	}
)

func setUtilFunctions(vm *goja.Runtime) {
	vm.Set(`UtilGetRoundNumber`, UtilGetRoundNumber)
	vm.Set(`UtilFindMatchIn`, UtilFindMatchIn)
	vm.Set(`UtilGetSecondsToRounds`, UtilGetSecondsToRounds)
	vm.Set(`UtilGetMinutesToRounds`, UtilGetMinutesToRounds)
	vm.Set(`UtilGetSecondsToTurns`, UtilGetSecondsToTurns)
	vm.Set(`UtilGetMinutesToTurns`, UtilGetMinutesToTurns)
	vm.Set(`UtilStripPrepositions`, UtilStripPrepositions)
	vm.Set(`UtilDiceRoll`, UtilDiceRoll)
	vm.Set(`UtilGetTime`, UtilGetTime)
	vm.Set(`UtilGetTimeString`, UtilGetTimeString)
	vm.Set(`UtilSetTime`, UtilSetTime)
	vm.Set(`UtilSetTimeDay`, UtilSetTimeDay)
	vm.Set(`UtilSetTimeNight`, UtilSetTimeNight)
	vm.Set(`UtilIsDay`, UtilIsDay)
	vm.Set(`UtilLocateUser`, UtilLocateUser)
	vm.Set(`UtilApplyColorPattern`, UtilApplyColorPattern)
	vm.Set(`UtilGetConfig`, UtilGetConfig)
	vm.Set(`UtilGetDamageDescription`, UtilGetDamageDescription)
	vm.Set(`UtilGetHealDescription`, UtilGetHealDescription)
	vm.Set(`ColorWrap`, ColorWrap)
	vm.Set(`EventFlags`, EventFlags)
	vm.Set(`RaiseEvent`, RaiseEvent)
	vm.Set(`ExpandCommand`, ExpandCommand)

}

// ////////////////////////////////////////////////////////
//
// # These functions get exported to the scripting engine
//
// ////////////////////////////////////////////////////////
func UtilGetRoundNumber() uint64 {
	return util.GetRoundCount()
}

func UtilFindMatchIn(search string, items []string) map[string]any {

	matches := map[string]any{
		`found`: false,
		`exact`: ``,
		`close`: ``,
	}

	if len(search) == 0 {
		return matches
	}
	match, closeMatch := util.FindMatchIn(search, items...)

	// Only allow close matches that the search string is a prefix of
	if len(closeMatch) > 0 {
		if !strings.HasPrefix(closeMatch, search) {
			closeMatch = ``
		}

		if len(search) < len(closeMatch) && len(search) < 3 {
			closeMatch = ``
		}
	}

	matches["found"] = len(match) > 0 || len(closeMatch) > 0
	matches["exact"] = match
	matches["close"] = closeMatch

	return matches
}

func UtilGetSecondsToRounds(seconds int) int {
	return configs.GetTimingConfig().SecondsToRounds(seconds)
}

func UtilGetMinutesToRounds(minutes int) int {
	return configs.GetTimingConfig().MinutesToRounds(minutes)
}

func UtilGetSecondsToTurns(seconds int) int {
	return configs.GetTimingConfig().SecondsToTurns(seconds)
}

func UtilGetMinutesToTurns(minutes int) int {
	return configs.GetTimingConfig().MinutesToTurns(minutes)
}

func UtilStripPrepositions(input string) string {
	return util.StripPrepositions(input)
}

func UtilDiceRoll(diceQty int, diceSides int) int {
	return util.RollDice(diceQty, diceSides)
}

func UtilGetTime() gametime.GameDate {
	return gametime.GetDate()
}

func UtilGetTimeString() string {
	gd := gametime.GetDate()
	return gd.String()
}

func UtilSetTimeDay() {
	gametime.SetToDay(-1)
}

func UtilSetTimeNight() {
	gametime.SetToNight(-1)
}

func UtilSetTime(hour int, minutes int) {
	gametime.SetTime(hour, minutes)
}

func UtilIsDay() bool {
	return !gametime.IsNight()
}

func UtilLocateUser(idOrName any) int {

	// check if is string
	if userName, ok := idOrName.(string); ok { // handle string
		if locateUser := users.GetByCharacterName(userName); locateUser != nil {
			return locateUser.Character.RoomId
		}
	} else if userId, ok := idOrName.(int); ok { // handle int
		if user := users.GetByUserId(userId); user != nil {
			return user.Character.RoomId
		}
	}

	return 0
}

func UtilApplyColorPattern(input string, patternName string, wordsOnly ...bool) string {

	if len(wordsOnly) > 0 && wordsOnly[0] {
		return colorpatterns.ApplyColorPattern(input, patternName, colorpatterns.Words)
	}
	return colorpatterns.ApplyColorPattern(input, patternName)
}

func UtilGetConfig() configs.Config {
	return configs.GetConfig()
}

func ColorWrap(txt string, colorClass ...string) string {

	if len(colorClass) > 0 && colorClass[0] != `` {
		txt = `<ansi fg="` + colorClass[0] + `">` + txt + `</ansi>`
	}

	if len(colorClass) > 1 && colorClass[1] != `` {
		txt = `<ansi bg="` + colorClass[1] + `">` + txt + `</ansi>`
	}

	return txt
}

func RaiseEvent(name string, data map[string]any) {
	events.AddToQueue(events.ScriptedEvent{Name: name, Data: data})
}

func UtilGetDamageDescription(amount int, targetMaxHP int) string {
	return combat.GetDamageDescription(amount, targetMaxHP)
}

func UtilGetHealDescription(amount int, targetMaxHP int) string {
	return combat.GetHealDescription(amount, targetMaxHP)
}

func ExpandCommand(cmd string, limit ...int) string {

	parts := util.SplitButRespectQuotes(cmd)

	expansionsLeft := -1
	if len(limit) > 0 {
		expansionsLeft = limit[0]
		if expansionsLeft < -1 {
			expansionsLeft = -1
		}
	}

	result := ""
	for _, abbr := range parts {
		if expansionsLeft != 0 {
			alias := keywords.TryCommandAlias(abbr)
			result += alias
			expansionsLeft--
		} else {
			result += abbr
		}
		result += " "
	}

	result = strings.Trim(result, " ")

	return result
}
