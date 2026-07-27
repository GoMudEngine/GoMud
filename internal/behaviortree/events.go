package behaviortree

// The pinned behavior-event vocabulary (5d). Events are matched by raw
// string equality at runtime with no boot validation — before this file, a
// typo'd `event:` was a silently dead node (the caravan wagon's on-death
// cargo drop authored `mob_death`, the QUEST engine's name; the btree engine
// fires `mob_die`, and the node never ran).
//
// events_test.go enforces BOTH directions against the Go dispatch sites:
// an event fired anywhere but missing here fails CI, and so does an entry
// here that nothing fires. The 5d editor refuses saves whose event isn't in
// this map.

import "sort"

var KnownBehaviorEvents = map[string]bool{
	"heard_callforhelp":      true, // a routine-matched packmate in an adjacent room called for help
	"mob_combat_round":       true, // fires each combat round for a fighting mob
	"mob_die":                true, // the mob just died (instant actions only — no respond)
	"mob_flee":               true, // the mob is fleeing
	"mob_hurt":               true, // the mob took damage
	"mob_idle":               true, // idle tick (out of combat)
	"packmate_hurt":          true, // a same-room routine-matched packmate was attacked
	"player_ask":             true, // a player asked this mob about a topic
	"player_attack_rejected": true, // a player's attack on this mob was rejected (e.g. non-combatant)
	"player_enter":           true, // a player entered the mob's room
	"player_give":            true, // a player gave this mob an item
	"room_command":           true, // room trees: a command typed in the room (interception)
	"room_enter":             true, // room trees: someone entered the room
	"room_exit":              true, // room trees: someone left the room
	"room_idle":              true, // room trees: idle tick
}

// EventNames returns the vocabulary sorted, for editor enums.
func EventNames() []string {
	out := make([]string, 0, len(KnownBehaviorEvents))
	for k := range KnownBehaviorEvents {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
