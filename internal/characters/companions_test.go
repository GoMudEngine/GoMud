package characters

import (
	"math"
	"testing"

	"gopkg.in/yaml.v2"

	"github.com/GoMudEngine/GoMud/internal/mutations"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ─── GetMaxCompanions ────────────────────────────────────────────────────────

func TestGetMaxCompanions_Ranks(t *testing.T) {
	// Seed a minimal manifestation spell so the "has spell → min 1" path works.
	cleanup := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"spirit-wolf": {
			SpellId: "spirit-wolf",
			Name:    "Spirit Wolf",
			Schools: []string{"manifestation"},
		},
	})
	defer cleanup()

	tests := []struct {
		name         string
		skill        int
		hasSpell     bool
		wantMax      int
	}{
		// No manifestation spell and no skill → 0
		{"skill=0 no spell", 0, false, 0},
		// Has a manifestation spell but skill=0 → floor(0/19)=0, bumped to 1
		{"skill=0 with spell", 0, true, 1},
		// Boundary cases around divisor 19
		{"skill=18", 18, false, 0},  // 18/19 = 0
		{"skill=19", 19, false, 1},  // 19/19 = 1
		{"skill=37", 37, false, 1},  // 37/19 = 1
		{"skill=38", 38, false, 2},  // 38/19 = 2
		{"skill=56", 56, false, 2},  // 56/19 = 2
		{"skill=57", 57, false, 3},  // 57/19 = 3
		{"skill=75", 75, false, 3},  // 75/19 = 3 (floor)
		{"skill=76", 76, false, 4},  // 76/19 = 4
		// Hard cap at 4 even if skill is very high
		{"skill=200 cap=4", 200, false, 4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := New()
			c.Skills[string(skills.Manifestation)] = tt.skill
			if tt.hasSpell {
				c.SpellBook["spirit-wolf"] = 1
			}
			assert.Equal(t, tt.wantMax, c.GetMaxCompanions())
		})
	}
}

// ─── AddCompanion ────────────────────────────────────────────────────────────

func TestAddCompanion_Success(t *testing.T) {
	c := New()
	// Give skill 19 → cap of 1
	c.Skills[string(skills.Manifestation)] = 19

	comp := CompanionInfo{
		MobId:      1001,
		InstanceId: 5,
		SourceType: CompanionSummoned,
		Name:       "Spirit Wolf",
	}

	ok := c.AddCompanion(comp)
	assert.True(t, ok, "expected AddCompanion to succeed when under cap")
	require.Len(t, c.Companions, 1)
	assert.Equal(t, comp, c.Companions[0])
}

func TestAddCompanion_AtCap(t *testing.T) {
	c := New()
	// cap = 1
	c.Skills[string(skills.Manifestation)] = 19

	first := CompanionInfo{MobId: 1001, InstanceId: 1, Name: "Wolf One"}
	second := CompanionInfo{MobId: 1002, InstanceId: 2, Name: "Wolf Two"}

	ok := c.AddCompanion(first)
	assert.True(t, ok, "first add should succeed")

	ok = c.AddCompanion(second)
	assert.False(t, ok, "second add should fail at cap")
	assert.Len(t, c.Companions, 1, "companion count should remain 1")
}

// ─── RemoveCompanion ─────────────────────────────────────────────────────────

func TestRemoveCompanion_ByInstanceId(t *testing.T) {
	c := New()
	// cap = 2
	c.Skills[string(skills.Manifestation)] = 38

	alpha := CompanionInfo{MobId: 1001, InstanceId: 10, Name: "Alpha Wolf"}
	beta := CompanionInfo{MobId: 1002, InstanceId: 20, Name: "Beta Bear"}

	c.AddCompanion(alpha)
	c.AddCompanion(beta)
	require.Len(t, c.Companions, 2)

	removed := c.RemoveCompanion(10)
	require.NotNil(t, removed, "expected a non-nil return for valid instance ID")
	assert.Equal(t, 10, removed.InstanceId)
	assert.Equal(t, "Alpha Wolf", removed.Name)

	// Beta should still be present
	require.Len(t, c.Companions, 1)
	assert.Equal(t, 20, c.Companions[0].InstanceId)
}

// ─── GetCompanion (partial name) ─────────────────────────────────────────────

func TestGetCompanion_PartialMatch(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 19

	comp := CompanionInfo{MobId: 1001, InstanceId: 7, Name: "Steppe Spirit Wolf"}
	c.AddCompanion(comp)

	found := c.GetCompanion("wolf")
	require.NotNil(t, found, "expected to find companion via partial match 'wolf'")
	assert.Equal(t, "Steppe Spirit Wolf", found.Name)
}

