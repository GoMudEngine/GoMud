package hooks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/itemvoices"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// pinnacle_tick.go — the always-on per-round layer for pinnacle items
// (Stage 1, Task 11). Procs (item_procs.go) are event-driven off combat
// chokepoints; THIS file is the passive upkeep that runs once per player per
// round from UserRoundTick: hunger drain, ambient-potion buffs, aging freeze,
// mutation drip, and sentient chatter.
//
// Per-round cost discipline: the whole thing is gated by PinnacleItemsEnabled,
// and each sub-tick reads only the belt/weapon/worn slots it needs (GetSpec on
// an empty slot returns a zero ItemSpec, so the flag checks short-circuit for
// players wearing nothing pinnacle-flagged). Real cost on the common path: one
// GetAllWornItems() slice allocation (built once here, shared by the mutation
// and voice sub-ticks) plus a few MiscData map lookups. The bandolier
// fingerprint build only runs for players wearing an ambient_potions belt.
//
// NOTE (GetSpec semantics): items.Item.GetSpec() returns an ItemSpec *value*,
// never nil (an unknown/empty item yields a zero ItemSpec). So every guard here
// is a field check (`!spec.PreservesContents`, `spec.VoiceId == ""`), not a nil
// check — the plan's `spec == nil` sketch would not compile against the real API.

// pinnacleUserTick runs once per player per round from UserRoundTick.
// It early-outs entirely unless the pinnacle-items feature is enabled.
func pinnacleUserTick(user *users.UserRecord, room *rooms.Room) {
	if !bool(configs.GetConfig().GamePlay.PinnacleItemsEnabled) {
		return
	}
	c := user.Character
	now := util.GetRoundCount()
	worn := c.GetAllWornItems() // one pass, shared by the sub-ticks below

	tickPreserveContents(c)
	tickAmbientPotions(user, now)
	tickHunger(c, user, now)
	tickMutationItems(user, worn, now)
	tickVoices(user, room, worn, now)
}

// readMiscIntSlice tolerantly reads a []int from MiscData. Like readMiscRound,
// it copes with yaml round-tripping: a persisted []int comes back as []any of
// int/int64/float64. Returns nil for absent/other values.
func readMiscIntSlice(v any) []int {
	switch s := v.(type) {
	case []int:
		return s
	case []any:
		out := make([]int, 0, len(s))
		for _, e := range s {
			if n, ok := readMiscRound(e); ok {
				out = append(out, int(n))
			}
		}
		return out
	}
	return nil
}

// tickPreserveContents freezes aging for potions inside a preserves_contents
// bandolier by advancing CraftedRound 1:1 with the round counter (aging elapsed
// is `now - CraftedRound` at every call site, so keeping the gap constant means
// the potions never age while stored). An empty belt yields a zero spec whose
// PreservesContents is false, so this is a no-op for normal players.
func tickPreserveContents(c *characters.Character) {
	if !c.Equipment.Belt.GetSpec().PreservesContents {
		return
	}
	for i := range c.PotionItems {
		if c.PotionItems[i].ItemId <= 0 {
			continue
		}
		c.PotionItems[i].CraftedRound++
	}
}

