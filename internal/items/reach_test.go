package items

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDefaultReachForSubtype_KnownSubtypes(t *testing.T) {
	cases := map[ItemSubType]float64{
		Fist:        0.1,
		Claws:       0.15,
		Bite:        0.15,
		Sting:       0.2,
		Slam:        0.3,
		Gore:        0.4,
		Whipping:    0.5,
		Stabbing:    0.3,
		Slashing:    1.0,
		Cleaving:    0.9,
		Bludgeoning: 0.8,
		Shooting:    1.0,
		Wand:        0.4,
		Sceptre:     0.6,
		Staff:       1.5,
	}
	for subtype, want := range cases {
		assert.Equal(t, want, DefaultReachForSubtype(subtype),
			"subtype %s", subtype)
	}
}

func TestDefaultReachForSubtype_UnknownReturnsZero(t *testing.T) {
	assert.Equal(t, 0.0, DefaultReachForSubtype(BlobContent))
	assert.Equal(t, 0.0, DefaultReachForSubtype(ItemSubType("nonexistent")))
}

func TestResolveReach_ExplicitOverridesSubtypeDefault(t *testing.T) {
	spec := &ItemSpec{Subtype: Slashing, Reach: 0.7} // shorter than 1.0 default
	assert.Equal(t, 0.7, ResolveReach(spec))
}

func TestResolveReach_ZeroFallsThroughToSubtypeDefault(t *testing.T) {
	spec := &ItemSpec{Subtype: Slashing, Reach: 0}
	assert.Equal(t, 1.0, ResolveReach(spec))
}

func TestResolveReach_NilSafe(t *testing.T) {
	assert.Equal(t, 0.0, ResolveReach(nil))
}

func TestResolveNaturalReach_DelegatesToDefault(t *testing.T) {
	assert.Equal(t, 0.15, ResolveNaturalReach(Claws))
	assert.Equal(t, 0.4, ResolveNaturalReach(Gore))
}
