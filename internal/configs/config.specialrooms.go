package configs

type SpecialRooms struct {
	StartRoom         ConfigInt         `yaml:"StartRoom"`         // Default starting room.
	DeathRecoveryRoom ConfigInt         `yaml:"DeathRecoveryRoom"` // Recovery room after dying.
	TutorialRooms     ConfigSliceString `yaml:"TutorialRooms"`     // List of all rooms that can be used to begin the tutorial process
}

func (s *SpecialRooms) Validate() {

	// Ignore StartRoom
	// Ignore DeathRecoveryRoom
	// Ignore TutorialRooms

}

func GetSpecialRoomsConfig() SpecialRooms {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.SpecialRooms
}
