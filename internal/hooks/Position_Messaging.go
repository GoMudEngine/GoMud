// Position_Messaging.go renders the three message classes called
// from Position_GrappleTick.go:
//
//   - fireGradientMessages: per-ControlLevel-boundary "you're losing
//     it" / "you're taking it" beats. One firing per level per
//     grapple — tracked in c.PerGrappleMessageCooldowns and cleared
//     when the grapple ends (handled in GrappleTick threshold escape).
//
//   - fireTransitionMessages: position-change broadcasts on escape.
//     Always fires.
//
//   - fireStaminaWarningIfLow: one-shot "you're getting gassed" beat
//     when stamina drops below GrappleStaminaLowThreshold. Reuses
//     the per-grapple cooldown map.
//
// Templates live at _datafiles/messages/position_control.yaml,
// loaded once via sync.Once.
package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v3"
)

type positionMsgPair struct {
	Self string `yaml:"self"`
	Room string `yaml:"room"`
}

// submissionMsgTriple holds attacker / target / room variants for
// a single submission message key. Attacker and target are personal
// messages; room goes to all other characters in the room.
type submissionMsgTriple struct {
	Attacker string `yaml:"attacker"`
	Target   string `yaml:"target"`
	Room     string `yaml:"room"`
}

type submissionMessageBlock struct {
	Opening            map[string]submissionMsgTriple `yaml:"opening"`
	EscapeBad          submissionMsgTriple            `yaml:"escape_bad"`
	Neutral            submissionMsgTriple            `yaml:"neutral"`
	OutcomeMercy       submissionMsgTriple            `yaml:"outcome_mercy"`
	OutcomeSubdue      submissionMsgTriple            `yaml:"outcome_subdue"`
	OutcomeCrippleArm  submissionMsgTriple            `yaml:"outcome_cripple_arm"`
	OutcomeCrippleShoulder submissionMsgTriple        `yaml:"outcome_cripple_shoulder"`
	OutcomeLethal      submissionMsgTriple            `yaml:"outcome_lethal"`
	CritFlag           submissionMsgTriple            `yaml:"crit_flag"`
}

type positionMessageTemplates struct {
	GradientMessages   map[string]map[string]positionMsgPair `yaml:"gradient_messages"`
	TransitionMessages map[string]positionMsgPair            `yaml:"transition_messages"`
	StaminaWarning     positionMsgPair                       `yaml:"stamina_warning"`
	Submission         submissionMessageBlock                 `yaml:"submission"`
}

var (
	posMsgOnce      sync.Once
	posMsgTemplates positionMessageTemplates
)

// loadPositionMessages reads + parses the YAML config once. Missing
// file is a Warn, not a fatal — the hook degrades to silent.
func loadPositionMessages() positionMessageTemplates {
	posMsgOnce.Do(func() {
		path := filepath.Join("_datafiles", "messages", "position_control.yaml")
		data, err := os.ReadFile(path)
		if err != nil {
			if os.IsNotExist(err) {
				mudlog.Warn("Position_Messaging: template file missing", "path", path)
				return
			}
			mudlog.Error("Position_Messaging: read failed", "err", err)
			return
		}
		if err := yaml.Unmarshal(data, &posMsgTemplates); err != nil {
			mudlog.Error("Position_Messaging: yaml parse failed", "err", err)
		}
	})
	return posMsgTemplates
}

// fireGradientMessages fires a per-level message when c's ControlLevel
// changes. Cooldown-gated: each level fires at most once per grapple
// session. Cooldowns clear on grapple-ending transition (handled in
// Position_GrappleTick.go).
func fireGradientMessages(c *characters.Character, from, to position.ControlLevel) {
	if from == to {
		return
	}
	if c == nil || c.Position == nil {
		return
	}
	role := "controlled"
	if c.IsController() {
		role = "controller"
	}
	levelKey := controlLevelKey(to)
	if levelKey == "" {
		return
	}
	cooldownKey := "gradient_" + role + "_" + levelKey
	if c.PerGrappleMessageCooldowns == nil {
		c.PerGrappleMessageCooldowns = map[string]bool{}
	}
	if c.PerGrappleMessageCooldowns[cooldownKey] {
		return
	}
	c.PerGrappleMessageCooldowns[cooldownKey] = true

	templates := loadPositionMessages()
	roleBlock, ok := templates.GradientMessages[role]
	if !ok {
		return
	}
	msg, ok := roleBlock[levelKey]
	if !ok {
		return
	}

	subs := substitutionsForCharacter(c)
	sendCharacterMsg(c, substitute(msg.Self, subs), substitute(msg.Room, subs))
}

