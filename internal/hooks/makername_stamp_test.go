package hooks

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/items"
)

// ShouldStampMakerName is the extracted predicate for the craft-completion
// MakerName stamping in UserRoundTick. The full async completion path sits
// behind a non-deterministic success roll (util.Rand vs CalcSuccessChance,
// capped below 100%), so the condition is tested here as a pure function;
// the call site is trivial wiring.
func TestShouldStampMakerName(t *testing.T) {
	tests := []struct {
		name       string
		craftSkill int
		spec       items.ItemSpec
		want       bool
	}{
		{
			name:       "skilled crafter, equipment output stamps",
			craftSkill: 30,
			spec:       items.ItemSpec{Type: items.Weapon},
			want:       true,
		},
		{
			name:       "skilled crafter, plain Object output does NOT stamp",
			craftSkill: 30,
			spec:       items.ItemSpec{Type: items.Object},
			want:       false,
		},
		{
			name:       "skilled crafter, Object-typed COMPONENT output stamps (require_own_components provenance)",
			craftSkill: 30,
			spec:       items.ItemSpec{Type: items.Object, IsComponent: true},
			want:       true,
		},
		{
			name:       "skilled crafter, non-Object component also stamps",
			craftSkill: 30,
			spec:       items.ItemSpec{Type: items.Gemstone, IsComponent: true},
			want:       true,
		},
		{
			name:       "unskilled crafter never stamps, even components",
			craftSkill: 29,
			spec:       items.ItemSpec{Type: items.Object, IsComponent: true},
			want:       false,
		},
		{
			name:       "unskilled crafter, equipment does not stamp",
			craftSkill: 0,
			spec:       items.ItemSpec{Type: items.Weapon},
			want:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ShouldStampMakerName(tt.craftSkill, tt.spec); got != tt.want {
				t.Errorf("ShouldStampMakerName(%d, Type=%s IsComponent=%v) = %v, want %v",
					tt.craftSkill, tt.spec.Type, tt.spec.IsComponent, got, tt.want)
			}
		})
	}
}
