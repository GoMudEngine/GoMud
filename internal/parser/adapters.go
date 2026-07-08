package parser

import "github.com/GoMudEngine/GoMud/internal/items"

func nounAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	found, _ := s.Room.FindNoun(candidate)
	if found == "" {
		return Match{}, false
	}
	return Match{Kind: KindNoun, Name: found}, true
}

func exitAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	if _, ok := s.Room.Exits[candidate]; ok {
		return Match{Kind: KindExit, Name: candidate}, true
	}
	return Match{}, false
}

func containerAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	name := s.Room.FindContainerByName(candidate)
	if name == "" {
		return Match{}, false
	}
	return Match{Kind: KindRoomContainer, Name: name, ContainerName: name}, true
}

func corpseAdapter(s Scope, candidate string) (Match, bool) {
	if s.Room == nil {
		return Match{}, false
	}
	idx := s.Room.FindCorpseIndex(candidate)
	if idx < 0 {
		return Match{}, false
	}
	return Match{Kind: KindCorpse, Name: s.Room.Corpses[idx].DisplayName(), CorpseIdx: idx}, true
}

// matchInSlice resolves candidate against an item slice via items.FindMatchIn,
// preferring a full match over a partial one.
func matchInSlice(candidate string, list []items.Item) (items.Item, bool) {
	pMatch, fMatch := items.FindMatchIn(candidate, list...)
	if fMatch.ItemId != 0 {
		return fMatch, true
	}
	if pMatch.ItemId != 0 {
		return pMatch, true
	}
	return items.Item{}, false
}
