// Package playtestenv implements the local-only ephemeral Docker playtest
// supervisor. It turns one selected DOGMud checkout into one isolated,
// lease-bound, verified Docker server endpoint and cleans up only that run's
// resources.
package playtestenv

import (
	"io"
	"time"
)

// State is a run's lifecycle state.
type State string

const (
	StateValidating State = "validating"
	StateBuilding   State = "building"
	StateStarting   State = "starting"
	StateReady      State = "ready"
	StateStopping   State = "stopping"
	StateStopped    State = "stopped"
	StateFailed     State = "failed"
)

// Endpoint is a loopback-only Docker-assigned AI port publication.
type Endpoint struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

// FailureCategory classifies a non-secret startup or lifecycle failure.
type FailureCategory string

// Exact failure-category string values persisted in manifests, reports, and
// machine-readable Result.Failure records.
const (
	FailureInvalidCheckout   FailureCategory = "invalid_checkout"
	FailureDockerUnavailable FailureCategory = "docker_unavailable"
	FailureBuild             FailureCategory = "build_failure"
	FailureContainerExited   FailureCategory = "container_exited"
	FailureBootPanic         FailureCategory = "boot_panic"
	FailureListenerCreation  FailureCategory = "listener_creation_failure"
	FailurePortPublication   FailureCategory = "port_publication_failure"
	FailureNonLoopback       FailureCategory = "non_loopback_publication"
	FailureReadinessTimeout  FailureCategory = "readiness_timeout"
	FailureConnectionProbe   FailureCategory = "connection_probe_failure"
	FailureManifest          FailureCategory = "manifest_failure"
	FailureCleanup           FailureCategory = "cleanup_failure"
	FailureLockBusy          FailureCategory = "lock_busy"
	FailureAbandonedRun      FailureCategory = "abandoned_run"
	FailureIdentityMismatch  FailureCategory = "identity_mismatch"
)

// FailureRecord is the structured, non-secret evidence of a run failure.
type FailureRecord struct {
	Category  FailureCategory `json:"category"`
	Phase     State           `json:"phase"`
	Summary   string          `json:"summary"`
	Retryable bool            `json:"retryable,omitempty"`
}

// ResourceRef identifies one Docker resource by kind and ID for cleanup
// reporting.
type ResourceRef struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// CleanupResult reports whether a run's resources were fully removed.
type CleanupResult struct {
	Complete  bool          `json:"complete"`
	Leftovers []ResourceRef `json:"leftovers,omitempty"`
	Summary   string        `json:"summary,omitempty"`
}

// ReadinessObservation is the compound evidence collected while waiting for a
// run to become ready.
type ReadinessObservation struct {
	ContainerRunning bool      `json:"container_running"`
	ServerReady      bool      `json:"server_ready"`
	PanicSeen        bool      `json:"panic_seen"`
	ListenerError    bool      `json:"listener_error"`
	PortMappings     int       `json:"port_mappings"`
	Endpoint         *Endpoint `json:"endpoint,omitempty"`
	TCPConnected     bool      `json:"tcp_connected"`
	ObservedAt       time.Time `json:"observed_at"`
}

// ArtifactPaths records the run-scoped files a caller may inspect.
type ArtifactPaths struct {
	Manifest  string `json:"manifest"`
	BuildLog  string `json:"build_log"`
	ServerLog string `json:"server_log"`
	Inspect   string `json:"inspect,omitempty"`
	Compose   string `json:"compose"`
	Config    string `json:"config"`
	Creds     string `json:"creds,omitempty"`
	Report    string `json:"report,omitempty"`
}

// ProfileRequest is one explicit synthetic-profile materialization request.
// Overlay YAML keys match internal/playtestprofiles.Overlays; unknown keys are
// rejected by the server materializer (KnownFields), not by this supervisor.
type ProfileRequest struct {
	Profile   string          `yaml:"profile" json:"profile"`
	StartRoom int             `yaml:"start_room" json:"start_room"`
	Overlays  ProfileOverlays `yaml:"overlays,omitempty" json:"overlays,omitempty"`
}

// ProfileOverlays are declarative grants/sets applied at materialize time.
type ProfileOverlays struct {
	GrantSpells    map[string]int    `yaml:"grant_spells,omitempty" json:"grant_spells,omitempty"`
	GrantSkills    map[string]int    `yaml:"grant_skills,omitempty" json:"grant_skills,omitempty"`
	GrantItems     []int             `yaml:"grant_items,omitempty" json:"grant_items,omitempty"`
	Equip          map[string]int    `yaml:"equip,omitempty" json:"equip,omitempty"`
	SetQuestTokens []string          `yaml:"set_quest_tokens,omitempty" json:"set_quest_tokens,omitempty"`
	SetQuestFlags  map[string]string `yaml:"set_quest_flags,omitempty" json:"set_quest_flags,omitempty"`
	SetGold        *int              `yaml:"set_gold,omitempty" json:"set_gold,omitempty"`
}

// GitEntry is one path/status record from a machine-readable Git status line.
type GitEntry struct {
	Status   string `json:"status"`
	Path     string `json:"path"`
	OrigPath string `json:"orig_path,omitempty"`
}

// GitBaseline is the pre-run, metadata-only Git snapshot of the selected
// checkout.
type GitBaseline struct {
	Commit  string     `json:"commit,omitempty"`
	Entries []GitEntry `json:"entries,omitempty"`
}

// StartOptions configures a new ephemeral run.
type StartOptions struct {
	Checkout         string
	Lease            time.Duration
	ReadinessTimeout time.Duration
	// Profiles, when non-empty, writes control/profiles-manifest.yaml and
	// sets Playtest config overrides so the container materializes those
	// users before listeners. Empty/omitted means creation-flow (no-op).
	Profiles []ProfileRequest
}

// RunOptions identifies an existing run for status, renew, or stop.
type RunOptions struct {
	Checkout string
	RunID    string
}

// LogsOptions configures a log-retrieval or log-follow operation.
type LogsOptions struct {
	Checkout string
	RunID    string
	Follow   bool
	Output   io.Writer
}

// RenewOptions extends a run's lease.
type RenewOptions struct {
	Checkout string
	RunID    string
	Lease    time.Duration
}

// Result is the stable machine-readable outcome of any supervisor operation.
type Result struct {
	Operation string         `json:"operation"`
	RunID     string         `json:"run_id,omitempty"`
	Project   string         `json:"project,omitempty"`
	State     State          `json:"state,omitempty"`
	Endpoint  *Endpoint      `json:"endpoint,omitempty"`
	Manifest  string         `json:"manifest,omitempty"`
	ServerLog string         `json:"server_log,omitempty"`
	Report    string         `json:"report,omitempty"`
	Artifacts *ArtifactPaths `json:"artifacts,omitempty"`
	Cleanup   *CleanupResult `json:"cleanup,omitempty"`
	Failure   *FailureRecord `json:"failure,omitempty"`
}
