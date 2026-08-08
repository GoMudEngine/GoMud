package playtestenv

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// Test-only Compose policy helpers. These mutate the package-level
// embeddedComposePolicy under a mutex so Start materializes the override.
// They are unreachable from New() and the CLI (constructors live only in
// *_test.go). Concurrent success cases must not run while an override is
// installed.

var (
	composeOverrideMu     sync.Mutex
	composePolicyOriginal []byte
)

func init() {
	composePolicyOriginal = append([]byte(nil), embeddedComposePolicy...)
}

// withTestComposePolicy installs policy for the remainder of the test and
// restores the embedded default afterward.
func withTestComposePolicy(t *testing.T, policy []byte) {
	t.Helper()
	composeOverrideMu.Lock()
	embeddedComposePolicy = append([]byte(nil), policy...)
	t.Cleanup(func() {
		embeddedComposePolicy = append([]byte(nil), composePolicyOriginal...)
		composeOverrideMu.Unlock()
	})
}

func testComposePolicyNoPort(t *testing.T) []byte {
	t.Helper()
	return mutateEmbeddedCompose(t, func(server map[string]any) {
		delete(server, "ports")
	})
}

func testComposePolicyNonLoopback(t *testing.T) []byte {
	t.Helper()
	return mutateEmbeddedCompose(t, func(server map[string]any) {
		server["ports"] = []any{
			map[string]any{
				"target":   55555,
				"host_ip":  "0.0.0.0",
				"protocol": "tcp",
			},
		}
	})
}

// testComposePolicyIgnoreTERM wraps the server binary so PID 1 is a shell
// that ignores SIGTERM. Graceful stop must then fall through to force-remove.
func testComposePolicyIgnoreTERM(t *testing.T) []byte {
	t.Helper()
	return mutateEmbeddedCompose(t, func(server map[string]any) {
		server["entrypoint"] = []any{"/bin/sh", "-c"}
		server["command"] = []any{"trap '' TERM; /app/go-mud-server & wait"}
	})
}

func mutateEmbeddedCompose(t *testing.T, mut func(server map[string]any)) []byte {
	t.Helper()
	var doc map[string]any
	require.NoError(t, yaml.Unmarshal(composePolicyOriginal, &doc))
	services, ok := doc["services"].(map[string]any)
	require.True(t, ok, "compose services map")
	server, ok := services["server"].(map[string]any)
	require.True(t, ok, "compose server service")
	mut(server)
	out, err := yaml.Marshal(doc)
	require.NoError(t, err)
	return out
}
