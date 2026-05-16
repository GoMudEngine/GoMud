// Position_ConsistencyCheck.go is the fail-safe scanner for grapple
// pair invariants. Runs every PositionConsistencyCheckRounds rounds
// (config knob, default 10); walks every grappling character; force-
// breaks any pair that fails ValidateGrapplePair.
//
// In a correctly-functioning engine this should never trigger —
// TransitionPair + GrappleTick maintain the four pair invariants by
// construction. The log line IS the bug report: production failures
// surface as engineering tickets instead of stuck players.
package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/state"
	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// processPositionConsistencyCheck runs every
// PositionConsistencyCheckRounds rounds. Walks all grappling
// characters; validates each pair; force-breaks invalid pairs to
// Standing. The `checked` set ensures each pair is validated once
// even though both sides iterate.
func processPositionConsistencyCheck(e events.Event) events.ListenerReturn {
	cfg := configs.GetBalanceConfig()
	interval := int(cfg.PositionConsistencyCheckRounds)
	if interval <= 0 {
		interval = 10
	}
	if util.GetRoundCount()%uint64(interval) != 0 {
		return events.Continue
	}

	checked := map[characterRef]bool{}

	for _, u := range users.GetAllActiveUsers() {
		if u == nil || u.Character == nil {
			continue
		}
		if !u.Character.IsGrappling() {
			continue
		}
		ref := characterRef{UserId: u.UserId}
		if checked[ref] {
			continue
		}
		partner := resolvePartner(u.Character)
		if partner == nil {
			forceBreakSolo(u.Character, "no resolvable partner")
			checked[ref] = true
			continue
		}
		if err := position.ValidateGrapplePair(u.Character, partner); err != nil {
			forceBreakPair(u.Character, partner, err)
		}
		checked[ref] = true
		checked[characterRefForCharacter(partner)] = true
	}

	for _, mobInstId := range mobs.GetAllMobInstanceIds() {
		m := mobs.GetInstance(mobInstId)
		if m == nil {
			continue
		}
		if !m.Character.IsGrappling() {
			continue
		}
		ref := characterRef{MobInstanceId: m.InstanceId}
		if checked[ref] {
			continue
		}
		partner := resolvePartner(&m.Character)
		if partner == nil {
			forceBreakSolo(&m.Character, "no resolvable partner")
			checked[ref] = true
			continue
		}
		if err := position.ValidateGrapplePair(&m.Character, partner); err != nil {
			forceBreakPair(&m.Character, partner, err)
		}
		checked[ref] = true
		checked[characterRefForCharacter(partner)] = true
	}

	return events.Continue
}

// characterRef de-duplicates checked characters across the user +
// mob iteration passes. UserId and MobInstanceId are mutually
// exclusive (a record is one or the other).
type characterRef struct {
	UserId        int
	MobInstanceId int
}

func characterRefForCharacter(c *characters.Character) characterRef {
	return characterRef{
		UserId:        c.GetUserId(),
		MobInstanceId: c.MobInstanceId,
	}
}

const triggerConsistencyBreak = "consistency_check_break"

// forceBreakSolo unwinds a grappling character whose partner can't
// be resolved (Turtle-solo is excluded by ValidateGrapplePair). The
// log entry is the bug report — production hits should never happen.
func forceBreakSolo(c *characters.Character, reason string) {
	mudlog.Warn("Position consistency: force-break solo",
		"user", c.GetUserId(), "mob", c.MobInstanceId, "reason", reason)
	if c.Position != nil {
		_ = c.Position.TransitionToStanding(state.TransitionReason{
			Trigger: triggerConsistencyBreak,
		})
	}
	notifyForceBreak(c)
}

// forceBreakPair unwinds both sides of an invalid pair. Each side
// goes to Standing independently — TransitionPair-with-Standing isn't
// used because we don't want a rollback on this fail-safe path.
func forceBreakPair(a, b *characters.Character, err error) {
	mudlog.Warn("Position consistency: force-break pair",
		"a_user", a.GetUserId(), "a_mob", a.MobInstanceId,
		"b_user", b.GetUserId(), "b_mob", b.MobInstanceId,
		"err", err)
	if a.Position != nil {
		_ = a.Position.TransitionToStanding(state.TransitionReason{
			Trigger: triggerConsistencyBreak,
		})
	}
	if b.Position != nil {
		_ = b.Position.TransitionToStanding(state.TransitionReason{
			Trigger: triggerConsistencyBreak,
		})
	}
	notifyForceBreak(a)
	notifyForceBreak(b)
}

// notifyForceBreak sends a generic "grapple breaks apart" beat to
// the character's room. Intentionally non-specific — the player
// shouldn't be exposed to invariant-violation diagnostics.
func notifyForceBreak(c *characters.Character) {
	r := rooms.LoadRoom(c.RoomId)
	if r == nil {
		return
	}
	r.SendText("The grapple suddenly breaks apart.")
}

func init() {
	events.RegisterListener(events.NewRound{}, processPositionConsistencyCheck)
}
