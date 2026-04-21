package hooks

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/questengine"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// respawnCompanions re-creates mob instances for all stored companions.
// Called once the player is fully in the world.
func respawnCompanions(user *users.UserRecord) {
	// First: clean up any stale instances from a previous session
	// (e.g., browser refresh without clean logout).
	for i := range user.Character.Companions {
		comp := &user.Character.Companions[i]
		if comp.InstanceId > 0 {
			if oldMob := mobs.GetInstance(comp.InstanceId); oldMob != nil {
				oldMob.Character.RemoveCharm()
				if oldRoom := rooms.LoadRoom(oldMob.Character.RoomId); oldRoom != nil {
					oldRoom.RemoveMob(comp.InstanceId)
				}
				mobs.DestroyInstance(comp.InstanceId)
			}
			comp.InstanceId = 0
		}
	}
	// Also clean up any charmed mobs that aren't in the Companions list
	for _, charmId := range user.Character.GetCharmIds() {
		user.Character.TrackCharmed(charmId, false)
		if oldMob := mobs.GetInstance(charmId); oldMob != nil {
			oldMob.Character.RemoveCharm()
			if oldRoom := rooms.LoadRoom(oldMob.Character.RoomId); oldRoom != nil {
				oldRoom.RemoveMob(charmId)
			}
			mobs.DestroyInstance(charmId)
		}
	}

	// Remove charmed companions — they don't persist through restart.
	// Charmed mobs are temporary by nature (borrowed, not created).
	cleaned := make([]characters.CompanionInfo, 0, len(user.Character.Companions))
	for _, comp := range user.Character.Companions {
		if comp.SourceType != characters.CompanionCharmed {
			cleaned = append(cleaned, comp)
		}
	}
	user.Character.Companions = cleaned

	for i := range user.Character.Companions {
		comp := &user.Character.Companions[i]
		if comp.MobId == 0 {
			continue
		}

		mob := mobs.NewMobByIdFresh(mobs.MobId(comp.MobId), user.Character.RoomId)
		if mob == nil {
			mudlog.Error("respawnCompanions", "error", fmt.Sprintf("mob template %d not found for companion %s", comp.MobId, comp.Name))
			continue
		}

		// Apply saved progression back onto the fresh mob instance.
		applyCompanionState(mob, comp)

		// Apply nickname if the companion was renamed.
		if comp.Nickname != "" {
			mob.Character.Name = comp.Name
		}

		// Restore to full resources.
		mob.Character.Health = mob.Character.HealthMax.Value
		mob.Character.Stamina = mob.Character.StaminaMax.Value
		mob.Character.Conviction = mob.Character.ConvictionMax.Value

		// Charm permanently to the player (-1 rounds = permanent).
		mob.Character.Charm(user.UserId, -1, "")
		user.Character.TrackCharmed(mob.InstanceId, true)

		// Place in the player's current room.
		if room := rooms.LoadRoom(user.Character.RoomId); room != nil {
			room.AddMob(mob.InstanceId)
		}

		comp.InstanceId = mob.InstanceId
	}
}

// applyCompanionState writes saved CompanionInfo progression onto a fresh mob.
func applyCompanionState(mob *mobs.Mob, comp *characters.CompanionInfo) {
	// Stat training.
	if comp.StatTraining != nil {
		mob.Character.Stats.Strength.Training = comp.StatTraining["strength"]
		mob.Character.Stats.Dexterity.Training = comp.StatTraining["dexterity"]
		mob.Character.Stats.Perception.Training = comp.StatTraining["perception"]
		mob.Character.Stats.Vitality.Training = comp.StatTraining["vitality"]
		mob.Character.Stats.Willpower.Training = comp.StatTraining["willpower"]
		mob.Character.Stats.Charisma.Training = comp.StatTraining["charisma"]
	}

	// Skill maps.
	if comp.Skills != nil {
		mob.Character.Skills = copyIntMap(comp.Skills)
	}
	if comp.SkillUseCount != nil {
		mob.Character.SkillUseCount = copyIntMap(comp.SkillUseCount)
	}
	if comp.Mutations != nil {
		mob.Character.Mutations = copyIntMap(comp.Mutations)
	}
	if comp.SpellBook != nil {
		mob.Character.SpellBook = copyIntMap(comp.SpellBook)
	}
	mob.Character.MutationProgress = comp.MutationProgress

	// Recalculate derived stats from the applied training.
	mob.Character.Validate()
}

//
// Execute on join commands
//

func HandleJoin(e events.Event) events.ListenerReturn {

	evt, typeOk := e.(events.PlayerSpawn)
	if !typeOk {
		mudlog.Error("Event", "Expected Type", "PlayerSpawn", "Actual Type", e.Type())
		return events.Cancel
	}

	user := users.GetByUserId(evt.UserId)
	if user == nil {
		mudlog.Error("HandleJoin", "error", fmt.Sprintf(`user %d not found`, evt.UserId))
		return events.Cancel
	}

	user.EventLog.Add(`conn`, fmt.Sprintf(`<ansi fg="username">%s</ansi> entered the world`, user.Character.Name))

	users.RemoveZombieUser(evt.UserId)

	room := rooms.LoadRoom(user.Character.RoomId)
	if room == nil {

		mudlog.Error("EnterWorld", "error", fmt.Sprintf(`room %d not found`, user.Character.RoomId))

		if err := rooms.MoveToRoom(user.UserId, 0); err != nil {
			mudlog.Error("EnterWorld", "msg", "could not move to room 0", "error", err)
		}

		room = rooms.LoadRoom(user.Character.RoomId)
	}

	// Respawn any saved companions.
	respawnCompanions(user)

	// TODO HERE
	loginCmds := configs.GetConfig().Server.OnLoginCommands
	if len(loginCmds) > 0 {

		for _, cmd := range loginCmds {

			events.AddToQueue(events.Input{
				UserId:    evt.UserId,
				InputText: cmd,
				ReadyTurn: 0, // No delay between execution of commands
			})

		}

	}

	if room != nil {
		// Quest engine: room_enter notification on login/spawn
		bridge := questengine.NewGameBridge(user, user.Character.RoomId)
		questengine.GetEngine().Notify("room_enter", questengine.EventDetails{
			UserId: user.UserId,
			RoomId: user.Character.RoomId,
		}, bridge, bridge)

		user.CommandFlagged(`look`, events.CmdSecretly) // Do a secret look.
	}

	return events.Continue
}
