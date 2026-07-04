package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// item_procs.go — the pinnacle item proc engine (Stage 1, Task 4).
//
// Procs are authored on ItemSpec (items.ItemProc: Trigger/Chance/
// CooldownRounds/Effect/Params, validated at load). This file gates them
// (chance roll + per-item-slot cooldown via MiscData) and dispatches the
// concrete effects. Effects live in Go — this is the combat hot path, so no
// scripting — and per-swing overhead is held to 1-2 spec lookups by
// procBearingItems narrowing which slots each trigger consults.
//
// Tasks 5-7 replace the three effect stubs (steal_pool / aoe_stun /
// apply_condition) and wire the remaining triggers (on_block / on_grapple /
// on_spell_hit) at their chokepoints. on_hit + on_kill are wired here.

// procCooldownKey identifies one proc slot on one item for MiscData
// bookkeeping. Cooldowns persist per (itemId, procIdx) so multiple procs on
// the same weapon track independently.
func procCooldownKey(itemId, procIdx int) string {
	return fmt.Sprintf("pinnacle_proc_cd_%d_%d", itemId, procIdx)
}

// readMiscRound tolerates int/uint64/float64 from yaml round-tripping —
// MiscData persists to player YAML and numeric types are not stable across a
// save/load cycle.
func readMiscRound(v any) (uint64, bool) {
	switch n := v.(type) {
	case uint64:
		return n, true
	case int:
		return uint64(n), true
	case int64:
		return uint64(n), true
	case float64:
		return uint64(n), true
	}
	return 0, false
}

// procGateOpen rolls chance and checks the cooldown. It does NOT mark the
// cooldown — callers mark only when the effect actually executed, so a proc
// that rolled but no-op'd (e.g. lifesteal on a 0-damage hit) doesn't burn its
// cooldown.
func procGateOpen(c *characters.Character, itemId, procIdx int, p items.ItemProc) bool {
	if c == nil {
		return false
	}
	if !bool(configs.GetConfig().GamePlay.ItemProcsEnabled) {
		return false
	}
	if p.CooldownRounds > 0 {
		if until, ok := readMiscRound(c.GetMiscData(procCooldownKey(itemId, procIdx))); ok {
			if util.GetRoundCount() < until {
				return false
			}
		}
	}
	if p.Chance < 100 && util.Rand(100) >= p.Chance {
		return false
	}
	return true
}

// markProcCooldown records the round at which this proc slot may fire again.
func markProcCooldown(c *characters.Character, itemId, procIdx int, p items.ItemProc) {
	if c == nil || p.CooldownRounds <= 0 {
		return
	}
	c.SetMiscData(procCooldownKey(itemId, procIdx), util.GetRoundCount()+uint64(p.CooldownRounds))
}

// procLifesteal heals the attacker for ratio*damage, clamped to HealthMax.
// Returns the amount actually healed.
func procLifesteal(attacker *characters.Character, damage int, params map[string]float64) int {
	if attacker == nil {
		return 0
	}
	ratio := params["ratio"]
	if ratio <= 0 || damage <= 0 {
		return 0
	}
	amt := int(float64(damage) * ratio)
	if amt < 1 {
		amt = 1
	}
	return attacker.Heal(amt)
}

// dispatchItemProcs fires all procs on the OWNER's relevant equipment for a
// trigger. owner = whoever the trigger belongs to (attacker for on_hit/
// on_kill/on_spell_hit, defender for on_block, either for on_grapple).
// other may be nil (on_kill after despawn). room may be nil where unknown.
func dispatchItemProcs(trigger string, owner, other *characters.Character, room *rooms.Room, damage int) {
	if owner == nil {
		return
	}
	for _, itm := range procBearingItems(owner, trigger) {
		if itm.ItemId <= 0 {
			continue
		}
		spec := itm.GetSpec()
		procs := spec.ProcsFor(trigger)
		if len(procs) == 0 {
			continue
		}
		for idx, p := range procs {
			if !procGateOpen(owner, itm.ItemId, idx, p) {
				continue
			}
			executed := false
			switch p.Effect {
			case "lifesteal":
				executed = procLifesteal(owner, damage, p.Params) > 0
			case "steal_pool":
				executed = procStealPool(owner, other, p.Params)
			case "aoe_stun":
				executed = procAoeStun(owner, room, p.Params)
			case "apply_condition":
				executed = procApplyCondition(other, p.Params)
			}
			if executed {
				markProcCooldown(owner, itm.ItemId, idx, p)
			}
		}
	}
}

