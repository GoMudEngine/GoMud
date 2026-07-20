package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MeleeTargetOpts carries the per-verb phrasing for AcquireMeleeTarget.
//
// Only Verb is required; the message fields default from it and exist so verbs
// whose wording differs (trip, pounce) keep their exact player-facing text.
type MeleeTargetOpts struct {
	// Verb is the lowercase command name, e.g. "bash". It drives the defaults
	// for every message below.
	Verb string

	// CraftingVerb overrides the verb inside the crafting-guard message.
	// Defaults to Verb. `trip` uses "trip someone".
	CraftingVerb string

	// PromptMsg overrides the empty-target prompt.
	// Defaults to "<Verb> whom?" (title-cased). `pounce` uses "Pounce on whom?".
	PromptMsg string

	// SelfTargetMsg overrides the self-targeting rejection.
	// Defaults to "You can't <Verb> yourself." `pounce` adds "on".
	SelfTargetMsg string

	// CharmedMsg overrides the charmed-target (player companion) rejection.
	// Defaults to "You can't <Verb> a companion." `pounce` adds "on".
	CharmedMsg string
}

func (o MeleeTargetOpts) craftingMsg() string {
	v := o.CraftingVerb
	if v == "" {
		v = o.Verb
	}
	return fmt.Sprintf(
		`<ansi fg="red">You can't %s while focused on your work. Finish or be interrupted first.</ansi>`, v)
}

func (o MeleeTargetOpts) promptMsg() string {
	if o.PromptMsg != "" {
		return o.PromptMsg
	}
	if o.Verb == "" {
		return "Attack whom?"
	}
	return strings.ToUpper(o.Verb[:1]) + o.Verb[1:] + " whom?"
}

func (o MeleeTargetOpts) charmedMsg() string {
	if o.CharmedMsg != "" {
		return o.CharmedMsg
	}
	return fmt.Sprintf("You can't %s a companion.", o.Verb)
}

func (o MeleeTargetOpts) selfTargetMsg() string {
	if o.SelfTargetMsg != "" {
		return o.SelfTargetMsg
	}
	return fmt.Sprintf("You can't %s yourself.", o.Verb)
}

// AcquireMeleeTarget runs the shared gate-and-engage preamble that every melee
// special move needs before delegating to its Execute* action.
//
// It returns handled=true when it has already sent the player a message and the
// calling command should return immediately. handled=false means the actor is
// engaged (or was already in combat) and the caller should proceed.
//
// Steps, in order:
//  1. Refuse if the actor is mid-activity (craft/salvage/cast).
//  2. If already in combat, do nothing — the existing target stands.
//  3. Otherwise require a target, resolve it, and reject self-targeting,
//     non-combatants, attack-immune mobs, player companions (charmed),
//     PvP-disallowed players, and the actor's own party members.
//  4. Set aggro so the subsequent Execute* call has an engagement to act on.
//
// This was copy-pasted across 11 command files (bash, drain, gore, grapple,
// kick, maul, pounce, rake, taunt, throttle, trip), differing only in wording.
// Beyond the duplication, each copy performed a redundant second room scan
// (room.FindByName) purely to distinguish "can't target yourself" from "not
// here" — that scan happens once here, and only on the error path.
func AcquireMeleeTarget(user *users.UserRecord, room *rooms.Room, rest string, opts MeleeTargetOpts) bool {

	if user.Character.IsActing() {
		user.SendText(messaging.CategorySystem, opts.craftingMsg())
		return true
	}

	// Already fighting: keep the current target.
	if user.Character.IsInCombat() {
		return false
	}

	if rest == "" {
		user.SendText(messaging.CategorySystem, opts.promptMsg())
		return true
	}

	target, err := ResolveTargetActor(room, rest, ResolveTargetOptions{
		ExcludeUserId: user.UserId,
	})
	if err != nil {
		// Self-exclusion collapses to NotFound, so distinguish the two here.
		if pId, _ := room.FindByName(rest); pId == user.UserId {
			user.SendText(messaging.CategorySystem, opts.selfTargetMsg())
			return true
		}
		user.SendText(messaging.CategorySystem, "You don't see them here.")
		return true
	}

	if !target.IsPlayer() {
		mob := target.(*MobActor).Mob
		if mob.IsNonCombatant() || mob.PlayerAttackImmune {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf(`You can't attack <ansi fg="mobname">%s</ansi>.`, mob.Character.Name))
			return true
		}
		// Player companions are off-limits to every melee special move, the
		// same way `attack` already refuses them. Before this was centralised,
		// only taunt enforced it, so a companion could be bashed/kicked/
		// grappled/tripped with no message at all.
		if mob.Character.IsCharmed() {
			user.SendText(messaging.CategorySystem, opts.charmedMsg())
			return true
		}
		user.Character.SetAggro(0, mob.InstanceId, characters.DefaultAttack)
		return false
	}

	p := target.(*UserActor).User
	if pvpErr := room.CanPvp(user, p); pvpErr != nil {
		user.SendText(messaging.CategorySystem, pvpErr.Error())
		return true
	}

	// Party members are not valid targets. room.CanPvp does NOT cover this — it
	// only checks fighting-allowed, PVP-enabled, the experience threshold and
	// PVP-area — so without this an enabled-PvP area let a player bash, kick,
	// grapple or trip their own party member, while `attack` and `shoot` both
	// refused. Same bypass shape as the charmed-companion gate above.
	if partyInfo := parties.Get(user.UserId); partyInfo != nil && partyInfo.IsMember(p.UserId) {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf(`<ansi fg="username">%s</ansi> is in your party!`, p.Character.Name))
		return true
	}

	user.Character.SetAggro(p.UserId, 0, characters.DefaultAttack)
	return false
}
