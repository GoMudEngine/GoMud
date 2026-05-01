package parties

import "fmt"

// actorIdentity is the minimal interface ActorKeyFor needs. It matches the
// corresponding methods on actions.Actor, letting callers pass any Actor
// without creating an import cycle (internal/actions already imports
// internal/parties).
type actorIdentity interface {
	IsPlayer() bool
	GetUserId() int
	GetMobInstanceId() int
}

// ActorKey is an opaque map-key derived from an Actor. It encodes which
// concrete actor type the key represents (user or mob) and the underlying
// numeric ID, so a user with ID 42 and a mob instance with ID 42 produce
// distinct keys.
type ActorKey string

// ActorKeyFor returns the ActorKey for an Actor. Player actors yield
// "user:<userId>"; mob actors yield "mob:<mobInstanceId>".
// Passing nil returns an empty ActorKey.
func ActorKeyFor(a actorIdentity) ActorKey {
	if a == nil {
		return ""
	}
	if a.IsPlayer() {
		return ActorKey(fmt.Sprintf("user:%d", a.GetUserId()))
	}
	return ActorKey(fmt.Sprintf("mob:%d", a.GetMobInstanceId()))
}
