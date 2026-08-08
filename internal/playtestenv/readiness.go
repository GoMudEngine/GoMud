package playtestenv

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strconv"
	"strings"
	"time"
)

const (
	aiPublishedPortKey    = "55555/tcp"
	serverReadyMarker     = "Server Ready"
	listenerErrorMarker   = "Error creating server"
	panicMarker           = "panic:"
	structuredPanicMarker = "PANIC"
	readinessPollInterval = 200 * time.Millisecond
)

// dialFunc is the injectable TCP probe used by readiness checks.
type dialFunc func(ctx context.Context, network, address string) (net.Conn, error)

type containerInspectDoc struct {
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	NetworkSettings struct {
		Ports map[string][]struct {
			HostIp   string `json:"HostIp"`
			HostPort string `json:"HostPort"`
		} `json:"Ports"`
	} `json:"NetworkSettings"`
}

// evaluateReadiness performs one compound readiness observation against an
// already-resolved container ID: inspect + logs + port policy + TCP probe +
// a final running recheck after a successful TCP dial.
func evaluateReadiness(
	ctx context.Context,
	runner Runner,
	dc dockerContext,
	containerID string,
	dial dialFunc,
	observedAt time.Time,
) (ReadinessObservation, FailureCategory, error) {
	obs := ReadinessObservation{ObservedAt: observedAt}

	doc, err := inspectContainer(ctx, runner, dc, containerID)
	if err != nil {
		return obs, FailureContainerExited, err
	}
	obs.ContainerRunning = doc.State.Running
	if !obs.ContainerRunning {
		return obs, FailureContainerExited, fmt.Errorf("playtestenv: container %s is not running", containerID)
	}

	logs, err := containerLogs(ctx, runner, dc, containerID)
	if err != nil {
		return obs, FailureReadinessTimeout, err
	}
	classifyLogMarkers(logs, &obs)
	if obs.PanicSeen {
		return obs, FailureBootPanic, fmt.Errorf("playtestenv: panic observed in container logs")
	}
	if obs.ListenerError {
		return obs, FailureListenerCreation, fmt.Errorf("playtestenv: listener creation failure observed in container logs")
	}
	if !obs.ServerReady {
		return obs, FailureReadinessTimeout, fmt.Errorf("playtestenv: %q not yet observed", serverReadyMarker)
	}

	endpoint, mappings, cat, err := parsePublishedAIEndpoint(doc)
	obs.PortMappings = mappings
	if err != nil {
		return obs, cat, err
	}
	obs.Endpoint = endpoint

	addr := net.JoinHostPort(endpoint.Host, strconv.Itoa(endpoint.Port))
	conn, err := dial(ctx, "tcp", addr)
	if err != nil {
		return obs, FailureConnectionProbe, fmt.Errorf("playtestenv: tcp probe failed: %w", err)
	}
	_ = conn.Close()
	obs.TCPConnected = true

	// Final running recheck AFTER TCP success.
	finalDoc, err := inspectContainer(ctx, runner, dc, containerID)
	if err != nil {
		obs.ContainerRunning = false
		return obs, FailureContainerExited, err
	}
	obs.ContainerRunning = finalDoc.State.Running
	if !obs.ContainerRunning {
		return obs, FailureContainerExited, fmt.Errorf("playtestenv: container exited after tcp probe succeeded")
	}

	return obs, "", nil
}

// waitForReadiness polls evaluateReadiness until success, a terminal failure
// category, caller cancellation, or timeout. Soft "not ready yet" observations
// (missing Server Ready, transient TCP refusal) are retried until the deadline.
func waitForReadiness(
	ctx context.Context,
	runner Runner,
	dc dockerContext,
	containerID string,
	timeout time.Duration,
	dial dialFunc,
	now func() time.Time,
	after func(time.Duration) <-chan time.Time,
) (ReadinessObservation, FailureCategory, error) {
	if timeout <= 0 {
		timeout = DefaultReadinessTimeout
	}
	deadline := now().Add(timeout)
	var last ReadinessObservation
	var lastCat FailureCategory
	var lastErr error

	for {
		obs, cat, err := evaluateReadiness(ctx, runner, dc, containerID, dial, now())
		last, lastCat, lastErr = obs, cat, err
		if err == nil {
			return obs, "", nil
		}
		if isTerminalReadinessFailure(cat) {
			return obs, cat, err
		}
		// Soft failures: keep polling until timeout.
		if !now().Before(deadline) {
			if lastCat == FailureConnectionProbe {
				return last, FailureConnectionProbe, lastErr
			}
			return last, FailureReadinessTimeout, fmt.Errorf("playtestenv: readiness timed out after %s: %w", timeout, lastErr)
		}
		select {
		case <-ctx.Done():
			return last, lastCat, ctx.Err()
		case <-after(readinessPollInterval):
		}
	}
}

func isTerminalReadinessFailure(cat FailureCategory) bool {
	switch cat {
	case FailureContainerExited, FailureBootPanic, FailureListenerCreation,
		FailurePortPublication, FailureNonLoopback:
		return true
	default:
		return false
	}
}

