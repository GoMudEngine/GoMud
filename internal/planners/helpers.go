package planners

import (
	"sort"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ─── Shop helpers ─────────────────────────────────────────────────────────────

// findShopInZoneSelling returns the room id of the first shop in mob's zone
// that stocks an item matching itemId (when >0) or component tag (when non-"").
// Returns ok=false if no match is found.
func findShopInZoneSelling(mob *mobs.Mob, tag string, itemId int) (roomId int, ok bool) {
	if mob == nil {
		return 0, false
	}

	// Pre-resolve tag → itemId once, outside the inner loop.
	tagItemId := 0
	if tag != "" {
		if spec := items.FindSpecByComponentTag(tag); spec != nil {
			tagItemId = spec.Id()
		}
	}

	for _, shop := range shops.AllShops() {
		if shop.Zone != mob.Character.Zone {
			continue
		}
		for _, entry := range shop.Stock {
			if itemId > 0 && entry.ItemId == itemId {
				return shop.RoomId, true
			}
			if tagItemId > 0 && entry.ItemId == tagItemId {
				return shop.RoomId, true
			}
		}
	}
	return 0, false
}

// findShopInZoneBuying returns the room id of the first shop in mob's zone
// that has gold reserves (i.e., can buy items from the mob). ok=false if none.
func findShopInZoneBuying(mob *mobs.Mob) (roomId int, ok bool) {
	if mob == nil {
		return 0, false
	}
	for _, shop := range shops.AllShops() {
		if shop.Zone != mob.Character.Zone {
			continue
		}
		if shop.Gold > 0 {
			return shop.RoomId, true
		}
	}
	return 0, false
}

// ─── Faction helpers ──────────────────────────────────────────────────────────

// findFactionMemberInZone returns the first mob instance in the same zone as
// mob that belongs to factionId. If mustBeInCombat is true, only members
// with Aggro != nil (i.e., in active combat) match.
func findFactionMemberInZone(mob *mobs.Mob, factionId string, mustBeInCombat bool) (target *mobs.Mob, ok bool) {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if mustBeInCombat && inst.Character.Aggro == nil {
			continue
		}
		for _, fid := range factions.FactionsForMob(inst) {
			if fid == factionId {
				return inst, true
			}
		}
	}
	return nil, false
}

// findHostileInZone returns the first auto-aggro mob in mob's zone (excluding
// self).
func findHostileInZone(mob *mobs.Mob) (target *mobs.Mob, ok bool) {
	zone := mob.Character.Zone
	for _, instId := range mobs.GetAllMobInstanceIds() {
		inst := mobs.GetInstance(instId)
		if inst == nil || inst.Character.Zone != zone {
			continue
		}
		if inst.InstanceId == mob.InstanceId {
			continue
		}
		if inst.AutoAggro {
			return inst, true
		}
	}
	return nil, false
}

// ─── Crafting station helpers ─────────────────────────────────────────────────

// findCraftingStationInZone returns the room id of the first room in mob's
// zone whose Station field matches stationName (case-sensitive). ok=false if
// none is found.
//
// Enumeration: ZoneConfig.RoomIds holds the complete room-id set for the zone,
// populated by addRoomToMemory at boot. Each room's Station field is a free-
// form string set in YAML (e.g. "forge", "loom", "alchemy").
func findCraftingStationInZone(mob *mobs.Mob, stationName string) (roomId int, ok bool) {
	if mob == nil || stationName == "" {
		return 0, false
	}
	zc := rooms.GetZoneConfig(mob.Character.Zone)
	if zc == nil {
		return 0, false
	}
	for rid := range zc.RoomIds {
		room := rooms.LoadRoom(rid)
		if room == nil {
			continue
		}
		if room.Station == stationName {
			return rid, true
		}
	}
	return 0, false
}

// ─── Zone graph helpers ───────────────────────────────────────────────────────

var zoneAdjacencyCache map[string][]string

// zoneAdjacentTo returns the list of zones that share a room-exit border with
// zoneA. Computed lazily on first call per zone and cached (the room graph is
// static after boot).
//
// Implementation: for each room in zoneA (via ZoneConfig.RoomIds), iterate
// each exit's destination via rooms.LoadRoom; collect the destination room's
// Zone if it differs from zoneA. Deduplicated, sorted for determinism.
func zoneAdjacentTo(zoneA string) []string {
	if zoneAdjacencyCache == nil {
		zoneAdjacencyCache = map[string][]string{}
	}
	if cached, ok := zoneAdjacencyCache[zoneA]; ok {
		return cached
	}

	zc := rooms.GetZoneConfig(zoneA)
	if zc == nil {
		zoneAdjacencyCache[zoneA] = []string{}
		return zoneAdjacencyCache[zoneA]
	}

	seen := map[string]struct{}{}
	for rid := range zc.RoomIds {
		room := rooms.LoadRoom(rid)
		if room == nil {
			continue
		}
		for _, ex := range room.Exits {
			dest := rooms.LoadRoom(ex.RoomId)
			if dest == nil || dest.Zone == zoneA {
				continue
			}
			seen[dest.Zone] = struct{}{}
		}
	}

	result := make([]string, 0, len(seen))
	for z := range seen {
		result = append(result, z)
	}
	sort.Strings(result)
	zoneAdjacencyCache[zoneA] = result
	return result
}

