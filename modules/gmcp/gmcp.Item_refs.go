package gmcp

// Item-template reference scan for Build.Item.Delete. Lives in the gmcp layer
// (not internal/items) because it imports mobs/rooms/quests/crafting, which
// themselves import items — putting it in items would cycle. The scan runs
// against injected iterators so it is unit-testable without those packages.

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/quests"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

type mobRef struct {
	id   int
	name string
	ids  []int
}

// refIterators yields each candidate reference source; injected so the scan is
// testable with fakes.
type refIterators struct {
	mobs       func(yield func(mobRef))
	recipes    func(yield func(recipeId string, outputId int))
	quests     func(yield func(questId int, rewardIds []int))
	containers func(yield func(roomId int, itemIds []int))
}

func scanItemReferencesWith(itemId int, it refIterators) []itemRef {
	out := []itemRef{}
	if it.mobs != nil {
		it.mobs(func(m mobRef) {
			if containsInt(m.ids, itemId) {
				out = append(out, itemRef{Kind: "mob", Id: fmt.Sprintf("mob %d (%s)", m.id, m.name)})
			}
		})
	}
	if it.recipes != nil {
		it.recipes(func(rid string, outputId int) {
			if outputId == itemId {
				out = append(out, itemRef{Kind: "recipe", Id: "recipe " + rid})
			}
		})
	}
	if it.quests != nil {
		it.quests(func(qid int, rewardIds []int) {
			if containsInt(rewardIds, itemId) {
				out = append(out, itemRef{Kind: "quest", Id: fmt.Sprintf("quest %d", qid)})
			}
		})
	}
	if it.containers != nil {
		it.containers(func(roomId int, itemIds []int) {
			if containsInt(itemIds, itemId) {
				out = append(out, itemRef{Kind: "container", Id: fmt.Sprintf("room %d chest", roomId)})
			}
		})
	}
	return out
}

func containsInt(s []int, v int) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}

// parseItemInfoIds parses a quest ItemInfo stockpile string ("itemid[:qty],...")
// into item ids, skipping non-numeric tokens.
func parseItemInfoIds(s string) []int {
	out := []int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if i := strings.IndexByte(part, ':'); i >= 0 {
			part = part[:i]
		}
		if n, err := strconv.Atoi(strings.TrimSpace(part)); err == nil {
			out = append(out, n)
		}
	}
	return out
}

// scanItemReferences wires the real world iterators. Recipe INGREDIENTS reference
// items by component tag (not id), so only the recipe OUTPUT id is scanned. Shop
// stock is dynamic living state (self-heals on restock), so it is NOT scanned.
func scanItemReferences(itemId int) []itemRef {
	return scanItemReferencesWith(itemId, refIterators{
		mobs: func(yield func(mobRef)) {
			for _, m := range mobs.AllMobTemplates() {
				ids := append(append([]int{}, m.LootPool...), m.CrafterRestockMaterials...)
				yield(mobRef{id: int(m.MobId), name: m.Character.Name, ids: ids})
			}
		},
		recipes: func(yield func(string, int)) {
			for id, r := range crafting.GetAll() {
				yield(id, r.Output.ItemId)
			}
		},
		quests: func(yield func(int, []int)) {
			for _, q := range quests.GetAllQuests() {
				ids := []int{}
				if q.Rewards.ItemId > 0 {
					ids = append(ids, q.Rewards.ItemId)
				}
				ids = append(ids, parseItemInfoIds(q.Rewards.ItemInfo)...)
				yield(q.QuestId, ids)
			}
		},
		containers: func(yield func(int, []int)) {
			for _, roomId := range rooms.GetAllRoomIds() {
				r := rooms.LoadRoom(roomId)
				if r == nil || len(r.Containers) == 0 {
					continue
				}
				ids := []int{}
				for _, c := range r.Containers {
					for _, it := range c.Items {
						ids = append(ids, it.ItemId)
					}
				}
				if len(ids) > 0 {
					yield(roomId, ids)
				}
			}
		},
	})
}
