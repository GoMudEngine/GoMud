package playtestenv

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/GoMudEngine/GoMud/internal/version"
)

// decodeComposePolicy fully decodes raw as a generic YAML document (map
// keys as strings, aliases resolved), so tests can assert both the
// presence of every required field and the absence of any field this
// package's trusted policy must never introduce.
func decodeComposePolicy(t *testing.T, raw []byte) map[string]any {
	t.Helper()
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &doc))
	return doc
}

func mapAt(t *testing.T, doc map[string]any, key string) map[string]any {
	t.Helper()
	v, ok := doc[key]
	require.True(t, ok, "missing key %q", key)
	m, ok := v.(map[string]any)
	require.True(t, ok, "key %q is not a mapping: %T", key, v)
	return m
}

func requireExactKeys(t *testing.T, m map[string]any, want ...string) {
	t.Helper()
	got := make([]string, 0, len(m))
	for k := range m {
		got = append(got, k)
	}
	require.ElementsMatch(t, want, got)
}

func TestEmbeddedComposePolicy(t *testing.T) {
	doc := decodeComposePolicy(t, EmbeddedComposePolicy())

	requireExactKeys(t, doc, "services", "volumes", "networks")

	services := mapAt(t, doc, "services")
	requireExactKeys(t, services, "server")
	server := mapAt(t, services, "server")

	requireExactKeys(t, server, "image", "build", "labels", "restart", "entrypoint", "environment", "volumes", "ports")

	require.Equal(t, "dogmud-playtest:${DOGMUD_RUN_ID}", server["image"])

	build := mapAt(t, server, "build")
	requireExactKeys(t, build, "context", "dockerfile", "target", "labels")
	require.Equal(t, "${DOGMUD_CHECKOUT}", build["context"])
	require.Equal(t, "provisioning/Dockerfile", build["dockerfile"])
	require.Equal(t, "runner", build["target"])

	wantLabels := map[string]any{
		"dogmud.playtest.managed":    "true",
		"dogmud.playtest.run-id":     "${DOGMUD_RUN_ID}",
		"dogmud.playtest.project":    "${DOGMUD_PROJECT}",
		"dogmud.playtest.checkout":   "${DOGMUD_CHECKOUT_FINGERPRINT}",
		"dogmud.playtest.schema":     "1",
		"dogmud.playtest.created-at": "${DOGMUD_CREATED_AT}",
	}
	require.Equal(t, wantLabels, build["labels"], "build.labels must be the shared immutable label set")
	require.Equal(t, wantLabels, server["labels"], "service labels must be the same shared immutable label set (anchor/alias)")

	require.Equal(t, "no", server["restart"], "restart must be exactly the string \"no\", never bare no/false")

	require.Equal(t, []any{"/app/go-mud-server"}, server["entrypoint"])

	require.Equal(t, map[string]any{
		"CONFIG_PATH": "/run/dogmud/config-overrides.yaml",
		"LOG_NOCOLOR": "1",
	}, server["environment"])

	volumesList, ok := server["volumes"].([]any)
	require.True(t, ok, "service volumes must be a list")
	require.Len(t, volumesList, 2, "exactly one named data volume and one control bind, no source-checkout bind")
	require.Equal(t, "data:/app/_datafiles", volumesList[0], "named data volume must target /app/_datafiles")

	bind, ok := volumesList[1].(map[string]any)
	require.True(t, ok, "second volume entry must be the long-form control bind")
	// The control bind must stay read-write (no "read_only: true"), by
	// design: internal/configs.SetVal's live-admin config-write path
	// (migration completion, and later playtest-scoped config changes)
	// persists via util.SafeSave, which writes "<CONFIG_PATH>.new" and
	// renames it over config-overrides.yaml in this same directory. A
	// read-only bind would break that legitimate in-container write path.
	// requireExactKeys already fails if a future edit adds "read_only", but
	// this assertion documents *why* so a future reviewer does not
	// reintroduce it believing it is a pure hardening improvement.
	requireExactKeys(t, bind, "type", "source", "target")
	require.Equal(t, "bind", bind["type"])
	require.Equal(t, "${DOGMUD_CONTROL_DIR}", bind["source"])
	require.Equal(t, "/run/dogmud", bind["target"])
	_, hasReadOnly := bind["read_only"]
	require.False(t, hasReadOnly, "control bind must remain writable - see comment above")

	portsList, ok := server["ports"].([]any)
	require.True(t, ok, "service ports must be a list")
	require.Len(t, portsList, 1, "exactly one published port, no other host ports")
	port, ok := portsList[0].(map[string]any)
	require.True(t, ok, "port entry must be the long form")
	requireExactKeys(t, port, "target", "host_ip", "protocol")
	require.Equal(t, 55555, port["target"])
	require.Equal(t, "127.0.0.1", port["host_ip"])
	require.Equal(t, "tcp", port["protocol"])
	_, published := port["published"]
	require.False(t, published, "published must be omitted so Docker assigns a dynamic loopback host port")

	volumes := mapAt(t, doc, "volumes")
	requireExactKeys(t, volumes, "data")
	dataVolume := mapAt(t, volumes, "data")
	requireExactKeys(t, dataVolume, "labels")
	require.Equal(t, wantLabels, dataVolume["labels"])

	networks := mapAt(t, doc, "networks")
	requireExactKeys(t, networks, "default")
	defaultNetwork := mapAt(t, networks, "default")
	requireExactKeys(t, defaultNetwork, "labels")
	require.Equal(t, wantLabels, defaultNetwork["labels"])
}

