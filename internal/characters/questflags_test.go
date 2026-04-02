package characters

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetAndGetQuestFlag(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	assert.Equal(t, "rhett", c.GetQuestFlag("11-branch"))
}

func TestGetQuestFlag_Missing(t *testing.T) {
	c := &Character{}
	assert.Equal(t, "", c.GetQuestFlag("11-branch"))
}

func TestHasQuestFlag(t *testing.T) {
	c := &Character{}
	assert.False(t, c.HasQuestFlag("11-branch"))
	c.SetQuestFlag("11-branch", "rhett")
	assert.True(t, c.HasQuestFlag("11-branch"))
}

func TestClearQuestFlag(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	c.ClearQuestFlag("11-branch")
	assert.False(t, c.HasQuestFlag("11-branch"))
	assert.Equal(t, "", c.GetQuestFlag("11-branch"))
}

func TestSetQuestFlag_Overwrite(t *testing.T) {
	c := &Character{}
	c.SetQuestFlag("11-branch", "rhett")
	c.SetQuestFlag("11-branch", "sylara")
	assert.Equal(t, "sylara", c.GetQuestFlag("11-branch"))
}
