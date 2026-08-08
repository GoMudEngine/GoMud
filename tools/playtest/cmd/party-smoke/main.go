// Command party-smoke runs a serious 0.3d shared-env party combat smoke:
// distinct loadouts, party invite/accept, in-game party say, thug fight,
// leader shield-bash knockdown, joiner prone stomp.
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
	Profile   *string `json:"profile"`
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
		Profile  string `json:"profile"`
	} `json:"players"`
}

type evidence struct {
	PartyInvite   bool
	PartyAccept   bool
	PartyListOK   bool
	PartySaySeen  bool
	BashKnockdown bool
	StompHit      bool
	FightJoined   bool
	Snippets      []string
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
		"--wall-clock", "20m",
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

	ev := &evidence{}
	leaderChar := "Midroad Scout"
	joinerChar := "Early Strider"

	// Safe-room login (462). Do not enter alley until partied + plan said.
	// Pace under AICommandsPerRound=3 (see agent.send gap).
	time.Sleep(14 * time.Second)
	_ = lAgent.send("look")
	_ = jAgent.send("look")
	_ = lAgent.send("who")
	_ = jAgent.send("who")
	_ = lAgent.send("equip")
	// Blunt auto-DPS so the thug survives for a clean kick→stomp window.
	_ = lAgent.send("remove practice")
	_ = jAgent.send("remove practice")
	time.Sleep(5 * time.Second)

	if n := waitCapture(joiner.BridgeDir, regexp.MustCompile(`Early Strider`), 20*time.Second); n != "" {
		joinerChar = n
	}
	if n := waitCapture(leader.BridgeDir, regexp.MustCompile(`Midroad Scout`), 5*time.Second); n != "" {
		leaderChar = n
	}

	_ = writeBB(ready.BlackboardDir, "joiner-ready", "joiner", map[string]any{
		"username": jUser, "character_name": joinerChar, "personality": "bug-finder",
	})
	_ = lAgent.send("party invite " + joinerChar)
	ev.PartyInvite = waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)You invited`), 60*time.Second)
	_ = writeBB(ready.BlackboardDir, "leader-invited", "leader", map[string]any{
		"target": joinerChar, "invited": ev.PartyInvite, "personality": "feature-tester",
	})

	_ = jAgent.send("party accept")
	ev.PartyAccept = waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)joined the party`), 60*time.Second)
	_ = writeBB(ready.BlackboardDir, "joiner-accepted", "joiner", map[string]any{"joined": ev.PartyAccept})

	_ = lAgent.send("party list")
	_ = jAgent.send("party list")
	time.Sleep(4 * time.Second)
	// Party list / GMCP Party both prove two-member party.
	ev.PartyListOK = waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)Early Strider|Party Members|In Party`), 20*time.Second) &&
		waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)Midroad Scout|Party Members|In Party`), 5*time.Second)

	planMsg := "Plan: you step east off Main Street so you don't auto-follow. I south/west alone, bash prone, party-say. Then you west, south, west, kick."
	_ = lAgent.send("party say " + planMsg)
	time.Sleep(4 * time.Second)
	ev.PartySaySeen = waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)step east|bash prone|west, south, west, kick`), 30*time.Second)

	// Party auto-follow copies the leader's exit to same-room members. Joiner
	// must leave 462 before the leader moves, or they get dragged into 474.
	_ = jAgent.send("east")
	if !waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)Main Street|Exits:`), 15*time.Second) {
		ev.add("WARN: joiner east peel unclear")
	}
	time.Sleep(2 * time.Second)

	_ = lAgent.send("south")
	time.Sleep(3 * time.Second)
	_ = lAgent.send("west")
	time.Sleep(3 * time.Second)
	_ = lAgent.send("look")
	if !waitMatch(leader.BridgeDir, regexp.MustCompile(`(?i)Thornwall Thug|Back Alley`), 15*time.Second) {
		ev.add("WARN: leader did not see thug in alley")
	}
	if waitMatch(joiner.BridgeDir, regexp.MustCompile(`(?i)Back Alley|Thornwall Thug`), 1*time.Second) {
		ev.add("WARN: joiner entered alley early (unexpected auto-follow)")
	}

	lBase := fileSize(filepath.Join(leader.BridgeDir, "events.jsonl"))
	jBase := fileSize(filepath.Join(joiner.BridgeDir, "events.jsonl"))

	// Bash/trip knockdowns plus combat sweep lines ("crash" / "crashing").
	knockMsg := regexp.MustCompile(`(?i)(?:knocks .+ to the ground|crash(?:ing)? to the ground)`)
	droppedRe := regexp.MustCompile(`(?i)Command dropped`)
	stompReSelf := regexp.MustCompile(`(?i)You (?:bring your heel|slam your foot into the downed|drive a vicious stomp|crush your boot|stamp hard on the prone|grind your heel)`)
	stompReRoom := regexp.MustCompile(`(?i)(?:stomps on the downed|viciously stomps|drives their heel into|slams a boot into|crushes .+ underfoot|grinds their heel into)`)
	stompMiss := regexp.MustCompile(`(?i)(?:try to stomp|stomp misses|slam your foot down but)`)
	cooldownMsg := regexp.MustCompile(`(?i)need a moment to recover before attempting another special move`)

	// SpecialMoveCooldown=4 rounds (~16s). Wait a full CD between bash/trip tries.
	for i := 0; i < 8 && !ev.BashKnockdown; i++ {
		attemptBase := fileSize(filepath.Join(leader.BridgeDir, "events.jsonl"))
		if i%3 == 2 {
			_ = lAgent.send("trip thug")
		} else {
			_ = lAgent.send("bash thug")
		}
		if i == 0 {
			ev.FightJoined = waitMatchFrom(leader.BridgeDir, attemptBase, regexp.MustCompile(`(?i)shield bash|trip|Bash whom|Thornwall Thug`), 15*time.Second)
		}
		if waitMatchFrom(leader.BridgeDir, attemptBase, knockMsg, 12*time.Second) {
			ev.BashKnockdown = true
			break
		}
		if waitMatchFrom(leader.BridgeDir, attemptBase, cooldownMsg, 1*time.Second) {
			ev.add("NOTE: special-move cooldown; waiting")
		}
		time.Sleep(18 * time.Second)
	}

	if ev.BashKnockdown {
		_ = lAgent.send("party say They're down — west south west kick!")
		// Joiner path from east-of-462 (463): west→462, south→469, west→474.
		_ = jAgent.send("west")
		_ = jAgent.send("south")
		_ = jAgent.send("west")
		kickBase := fileSize(filepath.Join(joiner.BridgeDir, "events.jsonl"))
		_ = jAgent.send("kick thug")

		deadline := time.Now().Add(16 * time.Second)
		for time.Now().Before(deadline) && !ev.StompHit {
			if waitMatchFrom(joiner.BridgeDir, kickBase, stompReSelf, 1200*time.Millisecond) ||
				waitMatchFrom(leader.BridgeDir, lBase, stompReRoom, 400*time.Millisecond) {
				ev.StompHit = true
				break
			}
			if waitMatchFrom(joiner.BridgeDir, kickBase, droppedRe, 200*time.Millisecond) {
				ev.add("WARN: kick dropped by AI rate limit")
				time.Sleep(5 * time.Second)
				kickBase = fileSize(filepath.Join(joiner.BridgeDir, "events.jsonl"))
				_ = jAgent.send("kick thug")
				continue
			}
			if waitMatchFrom(joiner.BridgeDir, kickBase, stompMiss, 200*time.Millisecond) {
				ev.add("NOTE: kick→stomp missed; re-prone and retry")
				break
			}
			if waitMatchFrom(joiner.BridgeDir, kickBase, regexp.MustCompile(`(?i)You don't see them here`), 200*time.Millisecond) {
				ev.add("FAIL: joiner missed thug after entering alley")
				break
			}
			if waitMatchFrom(joiner.BridgeDir, kickBase, regexp.MustCompile(`(?i)swing a kick|kick sails`), 200*time.Millisecond) {
				ev.add("NOTE: standing kick — thug already up")
				break
			}
		}
	}

	// One CD-aware retry if first stomp missed / standing.
	for i := 0; i < 3 && ev.BashKnockdown && !ev.StompHit; i++ {
		time.Sleep(18 * time.Second)
		attemptBase := fileSize(filepath.Join(leader.BridgeDir, "events.jsonl"))
		_ = lAgent.send("bash thug")
		if !waitMatchFrom(leader.BridgeDir, attemptBase, knockMsg, 12*time.Second) {
			_ = lAgent.send("trip thug")
			if !waitMatchFrom(leader.BridgeDir, attemptBase, knockMsg, 12*time.Second) {
				continue
			}
		}
		_ = lAgent.send("party say Prone again — kick!")
		kickBase := fileSize(filepath.Join(joiner.BridgeDir, "events.jsonl"))
		_ = jAgent.send("kick thug")
		if waitMatchFrom(joiner.BridgeDir, kickBase, stompReSelf, 12*time.Second) ||
			waitMatchFrom(leader.BridgeDir, lBase, stompReRoom, 3*time.Second) {
			ev.StompHit = true
			break
		}
		if waitMatchFrom(joiner.BridgeDir, kickBase, stompMiss, 1*time.Second) {
			ev.add("NOTE: stomp miss on retry")
		}
	}

	if ev.StompHit {
		_ = jAgent.send("party say Stomp landed.")
	}

	for i := 0; i < 2; i++ {
		_ = lAgent.send("attack thug")
		_ = jAgent.send("attack thug")
		time.Sleep(4 * time.Second)
	}

	ev.Snippets = collectSnippetsFrom([]bridgeSlice{
		{leader.BridgeDir, lBase},
		{joiner.BridgeDir, jBase},
	}, []*regexp.Regexp{
		regexp.MustCompile(`(?i)You invited.+party`),
		regexp.MustCompile(`(?i)joined the party`),
		regexp.MustCompile(`(?i)\(party\).+(?:step east|They're down|west south west kick|Prone again|Stomp landed)`),
		regexp.MustCompile(`(?i)knocks .+ to the ground`),
		regexp.MustCompile(`(?i)crashing to the ground`),
		regexp.MustCompile(`(?i)You (?:bring your heel|slam your foot into the downed|drive a vicious stomp|stamp hard on the prone)`),
		regexp.MustCompile(`(?i)stomps on the downed`),
	})
	// Also keep party-form snippets from full logs.
	ev.Snippets = append(ev.Snippets, collectSnippets([]string{leader.BridgeDir, joiner.BridgeDir}, []*regexp.Regexp{
		regexp.MustCompile(`(?i)You invited.+party`),
		regexp.MustCompile(`(?i)You joined the party`),
		regexp.MustCompile(`(?i)step east|bash prone`),
	})...)

	lAgent.close()
	jAgent.close()

	stop := exec.Command(bin, "stop", "--checkout", checkout, "--run", ready.RunID)
	_ = stop.Run()
	_ = cmd.Wait()

	overall := ev.PartyInvite && ev.PartyAccept && ev.PartySaySeen && ev.BashKnockdown && ev.StompHit
	reportPath := filepath.Join(checkout, "tools", "playtest", "reports", "2026-08-08-party-combat-coordination-smoke.md")
	if err := writeReport(reportPath, ready, leader, joiner, leaderChar, joinerChar, lUser, jUser, ev, overall); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "report=%s overall=%v invite=%v accept=%v say=%v bash=%v stomp=%v\n",
		reportPath, overall, ev.PartyInvite, ev.PartyAccept, ev.PartySaySeen, ev.BashKnockdown, ev.StompHit)
	if !overall {
		return fmt.Errorf("smoke incomplete: invite=%v accept=%v say=%v bash/prone=%v stomp=%v",
			ev.PartyInvite, ev.PartyAccept, ev.PartySaySeen, ev.BashKnockdown, ev.StompHit)
	}
	return nil
}

