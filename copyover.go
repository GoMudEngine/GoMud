package main

import (
	"os"

	"github.com/GoMudEngine/GoMud/internal/caravan"
	"github.com/GoMudEngine/GoMud/internal/copyover"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/warehouse"
)

// triggerCopyover flushes all state to disk and re-execs the same binary,
// handing off active telnet sockets so players stay connected. It is a no-op on
// Windows (copyover.Execute returns an error there).
func triggerCopyover() error {
	binaryPath, err := os.Executable()
	if err != nil {
		return err
	}

	serverAlive.Store(false)

	_ = rooms.SaveAllRooms()
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
