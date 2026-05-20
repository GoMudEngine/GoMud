package mobcommands

import (
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

func Converse(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	// Don't bother if no players are present
	if mob.InConversation() {
		return true, nil
	}

	if !conversations.HasConverseFile(int(mob.MobId), mob.Character.Zone) {
		return true, nil
	}

	isSneaking := mob.Character.IsHidden()

	if isSneaking {
		return true, nil
	}

	for _, mobInstId := range room.GetMobs() {

		if mobInstId == mob.InstanceId { // no conversing with self
			continue
		}

		if m := mobs.GetInstance(mobInstId); m != nil {

			// Not allowed to start another conversation until this one concludes
			if m.InConversation() {
				continue
			}

			conversationId := 0
			if rest != `` {
				forceIndex, _ := strconv.Atoi(rest)
				conversationId = conversations.AttemptConversation(int(mob.MobId), mob.InstanceId, mob.Character.Name,
					m.InstanceId, m.Character.Name,
					mob.Character.Zone, forceIndex)
			} else {
				conversationId = conversations.AttemptConversation(int(mob.MobId), mob.InstanceId, mob.Character.Name,
					m.InstanceId, m.Character.Name,
					mob.Character.Zone)
			}

			if conversationId > 0 {
				mob.SetConversation(conversationId)
				m.SetConversation(conversationId)
				break
			}
		}
	}

	return true, nil
}