func TestEmbeddedComposePolicyReturnsIndependentCopies(t *testing.T) {
	a := EmbeddedComposePolicy()
	b := EmbeddedComposePolicy()
	require.True(t, &a[0] != &b[0], "each call must return an independent backing array")

	a[0] = 'X'
	require.NotEqual(t, a[0], EmbeddedComposePolicy()[0], "mutating a returned copy must never affect the embedded source")
}

func TestMaterializeControlFilesWritesResolvedComposeAndConfigOverrides(t *testing.T) {
	controlDir := t.TempDir()
	ver := version.New(0, 16, 0)

	composePath, configPath, err := materializeControlFiles(controlDir, ver)
	require.NoError(t, err)

	require.Equal(t, filepath.Join(controlDir, "compose.resolved.yml"), composePath)
	require.Equal(t, filepath.Join(controlDir, "config-overrides.yaml"), configPath)

	composeBytes, err := os.ReadFile(composePath)
	require.NoError(t, err)
	require.True(t, bytes.Equal(composeBytes, EmbeddedComposePolicy()), "resolved compose file must be byte-identical to the embedded policy")

	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)

	var overrides configOverridesDoc
	require.NoError(t, yaml.Unmarshal(configBytes, &overrides))
	require.Equal(t, "0.16.0", overrides.Server.CurrentVersion)
	require.Equal(t, 55555, overrides.Network.AIPort)
	require.False(t, overrides.Logging.LogToFile)
}

func TestMaterializeControlFilesRequiresWritableControlDir(t *testing.T) {
	parent := t.TempDir()
	// A control dir nested under a non-existent parent path can never be
	// written to - the probe write must fail before either file is
	// attempted.
	missingControlDir := filepath.Join(parent, "does-not-exist", "control")

	_, _, err := materializeControlFiles(missingControlDir, version.New(1, 0, 0))
	require.Error(t, err)
	require.True(t, errors.Is(err, ErrControlDirNotWritable))

	require.NoDirExists(t, missingControlDir)
	_, statErr := os.Stat(filepath.Join(missingControlDir, "compose.resolved.yml"))
	require.True(t, os.IsNotExist(statErr), "no compose file must be written when the control dir is not writable")
}

