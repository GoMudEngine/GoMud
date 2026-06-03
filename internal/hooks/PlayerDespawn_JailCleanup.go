package hooks

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/justice"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// PlayerDespawnJailCleanup tears down a jailed player's ephemeral cell
// instance when they log out / disconnect, preserving the jail record
// (the sentence clock is absolute) and rewriting the saved room to the
// faction's static fallback cell so the character never loads into a
// now-dead ephemeral room. RestoreJailOnLogin (a separate later task)
// handles re-instancing or releasing the player on return.
func PlayerDespawnJailCleanup(e events.Event) events.ListenerReturn {
	evt, ok := e.(events.PlayerDespawn)
	if !ok {
		return events.Continue
	}

	u := users.GetByUserId(evt.UserId)
	if u == nil {
		return events.Continue
	}

	justice.HandleJailedDespawn(u.Character)

	return events.Continue
}
