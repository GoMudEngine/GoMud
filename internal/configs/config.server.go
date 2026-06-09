package configs

type Server struct {
	MudName         ConfigString      `yaml:"MudName"`         // Name of the MUD
	CurrentVersion  ConfigString      `yaml:"CurrentVersion"`  // Current version this mud has been updated to
	Seed            ConfigSecret      `yaml:"Seed"`            // Seed that may be used for generating content
	MaxCPUCores     ConfigInt         `yaml:"MaxCPUCores"`     // How many cores to allow for multi-core operations
	OnLoginCommands ConfigSliceString `yaml:"OnLoginCommands"` // Commands to run when a user logs in
	Motd            ConfigString      `yaml:"Motd"`            // Message of the day to display when a user logs in
	NextRoomId      ConfigInt         `yaml:"NextRoomId"`      // The next room id to use when creating a new room
	Locked          ConfigSliceString `yaml:"Locked"`          // List of locked config properties that cannot be changed without editing the file directly.

	// Presence machine thresholds (chunk 5 — 2026-05-19)
	PresenceIdleAfterRounds       ConfigInt `yaml:"PresenceIdleAfterRounds"`       // Active → Idle after N rounds with no command
	PresenceAFKAfterRounds        ConfigInt `yaml:"PresenceAFKAfterRounds"`        // Idle → AFK after N rounds
	PresenceDisconnectAfterRounds ConfigInt `yaml:"PresenceDisconnectAfterRounds"` // AFK → Disconnected after N rounds
	PresenceMobDormantAfterRounds ConfigInt `yaml:"PresenceMobDormantAfterRounds"` // mob Active → Dormant after N rounds
	PresenceMobDespawnAfterRounds ConfigInt `yaml:"PresenceMobDespawnAfterRounds"` // mob Dormant → Despawning after N rounds
}

func (s *Server) Validate() {

	// Ignore MudName
	// Ignore OnLoginCommands
	// Ignore Motd
	// Ignore NextRoomId
	// Ignore Locked

	if s.Seed == `` {
		s.Seed = `Mud` // default
	}

	if s.MaxCPUCores < 0 {
		s.MaxCPUCores = 0 // default
	}

	if s.CurrentVersion == `` {
		s.CurrentVersion = `0.9.0` // If no version found, failover to a known version
	}

	if s.PresenceIdleAfterRounds < 1 {
		s.PresenceIdleAfterRounds = 8 // ~30s at 4s/round
	}
	if s.PresenceAFKAfterRounds < 1 {
		s.PresenceAFKAfterRounds = 75 // ~5min
	}
	if s.PresenceDisconnectAfterRounds < 1 {
		s.PresenceDisconnectAfterRounds = 900 // ~1h
	}
	if s.PresenceMobDormantAfterRounds < 1 {
		s.PresenceMobDormantAfterRounds = 30
	}
	if s.PresenceMobDespawnAfterRounds < 1 {
		s.PresenceMobDespawnAfterRounds = 60
	}

}

func GetServerConfig() Server {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Server
}
