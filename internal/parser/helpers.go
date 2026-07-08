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

// ResolveItem is the shared get/drop/look-item ladder. It resolves an item that
// may live in a trailing container / corpse ("get X from Y" or "get X Y"), or on
// the floor / in inventory. When the item comes from a container-like source,
// the returned Match carries that source (Kind + ContainerName / CorpseIdx) plus
// the item, so the command can still apply its own gates.
func ResolveItem(s Scope, input string) (Match, bool) {
	// Explicit "from <container>".
	if itemPart, containerPart, ok := splitOnConnective(input, "from"); ok {
		if cm, ok2 := Resolve(s, containerPart, KindRoomContainer, KindCorpse); ok2 {
			return lootFromContainer(s, cm, itemPart)
		}
		return Match{}, false
	}

	// No "from": try each "<item> <container>" split, from the longest item /
	// shortest container span down. Accept the first split where the container
	// resolves AND the item resolves inside it — validating the item avoids
	// mis-stripping when the typed word differs from the container's canonical
	// name (e.g. "sword corpse" vs. canonical "Skeleton corpse").
	tokens := strings.Fields(input)
	for start := 1; start < len(tokens); start++ {
		itemPart := strings.Join(tokens[:start], " ")
		containerPart := strings.Join(tokens[start:], " ")
		if cm, ok := Resolve(s, containerPart, KindRoomContainer, KindCorpse); ok {
			if m, ok2 := lootFromContainer(s, cm, itemPart); ok2 {
				return m, true
			}
		}
	}

	// No container: resolve the item from floor / inventory.
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
