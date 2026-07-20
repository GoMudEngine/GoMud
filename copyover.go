package main

import (
	"os"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

// triggerCopyoverLocked acquires the world lock, then performs a copyover.
//
// Use this from anywhere OUTSIDE MainWorker's tick loop — notably the SIGUSR1
// handler, which runs on its own goroutine. Without the lock, that handler
// serialised rooms, users and shops to disk while the tick loop was concurrently
// mutating them: at best a torn save that corrupts the copyover handoff, at
// worst a "concurrent map iteration and map write" fatal error during YAML
// marshal. The graceful-shutdown path already wraps the identical save sequence
// in this lock (world.go).
func triggerCopyoverLocked() error {
	util.LockMud()
	defer util.UnlockMud()

	return triggerCopyover()
}

// triggerCopyover flushes all state to disk and re-execs the same binary,
// handing off active telnet sockets so players stay connected. It is a no-op on
// Windows (copyover.Execute returns an error there).
//
// The caller MUST already hold the world lock. The admin `copyover` command
// satisfies this because commands execute inside MainWorker's locked EventLoop
// (world.go: LockMud / EventLoop / UnlockMud), and util's mutex is not
// reentrant — locking here would deadlock that path. Callers outside the tick
// loop must use triggerCopyoverLocked instead.
func triggerCopyover() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	serverAlive.Store(false)

	// Logged rather than discarded: this is the last chance to persist room
	// state before the process re-execs, so a failure here silently loses it.
	if err := rooms.SaveAllRooms(); err != nil {
		mudlog.Error("copyover", "action", "SaveAllRooms", "error", err)
	}
	users.SaveAllUsers()
	// Living-economy dirty-state persists only on graceful shutdown, so flush it
	// here too — otherwise a copyover silently rewinds it. Keeps the reboot seamless.
	shops.SaveAllShops()
	warehouse.SaveAll()
	forager.SaveAllThroughputs()
	caravan.SaveAllThroughputs()
	opinions.SaveAllOpinions()
	plugins.Save()

	return copyover.Execute(binaryPath, os.Args[1:])
}
