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

func unixInspectResponse() string { return `"unix:///var/run/docker.sock"` + "\n" }
func npipeInspectResponse() string {
	return `"npipe://./pipe/docker_engine"` + "\n"
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

// TestDockerContextRejectsDockerHostOverride proves a nonempty ambient
// DOCKER_HOST is rejected before any Docker command is invoked.
func TestDockerContextRejectsDockerHostOverride(t *testing.T) {
	fr := newFakeDockerRunner()
	ambient := []string{"PATH=/usr/bin", "DOCKER_HOST=tcp://1.2.3.4:2376"}

	_, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")

	require.ErrorIs(t, err, ErrDockerHostOverride)
	require.Empty(t, fr.calls, "must not invoke docker before rejecting a DOCKER_HOST override")
}

// TestDockerContextAcceptsLocalSelectedContext proves a nonempty
// DOCKER_CONTEXT is used directly as the candidate, skipping `docker context
// show`, and that a fully valid local context resolves successfully.
func TestDockerContextAcceptsLocalSelectedContext(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, false, "mycontext", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	ambient := []string{"DOCKER_CONTEXT=mycontext"}
	got, err := resolveLocalDockerContext(context.Background(), fr, ambient, "linux")

	require.NoError(t, err)
	require.Equal(t, "mycontext", got)
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

	got, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.NoError(t, err)
	require.Equal(t, "desktop-linux", got)
	require.NotEmpty(t, fr.calls)
	require.Equal(t, []string{"context", "show"}, fr.calls[0].Args)
}

// TestDockerPreflightOrderAndNoResourceCommandBeforeSuccess proves the
// preflight issues its commands in the documented order and that no
// resource-affecting command precedes a fully successful preflight.
func TestDockerPreflightOrderAndNoResourceCommandBeforeSuccess(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-linux", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	got, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")
	require.NoError(t, err)
	require.Equal(t, "desktop-linux", got)

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

	_, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.ErrorIs(t, err, ErrDockerContextNotLocal)
	require.Len(t, fr.calls, 2, "must not run compose/docker version checks after endpoint validation fails")
}

// TestDockerPreflightAcceptsWindowsNamedPipe proves an npipe:// endpoint is
// accepted when the injected platform is Windows.
func TestDockerPreflightAcceptsWindowsNamedPipe(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-windows", `"npipe://./pipe/docker_engine"`, "v2.20.0", "24.0.5")

	got, err := resolveLocalDockerContext(context.Background(), fr, nil, "windows")

	require.NoError(t, err)
	require.Equal(t, "desktop-windows", got)
}

// TestDockerPreflightAcceptsLinuxUnixSocket proves a unix:// endpoint is
// accepted when the injected platform is Linux.
func TestDockerPreflightAcceptsLinuxUnixSocket(t *testing.T) {
	fr := newFakeDockerRunner()
	scriptHappyPreflight(fr, true, "desktop-linux", `"unix:///var/run/docker.sock"`, "2.20.0", "24.0.5")

	got, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.NoError(t, err)
	require.Equal(t, "desktop-linux", got)
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

			_, err := resolveLocalDockerContext(context.Background(), fr, nil, tc.goos)

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

			_, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

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

	_, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.ErrorIs(t, err, ErrDockerServerVersionEmpty)
}

// TestDockerContextShowFailurePropagates proves a failing `docker context
// show` call surfaces as an error rather than silently resolving an empty
// candidate.
func TestDockerContextShowFailurePropagates(t *testing.T) {
	fr := newFakeDockerRunner()
	fr.script("context", "show").returns("", "docker daemon not running", fmt.Errorf("exit status 1"))

	_, err := resolveLocalDockerContext(context.Background(), fr, nil, "linux")

	require.Error(t, err)
	require.Len(t, fr.calls, 1)
}

// TestDockerPreflightScrubsOverridesFromEveryPreflightCommand proves that
// every command the preflight itself issues - not just those built later by
// dockerCommand - has the four ambient override variables scrubbed from its
// environment, even when the candidate context comes directly from
// DOCKER_CONTEXT.
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

// TestDockerCommandAlwaysUsesValidatedLocalContext proves the central
// command constructor always emits argv beginning "--context
// <validated-context>" ahead of the caller-supplied args.
func TestDockerCommandAlwaysUsesValidatedLocalContext(t *testing.T) {
	spec := dockerCommand("desktop-linux", []string{"compose", "up", "--detach", "--no-build", "server"}, "", nil, io.Discard, io.Discard)

	require.Equal(t, "docker", spec.Name)
	require.Equal(t, []string{"--context", "desktop-linux", "compose", "up", "--detach", "--no-build", "server"}, spec.Args)
}

// TestDockerCommandScrubsAmbientOverridesAfterResolution proves the four
// ambient Docker override variables never appear in a constructed command's
// Env, regardless of what the caller's ambient environment contains.
func TestDockerCommandScrubsAmbientOverridesAfterResolution(t *testing.T) {
	ambient := []string{
		"PATH=/usr/bin",
		"DOCKER_HOST=tcp://1.2.3.4:2376",
		"DOCKER_CONTEXT=other",
		"DOCKER_TLS_VERIFY=1",
		"DOCKER_CERT_PATH=/certs",
		"HOME=/home/user",
	}

	spec := dockerCommand("desktop-linux", []string{"ps"}, "", ambient, io.Discard, io.Discard)

	blocked := map[string]bool{
		"DOCKER_HOST":       true,
		"DOCKER_CONTEXT":    true,
		"DOCKER_TLS_VERIFY": true,
		"DOCKER_CERT_PATH":  true,
	}
	for _, kv := range spec.Env {
		key := kv
		if idx := strings.IndexByte(kv, '='); idx >= 0 {
			key = kv[:idx]
		}
		require.False(t, blocked[key], "env var %q must be scrubbed from every Docker/Compose child", key)
	}
	require.Contains(t, spec.Env, "PATH=/usr/bin")
	require.Contains(t, spec.Env, "HOME=/home/user")
}

// TestDockerCommandSetsDirAndWriters proves the constructor passes through
// Dir, Stdout, and Stderr unchanged.
func TestDockerCommandSetsDirAndWriters(t *testing.T) {
	var stdout, stderr strings.Builder
	spec := dockerCommand("desktop-linux", []string{"ps"}, "/checkout", nil, &stdout, &stderr)

	require.Equal(t, "/checkout", spec.Dir)
	require.Same(t, &stdout, spec.Stdout)
	require.Same(t, &stderr, spec.Stderr)
}
