// Package splash defines the reusable in-world splash event: one event, rendered
// per-client downstream (SVG on web, refined ASCII on terminal, caption for
// screen readers). Delivery lives in listeners (internal/hooks + modules/gmcp);
// this package only defines the event and resolves recipients.
package splash

import (
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

// Recipients resolves the users a splash should reach (the delivery hook then
// renders the scene per recipient).
func Recipients(s Splash) []*users.UserRecord {
	switch s.Target {
	case TargetUser:
		if u := users.GetByUserId(s.UserId); u != nil {
			return []*users.UserRecord{u}
		}
		return nil
	case TargetZone:
		// Resolve occupancy from the (small) online-player set, NOT by loading
		// the zone's rooms — loading rooms would disk-read + cache every empty
		// room in the zone just to find nobody.
		return filterByZone(users.GetAllActiveUsers(), s.Zone)
	default: // TargetGlobal
		return users.GetAllActiveUsers()
	}
}

// filterByZone returns the users whose current zone matches zone.
func filterByZone(all []*users.UserRecord, zone string) []*users.UserRecord {
	out := []*users.UserRecord{}
	for _, u := range all {
		if u != nil && u.Character != nil && u.Character.Zone == zone {
			out = append(out, u)
		}
	}
	return out
}
