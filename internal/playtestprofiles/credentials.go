package playtestprofiles

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/users"
)

const (
	usernameMaxAttempts = 8
	passwordHexLen      = 12
)

// GenerateCredentials creates a unique username and plaintext password, hashes
// the password onto u, and returns the plaintext for the creds artifact.
func GenerateCredentials(u *users.UserRecord, profileID string) (username, password string, err error) {
	if u == nil {
		return "", "", fmt.Errorf("playtestprofiles: nil user")
	}
	// Underscores only: Validation.NameRejectRegex rejects hyphens
	// (^[a-zA-Z0-9_]+$). Spec's pt-<profile>-<suffix> shape is preserved
	// with underscores substituted for the separators.
	base := "pt_" + sanitizeProfileToken(profileID)
	for attempt := 0; attempt < usernameMaxAttempts; attempt++ {
		suffix, genErr := randomHex(3)
		if genErr != nil {
			return "", "", genErr
		}
		candidate := base + "_" + suffix
		if err := users.ValidateName(candidate); err != nil {
			continue
		}
		if idx := users.NewUserIndex(); idx.Exists() {
			if _, found := idx.FindByUsername(candidate); found {
				continue
			}
		}
		username = candidate
		break
	}
	if username == "" {
		return "", "", fmt.Errorf("playtestprofiles: could not allocate unique username for %q", profileID)
	}
	password, err = randomPassword()
	if err != nil {
		return "", "", err
	}
	u.Username = username
	if err := u.SetPassword(password); err != nil {
		return "", "", fmt.Errorf("playtestprofiles: set password: %w", err)
	}
	return username, password, nil
}

func sanitizeProfileToken(profileID string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(profileID) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
		case r == '-' || r == '_':
			b.WriteByte('_')
		}
	}
	out := b.String()
	if out == "" {
		return "profile"
	}
	return out
}

func randomHex(nBytes int) (string, error) {
	buf := make([]byte, nBytes)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("playtestprofiles: random: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func randomPassword() (string, error) {
	raw, err := randomHex(passwordHexLen / 2)
	if err != nil {
		return "", err
	}
	if len(raw) > 16 {
		raw = raw[:16]
	}
	return raw, nil
}
