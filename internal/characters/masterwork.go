package characters

// HasOwnMasterwork reports whether the character carries any item they
// personally crafted (MakerName == their own Name) at craft-skill >=
// skillMin. Backs the pinnacle "show me a masterwork" entry gate (dialogue
// masterworkRequired + quest has_masterwork condition).
func (c *Character) HasOwnMasterwork(skillMin int) bool {
	for _, itm := range c.GetAllBackpackItems() {
		if itm.MakerName == c.Name && itm.CraftSkill >= skillMin {
			return true
		}
	}
	return false
}
