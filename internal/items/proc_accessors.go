package items

import "sort"

// ValidProcTriggers / ValidProcEffects return the accepted proc trigger and
// effect ids, sorted — for the web item editor's dropdowns and any external
// validation. Mirrors the validProcTriggers/validProcEffects maps used by
// ItemSpec.Validate.
func ValidProcTriggers() []string { return sortedBoolKeys(validProcTriggers) }
func ValidProcEffects() []string  { return sortedBoolKeys(validProcEffects) }

func sortedBoolKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