func TestMaterializeControlFilesLeavesNoWriteProbeBehind(t *testing.T) {
	controlDir := t.TempDir()
	_, _, err := materializeControlFiles(controlDir, version.New(2, 3, 4))
	require.NoError(t, err)

	entries, err := os.ReadDir(controlDir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.ElementsMatch(t, []string{"compose.resolved.yml", "config-overrides.yaml"}, names, "the writability probe file must never be left behind")
}

// TestWriteResolvedComposeFileAndConfigOverridesLeaveNoTempResidue proves
// both writers use atomic.WriteFile's write-to-temp-then-rename semantics:
// no stray temporary file is left in the control directory, whether writing
// for the first time or completely overwriting a prior (differently-sized)
// version of each file.
func TestWriteResolvedComposeFileAndConfigOverridesLeaveNoTempResidue(t *testing.T) {
	controlDir := t.TempDir()

	// Pre-seed both destinations with different-length content so a
	// non-atomic (truncate-then-write) implementation could leave a
	// corrupt short/garbled file if it were interrupted, and so this test
	// actually proves "complete overwrite" rather than "empty directory
	// write".
	composePath := filepath.Join(controlDir, "compose.resolved.yml")
	configPath := filepath.Join(controlDir, "config-overrides.yaml")
	require.NoError(t, os.WriteFile(composePath, []byte("stale-compose-content-that-is-much-longer-than-the-real-file"), 0o644))
	require.NoError(t, os.WriteFile(configPath, []byte("stale"), 0o644))

	gotComposePath, err := writeResolvedComposeFile(controlDir)
	require.NoError(t, err)
	require.Equal(t, composePath, gotComposePath)

	gotConfigPath, err := writeConfigOverrides(controlDir, version.New(9, 9, 9))
	require.NoError(t, err)
	require.Equal(t, configPath, gotConfigPath)

	composeBytes, err := os.ReadFile(composePath)
	require.NoError(t, err)
	require.True(t, bytes.Equal(composeBytes, EmbeddedComposePolicy()), "stale content must be completely overwritten, not appended to or truncated in place")

	configBytes, err := os.ReadFile(configPath)
	require.NoError(t, err)
	var overrides configOverridesDoc
	require.NoError(t, yaml.Unmarshal(configBytes, &overrides))
	require.Equal(t, "9.9.9", overrides.Server.CurrentVersion)

	entries, err := os.ReadDir(controlDir)
	require.NoError(t, err)
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	require.ElementsMatch(t, []string{"compose.resolved.yml", "config-overrides.yaml"}, names, "no atomic-write temp file may be left behind")
}

func TestWriteResolvedComposeFileIgnoresCheckoutComposeFile(t *testing.T) {
	// Simulate a hostile/decoy Compose file sitting right next to where
	// this package writes its own resolved Compose file. Production code
	// never takes a checkout path as an argument to any compose.go
	// function, so this proves the resolved file's content is always the
	// embedded policy, never merged with or influenced by any file already
	// on disk.
	controlDir := t.TempDir()
	decoyPath := filepath.Join(controlDir, "docker-compose.yml")
	decoyContent := []byte("services:\n  evil:\n    image: attacker/backdoor:latest\n    privileged: true\n")
	require.NoError(t, os.WriteFile(decoyPath, decoyContent, 0o644))

	composePath, err := writeResolvedComposeFile(controlDir)
	require.NoError(t, err)

	written, err := os.ReadFile(composePath)
	require.NoError(t, err)
	require.True(t, bytes.Equal(written, EmbeddedComposePolicy()))

	stillDecoy, err := os.ReadFile(decoyPath)
	require.NoError(t, err)
	require.Equal(t, decoyContent, stillDecoy, "the decoy file must be left completely untouched")
}

func TestComposeInterpolationEnvNormalizesPathsToForwardSlashesAndSorts(t *testing.T) {
	created := time.Date(2026, 8, 7, 12, 0, 0, 0, time.UTC)
	vars := composeRunVars{
		RunID:               "run-abc123",
		Project:             "dogmud-playtest-run-abc123",
		Checkout:            filepath.FromSlash("C:/checkouts/dogmud"),
		CheckoutFingerprint: "deadbeef",
		CreatedAt:           created,
		ControlDir:          filepath.FromSlash("C:/checkouts/dogmud/tools/playtest/.run/run-abc123"),
	}

	env := composeInterpolationEnv(vars)

	require.Equal(t, []string{
		"DOGMUD_CHECKOUT=C:/checkouts/dogmud",
		"DOGMUD_CHECKOUT_FINGERPRINT=deadbeef",
		"DOGMUD_CONTROL_DIR=C:/checkouts/dogmud/tools/playtest/.run/run-abc123",
		"DOGMUD_CREATED_AT=2026-08-07T12:00:00Z",
		"DOGMUD_PROJECT=dogmud-playtest-run-abc123",
		"DOGMUD_RUN_ID=run-abc123",
	}, env)
}

func TestComposeConfigCommandRendersDockerCommandArgvAndEnv(t *testing.T) {
	dc := dockerContext{name: "desktop-linux", env: []string{"PATH=/usr/bin", "HOME=/root"}}
	vars := composeRunVars{
		RunID:               "run-xyz",
		Project:             "dogmud-playtest-run-xyz",
		Checkout:            filepath.Join("checkouts", "dogmud"),
		CheckoutFingerprint: "cafef00d",
		CreatedAt:           time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		ControlDir:          filepath.Join("checkouts", "dogmud", "tools", "playtest", ".run", "run-xyz"),
	}
	composeFile := filepath.Join(vars.ControlDir, "compose.resolved.yml")

	spec := composeConfigCommand(dc, vars, composeFile, "/some/dir", nil, nil)

	require.Equal(t, "docker", spec.Name)
	require.Equal(t, []string{
		"--context", "desktop-linux",
		"compose", "--project-directory", vars.Checkout, "-f", composeFile, "-p", vars.Project,
		"config",
	}, spec.Args)
	require.Equal(t, "/some/dir", spec.Dir)

	require.Contains(t, spec.Env, "PATH=/usr/bin")
	require.Contains(t, spec.Env, "HOME=/root")
	require.Contains(t, spec.Env, "DOGMUD_RUN_ID=run-xyz")
	require.Contains(t, spec.Env, "DOGMUD_PROJECT=dogmud-playtest-run-xyz")
	require.Contains(t, spec.Env, "DOGMUD_CHECKOUT_FINGERPRINT=cafef00d")
	require.Contains(t, spec.Env, "DOGMUD_CREATED_AT=2026-01-02T03:04:05Z")
	require.Contains(t, spec.Env, "DOGMUD_CHECKOUT="+filepath.ToSlash(vars.Checkout))
	require.Contains(t, spec.Env, "DOGMUD_CONTROL_DIR="+filepath.ToSlash(vars.ControlDir))
	require.Len(t, spec.Env, 8, "env must be exactly dc.env plus the six DOGMUD_* interpolation vars")
}

// TestComposeCommandStripsAmbientReservedEnvBeforeAppendingTrustedValues
// proves that any ambient dc.env entry colliding with one of the six
// ${DOGMUD_*} interpolation names - whether canonical-case or mixed-case -
// is scrubbed before this package's trusted values are appended, so
// Compose interpolation never sees more than one value per logical key,
// and never an attacker/ambient-controlled one.
func TestComposeCommandStripsAmbientReservedEnvBeforeAppendingTrustedValues(t *testing.T) {
	dc := dockerContext{name: "ctx", env: []string{
		"PATH=/usr/bin",
		"DOGMUD_RUN_ID=malicious-ambient-run-id",   // canonical-case collision
		"dogmud_project=malicious-ambient-project", // lowercase collision
		"Dogmud_Checkout=/evil/ambient/checkout",   // mixed-case collision
		"DOGMUD_CHECKOUT_FINGERPRINT=evil-ambient-fingerprint",
		"dogmud_created_at=1970-01-01T00:00:00Z",
		"DOGMUD_CONTROL_DIR=/evil/ambient/control",
		"OTHER_VAR=keep-me",
	}}
	vars := composeRunVars{
		RunID:               "trusted-run",
		Project:             "trusted-project",
		Checkout:            "trusted-checkout",
		CheckoutFingerprint: "trusted-fingerprint",
		CreatedAt:           time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		ControlDir:          "trusted-control",
	}

	spec := composeConfigCommand(dc, vars, "compose.resolved.yml", "", nil, nil)

	require.Contains(t, spec.Env, "PATH=/usr/bin", "ordinary ambient env must be preserved")
	require.Contains(t, spec.Env, "OTHER_VAR=keep-me", "ordinary ambient env must be preserved")

	countsByReservedKey := map[string]int{}
	for _, kv := range spec.Env {
		key, _, ok := splitEnvEntry(kv)
		if !ok {
			continue
		}
		upper := strings.ToUpper(key)
		if isComposeReservedEnvName(upper) {
			countsByReservedKey[upper]++
		}
	}
	for _, reserved := range composeReservedEnvNames {
		require.Equal(t, 1, countsByReservedKey[reserved], "exactly one entry for %s: ambient collisions must be scrubbed", reserved)
	}

	require.Contains(t, spec.Env, "DOGMUD_RUN_ID=trusted-run")
	require.Contains(t, spec.Env, "DOGMUD_PROJECT=trusted-project")
	require.Contains(t, spec.Env, "DOGMUD_CHECKOUT=trusted-checkout")
	require.Contains(t, spec.Env, "DOGMUD_CHECKOUT_FINGERPRINT=trusted-fingerprint")
	require.Contains(t, spec.Env, "DOGMUD_CREATED_AT=2026-01-01T00:00:00Z")
	require.Contains(t, spec.Env, "DOGMUD_CONTROL_DIR=trusted-control")

	require.NotContains(t, spec.Env, "DOGMUD_RUN_ID=malicious-ambient-run-id")
	require.NotContains(t, spec.Env, "dogmud_project=malicious-ambient-project")
	require.NotContains(t, spec.Env, "Dogmud_Checkout=/evil/ambient/checkout")
	require.NotContains(t, spec.Env, "DOGMUD_CHECKOUT_FINGERPRINT=evil-ambient-fingerprint")
	require.NotContains(t, spec.Env, "dogmud_created_at=1970-01-01T00:00:00Z")
	require.NotContains(t, spec.Env, "DOGMUD_CONTROL_DIR=/evil/ambient/control")
}

// TestScrubComposeReservedEnvPreservesNilVsNonNilEmpty guards the same
// nil-vs-non-nil-empty distinction cloneEnv documents: scrubbing must never
// turn an already-empty (but non-nil) environment into nil, which would
// risk later code reinstating an unscrubbed real environment.
func TestScrubComposeReservedEnvPreservesNilVsNonNilEmpty(t *testing.T) {
	require.Nil(t, scrubComposeReservedEnv(nil))

	empty := scrubComposeReservedEnv([]string{})
	require.NotNil(t, empty)
	require.Empty(t, empty)
}

func TestComposeCommandEnvIsIndependentOfDockerContextEnv(t *testing.T) {
	dc := dockerContext{name: "ctx", env: []string{"PATH=/usr/bin"}}
	vars := composeRunVars{RunID: "r", Project: "p", Checkout: "c", CheckoutFingerprint: "f", ControlDir: "d"}

	spec := composeConfigCommand(dc, vars, "compose.resolved.yml", "", nil, nil)
	spec.Env[0] = "MUTATED=true"

	require.Equal(t, "PATH=/usr/bin", dc.env[0], "mutating a returned CommandSpec's Env must never affect the validated dockerContext")
}

func TestComposeLifecycleHelpersAppendExpectedSubcommands(t *testing.T) {
	dc := dockerContext{name: "ctx", env: []string{}}
	vars := composeRunVars{RunID: "r", Project: "p", Checkout: "c", CheckoutFingerprint: "f", ControlDir: "d"}
	composeFile := "compose.resolved.yml"

	tailArgs := func(spec CommandSpec) []string {
		return spec.Args[len(spec.Args)-2:]
	}

	upSpec := composeUpCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"up", "-d"}, tailArgs(upSpec))

	downSpec := composeDownCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"--volumes", "--remove-orphans"}, tailArgs(downSpec))
	require.Contains(t, downSpec.Args, "down")

	logsSpec := composeLogsCommand(dc, vars, composeFile, "", nil, nil)
	require.Equal(t, []string{"logs", "--no-color"}, tailArgs(logsSpec))
}

