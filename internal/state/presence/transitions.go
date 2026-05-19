package presence

import "github.com/GoMudEngine/GoMud/internal/state"

// playerTransitions enforces the Presence invariant matrix for player actors.
//
//	Connecting --in-world (character entered room)--> Active
//	Active     --N rounds no input------------------> Idle
//	Idle       --M rounds no input------------------> AFK
//	AFK        --hours OR TCP timeout---------------> Disconnected
//
//	[any non-Disconnected] --input received---------> Active
//	[any non-Connecting]   --manual `afk` cmd-------> AFK
//	[any]                  --TCP closed-------------> Disconnected
var playerTransitions = state.TransitionTable[State]{
	Connecting:   {Active, Disconnected},
	Active:       {Idle, AFK, Disconnected},
	Idle:         {Active, AFK, Disconnected},
	AFK:          {Active, Disconnected},
	Disconnected: {}, // terminal until reconnect (new Machine instance)
}

// mobTransitions enforces the Presence invariant matrix for mob actors.
//
//	Spawning   --auto next tick-------------------> Active
//	Active     --bored N rounds && !essential-----> Dormant
//	Active     --zone has no nearby players-------> Dormant
//	Dormant    --player nearby OR attacked--------> Active
//	Dormant    --too long alone && !essential-----> Despawning
//	Despawning --next tick------------------------> (removed)
var mobTransitions = state.TransitionTable[State]{
	Spawning:   {Active},
	Active:     {Dormant, Despawning}, // Despawning direct from Active for force-despawn
	Dormant:    {Active, Despawning},
	Despawning: {}, // terminal — mob is removed on next tick
}

// Trigger reason constants. Used in state.TransitionReason.Trigger.
const (
	TriggerInputReceived     = "input_received"
	TriggerManualAFK         = "manual_afk"
	TriggerTimeoutIdle       = "timeout_idle"
	TriggerTimeoutAFK        = "timeout_afk"
	TriggerTimeoutDisconnect = "timeout_disconnect"
	TriggerTCPClosed         = "tcp_closed"
	TriggerEnteredRoom       = "entered_room"
	TriggerPlayerEntry       = "player_entry"
	TriggerAttacked          = "attacked"
	TriggerBored             = "bored"
	TriggerZoneEmpty         = "zone_empty"
	TriggerDormantTooLong    = "dormant_too_long"
	TriggerSpawnTickResolve  = "spawn_tick_resolve"
)
