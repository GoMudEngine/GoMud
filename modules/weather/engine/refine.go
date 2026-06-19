package engine

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/mutators"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

// Per-room refinement (M4, spec §2): when PerRoomRefinement is on, weather
// mutators live on individual room lists instead of the zone list — outdoor
// rooms carry weather-<type>, indoor rooms carry weather-<type>-indoor. The
// list (zone vs room) is the isolation boundary; the same prefix-scoped
// reconcile core does the work. Stale persisted room mutators heal lazily:
// the engine runs the mutator lifecycle on room load (Prepare) and each
// RoundTick, so decayrate retires strays without any world-wide pass.

// roomWantId maps a zone's weather + a room's indoor flag to the room-level
// mutator id ("" = calm, no mutator).
func roomWantId(w sim.WeatherType, indoor bool) string {
	base := MutatorIdFor(w)
	if base == "" {
		return ""
	}
	if indoor {
		return base + "-indoor"
	}
	return base
}

// RefineRoom reconciles one live room to the current weather map. Unknown
// rooms are skipped; unknown variant specs warn once. Game loop only.
func RefineRoom(roomId int, weather map[sim.ZoneId]sim.WeatherType) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	w := weather[room.Zone] // zero value = unset -> want ""
	biomeId := ""
	if b := room.GetBiome(); b != nil {
		biomeId = b.BiomeId
	}
	indoor := !isOutdoorBiome(biomeId)
	want := roomWantId(w, indoor)
	current := weatherIds(&room.Mutators)
	if len(current) == 0 && want == "" {
		return
	}
	if !reconcileList(&room.Mutators, current, want) {
		warnUnknownMutatorId(want) // variant spec missing
	}
}

// RefineOccupiedRooms refines every room that currently holds players.
func RefineOccupiedRooms(weather map[sim.ZoneId]sim.WeatherType) {
	for _, roomId := range rooms.GetRoomsWithPlayers() {
		RefineRoom(roomId, weather)
	}
}

// ZoneRoomIds returns the room ids registered to one zone (thin wrapper over
// the engine's zone registry; rooms need not be loaded).
func ZoneRoomIds(zone string) []int {
	return rooms.GetAllZoneRoomsIds(zone)
}

// RoomHasPlayers reports whether a room currently holds players. Uses the
// room cache; a room a player just left is necessarily still in memory, so
// this never force-loads in practice.
func RoomHasPlayers(roomId int) bool {
	room := rooms.LoadRoom(roomId)
	return room != nil && room.PlayerCt() > 0
}

// OccupiedRoomCount counts the rooms currently holding players (snapshot
// stat; cheap — the manager keeps this set incrementally).
func OccupiedRoomCount() int {
	return len(rooms.GetRoomsWithPlayers())
}

// StripOccupiedRoomWeather strips weather-* room mutators from every room
// that currently holds players — the transition OUT of a room-scoped mode.
// Unoccupied rooms refined earlier (mode "all") keep their mutators until the
// specs' decayrate safety net retires them.
func StripOccupiedRoomWeather() {
	for _, roomId := range rooms.GetRoomsWithPlayers() {
		StripRoomWeather(roomId)
	}
}

// StripRoomWeather removes all weather-* mutators from one live room.
func StripRoomWeather(roomId int) {
	room := rooms.LoadRoom(roomId)
	if room == nil {
		return
	}
	if current := weatherIds(&room.Mutators); len(current) > 0 {
		reconcileList(&room.Mutators, current, "")
	}
}

// StripZoneWeather removes the zone-level weather mutators from every zone —
// the transition into room-scoped mode. Seasons stay zone-wide, untouched by
// construction: this gathers the weather- prefix only.
func StripZoneWeather(g *sim.Graph) {
	for _, zone := range g.Zones() {
		zc := rooms.GetZoneConfig(zone)
		if zc == nil {
			continue
		}
		if current := weatherIds(&zc.Mutators); len(current) > 0 {
			reconcileList(&zc.Mutators, current, "")
		}
	}
}

// weatherIds gathers the live weather-* mutator ids from one mutator list
// (room or zone — both are mutators.MutatorList).
func weatherIds(ml *mutators.MutatorList) []string {
	var ids []string
	for _, mut := range ml.GetActive() {
		if strings.HasPrefix(mut.MutatorId, WeatherMutatorPrefix) {
			ids = append(ids, mut.MutatorId)
		}
	}
	return ids
}
