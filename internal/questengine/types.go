package questengine

import "github.com/GoMudEngine/GoMud/internal/quests"

// The definition types live in internal/quests — the single owner of the
// quest file parse (5c-pre unification). These aliases keep questengine's
// API and its evaluation tests source-compatible. Evaluation logic stays in
// this package; only the data shapes moved.
type (
	QuestFlagDef     = quests.QuestFlagDef
	QuestDef         = quests.Quest
	QuestStep        = quests.QuestStep
	QuestRewards     = quests.QuestReward
	TriggerDef       = quests.TriggerDef
	Conditions       = quests.Conditions
	ActionDef        = quests.ActionDef
	BumpRepDef       = quests.BumpRepDef
	DeclareBountyDef = quests.DeclareBountyDef
	QuestFlagAction  = quests.QuestFlagAction
	NpcSayDef        = quests.NpcSayDef
	SayLineDef       = quests.SayLineDef
	SpawnDef         = quests.SpawnDef
	ExitLock         = quests.ExitLock
	SkillDef         = quests.SkillDef
	StatDef          = quests.StatDef
	RecipeDef        = quests.RecipeDef
	BuffDef          = quests.BuffDef
	SequenceDef      = quests.SequenceDef
)

// EventDetails carries context about the event that triggered evaluation.
// It describes an engine INVOCATION, not a quest definition, so it stays here.
type EventDetails struct {
	UserId     int
	RoomId     int
	MobId      int
	ItemId     int
	Skill      string
	Command    string
	Topic      string
	Noun       string
	Verb       string
	QuestToken string
}

// NotifyResult is returned by Notify to inform the caller how the event was handled.
type NotifyResult struct {
	Handled     bool // at least one trigger matched and executed
	ConsumeItem bool // the item should be consumed (not transferred to mob)
}