// composePolicyDockerTestEnvVar gates the three real-Docker integration
// tests below. They are opt-in: they build real images, start real
// containers, and require a validated local Docker context, so they never
// run as part of the default `go test` suite.
const composePolicyDockerTestEnvVar = "DOGMUD_COMPOSE_POLICY_TEST"

// requireComposePolicyDockerTests skips the calling test unless
// DOGMUD_COMPOSE_POLICY_TEST=1 is set.
func requireComposePolicyDockerTests(t *testing.T) {
	t.Helper()
	if os.Getenv(composePolicyDockerTestEnvVar) != "1" {
		t.Skipf("skipping real-Docker test: set %s=1 to enable", composePolicyDockerTestEnvVar)
	}
}

// repoRootForDockerTests resolves this package's enclosing repository root
// (two directories up from internal/playtestenv) to a canonical, absolute,
// symlink-free path for use as a real Docker build context.
func repoRootForDockerTests(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	require.NoError(t, err)
	root, err := filepath.Abs(filepath.Join(wd, "..", ".."))
	require.NoError(t, err)
	resolved, err := filepath.EvalSymlinks(root)
	require.NoError(t, err)
	require.FileExists(t, filepath.Join(resolved, "go.mod"), "resolved repo root must contain go.mod")
	return resolved
}

