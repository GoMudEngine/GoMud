package playtestenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"runtime"
	"strconv"
	"strings"
)

// dockerOverrideVars lists the ambient Docker environment variables that
// must never reach a Docker/Compose child once a local context has been
// selected: they either select a remote daemon directly (DOCKER_HOST,
// DOCKER_CONTEXT) or configure remote TLS transport (DOCKER_TLS_VERIFY,
// DOCKER_CERT_PATH).
var dockerOverrideVars = []string{
	"DOCKER_HOST",
	"DOCKER_CONTEXT",
	"DOCKER_TLS_VERIFY",
	"DOCKER_CERT_PATH",
}

// minComposeVersion is the lowest Docker Compose version this package
// trusts to render the dynamic-loopback-port and label policy the embedded
// Compose file depends on.
const minComposeVersion = "2.20.0"

// ErrAmbientEnvironmentRequired is returned when resolveLocalDockerContext
// is called with a nil ambientEnv. A nil slice is deliberately rejected
// rather than silently treated as an empty environment: proceeding with no
// ambient environment at all risks masking a caller bug that forgot to
// supply the real environment, and could erase entries (such as PATH) that
// later Docker/Compose invocations still need. A caller with a genuinely
// empty environment must pass a non-nil empty slice instead.
var ErrAmbientEnvironmentRequired = errors.New("playtestenv: ambient environment must not be nil")

// ErrDockerHostOverride is returned when the ambient environment sets a
// nonempty DOCKER_HOST. This package never talks to a daemon selected by
// DOCKER_HOST; callers must select a named local Docker context instead.
var ErrDockerHostOverride = errors.New("playtestenv: DOCKER_HOST is set; unset it and select a named local Docker context instead")

// ErrDockerContextNotLocal is returned when the selected Docker context's
// daemon endpoint is not a recognized local transport for the current
// platform (npipe:// on Windows, unix:// on Linux), the platform itself is
// outside the approved Windows/Linux support matrix, or the endpoint could
// not be determined at all.
var ErrDockerContextNotLocal = errors.New("playtestenv: selected Docker context is not a local context for this platform")

// ErrComposeVersionTooOld is returned when the selected context's `docker
// compose version` is older than minComposeVersion.
var ErrComposeVersionTooOld = errors.New("playtestenv: docker compose version is older than the required minimum")

// ErrDockerServerVersionEmpty is returned when the selected context's
// `docker version --format {{.Server.Version}}` reports an empty string,
// meaning the daemon behind the context could not be reached.
var ErrDockerServerVersionEmpty = errors.New("playtestenv: docker server version is empty")

// dockerVersionTriplePattern extracts a major.minor.patch triple from a
// version string, tolerating an optional leading "v"/"V" and any trailing
// pre-release/build suffix (for example "2.20.0-desktop.1").
var dockerVersionTriplePattern = regexp.MustCompile(`^[vV]?(\d+)\.(\d+)\.(\d+)`)

// dockerContext is the validated outcome of the local-Docker preflight: the
// selected context's name and the exact scrubbed environment that
// preflight already proved local and connectable. The two fields
// deliberately travel together, and both are unexported, so no file
// outside this package - and no other file within it - can construct a
// dockerCommand call against a different context name or a re-scrubbed
// (and therefore potentially stale or unsanitized) environment than the
// one the preflight actually validated. Producing one requires either a
// successful resolveLocalDockerContext call or deliberate literal
// construction with knowledge of this package's internals.
type dockerContext struct {
	name string
	env  []string
}

// resolveLocalDockerContextForHost performs the local-only Docker context
// preflight described on resolveLocalDockerContext, using the current
// process's real environment and platform. It is the single production
// entry point: os.Environ() and runtime.GOOS are read exactly once, here,
// and threaded through the rest of this package only as the returned
// dockerContext value.
func resolveLocalDockerContextForHost(ctx context.Context, runner Runner) (dockerContext, error) {
	return resolveLocalDockerContext(ctx, runner, os.Environ(), runtime.GOOS)
}

