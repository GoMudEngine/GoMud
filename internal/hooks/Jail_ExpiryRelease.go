package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/justice"
)

// releaseIfSentenceServed releases a jailed player whose sentence timer has
// expired. Pure decision (jailExpired) + effectful release, split so the
// decision is unit-testable without a live character/world. Called once per
// player per round from UserRoundTick.
func releaseIfSentenceServed(c *characters.Character, userId int, now uint64) bool {
	rec, ok := justice.JailInfo(c)
	if !ok {
		return false
	}
	if !jailExpired(rec.UntilRound, now) {
		return false
	}
	return justice.ResolveDetention(c, userId)
}

// jailExpired reports whether a sentence ending at untilRound has elapsed by
// now. Pure.
func jailExpired(untilRound, now uint64) bool {
	return untilRound > 0 && now >= untilRound
}
