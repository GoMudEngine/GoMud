# Dialogue Editor (5b) Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Full-parity dialogue authoring from `/build` — greetings, patterns, tree, memory — with every mechanical dialogue SOP enforced at save, launched from the mob editor.

**Architecture:** Pure validation (`ValidateDialogueFile` + injectable `DialogueValidators`) in the dialogue package; a dumb-but-correct writer that owns the loader's path/cache/sentinel contract; `Build.Dialogue.*` GMCP behind a `dialogueDeps` seam; a dedicated panel reached from the mob form. Validation lives at the handler with injected existence checks — the spawn-editor lesson: gmcp unit tests run with no world loaded, so the real registries must never be hardwired into the validated path.

**Tech Stack:** Go, yaml.v2, in-package dialogue tests (plain `testing`, testify where already used), vanilla JS panel.

**Spec:** `docs/superpowers/specs/2026-07-27-dialogue-editor-5b-design.md`

---

## Verified reference (do not re-derive)

```go
// loader.go — the contract the writer must honour
dialogue.Load(mobId int, zone string) *DialogueFile        // :21
key := fmt.Sprintf("%d:%s", mobId, zone)                   // cache + sentinel key
dialogueCache map[string]*DialogueFile                     // save must REPLACE entry
nilSentinel   map[string]bool                              // create must CLEAR; delete must SET
path = datafiles + `/dialogue/` + zoneNameSanitize(zone) + `/` + mobId + `.yaml`
// zoneNameSanitize is PRIVATE to the dialogue package — the writer calls it, never a copy.

// types.go
DialogueFile{MobId, Zone, DefaultMood string, Greetings []Greeting, Patterns []Pattern, Tree *Tree, Memory MemoryConfig}
Pattern{Keywords, Moods, Responses []string, MoodChange string, + quest-gate family}
Tree{Root TreeRoot, Nodes []TreeNode}
TreeRoot{Text, Hints string, Variants []QuestGreeting}
TreeNode{Id string, Triggers, Requires, Unlocks []string, Text, Hints string, + quest-gate family}
QuestGreeting{Text, Hints + quest-gate family}      // root variants
// quest-gate family: QuestRequired/QuestExcluded []string, GrantsQuest string,
//   RequiresItem/GivesItem int, QuestFlagRequired/QuestFlagExcluded map[string]string,
//   SetsQuestFlag *QuestFlagSet{Key,Value}, BumpsRep []RepBump{Faction,Delta},
//   GivesGold int, MasterworkRequired int
MemoryConfig{ExpiryPeriod string}
Mood constants: friendly, neutral, hostile, afraid, grateful   // types.go:36

// registries for pickers / live validators
quests.GetQuest(questToken string) *Quest       // nil = unknown token; Quest.QuestId int
quests.GetAllQuests() []Quest                   // Quest.Flags []QuestFlagDef{Key, Values, Description}
items.GetItemSpec(id int) *ItemSpec             // nil = unknown
// end-token convention (CLAUDE.md): fmt.Sprintf("%d-end", quest.QuestId)

// gmcp precedents
BuildResult (gmcp.Build.go:35) — gains Warnings []string
mobDeps / realMobDeps + handleBuildOp admin gate + GMCPBuildOp routing (gmcp.go ~:486)
mobs.js Advanced-drawer + list-row patterns; spawn editor's login-time Build.Item.List prefetch
```

**SOP sources being mechanized** (CLAUDE.md): quest re-grant prevention;
quest NPC dialogue (`quest`/`task` triggers); grant nodes FIRST (file-order
matching); no semicolons in text/hints; dialogue voice + trigger
discoverability; `expiryPeriod` almost never; prefer `questRequired` over
`requires` for quest gating.

---

## File Structure

