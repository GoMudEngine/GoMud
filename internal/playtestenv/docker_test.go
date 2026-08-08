package playtestenv

import (
	"context"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeDockerRunner is a Runner test double that records exact argv/env for
// every call and returns a scripted response keyed by the exact Args slice.
// It never invokes a real subprocess.
type fakeDockerRunner struct {
	mu        sync.Mutex
	calls     []CommandSpec
	responses map[string]dockerScriptedResponse
}

type dockerScriptedResponse struct {
	stdout string
	stderr string
	err    error
}

func newFakeDockerRunner() *fakeDockerRunner {
	return &fakeDockerRunner{responses: map[string]dockerScriptedResponse{}}
}

func dockerArgsKey(args []string) string {
	return strings.Join(args, "\x1f")
}

// script registers a scripted response for an exact argv. Call .returns on
// the result to supply stdout/stderr/error.
func (f *fakeDockerRunner) script(args ...string) *dockerScriptBuilder {
	return &dockerScriptBuilder{f: f, key: dockerArgsKey(args)}
}

type dockerScriptBuilder struct {
	f   *fakeDockerRunner
	key string
}

func (b *dockerScriptBuilder) returns(stdout, stderr string, err error) {
	b.f.mu.Lock()
	defer b.f.mu.Unlock()
	b.f.responses[b.key] = dockerScriptedResponse{stdout: stdout, stderr: stderr, err: err}
}

// Run implements Runner. It records a copy of the spec (Args/Env copied so
// later caller-side slice mutation cannot retroactively change a recorded
// call) and returns the scripted response for spec.Args, failing loudly if
// no response was registered - a test must script every command it expects
// to be issued.
func (f *fakeDockerRunner) Run(ctx context.Context, spec CommandSpec) error {
	f.mu.Lock()
	recorded := spec
	recorded.Args = append([]string(nil), spec.Args...)
	recorded.Env = append([]string(nil), spec.Env...)
	f.calls = append(f.calls, recorded)
	resp, ok := f.responses[dockerArgsKey(spec.Args)]
	f.mu.Unlock()

	if !ok {
		return fmt.Errorf("fakeDockerRunner: no scripted response for docker %v", spec.Args)
	}
	if spec.Stdout != nil && resp.stdout != "" {
		_, _ = spec.Stdout.Write([]byte(resp.stdout))
	}
	if spec.Stderr != nil && resp.stderr != "" {
		_, _ = spec.Stderr.Write([]byte(resp.stderr))
	}
	return resp.err
}

// scriptHappyPreflight registers a fully successful preflight sequence for
// the given candidate context name, endpoint JSON, and Compose version
// string, so individual tests can focus on the one dimension they vary.
func scriptHappyPreflight(f *fakeDockerRunner, includeContextShow bool, candidate, endpointJSON, composeVersion, serverVersion string) {
	if includeContextShow {
		f.script("context", "show").returns(candidate+"\n", "", nil)
	}
	f.script("--context", candidate, "context", "inspect", candidate, "--format", "{{json .Endpoints.docker.Host}}").
		returns(endpointJSON+"\n", "", nil)
	f.script("--context", candidate, "compose", "version", "--short").returns(composeVersion+"\n", "", nil)
	f.script("--context", candidate, "version", "--format", "{{.Server.Version}}").returns(serverVersion+"\n", "", nil)
}

// TestDockerContextRejectsNilAmbientEnvironment proves a nil ambientEnv is
// rejected outright rather than silently treated as an empty environment.
// Silently proceeding with no ambient environment risks masking a caller
// bug that forgot to supply the real environment, and could erase entries
// (such as PATH) that later commands still need.
func TestDockerContextRejectsNilAmbientEnvironment(t *testing.T) {
	fr := newFakeDockerRunner()

	_, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.ErrorIs(t, err, ErrAmbientEnvironmentRequired)
	require.Empty(t, fr.calls, "must not invoke docker with an unvalidated/absent ambient environment")
}

// TestCloneEnvPreservesNilVsNonNilEmpty proves cloneEnv distinguishes a nil
// slice (exec.Cmd's "inherit the real host environment" sentinel) from a
// non-nil empty slice (an explicit, deliberately empty environment), and
// that it copies rather than aliases a nonempty input's backing array.
func TestCloneEnvPreservesNilVsNonNilEmpty(t *testing.T) {
	require.Nil(t, cloneEnv(nil))

	got := cloneEnv([]string{})
	require.NotNil(t, got, "cloneEnv(non-nil empty) must stay non-nil - append([]string(nil), empty...) would collapse it to nil")
	require.Empty(t, got)

	src := []string{"A=1", "B=2"}
	cloned := cloneEnv(src)
	require.Equal(t, src, cloned)
	src[0] = "MUTATED"
	require.Equal(t, []string{"A=1", "B=2"}, cloned, "cloneEnv must not alias the input's backing array")
}

// TestDockerContextPreservesNonNilEmptyEnvironment proves that resolving
// against a non-nil, empty ambient environment produces a non-nil, empty
// dockerContext.env and, in turn, a non-nil, empty CommandSpec.Env. exec.Cmd
// treats a nil Env as "inherit the real host environment"; collapsing an
// intentionally empty scrubbed environment to nil would silently reinstate
// whatever DOCKER_* overrides exist in this test process's real
// environment.
func TestDockerContextPreservesNonNilEmptyEnvironment(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "ctx", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	dc, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")
	require.NoError(t, err)
	require.NotNil(t, dc.env, "a non-nil (even if empty) ambient environment must yield a non-nil dockerContext.env")
	require.Empty(t, dc.env)

	spec := dockerCommand(dc, []string{"ps"}, "", io.Discard, io.Discard)
	require.NotNil(t, spec.Env, "CommandSpec.Env must not collapse to nil - nil means \"inherit the real host environment\" to exec.Cmd")
	require.Empty(t, spec.Env)
}

// TestDockerContextScrubbingAllOverridesYieldsNonNilEmptyEnvironment proves
// that when every entry of a nonempty ambient environment is an override
// variable and all of them are scrubbed away, the resulting environment is
// non-nil and empty - never nil, which would cause the eventual
// Docker/Compose child to inherit the real, unsanitized host environment
// instead of running with none.
func TestDockerContextScrubbingAllOverridesYieldsNonNilEmptyEnvironment(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	ambient := []string{
		"DOCKER_CONTEXT=mycontext",
		"DOCKER_TLS_VERIFY=1",
		"DOCKER_CERT_PATH=/certs",
	}
	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")
	require.NoError(t, err)
	require.NotNil(t, dc.env, "scrubbing every ambient entry away must not collapse the environment to nil")
	require.Empty(t, dc.env)

	spec := dockerCommand(dc, []string{"ps"}, "", io.Discard, io.Discard)
	require.NotNil(t, spec.Env, "CommandSpec.Env must remain non-nil even when scrubbed to empty, or the child inherits the real unsanitized host environment")
	require.Empty(t, spec.Env)
}

// TestDockerContextRejectsDockerHostOverride proves a nonempty ambient
// DOCKER_HOST is rejected before any Docker command is invoked.
func TestDockerContextRejectsDockerHostOverride(t *testing.T) {
	fr := newFakeDockerRunner()
	ambient := []string{"PATH=/usr/bin", "DOCKER_HOST=tcp://1.2.3.4:2376"}

	_, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")

	require.ErrorIs(t, err, ErrDockerHostOverride)
	require.Empty(t, fr.calls, "must not invoke docker before rejecting a DOCKER_HOST override")
}

// TestDockerContextRejectsMixedCaseDockerHostOnWindows proves that on
// Windows - where environment variable names are case-insensitive - every
// differently-cased spelling of DOCKER_HOST is still recognized and
// rejected, not just the canonical all-caps form.
func TestDockerContextRejectsMixedCaseDockerHostOnWindows(t *testing.T) {
	cases := []string{"DOCKER_HOST", "Docker_Host", "docker_host", "DoCkEr_HoSt"}

	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			fr := newFakeDockerRunner()
			ambient := []string{name + "=tcp://1.2.3.4:2376"}

			_, err := resolveLocalDockerContext(context.Background(), fr, ambient, "windows")

			require.ErrorIs(t, err, ErrDockerHostOverride)
			require.Empty(t, fr.calls, "must not invoke docker before rejecting any case variant of DOCKER_HOST")
		})
	}
}

