// Package splash defines the reusable in-world splash event: one event, rendered
// per-client downstream (SVG on web, refined ASCII on terminal, caption for
// screen readers). Delivery lives in listeners (internal/hooks + modules/gmcp);
// this package only defines the event and resolves recipients.
package splash

import (
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

type SplashTarget uint8

const (
	TargetGlobal SplashTarget = iota // all active users
	TargetZone                       // all users currently in Zone
	TargetUser                       // just UserId
)

// Splash is emitted by any subsystem that wants a splash scene shown. SceneId
// selects the art (terminal template + client-side SVG); Caption is the
// screen-reader / fallback line; Data fills dynamic slots.
type Splash struct {
	SceneId string
	Caption string
	Target  SplashTarget
	Zone    string
	UserId  int
	// Data is passed straight to templates.Process and the GMCP payload. It is a
	// gametime.Date for celestial scenes, or a map[string]any (e.g. {"zone": …})
	// for season/weather scenes.
	Data any
}

func (Splash) Type() string { return "Splash" }

// Recipients resolves the users a splash should reach. Deterministic; callers
// (the terminal + gmcp listeners) partition by client type afterward.
func Recipients(s Splash) []*users.UserRecord {
	switch s.Target {
	case TargetUser:
		if u := users.GetByUserId(s.UserId); u != nil {
			return []*users.UserRecord{u}
		}
		return nil
	case TargetZone:
		return usersInZone(s.Zone)
	default: // TargetGlobal
		return users.GetAllActiveUsers()
	}
}

// usersInZone returns online users currently in any room of zone.
func usersInZone(zone string) []*users.UserRecord {
	out := []*users.UserRecord{}
	seen := map[int]bool{}
	for _, roomId := range rooms.GetAllZoneRoomsIds(zone) {
		room := rooms.LoadRoom(roomId)
		if room == nil {
			continue
		}
		for _, uid := range room.GetPlayers() {
			if seen[uid] {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				seen[uid] = true
				out = append(out, u)
			}
		}
	}
	return out
}
