package rooms

import (
	"github.com/GoMudEngine/GoMud/internal/gamelock"
	"github.com/GoMudEngine/GoMud/internal/items"
)

type Container struct {
	Lock         gamelock.Lock `yaml:"lock,omitempty"`         // 0 - no lock. greater than zero = difficulty to unlock.
	Items        []items.Item  `yaml:"items,omitempty"`        // Save contents now, since players can put new items in there
	Gold         int           `yaml:"gold,omitempty"`         // Save contents now, since players can put new items in there
	Hidden       bool          `yaml:"hidden,omitempty"`       // If true, container is invisible until discovered via search.
	DespawnRound uint64        `yaml:"despawnround,omitempty"` // If this is set, it's a chest that will disappear with time.
	Recipes      map[int][]int `yaml:"recipes,omitempty,flow"` // Item Id's (key) that are created when the recipe is present in the container (values) and it is "used"
}

func (c Container) HasLock() bool {
	return c.Lock.Difficulty > 0
}

func (c *Container) AddItem(i items.Item) {
	c.Items = append(c.Items, i)
}

func (c *Container) RemoveItem(i items.Item) {
	for j := len(c.Items) - 1; j >= 0; j-- {
		if c.Items[j].Equals(i) {
			c.Items = append(c.Items[:j], c.Items[j+1:]...)
			break
		}
	}
}

func (c *Container) FindItem(itemName string) (items.Item, bool) {

	// Search floor
	closeMatchItem, matchItem := items.FindMatchIn(itemName, c.Items...)

	if matchItem.ItemId != 0 {
		return matchItem, true
	}

	if closeMatchItem.ItemId != 0 {
		return closeMatchItem, true
	}

	return items.Item{}, false
}

func (c *Container) FindItemById(itemId int) (items.Item, bool) {

	// Search floor
	for _, matchItem := range c.Items {
		if matchItem.ItemId == itemId {
			return matchItem, true
		}
	}

	return items.Item{}, false
}

// Returns an itemId if it can produce one based on contents + recipe
func (c *Container) RecipeReady() int {

	if len(c.Recipes) == 0 {
		return 0
	}

	for finalItemId, recipeList := range c.Recipes {

		totalNeeded := 0
		neededItems := map[int]int{}

		for _, inputItemId := range recipeList {
			neededItems[inputItemId] += 1
			totalNeeded++
		}

		for _, containsItem := range c.Items {
			if neededItems[containsItem.ItemId] > 0 {
				neededItems[containsItem.ItemId] -= 1
				totalNeeded--
			}
			if totalNeeded == 0 {
				break
			}
		}

		if totalNeeded < 1 {
			return finalItemId
		}
	}

	return 0
}

func (c *Container) Count(itemId int) int {
	total := 0
	for _, containsItem := range c.Items {
		if containsItem.ItemId == itemId {
			total++
		}
	}
	return total
}
