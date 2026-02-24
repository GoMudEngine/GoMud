package configs

type Network struct {
	MaxTelnetConnections ConfigInt         `yaml:"MaxTelnetConnections"` // Maximum number of telnet connections to accept
	TelnetPort           ConfigSliceString `yaml:"TelnetPort"`           // One or more Ports used to accept telnet connections
	AIPort               ConfigInt         `yaml:"AIPort"`               // Dedicated port for AI clients (0 = disabled)
	MaxHumanConnections  ConfigInt         `yaml:"MaxHumanConnections"`  // Pool limit for human connections
	MaxAIConnections     ConfigInt         `yaml:"MaxAIConnections"`     // Pool limit for AI connections
	AICommandsPerRound   ConfigInt         `yaml:"AICommandsPerRound"`   // Max commands per game round for AI connections
	LocalPort            ConfigInt         `yaml:"LocalPort"`            // Port used for admin connections, localhost only
	HttpPort             ConfigInt         `yaml:"HttpPort"`             // Port used for web requests
	HttpsPort            ConfigInt         `yaml:"HttpsPort"`            // Port used for web https requests
	HttpsRedirect        ConfigBool        `yaml:"HttpsRedirect"`        // If true, http traffic will be redirected to https
	AfkSeconds           ConfigInt         `yaml:"AfkSeconds"`           // How long until a player is marked as afk?
	MaxIdleSeconds       ConfigInt         `yaml:"MaxIdleSeconds"`       // How many seconds a player can go without a command in game before being kicked.
	TimeoutMods          ConfigBool        `yaml:"TimeoutMods"`          // Whether to kick admin/mods when idle too long.
	ZombieSeconds        ConfigInt         `yaml:"ZombieSeconds"`        // How many seconds a player will be a zombie allowing them to reconnect.
	LogoutRounds         ConfigInt         `yaml:"LogoutRounds"`         // How many rounds of uninterrupted meditation must be completed to log out.
}

func (n *Network) Validate() {

	// Ignore TelnetPort
	// Ignore LocalPort
	// Ignore TimeoutMods

	if n.MaxTelnetConnections < 1 {
		n.MaxTelnetConnections = 50 // default
	}

	if n.AIPort < 0 {
		n.AIPort = 0 // disabled by default
	}

	if n.MaxHumanConnections < 1 {
		n.MaxHumanConnections = 50 // default
	}

	if n.MaxAIConnections < 1 {
		n.MaxAIConnections = 20 // default
	}

	if n.AICommandsPerRound < 1 {
		n.AICommandsPerRound = 2 // default
	}

	if n.HttpPort < 0 {
		n.HttpPort = 0 // default
	}

	if n.HttpsPort < 0 {
		n.HttpsPort = 0 // default
	}

	if n.AfkSeconds < 0 {
		n.AfkSeconds = 0
	}

	if n.MaxIdleSeconds < 0 {
		n.MaxIdleSeconds = 0
	}

	if n.ZombieSeconds < 0 {
		n.ZombieSeconds = 0
	}

	if n.LogoutRounds < 0 {
		n.LogoutRounds = 0 // default
	}

}

func GetNetworkConfig() Network {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Network
}
