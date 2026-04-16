package configs

type Logging struct {
	LogToFile   ConfigBool   `yaml:"LogToFile"`   // Write logs to a file in addition to terminal
	LogFilePath ConfigString `yaml:"LogFilePath"` // Path for the log file (default _datafiles/logs/server.log)
	LogLevel    ConfigString `yaml:"LogLevel"`    // debug, info, warn, error (default debug)
}

func (l *Logging) Validate() {
	if l.LogFilePath == "" {
		l.LogFilePath = "_datafiles/logs/server.log"
	}
	if l.LogLevel == "" {
		l.LogLevel = "debug"
	}
}

func GetLoggingConfig() Logging {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.Logging
}