func (e *evidence) add(s string) { e.Snippets = append(e.Snippets, s) }

func readReady(r io.Reader, timeout time.Duration) (readyPayload, error) {
	type result struct {
		p   readyPayload
		err error
	}
	ch := make(chan result, 1)
	go func() {
		sc := bufio.NewScanner(r)
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
	cmd      *exec.Cmd
	stdin    io.WriteCloser
	bridge   string
	lastSend time.Time
}

// AIConnections enforce AICommandsPerRound (shipped: 3/round, RoundSeconds=4).
// Pace sends so kicks are not dropped with "Command dropped — AI rate limit".
const minCommandGap = 1500 * time.Millisecond

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
	_ = outF.Close()
	_ = errF.Close()
	return &agent{cmd: cmd, stdin: stdin, bridge: bridge}, nil
}

func (a *agent) send(line string) error {
	if a == nil || a.stdin == nil {
		return fmt.Errorf("agent closed")
	}
	if !a.lastSend.IsZero() {
		if wait := minCommandGap - time.Since(a.lastSend); wait > 0 {
			time.Sleep(wait)
		}
	}
	if f, err := os.OpenFile(filepath.Join(a.bridge, "commands.txt"), os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0o644); err == nil {
		_, _ = f.WriteString(line + "\n")
		_ = f.Close()
	}
	_, err := io.WriteString(a.stdin, line+"\n")
	a.lastSend = time.Now()
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
	return waitCaptureFrom(bridge, 0, re, timeout)
}

