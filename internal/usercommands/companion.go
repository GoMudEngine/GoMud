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

	// Stat comparisons (relative to player — no raw numbers)
	pStr := user.Character.Stats.Strength.ValueAdj
	pDex := user.Character.Stats.Dexterity.ValueAdj
	pPer := user.Character.Stats.Perception.ValueAdj
	pVit := user.Character.Stats.Vitality.ValueAdj
	pWil := user.Character.Stats.Willpower.ValueAdj
	pCha := user.Character.Stats.Charisma.ValueAdj

	mStr := mob.Character.Stats.Strength.ValueAdj
	mDex := mob.Character.Stats.Dexterity.ValueAdj
	mPer := mob.Character.Stats.Perception.ValueAdj
	mVit := mob.Character.Stats.Vitality.ValueAdj
	mWil := mob.Character.Stats.Willpower.ValueAdj
	mCha := mob.Character.Stats.Charisma.ValueAdj

	strLine := statCompare("strength", mStr, pStr)
	dexLine := statCompare("agility", mDex, pDex)
	perLine := statCompare("perception", mPer, pPer)
	vitLine := statCompare("hardiness", mVit, pVit)
	wilLine := statCompare("willpower", mWil, pWil)
	chaLine := statCompare("force of personality", mCha, pCha)

	user.SendText(
		`  Compared to you, it seems:`,
	)
	for _, line := range []string{strLine, dexLine, perLine, vitLine, wilLine, chaLine} {
		user.SendText(`    ` + line)
	}

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

// statCompare returns a one-line description of how the companion's stat
// compares to the player's. Uses relative language only — no raw values.
func statCompare(label string, compStat, playerStat int) string {
	if playerStat <= 0 {
		playerStat = 1
	}
	ratio := float64(compStat) / float64(playerStat)
	var rel string
	switch {
	case ratio >= 1.40:
		rel = "far greater"
	case ratio >= 1.15:
		rel = "noticeably greater"
	case ratio >= 0.90:
		rel = "roughly matched"
	case ratio >= 0.70:
		rel = "somewhat lesser"
	default:
		rel = "far lesser"
	}
	return fmt.Sprintf(
		`<ansi fg="yellow">%s</ansi> — %s`,
		label, rel,
	)
}

