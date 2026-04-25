package users

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

func TestValidateActorName(t *testing.T) {
	// Ensure validation config has sane bounds — default zero-value fails
	// all non-empty names because NameSizeMax == 0.
	_ = configs.AddOverlayOverrides(map[string]any{
		"Validation.NameSizeMin": 1,
		"Validation.NameSizeMax": 20,
	})


	tests := []struct {
		name      string
		input     string
		opts      ValidateActorOpts
		wantErr   bool
		errSubstr string
	}{
		{name: "empty", input: "", wantErr: true, errSubstr: "between"},
		{name: "valid_novel_name", input: "Bobblesworth", wantErr: false},
		{name: "skip_mob_check_passes", input: "Bobblesworth", opts: ValidateActorOpts{SkipMobCheck: true}, wantErr: false},
		{name: "skip_banned_check_passes", input: "Bobblesworth", opts: ValidateActorOpts{SkipBannedCheck: true}, wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateActorName(tt.input, tt.opts)
			if tt.wantErr && err == nil {
				t.Fatalf("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("expected no error, got %v", err)
			}
			if tt.wantErr && tt.errSubstr != "" && !strings.Contains(err.Error(), tt.errSubstr) {
				t.Fatalf("expected error to contain %q, got %v", tt.errSubstr, err)
			}
		})
	}
}
