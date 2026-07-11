package mutations

// GetAllyAuraBuffs returns the buff ids that owned mutations project onto
// nearby allies (effect type "aura_ally_buff", Value = buff id).
func GetAllyAuraBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_ally_buff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}

// GetEnemyAuraBuffs returns the debuff ids that owned mutations project onto
// nearby enemies (effect type "aura_enemy_debuff", Value = buff id).
func GetEnemyAuraBuffs(owned map[string]int) []int {
	var out []int
	for id := range owned {
		spec := GetMutation(id)
		if spec == nil {
			continue
		}
		for _, p := range spec.Pros {
			if p.Type == "aura_enemy_debuff" && p.Value > 0 {
				out = append(out, int(p.Value))
			}
		}
	}
	return out
}