// TestDockerEnvLookupIsCaseSensitiveOnLinux proves that on Linux (and every
// non-Windows platform) environment variable names are compared exactly.
// "Docker_Host" is a distinct, unrelated variable name on POSIX and must
// never be treated as DOCKER_HOST.
func TestDockerEnvLookupIsCaseSensitiveOnLinux(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "ctx", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")
	ambient := []string{"Docker_Host=tcp://1.2.3.4:2376"}

	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")

	require.NoError(t, err, "a differently-cased variable name is a distinct variable on POSIX, not DOCKER_HOST")
	require.Equal(t, "ctx", dc.name)
}

// TestDockerContextAcceptsMixedCaseDockerContextOnWindows proves that on
// Windows, DOCKER_CONTEXT is discovered regardless of its exact casing.
func TestDockerContextAcceptsMixedCaseDockerContextOnWindows(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"npipe://./pipe/docker_engine"`, "2.20.0", "24.0.5")

	ambient := []string{"docker_context=mycontext"}
	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "windows")

	require.NoError(t, err)
	require.Equal(t, "mycontext", dc.name)
	for _, c := range fr.calls {
		require.NotEqual(t, []string{"context", "show"}, c.Args, "must not call docker context show once a case-insensitive DOCKER_CONTEXT match is found")
	}
}

// TestDockerPreflightScrubsMixedCaseOverridesOnWindows proves that on
// Windows, every mixed-case spelling of all four override variables is
// scrubbed from the returned dockerContext's environment, while an
// ordinary variable (Path) is preserved untouched.
func TestDockerPreflightScrubsMixedCaseOverridesOnWindows(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"npipe://./pipe/docker_engine"`, "2.20.0", "24.0.5")

	ambient := []string{
		`Path=C:\Windows`,
		"docker_context=mycontext",
		"Docker_Tls_Verify=1",
		`DOCKER_CERT_PATH=C:\certs`,
	}
	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "windows")
	require.NoError(t, err)
	require.Equal(t, "mycontext", dc.name)

	blockedUpper := map[string]bool{
		"DOCKER_HOST":       true,
		"DOCKER_CONTEXT":    true,
		"DOCKER_TLS_VERIFY": true,
		"DOCKER_CERT_PATH":  true,
	}
	for _, kv := range dc.env {
		require.False(t, blockedUpper[strings.ToUpper(envKey(kv))], "env var %q must be scrubbed regardless of case on Windows", kv)
	}
	require.Contains(t, dc.env, `Path=C:\Windows`)
}

