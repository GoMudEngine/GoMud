package playtestenv

import (
	_ "embed"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

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
const (
	composeResolvedFileName = "compose.resolved.yml"
	configOverridesFileName = "config-overrides.yaml"
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
func requireWritableControlDir(controlDir string) error {
	probe := filepath.Join(controlDir, ".playtestenv-write-probe")
	if err := os.WriteFile(probe, []byte{}, 0o644); err != nil {
		return fmt.Errorf("%w: %s: %v", ErrControlDirNotWritable, controlDir, err)
	}
	_ = os.Remove(probe)
	return nil
}

// writeResolvedComposeFile writes the embedded Compose policy verbatim to
// <controlDir>/compose.resolved.yml and returns its absolute path. It never
// reads, merges, or otherwise derives from any Compose file already present
// in the selected checkout.
func writeResolvedComposeFile(controlDir string) (string, error) {
	path := filepath.Join(controlDir, composeResolvedFileName)
	if err := os.WriteFile(path, embeddedComposePolicy, 0o644); err != nil {
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
}

// buildConfigOverridesDoc derives the nested config-overrides.yaml document
// from ver alone: Server.CurrentVersion is ver's parsed, canonical string
// form (never the raw literal from the checkout's main.go); Network.AIPort
// always matches the Compose policy's published port; Logging.LogToFile is
// always false so a run never writes a log file into the ephemeral
// container's filesystem.
func buildConfigOverridesDoc(ver version.Version) configOverridesDoc {
	var doc configOverridesDoc
	doc.Server.CurrentVersion = ver.String()
	doc.Network.AIPort = controlAIPort
	doc.Logging.LogToFile = false
	return doc
}

// writeConfigOverrides marshals buildConfigOverridesDoc(ver) as YAML and
// writes it to <controlDir>/config-overrides.yaml, returning its absolute
// path.
func writeConfigOverrides(controlDir string, ver version.Version) (string, error) {
	data, err := yaml.Marshal(buildConfigOverridesDoc(ver))
	if err != nil {
		return "", fmt.Errorf("playtestenv: encode config overrides: %w", err)
	}
	path := filepath.Join(controlDir, configOverridesFileName)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", fmt.Errorf("playtestenv: write config overrides: %w", err)
	}
	return path, nil
}

// materializeControlFiles requires controlDir to already exist and be
// writable, then writes both the resolved Compose policy and the nested
// config overrides file into it, returning their absolute paths.
func materializeControlFiles(controlDir string, ver version.Version) (composePath, configPath string, err error) {
	if err := requireWritableControlDir(controlDir); err != nil {
		return "", "", err
	}
	composePath, err = writeResolvedComposeFile(controlDir)
	if err != nil {
		return "", "", err
	}
	configPath, err = writeConfigOverrides(controlDir, ver)
	if err != nil {
		return "", "", err
	}
	return composePath, configPath, nil
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

// composeInterpolationEnv renders v as a sorted slice of "KEY=VALUE"
// entries for the six ${DOGMUD_*} placeholders the embedded Compose policy
// references. Checkout and ControlDir are rendered with filepath.ToSlash so
// Compose's YAML interpolation never sees a Windows backslash path (a
// no-op on non-Windows platforms, where paths are already slash-separated).
func composeInterpolationEnv(v composeRunVars) []string {
	entries := map[string]string{
		"DOGMUD_RUN_ID":               v.RunID,
		"DOGMUD_PROJECT":              v.Project,
		"DOGMUD_CHECKOUT":             filepath.ToSlash(v.Checkout),
		"DOGMUD_CHECKOUT_FINGERPRINT": v.CheckoutFingerprint,
		"DOGMUD_CREATED_AT":           v.CreatedAt.UTC().Format(time.RFC3339),
		"DOGMUD_CONTROL_DIR":          filepath.ToSlash(v.ControlDir),
	}
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	out := make([]string, 0, len(keys))
	for _, k := range keys {
		out = append(out, k+"="+entries[k])
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
// then replaces its Env with dc's already-validated scrubbed environment
// plus composeInterpolationEnv(vars) - never a fresh, unscrubbed ambient
// environment, and never any variable beyond the six ${DOGMUD_*}
// placeholders the embedded policy references.
func composeCommand(dc dockerContext, vars composeRunVars, composeFile string, extra []string, dir string, stdout, stderr io.Writer) CommandSpec {
	spec := dockerCommand(dc, composeArgs(vars.Checkout, composeFile, vars.Project, extra), dir, stdout, stderr)
	spec.Env = append(cloneEnv(dc.env), composeInterpolationEnv(vars)...)
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
