package configs

import (
	"strings"
)

type TextFormats struct {
	Prompt                  ConfigString `yaml:"Prompt"`                  // The in-game status prompt style
	EnterRoomMessageWrapper ConfigString `yaml:"EnterRoomMessageWrapper"` // Special enter messages
	ExitRoomMessageWrapper  ConfigString `yaml:"ExitRoomMessageWrapper"`  // Special exit messages
	Time                    ConfigString `yaml:"Time"`                    // How to format time when displaying real time
	TimeShort               ConfigString `yaml:"TimeShort"`               // How to format time when displaying real time (shortform)
}

func (m *TextFormats) Validate() {

	if m.Prompt == `` {
		m.Prompt = `{8}[{t} {255}HP:{hp}{8}/{HP} {255}SP:{st}{8}/{ST} {255}CP:{13}{cv}{8}/{13}{CV}{8}]{239}{h}{8}:`
	}

	// Must have a message wrapper...
	if m.EnterRoomMessageWrapper == `` {
		m.EnterRoomMessageWrapper = `%s` // default
	}
	if strings.LastIndex(string(m.EnterRoomMessageWrapper), `%s`) < 0 {
		m.EnterRoomMessageWrapper += `%s` // default
	}

	// Must have a message wrapper...
	if m.ExitRoomMessageWrapper == `` {
		m.ExitRoomMessageWrapper = `%s` // default
	}
	if strings.LastIndex(string(m.ExitRoomMessageWrapper), `%s`) < 0 {
		m.ExitRoomMessageWrapper += `%s` // default
	}

	if m.Time == `` {
		m.Time = `Monday, 02-Jan-2006 03:04:05PM`
	}

	if m.TimeShort == `` {
		m.TimeShort = `Jan 2 '06 3:04PM`
	}

}

func GetTextFormatsConfig() TextFormats {
	configDataLock.RLock()
	defer configDataLock.RUnlock()

	if !configData.validated {
		configData.Validate()
	}
	return configData.TextFormats
}
