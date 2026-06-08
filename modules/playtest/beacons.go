package playtest

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/plugins"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// beaconPayload is the body of a Playtest.Round GMCP beacon: a per-round tick
// plus a compact state snapshot the agent can pace and score goals against.
// DOGMud has three resource pools: Health (hp), Stamina (sp), Conviction (cp).
type beaconPayload struct {
	Round  uint64 `json:"round"`
	HP     int    `json:"hp"`
	HPMax  int    `json:"hp_max"`
	SP     int    `json:"sp"`
	SPMax  int    `json:"sp_max"`
	CP     int    `json:"cp"`
	CPMax  int    `json:"cp_max"`
	RoomID int    `json:"room_id"`
}

// registerBeacons wires the per-round beacon if enabled and gmcp's sender is
// available. Beacons fire on events.NewRound for each connected IsAI user. If
// the gmcp module is absent, it logs a warning and stays disabled.
func (m *PlaytestModule) registerBeacons() {
	if !m.cfg.Beacons {
		return
	}
	f, ok := plugins.GetPluginRegistry().GetExportedFunction("SendGMCPEvent")
	if !ok {
		mudlog.Warn("playtest", "msg", "Beacons enabled but gmcp SendGMCPEvent not found; is the gmcp module installed?")
		return
	}
	send, ok := f.(func(int, string, any))
	if !ok {
		mudlog.Error("playtest", "msg", "SendGMCPEvent has unexpected signature; beacons disabled")
		return
	}
	m.sendGMCP = send
	events.RegisterListener(events.NewRound{}, m.onNewRound)
}

// onNewRound emits a Playtest.Round beacon to every live session on the AI
// port. Targeting the connection (not an IsAI account flag) means no account
// provisioning is required — a client on the AI port is a tester by definition.
func (m *PlaytestModule) onNewRound(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.NewRound)
	if !ok || m.sendGMCP == nil {
		return events.Continue
	}
	for _, u := range users.GetAllActiveUsers() {
		if u.Character == nil || !isAIConnection(u.ConnectionId()) {
			continue
		}
		m.sendGMCP(u.UserId, "Playtest.Round", beaconPayload{
			Round:  evt.RoundNumber,
			HP:     u.Character.Health,
			HPMax:  u.Character.HealthMax.Value,
			SP:     u.Character.Stamina,
			SPMax:  u.Character.StaminaMax.Value,
			CP:     u.Character.Conviction,
			CPMax:  u.Character.ConvictionMax.Value,
			RoomID: u.Character.RoomId,
		})
	}
	return events.Continue
}
