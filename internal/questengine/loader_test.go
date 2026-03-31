package questengine

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQuestDef_Validate_Valid(t *testing.T) {
	q := &QuestDef{
		QuestId: 3,
		Name:    "Test Quest",
		Steps:   []QuestStep{{Id: "start"}, {Id: "end"}},
		Triggers: []TriggerDef{
			{Event: "room_enter", Room: 100, Actions: []ActionDef{{Grant: "3-start"}}},
			{Event: "item_give", Item: 14, Mob: 79, Actions: []ActionDef{{Grant: "3-end"}}},
		},
	}
	assert.NoError(t, q.Validate())
}

func TestQuestDef_Validate_MissingId(t *testing.T) {
	q := &QuestDef{Name: "No ID", Steps: []QuestStep{{Id: "start"}}}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_MissingName(t *testing.T) {
	q := &QuestDef{QuestId: 1, Steps: []QuestStep{{Id: "start"}}}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_NoSteps(t *testing.T) {
	q := &QuestDef{QuestId: 1, Name: "No Steps"}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_TriggerNoEvent(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "Bad Trigger",
		Steps:    []QuestStep{{Id: "start"}},
		Triggers: []TriggerDef{{Actions: []ActionDef{{Grant: "1-start"}}}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_TriggerInvalidEvent(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "Bad Event",
		Steps:    []QuestStep{{Id: "start"}},
		Triggers: []TriggerDef{{Event: "invalid_event", Actions: []ActionDef{{Grant: "1-start"}}}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_TriggerNoActions(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "No Actions",
		Steps:    []QuestStep{{Id: "start"}},
		Triggers: []TriggerDef{{Event: "room_enter", Room: 100}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_DuplicateStep(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "Dup Step",
		Steps: []QuestStep{{Id: "start"}, {Id: "start"}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_EmptyStepId(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "Empty Step",
		Steps: []QuestStep{{Id: ""}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_Validate_GrantUnknownStep(t *testing.T) {
	q := &QuestDef{
		QuestId: 1, Name: "Bad Grant",
		Steps:    []QuestStep{{Id: "start"}},
		Triggers: []TriggerDef{{Event: "room_enter", Room: 100, Actions: []ActionDef{{Grant: "1-nonexistent"}}}},
	}
	assert.Error(t, q.Validate())
}

func TestQuestDef_FilepathAndId(t *testing.T) {
	q := &QuestDef{QuestId: 3, Name: "The Scholar's Collection"}
	assert.Equal(t, 3, q.Id())
	fp := q.Filepath()
	assert.Contains(t, fp, "3-")
	assert.Contains(t, fp, ".yaml")
}

func TestGetEngine_InitializesIfNil(t *testing.T) {
	globalEngine = nil
	e := GetEngine()
	assert.NotNil(t, e)
}