// newDockerTestGUID returns a random hex string suitable for a
// collision-resistant, single-test-run Compose project name or image tag.
func newDockerTestGUID(t *testing.T) string {
	t.Helper()
	var buf [8]byte
	_, err := rand.Read(buf[:])
	require.NoError(t, err)
	return hex.EncodeToString(buf[:])
}

// runDockerAndCapture runs spec's args via dc and returns captured
// stdout/stderr as strings.
func runDockerAndCapture(t *testing.T, ctx context.Context, runner Runner, dc dockerContext, args []string) (stdout, stderr string, err error) {
	t.Helper()
	var out, errOut bytes.Buffer
	spec := dockerCommand(dc, args, "", &out, &errOut)
	err = runner.Run(ctx, spec)
	return out.String(), errOut.String(), err
}

// TestDockerComposePolicyRenders materializes the real embedded Compose
// policy against this repository's own checkout, renders it with a real
// `docker compose config`, and asserts the build context and control-dir
// bind resolve to absolute paths matching the checkout/control dir, and
// that the sole published port declares no fixed host port (dynamic
// loopback publication only).
func TestDockerComposePolicyRenders(t *testing.T) {
	requireComposePolicyDockerTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	runner := execRunner{}
	dc, err := resolveLocalDockerContextForHost(ctx, runner)
	require.NoError(t, err, "requires a validated local Docker context")

	repoRoot := repoRootForDockerTests(t)
	controlDir := t.TempDir()

	composeFile, err := writeResolvedComposeFile(controlDir)
	require.NoError(t, err)

	vars := composeRunVars{
		RunID:               "policy-render-test",
		Project:             "dogmud-playtest-policy-render-test",
		Checkout:            repoRoot,
		CheckoutFingerprint: "policy-render-fingerprint",
		CreatedAt:           time.Now(),
		ControlDir:          controlDir,
	}

	var stdout, stderr bytes.Buffer
	spec := composeConfigCommand(dc, vars, composeFile, "", &stdout, &stderr)
	require.NoError(t, runner.Run(ctx, spec), "docker compose config failed: %s", stderr.String())

	rendered := decodeComposePolicy(t, stdout.Bytes())
	services := mapAt(t, rendered, "services")
	server := mapAt(t, services, "server")

	build := mapAt(t, server, "build")
	buildContext, _ := build["context"].(string)
	require.True(t, filepath.IsAbs(filepath.FromSlash(buildContext)), "build context must be absolute: %q", buildContext)
	gotRoot, err := filepath.EvalSymlinks(filepath.FromSlash(buildContext))
	require.NoError(t, err)
	wantRoot, err := filepath.EvalSymlinks(repoRoot)
	require.NoError(t, err)
	require.Equal(t, wantRoot, gotRoot, "build context must resolve to the checkout")

	volumesList, ok := server["volumes"].([]any)
	require.True(t, ok, "service volumes must render as a list")
	var bindFound bool
	for _, v := range volumesList {
		vm, ok := v.(map[string]any)
		if !ok || vm["type"] != "bind" || vm["target"] != "/run/dogmud" {
			continue
		}
		bindFound = true
		source, _ := vm["source"].(string)
		require.True(t, filepath.IsAbs(filepath.FromSlash(source)), "control bind source must be absolute: %q", source)
		gotControl, err := filepath.EvalSymlinks(filepath.FromSlash(source))
		require.NoError(t, err)
		wantControl, err := filepath.EvalSymlinks(controlDir)
		require.NoError(t, err)
		require.Equal(t, wantControl, gotControl, "control bind source must resolve to the run's control directory")
	}
	require.True(t, bindFound, "must have exactly one bind mount targeting /run/dogmud")

	portsList, ok := server["ports"].([]any)
	require.True(t, ok, "service ports must render as a list")
	require.Len(t, portsList, 1, "must declare exactly one port")
	port, ok := portsList[0].(map[string]any)
	require.True(t, ok)
	require.Equal(t, 55555, port["target"])
	require.Equal(t, "127.0.0.1", port["host_ip"])
	require.Equal(t, "tcp", port["protocol"])
	if published, ok := port["published"]; ok {
		require.Empty(t, published, "published must never be a fixed nonzero value - dynamic loopback publication only")
	}
}

