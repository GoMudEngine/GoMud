package configs

// Playtest holds optional ephemeral-playtest materialization settings.
// Empty ProfilesManifest is the production default (materializer no-op).
type Playtest struct {
	ProfilesDir      ConfigString `yaml:"ProfilesDir"`
	ProfilesManifest ConfigString `yaml:"ProfilesManifest"`
}

func (p *Playtest) Validate() {
	if p.ProfilesDir == `` {
		p.ProfilesDir = `tools/playtest/profiles`
	}
}

func GetPlaytestConfig() Playtest {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Playtest
}