| File | Responsibility |
|------|----------------|
| `internal/dialogue/validate.go` **(create)** | `DialogueValidators`, `ValidateDialogueFile` (errors + warnings) |
| `internal/dialogue/validate_test.go` **(create)** | One test per rule |
| `internal/dialogue/save.go` **(create)** | `SaveDialogueFile`, `CreateNewDialogueFile`, `DeleteDialogueFile` |
| `internal/dialogue/save_test.go` **(create)** | Cache/sentinel/path semantics + 302-file round-trip |
| `modules/gmcp/gmcp.Dialogue.go` **(create)** | `dialogueDeps`, payloads, handlers, enums |
| `modules/gmcp/gmcp.Dialogue_test.go` **(create)** | Handler tests vs fakes |
| `modules/gmcp/gmcp.Build.go` **(modify)** | `Warnings` on BuildResult + dispatch cases |
| `modules/gmcp/gmcp.go` **(modify)** | Route the verbs |
| `modules/gmcp/gmcp.Mob.go` **(modify)** | `HasDialogue` on mob detail |
| `_datafiles/html/public/static/js/dialogue.js` **(create)** | The panel |
| `_datafiles/html/public/static/js/mobs.js` **(modify)** | Dialogue… button |
| `_datafiles/html/public/build.html` **(modify)** | Script tag + GMCP routing |

---

### Task 1: Validation — the refusal rules

**Files:** Create `internal/dialogue/validate.go`, `internal/dialogue/validate_test.go`

- [ ] **Step 1: Write the failing tests**

```go
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
	errsContaining(t, errs, "first")

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
```

- [ ] **Step 2: Run to verify RED**

Run: `go test ./internal/dialogue/ -run TestValidate_ -count=1 2>&1 | head -4`
Expected: FAIL — `undefined: DialogueValidators`

- [ ] **Step 3: Implement**

Create `internal/dialogue/validate.go`:

```go
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
		for i, n := range df.Tree.Nodes {
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
			_ = i
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
```

- [ ] **Step 4: Run to verify GREEN**

Run: `go test ./internal/dialogue/ -run TestValidate_ -count=1 -v 2>&1 | tail -8`
Expected: PASS (all six)

- [ ] **Step 5: Commit**

```bash
git add internal/dialogue/validate.go internal/dialogue/validate_test.go
git commit -m "feat(dialogue): save-time SOP validation (refusal rules)"
```

---

### Task 2: Validation — the warnings

**Files:** Modify `internal/dialogue/validate.go`, `internal/dialogue/validate_test.go`

- [ ] **Step 1: Write the failing tests**

```go
func warnsContaining(t *testing.T, warns []string, want string) {
	t.Helper()
	for _, w := range warns {
		if strings.Contains(w, want) {
			return
		}
	}
	t.Errorf("expected a warning containing %q, got %v", want, warns)
}

// expiryPeriod is almost never right (SOP) — it can brick quest chains.
func TestValidate_WarnsOnExpiryPeriod(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z", Memory: MemoryConfig{ExpiryPeriod: "2 real days"}}
	_, warns := ValidateDialogueFile(df, permissiveValidators())
	warnsContaining(t, warns, "expiryPeriod")
}

// "Undiscoverable triggers are broken triggers": every tree trigger must
// appear somewhere a player can read — root text/hints or any node text/hints.
func TestValidate_WarnsOnUndiscoverableTrigger(t *testing.T) {
	df := DialogueFile{MobId: 1, Zone: "z", Tree: &Tree{
		Root: TreeRoot{Text: "Ask me about the harvest.", Hints: "You could ask about the harvest."},
		Nodes: []TreeNode{
			{Id: "harvest", Triggers: []string{"harvest"}, Text: "It was thin this year."},
			{Id: "secret", Triggers: []string{"smuggling"}, Text: "Keep your voice down."},
		}}}
	_, warns := ValidateDialogueFile(df, permissiveValidators())
	warnsContaining(t, warns, "smuggling")
	for _, w := range warns {
		if strings.Contains(w, "\"harvest\"") {
			t.Errorf("harvest IS discoverable (root text) — must not warn: %v", w)
		}
	}
}
```