// tickHunger drains the wielder when a hunger-flagged weapon (Blackrazor) has
// gone too long without a kill. Escalation grows with overdue time, capped at
// 3x, and NEVER kills outright — it clamps at 1 HP (the sword wants a living
// larder; item-suicide would be bad UX).
//
// Stale-anchor note: when the equipped weapon is NOT a hunger item this returns
// immediately, leaving any old pinnacle_hunger_anchor untouched. That is
// harmless — the gate is the *currently equipped* weapon's spec, so a leftover
// anchor is inert until a hunger weapon is wielded again (at which point the
// kill-round advance below re-bases it). No cleanup machinery, consistent with
// the Task 4 decision not to prune stale cooldown keys.
func tickHunger(c *characters.Character, user *users.UserRecord, now uint64) {
	spec := c.Equipment.Weapon.GetSpec()
	if spec.HungerRounds <= 0 || spec.HungerDrainPct <= 0 {
		return
	}
	anchor, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_anchor"))
	if !ok {
		// First tick wielding it — the hunger clock starts now.
		c.SetMiscData("pinnacle_hunger_anchor", now)
		return
	}
	// A kill since the anchor resets the clock (the blade was recently fed).
	if kill, ok := readMiscRound(c.GetMiscData("pinnacle_last_kill_round")); ok && kill > anchor {
		anchor = kill
		c.SetMiscData("pinnacle_hunger_anchor", kill)
	}
	if now <= anchor {
		return
	}
	elapsed := now - anchor
	if elapsed <= uint64(spec.HungerRounds) {
		return
	}
	overdue := elapsed - uint64(spec.HungerRounds)
	escalation := 1.0 + float64(overdue)/float64(spec.HungerRounds)
	if escalation > 3.0 {
		escalation = 3.0
	}
	drain := int(float64(c.HealthMax.Value) * spec.HungerDrainPct * escalation)
	if drain < 1 {
		drain = 1
	}
	if c.Health-drain < 1 {
		drain = c.Health - 1
	}
	if drain <= 0 {
		return
	}
	// Deliberate design decision: the drain is non-combat attrition applied
	// directly to Health, bypassing the damage hooks entirely (no sleep-wake,
	// no aggro, no mitigation) — the blade's toll, not an attack.
	c.Health -= drain
	// The drain repeats every overdue round, but the feeding LINE is paced by
	// its own cooldown (reusing the chatter knob) so an ignored hunger debt
	// doesn't spam the player every round.
	if user != nil {
		if next, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_msg_next_round")); !ok || now >= next {
			emitVoiceLine(user, nil, spec, "on_hunger_feeding",
				`<ansi fg="red">The blade feeds on you — a cold pull beneath your grip.</ansi>`)
			c.SetMiscData("pinnacle_hunger_msg_next_round",
				now+uint64(configs.GetBalanceConfig().SentientChatterCooldownRounds))
		}
	}
}

// tickMutationItems rolls each worn mutation-tick item (the Seething Prism).
// The roll only happens on interval-aligned rounds, then a percent gate, then
// a rarity-floored grant. worn is the caller's single GetAllWornItems() pass.
func tickMutationItems(user *users.UserRecord, worn []items.Item, now uint64) {
	c := user.Character
	for _, itm := range worn {
		spec := itm.GetSpec()
		if spec.MutationTickInterval <= 0 {
			continue
		}
		if now%uint64(spec.MutationTickInterval) != 0 {
			continue
		}
		if util.Rand(100) >= spec.MutationTickChance {
			continue
		}
		granted := c.GrantRandomMutationRare(spec.MutationRarityFloor)
		if granted == "" {
			continue
		}
		name := granted
		if ms := mutations.GetMutation(granted); ms != nil {
			name = ms.Name
		}
		user.SendText(messaging.CategoryMutation, fmt.Sprintf(
			`<ansi fg="magenta">Something stirs beneath your skin... <ansi fg="yellow">%s</ansi> takes root.</ansi>`, name))
	}
}

// ── Ambient potions (Vitalis Bandolier) ─────────────────────────────────────
//
// Attunement mechanism: CONTENT FINGERPRINTING (chosen over per-call-site
// stamping). The tick remembers a fingerprint (belt itemId + sorted potion
// itemIds) in MiscData; whenever it changes — the player drank a slotted
// potion, added/removed one, or swapped the belt — we stamp the attunement
// cooldown and revoke the ambience. This catches ALL mutation paths with zero
// edits to drink/get/remove/equip commands and is robust to future code, at
// the cost of one MiscData key + a fingerprint build — paid ONLY while an
// ambient_potions belt is worn (the flag-off path never builds it). First-ever
// equip trips attunement too (fp goes ""→something) — acceptable, arguably
// correct flavor. One leniency from flag-off-skips-fingerprint: unequipping the
// belt DURING the attunement window (no buffs applied yet) and re-equipping it
// with identical contents resumes the same attunement clock rather than
// resetting it — fine, the contents never changed.
//
// Deferred (Stage 2): the item card's "slotted potions can't be drunk" rule is
// item-level behavior for a later stage; the attunement cooldown is the Stage-1
// mechanical cost. We do NOT touch the toxicity path — ambient buffs never
// apply toxicity by construction.
//
// "Always-on" semantics, precisely: the !HasBuff guard RE-ADDS a buff after it
// expires (and is pruned); it does not refresh duration while active. Because
// prune runs on the turn tick, an ambient buff can lapse for up to one round
// between expiry and re-application. Accepted — matches WornBuffIds semantics;
// a per-tick unconditional refresh would cost a Validate() per player per round.

