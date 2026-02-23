package configs

// LLM holds configuration for the optional local LLM integration (Ollama).
type LLM struct {
	Enabled  ConfigBool   `yaml:"Enabled"`  // master on/off switch
	Endpoint ConfigString `yaml:"Endpoint"` // e.g. "http://localhost:11434"
	Timeout  ConfigInt    `yaml:"Timeout"`  // seconds before giving up; default 10
}

func (l *LLM) Validate() {
	if int(l.Timeout) <= 0 {
		l.Timeout.Set("10")
	}
	if string(l.Endpoint) == "" {
		l.Endpoint.Set("http://localhost:11434")
	}
}

// GetLLMConfig returns a thread-safe copy of the LLM config section.
func GetLLMConfig() LLM {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.LLM
}
