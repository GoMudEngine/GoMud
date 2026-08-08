// Package playtestprofiles materializes sanitized synthetic playtest users
// from tracked templates and an optional per-run manifest.
package playtestprofiles

// KnownTemplateIDs are the six tracked synthetic profiles.
var KnownTemplateIDs = []string{
	"fresh",
	"early",
	"mid",
	"veteran",
	"specialist-caster",
	"admin",
}

// Manifest is the ephemeral per-run materialization request.
type Manifest struct {
	Entries []ManifestEntry `yaml:"entries"`
}

// ManifestEntry selects one template, start room, and optional overlays.
type ManifestEntry struct {
	Profile   string   `yaml:"profile"`
	StartRoom int      `yaml:"start_room"`
	ActorID   string   `yaml:"actor_id,omitempty"`
	Overlays  Overlays `yaml:"overlays,omitempty"`
}

// Overlays are declarative grants/sets applied at materialize time.
type Overlays struct {
	GrantSpells    map[string]int    `yaml:"grant_spells,omitempty"`
	GrantSkills    map[string]int    `yaml:"grant_skills,omitempty"`
	GrantItems     []int             `yaml:"grant_items,omitempty"`
	Equip          map[string]int    `yaml:"equip,omitempty"`
	SetQuestTokens []string          `yaml:"set_quest_tokens,omitempty"`
	SetQuestFlags  map[string]string `yaml:"set_quest_flags,omitempty"`
	SetGold        *int              `yaml:"set_gold,omitempty"`
}

// CredsFile is the run-local credential artifact (control bind).
type CredsFile struct {
	RunID   string        `json:"run_id,omitempty"`
	Players []PlayerCreds `json:"players"`
}

// PlayerCreds is one materialized player's login material.
type PlayerCreds struct {
	Profile  string `json:"profile"`
	ActorID  string `json:"actor_id,omitempty"`
	Username string `json:"username"`
	Password string `json:"password"`
	UserID   int    `json:"user_id"`
	RoomID   int    `json:"room_id"`
}
