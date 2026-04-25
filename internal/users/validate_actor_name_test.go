package users

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mobs"
)

// setupValidationConfig ensures NameSizeMin/Max have sane bounds.
// The default zero-value config rejects every non-empty name.
func setupValidationConfig(t *testing.T) {
	t.Helper()
	_ = configs.AddOverlayOverrides(map[string]any{
		"Validation.NameSizeMin": 1,
		"Validation.NameSizeMax": 20,
	})
}

// seedOfflineUser injects a UserRecord into the in-memory userManager so that
// ValidateActorName's username-collision check finds it. The returned cleanup
// function removes the seeded entry; call it (or defer it) in your test.
//
// Note: We seed via userManager.Usernames (keyed lowercase) and userManager.Users
// so that both the username check (line 57 of validate_actor_name.go) and any
// calls to GetByUserId succeed.
func seedOfflineUser(t *testing.T, userId int, charName string) func() {
	t.Helper()

	usernameLower := strings.ToLower(charName)

	u := &UserRecord{
		UserId:   userId,
		Username: usernameLower,
		Role:     RoleUser,
		Character: &characters.Character{
			Name: charName,
		},
		// No connectionId → treated as offline (not in Connections map)
	}

	userManager.mu.Lock()
	// Save originals so we can restore on cleanup
	origUser, hadUser := userManager.Users[userId]
	origId, hadUsername := userManager.Usernames[usernameLower]

	userManager.Users[userId] = u
	userManager.Usernames[usernameLower] = userId
	userManager.mu.Unlock()

	return func() {
		userManager.mu.Lock()
		if hadUser {
			userManager.Users[userId] = origUser
		} else {
			delete(userManager.Users, userId)
		}
		if hadUsername {
			userManager.Usernames[usernameLower] = origId
		} else {
			delete(userManager.Usernames, usernameLower)
		}
		userManager.mu.Unlock()
	}
}

// seedMobName injects a single mob template name into mobs.allMobNames via the
// exported SeedMobsForTest helper. Returns a cleanup function.
func seedMobName(t *testing.T, name string) func() {
	t.Helper()

	spec := map[int]*mobs.Mob{
		9999: {
			Character: characters.Character{Name: name},
		},
	}
	return mobs.SeedMobsForTest(spec, map[int]*mobs.Mob{})
}

func TestValidateActorName(t *testing.T) {
	setupValidationConfig(t)


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

func TestValidateActorName_PlayerCollision(t *testing.T) {
	setupValidationConfig(t)

	cleanup := seedOfflineUser(t, 42, "Calabean")
	defer cleanup()

	if err := ValidateActorName("Calabean", ValidateActorOpts{}); err == nil {
		t.Fatal("expected collision error, got nil")
	}

	if err := ValidateActorName("Calabean", ValidateActorOpts{ExcludeUserId: 42}); err != nil {
		t.Fatalf("expected ExcludeUserId to bypass own-name collision, got %v", err)
	}
}

func TestValidateActorName_MobCollision(t *testing.T) {
	setupValidationConfig(t)

	cleanup := seedMobName(t, "Goblin")
	defer cleanup()

	if err := ValidateActorName("Goblin", ValidateActorOpts{}); err == nil {
		t.Fatal("expected mob collision error, got nil")
	}

	if err := ValidateActorName("Goblin", ValidateActorOpts{SkipMobCheck: true}); err != nil {
		t.Fatalf("SkipMobCheck should bypass mob collision, got %v", err)
	}
}