// resolveLocalDockerContext is resolveLocalDockerContextForHost with its
// ambient environment and platform injected, so unit tests can exercise
// every branch deterministically without depending on the real host's
// Docker installation or operating system. ambientEnv must be non-nil (see
// ErrAmbientEnvironmentRequired); resolveLocalDockerContextForHost's
// os.Environ() call satisfies this on every real invocation.
//
// It performs, in order:
//
//  1. Reject any nonempty DOCKER_HOST.
//  2. Determine the candidate context: DOCKER_CONTEXT if nonempty,
//     otherwise the output of `docker context show` (run with all four
//     override variables scrubbed from its environment).
//  3. Inspect the candidate's daemon endpoint via `docker --context
//     <candidate> context inspect <candidate> --format
//     {{json .Endpoints.docker.Host}}`, JSON-decode it, and require
//     npipe:// on Windows or unix:// on Linux. Every other GOOS is
//     rejected outright: Windows and Linux are the only approved
//     platforms, so there is no default/fallback local transport.
//  4. Require `docker --context <candidate> compose version --short` to be
//     >= minComposeVersion (accepting an optional leading "v").
//  5. Require `docker --context <candidate> version --format
//     {{.Server.Version}}` to be nonempty.
//
// On Windows, every environment variable name above (DOCKER_HOST,
// DOCKER_CONTEXT, and the four scrubbed override variables) is looked up
// and matched case-insensitively, matching Windows environment-variable
// semantics; on every other platform the match is exact, matching POSIX
// semantics.
//
// Every command issued during this preflight also has the four override
// variables scrubbed from its environment, and no command beyond `docker
// context show` is issued once any step fails. The returned dockerContext
// carries a private clone of the exact scrubbed environment used to
// validate it; every later Docker/Compose command must be built from that
// value via dockerCommand.
func resolveLocalDockerContext(ctx context.Context, runner Runner, ambientEnv []string, goos string) (dockerContext, error) {
	if ambientEnv == nil {
		return dockerContext{}, ErrAmbientEnvironmentRequired
	}

	if hostVal, ok := lookupEnvVar(ambientEnv, "DOCKER_HOST", goos); ok && hostVal != "" {
		return dockerContext{}, ErrDockerHostOverride
	}

	scrubbedEnv := scrubDockerOverrides(ambientEnv, goos)

	candidate, ok := lookupEnvVar(ambientEnv, "DOCKER_CONTEXT", goos)
	if !ok || candidate == "" {
		out, err := runDockerCaptureOutput(ctx, runner, scrubbedEnv, "context", "show")
		if err != nil {
			return dockerContext{}, fmt.Errorf("playtestenv: docker context show: %w", err)
		}
		candidate = strings.TrimSpace(out)
	}
	if candidate == "" {
		return dockerContext{}, fmt.Errorf("%w: docker context show returned an empty context name", ErrDockerContextNotLocal)
	}

	endpoint, err := dockerContextEndpoint(ctx, runner, candidate, scrubbedEnv)
	if err != nil {
		return dockerContext{}, err
	}
	if err := validateLocalDockerEndpoint(endpoint, goos); err != nil {
		return dockerContext{}, err
	}

	if err := checkDockerComposeVersion(ctx, runner, candidate, scrubbedEnv); err != nil {
		return dockerContext{}, err
	}

	if err := checkDockerServerVersion(ctx, runner, candidate, scrubbedEnv); err != nil {
		return dockerContext{}, err
	}

	return dockerContext{name: candidate, env: cloneEnv(scrubbedEnv)}, nil
}

// dockerContextEndpoint runs `docker --context <candidate> context inspect
// <candidate> --format {{json .Endpoints.docker.Host}}` and JSON-decodes
// its output as a plain string.
func dockerContextEndpoint(ctx context.Context, runner Runner, candidate string, env []string) (string, error) {
	out, err := runDockerCaptureOutput(ctx, runner, env,
		dockerArgsWithContext(candidate, []string{"context", "inspect", candidate, "--format", "{{json .Endpoints.docker.Host}}"})...)
	if err != nil {
		return "", fmt.Errorf("playtestenv: docker context inspect: %w", err)
	}

	var endpoint string
	if jsonErr := json.Unmarshal([]byte(strings.TrimSpace(out)), &endpoint); jsonErr != nil {
		return "", fmt.Errorf("%w: could not decode context inspect endpoint %q: %v", ErrDockerContextNotLocal, strings.TrimSpace(out), jsonErr)
	}
	return endpoint, nil
}