- [ ] **Step 2: RED**

Run: `go test ./internal/dialogue/ -run "TestValidate_Warns" -count=1 2>&1 | tail -4`
Expected: FAIL (warnings not yet produced)

- [ ] **Step 3: Implement**

In `ValidateDialogueFile`, add after the gate loop:

```go
	if df.Memory.ExpiryPeriod != "" {
		warns = append(warns, fmt.Sprintf("memory.expiryPeriod is set (%q): the SOP is to leave it empty except for deliberately timed quests — expiring memory can brick quest chains", df.Memory.ExpiryPeriod))
	}

	// Discoverability: a trigger no hint, node text, or root ever mentions is
	// a dead branch no player will find. Case-insensitive substring over all
	// player-readable prose in the file.
	if df.Tree != nil {
		var prose strings.Builder
		prose.WriteString(strings.ToLower(df.Tree.Root.Text))
		prose.WriteString(" " + strings.ToLower(df.Tree.Root.Hints))
		for _, rv := range df.Tree.Root.Variants {
			prose.WriteString(" " + strings.ToLower(rv.Text) + " " + strings.ToLower(rv.Hints))
		}
		for _, n := range df.Tree.Nodes {
			prose.WriteString(" " + strings.ToLower(n.Text) + " " + strings.ToLower(n.Hints))
		}
		haystack := prose.String()
		for _, n := range df.Tree.Nodes {
			for _, trig := range n.Triggers {
				lt := strings.ToLower(trig)
				if lt == "quest" || lt == "task" {
					continue // universal ask-words, discoverable by convention
				}
				if !strings.Contains(haystack, lt) {
					warns = append(warns, fmt.Sprintf("node %q: trigger %q appears in no hint, text, or root — undiscoverable triggers are broken triggers", n.Id, trig))
				}
			}
		}
	}
```

- [ ] **Step 4: GREEN + full package + commit**

Run: `go test ./internal/dialogue/ -count=1 2>&1 | tail -2`
Expected: PASS.

```bash
git add internal/dialogue/validate.go internal/dialogue/validate_test.go
git commit -m "feat(dialogue): validation warnings — expiryPeriod + undiscoverable triggers"
```

---

### Task 3: The writer — save / create / delete

**Files:** Create `internal/dialogue/save.go`, `internal/dialogue/save_test.go`

- [ ] **Step 1: Write the failing tests**

In-package tests may touch `dialogueCache` and `nilSentinel` directly.

