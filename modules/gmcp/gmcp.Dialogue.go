package gmcp

// Build.Dialogue.* GMCP packages — the server side of the admin web dialogue
// editor (admin web-building 5b). Handlers take a dialogueDeps so they
// unit-test against fakes; realDialogueDeps wires the live engine, including
// the registry-backed validators (injected rather than hardwired so these
// tests run with no world loaded).

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/dialogue"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/quests"
)

type dialogueDeps struct {
	load       func(mobId int, zone string) *dialogue.DialogueFile
	save       func(df dialogue.DialogueFile) error
	create     func(mobId int, zone string) error
	del        func(mobId int, zone string) error
	validators dialogue.DialogueValidators
}

func realDialogueDeps() dialogueDeps {
	return dialogueDeps{
		load:   dialogue.Load,
		save:   dialogue.SaveDialogueFile,
		create: dialogue.CreateNewDialogueFile,
		del:    dialogue.DeleteDialogueFile,
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
	}
}

// ---- payloads ----

type dialogueGetReq struct {
	MobId int    `json:"mobId"`
	Zone  string `json:"zone"`
}

type dialogueQuestToken struct {
	Token     string `json:"token"`
	QuestName string `json:"questName"`
}

type dialogueQuestFlags struct {
	QuestName string              `json:"questName"`
	Flags     map[string][]string `json:"flags"` // key -> allowed values
}

type dialogueEnums struct {
	QuestTokens []dialogueQuestToken `json:"questTokens"`
	QuestFlags  []dialogueQuestFlags `json:"questFlags"`
	Moods       []string             `json:"moods"`
}

type dialogueDetail struct {
	File  dialogue.DialogueFile `json:"file"`
	Found bool                  `json:"found"`
	Enums dialogueEnums         `json:"enums"`
}

// ---- handlers ----

func buildDialogueGet(d dialogueDeps, mobId int, zone string) (dialogueDetail, bool) {
	df := d.load(mobId, zone)
	if df == nil {
		return dialogueDetail{Found: false, Enums: collectDialogueEnums()}, false
	}
	return dialogueDetail{File: *df, Found: true, Enums: collectDialogueEnums()}, true
}

func buildDialogueUpdate(d dialogueDeps, df dialogue.DialogueFile) BuildResult {
	errs, warns := dialogue.ValidateDialogueFile(df, d.validators)
	if len(errs) > 0 {
		return buildErr("dialogue for mob %d refused:\n%s", df.MobId, strings.Join(errs, "\n"))
	}
	if err := d.save(df); err != nil {
		return buildErr("could not save dialogue for mob %d: %s", df.MobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: df.MobId, Message: fmt.Sprintf("dialogue for mob %d saved", df.MobId), Warnings: warns}
}

func buildDialogueCreate(d dialogueDeps, mobId int, zone string) BuildResult {
	if d.load(mobId, zone) != nil {
		return buildErr("mob %d already has a dialogue file", mobId)
	}
	if err := d.create(mobId, zone); err != nil {
		return buildErr("%s", err.Error())
	}
	return BuildResult{Ok: true, MobId: mobId, Message: fmt.Sprintf("dialogue created for mob %d", mobId)}
}

func buildDialogueDelete(d dialogueDeps, mobId int, zone string) BuildResult {
	if d.load(mobId, zone) == nil {
		return buildErr("mob %d has no dialogue file", mobId)
	}
	if err := d.del(mobId, zone); err != nil {
		return buildErr("could not delete dialogue for mob %d: %s", mobId, err.Error())
	}
	return BuildResult{Ok: true, MobId: mobId, Message: fmt.Sprintf("dialogue for mob %d deleted — the NPC is mute until a new file is created", mobId)}
}

// ---- enums ----

// collectDialogueEnums builds the picker payloads: one token per quest step
// ("%d-%s" over each quest's step ids — the composition quests.GetQuest
// parses via TokenToParts), per-quest declared flags, and the mood set.
func collectDialogueEnums() dialogueEnums {
	e := dialogueEnums{
		Moods: []string{
			string(dialogue.MoodFriendly), string(dialogue.MoodNeutral),
			string(dialogue.MoodHostile), string(dialogue.MoodAfraid),
			string(dialogue.MoodGrateful),
		},
	}
	for _, q := range quests.GetAllQuests() {
		for _, step := range q.Steps {
			e.QuestTokens = append(e.QuestTokens, dialogueQuestToken{
				Token:     fmt.Sprintf("%d-%s", q.QuestId, step.Id),
				QuestName: q.Name,
			})
		}
		if len(q.Flags) > 0 {
			qf := dialogueQuestFlags{QuestName: q.Name, Flags: map[string][]string{}}
			for _, f := range q.Flags {
				qf.Flags[f.Key] = f.Values
			}
			e.QuestFlags = append(e.QuestFlags, qf)
		}
	}
	sort.Slice(e.QuestTokens, func(i, j int) bool { return e.QuestTokens[i].Token < e.QuestTokens[j].Token })
	return e
}
