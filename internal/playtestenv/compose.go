package playtestenv

import (
	"bytes"
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/natefinch/atomic"
	"gopkg.in/yaml.v3"

	"github.com/GoMudEngine/GoMud/internal/version"
)

// embeddedComposePolicy is the trusted, compiled-in Compose policy. It is
// never read from a selected checkout - the checkout's own Compose file (if
// any) is never opened by this package - and it is never mutated: every
// caller that needs a copy goes through EmbeddedComposePolicy, which
// defensively clones the backing array.
//
//go:embed compose.playtest.yml
var embeddedComposePolicy []byte

// composeResolvedFileName and configOverridesFileName are the basenames of
// the two files materialized into a run's control directory: the verbatim
// embedded Compose policy, and this package's nested config-overrides.yaml
// consumed by the containerized server via CONFIG_PATH.
//
// profilesManifestFileName / credsFileName live in the same writable control
// bind (/run/dogmud). The supervisor writes the manifest; the server writes
// creds.json after materialization.
const (
	composeResolvedFileName  = "compose.resolved.yml"
	configOverridesFileName  = "config-overrides.yaml"
	profilesManifestFileName = "profiles-manifest.yaml"
	credsFileName            = "creds.json"

	containerProfilesDir      = "/app/playtest/profiles"
	containerProfilesManifest = "/run/dogmud/profiles-manifest.yaml"
)

// controlAIPort is the single AI-listener port this package's Compose
// policy publishes (see compose.playtest.yml's "target: 55555") and the
// port written into the materialized config-overrides.yaml's
// Network.AIPort. It is a compile-time constant, not derived from any
// external input, because the embedded Compose policy hard-codes the same
// value.
const controlAIPort = 55555

// ErrControlDirNotWritable is returned when a run's control directory
// cannot be written to.
var ErrControlDirNotWritable = errors.New("playtestenv: run control directory is not writable")

// EmbeddedComposePolicy returns a copy of the embedded compose.playtest.yml
// bytes. It is exported only for test/inspection use; production
// materialization uses the package-private embeddedComposePolicy directly
// so no caller can mutate the shared embedded array.
func EmbeddedComposePolicy() []byte {
	out := make([]byte, len(embeddedComposePolicy))
	copy(out, embeddedComposePolicy)
	return out
}

// requireWritableControlDir proves controlDir is writable by creating and
// immediately removing a uniquely-named probe file inside it. It never
// leaves the probe file behind, whether the write succeeds or fails.
//
// The control directory - and therefore its /run/dogmud bind mount inside
// the container (see compose.playtest.yml) - is deliberately never
// read-only. internal/configs.SetVal, the live-admin config-write path a
// running server uses for both migration completion and later
// playtest-scoped config changes, persists via util.SafeSave: it writes
// "<CONFIG_PATH>.new" and then os.Renames it over the original
// config-overrides.yaml in the same directory. Both the temp-file create
// and the rename require the directory itself to be writable, not just the
// target file; a read-only bind would make every one of those legitimate,
// in-container config writes fail. The bind stays scoped to exactly this
// ignored per-run control directory - never the checkout source tree and
// never the run manifest - so this writability requirement never exposes
// anything beyond one run's own disposable control files.
func requireWritableControlDir(controlDir string) error {
	probe := filepath.Join(controlDir, ".playtestenv-write-probe")
	if err := os.WriteFile(probe, []byte{}, 0o644); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrControlDirNotWritable, controlDir, err)
	}
	_ = os.Remove(probe)
	return nil
}

// writeResolvedComposeFile writes the embedded Compose policy verbatim to
// <runDir>/compose.resolved.yml and returns its absolute path. The file
// lives at the run root (sibling to control/), never inside the writable
// control bind, matching the approved artifact layout. It never reads,
// merges, or otherwise derives from any Compose file already present in
// the selected checkout.
//
// It writes via natefinch/atomic.WriteFile (write-to-temporary-file-plus-
// atomic-rename, the same mechanism manifest.go's writeManifest uses), so a
// process interrupted mid-write leaves either the complete new file or the
// untouched prior one - never a truncated/corrupt file - and never a
// leftover temporary file in the run directory either way.
func writeResolvedComposeFile(runDir string) (string, error) {
	path := filepath.Join(runDir, composeResolvedFileName)
	if err := atomic.WriteFile(path, bytes.NewReader(embeddedComposePolicy)); err != nil {
		return "", fmt.Errorf("playtestenv: write resolved compose file: %w", err)
	}
	return path, nil
}

