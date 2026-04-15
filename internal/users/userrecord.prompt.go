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
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/spells"
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
	promptDefaultCompiled      = ``
	fightPromptDefaultCompiled = ``
	promptColorRegex           = regexp.MustCompile(`\{(\d*)(?::)?(\d*)?\}`)
	promptFindTagsRegex        = regexp.MustCompile(`\{[a-zA-Z%:\-]+\}`)
)

// RenderVitalBar is the exported version of renderVitalBar.
func RenderVitalBar(current, max, reserved int) string {
	return renderVitalBar(current, max, reserved)
}

// renderVitalBar returns a 10-block ANSI progress bar for a vital stat.
// Color breakpoints match the web client vitals window gradient:
//
//	>60% → green (ANSI 82), >30% → yellow (ANSI 226), ≤30% → red (ANSI 196)
//
// The reserved parameter adds dark magenta blocks representing pool
// reservation from Chrysalis enchantments. When reserved=0 the bar
// renders identically to the original two-arg version.
func renderVitalBar(current, max, reserved int) string {
	if max <= 0 {
		max = 1
	}
	if current < 0 {
		current = 0
	}
	if reserved < 0 {
		reserved = 0
	}

	effectiveMax := max - reserved
	if effectiveMax < 1 {
		effectiveMax = 1
	}
	if current > effectiveMax {
		current = effectiveMax
	}

	// Calculate block counts out of 10
	reservedBlocks := int(math.Round(float64(reserved) / float64(max) * 10.0))
	if reservedBlocks > 10 {
		reservedBlocks = 10
	}

	usableBlocks := int(math.Round(float64(current) / float64(max) * 10.0))
	if usableBlocks+reservedBlocks > 10 {
		usableBlocks = 10 - reservedBlocks
	}

	emptyBlocks := 10 - usableBlocks - reservedBlocks
	if emptyBlocks < 0 {
		emptyBlocks = 0
	}

	// Color based on current vs effective max
	effectivePct := float64(current) / float64(effectiveMax) * 100.0
	var barColor string
	switch {
	case effectivePct > 60:
		barColor = "82"  // bright green
	case effectivePct > 30:
		barColor = "226" // yellow
	default:
		barColor = "196" // red
	}

	result := fmt.Sprintf(`<ansi fg="%s">%s</ansi>`,
		barColor,
		strings.Repeat("█", usableBlocks))

	if reservedBlocks > 0 {
		result += fmt.Sprintf(`<ansi fg="53">%s</ansi>`,
			strings.Repeat("▓", reservedBlocks))
	}

	if emptyBlocks > 0 {
		result += fmt.Sprintf(`<ansi fg="238">%s</ansi>`,
			strings.Repeat("░", emptyBlocks))
	}

	return result
}

// getPromptToggle returns the toggle state for a fight prompt element.
// Returns true (on) by default when not explicitly set.
func (u *UserRecord) getPromptToggle(key string) bool {
	val := u.GetConfigOption(`fprompt-tog-` + key)
	if val == nil {
		return true
	}
	if b, ok := val.(bool); ok {
		return b
	}
	return true
}

// targetHealthDesc returns a descriptive label and color class for a target's health percentage.
func targetHealthDesc(health, maxHealth int) (string, string) {
	if maxHealth <= 0 {
		maxHealth = 1
	}
	if health <= 0 {
		return "dead", "health-0"
	}
	pct := int(float64(health) / float64(maxHealth) * 100.0)
	switch {
	case pct >= 80:
		return "healthy", "health-100"
	case pct >= 60:
		return "bruised", "health-60"
	case pct >= 40:
		return "wounded", "health-40"
	case pct >= 20:
		return "badly wounded", "health-20"
	default:
		return "near death", "health-10"
	}
}

