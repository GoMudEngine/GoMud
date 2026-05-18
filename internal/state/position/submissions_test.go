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

// TestIsTopSubEligible — chunk 4b-fixup T18: ControlLevel replaced by
// IsControllerRole bool. Top subs require IsControllerRole=true.
func TestIsTopSubEligible(t *testing.T) {
	assert.True(t, position.IsTopSubEligible(position.Mount, true),
		"controller (IsControllerRole=true) can attempt top subs")
	assert.False(t, position.IsTopSubEligible(position.Mount, false),
		"controlled (IsControllerRole=false) cannot attempt top subs")
	assert.False(t, position.IsTopSubEligible(position.Standing, true),
		"no top subs at Standing regardless of role")
}

// TestIsBottomSubEligible — chunk 4b-fixup T18: ControlLevel replaced by
// IsControllerRole bool. Bottom subs require IsControllerRole=false.
func TestIsBottomSubEligible(t *testing.T) {
	assert.True(t, position.IsBottomSubEligible(position.Mount, false),
		"controlled (IsControllerRole=false) can attempt bottom subs")
	assert.False(t, position.IsBottomSubEligible(position.Mount, true),
		"controller (IsControllerRole=true) cannot attempt bottom subs")
	assert.False(t, position.IsBottomSubEligible(position.KneeOnBelly, false),
		"no bottom subs at KOB regardless of role")
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
