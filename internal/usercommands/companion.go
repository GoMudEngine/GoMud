package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Companion handles the `companion` command in four forms:
//
//	companion                          — list all companions
//	companion <name>                   — detailed view (no hard numbers)
//	companion <name> assist on/off     — toggle auto-assist
//	companion <name> name <nickname>   — rename a companion
func Companion(rest string, user *users.UserRecord,
	room *rooms.Room, flags events.EventFlag) (bool, error) {

	companions := user.Character.Companions

	// ── Form 1: no args → list ──────────────────────────────────────────────
	if rest == "" {
		if len(companions) == 0 {
			user.SendText("You have no companions.")
			return true, nil
		}

		user.SendText(`<ansi fg="yellow">━━━ Companions ━━━</ansi>`)
		for _, comp := range companions {
			mob := mobs.GetInstance(comp.InstanceId)

			assistStr := "off"
			if comp.AutoAssist {
				assistStr = "on"
			}

			if mob == nil {
				user.SendText(fmt.Sprintf(
					`  <ansi fg="mobname">%s</ansi> [%s] `+
						`<ansi fg="239">[offline]</ansi> `+
						`(assist: %s)`,
					comp.Name, comp.SourceType, assistStr,
				))
				continue
			}

			hpBar := users.RenderVitalBar(
				mob.Character.Health,
				mob.Character.HealthMax.Value,
				0,
			)
			spBar := users.RenderVitalBar(
				mob.Character.Stamina,
				mob.Character.StaminaMax.Value,
				0,
			)
			cpBar := users.RenderVitalBar(
				mob.Character.Conviction,
				mob.Character.ConvictionMax.Value,
				0,
			)

			user.SendText(fmt.Sprintf(
				`  <ansi fg="mobname">%s</ansi> [%s] (assist: %s)`,
				comp.Name, comp.SourceType, assistStr,
			))
			user.SendText(fmt.Sprintf(
				`    HP: %s  SP: %s  CP: %s`,
				hpBar, spBar, cpBar,
			))
			if mutList := formatCompanionMutations(mob.Character.Mutations); mutList != "" {
				user.SendText(fmt.Sprintf(`    Mutations: %s`, mutList))
			}
		}
		return true, nil
	}

	// ── Parse remaining forms ────────────────────────────────────────────────
	lowerRest := strings.ToLower(rest)
	var assistToggle string // "on", "off", or "" (no toggle)
	var renameNick string   // non-empty if "name <nick>" form
	compName := rest

	if strings.HasSuffix(lowerRest, " assist on") {
		assistToggle = "on"
		compName = rest[:len(rest)-len(" assist on")]
	} else if strings.HasSuffix(lowerRest, " assist off") {
		assistToggle = "off"
		compName = rest[:len(rest)-len(" assist off")]
	} else if idx := strings.Index(lowerRest, " name "); idx >= 0 {
		// "companion <name> name <nickname>"
		compName = rest[:idx]
		renameNick = strings.TrimSpace(rest[idx+len(" name "):])
	}
	compName = strings.TrimSpace(compName)

	comp := user.Character.GetCompanion(compName)
	if comp == nil {
		user.SendText(fmt.Sprintf(
			`You have no companion named "%s".`, compName,
		))
		return true, nil
	}

	// ── Form 4: rename companion ─────────────────────────────────────────────
	if renameNick != "" {
		if err := validateCompanionName(renameNick); err != nil {
			user.SendText(err.Error())
			return true, nil
		}

		// Backfill BaseName for pre-existing companions that lack it.
		if comp.BaseName == "" {
			comp.BaseName = comp.Name
		}

		oldName := comp.Name
		newDisplayName := renameNick + " the " + comp.BaseName
		comp.Nickname = renameNick
		comp.Name = newDisplayName

		// Update the live mob's name so it shows in room/combat text.
		if mob := mobs.GetInstance(comp.InstanceId); mob != nil {
			mob.Character.Name = newDisplayName
		}

		user.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi> is now known as `+
				`<ansi fg="mobname">%s</ansi>.`,
			oldName, newDisplayName,
		))
		return true, nil
	}

	// ── Form 3: toggle assist ────────────────────────────────────────────────
	if assistToggle != "" {
		comp.AutoAssist = assistToggle == "on"
		stateStr := "off"
		if comp.AutoAssist {
			stateStr = "on"
		}
		user.SendText(fmt.Sprintf(
			`<ansi fg="mobname">%s</ansi>'s auto-assist is now `+
				`<ansi fg="yellow">%s</ansi>.`,
			comp.Name, stateStr,
		))
		return true, nil
	}

	// ── Form 2: detailed view ────────────────────────────────────────────────
	mob := mobs.GetInstance(comp.InstanceId)

	assistStr := "off"
	if comp.AutoAssist {
		assistStr = "on"
	}

	user.SendText(fmt.Sprintf(
		`<ansi fg="yellow">━━━ %s [%s] ━━━</ansi>`,
		comp.Name, comp.SourceType,
	))

	if mob == nil {
		user.SendText(`  This companion is not currently present.`)
		user.SendText(fmt.Sprintf(`  Auto-Assist: %s`, assistStr))
		return true, nil
	}

	// Vitals descriptions (no hard numbers)
	user.SendText(fmt.Sprintf(
		`  Health: %-10s Stamina: %-10s Conviction: %s`,
		vitalDesc(mob.Character.Health, mob.Character.HealthMax.Value),
		vitalDesc(mob.Character.Stamina, mob.Character.StaminaMax.Value),
		vitalDesc(mob.Character.Conviction, mob.Character.ConvictionMax.Value),
	))

	// Stat descriptions using the same tiers as the player status command
	user.SendText(fmt.Sprintf(
		`  <ansi fg="yellow">Strength:</ansi>   %-13s <ansi fg="yellow">Dexterity:</ansi>  %-13s <ansi fg="yellow">Perception:</ansi> %s`,
		statQualityDesc(mob.Character.Stats.Strength.ValueAdj),
		statQualityDesc(mob.Character.Stats.Dexterity.ValueAdj),
		statQualityDesc(mob.Character.Stats.Perception.ValueAdj),
	))
	user.SendText(fmt.Sprintf(
		`  <ansi fg="yellow">Vitality:</ansi>   %-13s <ansi fg="yellow">Willpower:</ansi>  %-13s <ansi fg="yellow">Charisma:</ansi>   %s`,
		statQualityDesc(mob.Character.Stats.Vitality.ValueAdj),
		statQualityDesc(mob.Character.Stats.Willpower.ValueAdj),
		statQualityDesc(mob.Character.Stats.Charisma.ValueAdj),
	))

	if mutList := formatCompanionMutations(mob.Character.Mutations); mutList != "" {
		user.SendText(fmt.Sprintf(`  <ansi fg="yellow">Mutations:</ansi>  %s`, mutList))
	}

	user.SendText(fmt.Sprintf(`  Auto-Assist: %s`, assistStr))

	return true, nil
}

