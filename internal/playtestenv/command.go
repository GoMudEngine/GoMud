package playtestenv

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os/exec"
	"strings"
	"time"
)

// execRunnerWaitDelay bounds how long Run waits, once the child process
// itself has exited or been killed via context cancellation, for its
// stdout/stderr copy goroutines to finish. Without a WaitDelay, a
// grandchild process that inherited the child's stdout/stderr handles -
// and is still running after the immediate child exits - can keep those
// pipes open indefinitely: exec.Cmd.Wait would then block until every
// holder of the handle closes it, even though the process this package is
// actually tracking has already terminated. Once the delay elapses, the
// standard library force-closes the pipes, the copy goroutines return, and
// exec.ErrWaitDelay is folded into the returned error.
const execRunnerWaitDelay = 3 * time.Second

// CommandSpec fully describes one subprocess invocation. Values are always
// argument slices, never a shell command line, so no argument is ever
// re-parsed or word-split by a shell.
type CommandSpec struct {
	Name   string
	Args   []string
	Dir    string
	Env    []string
	Stdout io.Writer
	Stderr io.Writer
}

// Runner executes one CommandSpec and reports its outcome. Production code
// uses execRunner; tests inject a fake to assert exact commands without
// invoking any real subprocess.
type Runner interface {
	Run(context.Context, CommandSpec) error
}

// ExitError reports a subprocess that started and ran to completion with a
// nonzero exit status. It wraps the underlying *exec.ExitError so callers
// may use errors.As with either type, without every caller needing to
// import os/exec itself.
type ExitError struct {
	Name     string
	Args     []string
	ExitCode int

	err error
}

func (e *ExitError) Error() string {
	return fmt.Sprintf("playtestenv: command %q exited with status %d", strings.Join(append([]string{e.Name}, e.Args...), " "), e.ExitCode)
}

// Unwrap exposes the underlying *exec.ExitError so errors.As/errors.Is
// continue to work against the standard library type.
func (e *ExitError) Unwrap() error { return e.err }

// execRunner is the production Runner. It uses exec.CommandContext directly
// - no Docker SDK and no shell.
type execRunner struct{}

// Run starts spec as a subprocess and waits for it to complete.
//
// Stdout and Stderr are assigned to cmd.Stdout/cmd.Stderr before Start, so
// exec.Cmd copies both streams concurrently on its own internal goroutines.
// This function deliberately never calls StdoutPipe/StderrPipe and drains
// them itself: draining two pipes serially deadlocks as soon as either
// stream produces more data than one OS pipe buffer, because the child
// blocks writing to the second, still-unread pipe while the parent is stuck
// finishing the first.
//
// Cancelling ctx terminates the child process using exec.CommandContext's
// default Cancel behavior (kill).
//
// If spec.Env is nil, the child inherits the current process's environment
// unchanged. If spec.Env is non-nil (even empty), it entirely replaces the
// child's environment; Run never reads or mutates the parent's real
// environment itself; that is the caller's responsibility.
func (execRunner) Run(ctx context.Context, spec CommandSpec) error {
	cmd := exec.CommandContext(ctx, spec.Name, spec.Args...)
	cmd.WaitDelay = execRunnerWaitDelay
	cmd.Dir = spec.Dir
	cmd.Env = spec.Env
	cmd.Stdout = spec.Stdout
	cmd.Stderr = spec.Stderr

	if err := cmd.Run(); err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			return &ExitError{
				Name:     spec.Name,
				Args:     append([]string(nil), spec.Args...),
				ExitCode: exitErr.ExitCode(),
				err:      exitErr,
			}
		}
		return err
	}
	return nil
}