```go
package dialogue

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// The writer owns the loader's contract: path from the loader's own
// sanitizer, cache REPLACED on save, sentinel CLEARED on create and SET on
// delete. The sentinel rule is the one that burns people: Load caches
// "no dialogue forever" per (mob,zone), so a create that forgets to clear it
// ships a file the running server can never see.
func TestWriter_CacheAndSentinelContract(t *testing.T) {
	dir := t.TempDir()
	restore := overrideDataFilesDir(t, dir)
	defer restore()

	const mobId, zone = 424242, "Writer Probe Zone"
	key := fmt.Sprintf("%d:%s", mobId, zone)
	delete(dialogueCache, key)
	delete(nilSentinel, key)
	t.Cleanup(func() { delete(dialogueCache, key); delete(nilSentinel, key) })

	// Simulate the burn: Load before the file exists → sentinel set.
	if Load(mobId, zone) != nil {
		t.Fatal("no file yet — Load must return nil")
	}
	if !nilSentinel[key] {
		t.Fatal("Load must set the nil sentinel (loader contract)")
	}

	if err := CreateNewDialogueFile(mobId, zone); err != nil {
		t.Fatalf("create: %v", err)
	}
	if nilSentinel[key] {
		t.Error("create must CLEAR the nil sentinel or the new file is invisible until reboot")
	}
	if df := Load(mobId, zone); df == nil {
		t.Fatal("Load must now see the created file")
	}

	// Save replaces the cache entry in place.
	df := *Load(mobId, zone)
	df.DefaultMood = "hostile"
	if err := SaveDialogueFile(df); err != nil {
		t.Fatalf("save: %v", err)
	}
	if got := Load(mobId, zone); got.DefaultMood != "hostile" {
		t.Errorf("live cache must serve the edit immediately, got mood %q", got.DefaultMood)
	}

	// The file landed at the loader's exact path.
	p := filepath.Join(dir, "dialogue", zoneNameSanitize(zone), fmt.Sprintf("%d.yaml", mobId))
	if _, err := os.Stat(p); err != nil {
		t.Errorf("file not at the loader's path: %v", err)
	}

	// Delete removes the file, drops the cache, and SETS the sentinel.
	if err := DeleteDialogueFile(mobId, zone); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if _, ok := dialogueCache[key]; ok {
		t.Error("delete must drop the cache entry")
	}
	if !nilSentinel[key] {
		t.Error("delete must set the sentinel — the mob genuinely has no dialogue now")
	}
	if Load(mobId, zone) != nil {
		t.Error("Load after delete must return nil")
	}
}

func TestWriter_CreateRefusesWhenFileExists(t *testing.T) {
	dir := t.TempDir()
	restore := overrideDataFilesDir(t, dir)
	defer restore()
	const mobId, zone = 424243, "Writer Probe Zone"
	key := fmt.Sprintf("%d:%s", mobId, zone)
	t.Cleanup(func() { delete(dialogueCache, key); delete(nilSentinel, key) })

	if err := CreateNewDialogueFile(mobId, zone); err != nil {
		t.Fatal(err)
	}
	if err := CreateNewDialogueFile(mobId, zone); err == nil {
		t.Error("second create must refuse — the file exists")
	}
}
```

`overrideDataFilesDir` is a small test helper using the codebase's supported
mechanism (verified in `internal/web/auth_test.go:104-126`): chdir to repo
root (ReloadConfig reads `_datafiles/config.yaml` from CWD), write a
`config-overrides.yaml` into the TempDir containing
`FilePaths:
  DataFiles: <tempdir-slash>
  CarefulSaveFiles: false`,
`t.Setenv("CONFIG_PATH", overridePath)`, `configs.ReloadConfig()`, and a
cleanup that chdirs back and reloads without the override. Write it once at
the top of `save_test.go`.

- [ ] **Step 2: RED**

Run: `go test ./internal/dialogue/ -run TestWriter_ -count=1 2>&1 | head -4`
Expected: FAIL — `undefined: CreateNewDialogueFile`

- [ ] **Step 3: Implement `internal/dialogue/save.go`**