// fireTransitionMessages fires the position-transition message pair
// on both sides + a room broadcast. Always fires (no cooldown — the
// transition itself is rare).
func fireTransitionMessages(controller, controlled *characters.Character, target position.State) {
	if controller == nil || controlled == nil ||
		controller.Position == nil || controlled.Position == nil {
		return
	}
	templates := loadPositionMessages()
	subs := map[string]string{
		"old_position": controller.Position.State().String(),
		"new_position": target.String(),
		"position":     controller.Position.State().String(),
		"Controller":   controller.Name,
		"Controlled":   controlled.Name,
		"Character":    controller.Name,
	}
	ctrlMsg := templates.TransitionMessages["controller"]
	cdMsg := templates.TransitionMessages["controlled"]

	sendCharacterMsg(controller, substitute(ctrlMsg.Self, subs), substitute(ctrlMsg.Room, subs))
	sendCharacterMsg(controlled, substitute(cdMsg.Self, subs), substitute(cdMsg.Room, subs))
}

// fireStaminaWarningIfLow fires a one-shot "getting gassed" beat
// when c's stamina drops below the grapple-low threshold. Reuses the
// per-grapple cooldown map so it cannot spam.
func fireStaminaWarningIfLow(c *characters.Character) {
	if c == nil || c.Position == nil {
		return
	}
	if !c.IsLowGrappleStamina() {
		return
	}
	if c.PerGrappleMessageCooldowns == nil {
		c.PerGrappleMessageCooldowns = map[string]bool{}
	}
	const cooldownKey = "stamina_low"
	if c.PerGrappleMessageCooldowns[cooldownKey] {
		return
	}
	c.PerGrappleMessageCooldowns[cooldownKey] = true

	templates := loadPositionMessages()
	subs := substitutionsForCharacter(c)
	sendCharacterMsg(c,
		substitute(templates.StaminaWarning.Self, subs),
		substitute(templates.StaminaWarning.Room, subs),
	)
}

// substitutionsForCharacter builds the standard token map for c's
// current grapple context. Resolves the partner's display name via
// GrappleData.Partner so room broadcasts get real names instead of
// "the other grappler" filler.
func substitutionsForCharacter(c *characters.Character) map[string]string {
	subs := map[string]string{
		"position":  c.Position.State().String(),
		"Character": c.Name,
	}
	partner := resolvePartner(c)
	partnerName := ""
	if partner != nil {
		partnerName = partner.Name
	}
	if c.IsController() {
		subs["Controller"] = c.Name
		subs["Controlled"] = partnerName
	} else {
		subs["Controlled"] = c.Name
		subs["Controller"] = partnerName
	}
	return subs
}

// controlLevelKey maps a ControlLevel to its YAML key.
func controlLevelKey(c position.ControlLevel) string {
	switch c {
	case position.InControl:
		return "in_control"
	case position.LosingControl:
		return "losing_control"
	case position.Neutral:
		return "neutral"
	case position.BecomingControlled:
		return "becoming_controlled"
	case position.Controlled:
		return "controlled"
	}
	return ""
}

// substitute does simple {key} replacement.
func substitute(template string, subs map[string]string) string {
	if template == "" {
		return ""
	}
	out := template
	for k, v := range subs {
		out = strings.ReplaceAll(out, "{"+k+"}", v)
	}
	return out
}

// sendCharacterMsg dispatches the self and room halves of a beat.
// Self goes to the player's connection (no-op for mobs). Room goes to
// the rest of the room, excluding the speaker so they don't read it
// twice.
func sendCharacterMsg(c *characters.Character, selfMsg, roomMsg string) {
	var excludeId int
	if u := userForCharacter(c); u != nil {
		excludeId = u.UserId
		if selfMsg != "" {
			u.SendText(selfMsg)
		}
	}
	if roomMsg == "" {
		return
	}
	r := rooms.LoadRoom(c.RoomId)
	if r == nil {
		return
	}
	if excludeId > 0 {
		r.SendText(roomMsg, excludeId)
	} else {
		r.SendText(roomMsg)
	}
}

// userForCharacter resolves the live user record for a player-side
// character. Mobs return nil. Uses the mobs package import indirectly
// — kept here to ensure we don't accidentally try to message a mob's
// "self" channel.
func userForCharacter(c *characters.Character) *users.UserRecord {
	if uid := c.GetUserId(); uid > 0 {
		return users.GetByUserId(uid)
	}
	return nil
}

// submissionTypeKey converts a SubmissionType to the lowercase YAML
// key used under submission.opening. Must match the keys in
// position_control.yaml exactly.
func submissionTypeKey(t position.SubmissionType) string {
	switch t {
	case position.SubArmbar:
		return "armbar"
	case position.SubRNC:
		return "rnc"
	case position.SubTriangle:
		return "triangle"
	case position.SubKimura:
		return "kimura"
	case position.SubAmericana:
		return "americana"
	case position.SubOmoplata:
		return "omoplata"
	case position.SubAnaconda:
		return "anaconda"
	default:
		return ""
	}
}

