package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

const guideSkillRankThreshold = 20

// CheckGuideBySkillRanks dismisses the guide NPC when a player's total skill
// ranks reach the threshold. Called from combat/progression hooks.
func CheckGuideBySkillRanks(userId int) {
	user := users.GetByUserId(userId)
	if user == nil {
		return
	}

	if user.Character.GetTotalSkillRanks() < guideSkillRankThreshold {
		return
	}

	for _, mobInstanceId := range user.Character.CharmedMobs {
		if mob := mobs.GetInstance(mobInstanceId); mob != nil {
			if mob.MobId == 38 {
				mob.Command(`say I see you have grown much stronger and more skilled. My assistance is now needed elsewhere. I wish you good luck!`)
				mob.Command(`emote clicks their heels together and disappears in a cloud of smoke.`, 10)
				mob.Command(`suicide vanish`, 10)
			}
		}
	}
}