// validateLocalDockerEndpoint accepts only npipe:// on Windows and unix://
// on Linux, rejecting tcp://, ssh://, empty, and platform-mismatched
// endpoints. Approved support is explicitly Windows and Linux only: there
// is no default/fallback local transport for any other GOOS (darwin,
// freebsd, etc.), so an unsupported platform is rejected outright even when
// the endpoint would otherwise look like a legitimate local socket.
func validateLocalDockerEndpoint(endpoint, goos string) error {
	var wantPrefix string
	switch goos {
	case "windows":
		wantPrefix = "npipe://"
	case "linux":
		wantPrefix = "unix://"
	default:
		return fmt.Errorf("%w: GOOS %q is not a supported platform (only windows and linux are supported)", ErrDockerContextNotLocal, goos)
	}
	if endpoint != "" && strings.HasPrefix(endpoint, wantPrefix) {
		return nil
	}
	return fmt.Errorf("%w: endpoint %q is not a local %s endpoint for GOOS %q", ErrDockerContextNotLocal, endpoint, wantPrefix, goos)
}

// checkDockerComposeVersion requires `docker --context <candidate> compose
// version --short` to parse as >= minComposeVersion.
func checkDockerComposeVersion(ctx context.Context, runner Runner, candidate string, env []string) error {
	out, err := runDockerCaptureOutput(ctx, runner, env, dockerArgsWithContext(candidate, []string{"compose", "version", "--short"})...)
	if err != nil {
		return fmt.Errorf("playtestenv: docker compose version: %w", err)
	}

	version := strings.TrimSpace(out)
	major, minor, patch, err := parseDockerVersionTriple(version)
	if err != nil {
		return fmt.Errorf("%w: could not parse compose version %q: %v", ErrComposeVersionTooOld, version, err)
	}
	minMajor, minMinor, minPatch, _ := parseDockerVersionTriple(minComposeVersion)
	if !dockerVersionAtLeast(major, minor, patch, minMajor, minMinor, minPatch) {
		return fmt.Errorf("%w: found %q, require >= %s", ErrComposeVersionTooOld, version, minComposeVersion)
	}
	return nil
}

// checkDockerServerVersion requires `docker --context <candidate> version
// --format {{.Server.Version}}` to report a nonempty version string.
func checkDockerServerVersion(ctx context.Context, runner Runner, candidate string, env []string) error {
	out, err := runDockerCaptureOutput(ctx, runner, env, dockerArgsWithContext(candidate, []string{"version", "--format", "{{.Server.Version}}"})...)
	if err != nil {
		return fmt.Errorf("playtestenv: docker version: %w", err)
	}
	if strings.TrimSpace(out) == "" {
		return ErrDockerServerVersionEmpty
	}
	return nil
}

// parseDockerVersionTriple extracts a major.minor.patch triple, tolerating
// an optional leading "v"/"V" and any trailing pre-release/build suffix.
func parseDockerVersionTriple(s string) (major, minor, patch int, err error) {
	m := dockerVersionTriplePattern.FindStringSubmatch(s)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("does not match major.minor.patch: %q", s)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return major, minor, patch, nil
}

// dockerVersionAtLeast reports whether major.minor.patch is >=
// minMajor.minMinor.minPatch.
func dockerVersionAtLeast(major, minor, patch, minMajor, minMinor, minPatch int) bool {
	if major != minMajor {
		return major > minMajor
	}
	if minor != minMinor {
		return minor > minMinor
	}
	return patch >= minPatch
}

// runDockerCaptureOutput runs `docker <args...>` with env and returns its
// captured stdout as a string. On failure, the error includes captured
// stderr so preflight failures are diagnosable without a separate log.
func runDockerCaptureOutput(ctx context.Context, runner Runner, env []string, args ...string) (string, error) {
	var stdout, stderr strings.Builder
	err := runner.Run(ctx, CommandSpec{
		Name:   "docker",
		Args:   args,
		Env:    env,
		Stdout: &stdout,
		Stderr: &stderr,
	})
	if err != nil {
		if stderrText := strings.TrimSpace(stderr.String()); stderrText != "" {
			return "", fmt.Errorf("%w: %s", err, stderrText)
		}
		return "", err
	}
	return stdout.String(), nil
}