// TestDockerContextAcceptsLocalSelectedContext proves a nonempty
// DOCKER_CONTEXT is used directly as the candidate, skipping `docker context
// show`, and that a fully valid local context resolves successfully.
func TestDockerContextAcceptsLocalSelectedContext(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	ambient := []string{"DOCKER_CONTEXT=mycontext"}
	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")

	require.NoError(t, err)
	require.Equal(t, "mycontext", dc.name)
	for _, c := range fr.calls {
		require.NotEqual(t, []string{"context", "show"}, c.Args, "must not call docker context show when DOCKER_CONTEXT is set")
	}
}

// TestDockerPreflightFallsBackToContextShowWhenUnset proves that when
// DOCKER_CONTEXT is absent (or empty), the candidate comes from `docker
// context show` instead.
func TestDockerPreflightFallsBackToContextShowWhenUnset(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-linux", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	dc, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

	require.NoError(t, err)
	require.Equal(t, "desktop-linux", dc.name)
	require.NotEmpty(t, fr.calls)
	require.Equal(t, []string{"context", "show"}, fr.calls[0].Args)
}

// TestDockerPreflightOrderAndNoResourceCommandBeforeSuccess proves the
// preflight issues its commands in the documented order and that no
// resource-affecting command precedes a fully successful preflight.
func TestDockerPreflightOrderAndNoResourceCommandBeforeSuccess(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-linux", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	dc, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")
	require.NoError(t, err)
	require.Equal(t, "desktop-linux", dc.name)

	require.Len(t, fr.calls, 4)
	require.Equal(t, []string{"context", "show"}, fr.calls[0].Args)
	require.Equal(t, []string{"--context", "desktop-linux", "context", "inspect", "desktop-linux", "--format", "{{json .Endpoints.docker.Host}}"}, fr.calls[1].Args)
	require.Equal(t, []string{"--context", "desktop-linux", "compose", "version", "--short"}, fr.calls[2].Args)
	require.Equal(t, []string{"--context", "desktop-linux", "version", "--format", "{{.Server.Version}}"}, fr.calls[3].Args)
}

