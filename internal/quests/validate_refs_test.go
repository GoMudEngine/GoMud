package quests

import (
	"strings"
	"testing"
)

func permissiveQuestValidators() QuestValidators {
	return QuestValidators{
		StepExists:     func(string) bool { return true },
		MobExists:      func(int) bool { return true },
		ItemExists:     func(int) bool { return true },
		RoomExists:     func(int) bool { return true },
		BuffExists:     func(int) bool { return true },
		SpellExists:    func(string) bool { return true },
		SkillExists:    func(string) bool { return true },
		StatExists:     func(string) bool { return true },
		RecipeExists:   func(string) bool { return true },
		FactionExists:  func(string) bool { return true },
		FlagDeclared:   func(string, string) bool { return true },
		DialogueGrants: func(string) bool { return true },
		MobHasDialogue: func(int) bool { return true },
	}
}

func refsBaseQuest() Quest {
	return Quest{QuestId: 500, Name: "Refs Test", Steps: []QuestStep{
		{Id: "start"}, {Id: "end"},
	}, Triggers: []TriggerDef{
		{Event: "room_enter", Room: 100, Actions: []ActionDef{{Grant: "500-start"}}},
		{Event: "item_give", Mob: 9, Item: 30, Actions: []ActionDef{{Grant: "500-end"}}},
	}}
}

func errsContainingRefs(t *testing.T, errs []string, want string) {
	t.Helper()
	for _, e := range errs {
		if strings.Contains(e, want) {
			return
		}
	}
	t.Errorf("expected an error containing %q, got %v", want, errs)
}

func warnsContainingRefs(t *testing.T, warns []string, want string) {
	t.Helper()
	for _, w := range warns {
		if strings.Contains(w, want) {
			return
		}
	}
	t.Errorf("expected a warning containing %q, got %v", want, warns)
}

func TestValidateRefs_CleanQuestPasses(t *testing.T) {
	errs, warns := ValidateQuestRefs(refsBaseQuest(), permissiveQuestValidators())
	if len(errs) != 0 || len(warns) != 0 {
		t.Fatalf("clean quest should pass, got errs=%v warns=%v", errs, warns)
	}
}

func TestValidateRefs_TokenGates(t *testing.T) {
	v := permissiveQuestValidators()
	v.StepExists = func(tok string) bool { return tok == "7-start" }

	q := refsBaseQuest()
	q.Triggers[0].Conditions.Has = []string{"7-start", "8-bogus"}
	errs, _ := ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "8-bogus")

	// An OWN-quest token must name a step in the INCOMING definition, never
	// go through StepExists (the cache may hold the pre-save step list).
	q = refsBaseQuest()
	q.Triggers[0].Conditions.Missing = []string{"500-nonstep"}
	errs, _ = ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "500-nonstep")

	q = refsBaseQuest()
	q.Rewards.QuestId = "8-bogus"
	errs, _ = ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "8-bogus")
}

func TestValidateRefs_IdExistence(t *testing.T) {
	v := permissiveQuestValidators()
	v.MobExists = func(id int) bool { return id != 999 }
	v.ItemExists = func(id int) bool { return id != 888 }
	v.RoomExists = func(id int) bool { return id != 777 }
	v.BuffExists = func(id int) bool { return id != 666 }

	q := refsBaseQuest()
	q.Triggers[0].Actions = append(q.Triggers[0].Actions,
		ActionDef{NpcSay: &NpcSayDef{Mob: 999, Lines: []SayLineDef{{Text: "hi"}}}},
		ActionDef{SpawnItem: &SpawnDef{Id: 888, Room: 777}},
		ActionDef{ApplyBuff: &BuffDef{Buff: 666}},
	)
	q.Steps[0].MapTarget = 777
	q.Rewards.ItemId = 888
	errs, _ := ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "mob 999")
	errsContainingRefs(t, errs, "item 888")
	errsContainingRefs(t, errs, "room 777")
	errsContainingRefs(t, errs, "buff 666")
	errsContainingRefs(t, errs, "map_target")
}

