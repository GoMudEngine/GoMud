// Command playtestenv is the agent-facing CLI for the local ephemeral
// Docker playtest supervisor.
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/GoMudEngine/GoMud/internal/playtestenv"
)

const (
	exitOK       = 0
	exitFail     = 1
	exitUsage    = 2
	defaultLease = 2 * time.Hour
	defaultReady = 90 * time.Second
)

// envSupervisor is the narrow CLI-facing surface of playtestenv.Supervisor.
type envSupervisor interface {
	Start(context.Context, playtestenv.StartOptions) (playtestenv.Result, error)
	Status(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
	Logs(context.Context, playtestenv.LogsOptions) (playtestenv.Result, error)
	Renew(context.Context, playtestenv.RenewOptions) (playtestenv.Result, error)
	Stop(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
	Reap(context.Context, string) ([]playtestenv.Result, error)
}

// reapJSON is the single JSON object emitted for `reap --json`.
// Documented shape: {"operation":"reap","results":[...Result]}.
type reapJSON struct {
	Operation string               `json:"operation"`
	Results   []playtestenv.Result `json:"results"`
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, playtestenv.New()))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, supervisor envSupervisor) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: playtestenv <start|status|logs|renew|stop|reap> [flags]")
		return exitUsage
	}
	if err := rejectForbiddenFlags(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}

	cmd := args[0]
	rest := args[1:]
	switch cmd {
	case "start":
		return cmdStart(ctx, rest, stdout, stderr, supervisor)
	case "status":
		return cmdStatus(ctx, rest, stdout, stderr, supervisor)
	case "logs":
		return cmdLogs(ctx, rest, stdout, stderr, supervisor)
	case "renew":
		return cmdRenew(ctx, rest, stdout, stderr, supervisor)
	case "stop":
		return cmdStop(ctx, rest, stdout, stderr, supervisor)
	case "reap":
		return cmdReap(ctx, rest, stdout, stderr, supervisor)
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", cmd)
		return exitUsage
	}
}

func rejectForbiddenFlags(args []string) error {
	forbidden := []string{
		"--host",
		"--target",
		"--context",
		"--compose-file",
		"--source-mount",
		"--export",
	}
	for _, a := range args {
		name := a
		if i := strings.IndexByte(a, '='); i >= 0 {
			name = a[:i]
		}
		for _, f := range forbidden {
			if name == f {
				return fmt.Errorf("flag %s is not supported", f)
			}
		}
	}
	return nil
}

func newFlagSet(name string) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	return fs
}

func cmdStart(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("start")
	checkout := fs.String("checkout", "", "checkout path (default: cwd)")
	lease := fs.Duration("lease", defaultLease, "run lease duration")
	asJSON := fs.Bool("json", false, "emit one JSON result object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if fs.NArg() != 0 {
		fmt.Fprintf(stderr, "unexpected argument %q\n", fs.Arg(0))
		return exitUsage
	}

	path := *checkout
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return exitFail
		}
		path = cwd
	}

	res, err := s.Start(ctx, playtestenv.StartOptions{
		Checkout:         path,
		Lease:            *lease,
		ReadinessTimeout: defaultReady,
	})
	return emitResult(stdout, stderr, *asJSON, res, err)
}

