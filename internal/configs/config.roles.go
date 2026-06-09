package configs

type Roles map[string]ConfigSliceString

func (m *Roles) Validate() {
}

func GetRolesConfig() Roles {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()

	return configData.Roles
}
