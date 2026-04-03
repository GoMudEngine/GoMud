package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Companion handles the `companion` command in three forms:
//
//	companion             — list all companions
//	companion <name>      — detailed view (no hard numbers)
//	companion <name> assist on/off — toggle auto-assist
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
		}
		return true, nil
	}

	// ── Parse remaining forms ────────────────────────────────────────────────
	// Check for trailing "assist on" / "assist off"
	lowerRest := strings.ToLower(rest)
	var assistToggle string // "on", "off", or "" (no toggle)
	compName := rest

	if strings.HasSuffix(lowerRest, " assist on") {
		assistToggle = "on"
		compName = rest[:len(rest)-len(" assist on")]
	} else if strings.HasSuffix(lowerRest, " assist off") {
		assistToggle = "off"
		compName = rest[:len(rest)-len(" assist off")]
	}
	compName = strings.TrimSpace(compName)

	comp := user.Character.GetCompanion(compName)
	if comp == nil {
		user.SendText(fmt.Sprintf(
			`You have no companion named "%s".`, compName,
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

	user.SendText(fmt.Sprintf(`  Auto-Assist: %s`, assistStr))

	return true, nil
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

