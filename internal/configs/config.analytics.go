package configs

// Analytics holds configuration for the combat analytics subsystem.
type Analytics struct {
	Enabled          ConfigBool   `yaml:"Enabled"`          // master on/off switch
	MaxEvents        ConfigInt    `yaml:"MaxEvents"`        // ring buffer capacity; default 10000
	FlushIntervalSec ConfigInt    `yaml:"FlushIntervalSec"` // seconds between log flushes; default 300
	LogPath          ConfigString `yaml:"LogPath"`          // output path; default _datafiles/logs/combat-analytics.jsonl
}

func (a *Analytics) Validate() {
	if int(a.MaxEvents) <= 0 {
		a.MaxEvents.Set("10000")
	}
	if int(a.FlushIntervalSec) <= 0 {
		a.FlushIntervalSec.Set("300")
	}
	if string(a.LogPath) == "" {
		a.LogPath.Set("_datafiles/logs/combat-analytics.jsonl")
	}
}

// GetAnalyticsConfig returns a thread-safe copy of the Analytics config section.
func GetAnalyticsConfig() Analytics {
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.Analytics
}
