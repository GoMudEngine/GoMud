package playtestenv

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// readinessFakeClock is a controllable clock for readiness polling tests.
type readinessFakeClock struct {
	mu      sync.Mutex
	now     time.Time
	advance time.Duration
}

func newReadinessFakeClock(start time.Time) *readinessFakeClock {
	return &readinessFakeClock{now: start, advance: 200 * time.Millisecond}
}

func (c *readinessFakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *readinessFakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	c.now = c.now.Add(d)
	if c.advance > 0 && d == 0 {
		c.now = c.now.Add(c.advance)
	}
	now := c.now
	c.mu.Unlock()
	ch := make(chan time.Time, 1)
	ch <- now
	return ch
}

type dialResult struct {
	err error
}

type fakeDialer struct {
	mu    sync.Mutex
	calls []string
	queue []dialResult
	err   error
}

func (d *fakeDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	d.mu.Lock()
	d.calls = append(d.calls, network+":"+address)
	var res dialResult
	if len(d.queue) > 0 {
		res = d.queue[0]
		d.queue = d.queue[1:]
	} else {
		res.err = d.err
	}
	d.mu.Unlock()
	if res.err != nil {
		return nil, res.err
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	c1, c2 := net.Pipe()
	_ = c2.Close()
	return c1, nil
}

func inspectJSON(running bool, hostIP, hostPort string, extraMappings ...map[string]string) string {
	type mapping struct {
		HostIp   string `json:"HostIp"`
		HostPort string `json:"HostPort"`
	}
	ports := map[string][]mapping{}
	if hostIP != "" || hostPort != "" {
		entries := []mapping{{HostIp: hostIP, HostPort: hostPort}}
		for _, m := range extraMappings {
			entries = append(entries, mapping{HostIp: m["HostIp"], HostPort: m["HostPort"]})
		}
		ports["55555/tcp"] = entries
	}
	doc := map[string]any{
		"State": map[string]any{
			"Running":    running,
			"Status":     map[bool]string{true: "running", false: "exited"}[running],
			"ExitCode":   map[bool]int{true: 0, false: 1}[running],
			"OOMKilled":  false,
			"Dead":       false,
			"Pid":        map[bool]int{true: 42, false: 0}[running],
			"StartedAt":  "2026-08-08T00:00:00Z",
			"FinishedAt": "0001-01-01T00:00:00Z",
		},
		"Config": map[string]any{
			"Labels": map[string]string{
				"dogmud.playtest.managed": "true",
			},
		},
		"NetworkSettings": map[string]any{
			"Ports": ports,
		},
	}
	b, _ := json.Marshal(doc)
	return string(b)
}

func scriptReadinessDocker(fr *fakeDockerRunner, ctxName, containerID, logs, inspect string) {
	fr.script("--context", ctxName, "compose", "--project-directory", "checkout", "-f", "compose.resolved.yml", "-p", "dogmud-playtest-run-a", "ps", "-q", "server").
		returns(containerID+"\n", "", nil)
	fr.script("--context", ctxName, "inspect", containerID).returns(inspect+"\n", "", nil)
	fr.script("--context", ctxName, "logs", containerID).returns(logs, "", nil)
}

func TestReadinessHappyPathRequiresFinalRunningRecheckAfterTCP(t *testing.T) {
	fr := newFakeDockerRunner()
	ctxName := "desktop-linux"
	containerID := "cidhappy"
	dialer := &fakeDialer{}
	clock := newReadinessFakeClock(time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC))

	inspectRunning := inspectJSON(true, "127.0.0.1", "54321")
	// First observe: running + ready + port. Dial succeeds. Final recheck must inspect again.
	fr.script("--context", ctxName, "inspect", containerID).returns(inspectRunning+"\n", "", nil)
	fr.script("--context", ctxName, "logs", containerID).returns("boot\nServer Ready\n", "", nil)
	// Final running recheck after TCP.
	fr.script("--context", ctxName, "inspect", containerID).returns(inspectRunning+"\n", "", nil)

	dc := dockerContext{name: ctxName, env: []string{}}
	obs, cat, err := evaluateReadiness(context.Background(), fr, dc, containerID, dialer.DialContext, clock.Now())
	require.NoError(t, err)
	require.Equal(t, FailureCategory(""), cat)
	require.True(t, obs.ContainerRunning)
	require.True(t, obs.ServerReady)
	require.True(t, obs.TCPConnected)
	require.NotNil(t, obs.Endpoint)
	require.Equal(t, "127.0.0.1", obs.Endpoint.Host)
	require.Equal(t, 54321, obs.Endpoint.Port)
	require.Equal(t, []string{"tcp:127.0.0.1:54321"}, dialer.calls)
	require.GreaterOrEqual(t, len(fr.calls), 3, "must inspect, fetch logs, then re-inspect after TCP")
	require.Equal(t, []string{"--context", ctxName, "inspect", containerID}, fr.calls[len(fr.calls)-1].Args)
}