// TestDockerPreflightStopsAtFirstFailureNoLaterResourceCommand proves that
// once endpoint validation fails, neither the compose-version nor the
// docker-version resource commands are issued.
func TestDockerPreflightStopsAtFirstFailureNoLaterResourceCommand(t *testing.T) {
	fr := newFakeDockerRunner()
	fr.script("context", "show").returns("desktop-linux\n", "", nil)
	fr.script("--context", "desktop-linux", "context", "inspect", "desktop-linux", "--format", "{{json .Endpoints.docker.Host}}").
		returns(`"tcp://1.2.3.4:2376"`+"\n", "", nil)

	_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

	require.ErrorIs(t, err, ErrDockerContextNotLocal)
	require.Len(t, fr.calls, 2, "must not run compose/docker version checks after endpoint validation fails")
}

// TestDockerPreflightAcceptsWindowsNamedPipe proves an npipe:// endpoint is
// accepted when the injected platform is Windows.
func TestDockerPreflightAcceptsWindowsNamedPipe(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-windows", `"npipe://./pipe/docker_engine"`, "v2.20.0", "24.0.5")

	dc, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "windows")

	require.NoError(t, err)
	require.Equal(t, "desktop-windows", dc.name)
}

// TestDockerPreflightAcceptsLinuxUnixSocket proves a unix:// endpoint is
// accepted when the injected platform is Linux.
func TestDockerPreflightAcceptsLinuxUnixSocket(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-linux", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	dc, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

	require.NoError(t, err)
	require.Equal(t, "desktop-linux", dc.name)
}

// TestDockerPreflightRejectsTCPSSHAndMalformedEndpoints proves every
// non-local, empty, malformed, or platform-mismatched endpoint is rejected
// before any compose/version resource command runs.
func TestDockerPreflightRejectsTCPSSHAndMalformedEndpoints(t *testing.T) {
	cases := []struct {
		name         string
		goos         string
		endpointJSON string
	}{
		{"tcp transport", "linux", `"tcp://1.2.3.4:2376"`},
		{"ssh transport", "linux", `"ssh://user@host"`},
		{"empty endpoint", "linux", `""`},
		{"malformed json", "linux", `not-json`},
		{"npipe endpoint on linux goos", "linux", `"npipe://./pipe/docker_engine"`},
		{"unix endpoint on windows goos", "windows", `"unix:///var/run/docker.sock"`},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newFakeDockerRunner()
			fr.script("context", "show").returns("ctx\n", "", nil)
			fr.script("--context", "ctx", "context", "inspect", "ctx", "--format", "{{json .Endpoints.docker.Host}}").
				returns(tc.endpointJSON+"\n", "", nil)

			_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, tc.goos)

			require.Error(t, err)
			require.ErrorIs(t, err, ErrDockerContextNotLocal)
			require.Len(t, fr.calls, 2, "must reject before any compose/docker version command")
		})
	}
}