// TestDockerDynamicLoopbackPublication proves the omitted-published port
// stanza this package's embedded Compose policy uses actually causes
// Docker to publish 55555/tcp on a dynamically-assigned, nonzero loopback
// port - never a fixed port and never a free-port probe performed by this
// package itself. It uses a disposable BusyBox service under a random,
// single-test-run project name so it never touches this package's own
// embedded policy or real image.
func TestDockerDynamicLoopbackPublication(t *testing.T) {
	requireComposePolicyDockerTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	runner := execRunner{}
	dc, err := resolveLocalDockerContextForHost(ctx, runner)
	require.NoError(t, err, "requires a validated local Docker context")

	project := "dogmud-playtest-loopback-" + newDockerTestGUID(t)
	tmpDir := t.TempDir()
	composeFile := filepath.Join(tmpDir, "loopback-probe.compose.yml")

	longFormYAML := []byte("services:\n" +
		"  probe:\n" +
		"    image: busybox:latest\n" +
		"    command: [\"sleep\", \"300\"]\n" +
		"    ports:\n" +
		"      - target: 55555\n" +
		"        host_ip: 127.0.0.1\n" +
		"        protocol: tcp\n")
	require.NoError(t, os.WriteFile(composeFile, longFormYAML, 0o644))

	upArgs := []string{"compose", "-p", project, "-f", composeFile, "up", "-d"}
	downArgs := []string{"compose", "-p", project, "-f", composeFile, "down", "--volumes", "--remove-orphans"}
	psArgs := []string{"compose", "-p", project, "-f", composeFile, "ps", "-q", "probe"}

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		if _, stderr, err := runDockerAndCapture(t, cleanupCtx, runner, dc, downArgs); err != nil {
			t.Logf("cleanup: docker compose down failed: %v: %s", err, stderr)
		}
	})

	_, upStderr, err := runDockerAndCapture(t, ctx, runner, dc, upArgs)
	require.NoError(t, err, "docker compose up failed: %s", upStderr)

	inspectLoopbackPort := func() (map[string][]struct {
		HostIP   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}, error) {
		containerID, psStderr, err := runDockerAndCapture(t, ctx, runner, dc, psArgs)
		if err != nil {
			return nil, fmt.Errorf("docker compose ps failed: %w: %s", err, psStderr)
		}
		containerID = strings.TrimSpace(containerID)
		if containerID == "" {
			return nil, fmt.Errorf("docker compose ps returned no container id")
		}
		inspectOut, inspectErr, err := runDockerAndCapture(t, ctx, runner, dc,
			[]string{"inspect", "--format", "{{json .NetworkSettings.Ports}}", containerID})
		if err != nil {
			return nil, fmt.Errorf("docker inspect failed: %w: %s", err, inspectErr)
		}
		var portMap map[string][]struct {
			HostIP   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		}
		if err := json.Unmarshal([]byte(inspectOut), &portMap); err != nil {
			return nil, fmt.Errorf("decode NetworkSettings.Ports: %w", err)
		}
		return portMap, nil
	}

	portMap, err := inspectLoopbackPort()
	require.NoError(t, err)
	mappings, ok := portMap["55555/tcp"]

	if !ok || len(mappings) == 0 || mappings[0].HostIP == "" {
		// Documented fallback only: some Compose builds drop host_ip when
		// "published" is omitted from the long-form stanza. The only
		// permitted fallback is the short-form "127.0.0.1::55555" syntax -
		// never a free host-port probe, and never a fixed nonzero
		// published port.
		t.Logf("long-form omitted-published stanza did not yield a loopback mapping (%v); retrying with short-form 127.0.0.1::55555", mappings)

		_, downStderr, err := runDockerAndCapture(t, ctx, runner, dc, downArgs)
		require.NoError(t, err, "docker compose down (before short-form retry) failed: %s", downStderr)

		shortFormYAML := []byte("services:\n" +
			"  probe:\n" +
			"    image: busybox:latest\n" +
			"    command: [\"sleep\", \"300\"]\n" +
			"    ports:\n" +
			"      - \"127.0.0.1::55555\"\n")
		require.NoError(t, os.WriteFile(composeFile, shortFormYAML, 0o644))

		_, upStderr, err = runDockerAndCapture(t, ctx, runner, dc, upArgs)
		require.NoError(t, err, "docker compose up (short-form retry) failed: %s", upStderr)

		portMap, err = inspectLoopbackPort()
		require.NoError(t, err)
		mappings, ok = portMap["55555/tcp"]
	}

	require.True(t, ok, "container must publish 55555/tcp")
	require.Len(t, mappings, 1, "exactly one host mapping for 55555/tcp")
	require.Equal(t, "127.0.0.1", mappings[0].HostIP)
	require.NotEmpty(t, mappings[0].HostPort)
	require.NotEqual(t, "0", mappings[0].HostPort, "host port must be a real dynamically-assigned nonzero port")
}

