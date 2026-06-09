package casing

import "fmt"

// AssertCanonical panics if name is not already in canonical Title form. Call
// it from data loaders on player-visible display-name fields so non-canonical
// authored names fail fast at startup (caught by the pre-push boot test).
// Empty names are allowed (skip the check at the call site if desired).
func AssertCanonical(name, kind, source string) {
	if got := Title(name); got != name {
		panic(fmt.Sprintf(
			"casing: non-canonical %s name in %s: %q (expected %q)",
			kind, source, name, got))
	}
}
