package users

import (
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/term"
	"github.com/GoMudEngine/GoMud/internal/util"
)

//
// This file contains vars/receiver methods for the UserRecord struct dealing with the prmopt.
// This just makes it easier to find and make adjustments to. It got annoying searching userrecord.go
// NOTE: NOT to be confused with an interactive question/answer prompt.
//
// Prompt Helpfile: templates/help/set-prompt.template
//

var (
	promptDefaultCompiled = ``
	promptColorRegex      = regexp.MustCompile(`\{(\d*)(?::)?(\d*)?\}`)
	promptFindTagsRegex   = regexp.MustCompile(`\{[a-zA-Z%:\-]+\}`)
)

// renderVitalBar returns a 10-block ANSI progress bar for a vital stat.
// Color breakpoints match the web client vitals window gradient:
//
//	>60% → green (ANSI 82), >30% → yellow (ANSI 226), ≤30% → red (ANSI 196)
func renderVitalBar(current, max int) string {
	if max <= 0 {
		max = 1
	}
	if current < 0 {
		current = 0
	}
	pct := float64(current) / float64(max) * 100.0
	if pct > 100 {
		pct = 100
	}

	filled := int(math.Round(pct / 10.0))
	empty := 10 - filled

	var barColor string
	switch {
	case pct > 60:
		barColor = "82"  // bright green
	case pct > 30:
		barColor = "226" // yellow
	default:
		barColor = "196" // red
	}

	return fmt.Sprintf(`<ansi fg="%s">%s</ansi><ansi fg="238">%s</ansi>`,
		barColor,
		strings.Repeat("█", filled),
		strings.Repeat("░", empty))
}

func (u *UserRecord) GetCommandPrompt() string {

	promptOut := ``

	if u.activePrompt != nil {

		if activeQuestion := u.activePrompt.GetNextQuestion(); activeQuestion != nil {
			promptOut = activeQuestion.String()
		}
	}

	goAhead := ``
	if connections.GetClientSettings(u.ConnectionId()).SendTelnetGoAhead {
		goAhead = term.TelnetGoAhead.String()
	}

	if len(promptOut) == 0 {

		if promptDefaultCompiled == `` {
			promptDefaultCompiled = util.ConvertColorShortTags(configs.GetTextFormatsConfig().Prompt.String())
		}

		var customPrompt any = nil
		var inCombat bool = u.Character.Aggro != nil

		if inCombat {
			customPrompt = u.GetConfigOption(`fprompt-compiled`)
		}

		// No other custom prompts? try the default setting
		if customPrompt == nil {
			customPrompt = u.GetConfigOption(`prompt-compiled`)
		}

		if customPrompt != nil {
			if ansiPrompt, ok := customPrompt.(string); ok {
				promptOut = u.ProcessPromptString(ansiPrompt)
			}
		}

		// Still nothing? Default to ... default
		if len(promptOut) == 0 {
			promptOut = u.ProcessPromptString(promptDefaultCompiled)
		}

	}

	unsent, suggested := u.GetUnsentText()
	if len(suggested) > 0 {
		suggested = `<ansi fg="suggested-text">` + suggested + `</ansi>`
	}
	return term.AnsiMoveCursorColumn.String() + term.AnsiEraseLine.String() + promptOut + unsent + suggested + goAhead

}

