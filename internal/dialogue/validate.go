package dialogue

import (
	"fmt"
	"strings"
)

// DialogueValidators injects the registry checks so validation is testable
// with no loaded world — and so modules/gmcp can wire them from its own deps
// (its unit tests run with empty registries; the spawn editor learned this
// the hard way).
type DialogueValidators struct {
	QuestExists   func(token string) bool
	QuestEndToken func(token string) (string, bool) // end token of the quest this token belongs to
	FlagDeclared  func(key, value string) bool
	ItemExists    func(id int) bool
}

// gateFields is the quest-gate family shared by patterns, tree nodes and root
// variants, flattened for validation.
type gateFields struct {
	where         string // "pattern 2", "node offer", "root variant 1"
	grantsQuest   string
	questRequired []string
	questExcluded []string
	flagRequired  map[string]string
	flagExcluded  map[string]string
	setsFlag      *QuestFlagSet
	requiresItem  int
	givesItem     int
	askWords      []string // keywords or triggers; nil = not applicable (root variants)
}

// ValidateDialogueFile applies every mechanical dialogue SOP. Errors block a
// save; warnings accompany a successful one.
func ValidateDialogueFile(df DialogueFile, v DialogueValidators) (errs []string, warns []string) {
	noSemi := func(where, text string) {
		if strings.Contains(text, ";") {
			errs = append(errs, fmt.Sprintf("%s: semicolons are the command separator and truncate spoken text — rewrite without ';'", where))
		}
	}

	gates := []gateFields{}
	for i, p := range df.Patterns {
		w := fmt.Sprintf("pattern %d (%s)", i+1, strings.Join(p.Keywords, ","))
		for _, r := range p.Responses {
			noSemi(w, r)
		}
		gates = append(gates, gateFields{where: w, grantsQuest: p.GrantsQuest,
			questRequired: p.QuestRequired, questExcluded: p.QuestExcluded,
			flagRequired: p.QuestFlagRequired, flagExcluded: p.QuestFlagExcluded,
			setsFlag: p.SetsQuestFlag, requiresItem: p.RequiresItem, givesItem: p.GivesItem,
			askWords: p.Keywords})
	}
	if df.Tree != nil {
		noSemi("tree root text", df.Tree.Root.Text)
		noSemi("tree root hints", df.Tree.Root.Hints)
		for i, rv := range df.Tree.Root.Variants {
			w := fmt.Sprintf("root variant %d", i+1)
			noSemi(w+" text", rv.Text)
			noSemi(w+" hints", rv.Hints)
			gates = append(gates, gateFields{where: w, grantsQuest: rv.GrantsQuest,
				questRequired: rv.QuestRequired, questExcluded: rv.QuestExcluded,
				flagRequired: rv.QuestFlagRequired, flagExcluded: rv.QuestFlagExcluded,
				setsFlag: rv.SetsQuestFlag, requiresItem: rv.RequiresItem, givesItem: rv.GivesItem})
		}

		ids := map[string]bool{}
		for _, n := range df.Tree.Nodes {
			ids[n.Id] = true
		}
		grantSeenAfterPlain := false
		plainSeen := false
		for _, n := range df.Tree.Nodes {
			w := fmt.Sprintf("node %q", n.Id)
			noSemi(w+" text", n.Text)
			noSemi(w+" hints", n.Hints)
			if n.GrantsQuest != "" && plainSeen {
				grantSeenAfterPlain = true
			}
			if n.GrantsQuest == "" {
				plainSeen = true
			}
			for _, ref := range append(append([]string{}, n.Requires...), n.Unlocks...) {
				if !ids[ref] {
					errs = append(errs, fmt.Sprintf("%s: requires/unlocks references %q, which is not a node id in this file", w, ref))
				}
			}
			gates = append(gates, gateFields{where: w, grantsQuest: n.GrantsQuest,
				questRequired: n.QuestRequired, questExcluded: n.QuestExcluded,
				flagRequired: n.QuestFlagRequired, flagExcluded: n.QuestFlagExcluded,
				setsFlag: n.SetsQuestFlag, requiresItem: n.RequiresItem, givesItem: n.GivesItem,
				askWords: n.Triggers})
			if n.GrantsQuest != "" && len(n.Requires) > 0 {
				warns = append(warns, fmt.Sprintf("%s: grant node gated by requires — prefer questRequired; per-player memory can expire and brick the quest", w))
			}
		}
		if grantSeenAfterPlain {
			errs = append(errs, "tree.nodes: quest-grant nodes must come FIRST — matching walks nodes in file order, so an earlier plain node can shadow a later gated grant")
		}
	}
	for i, g := range df.Greetings {
		noSemi(fmt.Sprintf("greeting %d", i+1), g.Text)
	}

	for _, g := range gates {
		if g.grantsQuest != "" {
			has := func(tok string) bool {
				for _, x := range g.questExcluded {
					if x == tok {
						return true
					}
				}
				return false
			}
			if !has(g.grantsQuest) {
				errs = append(errs, fmt.Sprintf("%s: grantsQuest %q must also appear in questExcluded, or the offer repeats forever", g.where, g.grantsQuest))
			}
			if end, ok := v.QuestEndToken(g.grantsQuest); ok {
				if !has(end) {
					errs = append(errs, fmt.Sprintf("%s: questExcluded must include the end token %q, or a player who FINISHED the quest gets it re-offered", g.where, end))
				}
			}
			if g.askWords != nil {
				lower := map[string]bool{}
				for _, wd := range g.askWords {
					lower[strings.ToLower(wd)] = true
				}
				if !lower["quest"] || !lower["task"] {
					errs = append(errs, fmt.Sprintf("%s: quest-granting entries must include \"quest\" and \"task\" in their triggers/keywords so `ask <npc> quest` works", g.where))
				}
			}
		}
		for _, tok := range append(append([]string{g.grantsQuest}, g.questRequired...), g.questExcluded...) {
			if tok != "" && !v.QuestExists(tok) {
				errs = append(errs, fmt.Sprintf("%s: quest token %q does not exist", g.where, tok))
			}
		}
		checkFlags := func(m map[string]string) {
			for k, val := range m {
				if !v.FlagDeclared(k, val) {
					errs = append(errs, fmt.Sprintf("%s: quest flag %q=%q is not declared by any quest — undeclared flags panic at boot", g.where, k, val))
				}
			}
		}
		checkFlags(g.flagRequired)
		checkFlags(g.flagExcluded)
		if g.setsFlag != nil && !v.FlagDeclared(g.setsFlag.Key, g.setsFlag.Value) {
			errs = append(errs, fmt.Sprintf("%s: setsQuestFlag %q=%q is not declared by any quest", g.where, g.setsFlag.Key, g.setsFlag.Value))
		}
		for _, id := range []int{g.requiresItem, g.givesItem} {
			if id != 0 && !v.ItemExists(id) {
				errs = append(errs, fmt.Sprintf("%s: item %d does not exist", g.where, id))
			}
		}
	}

	return errs, warns
}
