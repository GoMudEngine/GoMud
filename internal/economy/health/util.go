package health

// sortUint64s sorts a uint64 slice in-place using insertion sort.
// Insertion sort is efficient for the small slices that appear in
// TtR calculations (typically <20 events per shop).
func sortUint64s(s []uint64) {
	for i := 1; i < len(s); i++ {
		key := s[i]
		j := i - 1
		for j >= 0 && s[j] > key {
			s[j+1] = s[j]
			j--
		}
		s[j+1] = key
	}
}
