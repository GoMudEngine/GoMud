package mobcommands

import (
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/species"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func Aid(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error) {

	raceInfo := species.GetSpecies(mob.Character.SpeciesId)
	if !raceInfo.KnowsFirstAid {

		mob.Command(`emote doesn't know first aid.`)

		return true, nil
	}

	if !room.IsCalm() {
		return true, nil
	}

	if rest == `` {
		return true, nil
	}

	aidPlayerId, _ := room.FindByName(rest, rooms.FindDowned)

	if aidPlayerId > 0 {

		p := users.GetByUserId(aidPlayerId)

		if p != nil {

			if p.Character.Health > 0 {
				return true, nil
			}

			mob.Character.CancelBuffsWithFlag(buffs.Hidden)

			// Set spell Aid
			spellAggro := characters.SpellAggroInfo{
				SpellId:              `aidskill`,
				SpellRest:            ``,
				TargetUserIds:        []int{aidPlayerId},
				TargetMobInstanceIds: []int{},
			}

			continueCasting := true
			if allowToContinue, err := scripting.TrySpellScriptEvent(`onCast`, 0, mob.InstanceId, spellAggro); err == nil {
				continueCasting = allowToContinue
			}

			if continueCasting {

				spellInfo := spells.GetSpell(`aidskill`)
				mob.Character.CancelBuffsWithFlag(buffs.Hidden)
				mob.Character.SetCast(spellInfo.WaitRounds, spellAggro)
			}

		}

	}

	return true, nil
}