```go
package dialogue

// The dialogue writer (5b). No writer existed before this; the loader's
// contract it must honour is documented at each step. Validation is NOT
// called here — the GMCP handler validates with injected registries first
// (the spawn-editor lesson: hardwiring real registries into the save path
// makes it untestable without a loaded world).

import (
	"errors"
	"fmt"
	"os"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/util"
	yaml "gopkg.in/yaml.v2"
)

func dialogueFilePath(mobId int, zone string) string {
	dataFiles := string(configs.GetFilePathsConfig().DataFiles)
	return util.FilePath(dataFiles + `/dialogue/` + zoneNameSanitize(zone) + `/` + fmt.Sprintf("%d", mobId) + `.yaml`)
}

// SaveDialogueFile writes the file at the loader's exact path and replaces
// the cache entry in place so live NPCs serve the edit immediately.
func SaveDialogueFile(df DialogueFile) error {
	if df.MobId == 0 || df.Zone == "" {
		return errors.New("dialogue file needs a mob id and zone")
	}
	data, err := yaml.Marshal(&df)
	if err != nil {
		return err
	}
	path := dialogueFilePath(df.MobId, df.Zone)
	if err := os.MkdirAll(filepathDir(path), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%s", df.MobId, df.Zone)
	cp := df
	dialogueCache[key] = &cp
	delete(nilSentinel, key) // a save proves the file exists
	return nil
}

// CreateNewDialogueFile writes a minimal skeleton and clears the nil
// sentinel — without that, a mob whose Load already ran stays "dialogueless"
// until reboot no matter what is on disk.
func CreateNewDialogueFile(mobId int, zone string) error {
	if _, err := os.Stat(dialogueFilePath(mobId, zone)); err == nil {
		return fmt.Errorf("mob %d already has a dialogue file", mobId)
	}
	df := DialogueFile{
		MobId: mobId, Zone: zone, DefaultMood: string(MoodNeutral),
		Patterns: []Pattern{{Keywords: []string{""}, Responses: []string{"..."}}},
	}
	return SaveDialogueFile(df)
}

// DeleteDialogueFile removes the file, drops the cache entry, and SETS the
// sentinel: the mob now genuinely has no dialogue.
func DeleteDialogueFile(mobId int, zone string) error {
	path := dialogueFilePath(mobId, zone)
	if err := os.Remove(path); err != nil {
		return err
	}
	key := fmt.Sprintf("%d:%s", mobId, zone)
	delete(dialogueCache, key)
	nilSentinel[key] = true
	return nil
}
```

Use `filepath.Dir(path)` (import `path/filepath`) where the sketch says
`filepathDir`. `string(MoodNeutral)` casts the Mood constant for the string
field.

- [ ] **Step 4: GREEN + commit**

Run: `go test ./internal/dialogue/ -run TestWriter_ -count=1 -v 2>&1 | tail -4 && go build ./...`
Expected: PASS, build clean.

```bash
git add internal/dialogue/save.go internal/dialogue/save_test.go
git commit -m "feat(dialogue): writer with cache + nil-sentinel contract"
```

---

### Task 4: Lossless round-trip over all 302 live files

**Files:** Modify `internal/dialogue/save_test.go`

- [ ] **Step 1: Write the test**

Marshal-level round trip (bytes → struct → marshal → struct → DeepEqual):
this is the substance of "the writer is lossless" without writing into the
real tree. Refinement over the spec's "temp mirror" wording — same proof,
no filesystem risk.

```go
func TestWriter_RoundTripsEveryLiveFile(t *testing.T) {
	root := filepath.Join("..", "..", "_datafiles", "world", "dogmud", "dialogue")
	checked := 0
	_ = filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		raw, _ := os.ReadFile(path)
		var orig DialogueFile
		if yaml.Unmarshal(raw, &orig) != nil {
			return nil // the parse sweep owns malformed files
		}
		out, err := yaml.Marshal(&orig)
		if err != nil {
			t.Errorf("%s: marshal: %v", path, err)
			return nil
		}
		var back DialogueFile
		if err := yaml.Unmarshal(out, &back); err != nil {
			t.Errorf("%s: re-unmarshal: %v", path, err)
			return nil
		}
		if !reflect.DeepEqual(orig, back) {
			t.Errorf("%s: NOT LOSSLESS through the writer's marshal", path)
		}
		checked++
		return nil
	})
	if checked < 300 {
		t.Errorf("round-tripped only %d files — wrong path?", checked)
	}
	t.Logf("round-tripped %d live dialogue files losslessly", checked)
}
```

Add imports (`reflect`, `strings`, `path/filepath`, yaml) as needed.

- [ ] **Step 2: Run — expected PASS; any failure is a real finding**

Run: `go test ./internal/dialogue/ -run TestWriter_RoundTripsEveryLiveFile -count=1 -v 2>&1 | tail -3`
If any file is not lossless, STOP and report the file — that is a struct
shape gap (the greetings incident again), not a test to weaken.

- [ ] **Step 3: Commit**