// configOverridesDoc is the nested-YAML shape consumed by
// internal/configs.ReloadConfig's CONFIG_PATH override mechanism: each
// top-level key names a Config subsection, matching config.yaml's own
// layout, so the loader's key-path matching resolves them without any
// dotted-key rewriting.
type configOverridesDoc struct {
	Server struct {
		CurrentVersion string `yaml:"CurrentVersion"`
	} `yaml:"Server"`
	Network struct {
		AIPort int `yaml:"AIPort"`
	} `yaml:"Network"`
	Logging struct {
		LogToFile bool `yaml:"LogToFile"`
	} `yaml:"Logging"`
	Playtest *playtestOverrides `yaml:"Playtest,omitempty"`
}

// playtestOverrides gate synthetic profile materialization inside the
// container. Omitted entirely for creation-flow runs (empty Profiles list).
type playtestOverrides struct {
	ProfilesDir      string `yaml:"ProfilesDir"`
	ProfilesManifest string `yaml:"ProfilesManifest"`
}

// profilesManifestDoc is the YAML shape written to control/profiles-manifest.yaml.
type profilesManifestDoc struct {
	Entries []ProfileRequest `yaml:"entries"`
}

// buildConfigOverridesDoc derives the nested config-overrides.yaml document
// from ver and whether profiles were requested: Server.CurrentVersion is
// ver's parsed, canonical string form (never the raw literal from the
// checkout's main.go); Network.AIPort always matches the Compose policy's
// published port; Logging.LogToFile is always false so a run never writes a
// log file into the ephemeral container's filesystem. Playtest overrides are
// set only when withProfiles is true.
func buildConfigOverridesDoc(ver version.Version, withProfiles bool) configOverridesDoc {
	var doc configOverridesDoc
	doc.Server.CurrentVersion = ver.String()
	doc.Network.AIPort = controlAIPort
	doc.Logging.LogToFile = false
	if withProfiles {
		doc.Playtest = &playtestOverrides{
			ProfilesDir:      containerProfilesDir,
			ProfilesManifest: containerProfilesManifest,
		}
	}
	return doc
}

// writeConfigOverrides marshals buildConfigOverridesDoc and writes it to
// <controlDir>/config-overrides.yaml, returning its absolute path. Like
// writeResolvedComposeFile, it writes via atomic.WriteFile so a
// re-materialization (e.g. on run renewal) always either fully replaces the
// prior file or leaves it untouched, and never leaks a temporary file.
func writeConfigOverrides(controlDir string, ver version.Version, withProfiles bool) (string, error) {
	data, err := yaml.Marshal(buildConfigOverridesDoc(ver, withProfiles))
	if err != nil {
		return "", fmt.Errorf("playtestenv: encode config overrides: %w", err)
	}
	path := filepath.Join(controlDir, configOverridesFileName)
	if err := atomic.WriteFile(path, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("playtestenv: write config overrides: %w", err)
	}
	return path, nil
}

// writeProfilesManifest marshals profiles into
// <controlDir>/profiles-manifest.yaml. Call only when profiles is non-empty.
func writeProfilesManifest(controlDir string, profiles []ProfileRequest) (string, error) {
	if len(profiles) == 0 {
		return "", fmt.Errorf("playtestenv: write profiles manifest requires at least one profile")
	}
	for i, p := range profiles {
		if strings.TrimSpace(p.Profile) == "" {
			return "", fmt.Errorf("playtestenv: profiles[%d]: profile id is required", i)
		}
		if p.StartRoom <= 0 {
			return "", fmt.Errorf("playtestenv: profiles[%d]: start_room must be positive", i)
		}
	}
	data, err := yaml.Marshal(profilesManifestDoc{Entries: profiles})
	if err != nil {
		return "", fmt.Errorf("playtestenv: encode profiles manifest: %w", err)
	}
	path := filepath.Join(controlDir, profilesManifestFileName)
	if err := atomic.WriteFile(path, bytes.NewReader(data)); err != nil {
		return "", fmt.Errorf("playtestenv: write profiles manifest: %w", err)
	}
	return path, nil
}