// bandolierFingerprint is a stable string of belt itemId + sorted potion
// itemIds. Changes iff the player-visible bandolier contents change.
func bandolierFingerprint(belt items.Item, potions []items.Item) string {
	ids := make([]int, 0, len(potions))
	for _, p := range potions {
		ids = append(ids, p.ItemId)
	}
	sort.Ints(ids)
	var b strings.Builder
	fmt.Fprintf(&b, "%d:", belt.ItemId)
	for _, id := range ids {
		fmt.Fprintf(&b, "%d,", id)
	}
	return b.String()
}

// tickAmbientPotions keeps slotted potion buffs active at Peak potency while an
// ambient_potions bandolier is worn and attuned. Buffs applied this way are
// recorded (pinnacle_bandolier_buffs) so removal can revoke them.
func tickAmbientPotions(user *users.UserRecord, now uint64) {
	c := user.Character
	belt := c.Equipment.Belt
	spec := belt.GetSpec()

	if !spec.AmbientPotions {
		// Flag-off common path: no fingerprint work at all. Revoke any
		// lingering ambience from a previously-worn ambient bandolier and
		// clear the stored fingerprint so re-equipping one re-attunes.
		if applied := readMiscIntSlice(c.GetMiscData("pinnacle_bandolier_buffs")); len(applied) > 0 {
			revokeAmbient(c, applied)
			c.SetMiscData("pinnacle_bandolier_fingerprint", nil)
		}
		return
	}

	applied := readMiscIntSlice(c.GetMiscData("pinnacle_bandolier_buffs"))

	// Content-change detection (see mechanism note above).
	fp := bandolierFingerprint(belt, c.PotionItems)
	prevFp, _ := c.GetMiscData("pinnacle_bandolier_fingerprint").(string)
	if fp != prevFp {
		c.SetMiscData("pinnacle_bandolier_fingerprint", fp)
		c.SetMiscData("pinnacle_bandolier_attune_round",
			now+uint64(configs.GetBalanceConfig().BandolierAttuneRounds))
		revokeAmbient(c, applied)
		return
	}
	if attune, ok := readMiscRound(c.GetMiscData("pinnacle_bandolier_attune_round")); ok && now < attune {
		revokeAmbient(c, applied)
		return
	}

	// Apply/refresh the buffs of every slotted potion at Peak potency (1.30).
	current := map[int]bool{}
	for _, p := range c.PotionItems {
		for _, buffId := range p.GetSpec().BuffIds {
			current[buffId] = true
			if !c.Buffs.HasBuff(buffId) {
				_ = c.AddBuffScaled(buffId, 1.30)
			}
		}
	}
	// Revoke any buff whose potion left the bandolier since last tick.
	for _, id := range applied {
		if !current[id] {
			c.RemoveBuff(id)
		}
	}
	// Persist the applied set as []int (readMiscIntSlice reads it tolerantly).
	ids := make([]int, 0, len(current))
	for id := range current {
		ids = append(ids, id)
	}
	c.SetMiscData("pinnacle_bandolier_buffs", ids)
}

// revokeAmbient removes every previously-applied ambient buff and clears the
// tracking key.
func revokeAmbient(c *characters.Character, applied []int) {
	for _, id := range applied {
		c.RemoveBuff(id)
	}
	if len(applied) > 0 {
		c.SetMiscData("pinnacle_bandolier_buffs", []int{})
	}
}