func TestGetCompanion_NotFound(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 19

	comp := CompanionInfo{MobId: 1001, InstanceId: 7, Name: "Steppe Spirit Wolf"}
	c.AddCompanion(comp)

	found := c.GetCompanion("dragon")
	assert.Nil(t, found, "expected nil for name not matching any companion")
}

// ─── GetCompanionByInstanceId ─────────────────────────────────────────────────

func TestGetCompanionByInstanceId(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 38

	alpha := CompanionInfo{MobId: 1001, InstanceId: 42, Name: "Alpha"}
	beta := CompanionInfo{MobId: 1002, InstanceId: 99, Name: "Beta"}
	c.AddCompanion(alpha)
	c.AddCompanion(beta)

	found := c.GetCompanionByInstanceId(42)
	require.NotNil(t, found)
	assert.Equal(t, 42, found.InstanceId)
	assert.Equal(t, "Alpha", found.Name)

	notFound := c.GetCompanionByInstanceId(999)
	assert.Nil(t, notFound)
}

// ─── ConvictionReserve field ─────────────────────────────────────────────────

func TestCompanionInfo_ConvictionReserveField(t *testing.T) {
	c := New()
	c.Skills[string(skills.Manifestation)] = 19
	comp := CompanionInfo{MobId: 1001, InstanceId: 5, Name: "Spirit Wolf", ConvictionReserve: 333}
	require.True(t, c.AddCompanion(comp))
	assert.Equal(t, 333, c.Companions[0].ConvictionReserve)
}

// ─── CalcCompanionReserve ─────────────────────────────────────────────────────

func TestCalcCompanionReserve_Calibration(t *testing.T) {
	seedMut := mutations.SeedMutationsForTest(map[string]*mutations.MutationSpec{
		"broodmaster": {
			MutationId: "broodmaster", Name: "Broodmaster", Rarity: 5, Pole: "belief",
			Pros: []mutations.MutationEffect{{Type: "companion_reserve_reduction"}},
		},
	})
	defer seedMut()

	newbie := New()
	newbie.Skills[string(skills.Manifestation)] = 5
	assert.Equal(t, 333, newbie.CalcCompanionReserve(350)) // 350*(1-0.05)=332.5 -> 333

	meirok := New()
	meirok.Skills[string(skills.Manifestation)] = 48
	assert.Equal(t, 229, meirok.CalcCompanionReserve(440)) // 440*(1-0.48)=228.8 -> 229

	archety := New()
	archety.Skills[string(skills.Manifestation)] = 55
	archety.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 92, archety.CalcCompanionReserve(440))  // 440*0.21 = 92.4 -> 92
	assert.Equal(t, 154, archety.CalcCompanionReserve(735)) // 735*0.21 = 154.35 -> 154

	unit := New()
	unit.Skills[string(skills.Manifestation)] = 65
	unit.Mutations = map[string]int{"broodmaster": 4}
	assert.Equal(t, 154, unit.CalcCompanionReserve(735)) // reduction caps at 0.79 -> same as archetyped
}

// ─── CalcCompanionStatPool ────────────────────────────────────────────────────

func TestCalcCompanionStatPool(t *testing.T) {
	// Config defaults kick in when no config is loaded:
	//   ManifestStatScaleChaFactor  = 150  (zero-value triggers default)
	//   ManifestStatScaleSkillFactor = 0.02 (zero-value triggers default)

	tests := []struct {
		name              string
		baseStatPool      int
		charisma          int
		manifestationSkill int
		want              int
	}{
		{
			// scale = 1.0 + 100/150 + 0*0.02 = 1.667  →  120 * 1.667 = 200
			name: "wolf base cha=100 manifest=0",
			baseStatPool: 120, charisma: 100, manifestationSkill: 0,
			want: 200,
		},
		{
			// scale = 1.0 + 100/150 + 25*0.02 = 1.0 + 0.667 + 0.5 = 2.167  →  120 * 2.167 = 260
			name: "wolf base cha=100 manifest=25",
			baseStatPool: 120, charisma: 100, manifestationSkill: 25,
			want: 260,
		},
		{
			// scale = 1.667 (same as first case)  →  18 * 1.667 = 30
			name: "swarm base cha=100 manifest=0",
			baseStatPool: 18, charisma: 100, manifestationSkill: 0,
			want: 30,
		},
		{
			// scale = 1.0 + 150/150 + 50*0.02 = 1.0 + 1.0 + 1.0 = 3.0  →  120 * 3.0 = 360
			name: "wolf base cha=150 manifest=50",
			baseStatPool: 120, charisma: 150, manifestationSkill: 50,
			want: 360,
		},
		{
			// scale = 1.0 + 0/150 + 0*0.02 = 1.0  →  120 * 1.0 = 120 (no charisma boost)
			name: "cha=0 manifest=0 no boost",
			baseStatPool: 120, charisma: 0, manifestationSkill: 0,
			want: 120,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CalcCompanionStatPool(tt.baseStatPool, tt.charisma, tt.manifestationSkill)
			assert.Equal(t, tt.want, got)
		})
	}
}