// materializeRunFiles requires controlDir to already exist and be writable,
// then writes the resolved Compose policy to <runDir>/compose.resolved.yml
// and the nested config overrides to <controlDir>/config-overrides.yaml.
// When profiles is non-empty it also writes profiles-manifest.yaml and sets
// Playtest overrides. Only control files live inside the writable control
// bind; compose.resolved.yml stays at the run root.
func materializeRunFiles(runDir, controlDir string, ver version.Version, profiles []ProfileRequest) (composePath, configPath, profilesManifestPath string, err error) {
	if err := requireWritableControlDir(controlDir); err != nil {
		return "", "", "", err
	}
	composePath, err = writeResolvedComposeFile(runDir)
	if err != nil {
		return "", "", "", err
	}
	withProfiles := len(profiles) > 0
	configPath, err = writeConfigOverrides(controlDir, ver, withProfiles)
	if err != nil {
		return "", "", "", err
	}
	if withProfiles {
		profilesManifestPath, err = writeProfilesManifest(controlDir, profiles)
		if err != nil {
			return "", "", "", err
		}
	}
	return composePath, configPath, profilesManifestPath, nil
}

// composeRunVars is the exact, validated set of values this package ever
// substitutes into the embedded Compose policy's ${DOGMUD_*} placeholders.
// Every field must come from an already-validated manifest, checkout, or
// control-directory value - never from raw ambient environment or from any
// content read out of the selected checkout.
type composeRunVars struct {
	RunID               string
	Project             string
	Checkout            string
	CheckoutFingerprint string
	CreatedAt           time.Time
	ControlDir          string
}

// composeReservedEnvNames lists the six ${DOGMUD_*} interpolation keys this
// package ever sets in a Compose invocation's environment. It is the single
// source of truth for both composeInterpolationEnv (which supplies their
// trusted values) and scrubComposeReservedEnv (which strips any ambient
// collision before those trusted values are appended), so the two can never
// drift out of sync.
var composeReservedEnvNames = []string{
	"DOGMUD_RUN_ID",
	"DOGMUD_PROJECT",
	"DOGMUD_CHECKOUT",
	"DOGMUD_CHECKOUT_FINGERPRINT",
	"DOGMUD_CREATED_AT",
	"DOGMUD_CONTROL_DIR",
}

// isComposeReservedEnvName reports whether key matches one of
// composeReservedEnvNames. The comparison is case-insensitive on every
// platform: these six names are this package's own interpolation
// convention, not OS-level environment-variable semantics (contrast
// envNamesEqual in docker.go, which matches per-platform for genuine
// ambient Docker variables), so a mixed-case ambient collision must be
// scrubbed identically on Windows and POSIX alike.
func isComposeReservedEnvName(key string) bool {
	for _, reserved := range composeReservedEnvNames {
		if strings.EqualFold(key, reserved) {
			return true
		}
	}
	return false
}

// scrubComposeReservedEnv returns a copy of env with every entry whose key
// matches a reserved ${DOGMUD_*} interpolation name removed,
// case-insensitively. It preserves the nil-vs-non-nil-empty distinction
// documented on cloneEnv: scrubbing a nil environment yields nil, and
// scrubbing a non-nil (even already-empty) environment always yields a
// non-nil slice, so this scrub can never masquerade as "no Env override"
// and reinstate the real host environment.
func scrubComposeReservedEnv(env []string) []string {
	if env == nil {
		return nil
	}
	scrubbed := make([]string, 0, len(env))
	for _, kv := range env {
		if isComposeReservedEnvName(envKey(kv)) {
			continue
		}
		scrubbed = append(scrubbed, kv)
	}
	return scrubbed
}

