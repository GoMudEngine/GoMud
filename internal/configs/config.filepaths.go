package configs

type FilePaths struct {
	WebDomain        ConfigString `yaml:"WebDomain"`
	WebCDNLocation   ConfigString `yaml:"WebCDNLocation"`
	DataFiles        ConfigString `yaml:"DataFiles"`
	PublicHtml       ConfigString `yaml:"PublicHtml"`
	AdminHtml        ConfigString `yaml:"AdminHtml"`
	HttpsCertFile    ConfigString `yaml:"HttpsCertFile"`
	HttpsKeyFile     ConfigString `yaml:"HttpsKeyFile"`
	CarefulSaveFiles ConfigBool   `yaml:"CarefulSaveFiles"`
}

func (f *FilePaths) Validate() {

	// Ignore WebDomain
	// Ignore WebCDNLocation
	// Ignore PublicHtml
	// Ignore AdminHtml
	// Ignore CarefulSaveFiles

	if f.DataFiles == `` {
		f.DataFiles = `_datafiles/world/default` // default
	}

}

func GetFilePathsConfig() FilePaths {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.FilePaths
}
