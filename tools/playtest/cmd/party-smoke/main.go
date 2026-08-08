// Command party-smoke is a pre-merge 0.3d live smoke: playtestrun scenario
// party-formation + two mudagents exercising party invite/accept.
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type readyActor struct {
	ID        string  `json:"id"`
	BridgeDir string  `json:"bridge_dir"`
	Creds     *string `json:"creds"`
	Username  string  `json:"username"`
}

type readyPayload struct {
	RunID         string `json:"run_id"`
	Checkout      string `json:"checkout"`
	Commit        string `json:"commit"`
	Dirty         bool   `json:"dirty"`
	OnActorStop   string `json:"on_actor_stop"`
	BlackboardDir string `json:"blackboard_dir"`
	Endpoint      *struct {
		Host string `json:"host"`
		Port int    `json:"port"`
	} `json:"endpoint"`
	Actors []readyActor `json:"actors"`
}

type credsFile struct {
	Players []struct {
		ActorID  string `json:"actor_id"`
		Username string `json:"username"`
		Password string `json:"password"`
	} `json:"players"`
}

func main() {
	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "party-smoke: %v\n", err)
		os.Exit(1)
	}
}

func run() error {
	checkout, err := os.Getwd()
	if err != nil {
		return err
	}
	harness := os.Getenv("GOMUD_HARNESS_DIR")
	if harness == "" {
		harness = filepath.Join(filepath.Dir(checkout), "gomud-playtest-harness")
	}
	mudagent := filepath.Join(harness, "mudagent.exe")
	if _, err := os.Stat(mudagent); err != nil {
		return fmt.Errorf("mudagent not found at %s", mudagent)
	}

	bin := filepath.Join(checkout, "playtestrun-smoke.exe")
	if out, err := exec.Command("go", "build", "-o", bin, "./cmd/playtestrun").CombinedOutput(); err != nil {
		return fmt.Errorf("build playtestrun: %w\n%s", err, out)
	}

	scenario := filepath.Join(checkout, "tools", "playtest", "scenarios", "party-formation.yaml")
	cmd := exec.Command(bin, "scenario",
		"--checkout", checkout,
		"--scenario", scenario,
		"--wall-clock", "12m",
	)
	cmd.Dir = checkout
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return err
	}
	if err := cmd.Start(); err != nil {
		return err
	}

	go func() {
		sc := bufio.NewScanner(stderr)
		for sc.Scan() {
			fmt.Fprintln(os.Stderr, sc.Text())
		}
	}()

	ready, err := readReady(stdout, 20*time.Minute)
	if err != nil {
		_ = cmd.Process.Kill()
		return err
	}
	fmt.Fprintf(os.Stderr, "ready run_id=%s endpoint=%s:%d\n", ready.RunID, ready.Endpoint.Host, ready.Endpoint.Port)

	leader := actorByID(ready, "leader")
	joiner := actorByID(ready, "joiner")
	if leader == nil || joiner == nil {
		return fmt.Errorf("ready JSON missing leader/joiner")
	}
	lUser, lPass, err := credsFor(*leader.Creds, "leader")
	if err != nil {
		return err
	}
	jUser, jPass, err := credsFor(*joiner.Creds, "joiner")
	if err != nil {
		return err
	}

	lAgent, err := startMudagent(mudagent, harness, leader.BridgeDir, ready.Endpoint.Host, ready.Endpoint.Port, lUser, lPass)
	if err != nil {
		return err
	}
	defer lAgent.close()
	jAgent, err := startMudagent(mudagent, harness, joiner.BridgeDir, ready.Endpoint.Host, ready.Endpoint.Port, jUser, jPass)
	if err != nil {
		return err
	}
	defer jAgent.close()

	time.Sleep(12 * time.Second)
	_ = lAgent.send("look")
	_ = jAgent.send("look")
	_ = lAgent.send("who")
	_ = jAgent.send("who")
	time.Sleep(5 * time.Second)

	// Party invite targets character names (not login usernames).
	joinerChar := waitCapture(joiner.BridgeDir, regexp.MustCompile(`(?i)Fresh Recruit`), 30*time.Second)
	if joinerChar == "" {
		joinerChar = "Fresh Recruit"
	}
	_ = writeBB(ready.BlackboardDir, "joiner-ready", "joiner", map[string]any{
		"username":       jUser,
		"character_name": joinerChar,
	})
	_ = lAgent.send("party invite " + joinerChar)
	invited := waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)You invited`), 90*time.Second) ||
		waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)invited you to their party`), 5*time.Second)
	_ = writeBB(ready.BlackboardDir, "leader-invited", "leader", map[string]any{"invited_seen": invited, "target": joinerChar})

	_ = jAgent.send("party accept")
	joined := waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)joined the party`), 90*time.Second) ||
		waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)joined the party`), 5*time.Second)
	_ = writeBB(ready.BlackboardDir, "joiner-accepted", "joiner", map[string]any{"joined_seen": joined})

	_ = lAgent.send("party list")
	_ = jAgent.send("party list")
	_ = lAgent.send("party say Greetings from the leader")
	time.Sleep(8 * time.Second)
	listOK := waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)Party Members`), 30*time.Second) &&
		(waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)Party Members`), 5*time.Second) ||
			waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)Early Strider`), 5*time.Second))

	lAgent.close()
	jAgent.close()

	stop := exec.Command(bin, "stop", "--checkout", checkout, "--run", ready.RunID)
	_ = stop.Run()
	_ = cmd.Wait()

	signals := listSignals(ready.BlackboardDir)
	overall := invited && joined
	reportPath := filepath.Join(checkout, "tools", "playtest", "reports", "2026-08-08-party-formation-smoke.md")
	if err := writeReport(reportPath, ready, invited, joined, listOK, overall, signals, leader.BridgeDir, joiner.BridgeDir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "report=%s overall=%v invite=%v accept=%v list=%v\n", reportPath, overall, invited, joined, listOK)
	if !overall {
		return fmt.Errorf("smoke incomplete: invite=%v accept=%v", invited, joined)
	}
	return nil
}

func readReady(r io.Reader, timeout time.Duration) (readyPayload, error) {
	type result struct {
		p   readyPayload
		err error
	}
	ch := make(chan result, 1)
	go func() {
		sc := bufio.NewScanner(r)
		// Ready JSON can be large.
		buf := make([]byte, 0, 64*1024)
		sc.Buffer(buf, 1024*1024)
		for sc.Scan() {
			line := strings.TrimSpace(sc.Text())
			if line == "" {
				continue
			}
			var p readyPayload
			if err := json.Unmarshal([]byte(line), &p); err != nil {
				continue
			}
			if p.RunID != "" && p.Endpoint != nil {
				ch <- result{p: p}
				return
			}
		}
		ch <- result{err: fmt.Errorf("stdout closed before ready JSON: %w", sc.Err())}
	}()
	select {
	case res := <-ch:
		return res.p, res.err
	case <-time.After(timeout):
		return readyPayload{}, fmt.Errorf("timeout waiting for ready JSON")
	}
}

func actorByID(r readyPayload, id string) *readyActor {
	for i := range r.Actors {
		if r.Actors[i].ID == id {
			return &r.Actors[i]
		}
	}
	return nil
}

func credsFor(path, actorID string) (user, pass string, err error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	var f credsFile
	if err := json.Unmarshal(raw, &f); err != nil {
		return "", "", err
	}
	for _, p := range f.Players {
		if p.ActorID == actorID {
			return p.Username, p.Password, nil
		}
	}
	return "", "", fmt.Errorf("actor_id %q not in %s", actorID, path)
}

type agent struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	bridge string
}

func startMudagent(bin, harness, bridge, host string, port int, user, pass string) (*agent, error) {
	if err := os.MkdirAll(bridge, 0o755); err != nil {
		return nil, err
	}
	evt := filepath.Join(bridge, "events.jsonl")
	errPath := filepath.Join(bridge, "mudagent.err")
	_ = os.WriteFile(evt, nil, 0o644)
	_ = os.WriteFile(filepath.Join(bridge, "commands.txt"), nil, 0o644)

	cmd := exec.Command(bin,
		"--target", fmt.Sprintf("%s:%d", host, port),
		"--user", user,
		"--password", pass,
	)
	cmd.Dir = harness
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	outF, err := os.OpenFile(evt, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return nil, err
	}
	errF, err := os.OpenFile(errPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		_ = outF.Close()
		return nil, err
	}
	cmd.Stdout = outF
	cmd.Stderr = errF
	if err := cmd.Start(); err != nil {
		_ = outF.Close()
		_ = errF.Close()
		return nil, err
	}
	// Close file handles in parent after Start duplicates them for the child.
	_ = outF.Close()
	_ = errF.Close()
	return &agent{cmd: cmd, stdin: stdin, bridge: bridge}, nil
}

func (a *agent) send(line string) error {
	if a == nil || a.stdin == nil {
		return fmt.Errorf("agent closed")
	}
	_, _ = os.OpenFile(filepath.Join(a.bridge, "commands.txt"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	f, err := os.OpenFile(filepath.Join(a.bridge, "commands.txt"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644)
	if err == nil {
		_, _ = f.WriteString(line + "\n")
		_ = f.Close()
	}
	_, err = io.WriteString(a.stdin, line+"\n")
	return err
}

func (a *agent) close() {
	if a == nil {
		return
	}
	if a.stdin != nil {
		_ = a.stdin.Close()
	}
	if a.cmd != nil && a.cmd.Process != nil {
		_ = a.cmd.Process.Kill()
		_, _ = a.cmd.Process.Wait()
	}
}

func waitMatch(bridge string, re *regexp.Regexp, timeout time.Duration) bool {
	return waitCapture(bridge, re, timeout) != ""
}

func waitCapture(bridge string, re *regexp.Regexp, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	path := filepath.Join(bridge, "events.jsonl")
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			if m := re.Find(raw); m != nil {
				return string(m)
			}
		}
		time.Sleep(time.Second)
	}
	return ""
}

func writeBB(dir, signal, actorID string, data map[string]any) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	payload := map[string]any{
		"signal":   signal,
		"actor_id": actorID,
		"ts":       time.Now().UTC().Format(time.RFC3339),
		"data":     data,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	tmp := filepath.Join(dir, "."+signal+".tmp")
	dst := filepath.Join(dir, signal+".json")
	if err := os.WriteFile(tmp, append(raw, '\n'), 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dst)
}

func listSignals(dir string) []string {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".json") {
			out = append(out, strings.TrimSuffix(e.Name(), ".json"))
		}
	}
	return out
}

func writeReport(path string, ready readyPayload, invited, joined, listOK, overall bool, signals []string, leaderBridge, joinerBridge string) error {
	mark := func(ok bool) string {
		if ok {
			return "[x]"
		}
		return "[ ]"
	}
	body := fmt.Sprintf(`# Multi-Agent Playtest Report: party-formation