// TestDockerContextExcludesSensitiveAndRunState builds the real "builder"
// Dockerfile stage (which COPYs the full build context) from this
// repository's own checkout, with a throwaway sentinel file planted at a
// path this package's own future run-state lives under, then runs a shell
// inside the built image to prove none of the checkout's credential file,
// _archive tree, or the sentinel ever reached the build context. It never
// reads or prints the credential file's contents.
func TestDockerContextExcludesSensitiveAndRunState(t *testing.T) {
	requireComposePolicyDockerTests(t)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Minute)
	defer cancel()

	runner := execRunner{}
	dc, err := resolveLocalDockerContextForHost(ctx, runner)
	require.NoError(t, err, "requires a validated local Docker context")

	repoRoot := repoRootForDockerTests(t)
	guid := newDockerTestGUID(t)

	sentinelRelPath := "tools/playtest/.run-dockerignore-test-" + guid
	sentinelAbsPath := filepath.Join(repoRoot, filepath.FromSlash(sentinelRelPath))
	require.NoError(t, os.WriteFile(sentinelAbsPath, []byte("dockerignore-probe\n"), 0o644))
	t.Cleanup(func() {
		_ = os.Remove(sentinelAbsPath)
	})

	tag := "dogmud-playtest-dockerignore-test:" + guid

	_, buildStderr, err := runDockerAndCapture(t, ctx, runner, dc, []string{
		"build",
		"--target", "builder",
		"-t", tag,
		"-f", filepath.Join(repoRoot, "provisioning", "Dockerfile"),
		repoRoot,
	})
	require.NoError(t, err, "docker build failed: %s", buildStderr)

	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cleanupCancel()
		if _, stderr, err := runDockerAndCapture(t, cleanupCtx, runner, dc, []string{"image", "rm", "-f", tag}); err != nil {
			t.Logf("cleanup: docker image rm failed: %v: %s", err, stderr)
		}
	})

	pathExistsInImage := func(containerPath string) bool {
		_, _, err := runDockerAndCapture(t, ctx, runner, dc,
			[]string{"run", "--rm", tag, "/bin/sh", "-c", "test -e " + containerPath})
		return err == nil
	}

	require.False(t, pathExistsInImage("/src/tools/playtest/targets.yaml"), "playtest target credentials must never reach the build context")
	require.False(t, pathExistsInImage("/src/_archive"), "_archive must never reach the build context")
	require.False(t, pathExistsInImage("/src/"+sentinelRelPath), "supervisor run-state sentinel must never reach the build context")
}