// BuildFightPromptTemplate assembles the fight prompt template string from enabled toggles.
// It is exported so that usercommands/set.go can rebuild and cache it when a toggle changes.
func (u *UserRecord) BuildFightPromptTemplate() string {
	var b strings.Builder
	b.WriteString(`{8}[{t}`)
	if u.getPromptToggle(`bars`) {
		b.WriteString(` {255}HP:{hpbar} SP:{stbar} CP:{cvbar}`)
	}
	if u.getPromptToggle(`pos`) {
		b.WriteString(`{pos}`)
	}
	b.WriteString(`{8}]`)
	if u.getPromptToggle(`target`) {
		b.WriteString(` {255}» {target}`)
	}
	showTargetHealth := u.getPromptToggle(`targethealth`)
	showTargetPos := u.getPromptToggle(`targetpos`)
	if showTargetPos || showTargetHealth {
		b.WriteString(`{8}[`)
		if showTargetPos {
			b.WriteString(`{targetpos}`)
		}
		if showTargetPos && showTargetHealth {
			b.WriteString(`{8}|`)
		}
		if showTargetHealth {
			b.WriteString(`{targethealth}`)
		}
		b.WriteString(`{8}]`)
	}
	if u.getPromptToggle(`tank`) {
		b.WriteString(`{tank}`)
	}
	b.WriteString(`{casting}`)
	b.WriteString(`{239}{h}{8}:`)
	return b.String()
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
			if customPrompt == nil {
				// Toggle-driven cache (kept current by set.go when a toggle changes)
				cached := u.GetConfigOption(`fprompt-default-compiled`)
				if cached == nil {
					// No toggle customizations — use the server config default
					if fightPromptDefaultCompiled == `` {
						fightPromptDefaultCompiled = util.ConvertColorShortTags(configs.GetTextFormatsConfig().FightPrompt.String())
					}
					customPrompt = fightPromptDefaultCompiled
				} else {
					customPrompt = cached
				}
			}
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
				promptOut.WriteString(renderVitalBar(u.Character.Health, u.Character.HealthMax.Value,
					u.Character.GetPoolReservation("health", u.Character.HealthMax.Value)))

			case `{stbar}`:
				promptOut.WriteString(renderVitalBar(u.Character.Stamina, u.Character.StaminaMax.Value,
					u.Character.GetPoolReservation("stamina", u.Character.StaminaMax.Value)))

			case `{cvbar}`:
				promptOut.WriteString(renderVitalBar(u.Character.Conviction, u.Character.ConvictionMax.Value,
					u.Character.GetPoolReservation("conviction", u.Character.ConvictionMax.Value)))

			case `{pet_hp}`:
				if len(u.Character.Companions) > 0 && u.Character.Companions[0].InstanceId > 0 {
					if pet := mobs.GetInstance(u.Character.Companions[0].InstanceId); pet != nil && pet.Character.HealthMax.Value > 0 {
						petClass := fmt.Sprintf(`health-%d`, util.QuantizeTens(pet.Character.Health, pet.Character.HealthMax.Value))
						pct := int(math.Floor(float64(pet.Character.Health) / float64(pet.Character.HealthMax.Value) * 100))
						promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, petClass, pct))
					}
				}

			case `{pet_sp}`:
				if len(u.Character.Companions) > 0 && u.Character.Companions[0].InstanceId > 0 {
					if pet := mobs.GetInstance(u.Character.Companions[0].InstanceId); pet != nil && pet.Character.StaminaMax.Value > 0 {
						petClass := fmt.Sprintf(`health-%d`, util.QuantizeTens(pet.Character.Stamina, pet.Character.StaminaMax.Value))
						pct := int(math.Floor(float64(pet.Character.Stamina) / float64(pet.Character.StaminaMax.Value) * 100))
						promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, petClass, pct))
					}
				}

			case `{pet_cp}`:
				if len(u.Character.Companions) > 0 && u.Character.Companions[0].InstanceId > 0 {
					if pet := mobs.GetInstance(u.Character.Companions[0].InstanceId); pet != nil && pet.Character.ConvictionMax.Value > 0 {
						petClass := fmt.Sprintf(`mana-%d`, util.QuantizeTens(pet.Character.Conviction, pet.Character.ConvictionMax.Value))
						pct := int(math.Floor(float64(pet.Character.Conviction) / float64(pet.Character.ConvictionMax.Value) * 100))
						promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%d%%</ansi>`, petClass, pct))
					}
				}

			case `{target}`:
				if u.Character.Aggro != nil {
					if u.Character.Aggro.MobInstanceId > 0 {
						if m := mobs.GetInstance(u.Character.Aggro.MobInstanceId); m != nil {
							promptOut.WriteString(fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, m.Character.Name))
						}
					} else if u.Character.Aggro.UserId > 0 {
						if target := GetByUserId(u.Character.Aggro.UserId); target != nil {
							promptOut.WriteString(fmt.Sprintf(`<ansi fg="username">%s</ansi>`, target.Character.Name))
						}
					}
				}

			case `{targethealth}`:
				if u.Character.Aggro != nil {
					var tHealth, tMax int
					if u.Character.Aggro.MobInstanceId > 0 {
						if m := mobs.GetInstance(u.Character.Aggro.MobInstanceId); m != nil {
							tHealth, tMax = m.Character.Health, m.Character.HealthMax.Value
						}
					} else if u.Character.Aggro.UserId > 0 {
						if target := GetByUserId(u.Character.Aggro.UserId); target != nil {
							tHealth, tMax = target.Character.Health, target.Character.HealthMax.Value
						}
					}
					if tMax > 0 {
						desc, color := targetHealthDesc(tHealth, tMax)
						promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, color, desc))
					}
				}

			case `{targetpos}`:
				if u.Character.Aggro != nil {
					var tPos characters.CombatPosition
					if u.Character.Aggro.MobInstanceId > 0 {
						if m := mobs.GetInstance(u.Character.Aggro.MobInstanceId); m != nil {
							tPos = m.Character.CombatPosition
						}
					} else if u.Character.Aggro.UserId > 0 {
						if target := GetByUserId(u.Character.Aggro.UserId); target != nil {
							tPos = target.Character.CombatPosition
						}
					}
					if tPos != `` {
						promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`,
							tPos.GetPositionColor(), tPos.String()))
					}
				}

			case `{tank}`:
				if u.Character.Aggro != nil && u.Character.Aggro.MobInstanceId > 0 {
					if m := mobs.GetInstance(u.Character.Aggro.MobInstanceId); m != nil {
						if m.Character.Aggro != nil && m.Character.Aggro.UserId > 0 &&
							m.Character.Aggro.UserId != u.UserId {
							if tankUser := GetByUserId(m.Character.Aggro.UserId); tankUser != nil {
								tankBar := renderVitalBar(
									tankUser.Character.Health,
									tankUser.Character.HealthMax.Value,
									tankUser.Character.GetPoolReservation("health",
										tankUser.Character.HealthMax.Value))
								promptOut.WriteString(fmt.Sprintf(
									` <ansi fg="255">T:</ansi><ansi fg="username">%s</ansi> <ansi fg="255">Thp:</ansi>%s`,
									tankUser.Character.Name, tankBar))
							}
						}
					}
				}

			case `{ap}`:
				promptOut.WriteString(strconv.Itoa(u.Character.ActionPoints))

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

			case `{casting}`:
				if u.Character.CastingState != nil {
					cs := u.Character.CastingState
					spellName := cs.SpellId
					if sd := spells.GetSpell(cs.SpellId); sd != nil {
						spellName = sd.Name
					}
					promptOut.WriteString(fmt.Sprintf(
						`<ansi fg="cyan"> [%s %d/%d]</ansi>`,
						spellName, cs.FoldsAccumulated, cs.FoldsNeeded))
				}

			case `{g}`:
				promptOut.WriteString(strconv.Itoa(u.Character.Gold))

			case `{enc}`:
				weight := u.Character.GetCarriedWeight()
				capacity := u.Character.CarryCapacity()
				var encLabel, encColor string
				if capacity <= 0 {
					encLabel, encColor = "crushed", "magenta-bold"
				} else {
					ratio := weight / capacity
					switch {
					case ratio <= 0.25:
						encLabel, encColor = "light", "green"
					case ratio <= 0.50:
						encLabel, encColor = "moderate", "yellow"
					case ratio <= 0.75:
						encLabel, encColor = "heavy", "red"
					case ratio <= 1.00:
						encLabel, encColor = "overburdened", "red-bold"
					default:
						encLabel, encColor = "crushed", "magenta-bold"
					}
				}
				promptOut.WriteString(fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, encColor, encLabel))

			case `{i}`:
				promptOut.WriteString(strconv.Itoa(len(u.Character.Items)))

			case `{I}`:
				promptOut.WriteString(strconv.Itoa(int(u.Character.CarryCapacity())))

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
