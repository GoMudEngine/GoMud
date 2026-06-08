package playtest

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// containsTag reports whether tags includes tag.
func containsTag(tags []string, tag string) bool {
	for _, t := range tags {
		if t == tag {
			return true
		}
	}
	return false
}

// shouldSnapBack decides whether a moved tester must be returned to the sandbox.
// Pure for testability. Snap back when: the mover is on the AI port, a sandbox
// tag is configured, and the destination room does not carry that tag (fail
// closed: not provably inside the sandbox -> refuse).
func shouldSnapBack(isAITester bool, sandboxTag string, destTags []string) bool {
	if !isAITester || sandboxTag == "" {
		return false
	}
	return !containsTag(destTags, sandboxTag)
}

func (m *PlaytestModule) registerSafeMode() {
	// Sandbox confinement needs a tag to confine to.
	// DOGMud's Plugin has no ReserveTags method; we skip that call and rely
	// on the per-move tag check in onRoomChange instead.
	if m.cfg.SafeMode && m.cfg.SandboxZoneTag != "" {
		events.RegisterListener(events.RoomChange{}, m.onRoomChange)
	}
	// Death protection is applied to a tester when it spawns into the world.
	if m.cfg.DeathProtection {
		events.RegisterListener(events.PlayerSpawn{}, m.onPlayerSpawn)
	}
}

func (m *PlaytestModule) onRoomChange(e events.Event) events.ListenerReturn {
	// DOGMud's Room struct has no Tags field; sandbox zone confinement
	// (shouldSnapBack) requires per-room tags and cannot function in this
	// engine fork. The handler is registered so event wiring is present, but
	// no snap-back action is taken. Configure SandboxZoneTag only if a future
	// DOGMud build adds room tags.
	_ = e
	return events.Continue
}

// onPlayerSpawn applies death protection to a tester as it enters the world,
// keyed off the AI-port connection rather than an account flag.
func (m *PlaytestModule) onPlayerSpawn(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerSpawn)
	if !ok || !isAIConnection(evt.ConnectionId) {
		return events.Continue
	}
	if u := users.GetByUserId(evt.UserId); u != nil {
		m.applyDeathProtection(u)
	}
	return events.Continue
}

// applyDeathProtection is a no-op on DOGMud: the engine has no permadeath /
// extra-lives mechanic (death routes through justice/jail; bleedout/downed were
// removed), so an AI tester cannot permanently die. Kept for interface parity.
func (m *PlaytestModule) applyDeathProtection(u *users.UserRecord) {
	_ = u
}