func fileSize(path string) int64 {
	st, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return st.Size()
}

func waitMatchFrom(bridge string, from int64, re *regexp.Regexp, timeout time.Duration) bool {
	return waitCaptureFrom(bridge, from, re, timeout) != ""
}

func waitCaptureFrom(bridge string, from int64, re *regexp.Regexp, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	path := filepath.Join(bridge, "events.jsonl")
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(path)
		if err == nil {
			chunk := raw
			if from > 0 && int64(len(raw)) > from {
				chunk = raw[from:]
			} else if from > 0 && int64(len(raw)) <= from {
				chunk = nil
			}
			if m := re.Find(chunk); m != nil {
				return string(m)
			}
		}
		time.Sleep(500 * time.Millisecond)
	}
	return ""
}

type bridgeSlice struct {
	bridge string
	from   int64
}

func collectSnippets(bridges []string, res []*regexp.Regexp) []string {
	var slices []bridgeSlice
	for _, b := range bridges {
		slices = append(slices, bridgeSlice{bridge: b, from: 0})
	}
	return collectSnippetsFrom(slices, res)
}

func collectSnippetsFrom(bridges []bridgeSlice, res []*regexp.Regexp) []string {
	var out []string
	seen := map[string]bool{}
	for _, b := range bridges {
		raw, err := os.ReadFile(filepath.Join(b.bridge, "events.jsonl"))
		if err != nil {
			continue
		}
		if b.from > 0 && int64(len(raw)) > b.from {
			raw = raw[b.from:]
		} else if b.from > 0 {
			continue
		}
		text := stripANSI(string(raw))
		for _, re := range res {
			for _, m := range re.FindAllString(text, 3) {
				m = strings.TrimSpace(regexp.MustCompile(`\s+`).ReplaceAllString(m, " "))
				if len(m) > 180 {
					m = m[:180] + "…"
				}
				if m != "" && !seen[m] {
					seen[m] = true
					out = append(out, m)
				}
			}
		}
	}
	return out
}

