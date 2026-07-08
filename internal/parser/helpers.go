package parser

import "strings"

// splitOnConnective splits input on the first standalone occurrence of the
// connective word (e.g. "to"), returning the trimmed left and right sides.
// found=false when the connective is absent (or leads/trails the input).
func splitOnConnective(input, connective string) (left, right string, found bool) {
	tokens := strings.Fields(input)
	for i, tok := range tokens {
		if tok == connective && i > 0 && i < len(tokens)-1 {
			return strings.Join(tokens[:i], " "), strings.Join(tokens[i+1:], " "), true
		}
	}
	return input, "", false
}

// SplitTrailingContainer detects whether input ends in a container / corpse /
// pet and, if so, returns the leading item span plus the container Match. It
// does NOT resolve the item inside — callers (e.g. get.go) apply their own gates
// and item lookup. Handles both "X from Y" and "X Y" forms.
//
// The no-"from" form tries each "<item> <container>" split from the longest item
// / shortest container span down and returns the first split whose trailing span
// resolves to a container/corpse/pet.
func SplitTrailingContainer(s Scope, input string) (itemPart string, cm Match, ok bool) {
	// Explicit "from <container>".
	if left, right, found := splitOnConnective(input, "from"); found {
		if m, matched := Resolve(s, right, KindRoomContainer, KindCorpse, KindPet); matched {
			return left, m, true
		}
		return "", Match{}, false
	}
	// No "from".
	tokens := strings.Fields(input)
	for start := 1; start < len(tokens); start++ {
		left := strings.Join(tokens[:start], " ")
		right := strings.Join(tokens[start:], " ")
		if m, matched := Resolve(s, right, KindRoomContainer, KindCorpse, KindPet); matched {
			return left, m, true
		}
	}
	return "", Match{}, false
}

// ResolveItem is the shared get/drop/look-item ladder. It resolves an item that
// may live in a trailing container / corpse ("get X from Y" or "get X Y"), or on
// the floor / in inventory. When the item comes from a container-like source,
// the returned Match carries that source (Kind + ContainerName / CorpseIdx) plus
// the item, so the command can still apply its own gates.
func ResolveItem(s Scope, input string) (Match, bool) {
	if itemPart, cm, ok := SplitTrailingContainer(s, input); ok {
		if m, ok2 := lootFromContainer(s, cm, itemPart); ok2 {
			return m, true
		}
		// A trailing container matched but held no such item — fall through so a
		// bare floor/inventory item of that literal name can still resolve.
	}
	return Resolve(s, input, KindFloorItem, KindInventoryItem)
}

// lootFromContainer resolves itemName inside the container/corpse identified by
// cm and returns a Match that carries both the item and the source handle.
func lootFromContainer(s Scope, cm Match, itemName string) (Match, bool) {
	switch cm.Kind {
	case KindRoomContainer:
		container := s.Room.Containers[cm.ContainerName]
		it, ok := container.FindItem(itemName)
		if !ok {
			return Match{}, false
		}
		return Match{Kind: KindRoomContainer, Name: it.Name(), Item: it, ContainerName: cm.ContainerName}, true
	case KindCorpse:
		it, ok := s.Room.Corpses[cm.CorpseIdx].Loot.FindItem(itemName)
		if !ok {
			return Match{}, false
		}
		return Match{Kind: KindCorpse, Name: it.Name(), Item: it, CorpseIdx: cm.CorpseIdx}, true
	}
	return Match{}, false
}

// ResolveActor resolves a mob / player / pet target from input.
func ResolveActor(s Scope, input string, kinds ...Kind) (Match, bool) {
	return Resolve(s, input, kinds...)
}

// SplitLeadingMatch finds the longest leading token span of input for which
// matches() returns true, and returns that span (head) plus the remaining tail.
// It is scope-agnostic — the caller injects the validator — so it serves
// global-scoped commands (e.g. admin "<mob-template-name> <player> [value]")
// that the room-scoped adapters don't fit. ok=false when no leading span
// matches.
func SplitLeadingMatch(input string, matches func(candidate string) bool) (head, tail string, ok bool) {
	tokens := strings.Fields(input)
	for headLen := len(tokens); headLen >= 1; headLen-- {
		candidate := strings.Join(tokens[:headLen], " ")
		if matches(candidate) {
			return candidate, strings.Join(tokens[headLen:], " "), true
		}
	}
	return "", "", false
}
