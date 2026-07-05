package questengine

// PlayerState is the interface the quest engine uses to check player state.
// This avoids importing the characters package directly.
type PlayerState interface {
	HasQuest(token string) bool
	HasItem(itemId int) bool
	GetRoomId() int
	GetQuestFlag(key string) string
	GetGold() int
	HasOwnMasterwork(skillMin int) bool
}

// EvalConditions checks all conditions against the player's current state.
// Returns true if ALL conditions pass.
func EvalConditions(c Conditions, p PlayerState) bool {
	for _, token := range c.Has {
		if !p.HasQuest(token) {
			return false
		}
	}
	for _, token := range c.Missing {
		if p.HasQuest(token) {
			return false
		}
	}
	if c.InRoom > 0 && p.GetRoomId() != c.InRoom {
		return false
	}
	if c.HasItem > 0 && !p.HasItem(c.HasItem) {
		return false
	}
	if c.MissingItem > 0 && p.HasItem(c.MissingItem) {
		return false
	}
	for key, val := range c.HasFlag {
		if p.GetQuestFlag(key) != val {
			return false
		}
	}
	for key, val := range c.MissingFlag {
		if p.GetQuestFlag(key) == val {
			return false
		}
	}
	if c.HasGold > 0 && p.GetGold() < c.HasGold {
		return false
	}
	if c.HasMasterwork > 0 && !p.HasOwnMasterwork(c.HasMasterwork) {
		return false
	}
	return true
}
