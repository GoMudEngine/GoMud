package configs

type Memory struct {
	// Mob/Room memory unload thresholds
	MobUnloadThreshold  ConfigInt `yaml:"MobUnloadThreshold"`
	RoomUnloadRounds    ConfigInt `yaml:"RoomUnloadRounds"`
	RoomUnloadThreshold ConfigInt `yaml:"RoomUnloadThreshold"`
}

func (m *Memory) Validate() {

	if m.MobUnloadThreshold < 0 {
		m.MobUnloadThreshold = 0
	}

	if m.RoomUnloadRounds < 5 {
		m.RoomUnloadRounds = 5
	}

	if m.RoomUnloadThreshold < 0 {
		m.RoomUnloadThreshold = 0
	}

}

func GetMemoryConfig() Memory {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Memory
}
