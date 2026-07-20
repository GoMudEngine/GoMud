package rooms

// ForEachAdjacentRoom visits every room reachable from r through any exit
// source — standard exits, temporary exits, and exits contributed by active
// room mutators — invoking fn with that room and the name of the exit leading
// back to r.
//
// This exists because "walk everything adjacent" was hand-rolled per caller,
// and the copies drifted: usercommands/shout.go walked all three sources while
// mobcommands/shout.go walked only Exits, so mob shouts silently failed to
// reach rooms connected by a temporary or mutator-added exit.
//
// Rooms reachable through more than one source are visited once. Callers that
// need the room's own exit set (rather than the way back to r) should walk
// r.Exits directly — this helper is for outward sound/effect propagation.
//
// Note this is deliberately not SendTextToExits: that helper covers only Exits
// and ExitsTemp, and hard-codes its message format, which is why callers
// needing mutator exits or custom wording rolled their own loops.
func (r *Room) ForEachAdjacentRoom(fn func(otherRoom *Room, sourceExit string)) {

	seen := map[int]struct{}{}

	visit := func(roomId int) {
		if roomId == r.RoomId {
			return
		}
		if _, dup := seen[roomId]; dup {
			return
		}

		otherRoom := LoadRoom(roomId)
		if otherRoom == nil {
			return
		}

		// Only propagate where there is a way back — a one-way exit into r
		// gives the far room no direction to attribute the sound to.
		sourceExit := otherRoom.FindExitTo(r.RoomId)
		if sourceExit == `` {
			return
		}

		seen[roomId] = struct{}{}
		fn(otherRoom, sourceExit)
	}

	for _, exitInfo := range r.Exits {
		visit(exitInfo.RoomId)
	}

	for _, exitInfo := range r.ExitsTemp {
		visit(exitInfo.RoomId)
	}

	for mut := range r.ActiveMutators {
		spec := mut.GetSpec()
		for _, exitInfo := range spec.Exits {
			visit(exitInfo.RoomId)
		}
	}
}