func inspectContainer(ctx context.Context, runner Runner, dc dockerContext, containerID string) (containerInspectDoc, error) {
	var stdout, stderr strings.Builder
	spec := dockerCommand(dc, []string{"inspect", containerID}, "", &stdout, &stderr)
	if err := runner.Run(ctx, spec); err != nil {
		if strings.TrimSpace(stderr.String()) != "" {
			return containerInspectDoc{}, fmt.Errorf("playtestenv: docker inspect: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return containerInspectDoc{}, fmt.Errorf("playtestenv: docker inspect: %w", err)
	}
	raw := strings.TrimSpace(stdout.String())
	// docker inspect returns a JSON array when no --format is given.
	if strings.HasPrefix(raw, "[") {
		var docs []containerInspectDoc
		if err := json.Unmarshal([]byte(raw), &docs); err != nil {
			return containerInspectDoc{}, fmt.Errorf("playtestenv: decode docker inspect: %w", err)
		}
		if len(docs) != 1 {
			return containerInspectDoc{}, fmt.Errorf("playtestenv: docker inspect returned %d objects, want 1", len(docs))
		}
		return docs[0], nil
	}
	var doc containerInspectDoc
	if err := json.Unmarshal([]byte(raw), &doc); err != nil {
		return containerInspectDoc{}, fmt.Errorf("playtestenv: decode docker inspect: %w", err)
	}
	return doc, nil
}

func containerLogs(ctx context.Context, runner Runner, dc dockerContext, containerID string) (string, error) {
	var stdout, stderr strings.Builder
	spec := dockerCommand(dc, []string{"logs", containerID}, "", &stdout, &stderr)
	if err := runner.Run(ctx, spec); err != nil {
		if strings.TrimSpace(stderr.String()) != "" {
			return stdout.String(), fmt.Errorf("playtestenv: docker logs: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return stdout.String(), fmt.Errorf("playtestenv: docker logs: %w", err)
	}
	// docker logs writes the container's stdout/stderr streams; some CLIs put
	// log text on the process stderr writer. Combine both for marker scans.
	return stdout.String() + stderr.String(), nil
}

func classifyLogMarkers(logs string, obs *ReadinessObservation) {
	if strings.Contains(logs, panicMarker) || strings.Contains(logs, structuredPanicMarker) {
		obs.PanicSeen = true
	}
	if strings.Contains(logs, listenerErrorMarker) {
		obs.ListenerError = true
	}
	if strings.Contains(logs, serverReadyMarker) {
		obs.ServerReady = true
	}
}

func parsePublishedAIEndpoint(doc containerInspectDoc) (*Endpoint, int, FailureCategory, error) {
	mappings := doc.NetworkSettings.Ports[aiPublishedPortKey]
	n := len(mappings)
	if n != 1 {
		return nil, n, FailurePortPublication, fmt.Errorf("playtestenv: want exactly one %s mapping, found %d", aiPublishedPortKey, n)
	}
	hostIP := strings.TrimSpace(mappings[0].HostIp)
	hostPort := strings.TrimSpace(mappings[0].HostPort)
	if hostIP == "" || hostPort == "" {
		return nil, n, FailurePortPublication, fmt.Errorf("playtestenv: malformed %s mapping", aiPublishedPortKey)
	}
	if !isLoopbackHost(hostIP) {
		return nil, n, FailureNonLoopback, fmt.Errorf("playtestenv: published host %q is not loopback", hostIP)
	}
	port, err := strconv.Atoi(hostPort)
	if err != nil || port < 1 || port > 65535 {
		return nil, n, FailurePortPublication, fmt.Errorf("playtestenv: invalid published host port %q", hostPort)
	}
	return &Endpoint{Host: hostIP, Port: port}, n, "", nil
}

func isLoopbackHost(host string) bool {
	if host == "0.0.0.0" || host == "::" || host == "[::]" {
		return false
	}
	ip := net.ParseIP(strings.Trim(host, "[]"))
	if ip == nil {
		return false
	}
	return ip.IsLoopback()
}

// resolveComposeContainerID runs `compose ps -q server` and returns the
// trimmed container ID.
func resolveComposeContainerID(ctx context.Context, runner Runner, dc dockerContext, vars composeRunVars, composeFile, dir string) (string, error) {
	var stdout, stderr strings.Builder
	spec := composeCommand(dc, vars, composeFile, []string{"ps", "-q", "server"}, dir, &stdout, &stderr)
	if err := runner.Run(ctx, spec); err != nil {
		if strings.TrimSpace(stderr.String()) != "" {
			return "", fmt.Errorf("playtestenv: compose ps: %w: %s", err, strings.TrimSpace(stderr.String()))
		}
		return "", fmt.Errorf("playtestenv: compose ps: %w", err)
	}
	id := strings.TrimSpace(stdout.String())
	if id == "" {
		return "", fmt.Errorf("playtestenv: compose ps returned no container id")
	}
	// Compose may print multiple IDs; require exactly one nonempty line.
	lines := strings.Split(id, "\n")
	var cleaned []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			cleaned = append(cleaned, line)
		}
	}
	if len(cleaned) != 1 {
		return "", fmt.Errorf("playtestenv: compose ps returned %d container ids, want 1", len(cleaned))
	}
	return cleaned[0], nil
}