// ─── Inventory helpers ────────────────────────────────────────────────────────

// pickGiftItemFromInventory returns the highest-value non-quest item in mob's
// backpack suitable for gifting. Returns nil if the backpack is empty or
// contains only quest items.
//
// Filter: ItemSpec.QuestToken != "" marks an item as quest-related — skip it.
// Scoring: ItemSpec.Value (int, gold-equivalent) — higher is better.
func pickGiftItemFromInventory(mob *mobs.Mob) *items.Item {
	if mob == nil || len(mob.Character.Items) == 0 {
		return nil
	}
	var best *items.Item
	bestValue := 0
	for i := range mob.Character.Items {
		it := &mob.Character.Items[i]
		spec := it.GetSpec()
		if spec.QuestToken != "" {
			continue
		}
		if spec.Value > bestValue {
			best = it
			bestValue = spec.Value
		}
	}
	return best
}

// ─── Exit helpers ─────────────────────────────────────────────────────────────

// pickRandomExit returns a random exit direction name from mob's current room.
// Returns "" if the mob is nil, the room doesn't exist, or has no exits.
func pickRandomExit(mob *mobs.Mob) string {
	if mob == nil {
		return ""
	}
	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil || len(room.Exits) == 0 {
		return ""
	}
	names := make([]string, 0, len(room.Exits))
	for name := range room.Exits {
		names = append(names, name)
	}
	return names[util.Rand(len(names))]
}

// ─── Emote helpers ────────────────────────────────────────────────────────────

// pickSocialEmote returns a random social emote command string. Used by the
// befriend planner's interaction rotation and the mastery-skill planner's
// social branch.
func pickSocialEmote() string {
	emotes := []string{
		"emote nods",
		"emote bows",
		"emote smiles warmly",
		"emote waves",
		"emote grins",
	}
	return emotes[util.Rand(len(emotes))]
}

// ─── Recipe helpers ───────────────────────────────────────────────────────────

// pickKnownRecipeForSkill returns the id of the lowest-SkillMinimum recipe
// that (a) trains skillName and (b) is known by this mob. Returns "" if none
// found.
//
// Implementation: crafting.GetAllForSkill returns every recipe for the given
// skill sorted by name; we filter to those whose RecipeId appears in
// mob.Character.KnownRecipes, then pick the one with the lowest SkillMinimum.
func pickKnownRecipeForSkill(mob *mobs.Mob, skillName string) string {
	if mob == nil || len(mob.Character.KnownRecipes) == 0 {
		return ""
	}

	type candidate struct {
		id  string
		min int
	}
	var cands []candidate

	for _, spec := range crafting.GetAllForSkill(skillName) {
		if _, known := mob.Character.KnownRecipes[spec.RecipeId]; !known {
			continue
		}
		cands = append(cands, candidate{id: spec.RecipeId, min: spec.SkillMinimum})
	}

	if len(cands) == 0 {
		return ""
	}
	sort.SliceStable(cands, func(i, j int) bool { return cands[i].min < cands[j].min })
	return cands[0].id
}

// ─── MiscData helpers ─────────────────────────────────────────────────────────

// mobMiscIntOr reads an int from mob.Character.MiscData with a default value.
func mobMiscIntOr(mob *mobs.Mob, key string, def int) int {
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

// mobMiscStringOr reads a string from mob.Character.MiscData with a default
// value.
func mobMiscStringOr(mob *mobs.Mob, key string, def string) string {
	if mob == nil {
		return def
	}
	raw := mob.Character.GetMiscData(key)
	if s, ok := raw.(string); ok {
		return s
	}
	return def
}

// mobSetMisc writes a value to mob.Character.MiscData. Nil-safe; no-op if mob
// is nil.
func mobSetMisc(mob *mobs.Mob, key string, val any) {
	if mob == nil {
		return
	}
	mob.Character.SetMiscData(key, val)
}

// ─── Goal param helpers ───────────────────────────────────────────────────────

// goalParamIntOr reads an int param from goal.Params with a default value.
func goalParamIntOr(g *goals.Goal, key string, def int) int {
	if g == nil || g.Params == nil {
		return def
	}
	raw, ok := g.Params[key]
	if !ok {
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

// goalParamStringOr reads a string param from goal.Params with a default
// value.
func goalParamStringOr(g *goals.Goal, key string, def string) string {
	if g == nil || g.Params == nil {
		return def
	}
	if s, ok := g.Params[key].(string); ok {
		return s
	}
	return def
}