func TestValidateRefs_NamedRegistries(t *testing.T) {
	v := permissiveQuestValidators()
	v.SpellExists = func(id string) bool { return id != "nospell" }
	v.SkillExists = func(n string) bool { return n != "noskill" }
	v.StatExists = func(n string) bool { return n != "nostat" }
	v.RecipeExists = func(id string) bool { return id != "norecipe" }
	v.FactionExists = func(id string) bool { return id != "nofaction" }

	q := refsBaseQuest()
	q.Triggers[0].Actions = append(q.Triggers[0].Actions,
		ActionDef{TeachSpell: "nospell"},
		ActionDef{TrainSkill: &SkillDef{Skill: "noskill", Level: 2}},
		ActionDef{TrainStat: &StatDef{Stat: "nostat", Amount: 1}},
		ActionDef{LearnRecipe: &RecipeDef{Recipe: "norecipe"}},
		ActionDef{BumpRep: &BumpRepDef{Faction: "nofaction", Delta: 5}},
	)
	q.Rewards.SkillInfo = "noskill:3"
	q.Rewards.StatInfo = "nostat:2"
	errs, _ := ValidateQuestRefs(q, v)
	for _, want := range []string{"nospell", "noskill", "nostat", "norecipe", "nofaction"} {
		errsContainingRefs(t, errs, want)
	}
}

func TestValidateRefs_Flags(t *testing.T) {
	v := permissiveQuestValidators()
	v.FlagDeclared = func(key, value string) bool { return false }

	// Own flags validate against the INCOMING declaration.
	q := refsBaseQuest()
	q.Flags = []QuestFlagDef{{Key: "branch", Values: []string{"a", "b"}}}
	q.Triggers[0].Actions = append(q.Triggers[0].Actions,
		ActionDef{SetFlag: &QuestFlagAction{Key: "500-branch", Value: "a"}})
	errs, _ := ValidateQuestRefs(q, v)
	if len(errs) != 0 {
		t.Fatalf("own declared flag must pass, got %v", errs)
	}

	// Wrong value against own declaration.
	q.Triggers[0].Actions[len(q.Triggers[0].Actions)-1] =
		ActionDef{SetFlag: &QuestFlagAction{Key: "500-branch", Value: "z"}}
	errs, _ = ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "500-branch")

	// Foreign flag falls to FlagDeclared.
	q = refsBaseQuest()
	q.Triggers[0].Conditions.HasFlag = map[string]string{"11-branch": "rhett"}
	errs, _ = ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "11-branch")
}

func TestValidateRefs_SequenceNesting(t *testing.T) {
	v := permissiveQuestValidators()
	v.StepExists = func(string) bool { return false }

	q := refsBaseQuest()
	q.Triggers[0].Actions = append(q.Triggers[0].Actions, ActionDef{
		Sequence: &SequenceDef{Lines: []SayLineDef{{Text: "..."}},
			OnComplete: []ActionDef{{Grant: "9-hidden"}}},
	})
	errs, _ := ValidateQuestRefs(q, v)
	errsContainingRefs(t, errs, "9-hidden")
}

func TestValidateRefs_Warnings(t *testing.T) {
	v := permissiveQuestValidators()
	v.DialogueGrants = func(string) bool { return false }
	v.MobHasDialogue = func(int) bool { return false }

	// "end" is granted by a trigger; "orphan" is granted by nothing.
	q := refsBaseQuest()
	q.Steps = append(q.Steps, QuestStep{Id: "orphan"})
	q.Triggers[1].Actions = append(q.Triggers[1].Actions,
		ActionDef{NpcSay: &NpcSayDef{Mob: 9, Lines: []SayLineDef{{Text: "hello"}}}})
	// "start" is granted by trigger 0, so with DialogueGrants=false it does
	// NOT warn; only the truly ungranted step does.
	errs, warns := ValidateQuestRefs(q, v)
	if len(errs) != 0 {
		t.Fatalf("warnings-only quest must not refuse: %v", errs)
	}
	warnsContainingRefs(t, warns, "orphan")
	warnsContainingRefs(t, warns, "mob 9")
	for _, w := range warns {
		if strings.Contains(w, `"start"`) || strings.Contains(w, `"end"`) {
			t.Errorf("granted steps must not warn: %v", w)
		}
	}
}