```bash
git add internal/dialogue/save_test.go
git commit -m "test(dialogue): lossless round-trip over all live dialogue files"
```

---

### Task 5: GMCP — deps, payloads, handlers

**Files:** Create `modules/gmcp/gmcp.Dialogue.go`, `modules/gmcp/gmcp.Dialogue_test.go`; modify `modules/gmcp/gmcp.Build.go` (BuildResult)

- [ ] **Step 1: Add `Warnings` to BuildResult** (gmcp.Build.go:35 family):

```go
	Warnings []string `json:"warnings,omitempty"` // non-blocking validation notes (Build.Dialogue.Update first)
```

- [ ] **Step 2: Write the failing handler tests**

```go
package gmcp

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/dialogue"
)

type fakeDialogueWorld struct {
	files   map[string]*dialogue.DialogueFile
	saved   []dialogue.DialogueFile
	created []string
	deleted []string
}

func newFakeDialogueWorld() *fakeDialogueWorld {
	return &fakeDialogueWorld{files: map[string]*dialogue.DialogueFile{
		"9517:Greenford": {MobId: 9517, Zone: "Greenford", DefaultMood: "friendly"},
	}}
}

func dlgKey(mobId int, zone string) string { return fmt.Sprintf("%d:%s", mobId, zone) }

func (w *fakeDialogueWorld) deps() dialogueDeps {
	return dialogueDeps{
		load: func(mobId int, zone string) *dialogue.DialogueFile { return w.files[dlgKey(mobId, zone)] },
		save: func(df dialogue.DialogueFile) error {
			w.saved = append(w.saved, df)
			cp := df
			w.files[dlgKey(df.MobId, df.Zone)] = &cp
			return nil
		},
		create: func(mobId int, zone string) error {
			w.created = append(w.created, dlgKey(mobId, zone))
			w.files[dlgKey(mobId, zone)] = &dialogue.DialogueFile{MobId: mobId, Zone: zone}
			return nil
		},
		del: func(mobId int, zone string) error {
			w.deleted = append(w.deleted, dlgKey(mobId, zone))
			delete(w.files, dlgKey(mobId, zone))
			return nil
		},
		validators: dialogue.DialogueValidators{
			QuestExists:   func(string) bool { return true },
			QuestEndToken: func(string) (string, bool) { return "10-end", true },
			FlagDeclared:  func(string, string) bool { return true },
			ItemExists:    func(int) bool { return true },
		},
	}
}

func TestBuildDialogueUpdate_RefusesInvalidAndSavesNothing(t *testing.T) {
	w := newFakeDialogueWorld()
	df := dialogue.DialogueFile{MobId: 9517, Zone: "Greenford",
		Greetings: []dialogue.Greeting{{Text: "Hello; there."}}} // semicolon
	res := buildDialogueUpdate(w.deps(), df)
	if res.Ok {
		t.Error("invalid dialogue must be refused")
	}
	if len(w.saved) != 0 {
		t.Error("nothing may be saved when validation fails")
	}
}

func TestBuildDialogueUpdate_SavesAndSurfacesWarnings(t *testing.T) {
	w := newFakeDialogueWorld()
	df := dialogue.DialogueFile{MobId: 9517, Zone: "Greenford",
		Memory: dialogue.MemoryConfig{ExpiryPeriod: "2 real days"}} // warn, not error
	res := buildDialogueUpdate(w.deps(), df)
	if !res.Ok {
		t.Fatalf("warn-only file must save: %+v", res)
	}
	if len(res.Warnings) == 0 {
		t.Error("warnings must ride the successful result")
	}
	if len(w.saved) != 1 {
		t.Error("expected exactly one save")
	}
}

func TestBuildDialogueGetCreateDelete(t *testing.T) {
	w := newFakeDialogueWorld()
	if d, ok := buildDialogueGet(w.deps(), 9517, "Greenford"); !ok || d.File.DefaultMood != "friendly" {
		t.Errorf("get existing: ok=%v %+v", ok, d)
	}
	if _, ok := buildDialogueGet(w.deps(), 1, "Nowhere"); ok {
		t.Error("get missing file must report no-file (so the client offers Create)")
	}
	if res := buildDialogueCreate(w.deps(), 42, "Greenford"); !res.Ok || len(w.created) != 1 {
		t.Errorf("create: %+v", res)
	}
	if res := buildDialogueDelete(w.deps(), 9517, "Greenford"); !res.Ok || len(w.deleted) != 1 {
		t.Errorf("delete: %+v", res)
	}
}
```

