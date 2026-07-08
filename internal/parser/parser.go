// Package parser is DOGMud's shared command target-resolution seam. Commands
// declare which Kinds of target they want; the package tokenizes the input,
// tries the longest multi-word span first, and dispatches each candidate span
// to a per-Kind adapter that wraps an existing resolver. See
// docs/superpowers/specs/2026-07-08-unified-parser-seam-design.md.
package parser

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

type Kind int

const (
	KindMob Kind = iota
	KindPlayer
	KindPet
	KindFloorItem
	KindInventoryItem // backpack + equipped, via character.FindItem
	KindComponentItem // component bag
	KindPotionItem    // bandolier
	KindRoomContainer
	KindCorpse
	KindNoun
	KindExit
)

// Scope is the search context. Room is required; User is required for
// inventory/component/potion kinds and ownership-sensitive lookups.
type Scope struct {
	User *users.UserRecord
	Room *rooms.Room
}

// Match is the typed result. Only the fields relevant to Kind are populated.
type Match struct {
	Kind          Kind
	Name          string     // canonical resolved name (for messaging)
	Item          items.Item // item-ish kinds
	Source        string     // free-form source note ("in your backpack", "wielded")
	MobInstanceId int        // KindMob
	UserId        int        // KindPlayer / KindPet
	ContainerName string     // KindRoomContainer
	CorpseIdx     int        // KindCorpse
	Leftover      string     // unconsumed tokens (used by later stages, e.g. recipes)
}

// adapter resolves a single candidate span to a Match. ok=false means no match.
type adapter func(s Scope, candidate string) (Match, bool)

// resolveWith runs the greedy longest-span algorithm: it tries the full token
// span first, then drops trailing tokens one at a time. At each span length it
// tries the adapters in order; the first hit wins. This makes a 2-word match
// ("bank clerk") beat a 1-word match ("bank"), and adapter order breaks ties
// within one span length.
func resolveWith(tokens []string, s Scope, adapters []adapter) (Match, bool) {
	for l := len(tokens); l >= 1; l-- {
		candidate := strings.Join(tokens[:l], " ")
		for _, a := range adapters {
			if m, ok := a(s, candidate); ok {
				return m, true
			}
		}
	}
	return Match{}, false
}

// tokenize splits raw command input, honoring quotes and lower-casing so
// matching is case-insensitive.
func tokenize(input string) []string {
	return util.SplitButRespectQuotes(strings.ToLower(strings.TrimSpace(input)))
}
