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

	MSSP MSSPConfig `yaml:"MSSP"` // MUD Server Status Protocol (telnet option 70) advertising.
}

// MSSPConfig holds the static descriptive fields advertised over MSSP. Live
// fields (player count, uptime) and auto-derived fields (world counts,
// capability flags) are computed at request time, not stored here.
type MSSPConfig struct {
	Enabled  ConfigBool        `yaml:"Enabled"`  // Master toggle for the MSSP option.
	Website  ConfigString      `yaml:"Website"`  // Public website URL.
	Genre    ConfigString      `yaml:"Genre"`    // e.g. Fantasy.
	Gameplay ConfigSliceString `yaml:"Gameplay"` // e.g. Adventure, Roleplaying (multi-value).
	Status   ConfigString      `yaml:"Status"`   // Alpha / Beta / Open Beta / Live.
	Language ConfigString      `yaml:"Language"` // e.g. English.
	Family   ConfigString      `yaml:"Family"`   // Codebase family; GoMud is Custom.
	Location ConfigString      `yaml:"Location"` // Server region.
	Created  ConfigString      `yaml:"Created"`  // Year created.
	Contact  ConfigString      `yaml:"Contact"`  // Public contact email; EMPTY by default (privacy — public repo).
	Hostname ConfigString      `yaml:"Hostname"` // Canonical connect host; empty => crawler uses the connecting socket.
	Port     ConfigString      `yaml:"Port"`     // Canonical telnet port; empty => omitted.
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

	// MSSP descriptive defaults (Contact/Hostname/Port intentionally left empty).
	if s.MSSP.Website == `` {
		s.MSSP.Website = `https://www.dogmud.org`
	}
	if s.MSSP.Genre == `` {
		s.MSSP.Genre = `Fantasy`
	}
	if len(s.MSSP.Gameplay) == 0 {
		s.MSSP.Gameplay = ConfigSliceString{`Adventure`, `Roleplaying`}
	}
	if s.MSSP.Status == `` {
		s.MSSP.Status = `Open Beta`
	}
	if s.MSSP.Language == `` {
		s.MSSP.Language = `English`
	}
	if s.MSSP.Family == `` {
		s.MSSP.Family = `Custom`
	}
	if s.MSSP.Location == `` {
		s.MSSP.Location = `United States`
	}
	if s.MSSP.Created == `` {
		s.MSSP.Created = `2026`
	}

}

func GetServerConfig() Server {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Server
}