// dockerArgsWithContext builds argv beginning "--context <contextName>"
// followed by args. It is the single place that prefixes a Docker context
// flag, shared by the preflight's own probing commands (which have not yet
// produced a validated dockerContext) and by dockerCommand (which has).
func dockerArgsWithContext(contextName string, args []string) []string {
	full := make([]string, 0, len(args)+2)
	full = append(full, "--context", contextName)
	full = append(full, args...)
	return full
}

// dockerCommand is the central constructor every later Docker/Compose
// invocation must use once resolveLocalDockerContext (or
// resolveLocalDockerContextForHost) has returned a validated dockerContext.
// It builds argv beginning "--context <dc.name>" followed by args, and its
// Env is always a clone of dc.env - the exact scrubbed environment already
// proved local and connectable during preflight - never a fresh ambient
// environment. args stays a plain argument slice; it is never joined into
// a shell command line.
//
// Because dockerContext's fields are unexported, no caller can select a
// different context name or environment without either calling
// resolveLocalDockerContext or constructing a dockerContext literal from
// within this package - there is no way to pass a bare string or a fresh
// ambient environment into this function.
func dockerCommand(dc dockerContext, args []string, dir string, stdout, stderr io.Writer) CommandSpec {
	return CommandSpec{
		Name:   "docker",
		Args:   dockerArgsWithContext(dc.name, args),
		Dir:    dir,
		Env:    cloneEnv(dc.env),
		Stdout: stdout,
		Stderr: stderr,
	}
}

// cloneEnv copies an environment slice such as dc.env, preserving the
// distinction between nil and a non-nil empty slice. This distinction
// matters because exec.Cmd treats a nil Env as "inherit the real host
// environment" but a non-nil empty Env as "run with no environment
// variables at all" - so append([]string(nil), env...), which collapses a
// non-nil empty env to nil, would silently reinstate the unsanitized host
// environment (and any DOCKER_* overrides in it) whenever scrubbing (or an
// already-empty ambient environment) legitimately produced zero entries.
func cloneEnv(env []string) []string {
	if env == nil {
		return nil
	}
	cloned := make([]string, len(env))
	copy(cloned, env)
	return cloned
}

// scrubDockerOverrides returns a new slice containing every entry of env
// except the four ambient Docker override variables, matched via
// envNamesEqual (case-insensitive on Windows, exact elsewhere). It never
// mutates env itself, so callers may safely pass os.Environ() directly.
func scrubDockerOverrides(env []string, goos string) []string {
	scrubbed := make([]string, 0, len(env))
	for _, kv := range env {
		if isDockerOverrideVar(envKey(kv), goos) {
			continue
		}
		scrubbed = append(scrubbed, kv)
	}
	return scrubbed
}

// isDockerOverrideVar reports whether key is one of the four ambient Docker
// override variables that must never reach a Docker/Compose child.
func isDockerOverrideVar(key, goos string) bool {
	for _, v := range dockerOverrideVars {
		if envNamesEqual(key, v, goos) {
			return true
		}
	}
	return false
}

// envNamesEqual compares two environment variable names using
// platform-appropriate semantics: case-insensitive on Windows (matching
// how the Windows environment block itself treats variable names) and
// exact/case-sensitive on every other (POSIX) platform.
func envNamesEqual(a, b, goos string) bool {
	if goos == "windows" {
		return strings.EqualFold(a, b)
	}
	return a == b
}

// envKey returns the key portion of a "KEY=VALUE" environment entry.
func envKey(kv string) string {
	key, _, ok := splitEnvEntry(kv)
	if !ok {
		return kv
	}
	return key
}

// splitEnvEntry splits a "KEY=VALUE" environment entry into its key and
// value. ok is false if kv contains no "=".
func splitEnvEntry(kv string) (key, value string, ok bool) {
	idx := strings.IndexByte(kv, '=')
	if idx < 0 {
		return "", "", false
	}
	return kv[:idx], kv[idx+1:], true
}

// lookupEnvVar looks up key in a "KEY=VALUE" slice such as os.Environ(),
// returning its value and whether it was present at all (present-but-empty
// is distinct from absent). The comparison uses envNamesEqual, so the
// lookup is case-insensitive on Windows and case-sensitive elsewhere.
func lookupEnvVar(env []string, key, goos string) (string, bool) {
	for _, kv := range env {
		k, v, ok := splitEnvEntry(kv)
		if !ok {
			continue
		}
		if envNamesEqual(k, key, goos) {
			return v, true
		}
	}
	return "", false
}