// procBearingItems narrows which equipment slots a trigger consults: weapon
// triggers read the weapon; on_block reads the offhand; on_grapple reads body
// armor. Keeps the per-swing cost to 1-2 spec lookups.
func procBearingItems(c *characters.Character, trigger string) []items.Item {
	switch trigger {
	case "on_hit", "on_kill", "on_spell_hit":
		return []items.Item{c.Equipment.Weapon}
	case "on_block":
		return []items.Item{c.Equipment.Offhand}
	case "on_grapple":
		return []items.Item{c.Equipment.Body}
	}
	return nil
}

// procStealPool drains a pool from the target into the owner. Params:
// pool (3=conviction; 1=health/2=stamina reserved, unimplemented — YAGNI
// until an item needs them), amount_pct (fraction of the TARGET's pool
// max, capped by what they actually have). Executes only when something
// was actually stolen (so an empty-pool target does not burn the cooldown).
func procStealPool(owner, other *characters.Character, params map[string]float64) bool {
	if owner == nil || other == nil {
		return false
	}
	pct := params["amount_pct"]
	if pct <= 0 {
		return false
	}
	switch int(params["pool"]) {
	case 3: // conviction
		amt := int(float64(other.ConvictionMax.Value) * pct)
		if amt < 1 {
			amt = 1
		}
		if amt > other.Conviction {
			amt = other.Conviction
		}
		if amt <= 0 {
			return false
		}
		other.Conviction -= amt
		owner.Conviction += amt
		if owner.Conviction > owner.ConvictionMax.Value {
			owner.Conviction = owner.ConvictionMax.Value
		}
		return true
	}
	return false
}

// procApplyCondition applies a combat condition to the target. Params:
// condition (1=bleeding — the switch is the extension point for future
// condition ids; only bleeding is wired here, YAGNI), duration (rounds,
// default 4 if unset/<1), magnitude (per-tick, default 2 if unset/<1).
// Unknown condition ids do not execute (so the caller's cooldown isn't
// marked — see dispatchItemProcs).
func procApplyCondition(target *characters.Character, params map[string]float64) bool {
	if target == nil {
		return false
	}
	dur := int(params["duration"])
	if dur < 1 {
		dur = 4
	}
	mag := params["magnitude"]
	if mag < 1 {
		mag = 2
	}
	switch int(params["condition"]) {
	case 1:
		target.AddCondition(characters.ConditionBleeding, dur, mag, "itemproc")
		return true
	}
	return false
}

// procAoeStun applies the stagger-stun buff (84 — a 1-round Stunned) to every
// hostile, stun-eligible mob in the owner's room. Non-combatants,
// attack-immune, and charmed mobs are never targeted — stunning someone's
// companion or a town NPC would be a prod incident. Returns true if
// at least one target was stunned (only then is the proc's cooldown marked).
//
// Mob owners are a no-op: no Stage-2 mob wields an aoe_stun item, and "hostile
// to a mob" has no clean definition here, so we return false (cooldown
// unburned) rather than guess. owner.GetUserId() is 0 for mobs.
//
// The stun_rounds param is intentionally IGNORED: buff 84 is a fixed 1-round
// stagger (triggercount:1 in its YAML) and cannot be duration-scaled from data
// without hacking buff internals. The Stage-2 Aegis-of-Mockery shield tunes
// its strength via the proc's chance/cooldown fields instead.
func procAoeStun(owner *characters.Character, room *rooms.Room, params map[string]float64) bool {
	if owner == nil {
		return false
	}
	ownerUserId := owner.GetUserId()
	if ownerUserId <= 0 {
		// Mob (or unassigned) owner — no-op, see doc comment.
		return false
	}

	if room == nil {
		room = rooms.LoadRoom(owner.RoomId)
	}
	if room == nil {
		return false
	}

	stunned := 0
	for _, mobId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(mobId)
		if m == nil {
			continue
		}
		if m.IsNonCombatant() || m.PlayerAttackImmune {
			continue
		}
		// Spare ALL charmed companions, whoever their master is — the
		// player-cast HarmArea precedent (spell_resolution.go, resolveSpell)
		// uses bare IsCharmed() for exactly this: a non-party bystander's
		// companion caught in the shockwave would be a prod incident just as
		// surely as a party member's.
		if m.Character.IsCharmed() {
			continue
		}
		_ = m.Character.AddBuff(84, false)
		stunned++
	}

	if stunned == 0 {
		return false
	}

	// Room-wide narration, no raw numbers (project rule). CategorySubmission
	// matches buff 84's own submission-stagger flavor.
	room.SendTextVisual(messaging.CategorySubmission,
		`<ansi fg="yellow">A jarring shockwave ripples outward, staggering the hostile creatures nearby!</ansi>`)
	return true
}