// formatCompanionMutations renders a companion's mutation list as a
// comma-separated string of display names, with "(Ln)" suffix for levels
// greater than 1. Returns "" if the mob has no mutations.
func formatCompanionMutations(muts map[string]int) string {
	if len(muts) == 0 {
		return ""
	}

	ids := make([]string, 0, len(muts))
	for id := range muts {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	parts := make([]string, 0, len(ids))
	for _, id := range ids {
		spec := mutations.GetMutation(id)
		name := id
		if spec != nil && spec.Name != "" {
			name = spec.Name
		}
		if lvl := muts[id]; lvl > 1 {
			parts = append(parts, fmt.Sprintf("%s (L%d)", name, lvl))
		} else {
			parts = append(parts, name)
		}
	}
	return strings.Join(parts, ", ")
}

// vitalDesc returns a qualitative description of a vital resource pool.
func vitalDesc(current, max int) string {
	if max <= 0 {
		return "unknown"
	}
	pct := float64(current) / float64(max)
	switch {
	case pct >= 0.95:
		return "pristine"
	case pct >= 0.75:
		return "strong"
	case pct >= 0.50:
		return "steady"
	case pct >= 0.25:
		return "waning"
	case pct >= 0.10:
		return "critical"
	default:
		return "failing"
	}
}

// statQualityDesc returns the same descriptive tier used by the player
// status command. Matches the statQuality template function exactly.
func statQualityDesc(value int) string {
	switch {
	case value <= 60:
		return "feeble"
	case value <= 75:
		return "poor"
	case value <= 90:
		return "modest"
	case value <= 110:
		return "average"
	case value <= 130:
		return "keen"
	case value <= 150:
		return "exceptional"
	case value <= 200:
		return "remarkable"
	case value <= 300:
		return "transcendent"
	default:
		return "godlike"
	}
}

// validateCompanionName checks a proposed companion nickname against the
// banned-names list, existing player character names, and existing companion
// nicknames. Names are freed when companions die or are dismissed.
func validateCompanionName(name string) error {
	if len(name) < 2 || len(name) > 20 {
		return fmt.Errorf("Companion names must be between 2 and 20 characters.")
	}

	// Only allow letters (no numbers, symbols, spaces).
	for _, ch := range name {
		if (ch < 'a' || ch > 'z') && (ch < 'A' || ch > 'Z') {
			return fmt.Errorf("Companion names may only contain letters.")
		}
	}

	if bannedPattern, ok := configs.GetConfig().IsBannedName(name); ok {
		return fmt.Errorf("That name matched the prohibited pattern: %q.", bannedPattern)
	}

	// Check against existing player character names.
	if foundUserId, _ := users.CharacterNameSearch(name); foundUserId > 0 {
		return fmt.Errorf("That name is already in use by a player.")
	}

	// Check against other companions' nicknames.
	if users.CompanionNameExists(name) {
		return fmt.Errorf("That name is already in use by another companion.")
	}

	return nil
}

