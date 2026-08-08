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

// ErrDockerHostOverride is returned when the ambient environment sets a
// nonempty DOCKER_HOST. This package never talks to a daemon selected by
// DOCKER_HOST; callers must select a named local Docker context instead.
var ErrDockerHostOverride = errors.New("playtestenv: DOCKER_HOST is set; unset it and select a named local Docker context instead")

// ErrDockerContextNotLocal is returned when the selected Docker context's
// daemon endpoint is not a recognized local transport for the current
// platform (npipe:// on Windows, unix:// elsewhere), or could not be
// determined at all.
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

// resolveLocalDockerContextForHost performs the local-only Docker context
// preflight described in this file's package documentation, using the
// current process's real environment and platform. The returned string is
// the only Docker context value later code may use; it must be threaded
// through dockerCommand for every subsequent Docker/Compose invocation.
func resolveLocalDockerContextForHost(ctx context.Context, runner Runner) (string, error) {
	return resolveLocalDockerContext(ctx, runner, os.Environ(), runtime.GOOS)
}

// resolveLocalDockerContext is resolveLocalDockerContextForHost with its
// ambient environment and platform injected, so unit tests can exercise
// every branch deterministically without depending on the real host's
// Docker installation or operating system.
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
//     npipe:// on Windows or unix:// elsewhere.
//  4. Require `docker --context <candidate> compose version --short` to be
//     >= minComposeVersion (accepting an optional leading "v").
//  5. Require `docker --context <candidate> version --format
//     {{.Server.Version}}` to be nonempty.
//
// Every command issued during this preflight also has the four override
// variables scrubbed from its environment, and no command beyond `docker
// context show` is issued once any step fails.
func resolveLocalDockerContext(ctx context.Context, runner Runner, ambientEnv []string, goos string) (string, error) {
	if hostVal, ok := lookupEnvVar(ambientEnv, "DOCKER_HOST"); ok && hostVal != "" {
		return "", ErrDockerHostOverride
	}

	scrubbedEnv := scrubDockerOverrides(ambientEnv)

	candidate, ok := lookupEnvVar(ambientEnv, "DOCKER_CONTEXT")
	if !ok || candidate == "" {
		out, err := runDockerCaptureOutput(ctx, runner, scrubbedEnv, "context", "show")
		if err != nil {
			return "", fmt.Errorf("playtestenv: docker context show: %w", err)
		}
		candidate = strings.TrimSpace(out)
	}
	if candidate == "" {
		return "", fmt.Errorf("%w: docker context show returned an empty context name", ErrDockerContextNotLocal)
	}

	endpoint, err := dockerContextEndpoint(ctx, runner, candidate, scrubbedEnv)
	if err != nil {
		return "", err
	}
	if err := validateLocalDockerEndpoint(endpoint, goos); err != nil {
		return "", err
	}

	if err := checkDockerComposeVersion(ctx, runner, candidate, scrubbedEnv); err != nil {
		return "", err
	}

	if err := checkDockerServerVersion(ctx, runner, candidate, scrubbedEnv); err != nil {
		return "", err
	}

	return candidate, nil
}

// dockerContextEndpoint runs `docker --context <candidate> context inspect
// <candidate> --format {{json .Endpoints.docker.Host}}` and JSON-decodes
// its output as a plain string.
func dockerContextEndpoint(ctx context.Context, runner Runner, candidate string, env []string) (string, error) {
	out, err := runDockerCaptureOutput(ctx, runner, env,
		"--context", candidate, "context", "inspect", candidate, "--format", "{{json .Endpoints.docker.Host}}")
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
// on every other platform, rejecting tcp://, ssh://, empty, and
// platform-mismatched endpoints.
func validateLocalDockerEndpoint(endpoint, goos string) error {
	wantPrefix := "unix://"
	if goos == "windows" {
		wantPrefix = "npipe://"
	}
	if endpoint != "" && strings.HasPrefix(endpoint, wantPrefix) {
		return nil
	}
	return fmt.Errorf("%w: endpoint %q is not a local %s endpoint for GOOS %q", ErrDockerContextNotLocal, endpoint, wantPrefix, goos)
}

// checkDockerComposeVersion requires `docker --context <candidate> compose
// version --short` to parse as >= minComposeVersion.
func checkDockerComposeVersion(ctx context.Context, runner Runner, candidate string, env []string) error {
	out, err := runDockerCaptureOutput(ctx, runner, env, "--context", candidate, "compose", "version", "--short")
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
	out, err := runDockerCaptureOutput(ctx, runner, env, "--context", candidate, "version", "--format", "{{.Server.Version}}")
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

// dockerCommand is the central constructor every later Docker/Compose
// invocation must use once resolveLocalDockerContext has returned a
// validated context name. It always builds argv beginning "--context
// <validatedContext>" followed by args, and it always scrubs the four
// ambient override variables from ambientEnv before use, so no caller needs
// to repeat that scrubbing itself. args stays a plain argument slice - it
// is never joined into a shell command line.
//
// The Docker context concept is intentionally not exposed as its own type
// outside this file: every other file in this package receives and passes
// along only the plain validated context string returned by
// resolveLocalDockerContext.
func dockerCommand(validatedContext string, args []string, dir string, ambientEnv []string, stdout, stderr io.Writer) CommandSpec {
	fullArgs := make([]string, 0, len(args)+2)
	fullArgs = append(fullArgs, "--context", validatedContext)
	fullArgs = append(fullArgs, args...)

	return CommandSpec{
		Name:   "docker",
		Args:   fullArgs,
		Dir:    dir,
		Env:    scrubDockerOverrides(ambientEnv),
		Stdout: stdout,
		Stderr: stderr,
	}
}

// scrubDockerOverrides returns a new slice containing every entry of env
// except the four ambient Docker override variables. It never mutates env
// itself, so callers may safely pass os.Environ() directly.
func scrubDockerOverrides(env []string) []string {
	scrubbed := make([]string, 0, len(env))
	for _, kv := range env {
		if isDockerOverrideVar(envKey(kv)) {
			continue
		}
		scrubbed = append(scrubbed, kv)
	}
	return scrubbed
}

// isDockerOverrideVar reports whether key is one of the four ambient Docker
// override variables that must never reach a Docker/Compose child.
func isDockerOverrideVar(key string) bool {
	for _, v := range dockerOverrideVars {
		if key == v {
			return true
		}
	}
	return false
}

// envKey returns the key portion of a "KEY=VALUE" environment entry.
func envKey(kv string) string {
	if idx := strings.IndexByte(kv, '='); idx >= 0 {
		return kv[:idx]
	}
	return kv
}

// lookupEnvVar looks up key in a "KEY=VALUE" slice such as os.Environ(),
// returning its value and whether it was present at all (present-but-empty
// is distinct from absent).
func lookupEnvVar(env []string, key string) (string, bool) {
	prefix := key + "="
	for _, kv := range env {
		if strings.HasPrefix(kv, prefix) {
			return kv[len(prefix):], true
		}
	}
	return "", false
}