Imports: `fmt`, `strings`, `testing`, and the dialogue package.

- [ ] **Step 3: RED, then implement `gmcp.Dialogue.go`**

`dialogueDeps{load, save, create, del, validators}`; `realDialogueDeps()`
wires `dialogue.Load/SaveDialogueFile/CreateNewDialogueFile/
DeleteDialogueFile` and live validators:

```go
		validators: dialogue.DialogueValidators{
			QuestExists: func(tok string) bool { return quests.GetQuest(tok) != nil },
			QuestEndToken: func(tok string) (string, bool) {
				if q := quests.GetQuest(tok); q != nil {
					return fmt.Sprintf("%d-end", q.QuestId), true
				}
				return "", false
			},
			FlagDeclared: func(key, value string) bool {
				for _, q := range quests.GetAllQuests() {
					for _, f := range q.Flags {
						if f.Key != key {
							continue
						}
						for _, v := range f.Values {
							if v == value {
								return true
							}
						}
					}
				}
				return false
			},
			ItemExists: func(id int) bool { return items.GetItemSpec(id) != nil },
		},
```

Handlers: `buildDialogueUpdate` validates (errors → `buildErr` joining them;
warnings → on the Ok result), then saves. `buildDialogueGet` returns
`{File, Found bool, Enums}` — enums carry quest tokens `{token, questName}`
built from `quests.GetAllQuests()`: one token per quest step as
`fmt.Sprintf("%d-%s", q.QuestId, step.Id)` — verified against
`quests.GetQuest`, which parses tokens via `TokenToParts` into
`(questId, stepId)` and checks the step id against `quest.Steps` — plus
per-quest flag declarations (`q.Flags`) and the mood list.

- [ ] **Step 4: GREEN + commit**

Run: `go test ./modules/gmcp/ -run TestBuildDialogue -count=1 -v 2>&1 | tail -6 && go build ./...`

```bash
git add modules/gmcp/gmcp.Dialogue.go modules/gmcp/gmcp.Dialogue_test.go modules/gmcp/gmcp.Build.go
git commit -m "feat(build): Build.Dialogue handlers behind dialogueDeps"
```

---

### Task 6: Routing, dispatch, and the mob-detail flag

**Files:** Modify `modules/gmcp/gmcp.go`, `modules/gmcp/gmcp.Build.go`, `modules/gmcp/gmcp.Mob.go`

- [ ] **Step 1:** Add to the routed-verb case list (gmcp.go ~:486):
`` `Build.Dialogue.Get`, `Build.Dialogue.Update`, `Build.Dialogue.Create`, `Build.Dialogue.Delete`, ``

- [ ] **Step 2:** Dispatch cases in `handleBuildOp` following the Zone/Mob
pattern exactly (unmarshal → buildErr on bad payload → handler →
`sendBuildResult` / `sendDialogueDetail`).

- [ ] **Step 3:** `mobDetail` gains `HasDialogue bool` populated via
`dialogue.Load(int(m.MobId), m.Zone) != nil` in `buildMobGet` — the button
indicator. **Caveat:** `Load` sets the nil sentinel for dialogueless mobs;
that is correct behaviour (create clears it), but note it in a comment so
nobody "optimizes" the call away thinking it is side-effect-free.

