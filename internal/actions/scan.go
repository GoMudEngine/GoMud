package actions

import (
	"fmt"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ScanOptions parameterizes a one-step adjacent-room sweep.
type ScanOptions struct {
	// HostileOnly: when true, mob-actor btree wrappers may use this to
	// gate SoftTarget population on hostile-ness. The action itself does
	// not filter Sightings by hostility — that decision is up to the caller.
	HostileOnly bool
}

// ScanEntity describes one occupant in a sighted room.
type ScanEntity struct {
	Id    int
	Name  string
	IsMob bool
}

// ScanSighting describes one adjacent room's occupants.
type ScanSighting struct {
	ExitName  string
	RoomId    int
	RoomTitle string
	Mobs      []ScanEntity
	Players   []ScanEntity
}

// ScanResult is the structured outcome of a Scan call.
type ScanResult struct {
	Sightings []ScanSighting
}

// Scan walks each visible (non-secret) exit from the actor's room, loads
// the adjacent room, and lists non-hidden mobs and players in each.
// No skill check or cooldown is applied. UserActor receives a rendered
// text list via SendText; MobActor.SendText is a no-op (silent). The
// structured result is returned in both cases.
func Scan(actor Actor, opts ScanOptions) ScanResult {
	result := ScanResult{Sightings: []ScanSighting{}}

	room := actor.GetRoom()
	if room == nil {
		return result
	}

	for exitName, exitInfo := range room.Exits {
		if exitInfo.Secret {
			continue
		}

		adjRoom := rooms.LoadRoom(exitInfo.RoomId)
		if adjRoom == nil {
			continue
		}

		sighting := ScanSighting{
			ExitName:  exitName,
			RoomId:    adjRoom.RoomId,
			RoomTitle: adjRoom.Title,
		}

		for _, mobInstId := range adjRoom.GetMobs(rooms.FindAll) {
			m := mobs.GetInstance(mobInstId)
			if m == nil || m.Character.IsHidden() {
				continue
			}
			sighting.Mobs = append(sighting.Mobs, ScanEntity{
				Id:    mobInstId,
				Name:  m.Character.Name,
				IsMob: true,
			})
		}

		for _, pId := range adjRoom.GetPlayers(rooms.FindAll) {
			if pId == actor.GetUserId() {
				continue
			}
			u := users.GetByUserId(pId)
			if u == nil {
				continue
			}
			sighting.Players = append(sighting.Players, ScanEntity{
				Id:    pId,
				Name:  u.Character.Name,
				IsMob: false,
			})
		}

		result.Sightings = append(result.Sightings, sighting)
	}

	// UserActor text rendering. MobActor.SendText is a no-op.
	if actor.IsPlayer() {
		actor.SendText(messaging.CategorySystem, `You scan the surrounding area...`)
		actor.SendText(messaging.CategorySystem, ``)
		if len(result.Sightings) == 0 {
			actor.SendText(messaging.CategorySystem,
				`  There are no visible exits to scan.`)
		}
		for _, s := range result.Sightings {
			parts := []string{}
			for _, m := range s.Mobs {
				parts = append(parts,
					fmt.Sprintf(`<ansi fg="mobname">%s</ansi>`, m.Name))
			}
			for _, p := range s.Players {
				parts = append(parts,
					fmt.Sprintf(`<ansi fg="username">%s</ansi>`, p.Name))
			}
			dirLabel := fmt.Sprintf(`<ansi fg="exit">%s</ansi>`, s.ExitName)
			titleLabel := fmt.Sprintf(`<ansi fg="room-title">%s</ansi>`,
				s.RoomTitle)
			if len(parts) > 0 {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  %s (%s): %s`,
						dirLabel, titleLabel, strings.Join(parts, `, `)))
			} else {
				actor.SendText(messaging.CategorySystem,
					fmt.Sprintf(`  %s (%s): nothing of interest`,
						dirLabel, titleLabel))
			}
		}
		actor.SendText(messaging.CategorySystem, ``)
	}

	return result
}
