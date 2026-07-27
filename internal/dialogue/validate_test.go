package dialogue

import (
	"strings"
	"testing"
)

func permissiveValidators() DialogueValidators {
	return DialogueValidators{
		QuestExists:   func(string) bool { return true },
		QuestEndToken: func(tok string) (string, bool) { return "10-end", true },
		FlagDeclared:  func(string, string) bool { return true },
		ItemExists:    func(int) bool { return true },
	}
}

func errsContaining(t *testing.T, errs []string, want string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, want) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got %v", want, errs)
}

// Re-grant prevention: grantsQuest requires the granted token AND the quest's
// end token in questExcluded — otherwise a player who finished the quest gets
// it re-offered.
func TestValidate_GrantRequiresEndTokenExclusion(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{Nodes: []TreeNode{{
		Id: "offer", Triggers: []string{"quest", "task", "help"},
		Text: "I need a hand.", GrantsQuest: "10-start",
		QuestExcluded: []string{"10-start"}, // missing 10-end
	}}}}
	errs, _ := ValidateDialogueFile(df, permissiveValidators())
	errsContaining(t, errs, "10-end")

	df.Tree.Nodes[0].QuestExcluded = []string{"10-start", "10-end"}
	errs, _ = ValidateDialogueFile(df, permissiveValidators())
	if len(errs) != 0 {
		t.Errorf("complete exclusions should pass, got %v", errs)
	}
}

// Quest discovery: a granting node/pattern must answer `ask <npc> quest`.
func TestValidate_GrantRequiresQuestTaskTriggers(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{Nodes: []TreeNode{{
		Id: "offer", Triggers: []string{"help"}, Text: "T.",
		GrantsQuest: "10-start", QuestExcluded: []string{"10-start", "10-end"},
	}}}}
	errs, _ := ValidateDialogueFile(df, permissiveValidators())
	errsContaining(t, errs, "quest")

	df2 := DialogueFile{MobId: 1, Zone: "z", Patterns: []Pattern{{
		Keywords: []string{"work"}, Responses: []string{"R."},
		GrantsQuest: "10-start", QuestExcluded: []string{"10-start", "10-end"},
	}}}
	errs, _ = ValidateDialogueFile(df2, permissiveValidators())
	errsContaining(t, errs, "quest")
}

// Matching walks tree.nodes in file order; a plain node before a grant node
// shadows it.
func TestValidate_GrantNodesMustComeFirst(t *testing.T) {
	grant := TreeNode{Id: "offer", Triggers: []string{"quest", "task"},
		Text: "T.", GrantsQuest: "10-start", QuestExcluded: []string{"10-start", "10-end"}}
	lore := TreeNode{Id: "lore", Triggers: []string{"history"}, Text: "L."}

	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{Nodes: []TreeNode{lore, grant}}}
	errs, _ := ValidateDialogueFile(df, permissiveValidators())
	errsContaining(t, errs, "FIRST")

	df.Tree.Nodes = []TreeNode{grant, lore}
	errs, _ = ValidateDialogueFile(df, permissiveValidators())
	if len(errs) != 0 {
		t.Errorf("grant-first ordering should pass, got %v", errs)
	}
}

// Semicolons are the command separator — in spoken text they truncate.
func TestValidate_NoSemicolons(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z",
		Greetings: []Greeting{{Text: "Hello; friend."}},
		Patterns:  []Pattern{{Keywords: []string{"x"}, Responses: []string{"Fine; thanks."}}},
		Tree: &Tree{Root: TreeRoot{Text: "Root; text.", Hints: "Hint; here."},
			Nodes: []TreeNode{{Id: "n", Triggers: []string{"t"}, Text: "Node; text."}}}}
	errs, _ := ValidateDialogueFile(df, permissiveValidators())
	if len(errs) < 5 {
		t.Errorf("expected a semicolon error per offending field, got %v", errs)
	}
}

// Unknown quest tokens, undeclared flags, unknown items: refused via the
// injected checks.
func TestValidate_UnknownReferences(t *testing.T) {
	v := DialogueValidators{
		QuestExists:   func(string) bool { return false },
		QuestEndToken: func(string) (string, bool) { return "", false },
		FlagDeclared:  func(string, string) bool { return false },
		ItemExists:    func(int) bool { return false },
	}
	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{Nodes: []TreeNode{{
		Id: "n", Triggers: []string{"t"}, Text: "T.",
		QuestRequired:     []string{"99-start"},
		QuestFlagRequired: map[string]string{"11-branch": "rhett"},
		GivesItem:         40001,
	}}}}
	errs, _ := ValidateDialogueFile(df, v)
	errsContaining(t, errs, "99-start")
	errsContaining(t, errs, "11-branch")
	errsContaining(t, errs, "40001")
}

// requires/unlocks must reference node ids that exist in THIS file.
func TestValidate_NodeRefsResolve(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{Nodes: []TreeNode{
		{Id: "a", Triggers: []string{"t"}, Text: "A.", Unlocks: []string{"ghost"}},
	}}}
	errs, _ := ValidateDialogueFile(df, permissiveValidators())
	errsContaining(t, errs, "ghost")
}