func (u *UserRecord) ProcessPromptString(promptStr string) string {

	promptOut := strings.Builder{}

	var currentXP, tnlXP int = -1, -1
	var hpPct, mpPct int = -1, -1
	var hpClass, mpClass string

	promptLen := len(promptStr)
	tagStartPos := -1

	for i := 0; i < promptLen; i++ {
		if promptStr[i] == '{' {
			tagStartPos = i
			continue
		}
		if promptStr[i] == '}' {

			switch promptStr[tagStartPos : i+1] {

			case `{\n}`:
				promptOut.WriteString("\n")

			case `{hp}`:
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, hpClass, u.Character.Health))

			case `{hp:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Health))
			case `{HP}`:
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, hpClass, u.Character.HealthMax.Value))
			case `{HP:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.HealthMax.Value))
			case `{hp%}`:
				if hpPct == -1 {
					hpPct = int(math.Floor(float64(u.Character.Health) / float64(u.Character.HealthMax.Value) * 100))
				}
				if len(hpClass) == 0 {
					hpClass = fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Health, u.Character.HealthMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, hpClass, hpPct))

			case `{hp%:-}`:
				if hpPct == -1 {
					hpPct = int(math.Floor(float64(u.Character.Health) / float64(u.Character.HealthMax.Value) * 100))
				}
				promptOut.WriteString(strconv.Itoa(hpPct))
				promptOut.WriteString(`%`)

			case `{mp}`:
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Conviction, u.Character.ConvictionMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, mpClass, u.Character.Conviction))

			case `{mp:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Conviction))

			case `{MP}`:
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Conviction, u.Character.ConvictionMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, mpClass, u.Character.ConvictionMax.Value))

			case `{MP:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.ConvictionMax.Value))

			case `{mp%}`:
				if mpPct == -1 {
					mpPct = int(math.Floor(float64(u.Character.Conviction) / float64(u.Character.ConvictionMax.Value) * 100))
				}
				if len(mpClass) == 0 {
					mpClass = fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Conviction, u.Character.ConvictionMax.Value))
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, mpClass, mpPct))

			case `{mp%:-}`:
				if mpPct == -1 {
					mpPct = int(math.Floor(float64(u.Character.Conviction) / float64(u.Character.ConvictionMax.Value) * 100))
				}
				promptOut.WriteString(strconv.Itoa(mpPct))
				promptOut.WriteString(`%`)

			case `{st}`:
				stClass := fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Stamina, u.Character.StaminaMax.Value))
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, stClass, u.Character.Stamina))

			case `{st:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Stamina))

			case `{ST}`:
				stClass := fmt.Sprintf(`health-%d`, util.QuantizeTens(u.Character.Stamina, u.Character.StaminaMax.Value))
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, stClass, u.Character.StaminaMax.Value))

			case `{ST:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.StaminaMax.Value))

			case `{cv}`:
				cvClass := fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Conviction, u.Character.ConvictionMax.Value))
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, cvClass, u.Character.Conviction))

			case `{cv:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Conviction))

			case `{CV}`:
				cvClass := fmt.Sprintf(`mana-%d`, util.QuantizeTens(u.Character.Conviction, u.Character.ConvictionMax.Value))
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d</ansi>`, cvClass, u.Character.ConvictionMax.Value))

			case `{CV:-}`:
				promptOut.WriteString(strconv.Itoa(u.Character.ConvictionMax.Value))

			case `{hpbar}`:
				promptOut.WriteString(renderVitalBar(u.Character.Health, u.Character.HealthMax.Value))

			case `{stbar}`:
				promptOut.WriteString(renderVitalBar(u.Character.Stamina, u.Character.StaminaMax.Value))

			case `{cvbar}`:
				promptOut.WriteString(renderVitalBar(u.Character.Conviction, u.Character.ConvictionMax.Value))

			case `{ap}`:
				promptOut.WriteString(strconv.Itoa(u.Character.ActionPoints))

			case `{xp}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				promptOut.WriteString(strconv.Itoa(currentXP))

			case `{XP}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				promptOut.WriteString(strconv.Itoa(tnlXP))

			case `{xp%}`:
				if currentXP == -1 && tnlXP == -1 {
					currentXP, tnlXP = u.Character.XPTNLActual()
				}
				tnlPercent := int(math.Floor(float64(currentXP) / float64(tnlXP) * 100))
				promptOut.WriteString(strconv.Itoa(tnlPercent))
				promptOut.WriteString(`%`)

			case `{h}`:
				hiddenFlag := ``
				if u.Character.HasBuffFlag(buffs.Hidden) {
					hiddenFlag = `H`
				}
				promptOut.WriteString(hiddenFlag)

			case `{pos}`:
				// Combat position (Standing/Prone/Clinched/Grounded) - Stage 8.1
				if u.Character.CombatPosition != characters.PositionStanding {
					posColor := u.Character.CombatPosition.GetPositionColor()
					promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, posColor, u.Character.CombatPosition.String()))
				}

			case `{a}`:
				alignClass := u.Character.AlignmentName()
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignClass, alignClass[:1]))

			case `{A}`:
				alignClass := u.Character.AlignmentName()
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, alignClass, alignClass))

			case `{g}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Gold))

			case `{tp}`:
				promptOut.WriteString(strconv.Itoa(u.Character.TrainingPoints))

			case `{sp}`:
				promptOut.WriteString(strconv.Itoa(u.Character.StatPoints))

			case `{i}`:
				promptOut.WriteString(strconv.Itoa(len(u.Character.Items)))

			case `{I}`:
				promptOut.WriteString(strconv.Itoa(int(u.Character.CarryCapacity())))

			case `{lvl}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Level))

			case `{w}`:
				if u.Character.Aggro != nil {
					promptOut.WriteString(strconv.Itoa(u.Character.Aggro.RoundsWaiting))
				} else {
					promptOut.WriteString(`0`)
				}

			case `{t}`:
				gd := gametime.GetDate()
				promptOut.WriteString(gd.String(true))

			case `{T}`:
				gd := gametime.GetDate()
				promptOut.WriteString(gd.String())

			}
			tagStartPos = -1
			continue
		}

		if tagStartPos == -1 {
			promptOut.WriteByte(promptStr[i])
		}
	}

	return promptOut.String()
}