// ── Sentient chatter (Blackrazor / Aegis voices) ────────────────────────────

// pickVoiceEvent selects which voice event a worn sentient item should speak
// this round. Pure (no side effects) so it can be unit-tested directly:
// combat → taunt; a hungry weapon past 3/4 of its hunger window → hunger
// warning; otherwise idle.
func pickVoiceEvent(c *characters.Character, spec items.ItemSpec, now uint64) string {
	if c.Aggro != nil {
		return "on_taunt"
	}
	if spec.HungerRounds > 0 {
		if anchor, ok := readMiscRound(c.GetMiscData("pinnacle_hunger_anchor")); ok {
			if now > anchor && now-anchor > uint64(spec.HungerRounds)*3/4 {
				return "on_hunger_warning"
			}
		}
	}
	return "on_idle"
}

// tickVoices lets sentient items speak, paced by SentientChatterCooldownRounds
// and limited to one line per round across all worn sentient items.
//
// The fire chance (SentientChatterChancePct) is rolled ONCE per round (a single
// roll gating the whole voice tick), only after we confirm a speakable line
// exists — so quiet gear pays no RNG. This is the simpler of the two options in
// the spec and it inherently enforces the one-line-per-round rule. worn is the
// caller's single GetAllWornItems() pass.
//
// First-slot precedence is intentional: when a player wears MULTIPLE sentient
// items, the first one in slot order with a speakable line always wins the
// round (the later ones never roll). Dual-sentient loadouts are rare enough
// that round-robin fairness isn't worth the bookkeeping — revisit if a second
// wearable voice item ships.
func tickVoices(user *users.UserRecord, room *rooms.Room, worn []items.Item, now uint64) {
	c := user.Character
	if next, ok := readMiscRound(c.GetMiscData("pinnacle_voice_next_round")); ok && now < next {
		return
	}
	cool := uint64(configs.GetBalanceConfig().SentientChatterCooldownRounds)
	for _, itm := range worn {
		spec := itm.GetSpec()
		if spec.VoiceId == "" {
			continue
		}
		v := itemvoices.GetVoice(spec.VoiceId)
		if v == nil {
			continue
		}
		event := pickVoiceEvent(c, spec, now)
		// Cheap check for authored lines (no RNG pick) before the fire gate.
		if len(v.Lines[event]) == 0 {
			continue
		}
		// Occasional chatter: one roll for the whole tick.
		if util.Rand(100) >= int(configs.GetBalanceConfig().SentientChatterChancePct) {
			return
		}
		if emitVoiceLine(user, room, spec, event, "") {
			c.SetMiscData("pinnacle_voice_next_round", now+cool)
		}
		return // one line per round across all sentient items
	}
}

// emitVoiceLine sends an authored voice line for `event` from `spec` to the
// user, and — when room != nil — a muttered visual variant to the rest of the
// room. When the item has no voice or no authored line for the event it emits
// `fallback` (used by tickHunger's guaranteed feeding flavor); an empty
// fallback means "say nothing". Returns true iff something was actually sent
// (tickVoices uses this to arm the chatter cooldown only on a real utterance).
func emitVoiceLine(user *users.UserRecord, room *rooms.Room, spec items.ItemSpec, event, fallback string) bool {
	line := ""
	if spec.VoiceId != "" {
		if v := itemvoices.GetVoice(spec.VoiceId); v != nil {
			line = v.Line(event)
		}
	}
	if line == "" {
		if fallback == "" {
			return false
		}
		user.SendText(messaging.CategorySystem, fallback)
		return true
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(
		`<ansi fg="item">%s</ansi> says, "<ansi fg="yellow">%s</ansi>"`, spec.Name, line))
	if room != nil {
		room.SendTextVisual(messaging.CategorySystem, fmt.Sprintf(
			`<ansi fg="username">%s</ansi>'s <ansi fg="item">%s</ansi> mutters, "<ansi fg="yellow">%s</ansi>"`,
			user.Character.Name, spec.Name, line), user.UserId)
	}
	return true
}
