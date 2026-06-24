package characters

// GetEffectivePerception returns Perception adjusted for transient effects
// (currently toxicity). Use this for live combat/skill rolls; use
// Stats.Perception.ValueAdj directly only for training/progression/display.
func (c *Character) GetEffectivePerception() int {
	_, perMult, _ := c.GetToxicityPenalties()
	return int(float64(c.Stats.Perception.ValueAdj) * perMult)
}

// GetEffectiveDexterity returns Dexterity adjusted for transient effects
// (currently toxicity).
func (c *Character) GetEffectiveDexterity() int {
	_, _, dexMult := c.GetToxicityPenalties()
	return int(float64(c.Stats.Dexterity.ValueAdj) * dexMult)
}
