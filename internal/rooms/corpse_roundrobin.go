package rooms

// RoundRobinOrder returns the assigned member for each of nItems slots, dealing
// across members starting at startCursor, and the advanced cursor for the next
// deal. members must be non-empty; returns nil, startCursor if empty.
//
// The cursor is the index of the member who receives the NEXT dealt item, so a
// party can persist it across corpses to keep the rotation moving fairly.
func RoundRobinOrder(nItems int, members []int, startCursor int) (assignees []int, endCursor int) {
	n := len(members)
	if n == 0 {
		return nil, startCursor
	}

	// Normalize the incoming cursor into [0, n).
	cursor := startCursor % n
	if cursor < 0 {
		cursor += n
	}

	if nItems <= 0 {
		return nil, cursor
	}

	assignees = make([]int, nItems)
	for i := 0; i < nItems; i++ {
		assignees[i] = members[(cursor+i)%n]
	}
	endCursor = (cursor + nItems) % n
	return assignees, endCursor
}