// sendSubmissionTriple dispatches a submissionMsgTriple to the
// attempter (personal), recipient (personal), and room (everyone
// else). Empty string slots are silently skipped.
func sendSubmissionTriple(
	attempter, recipient *characters.Character,
	tmpl submissionMsgTriple,
	subs map[string]string,
) {
	atkMsg := substitute(tmpl.Attacker, subs)
	tgtMsg := substitute(tmpl.Target, subs)
	roomMsg := substitute(tmpl.Room, subs)

	var excludeIds []int
	if ua := userForCharacter(attempter); ua != nil {
		if atkMsg != "" {
			ua.SendText(atkMsg)
		}
		excludeIds = append(excludeIds, ua.UserId)
	}
	if ur := userForCharacter(recipient); ur != nil {
		if tgtMsg != "" {
			ur.SendText(tgtMsg)
		}
		excludeIds = append(excludeIds, ur.UserId)
	}

	if roomMsg == "" {
		return
	}
	r := rooms.LoadRoom(attempter.RoomId)
	if r == nil {
		return
	}
	switch len(excludeIds) {
	case 0:
		r.SendText(roomMsg)
	case 1:
		r.SendText(roomMsg, excludeIds[0])
	default:
		r.SendText(roomMsg, excludeIds[0], excludeIds[1])
	}
}

// fireSubmissionOpeningMessage sends the "opening" message for a
// submission when its window first fires, before the outcome
// resolves. Picks the phrase from position_control.yaml under
// submission.opening.<subtype-key>. Degrades gracefully when the
// key is missing (no-op).
func fireSubmissionOpeningMessage(
	attempter, recipient *characters.Character,
	subType position.SubmissionType,
) {
	if attempter == nil || recipient == nil {
		return
	}
	key := submissionTypeKey(subType)
	if key == "" {
		return
	}
	templates := loadPositionMessages()
	tmpl, ok := templates.Submission.Opening[key]
	if !ok {
		return
	}
	subs := map[string]string{
		"attacker": attempter.Name,
		"target":   recipient.Name,
	}
	sendSubmissionTriple(attempter, recipient, tmpl, subs)
}

// fireSubmissionResolutionMessage sends the outcome message after
// the submission resolves. Picks the key based on tier + policy +
// bodyPart:
//
//   - SubTierBad                        → escape_bad
//   - SubTierNeutral                    → neutral
//   - SubTierSuccess/Crit + mercy       → outcome_mercy
//   - SubTierSuccess/Crit + subdue      → outcome_subdue
//   - SubTierSuccess/Crit + cripple arm → outcome_cripple_arm
//   - SubTierSuccess/Crit + cripple shoulder → outcome_cripple_shoulder
//   - SubTierSuccess/Crit + lethal      → outcome_lethal
//
// On a Crit, the crit_flag attacker prefix is prepended to the
// attacker's message before dispatch.
func fireSubmissionResolutionMessage(
	attempter, recipient *characters.Character,
	subType position.SubmissionType,
	tier combat.SubmissionTier,
	policy characters.SubmissionPolicy,
	bodyPart string,
) {
	if attempter == nil || recipient == nil {
		return
	}
	templates := loadPositionMessages()
	sub := templates.Submission

	var tmpl submissionMsgTriple
	switch tier {
	case combat.SubTierBad:
		tmpl = sub.EscapeBad
	case combat.SubTierNeutral:
		tmpl = sub.Neutral
	default: // SubTierSuccess, SubTierCrit
		switch policy {
		case characters.PolicyMercy:
			tmpl = sub.OutcomeMercy
		case characters.PolicySubdue:
			tmpl = sub.OutcomeSubdue
		case characters.PolicyCripple:
			if bodyPart == "shoulder" {
				tmpl = sub.OutcomeCrippleShoulder
			} else {
				// "arm" or degraded choke (body part "" already
				// redirected to subdue by ResolveSubmissionOutcome;
				// fall back to arm bucket for safety)
				tmpl = sub.OutcomeCrippleArm
			}
		case characters.PolicyLethal:
			tmpl = sub.OutcomeLethal
		default:
			tmpl = sub.OutcomeSubdue
		}
		// Crit: prepend the crit_flag prefix to the attacker text.
		if tier == combat.SubTierCrit && sub.CritFlag.Attacker != "" {
			tmpl.Attacker = sub.CritFlag.Attacker + tmpl.Attacker
		}
	}

	subs := map[string]string{
		"attacker": attempter.Name,
		"target":   recipient.Name,
	}
	sendSubmissionTriple(attempter, recipient, tmpl, subs)
}

func init() {
	combat.RegisterSubmissionMessaging(
		fireSubmissionOpeningMessage,
		fireSubmissionResolutionMessage,
	)
}

// Keep mobs import live for future helpers that may want to resolve
// mob names — the package is used by sibling files in this directory
// (Position_GrappleTick.go) but isn't called directly here.
var _ = mobs.GetInstance
