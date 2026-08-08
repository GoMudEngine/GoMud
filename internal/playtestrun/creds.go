package playtestrun

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/playtestprofiles"
)

// SelectCredsPlayer returns the username/password for the unique player
// matching profileID in a playtestenv creds.json artifact. Errors never
// include the password.
func SelectCredsPlayer(credsPath, profileID string) (username, password string, err error) {
	profileID = strings.TrimSpace(profileID)
	if profileID == "" {
		return "", "", fmt.Errorf("playtestrun: profile id is required to select creds")
	}
	raw, err := os.ReadFile(credsPath)
	if err != nil {
		return "", "", fmt.Errorf("playtestrun: read creds %s: %w", credsPath, err)
	}
	var file playtestprofiles.CredsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", "", fmt.Errorf("playtestrun: parse creds %s: %w", credsPath, err)
	}

	var matches []playtestprofiles.PlayerCreds
	for _, p := range file.Players {
		if p.Profile == profileID {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("playtestrun: no creds player for profile %q in %s", profileID, credsPath)
	case 1:
		return matches[0].Username, matches[0].Password, nil
	default:
		return "", "", fmt.Errorf("playtestrun: ambiguous creds: %d players for profile %q in %s", len(matches), profileID, credsPath)
	}
}

// SelectCredsByActorID returns username/password for the unique player matching
// actorID. Multi-agent login must use this helper, never profile-only selection.
// Errors never include the password.
func SelectCredsByActorID(credsPath, actorID string) (username, password string, err error) {
	actorID = strings.TrimSpace(actorID)
	if actorID == "" {
		return "", "", fmt.Errorf("playtestrun: actor id is required to select creds")
	}
	raw, err := os.ReadFile(credsPath)
	if err != nil {
		return "", "", fmt.Errorf("playtestrun: read creds %s: %w", credsPath, err)
	}
	var file playtestprofiles.CredsFile
	if err := json.Unmarshal(raw, &file); err != nil {
		return "", "", fmt.Errorf("playtestrun: parse creds %s: %w", credsPath, err)
	}

	var matches []playtestprofiles.PlayerCreds
	for _, p := range file.Players {
		if p.ActorID == actorID {
			matches = append(matches, p)
		}
	}
	switch len(matches) {
	case 0:
		return "", "", fmt.Errorf("playtestrun: no creds player for actor_id %q in %s", actorID, credsPath)
	case 1:
		return matches[0].Username, matches[0].Password, nil
	default:
		return "", "", fmt.Errorf("playtestrun: ambiguous creds: %d players for actor_id %q in %s", len(matches), actorID, credsPath)
	}
}
