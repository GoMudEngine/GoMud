//go:build windows

package copyover

import (
	"errors"
	"os"
)

// execInPlace is unreachable on Windows — Execute rejects the platform before
// calling it. Present so the package compiles.
func execInPlace(binaryPath string, argv []string, stateFile *os.File) error {
	return errors.New("copyover is not supported on this platform")
}
