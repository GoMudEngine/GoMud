package position_test

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/state/position"
	"github.com/stretchr/testify/assert"
)

func TestTopSubmissionsForPosition_CoreMappings(t *testing.T) {
	cases := map[position.State][]position.SubmissionType{
		position.Standing:     nil,
		position.Prone:        nil,
		position.Supine:       nil,
		position.Turtle:       nil,
		position.Clinch:       nil,
		position.BackStanding: {position.SubRNC},
		position.Mount:        {position.SubAmericana, position.SubTriangle, position.SubArmbar},
		position.SideControl:  {position.SubKimura, position.SubAmericana},
		position.KneeOnBelly:  {position.SubArmbar},
		position.NorthSouth:   {position.SubKimura, position.SubAnaconda},
		position.Crucifix:     {position.SubArmbar},
		position.BackGround:   {position.SubRNC},
		position.HalfGuard:    {position.SubKimura},
		position.Guard:        {position.SubTriangle, position.SubArmbar, position.SubOmoplata},
	}
	for state, want := range cases {
		got := position.TopSubmissionsForPosition(state)
		assert.Equal(t, want, got, "TopSubmissionsForPosition(%s)", state)
	}
}

func TestBottomSubmissionsForPosition_CoreMappings(t *testing.T) {
	cases := map[position.State][]position.SubmissionType{
		position.Standing:     nil,
		position.Prone:        nil,
		position.Supine:       nil,
		position.Turtle:       nil,
		position.Clinch:       nil,
		position.BackStanding: nil,
		position.Mount:        {position.SubTriangle, position.SubArmbar},
		position.SideControl:  {position.SubKimura},
		position.KneeOnBelly:  nil,
		position.NorthSouth:   {position.SubKimura},
		position.Crucifix:     nil,
		position.BackGround:   nil,
		position.HalfGuard:    {position.SubKimura},
		position.Guard:        nil,
	}
	for state, want := range cases {
		got := position.BottomSubmissionsForPosition(state)
		assert.Equal(t, want, got, "BottomSubmissionsForPosition(%s)", state)
	}
}

func TestIsTopSubEligible(t *testing.T) {
	assert.True(t, position.IsTopSubEligible(position.Mount, position.InControl))
	assert.True(t, position.IsTopSubEligible(position.Mount, position.LosingControl))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.Neutral))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.BecomingControlled))
	assert.False(t, position.IsTopSubEligible(position.Mount, position.Controlled))
	assert.False(t, position.IsTopSubEligible(position.Standing, position.InControl), "no top subs at Standing")
}

func TestIsBottomSubEligible(t *testing.T) {
	assert.True(t, position.IsBottomSubEligible(position.Mount, position.Controlled))
	assert.True(t, position.IsBottomSubEligible(position.Mount, position.BecomingControlled))
	assert.False(t, position.IsBottomSubEligible(position.Mount, position.Neutral))
	assert.False(t, position.IsBottomSubEligible(position.Mount, position.InControl))
	assert.False(t, position.IsBottomSubEligible(position.KneeOnBelly, position.Controlled), "no bottom subs at KOB")
}

func TestCrippleBodyPart(t *testing.T) {
	assert.Equal(t, "arm", position.CrippleBodyPart(position.SubArmbar))
	assert.Equal(t, "arm", position.CrippleBodyPart(position.SubOmoplata))
	assert.Equal(t, "shoulder", position.CrippleBodyPart(position.SubKimura))
	assert.Equal(t, "shoulder", position.CrippleBodyPart(position.SubAmericana))
	assert.Equal(t, "", position.CrippleBodyPart(position.SubRNC))
	assert.Equal(t, "", position.CrippleBodyPart(position.SubTriangle))
	assert.Equal(t, "", position.CrippleBodyPart(position.SubAnaconda))
}
