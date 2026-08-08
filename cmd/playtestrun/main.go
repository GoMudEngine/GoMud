// Command playtestrun composes playtestenv for single-agent and multi-agent
// ephemeral local playtests with a wall-clock watchdog and session sidecar.
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
	"github.com/GoMudEngine/GoMud/internal/playtestrun"
)

const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

type envSupervisor interface {
	Start(context.Context, playtestenv.StartOptions) (playtestenv.Result, error)
	Stop(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
	Status(context.Context, playtestenv.RunOptions) (playtestenv.Result, error)
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	os.Exit(run(ctx, os.Args[1:], os.Stdout, os.Stderr, playtestenv.New()))
}

func run(ctx context.Context, args []string, stdout, stderr io.Writer, env envSupervisor) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "usage: playtestrun <run|scenario|status|stop> [flags]")
		return exitUsage
	}
	switch args[0] {
	case "run":
		return cmdRun(ctx, args[1:], stdout, stderr, env)
	case "scenario":
		return cmdScenario(ctx, args[1:], stdout, stderr, env)
	case "status":
		return cmdStatus(args[1:], stdout, stderr)
	case "stop":
		return cmdStop(args[1:], stderr)
	default:
		fmt.Fprintf(stderr, "unknown subcommand %q\n", args[0])
		return exitUsage
	}
}

func cmdRun(ctx context.Context, args []string, stdout, stderr io.Writer, env envSupervisor) int {
	fs := flag.NewFlagSet("run", flag.ContinueOnError)
	fs.SetOutput(stderr)
	checkout := fs.String("checkout", "", "absolute git checkout path (required)")
	goals := fs.String("goals", "", "goals YAML path (required)")
	personality := fs.String("personality", "", "playtest personality name (required)")
	wallClock := fs.String("wall-clock", "", "optional wall-clock override (e.g. 30m)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*checkout) == "" || strings.TrimSpace(*goals) == "" || strings.TrimSpace(*personality) == "" {
		fmt.Fprintln(stderr, "usage: playtestrun run --checkout PATH --goals PATH --personality NAME [--wall-clock 30m]")
		return exitUsage
	}
	var override time.Duration
	if strings.TrimSpace(*wallClock) != "" {
		d, err := time.ParseDuration(*wallClock)
		if err != nil || d <= 0 {
			fmt.Fprintf(stderr, "invalid --wall-clock %q\n", *wallClock)
			return exitUsage
		}
		override = d
	}
	err := playtestrun.Run(ctx, playtestrun.RunParams{
		Checkout:          *checkout,
		GoalsPath:         *goals,
		Personality:       *personality,
		WallClockOverride: override,
		Env:               env,
		Stdout:            stdout,
		Stderr:            stderr,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitFail
	}
	return exitOK
}

func cmdScenario(ctx context.Context, args []string, stdout, stderr io.Writer, env envSupervisor) int {
	fs := flag.NewFlagSet("scenario", flag.ContinueOnError)
	fs.SetOutput(stderr)
	checkout := fs.String("checkout", "", "absolute git checkout path (required)")
	scenario := fs.String("scenario", "", "scenario YAML path (required)")
	wallClock := fs.String("wall-clock", "", "optional scenario wall-clock override (e.g. 45m)")
	force := fs.Bool("force", false, "bypass MaxAIConnections roster-size check only")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*checkout) == "" || strings.TrimSpace(*scenario) == "" {
		fmt.Fprintln(stderr, "usage: playtestrun scenario --checkout PATH --scenario PATH [--wall-clock 45m] [--force]")
		return exitUsage
	}
	var override time.Duration
	if strings.TrimSpace(*wallClock) != "" {
		d, err := time.ParseDuration(*wallClock)
		if err != nil || d <= 0 {
			fmt.Fprintf(stderr, "invalid --wall-clock %q\n", *wallClock)
			return exitUsage
		}
		override = d
	}
	err := playtestrun.RunScenario(ctx, playtestrun.ScenarioParams{
		Checkout:          *checkout,
		ScenarioPath:      *scenario,
		WallClockOverride: override,
		Force:             *force,
		Env:               env,
		Stdout:            stdout,
		Stderr:            stderr,
	})
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitFail
	}
	return exitOK
}

func cmdStatus(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("status", flag.ContinueOnError)
	fs.SetOutput(stderr)
	checkout := fs.String("checkout", "", "absolute git checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*checkout) == "" || strings.TrimSpace(*runID) == "" {
		fmt.Fprintln(stderr, "usage: playtestrun status --checkout PATH --run ID")
		return exitUsage
	}
	sc, err := playtestrun.ReadSidecar(*checkout, *runID)
	if err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitFail
	}
	enc := json.NewEncoder(stdout)
	enc.SetIndent("", "  ")
	if err := enc.Encode(sc); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitFail
	}
	return exitOK
}

func cmdStop(args []string, stderr io.Writer) int {
	fs := flag.NewFlagSet("stop", flag.ContinueOnError)
	fs.SetOutput(stderr)
	checkout := fs.String("checkout", "", "absolute git checkout path (required)")
	runID := fs.String("run", "", "run id (required)")
	if err := fs.Parse(args); err != nil {
		return exitUsage
	}
	if strings.TrimSpace(*checkout) == "" || strings.TrimSpace(*runID) == "" {
		fmt.Fprintln(stderr, "usage: playtestrun stop --checkout PATH --run ID")
		return exitUsage
	}
	if err := playtestrun.WriteStopSignal(*checkout, *runID); err != nil {
		fmt.Fprintln(stderr, err.Error())
		return exitFail
	}
	return exitOK
}
