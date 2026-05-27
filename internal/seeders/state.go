package seeders

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// CooldownKeyPrefix is the MiscData key prefix used by applyCooldown.
// Distinct from 4.4's "plan:" prefix so ClearPlanState (fired on goal
// switch) does NOT wipe seeder cooldowns — the cooldown is independent
// of strategic-layer state.
const CooldownKeyPrefix = "seed_cooldown:"

// applyCooldown returns true and writes a fresh expiration round if no
// active cooldown exists for the (rule, key) pair on this mob. Returns
// false (no write) if the cooldown is still active.
//
// Cooldown markers live on the BENEFICIARY mob's MiscData under
//
//	"seed_cooldown:<rule_name>:<key>"
//
// where key is a per-rule identifier (e.g., "<userId>" for gift,
// "<attackerInstanceId>" for combat-assist).
//
// windowRounds is the cooldown duration. The stored value is the round
// at which the cooldown EXPIRES — so a fresh call writes
// (currentRound + windowRounds).
func applyCooldown(mob *mobs.Mob, ruleName, key string, windowRounds uint64) bool {
	if mob == nil {
		return false
	}
	miscKey := CooldownKeyPrefix + ruleName + ":" + key
	nowRound := util.GetRoundCount()

	expires := readMiscUint64(mob, miscKey)
	if nowRound < expires {
		return false // cooldown active
	}

	mob.Character.SetMiscData(miscKey, nowRound+windowRounds)
	return true
}

// seedRevengeGoalIfAbsent checks whether a revenge-mob goal targeting
// the same (kind, id) already exists on the mob; if so returns nil
// (dedup — don't escalate priority on repeat offense in 4.5).
// Otherwise calls goals.Add with the standard revenge-mob shape and
// returns the added Goal.
//
// Returns nil on goals.Add failure (logged at warn level).
func seedRevengeGoalIfAbsent(mob *mobs.Mob, targetKind string, targetId, priority int) *goals.Goal {
	if mob == nil || targetKind == "" || targetId == 0 {
		return nil
	}
	mobId := int(mob.MobId)
	name := util.ConvertForFilename(mob.Character.Name)

	// Pre-check: is there already a revenge-mob goal targeting this
	// same (kind, id) on the mob? Walk GoalsOf and inspect Params.
	existing := goals.GoalsOf(mobId, name)
	for _, g := range existing {
		if g.Type != "revenge-mob" {
			continue
		}
		eKind, _ := g.Params["target_kind"].(string)
		eId := paramAsInt(g.Params["target_id"])
		if eKind == targetKind && eId == targetId {
			return nil // already targeting this revenge
		}
	}

	g := &goals.Goal{
		Type:     "revenge-mob",
		Priority: priority,
		Params: map[string]any{
			"target_kind": targetKind,
			"target_id":   targetId,
		},
	}
	res, err := goals.Add(mobId, name, g)
	if err != nil {
		mudlog.Warn("seeders.seedRevenge: Add failed",
			"mob_id", mobId, "target_kind", targetKind,
			"target_id", targetId, "error", err)
		return nil
	}
	return res.Added
}

// readMiscUint64 reads a uint64 (or coerces int/int64) from MiscData.
// Returns 0 if absent or wrong type.
func readMiscUint64(mob *mobs.Mob, key string) uint64 {
	if mob == nil {
		return 0
	}
	raw := mob.Character.GetMiscData(key)
	if raw == nil {
		return 0
	}
	switch v := raw.(type) {
	case uint64:
		return v
	case int64:
		return uint64(v)
	case int:
		return uint64(v)
	}
	return 0
}

// readMiscInt reads an int from MiscData (coerces int/int64). Returns
// def if absent or wrong type. Mirrors the helpers in 4.3 catalog and
// 4.4 planners packages.
func readMiscInt(mob *mobs.Mob, key string, def int) int {
	if mob == nil {
		return def
	}
	raw := mob.Character.GetMiscData(key)
	if raw == nil {
		return def
	}
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return def
}

// bumpMiscInt increments an int MiscData value by delta. Initializes
// to delta if absent. Tolerates int / int64 coercion.
func bumpMiscInt(mob *mobs.Mob, key string, delta int) {
	if mob == nil {
		return
	}
	current := readMiscInt(mob, key, 0)
	mob.Character.SetMiscData(key, current+delta)
}

// paramAsInt coerces a Goal.Params value to int. Returns 0 on failure
// (matches the catalog's tolerance for int vs int64 from YAML).
func paramAsInt(raw any) int {
	switch v := raw.(type) {
	case int:
		return v
	case int64:
		return int(v)
	}
	return 0
}

// resolveKillerFromMobDeath inspects a MobDeath event and returns the
// killer's (kind, id) tuple. Returns ("", 0) if the event doesn't
// identify a mob killer or the type assertion fails.
//
// events.MobDeath has KillerMobInstanceId (0 = player or unclear).
// There is no KillerUserId field on the struct — player-kill attribution
// is implicit (non-zero PlayerDamage map). For chunk 4.5 rule 1 we only
// care about mob killers, so player kills return ("", 0) and the rule
// skips. Future rules wanting player-killer attribution can add a
// separate resolver helper that walks the PlayerDamage map.
func resolveKillerFromMobDeath(event events.Event) (kind string, id int) {
	md, ok := event.(events.MobDeath)
	if !ok {
		return "", 0
	}
	if md.KillerMobInstanceId != 0 {
		return "mob", md.KillerMobInstanceId
	}
	return "", 0
}

// instanceIdAsKey converts a mob InstanceId int to a stable string key
// for cooldown lookups.
func instanceIdAsKey(id int) string { return strconv.Itoa(id) }

// userIdAsKey converts a user id int to a stable string key.
func userIdAsKey(id int) string { return strconv.Itoa(id) }
