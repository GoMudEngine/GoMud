# playtestprofiles

## Purpose

Materializes run-scoped, AI-flagged offline players from tracked sanitized
templates plus an optional per-run manifest. Used by ephemeral playtest boots
(`Playtest.ProfilesManifest`) so 0.3c can later log in via `creds.json`.

Does **not** pick profiles from goals, drive mudagent, or call
`users.CreateUser` (that forces `RoleUser` and online maps).

## Files

- `types.go` — `KnownTemplateIDs`, `Manifest`, `Overlays`, `CredsFile`
- `manifest.go` — parse/load with KnownFields; known-template validation
- `sanitize.go` — reject passwords, inbox, identity-bearing fields, bad roles
- `template.go` — load one template YAML from `ProfilesDir`
- `overlay.go` — apply start room + declarative overlays against world checks
- `credentials.go` — generate `pt-<profile>-<suffix>` username + password
- `persist.go` — offline `GetUniqueUserId` + `SetPassword` + `SaveUser` + index
- `materialize.go` — orchestrate entries; write `creds.json`; config entrypoint
- `room_bridge.go` — `rooms.Load` existence check for default world checks
- `*_test.go`, `testdata/`, `templates_repo_test.go`

## Core types

```go
type Manifest struct {
    Entries []ManifestEntry `yaml:"entries"`
}

type ManifestEntry struct {
    Profile   string   `yaml:"profile"`
    StartRoom int      `yaml:"start_room"`
    Overlays  Overlays `yaml:"overlays,omitempty"`
}

type Overlays struct {
    GrantSpells    map[string]int    `yaml:"grant_spells,omitempty"`
    GrantSkills    map[string]int    `yaml:"grant_skills,omitempty"`
    GrantItems     []int             `yaml:"grant_items,omitempty"`
    Equip          map[string]int    `yaml:"equip,omitempty"`
    SetQuestTokens []string          `yaml:"set_quest_tokens,omitempty"`
    SetQuestFlags  map[string]string `yaml:"set_quest_flags,omitempty"`
    SetGold        *int              `yaml:"set_gold,omitempty"`
}
```

## Public API

```go
func ParseManifest(data []byte) (*Manifest, error)
func LoadManifest(path string) (*Manifest, error)
func IsKnownTemplateID(id string) bool

func LoadTemplate(profilesDir, profileID string) (*users.UserRecord, error)
func SanitizeTemplate(profileID string, u *users.UserRecord) error

func ApplyOverlays(u *users.UserRecord, startRoom int, o Overlays, world WorldChecks) error
func DefaultWorldChecks() WorldChecks

// Usernames: pt_<profile>_<suffix> (underscores; hyphens fail NameRejectRegex)
func GenerateCredentials(u *users.UserRecord, profileID string) (username, password string, err error)
func PersistOfflineUser(u *users.UserRecord) error

func Materialize(m *Manifest, opts MaterializeOptions) ([]PlayerCreds, error)
func MaterializeFromConfig() ([]PlayerCreds, error)
```

`MaterializeFromConfig` no-ops when `Playtest.ProfilesManifest` is empty.
Otherwise it fails closed (caller should exit before listeners).

## Gotchas

- **Never** use `users.CreateUser` here — offline persist only.
- Unknown YAML keys on the manifest/`overlays` object fail parse (KnownFields).
- Duplicate profile IDs in one manifest are allowed (separate users/creds).
- Creds artifact mode is `0600` where the OS supports it; Windows may map
  differently — do not assert exact mode on Windows.
- Boot logs materialized **count** only; never passwords or full creds JSON.
- Templates live at `tools/playtest/profiles/` in-repo and
  `/app/playtest/profiles` in the runner image (outside the data volume).

## Dependencies

`configs`, `users`, `characters`, `items`, `spells`, `rooms`, `yaml.v3`.

## Consumers

`main.go` (normal boot, skip copyover), `internal/playtestenv` (writes the
manifest + overrides; does not import this package for types).