// ─── CompanionInfo YAML persistence ──────────────────────────────────────────

func TestCompanionInfo_YAMLPersistence(t *testing.T) {
	original := CompanionInfo{
		MobId:      2345,
		InstanceId: 77, // should NOT survive round-trip (yaml:"-")
		SourceType: CompanionSummoned,
		Name:       "Hive Swarm",
		AutoAssist: true,
		StatTraining: map[string]int{
			"strength": 3,
		},
		Skills: map[string]int{
			"unarmed-combat": 5,
		},
		SkillUseCount: map[string]int{
			"unarmed-combat": 12,
		},
		Mutations: map[string]int{
			"carapace": 1,
		},
		SpellBook: map[string]int{
			"sparks": 1,
		},
		MutationProgress: 0.75,
	}

	data, err := yaml.Marshal(original)
	require.NoError(t, err)

	var restored CompanionInfo
	err = yaml.Unmarshal(data, &restored)
	require.NoError(t, err)

	// InstanceId must NOT survive (yaml:"-")
	assert.Equal(t, 0, restored.InstanceId, "InstanceId should not be persisted")

	// All other fields must survive
	assert.Equal(t, original.MobId, restored.MobId)
	assert.Equal(t, original.SourceType, restored.SourceType)
	assert.Equal(t, original.Name, restored.Name)
	assert.Equal(t, original.AutoAssist, restored.AutoAssist)
	assert.Equal(t, original.StatTraining, restored.StatTraining)
	assert.Equal(t, original.Skills, restored.Skills)
	assert.Equal(t, original.SkillUseCount, restored.SkillUseCount)
	assert.Equal(t, original.Mutations, restored.Mutations)
	assert.Equal(t, original.SpellBook, restored.SpellBook)
	assert.InDelta(t, original.MutationProgress, restored.MutationProgress, 1e-9)
}

// ─── CalcRaisedStatPool ──────────────────────────────────────────────────────

func TestCalcRaisedStatPool(t *testing.T) {
	// 50/50 split: companionScale * 0.5 + corpsePool * 0.5
	// Test uses hardcoded defaults (chaFactor=150, skillFactor=0.02)

	// Skeleton (base 60), Cha 100, Manifest 0, corpse pool 50
	// companionScale = CalcCompanionStatPool(60, 100, 0) = 60 * (1 + 100/150 + 0) = 60 * 1.667 = 100
	// raisedPool = 100 * 0.5 + 50 * 0.5 = 50 + 25 = 75
	companionScale := CalcCompanionStatPool(60, 100, 0)
	raisedPool := int(math.Round(float64(companionScale)*0.5 + float64(50)*0.5))
	assert.Equal(t, 75, raisedPool)

	// Wraith (base 70), Cha 150, Manifest 25, corpse pool 150
	// companionScale = 70 * (1 + 150/150 + 25*0.02) = 70 * (1 + 1.0 + 0.5) = 70 * 2.5 = 175
	// raisedPool = 175 * 0.5 + 150 * 0.5 = 87.5 + 75 = 162.5 → 163
	companionScale = CalcCompanionStatPool(70, 150, 25)
	raisedPool = int(math.Round(float64(companionScale)*0.5 + float64(150)*0.5))
	assert.Equal(t, 163, raisedPool)

	// Golem (base 120), Cha 100, Manifest 50, corpse pool 500
	// companionScale = 120 * (1 + 100/150 + 50*0.02) = 120 * (1 + 0.667 + 1.0) = 120 * 2.667 = 320
	// raisedPool = 320 * 0.5 + 500 * 0.5 = 160 + 250 = 410
	companionScale = CalcCompanionStatPool(120, 100, 50)
	raisedPool = int(math.Round(float64(companionScale)*0.5 + float64(500)*0.5))
	assert.Equal(t, 410, raisedPool)
}
