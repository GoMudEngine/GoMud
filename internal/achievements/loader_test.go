package achievements

import "testing"

func TestValidateDefinition(t *testing.T) {
	good := Definition{Id: "first-blood", Name: "First Blood", Description: "d", Category: "combat", Points: 5, Trigger: Trigger{Type: "mob_kills", Threshold: 1}}
	if err := validateDefinition(good, "first-blood"); err != nil {
		t.Errorf("good def should validate: %v", err)
	}

	cases := []struct {
		name string
		d    Definition
		file string
	}{
		{"unknown type", Definition{Id: "x", Name: "n", Category: "combat", Trigger: Trigger{Type: "bogus"}}, "x"},
		{"bad category", Definition{Id: "x", Name: "n", Category: "nope", Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "x"},
		{"stat missing param", Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Threshold: 1}}, "x"},
		{"bad stat name", Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Stat: "wisdom", Threshold: 1}}, "x"},
		{"quest missing token", Definition{Id: "x", Name: "n", Category: "quests", Trigger: Trigger{Type: "quest_completed"}}, "x"},
		{"filename mismatch", Definition{Id: "x", Name: "n", Category: "combat", Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "y"},
		{"negative points", Definition{Id: "x", Name: "n", Category: "combat", Points: -1, Trigger: Trigger{Type: "mob_kills", Threshold: 1}}, "x"},
		{"threshold type zero", Definition{Id: "x", Name: "n", Category: "combat", Trigger: Trigger{Type: "mob_kills", Threshold: 0}}, "x"},
	}
	for _, tc := range cases {
		if err := validateDefinition(tc.d, tc.file); err == nil {
			t.Errorf("%s: expected validation error, got nil", tc.name)
		}
	}

	// "any" is a valid stat/skill.
	anyStat := Definition{Id: "x", Name: "n", Category: "progression", Trigger: Trigger{Type: "stat_reached", Stat: "any", Threshold: 130}}
	if err := validateDefinition(anyStat, "x"); err != nil {
		t.Errorf("stat 'any' should validate: %v", err)
	}
	anySkill := Definition{Id: "y", Name: "n", Category: "progression", Trigger: Trigger{Type: "skill_reached", Skill: "any", Threshold: 25}}
	if err := validateDefinition(anySkill, "y"); err != nil {
		t.Errorf("skill 'any' should validate: %v", err)
	}
}
