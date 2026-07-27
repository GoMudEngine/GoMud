package gmcp

// The quest delete guard (admin web-building 5c): a quest may not be deleted
// while anything still references it — a dangling grant or gate is a
// silently broken NPC. Achievements are deliberately absent: they only count
// completions (quests_completed), they never name tokens.

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/quests"
)

// questRefEntry describes one live reference to a quest, surfaced verbatim in
// the delete refusal.
type questRefEntry struct {
	Kind   string `json:"kind"` // dialogue | quest
	Where  string `json:"where"`
	Detail string `json:"detail"`
}

// tokenBelongsTo reports whether a quest token or flag key carries this
// quest's "<id>-" prefix.
func tokenBelongsTo(questId int, tok string) bool {
	return strings.HasPrefix(tok, fmt.Sprintf("%d-", questId))
}

// dialogueGateRefs collects references to questId from one gate-carrying
// dialogue element (Pattern, TreeNode, or QuestGreeting share these fields).
type dialogueGate struct {
	grantsQuest       string
	questRequired     []string
	questExcluded     []string
	questFlagRequired map[string]string
	questFlagExcluded map[string]string
	setsQuestFlag     *dialogue.QuestFlagSet
}

func (g dialogueGate) refs(questId int) []string {
	out := []string{}
	if tokenBelongsTo(questId, g.grantsQuest) {
		out = append(out, fmt.Sprintf("grantsQuest %q", g.grantsQuest))
	}
	for _, tok := range g.questRequired {
		if tokenBelongsTo(questId, tok) {
			out = append(out, fmt.Sprintf("questRequired %q", tok))
		}
	}
	for _, tok := range g.questExcluded {
		if tokenBelongsTo(questId, tok) {
			out = append(out, fmt.Sprintf("questExcluded %q", tok))
		}
	}
	for k := range g.questFlagRequired {
		if tokenBelongsTo(questId, k) {
			out = append(out, fmt.Sprintf("questFlagRequired %q", k))
		}
	}
	for k := range g.questFlagExcluded {
		if tokenBelongsTo(questId, k) {
			out = append(out, fmt.Sprintf("questFlagExcluded %q", k))
		}
	}
	if g.setsQuestFlag != nil && tokenBelongsTo(questId, g.setsQuestFlag.Key) {
		out = append(out, fmt.Sprintf("setsQuestFlag %q", g.setsQuestFlag.Key))
	}
	return out
}

// walkDialogueGates visits every gate-carrying element of a dialogue file.
// DialogueFile.Tree is a POINTER — nil for pattern-only NPCs, which is most
// of the live tree — so the tree sections are guarded.
func walkDialogueGates(df *dialogue.DialogueFile, fn func(where string, g dialogueGate)) {
	for i, p := range df.Patterns {
		fn(fmt.Sprintf("pattern %d", i), dialogueGate{p.GrantsQuest, p.QuestRequired, p.QuestExcluded,
			p.QuestFlagRequired, p.QuestFlagExcluded, p.SetsQuestFlag})
	}
	if df.Tree == nil {
		return
	}
	for _, n := range df.Tree.Nodes {
		fn(fmt.Sprintf("node %q", n.Id), dialogueGate{n.GrantsQuest, n.QuestRequired, n.QuestExcluded,
			n.QuestFlagRequired, n.QuestFlagExcluded, n.SetsQuestFlag})
	}
	for i, v := range df.Tree.Root.Variants {
		fn(fmt.Sprintf("root variant %d", i), dialogueGate{v.GrantsQuest, v.QuestRequired, v.QuestExcluded,
			v.QuestFlagRequired, v.QuestFlagExcluded, v.SetsQuestFlag})
	}
}

// questTriggerTokens visits every quest token another quest's triggers and
// rewards carry (grant targets included — a foreign grant is a reference).
func questTriggerTokens(q quests.Quest, fn func(where, tok string)) {
	var walkActions func(where string, actions []quests.ActionDef, depth int)
	walkActions = func(where string, actions []quests.ActionDef, depth int) {
		for i, a := range actions {
			aw := fmt.Sprintf("%s action %d", where, i)
			if a.Grant != "" {
				fn(aw+" grant", a.Grant)
			}
			if a.SetFlag != nil {
				fn(aw+" set_flag", a.SetFlag.Key)
			}
			if a.Sequence != nil && depth < 2 {
				walkActions(aw+" sequence on_complete", a.Sequence.OnComplete, depth+1)
			}
		}
	}
	for i, t := range q.Triggers {
		tw := fmt.Sprintf("trigger %d", i)
		if t.QuestToken != "" {
			fn(tw+" quest_token", t.QuestToken)
		}
		for _, tok := range t.Conditions.Has {
			fn(tw+" has", tok)
		}
		for _, tok := range t.Conditions.Missing {
			fn(tw+" missing", tok)
		}
		for k := range t.Conditions.HasFlag {
			fn(tw+" has_flag", k)
		}
		for k := range t.Conditions.MissingFlag {
			fn(tw+" missing_flag", k)
		}
		walkActions(tw, t.Actions, 0)
	}
	if q.Rewards.QuestId != "" {
		fn("rewards questid", q.Rewards.QuestId)
	}
}

// scanQuestReferences finds every live reference to a quest: dialogue-file
// gates and other quests' triggers/rewards.
func scanQuestReferences(questId int) []questRefEntry {
	out := []questRefEntry{}

	dialogue.WalkAllFiles(func(mobId int, zone string, df *dialogue.DialogueFile) {
		walkDialogueGates(df, func(where string, g dialogueGate) {
			for _, detail := range g.refs(questId) {
				out = append(out, questRefEntry{Kind: "dialogue",
					Where: fmt.Sprintf("mob %d (%s) %s", mobId, zone, where), Detail: detail})
			}
		})
	})

	for _, q := range quests.GetAllQuests() {
		if q.QuestId == questId {
			continue
		}
		questTriggerTokens(q, func(where, tok string) {
			if tokenBelongsTo(questId, tok) {
				out = append(out, questRefEntry{Kind: "quest",
					Where:  fmt.Sprintf("quest %d (%s) %s", q.QuestId, q.Name, where),
					Detail: fmt.Sprintf("%q", tok)})
			}
		})
	}

	return out
}

// collectDialogueGrantTokens indexes every token any dialogue file grants —
// the "is this step reachable" warning's dialogue side.
func collectDialogueGrantTokens() map[string]bool {
	granted := map[string]bool{}
	dialogue.WalkAllFiles(func(mobId int, zone string, df *dialogue.DialogueFile) {
		walkDialogueGates(df, func(where string, g dialogueGate) {
			if g.grantsQuest != "" {
				granted[g.grantsQuest] = true
			}
		})
	})
	return granted
}

// mobHasDialogueFile checks by mob template zone. Uses dialogue.Load (and so
// shares its cache behavior — fine: MainWorker only, and Load's nil-sentinel
// semantics are the live game's own).
func mobHasDialogueFile(mobId int) bool {
	m := mobs.GetMobSpec(mobs.MobId(mobId))
	if m == nil {
		return false
	}
	return dialogue.Load(mobId, m.Zone) != nil
}