// TestDockerPreflightRejectsUnsupportedPlatforms proves that a platform
// outside the explicitly approved Windows/Linux support matrix is rejected
// outright - even when the endpoint would otherwise look like a legitimate
// local unix:// socket. Approved support is Windows and Linux only; there is
// no default/fallback local transport for any other GOOS.
func TestDockerPreflightRejectsUnsupportedPlatforms(t *testing.T) {
	cases := []struct {
		name string
		goos string
	}{
		{"darwin with unix socket endpoint", "darwin"},
		{"freebsd with unix socket endpoint", "freebsd"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newFakeDockerRunner()
			fr.script("context", "show").returns("ctx\n", "", nil)
			fr.script("--context", "ctx", "context", "inspect", "ctx", "--format", "{{json .Endpoints.docker.Host}}").
				returns(`"unix:///var/run/docker.sock"`+"\n", "", nil)

			_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, tc.goos)

			require.Error(t, err)
			require.ErrorIs(t, err, ErrDockerContextNotLocal)
			require.Len(t, fr.calls, 2, "must reject before any compose/docker version command")
		})
	}
}

// TestDockerPreflightComposeVersionFloor proves Compose >= 2.20.0 is
// required, that a leading "v" is accepted, and that anything below the
// floor is rejected.
func TestDockerPreflightComposeVersionFloor(t *testing.T) {
	cases := []struct {
		name    string
		version string
		wantErr bool
	}{
		{"exact floor", "2.20.0", false},
		{"exact floor leading v", "v2.20.0", false},
		{"above floor", "2.25.3", false},
		{"above floor leading v", "v2.25.3", false},
		{"below floor patch", "2.19.9", true},
		{"below floor minor", "2.9.9", true},
		{"below floor major", "1.29.2", true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newFakeDockerRunner()
			scriptHappyPreflight(fr, true, "ctx", `"unix:///var/run/docker.sock"`, tc.version, "24.0.5")

			_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

			if tc.wantErr {
				require.ErrorIs(t, err, ErrComposeVersionTooOld)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDockerPreflightRejectsEmptyServerVersion proves an empty Docker
// server version fails the preflight even after every earlier check passes.
func TestDockerPreflightRejectsEmptyServerVersion(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "ctx", `"unix:///var/run/docker.sock"`, "2.20.0", "")

	_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

	require.ErrorIs(t, err, ErrDockerServerVersionEmpty)
}

// TestDockerContextShowFailurePropagates proves a failing `docker context
// show` call surfaces as an error rather than silently resolving an empty
// candidate.
func TestDockerContextShowFailurePropagates(t *testing.T) {
	fr := newFakeDockerRunner()
	fr.script("context", "show").returns("", "docker daemon not running", fmt.Errorf("exit status 1"))

	_, err := resolveLocalDockerContext(context.Background(), fr, []string{}, "linux")

	require.Error(t, err)
	require.Len(t, fr.calls, 1)
}

// TestDockerPreflightScrubsOverridesFromEveryPreflightCommand proves that
// every command the preflight itself issues has the four ambient override
// variables scrubbed from its environment, even when the candidate context
// comes directly from DOCKER_CONTEXT.
func TestDockerPreflightScrubsOverridesFromEveryPreflightCommand(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	ambient := []string{
		"PATH=/usr/bin",
		"DOCKER_CONTEXT=mycontext",
		"DOCKER_TLS_VERIFY=1",
		"DOCKER_CERT_PATH=/certs",
	}
	_, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")
	require.NoError(t, err)
	require.NotEmpty(t, fr.calls)

	blocked := map[string]bool{
		"DOCKER_HOST":       true,
		"DOCKER_CONTEXT":    true,
		"DOCKER_TLS_VERIFY": true,
		"DOCKER_CERT_PATH":  true,
	}
	for _, call := range fr.calls {
		for _, kv := range call.Env {
			require.False(t, blocked[envKey(kv)], "command %v must not receive ambient override %q", call.Args, kv)
		}
		require.Contains(t, call.Env, "PATH=/usr/bin")
	}
}

// TestDockerCommandArgsAndEnvComeOnlyFromDockerContextValue proves
// dockerCommand has no free variables: its Args and Env are wholly
// determined by the dockerContext value passed in, with no implicit
// fallback to a package-level global or fresh ambient environment. This is
// the structural guarantee that a caller cannot select a different
// context/environment without constructing a different dockerContext.
func TestDockerCommandArgsAndEnvComeOnlyFromDockerContextValue(t *testing.T) {
	dcA := dockerContext{name: "context-a", env: []string{"ONLY_A=1"}}
	dcB := dockerContext{name: "context-b", env: []string{"ONLY_B=1"}}

	specA := dockerCommand(dcA, []string{"ps"}, "", io.Discard, io.Discard)
	specB := dockerCommand(dcB, []string{"ps"}, "", io.Discard, io.Discard)

	require.Equal(t, "docker", specA.Name)
	require.Equal(t, []string{"--context", "context-a", "ps"}, specA.Args)
	require.Equal(t, []string{"ONLY_A=1"}, specA.Env)
	require.Equal(t, []string{"--context", "context-b", "ps"}, specB.Args)
	require.Equal(t, []string{"ONLY_B=1"}, specB.Env)
	require.NotEqual(t, specA.Args, specB.Args)
	require.NotEqual(t, specA.Env, specB.Env)
}

// TestDockerCommandClonesContextEnvIntoCommandSpec proves dockerCommand
// clones dc.env into the returned CommandSpec rather than aliasing it, so
// later mutation of the caller's backing slice cannot retroactively change
// an already-built command.
func TestDockerCommandClonesContextEnvIntoCommandSpec(t *testing.T) {
	env := []string{"PATH=/usr/bin"}
	dc := dockerContext{name: "ctx", env: env}

	spec := dockerCommand(dc, []string{"ps"}, "", io.Discard, io.Discard)
	env[0] = "PATH=/mutated"

	require.Equal(t, []string{"PATH=/usr/bin"}, spec.Env, "dockerCommand must clone dc.env, not alias it")
}

// TestDockerCommandAlwaysUsesValidatedLocalContext proves the central
// command constructor always emits argv beginning "--context
// <validated-context>" ahead of the caller-supplied args.
func TestDockerCommandAlwaysUsesValidatedLocalContext(t *testing.T) {
	dc := dockerContext{name: "desktop-linux"}
	spec := dockerCommand(dc, []string{"compose", "up", "--detach", "--no-build", "server"}, "", io.Discard, io.Discard)

	require.Equal(t, "docker", spec.Name)
	require.Equal(t, []string{"--context", "desktop-linux", "compose", "up", "--detach", "--no-build", "server"}, spec.Args)
}

// TestDockerCommandScrubsAmbientOverridesAfterResolution proves that once a
// dockerContext has been produced by resolving a real (override-laden)
// ambient environment, every command dockerCommand builds from it carries
// only the already-scrubbed environment - never the four ambient overrides
// - while ordinary entries such as PATH and HOME survive untouched.
func TestDockerCommandScrubsAmbientOverridesAfterResolution(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	ambient := []string{
		"PATH=/usr/bin",
		"DOCKER_CONTEXT=mycontext",
		"DOCKER_TLS_VERIFY=1",
		"DOCKER_CERT_PATH=/certs",
		"HOME=/home/user",
	}
	dc, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")
	require.NoError(t, err)

	spec := dockerCommand(dc, []string{"ps"}, "", io.Discard, io.Discard)

	require.Equal(t, []string{"--context", "mycontext", "ps"}, spec.Args)
	blocked := map[string]bool{
		"DOCKER_HOST":       true,
		"DOCKER_CONTEXT":    true,
		"DOCKER_TLS_VERIFY": true,
		"DOCKER_CERT_PATH":  true,
	}
	for _, kv := range spec.Env {
		require.False(t, blocked[envKey(kv)], "env var %q must be scrubbed from every Docker/Compose child", kv)
	}
	require.Contains(t, spec.Env, "PATH=/usr/bin")
	require.Contains(t, spec.Env, "HOME=/home/user")
}

// TestDockerCommandSetsDirAndWriters proves the constructor passes through
// Dir, Stdout, and Stderr unchanged.
func TestDockerCommandSetsDirAndWriters(t *testing.T) {
	var stdout, stderr strings.Builder
	dc := dockerContext{name: "desktop-linux"}
	spec := dockerCommand(dc, []string{"ps"}, "/checkout", &stdout, &stderr)

	require.Equal(t, "/checkout", spec.Dir)
	require.Same(t, &stdout, spec.Stdout)
	require.Same(t, &stderr, spec.Stderr)
}
