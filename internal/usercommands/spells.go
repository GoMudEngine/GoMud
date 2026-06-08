package usercommands

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// spellEntry holds a display row together with its sort keys.
type spellEntry struct {
	category   int // 0=utility, 1=heal, 2=buff, 3=damage, 4=summon
	targetRank int // 0=self, 1=single, 2=group, 3=area
	difficulty int
	row        []string
	formatRow  []string
}

// spellCategory returns a sort-order category for a spell based on its
// effect type and school. Groups related spells together in the list.
func spellCategory(sp *spells.SpellData) int {
	switch sp.EffectType {
	case "identify":
		return 0 // utility
	case "heal":
		return 1
	case "buff", "shield", "purge":
		return 2
	case "damage", "dot", "knockdown":
		return 3
	}
	// Manifestation school spells (raise/conjure) are neutral type with no effect_type
	if sp.HasSchool(spells.SchoolManifestation) {
		return 4
	}
	// Remaining neutral spells (glow, fold-anchor, etc.) are utility
	if sp.Type == spells.Neutral {
		return 0
	}
	// Debuffs applied via harmful buff spells
	if sp.Type == spells.HarmSingle || sp.Type == spells.HarmMulti || sp.Type == spells.HarmArea {
		return 3
	}
	return 2
}

// targetRank returns a sort-order rank for target scope.
func targetRank(sp *spells.SpellData) int {
	switch sp.Type {
	case spells.Neutral:
		return 0
	case spells.HelpSingle, spells.HarmSingle:
		return 1
	case spells.HelpMulti, spells.HarmMulti:
		return 2
	case spells.HelpArea, spells.HarmArea:
		return 3
	}
	return 0
}

func Spells(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	headers := []string{`SpellId`, `Name`, `Target`, `Cost`}

	entries := []spellEntry{}

	for spellId, casts := range user.Character.GetSpells() {

		if casts < 0 {
			continue
		}

		if sp := spells.GetSpell(spellId); sp != nil {

			helpOrHarm := sp.Type.HelpOrHarmString()
			targetColor := `spell-` + helpOrHarm
			target := sp.Type.TargetTypeString(true)

			formatRow := []string{
				`<ansi fg="yellow-bold">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="` + targetColor + `">%s</ansi>`,
				`<ansi fg="magenta">%s</ansi>`,
			}

			costStr := combat.GetConvictionCostDescription(sp.Cost)
			if sp.HealthCost > 0 {
				costStr += ", drains health"
			}

			row := []string{sp.SpellId,
				sp.Name,
				target,
				costStr,
			}

			entries = append(entries, spellEntry{
				category:   spellCategory(sp),
				targetRank: targetRank(sp),
				difficulty: sp.Difficulty,
				row:        row,
				formatRow:  formatRow,
			})
		}
	}

	// Sort by category, then target scope, then difficulty.
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].category != entries[j].category {
			return entries[i].category < entries[j].category
		}
		if entries[i].targetRank != entries[j].targetRank {
			return entries[i].targetRank < entries[j].targetRank
		}
		return entries[i].difficulty < entries[j].difficulty
	})

	rowFormatting := make([][]string, 0, len(entries))
	rows := make([][]string, 0, len(entries))
	for _, e := range entries {
		rowFormatting = append(rowFormatting, e.formatRow)
		rows = append(rows, e.row)
	}

	onlineResultsTable := templates.GetTable(`Spells`, headers, rows, rowFormatting...)
	tplTxt, _ := templates.Process("tables/generic", onlineResultsTable, user.UserId)
	user.SendText(messaging.CategorySystem, tplTxt)

	return true, nil
}