**Date:** 2026-08-08
**Scenario:** party-formation (mode: party)
**Checkout:** %s (commit: %s, dirty: %v)
**Run ID:** %s
**On actor stop:** %s
**Wall-clock:** 12m (hard cut)
**Agents:** leader (feature-tester, early), joiner (feel-tester, fresh)
**Endpoint:** %s:%d (ephemeral; passwords omitted)

## Summary
Driver-contract + live mudagent smoke for 0.3d: shared ephemeral env, two
actor bridges, actor_id login, file blackboard signals, and party invite/accept.

## Group Goal Results
- %s invite — invit* seen=%v
- %s accept — join* seen=%v
- %s party-list/chat text — seen=%v

## Per-Agent Outcomes
- leader (feature-tester): bridge=%s
- joiner (feel-tester): bridge=%s

## Blackboard
- Dir: %s
- Signals observed: %s

## Findings
### OBSERVATION: automated mudagent smoke
Not a full LLM personality run. Evidence from mudagent events.jsonl regex
matches for invite/join/party text after actor_id login.

## Stats
- Agents: 2
- Overall smoke: %s
- Scenario sidecar: tools/playtest/.run/%s/session.json
`,
		ready.Checkout, ready.Commit, ready.Dirty,
		ready.RunID, ready.OnActorStop,
		ready.Endpoint.Host, ready.Endpoint.Port,
		mark(invited), invited,
		mark(joined), joined,
		mark(listOK), listOK,
		leaderBridge, joinerBridge,
		ready.BlackboardDir, strings.Join(signals, ", "),
		map[bool]string{true: "PASS", false: "FAIL/PARTIAL"}[overall],
		ready.RunID,
	)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