func TestReadinessFailureCategories(t *testing.T) {
	ctxName := "desktop-linux"
	containerID := "cid1"
	clockNow := time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name    string
		running bool
		hostIP  string
		port    string
		extra   []map[string]string
		logs    string
		dialErr error
		wantCat FailureCategory
	}{
		{
			name:    "stopped container",
			running: false,
			hostIP:  "127.0.0.1",
			port:    "12345",
			logs:    "Server Ready\n",
			wantCat: FailureContainerExited,
		},
		{
			name:    "panic colon",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "12345",
			logs:    "panic: boom\n",
			wantCat: FailureBootPanic,
		},
		{
			name:    "structured PANIC",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "12345",
			logs:    "ERROR: PANIC recovered\n",
			wantCat: FailureBootPanic,
		},
		{
			name:    "listener creation failure",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "12345",
			logs:    "Error creating server: bind failed\n",
			wantCat: FailureListenerCreation,
		},
		{
			name:    "missing port mapping",
			running: true,
			logs:    "Server Ready\n",
			wantCat: FailurePortPublication,
		},
		{
			name:    "duplicate port mapping",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "12345",
			extra:   []map[string]string{{"HostIp": "127.0.0.1", "HostPort": "12346"}},
			logs:    "Server Ready\n",
			wantCat: FailurePortPublication,
		},
		{
			name:    "malformed port",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "not-a-port",
			logs:    "Server Ready\n",
			wantCat: FailurePortPublication,
		},
		{
			name:    "port zero",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "0",
			logs:    "Server Ready\n",
			wantCat: FailurePortPublication,
		},
		{
			name:    "non-loopback 0.0.0.0",
			running: true,
			hostIP:  "0.0.0.0",
			port:    "12345",
			logs:    "Server Ready\n",
			wantCat: FailureNonLoopback,
		},
		{
			name:    "non-loopback public",
			running: true,
			hostIP:  "203.0.113.9",
			port:    "12345",
			logs:    "Server Ready\n",
			wantCat: FailureNonLoopback,
		},
		{
			name:    "tcp refusal",
			running: true,
			hostIP:  "127.0.0.1",
			port:    "12345",
			logs:    "Server Ready\n",
			dialErr: errors.New("connection refused"),
			wantCat: FailureConnectionProbe,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fr := newFakeDockerRunner()
			inspect := inspectJSON(tc.running, tc.hostIP, tc.port, tc.extra...)
			fr.script("--context", ctxName, "inspect", containerID).returns(inspect+"\n", "", nil)
			fr.script("--context", ctxName, "logs", containerID).returns(tc.logs, "", nil)
			dialer := &fakeDialer{err: tc.dialErr}
			dc := dockerContext{name: ctxName, env: []string{}}

			obs, cat, err := evaluateReadiness(context.Background(), fr, dc, containerID, dialer.DialContext, clockNow)
			require.Error(t, err)
			require.Equal(t, tc.wantCat, cat)
			if tc.wantCat == FailureConnectionProbe {
				require.False(t, obs.TCPConnected)
			}
			if tc.wantCat == FailureBootPanic {
				require.True(t, obs.PanicSeen)
			}
			if tc.wantCat == FailureListenerCreation {
				require.True(t, obs.ListenerError)
			}
		})
	}
}

