package caravan

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/shops"
)

// VisitVendorsInRoom calls Restock() on every shop-bearing mob in the
// given room and returns the list of mob names that received a delivery
// (for room-flavor message generation).
//
// "Shop-bearing" = HasShop() returns true AND the shops package can
// resolve a shop inventory for (zone, mobId, homeRoomId). A mob with a
// Shop spec but no registered inventory is silently skipped.
//
// Returns nil if the room doesn't exist.
func VisitVendorsInRoom(roomId int) []string {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return nil
	}
	var visited []string
	for _, instId := range room.GetMobs(rooms.FindAll) {
		mob := mobs.GetInstance(instId)
		if mob == nil || !mob.HasShop() {
			continue
		}
		si := shops.GetShopInventory(mob.Zone, int(mob.MobId), mob.HomeRoomId)
		if si == nil {
			continue
		}
		si.Restock()
		visited = append(visited, mob.Character.Name)
	}
	return visited
}

// FormatDeliveryMessage builds the room-visible flavor text for a
// caravan visit. Returns an empty string if no vendors were visited
// (caller should skip sending in that case).
func FormatDeliveryMessage(visited []string) string {
	if len(visited) == 0 {
		return ""
	}
	if len(visited) == 1 {
		return fmt.Sprintf(
			`<ansi fg="yellow">The caravan crew unloads a crate of supplies; <ansi fg="mobname">%s</ansi> nods their thanks.</ansi>`,
			visited[0])
	}
	// Two or more — list with commas.
	names := visited[0]
	for _, n := range visited[1:] {
		names += ", " + n
	}
	return fmt.Sprintf(
		`<ansi fg="yellow">The caravan crew unloads supplies for: %s.</ansi>`,
		names)
}