func stripANSI(s string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*m`)
	return re.ReplaceAllString(s, "")
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

func writeReport(path string, ready readyPayload, leader, joiner *readyActor, leaderChar, joinerChar, lUser, jUser string, ev *evidence, overall bool) error {
	mark := func(ok bool) string {
		if ok {
			return "[x]"
		}
		return "[ ]"
	}
	lProf, jProf := "mid", "early"
	if leader.Profile != nil {
		lProf = *leader.Profile
	}
	if joiner.Profile != nil {
		jProf = *joiner.Profile
	}
	snippets := strings.Join(ev.Snippets, "\n- ")
	if snippets != "" {
		snippets = "- " + snippets
	} else {
		snippets = "- (no regex snippets captured — see bridge events.jsonl)"
	}
	body := fmt.Sprintf(`# Multi-Agent Playtest Report: party-formation (combat coordination)

**Date:** 2026-08-08
**Scenario:** party-formation (mode: party)
**Checkout:** %s (commit: %s, dirty: %v)
**Run ID:** %s
**On actor stop:** %s
**Wall-clock:** 20m (hard cut)
**Endpoint:** %s:%d (ephemeral; passwords omitted)

## Agents / loadouts
| Actor | Personality | Profile | Character | Login | Bridge |
|-------|-------------|---------|-----------|-------|--------|
| leader | feature-tester | %s | %s | (path only) | %s |
| joiner | bug-finder | %s | %s | (path only) | %s |

Leader: feature-tester / mid — wooden shield (20004), practice sword removed before
fight, weapon-combat 6 (bash-focused, blunt DPS).
Joiner: bug-finder / early — unarmed-combat 25, practice sword removed. Stages east
of 462 so party auto-follow does not drag them into alley aggro; on callout paths
in and kicks (kick auto-selects stomp when target is prone).

## Summary
Shared ephemeral env. Implementer-driven mudagents executed the authored
personality/goals contracts (not LLM loops): party by character name, party-channel
comms, leader shield-bash knockdown, joiner kick→stomp on Thornwall Thug.

## Group Goal Results
- %s party-formed — invite=%v accept=%v list=%v
- %s party-comms — joiner saw leader party say=%v
- %s coordinated-prone-stomp — bash/prone=%v stomp=%v fight_engaged=%v

## Evidence snippets
%s

## Blackboard
- Dir: %s
- Signals: joiner-ready, leader-invited, joiner-accepted

## Findings
### OBSERVATION: driver mode
Mudagents were driven by tools/playtest/cmd/party-smoke following the goals
files under goals/scenarios/party-formation/. Personalities are those assigned
in the scenario roster (feature-tester / bug-finder).
### NOTE: kick vs stomp
There is no separate stomp verb for this test — agents issue kick; the engine
selects the stomp variant when the target is prone/supine.

## Stats
- Overall: %s
- Sidecar: tools/playtest/.run/%s/session.json
`,
		ready.Checkout, ready.Commit, ready.Dirty,
		ready.RunID, ready.OnActorStop,
		ready.Endpoint.Host, ready.Endpoint.Port,
		lProf, leaderChar, leader.BridgeDir,
		jProf, joinerChar, joiner.BridgeDir,
		mark(ev.PartyInvite && ev.PartyAccept), ev.PartyInvite, ev.PartyAccept, ev.PartyListOK,
		mark(ev.PartySaySeen), ev.PartySaySeen,
		mark(ev.BashKnockdown && ev.StompHit), ev.BashKnockdown, ev.StompHit, ev.FightJoined,
		snippets,
		ready.BlackboardDir,
		map[bool]string{true: "PASS", false: "FAIL/PARTIAL"}[overall],
		ready.RunID,
	)
	_ = lUser
	_ = jUser
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(body), 0o644)
}
