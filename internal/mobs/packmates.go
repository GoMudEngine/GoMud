package mobs

// FindPackmatesInRoom returns all mobs in the victim's room that should
// receive a packmate_hurt event when the victim is attacked.
//
// Inclusion rules (all must hold):
//   - Not the victim itself.
//   - Same RoomId as the victim.
//   - Health > 0 (dead mobs don't react).
//   - Not charmed (charmed mobs follow their owner, not the wild pack).
//   - Routine matches: either both mobs share a non-empty Routine, OR
//     the victim's Routine appears in the candidate's RoutineLinks, OR
//     the candidate's Routine appears in the victim's RoutineLinks.
//
// See docs/superpowers/specs/2026-04-22-pack-tactics-revamp-design.md
// (§ Packmate identification).
func FindPackmatesInRoom(victim *Mob) []*Mob {
	if victim == nil {
		return nil
	}
	if victim.Routine == "" && len(victim.RoutineLinks) == 0 {
		return nil
	}

	allIds := GetAllMobInstanceIds()
	packmates := make([]*Mob, 0, 4)

	for _, id := range allIds {
		if id == victim.InstanceId {
			continue
		}
		m := GetInstance(id)
		if m == nil {
			continue
		}
		if m.Character.RoomId != victim.Character.RoomId {
			continue
		}
		if m.Character.Health <= 0 {
			continue
		}
		if m.Character.IsCharmed() {
			continue
		}
		if !routinesMatch(victim, m) {
			continue
		}
		packmates = append(packmates, m)
	}
	return packmates
}

// routinesMatch returns true if a and b should be considered packmates
// for combat-reaction purposes.
func routinesMatch(a, b *Mob) bool {
	if a.Routine != "" && a.Routine == b.Routine {
		return true
	}
	for _, linked := range b.RoutineLinks {
		if linked != "" && linked == a.Routine {
			return true
		}
	}
	for _, linked := range a.RoutineLinks {
		if linked != "" && linked == b.Routine {
			return true
		}
	}
	return false
}
