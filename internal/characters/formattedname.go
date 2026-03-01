package characters

import (
	"fmt"
	"sort"

	"github.com/GoMudEngine/GoMud/internal/colorpatterns"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/ansitags"
)

type adjectiveStyle struct {
	LongForm     string
	ShortForm    string
	ColorPattern string
}

var (

	// -short suffix should also be defined in case shorthand symbols are preferred
	adjectiveStyles = map[string]adjectiveStyle{
		`charmed`:  {`♥friend`, `♥`, `pink`},     // Are they charmed/friendly?
		`downed`:   {`☠downed`, `☠`, `red`},      // Are they downed?
		`hidden`:   {`hidden`, `?`, `gray`},      // Are they hiding?
		`lit`:      {`☀️Lit`, `☀️`, `lit`},       // Does light come from this character?
		`sleeping`: {`asleep`, `zZz`, `gray`},    // Are they hiding?
		`zombie`:   {`zOmBie`, `z`, `zombie`},    // Have they disconnected and are zombie status?
		`poisoned`: {`☠poisoned`, `☠`, `purple`}, // Have they disconnected and are zombie status?
		`shop`:     {`shop`, `$`, `gold`},        // Do they sell stuff?
	}

	adjectiveSwaps = map[string]string{}
)

type FormattedName struct {
	Name               string
	Type               string // mob/user
	Suffix             string // What ansi alias suffix to use (if any)
	Adjectives         []string
	UseShortAdjectives bool   // Whether to failover to short adjectives
	QuestAlert         bool   // Whether this mob is relevant to a current quest
	PetName            string // Name of pet (if any)
	DuplicateIndex     int    // When > 0, appends #N to name and shifts color for duplicates
}

func (f FormattedName) String() string {

	name := f.Name
	ansiAlias := f.Type

	if f.DuplicateIndex > 0 {
		name = fmt.Sprintf(`%s #%d`, f.Name, f.DuplicateIndex)
		// Indices 2-4 get shifted colors; index 1 keeps base color; 5+ wraps
		idx := ((f.DuplicateIndex - 1) % 4) + 1
		if idx >= 2 {
			ansiAlias = fmt.Sprintf(`%s-dup%d`, f.Type, idx)
		}
	}

	if f.Suffix != `` {
		ansiAlias = fmt.Sprintf(`%s-%s`, ansiAlias, f.Suffix)
	}

	output := fmt.Sprintf(`<ansi fg="%s">%s</ansi>`, ansiAlias, name)

	adjectives := f.Adjectives

	shortSuffix := ``
	if f.UseShortAdjectives || len(adjectives) > 3 {
		shortSuffix = `-short`
	}

	if adjLen := len(adjectives); adjLen > 0 {
		output += ` <ansi fg="black-bold">(`
		for i, adj := range adjectives {
			if newAdj, ok := adjectiveSwaps[adj+shortSuffix]; ok {
				output += newAdj
			} else {
				output += adj
			}
			if i < adjLen-1 {
				output += `|`
			}
		}
		output += `)</ansi>`
	}

	if f.QuestAlert {
		output = `<ansi fg="questflag">★</ansi>` + output
	}

	if f.PetName != `` {
		output += ` and ` + f.PetName
	}

	return output
}

func GetFormattedAdjective(adjName string) string {
	if newAdj, ok := adjectiveSwaps[adjName]; ok {
		return newAdj
	}
	return adjName
}

func GetFormattedAdjectives(excludeShort bool) []string {

	ret := []string{}

	for name := range adjectiveSwaps {
		if excludeShort {
			if len(name) > 6 && name[len(name)-6:] == `-short` {
				continue
			}
		}
		ret = append(ret, name)
	}

	sort.Slice(ret, func(i, j int) bool { return ret[i] < ret[j] })

	return ret
}

func CompileAdjectiveSwaps() {
	clear(adjectiveSwaps)
	for adjName, styleDefinition := range adjectiveStyles {
		adjectiveSwaps[adjName] = colorpatterns.ApplyColorPattern(styleDefinition.LongForm, styleDefinition.ColorPattern)
		adjectiveSwaps[adjName+`-short`] = colorpatterns.ApplyColorPattern(styleDefinition.ShortForm, styleDefinition.ColorPattern)
	}

	for _, name := range GetFormattedAdjectives(true) {
		mudlog.Info("Color Test (Adjectives)", "name", name, "short", ansitags.Parse(GetFormattedAdjective(name+`-short`)), "full", ansitags.Parse(GetFormattedAdjective(name)))
	}
}
