//go:build windows

package copyover

import "syscall"

func newSysProcAttr() *syscall.SysProcAttr {
	return nil
}
