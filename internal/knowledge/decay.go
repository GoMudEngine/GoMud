package knowledge

import "sort"

type RoomCount struct {
	Room  int
	Count int
}

// FrequentedRooms returns the top-K rooms in this observer's bounded
// observation log of the subject, sorted by count descending. Stable
// secondary sort by room id (for deterministic output across ties).
func FrequentedRooms(observerMobId int, subject Subject, topK int) []RoomCount {
	r := Get(observerMobId, subject)
	if r == nil || len(r.Observations) == 0 {
		return nil
	}

	tally := make(map[int]int, 8)
	for _, o := range r.Observations {
		tally[o.Room]++
	}

	rows := make([]RoomCount, 0, len(tally))
	for room, count := range tally {
		rows = append(rows, RoomCount{Room: room, Count: count})
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count != rows[j].Count {
			return rows[i].Count > rows[j].Count
		}
		return rows[i].Room < rows[j].Room
	})

	if topK > 0 && len(rows) > topK {
		rows = rows[:topK]
	}
	return rows
}