func TestReadinessTimeoutAndExitAfterPositiveObservation(t *testing.T) {
	t.Run("timeout without server ready", func(t *testing.T) {
		fr := newFakeDockerRunner()
		ctxName := "desktop-linux"
		containerID := "cid-timeout"
		clock := newReadinessFakeClock(time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC))
		inspect := inspectJSON(true, "127.0.0.1", "12345")

		// Match any number of poll iterations.
		fr.script("--context", ctxName, "inspect", containerID).returns(inspect+"\n", "", nil)
		fr.script("--context", ctxName, "logs", containerID).returns("still booting\n", "", nil)

		// Override Run to keep returning the same responses for repeated polls.
		base := fr.Run
		fr.mu.Lock()
		_ = base
		fr.mu.Unlock()

		repeating := &repeatingReadinessRunner{
			inspect: inspect + "\n",
			logs:    "still booting\n",
		}

		dc := dockerContext{name: ctxName, env: []string{}}
		obs, cat, err := waitForReadiness(
			context.Background(),
			repeating,
			dc,
			containerID,
			50*time.Millisecond,
			(&fakeDialer{}).DialContext,
			clock.Now,
			clock.After,
		)
		require.Error(t, err)
		require.Equal(t, FailureReadinessTimeout, cat)
		require.False(t, obs.ServerReady)
		require.GreaterOrEqual(t, repeating.polls, 1)
	})

	t.Run("container exits after TCP success on final recheck", func(t *testing.T) {
		fr := newFakeDockerRunner()
		ctxName := "desktop-linux"
		containerID := "cid-flip"
		clock := newReadinessFakeClock(time.Date(2026, 8, 8, 0, 0, 0, 0, time.UTC))
		running := inspectJSON(true, "127.0.0.1", "12345")
		stopped := inspectJSON(false, "127.0.0.1", "12345")

		seq := &sequentialInspectRunner{
			inspectSeq: []string{running + "\n", stopped + "\n"},
			logs:       "Server Ready\n",
		}
		dc := dockerContext{name: ctxName, env: []string{}}
		obs, cat, err := evaluateReadiness(context.Background(), seq, dc, containerID, (&fakeDialer{}).DialContext, clock.Now())
		_ = fr
		require.Error(t, err)
		require.Equal(t, FailureContainerExited, cat)
		require.True(t, obs.ServerReady)
		require.True(t, obs.TCPConnected)
		require.False(t, obs.ContainerRunning)
	})
}

func TestReadinessAcceptsIPv6Loopback(t *testing.T) {
	fr := newFakeDockerRunner()
	ctxName := "desktop-linux"
	containerID := "cid6"
	inspect := inspectJSON(true, "::1", "5555")
	fr.script("--context", ctxName, "inspect", containerID).returns(inspect+"\n", "", nil)
	fr.script("--context", ctxName, "logs", containerID).returns("Server Ready\n", "", nil)
	fr.script("--context", ctxName, "inspect", containerID).returns(inspect+"\n", "", nil)

	dialer := &fakeDialer{}
	dc := dockerContext{name: ctxName, env: []string{}}
	obs, cat, err := evaluateReadiness(context.Background(), fr, dc, containerID, dialer.DialContext, time.Now())
	require.NoError(t, err)
	require.Equal(t, FailureCategory(""), cat)
	require.Equal(t, "::1", obs.Endpoint.Host)
	require.Equal(t, 5555, obs.Endpoint.Port)
	require.Equal(t, "tcp:[::1]:5555", dialer.calls[0])
}

// repeatingReadinessRunner always returns the same inspect/logs payload.
type repeatingReadinessRunner struct {
	mu      sync.Mutex
	polls   int
	inspect string
	logs    string
}

func (r *repeatingReadinessRunner) Run(ctx context.Context, spec CommandSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	args := strings.Join(spec.Args, " ")
	switch {
	case strings.Contains(args, " inspect "):
		r.polls++
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, r.inspect)
		}
	case strings.Contains(args, " logs "):
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, r.logs)
		}
	default:
		return fmt.Errorf("repeatingReadinessRunner: unexpected args %v", spec.Args)
	}
	return ctx.Err()
}

type sequentialInspectRunner struct {
	mu         sync.Mutex
	inspectSeq []string
	logs       string
	idx        int
}

func (r *sequentialInspectRunner) Run(ctx context.Context, spec CommandSpec) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	args := strings.Join(spec.Args, " ")
	switch {
	case strings.Contains(args, " inspect "):
		if r.idx >= len(r.inspectSeq) {
			return fmt.Errorf("sequentialInspectRunner: no more inspect responses")
		}
		out := r.inspectSeq[r.idx]
		r.idx++
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, out)
		}
	case strings.Contains(args, " logs "):
		if spec.Stdout != nil {
			_, _ = io.WriteString(spec.Stdout, r.logs)
		}
	default:
		return fmt.Errorf("sequentialInspectRunner: unexpected args %v", spec.Args)
	}
	return nil
}
