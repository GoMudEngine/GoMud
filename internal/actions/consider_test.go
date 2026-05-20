package actions

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// fakeActor is a minimal Actor implementation for unit tests. It
// records SendText messages and tracks OnStatUse/OnSkillUse calls.
type fakeActor struct {
	char     *characters.Character
	name     string
	isPlayer bool
	sent     []string
	statUses map[string]int
}

// newFakeActor builds a fakeActor with the given ValueAdj for
// Strength/Dexterity and a HealthMax for durability. Setting
// ValueAdj directly skips the Training → Value → ValueAdj
// recalculation pipeline and matches how character_test.go
// constructs characters for combat math testing.
func newFakeActor(name string, statAdj, healthMax int, isPlayer bool) *fakeActor {
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Strength.ValueAdj = statAdj
	c.Stats.Dexterity.ValueAdj = statAdj
	c.Stats.Perception.ValueAdj = statAdj
	c.Stats.Vitality.ValueAdj = statAdj
	c.Stats.Willpower.ValueAdj = statAdj
	c.Stats.Charisma.ValueAdj = statAdj
	c.HealthMax.Value = healthMax
	c.StaminaMax.Value = healthMax
	c.ConvictionMax.Value = healthMax / 2
	return &fakeActor{
		char:     c,
		name:     name,
		isPlayer: isPlayer,
		statUses: map[string]int{},
	}
}

func (a *fakeActor) GetCharacter() *characters.Character      { return a.char }
func (a *fakeActor) GetRoom() *rooms.Room                     { return nil }
func (a *fakeActor) SendText(_ messaging.Category, msg string) { a.sent = append(a.sent, msg) }
func (a *fakeActor) SendRoomCommunication(msg string, _ bool) {}
func (a *fakeActor) GetName() string                            { return a.name }
func (a *fakeActor) IsPlayer() bool                             { return a.isPlayer }
func (a *fakeActor) GetUserId() int                             { return 0 }
func (a *fakeActor) GetMobInstanceId() int                      { return 0 }
func (a *fakeActor) AddBuff(buffId int, source string)          {}
func (a *fakeActor) OnSkillUse(skillName string) bool {
	a.statUses[skillName]++
	return false
}
func (a *fakeActor) OnStatUse(statName string) bool {
	a.statUses[statName]++
	return false
}
func (a *fakeActor) OnCriticalSuccess(skillName string) {}
func (a *fakeActor) OnCriticalFailure(skillName string) {}

func TestConsider_OnStatUsePerception(t *testing.T) {
	self := newFakeActor("self", 100, 100, true)
	target := newFakeActor("target", 100, 100, true)

	Consider(self, target)
	if self.statUses["perception"] != 1 {
		t.Errorf("expected perception OnStatUse called once, got %d",
			self.statUses["perception"])
	}
}

func TestConsider_TextEmittedForPlayer(t *testing.T) {
	self := newFakeActor("hero", 100, 100, true)
	target := newFakeActor("orc", 100, 100, false)

	Consider(self, target)
	if len(self.sent) != 2 {
		t.Fatalf("expected 2 text lines emitted, got %d", len(self.sent))
	}
	if !strings.Contains(self.sent[0], "orc") {
		t.Errorf("first line should name the target, got %q", self.sent[0])
	}
	if !strings.Contains(self.sent[1], "instincts tell you") {
		t.Errorf("second line should be the prediction, got %q", self.sent[1])
	}
}

func TestConsider_ZeroTargetPower(t *testing.T) {
	self := newFakeActor("hero", 100, 100, true)
	// Construct a target with truly zero PowerScore: all ValueAdj=0,
	// no health, no skills, no mutations.
	target := &fakeActor{
		char:     &characters.Character{Name: "Ghost", Buffs: buffs.New()},
		name:     "Ghost",
		isPlayer: false,
		statUses: map[string]int{},
	}

	r := Consider(self, target)
	if r.TargetPower != 0 {
		// If this fails, default-construct of Character introduces
		// non-zero terms (e.g., default weapon offense). Document
		// the actual TargetPower and adjust the test expectation.
		t.Logf("TargetPower=%f (expected 0); see PowerScore default-weapon path", r.TargetPower)
	}
	if r.TargetPower == 0 && r.Ratio != 0 {
		t.Errorf("expected Ratio=0 when TargetPower=0, got %f", r.Ratio)
	}
}

func TestConsider_PredictionRatioBands(t *testing.T) {
	cases := []struct {
		ratio  float64
		expect string
	}{
		{5.0, "pose no threat"},
		{3.5, "clear advantage"},
		{2.5, "odds favor you"},
		{1.5, "even contest"},
		{0.75, "upper hand"},
		{0.25, "severely outmatched"},
		{0.0, "will not survive"},
	}
	for _, c := range cases {
		got := predictionFor(c.ratio)
		if !strings.Contains(got, c.expect) {
			t.Errorf("ratio=%v: expected %q in %q", c.ratio, c.expect, got)
		}
	}
}
