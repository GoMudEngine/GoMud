package usercommands

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/gametime"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/textutil"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Cast initiates fold-based spellcasting (Stage 11.2).
// Each round the combat hook adds folds until FoldsNeeded is reached,
// then Stage 11.4 resolves the actual spell effect.
func Cast(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	// 0. Can't cast while bleeding out
	if user.Character.IsDisabled() {
		user.SendText(`<ansi fg="red">You can't focus enough to cast while bleeding out.</ansi>`)
		return true, nil
	}

	// 1. Spellcasting or manifestation skill required (checked again after spell
	//    lookup for a better error message, but we need at least one non-zero).
	skillLevel := user.Character.GetSkillLevel(skills.Spellcasting)
	manifestLevel := user.Character.GetSkillLevel(skills.Manifestation)
	if skillLevel == 0 && manifestLevel == 0 {
		user.SendText(`<ansi fg="red">You have no spellcasting skill.</ansi>`)
		return true, nil
	}

	// 2. Parse spell name and optional target from rest
	rest = strings.TrimSpace(rest)
	if rest == `` {
		user.SendText(`<ansi fg="red">Cast what? (Usage: cast <spell> [target])</ansi>`)
		return true, nil
	}

	parts := strings.SplitN(rest, ` `, 2)
	spellName := strings.ToLower(parts[0])
	targetName := ``
	if len(parts) > 1 {
		targetName = strings.TrimSpace(parts[1])
	}

	// 3. Verify spell exists — lookup first so we can give a useful error
	//    message before spending a cooldown. InitiateCast will repeat the
	//    lookup; the second lookup is cheap (map key hit).
	spellInfo := spells.GetSpell(spellName)
	if spellInfo == nil {
		spellInfo = spells.FindSpellByName(spellName)
	}
	if spellInfo == nil {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">No spell found for "%s". Use the spell ID (e.g. `+
				`<ansi fg="cyan-bold">mm</ansi>, <ansi fg="cyan-bold">heal</ansi>). `+
				`Type <ansi fg="cyan-bold">spells</ansi> to list what you know.</ansi>`,
			spellName))
		return true, nil
	}
	if !user.Character.HasSpell(spellInfo.SpellId) {
		user.SendText(fmt.Sprintf(`<ansi fg="red">You haven't learned the spell "%s".</ansi>`, spellInfo.Name))
		return true, nil
	}

	// 3b. Verify the player has the skill appropriate for this spell's school.
	if spellInfo.HasSchool(spells.SchoolManifestation) {
		if manifestLevel == 0 {
			user.SendText(`<ansi fg="red">You have no manifestation skill.</ansi>`)
			return true, nil
		}
	} else {
		if skillLevel == 0 {
			user.SendText(`<ansi fg="red">You have no spellcasting skill.</ansi>`)
			return true, nil
		}
	}

	// 4. Already casting?
	if user.Character.CastingState != nil {
		cs := user.Character.CastingState
		user.SendText(`<ansi fg="cyan">` + spells.GetCastMessage("already_casting", cs.SpellId) + `</ansi>`)
		return true, nil
	}

	// 4.5. Can't cast while crafting
	if user.Character.CraftingState != nil {
		user.SendText(`You are busy crafting.`)
		return true, nil
	}

	// 5. Conviction check — must have enough for the full cast.
	// Stage 12.1: Apply conviction_cost_multiplier from Magical Resistance mutation.
	convMult := 1.0 + mutations.GetConvictionCostMultiplier(user.Character.Mutations)
	totalConvictionCost := spellInfo.GetTotalConvictionCost(convMult)
	if totalConvictionCost > 0 && user.Character.Conviction < totalConvictionCost {
		user.SendText(fmt.Sprintf(
			`<ansi fg="red">You don't have the conviction to cast %s.</ansi>`,
			spellInfo.Name))
		return true, nil
	}

	// 6. Check initiation cooldown (blocks if a prior attempt failed)
	if user.Character.GetCooldown(`cast-init`) > 0 {
		user.SendText(`<ansi fg="red">Your mind is still recovering from the effort.</ansi>`)
		return true, nil
	}

	// 6.5. Component check — must have the required item in inventory.
	// Two separate mechanisms: ComponentTag (generic tag-based, consumed at
	// resolution) and SummonComponentId (specific item id, used by summon
	// spells and consumed in companion_summon.go). Both pre-validated here
	// so a missing component rejects the cast BEFORE the animation runs.
	if spellInfo.ComponentTag != "" {
		found := false
		for _, itm := range user.Character.Items {
			if itm.GetSpec().ComponentTag == spellInfo.ComponentTag {
				found = true
				break
			}
		}
		if !found {
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">%s requires a %s in your inventory.</ansi>`,
				spellInfo.Name, spellInfo.ComponentTag))
			return true, nil
		}
	}
	if spellInfo.SummonComponentId > 0 {
		found := false
		for _, itm := range user.Character.Items {
			if itm.ItemId == spellInfo.SummonComponentId {
				found = true
				break
			}
		}
		if !found {
			componentName := fmt.Sprintf("item #%d", spellInfo.SummonComponentId)
			if spec := items.GetItemSpec(spellInfo.SummonComponentId); spec != nil && spec.NameSimple != "" {
				componentName = spec.NameSimple
			} else if spec != nil && spec.Name != "" {
				componentName = spec.Name
			}
			user.SendText(fmt.Sprintf(
				`<ansi fg="red">%s requires a %s in your inventory.</ansi>`,
				spellInfo.Name, componentName))
			return true, nil
		}
	}

	// 7. Initiation roll — player-only gate.
	initiationChance := characters.CalcInitiationChance(user.Character.Stats.Willpower.ValueAdj, skillLevel)
	roll := util.Rand(100)
	util.LogRoll(`Spell Initiation`, roll, initiationChance)

	if roll >= initiationChance {
		// Failed — apply 2-round cooldown and inform user
		user.Character.TryCooldown(`cast-init`, `2 rounds`)
		user.SendText(`<ansi fg="red">` + spells.GetCastMessage("concentration_slipped", spellInfo.Name) + `</ansi>`)
		room.SendTextVisual(fmt.Sprintf(
			`<ansi fg="username">%s</ansi> <ansi fg="red">loses their concentration.</ansi>`,
			user.Character.Name), user.UserId)
		return true, nil
	}

	// 8. Stage 17.2: The Eye modulates Perception → folds-per-round for mutated
	// traditional casters. Manifestation spells use Charisma instead and are
	// NOT modulated by The Eye.
	isManifestation := spellInfo.HasSchool(spells.SchoolManifestation)
	var primaryStatForCast int
	if isManifestation {
		primaryStatForCast = user.Character.Stats.Charisma.ValueAdj
	} else {
		primaryStatForCast = user.Character.Stats.Perception.ValueAdj
		if len(user.Character.Mutations) > 0 {
			eyeFrac := (gametime.GetEyePhase() - 0.5) * 2 * float64(configs.GetBalanceConfig().MoonStatModMax)
			primaryStatForCast += int(float64(primaryStatForCast) * eyeFrac)
		}
	}

	// 9. Shared initiation logic.
	actor := &actions.UserActor{User: user, Room: room}
	result := actions.InitiateCast(actor, spellName, targetName)

	switch {
	case result.OnCooldown:
		user.SendText(`You need a moment before you can do that.`)
		return true, nil
	case result.NoTarget:
		// HarmSingle / HelpSingle can set this; supply a context-aware message.
		if spellInfo.Type == spells.HelpSingle {
			user.SendText(fmt.Sprintf(`<ansi fg="red">You don't see "%s" here.</ansi>`, targetName))
		} else {
			user.SendText(`<ansi fg="red">You need a target to cast that spell.</ansi>`)
		}
		return true, nil
	case !result.Initiated:
		// Catch-all for any other early exit (AlreadyCasting was handled above;
		// InvalidSpell was handled above). Should not normally be reached.
		return true, nil
	}

	// 10. Apply stat override for folds-per-round.
	// For traditional spells, this captures The Eye modulation of Perception.
	// For manifestation spells, this ensures Charisma is used (InitiateCast
	// already used it, but we recompute with the correct skill level here).
	var baseStatForCast int
	if isManifestation {
		baseStatForCast = user.Character.Stats.Charisma.ValueAdj
	} else {
		baseStatForCast = user.Character.Stats.Perception.ValueAdj
	}
	if primaryStatForCast != baseStatForCast {
		manifestSkill := user.Character.GetSkillLevel(skills.Manifestation)
		castSkill := user.Character.GetSkillLevel(skills.Spellcasting)
		var overrideSkill int
		if isManifestation {
			overrideSkill = manifestSkill
		} else {
			overrideSkill = castSkill
		}
		result.CastingState.FoldsPerRound = characters.CalcFoldsPerRound(
			primaryStatForCast, overrideSkill)
	}

	// 11. Apply conviction cost multiplier to the CastingState.
	result.CastingState.TotalConvictionCost = totalConvictionCost

	// 12. Commit CastingState.
	user.Character.CastingState = result.CastingState

	// 12b. Send YAML cast text (if defined).
	if spellInfo.CastUserText != "" || spellInfo.CastRoomText != "" {
		castRoom := rooms.LoadRoom(user.Character.RoomId)
		tCtx := textutil.TokenContext{
			SourceName:      user.Character.GetCharacterName(true),
			SourcePlainName: user.Character.GetCharacterName(false),
		}
		if len(result.TargetUserIds) > 0 {
			if tUser := users.GetByUserId(result.TargetUserIds[0]); tUser != nil {
				tCtx.TargetName = tUser.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tUser.Character.GetCharacterName(false)
			}
		} else if len(result.TargetMobInstanceIds) > 0 {
			if tMob := mobs.GetInstance(result.TargetMobInstanceIds[0]); tMob != nil {
				tCtx.TargetName = tMob.Character.GetCharacterName(true)
				tCtx.TargetPlainName = tMob.Character.GetCharacterName(false)
			}
		}
		cfg := textutil.SendTextConfig{
			UserSendFunc: func(msg string) { user.SendText(msg) },
			RoomSendFunc: func(msg string, skip ...int) {
				if castRoom != nil {
					castRoom.SendText(msg, skip...)
				}
			},
			ExcludeId: user.UserId,
		}
		textutil.SendPhaseText(spellInfo.CastUserText, spellInfo.CastRoomText, tCtx, "pink", cfg)
	}

	// 13. Go onCast hooks — run before JS, can abort the cast.
	if spellInfo.SpellId == "fold-recall" {
		currentRoomId := user.Character.RoomId
		blocked := false
		if currentRoom := rooms.LoadRoom(currentRoomId); currentRoom != nil {
			if allowed, ok := currentRoom.GetTempData("allow_recall").(bool); ok && !allowed {
				user.SendText("Something about this place prevents you from recalling.")
				blocked = true
			}
		}
		if !blocked {
			anchorRoom := 0
			if v := user.Character.GetMiscData("fold-anchor-room"); v != nil {
				switch val := v.(type) {
				case int:
					anchorRoom = val
				case float64:
					anchorRoom = int(val)
				}
			}
			if anchorRoom <= 0 {
				user.SendText(`You reach for the Veil, but there is no anchor to ` +
					`pull you. Set one first with ` +
					`<ansi fg="command">cast fold-anchor</ansi>.`)
				blocked = true
			} else if anchorRoom == currentRoomId {
				user.SendText("You are already standing on your anchor.")
				blocked = true
			}
		}
		if blocked {
			user.Character.CastingState = nil
			return true, nil
		}
	}

	// 14. Announce the cast start (skill progression now fires in InitiateCast).
	user.SendText(`<ansi fg="cyan">` + spells.GetCastMessage("cast_started", spellInfo.Name) + `</ansi>`)
	room.SendTextVisual(fmt.Sprintf(
		`<ansi fg="username">%s</ansi> closes their eyes in concentration.`,
		user.Character.Name), user.UserId)

	return true, nil
}
