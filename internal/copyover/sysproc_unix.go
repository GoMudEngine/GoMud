//go:build !windows

package copyover

import (
	"fmt"
	"os"
	"syscall"
)

// execInPlace clears CLOEXEC on the state fd so it survives the exec, then
// replaces the current process image with binaryPath via execve. Same PID —
// under Docker/systemd the supervisor never sees the process exit.
// Never returns on success.
func execInPlace(binaryPath string, argv []string, stateFile *os.File) error {
	if _, _, errno := syscall.Syscall(syscall.SYS_FCNTL, stateFile.Fd(), syscall.F_SETFD, 0); errno != 0 {
		return fmt.Errorf("clear cloexec on state fd %d: %w", stateFile.Fd(), errno)
	}
	return syscall.Exec(binaryPath, argv, os.Environ())
}
