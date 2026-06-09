package configs

type Modules map[string]any

func (p *Modules) Validate() {

}

func GetModulesConfig() Modules {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Modules
}
