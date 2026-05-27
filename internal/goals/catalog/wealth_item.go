package catalog

import (
	"strconv"

	goals "github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

func init() {
	goals.RegisterGoalType("wealth-item", goals.GoalTypeMeta{
		Predicate:     wealthItemPredicate,
		ContextScore:  wealthItemContextScore,
		AllowMultiple: true,
		DedupKey:      wealthItemDedupKey,
		Params: []goals.ParamSchema{
			{Key: "item_tag", Required: false, GoType: "string"},
			{Key: "item_id", Required: false, GoType: "int"},
		},
	})
}

// wealthItemDedupKey: "tag:<tag>" or "id:<n>". Caller is responsible
// for providing exactly one (Params schema allows either; mob authors
// can violate that — the dedup key just picks tag if present).
func wealthItemDedupKey(g *goals.Goal) string {
	if tag, ok := g.Params["item_tag"].(string); ok && tag != "" {
		return "tag:" + tag
	}
	if id := paramIntOr(g, "item_id", 0); id > 0 {
		return "id:" + strconv.Itoa(id)
	}
	return ""
}

func wealthItemPredicate(g *goals.Goal, mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	tag, _ := g.Params["item_tag"].(string)
	id := paramIntOr(g, "item_id", 0)
	return mobHasItem(mob, tag, id)
}

// wealthItemContextScore: 0 if present; 1.0 baseline if absent.
// Spec also describes a +0.5 bump if a shop in the mob's zone sells
// the item — implementable via a future shops-in-zone scan; deferred
// here since the engine surface for that scan isn't in 4.3's scope.
// 4.4's planner will add the shop-aware bump.
func wealthItemContextScore(g *goals.Goal, mob *mobs.Mob) float64 {
	if mob == nil {
		return 0
	}
	tag, _ := g.Params["item_tag"].(string)
	id := paramIntOr(g, "item_id", 0)
	if mobHasItem(mob, tag, id) {
		return 0
	}
	return 1.0
}

// mobHasItem checks backpack + equipment for a matching item.
func mobHasItem(mob *mobs.Mob, tag string, id int) bool {
	for _, it := range mob.Character.Items {
		if matchesItem(it, tag, id) {
			return true
		}
	}
	for _, it := range mob.Character.Equipment.GetAllItems() {
		if matchesItem(it, tag, id) {
			return true
		}
	}
	return false
}

func matchesItem(it items.Item, tag string, id int) bool {
	if id > 0 && it.ItemId == id {
		return true
	}
	if tag != "" {
		spec := it.GetSpec()
		if spec.ComponentTag == tag {
			return true
		}
	}
	return false
}
