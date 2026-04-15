package dialogue

// FlagRef is a reference to a quest flag found in a dialogue file.
type FlagRef struct {
	Key    string
	Value  string
	Source string // e.g., "pattern", "node quest_start", "root variant"
}

// CollectFlagReferences scans a DialogueFile for all quest flag references.
// Returns refs (from questFlagRequired/questFlagExcluded) and sets (from setsQuestFlag).
func CollectFlagReferences(df *DialogueFile) (refs []FlagRef, sets []FlagRef) {
	if df == nil {
		return
	}

	for _, p := range df.Patterns {
		for k, v := range p.QuestFlagRequired {
			refs = append(refs, FlagRef{Key: k, Value: v, Source: "pattern"})
		}
		for k, v := range p.QuestFlagExcluded {
			refs = append(refs, FlagRef{Key: k, Value: v, Source: "pattern"})
		}
		if p.SetsQuestFlag != nil {
			sets = append(sets, FlagRef{Key: p.SetsQuestFlag.Key, Value: p.SetsQuestFlag.Value, Source: "pattern"})
		}
	}

	if df.Tree != nil {
		for _, v := range df.Tree.Root.Variants {
			for k, val := range v.QuestFlagRequired {
				refs = append(refs, FlagRef{Key: k, Value: val, Source: "root variant"})
			}
			for k, val := range v.QuestFlagExcluded {
				refs = append(refs, FlagRef{Key: k, Value: val, Source: "root variant"})
			}
		}
		for _, n := range df.Tree.Nodes {
			for k, v := range n.QuestFlagRequired {
				refs = append(refs, FlagRef{Key: k, Value: v, Source: "node " + n.Id})
			}
			for k, v := range n.QuestFlagExcluded {
				refs = append(refs, FlagRef{Key: k, Value: v, Source: "node " + n.Id})
			}
			if n.SetsQuestFlag != nil {
				sets = append(sets, FlagRef{Key: n.SetsQuestFlag.Key, Value: n.SetsQuestFlag.Value, Source: "node " + n.Id})
			}
		}
	}

	return
}