- [ ] **Step 4:** `go build ./... && go test ./modules/gmcp/ -count=1` → clean; commit.

```bash
git add modules/gmcp/gmcp.go modules/gmcp/gmcp.Build.go modules/gmcp/gmcp.Mob.go
git commit -m "feat(build): route Build.Dialogue verbs + HasDialogue on mob detail"
```

---

### Task 7: The panel

**Files:** Create `_datafiles/html/public/static/js/dialogue.js`; modify `mobs.js`, `build.html`

Read `mobs.js` end to end first; mirror its module shape, field helpers,
drawer pattern, and dirty/save flow. Server contract is Task 5's payloads.

- [ ] **Step 1: Entry point.** `mobs.js`: a **Dialogue…** button on the mob
form (label reflects `detail.hasDialogue`: "Dialogue…" vs "Add dialogue…").
Click → `Build.Dialogue.Get {mobId, zone}` → swap the panel to the dialogue
editor; a not-found reply renders a Create offer instead.

- [ ] **Step 2: Sections.** Identity (read-only mob/zone) + defaultMood
picker (enum) + memory expiryPeriod with the SOP warning inline. Greetings:
`{text, moods}` rows. Patterns: collapsible rows summarised by keywords;
keywords/moods/responses list editors + the shared quest-gate drawer.

- [ ] **Step 3: Tree.** Root text/hints; root variants list (quest-gate
drawer each). Then the **node list: ordered, draggable** (up/down + drag,
matching however mobs.js does reorder — or the spawn editor's arrows),
each node collapsible: id, triggers, requires/unlocks as **pickers over this
file's node ids**, text, hints, quest-gate drawer. A visible rule line above
the list: "Nodes match in ORDER — quest-grant nodes must come first."

- [ ] **Step 4: Quest-gate drawer** (one shared component): grantsQuest
picker (enum tokens with quest names), questRequired/Excluded token pickers,
flag gates driven by the chosen quest's declared flags, item pickers (reuse
the spawn editor's login-time item list), bumpsRep rows, givesGold,
masterworkRequired.

- [ ] **Step 5: Save flow.** Save → `Build.Dialogue.Update` with the whole
file. On refusal: render each error against its named node/pattern. On
success with `warnings`: render them as a non-blocking amber list that stays
visible. Delete → confirm naming the consequence ("this mutes the NPC") →
`Build.Dialogue.Delete` → return to the mob form.

- [ ] **Step 6:** `node --check` on dialogue.js and the inline-script parse
check on build.html; commit.

```bash
git add _datafiles/html/public/static/js/dialogue.js _datafiles/html/public/static/js/mobs.js _datafiles/html/public/build.html
git commit -m "feat(build): dialogue editor panel"
```

---

### Task 8: Verification gate

- [ ] **Step 1:** `go test ./... -count=1` clean; `gofmt -l internal modules` and `go vet ./...` clean.
- [ ] **Step 2:** Strict dialogue sweep still green: `DOGMUD_BOOT_SMOKE=1 go test . -run TestSmoke_AllDialogueFilesParse -count=1` — baseline still empty, greetings ≥186.
- [ ] **Step 3:** Boot under `MapConsistencyEnforce: panic` (skip-worktree file — check content): zero PANIC, `errors=0 warnings=0`.
- [ ] **Step 4:** PATCH_NOTES.md — staff-facing entry, house voice.
- [ ] **Step 5:** Commit; then hand the browser gate to the user: mob →
Dialogue…, edit a pattern, reorder nodes, trip each refusal (semicolon,
grant without end token, grant node out of order), see the discoverability
warning, save, and `talk` to the NPC in game — the edit must be live with
no reboot (the cache-replace contract, verified end to end).

**Content adversarial playtest: N/A** — tooling, not content (spec §6).

---

## Out of scope

Quest editor (5c), behavior-tree editor (5d), prose/voice linting, bulk
operations, and any change to matching semantics, moods, or memory.