// composeInterpolationEnv renders v as a sorted slice of "KEY=VALUE"
// entries for the six ${DOGMUD_*} placeholders the embedded Compose policy
// references. Checkout and ControlDir are rendered with filepath.ToSlash so
// Compose's YAML interpolation never sees a Windows backslash path (a
// no-op on non-Windows platforms, where paths are already slash-separated).
func composeInterpolationEnv(v composeRunVars) []string {
	values := map[string]string{
		"DOGMUD_RUN_ID":               v.RunID,
		"DOGMUD_PROJECT":              v.Project,
		"DOGMUD_CHECKOUT":             filepath.ToSlash(v.Checkout),
		"DOGMUD_CHECKOUT_FINGERPRINT": v.CheckoutFingerprint,
		"DOGMUD_CREATED_AT":           v.CreatedAt.UTC().Format(time.RFC3339),
		"DOGMUD_CONTROL_DIR":          filepath.ToSlash(v.ControlDir),
	}
	keys := append([]string(nil), composeReservedEnvNames...)
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+values[k])
	}
	return out
}

// composeArgs builds the shared argv prefix every Compose invocation from
// this package must use: "compose --project-directory <checkout> -f
// <composeFile> -p <project>", followed by extra (the specific
// subcommand, e.g. "config" or "up"/"-d"). It is never joined into a shell
// command line - args stay a plain slice throughout.
func composeArgs(checkout, composeFile, project string, extra []string) []string {
	full := make([]string, 0, 6+len(extra))
	full = append(full, "compose", "--project-directory", checkout, "-f", composeFile, "-p", project)
	full = append(full, extra...)
	return full
}

// composeCommand builds one Compose CommandSpec via dockerCommand(dc, ...),
// then replaces its Env with dc's already-validated scrubbed environment -
// itself further scrubbed of any of the six reserved ${DOGMUD_*}
// interpolation names, case-insensitively - plus composeInterpolationEnv(vars).
// This guarantees exactly one value per logical ${DOGMUD_*} key reaches
// Compose: never a fresh, unscrubbed ambient environment, never a
// duplicate/shadowed entry from an ambient collision, and never any
// variable beyond the six placeholders the embedded policy references.
func composeCommand(dc dockerContext, vars composeRunVars, composeFile string, extra []string, dir string, stdout, stderr io.Writer) CommandSpec {
	spec := dockerCommand(dc, composeArgs(vars.Checkout, composeFile, vars.Project, extra), dir, stdout, stderr)
	spec.Env = append(scrubComposeReservedEnv(dc.env), composeInterpolationEnv(vars)...)
	return spec
}

// composeConfigCommand builds `docker --context <dc> compose
// --project-directory <checkout> -f <composeFile> -p <project> config`, the
// read-only rendering/validation invocation used to confirm the resolved
// Compose policy without starting anything.
func composeConfigCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"config"}, dir, stdout, stderr)
}

// composeUpCommand builds the detached "up" invocation a future lifecycle
// task can call to start a run's container. It performs no lifecycle
// action itself.
func composeUpCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"up", "-d"}, dir, stdout, stderr)
}

// composeBuildCommand builds `compose ... build server` for the run's
// image. It performs no lifecycle action itself.
func composeBuildCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"build", "server"}, dir, stdout, stderr)
}

// composeUpNoBuildServerCommand builds the narrow startup invocation
// `compose ... up -d --no-build server` used after an explicit build step.
// It leaves composeUpCommand's broader `up -d` helper unchanged for Task 3
// callers/tests.
func composeUpNoBuildServerCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"up", "-d", "--no-build", "server"}, dir, stdout, stderr)
}

// composeDownCommand builds the "down" invocation (removing containers,
// networks, and the named data volume) a future lifecycle task can call to
// tear down a run. It performs no lifecycle action itself.
func composeDownCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"down", "--volumes", "--remove-orphans"}, dir, stdout, stderr)
}

// composeLogsCommand builds the "logs" invocation a future lifecycle task
// can call to retrieve a run's container logs.
func composeLogsCommand(dc dockerContext, vars composeRunVars, composeFile, dir string, stdout, stderr io.Writer) CommandSpec {
	return composeCommand(dc, vars, composeFile, []string{"logs", "--no-color"}, dir, stdout, stderr)
}
