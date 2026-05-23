package hooks

// MobRoomChange_ShadowFollow.go — Chunk 2.8 smoke fix
//
// Closes the gap where `shadow <mob>` applied buff 87 and set
// shadow-target-mob, but had no movement consumer. When a mob moves, any
// player shadowing it (buff 87 + hidden + in old room) is auto-followed to
// the destination, mirroring the user-target path at usercommands/go.go:380-421.

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// MobRoomChangeShadowFollow auto-moves any player who is shadowing the
// moving mob, as long as they are still hidden and in the old room. On
// entry into the new room, if the hidden buff is gone (room-entry detection
// already fired during Command("go ...")) the shadow is torn down inline —
// mirroring endShadow in skill.skullduggery.shadow.go without importing
// the usercommands package.
func MobRoomChangeShadowFollow(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.RoomChange)
	if !ok {
		return events.Continue
	}
	// Player move — not our concern.
	if evt.MobInstanceId == 0 {
		return events.Continue
	}
	// Derive the exit direction from the old room.
	fromRoom := rooms.LoadRoom(evt.FromRoomId)
	if fromRoom == nil {
		return events.Continue
	}
	exitName := fromRoom.FindExitTo(evt.ToRoomId)
	if exitName == "" {
		// No named exit found (teleport / force-move); cannot follow.
		return events.Continue
	}

	for _, u := range users.GetAllActiveUsers() {
		if u == nil {
			continue
		}
		// Must carry the shadowing buff.
		if !u.Character.HasBuff(shadowingBuff) {
			continue
		}
		// Must still be hidden.
		if !u.Character.IsHidden() {
			continue
		}
		// Must be targeting this specific mob instance.
		v := u.Character.GetMiscData("shadow-target-mob")
		if v == nil {
			continue
		}
		targetId, ok := v.(int)
		if !ok || targetId != evt.MobInstanceId {
			continue
		}
		// Must be in the mob's previous room.
		if u.Character.RoomId != evt.FromRoomId {
			continue
		}

		// Dispatch the follow — reuses the full go.go movement path
		// (stamina checks, room-entry detection, etc.).
		u.Command(exitName)

		// After-move spotted check: if the room-entry detection stripped
		// the hidden buff, end shadow now. Mirrors go.go:411-414.
		if !u.Character.IsHidden() {
			inlineShadowEnd(u, "You've been spotted -- your shadow ends.")
		}
	}

	return events.Continue
}

// inlineShadowEnd replicates endShadow from skill.skullduggery.shadow.go
// without importing the usercommands package. Clears shadow misc state,
// removes buff 87, starts the cooldown, and delivers the reason message.
func inlineShadowEnd(u *users.UserRecord, reason string) {
	u.Character.SetMiscData("shadow-target-user", nil)
	u.Character.SetMiscData("shadow-target-mob", nil)
	u.Character.RemoveBuff(shadowingBuff)

	cfg := configs.GetBalanceConfig()
	cooldownKey := skills.Skullduggery.String(`shadow`)
	u.Character.TryCooldown(cooldownKey,
		fmt.Sprintf(`%d rounds`, cfg.ShadowCooldown))

	if reason != "" {
		u.SendText(messaging.CategorySystem, reason)
	}
}