func cmdStatus(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("status")
	checkout := fs.String("checkout", "", "checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	asJSON := fs.Bool("json", false, "emit one JSON result object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if *checkout == "" || *runID == "" {
		fmt.Fprintln(stderr, "status requires --checkout PATH and --run ID")
		return exitUsage
	}
	res, err := s.Status(ctx, playtestenv.RunOptions{Checkout: *checkout, RunID: *runID})
	return emitResult(stdout, stderr, *asJSON, res, err)
}

func cmdLogs(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("logs")
	checkout := fs.String("checkout", "", "checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	follow := fs.Bool("follow", false, "stream live container logs")
	asJSON := fs.Bool("json", false, "emit one JSON result object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if *checkout == "" || *runID == "" {
		fmt.Fprintln(stderr, "logs requires --checkout PATH and --run ID")
		return exitUsage
	}
	if *follow && *asJSON {
		fmt.Fprintln(stderr, "--follow and --json are mutually exclusive")
		return exitUsage
	}
	opts := playtestenv.LogsOptions{
		Checkout: *checkout,
		RunID:    *runID,
		Follow:   *follow,
	}
	if *follow {
		opts.Output = stdout
	}
	res, err := s.Logs(ctx, opts)
	return emitResult(stdout, stderr, *asJSON, res, err)
}

func cmdRenew(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("renew")
	checkout := fs.String("checkout", "", "checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	lease := fs.Duration("lease", 0, "new lease duration (required)")
	asJSON := fs.Bool("json", false, "emit one JSON result object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if *checkout == "" || *runID == "" || *lease <= 0 {
		fmt.Fprintln(stderr, "renew requires --checkout PATH, --run ID, and --lease DURATION")
		return exitUsage
	}
	res, err := s.Renew(ctx, playtestenv.RenewOptions{
		Checkout: *checkout,
		RunID:    *runID,
		Lease:    *lease,
	})
	return emitResult(stdout, stderr, *asJSON, res, err)
}

func cmdStop(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("stop")
	checkout := fs.String("checkout", "", "checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	asJSON := fs.Bool("json", false, "emit one JSON result object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	if *checkout == "" || *runID == "" {
		fmt.Fprintln(stderr, "stop requires --checkout PATH and --run ID")
		return exitUsage
	}
	res, err := s.Stop(ctx, playtestenv.RunOptions{Checkout: *checkout, RunID: *runID})
	return emitResult(stdout, stderr, *asJSON, res, err)
}

func cmdReap(ctx context.Context, args []string, stdout, stderr io.Writer, s envSupervisor) int {
	fs := newFlagSet("reap")
	checkout := fs.String("checkout", "", "checkout path (default: cwd)")
	asJSON := fs.Bool("json", false, "emit one JSON wrapper object on stdout")
	if err := fs.Parse(args); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitUsage
	}
	path := *checkout
	if path == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
			return exitFail
		}
		path = cwd
	}
	results, err := s.Reap(ctx, path)
	return emitReap(stdout, stderr, *asJSON, results, err)
}

func emitResult(stdout, stderr io.Writer, asJSON bool, res playtestenv.Result, err error) int {
	if asJSON {
		if encErr := json.NewEncoder(stdout).Encode(res); encErr != nil {
			fmt.Fprintln(stderr, encErr.Error())
			return exitFail
		}
	} else {
		writeHumanResult(stdout, res)
		if err != nil || res.Failure != nil {
			writeFailureDiagnostics(stderr, res, err)
		}
	}
	if err != nil {
		return exitFail
	}
	return exitOK
}

func emitReap(stdout, stderr io.Writer, asJSON bool, results []playtestenv.Result, err error) int {
	if asJSON {
		wrap := reapJSON{Operation: "reap", Results: results}
		if wrap.Results == nil {
			wrap.Results = []playtestenv.Result{}
		}
		if encErr := json.NewEncoder(stdout).Encode(wrap); encErr != nil {
			fmt.Fprintln(stderr, encErr.Error())
			return exitFail
		}
	} else {
		if len(results) == 0 {
			fmt.Fprintln(stdout, "reap: no candidates")
		}
		for _, res := range results {
			writeHumanResult(stdout, res)
			if res.Failure != nil {
				writeFailureDiagnostics(stderr, res, nil)
			}
		}
		if err != nil {
			fmt.Fprintln(stderr, err.Error())
		}
	}
	if err != nil {
		return exitFail
	}
	return exitOK
}

func writeHumanResult(w io.Writer, res playtestenv.Result) {
	parts := []string{res.Operation}
	if res.RunID != "" {
		parts = append(parts, "run="+res.RunID)
	}
	if res.State != "" {
		parts = append(parts, "state="+string(res.State))
	}
	if res.Endpoint != nil {
		parts = append(parts, fmt.Sprintf("endpoint=%s:%d", res.Endpoint.Host, res.Endpoint.Port))
	}
	if res.ServerLog != "" {
		parts = append(parts, "server_log="+res.ServerLog)
	}
	if res.Cleanup != nil {
		if res.Cleanup.Complete {
			parts = append(parts, "cleanup=complete")
		} else {
			parts = append(parts, "cleanup=incomplete")
		}
	}
	if res.Failure != nil {
		parts = append(parts, "failure="+string(res.Failure.Category))
	}
	fmt.Fprintln(w, strings.Join(parts, " "))
}

func writeFailureDiagnostics(w io.Writer, res playtestenv.Result, err error) {
	if res.Failure != nil {
		fmt.Fprintf(w, "category=%s", res.Failure.Category)
		if res.Failure.Retryable {
			fmt.Fprint(w, " retryable=true")
		}
		if res.Failure.Summary != "" {
			fmt.Fprintf(w, " summary=%s", res.Failure.Summary)
		}
		fmt.Fprintln(w)
	} else if err != nil {
		fmt.Fprintln(w, err.Error())
	}
	if res.RunID != "" {
		fmt.Fprintf(w, "run_id=%s\n", res.RunID)
	}
	if res.Manifest != "" {
		fmt.Fprintf(w, "manifest=%s\n", res.Manifest)
	}
	if res.ServerLog != "" {
		fmt.Fprintf(w, "server_log=%s\n", res.ServerLog)
	}
	if res.Report != "" {
		fmt.Fprintf(w, "report=%s\n", res.Report)
	}
	if res.Artifacts != nil {
		if res.Artifacts.Manifest != "" && res.Manifest == "" {
			fmt.Fprintf(w, "manifest=%s\n", res.Artifacts.Manifest)
		}
		if res.Artifacts.ServerLog != "" && res.ServerLog == "" {
			fmt.Fprintf(w, "server_log=%s\n", res.Artifacts.ServerLog)
		}
		if res.Artifacts.Report != "" && res.Report == "" {
			fmt.Fprintf(w, "report=%s\n", res.Artifacts.Report)
		}
	}
	if res.Cleanup != nil {
		for _, left := range res.Cleanup.Leftovers {
			fmt.Fprintf(w, "leftover %s %s\n", left.Kind, left.ID)
		}
		if res.Cleanup.Summary != "" {
			fmt.Fprintf(w, "cleanup=%s\n", res.Cleanup.Summary)
		}
	}
}
