package configs

import (
	"strings"
)

type TextFormats struct {
	Prompt                  ConfigString `yaml:"Prompt"`                  // The in-game status prompt style
	FightPrompt             ConfigString `yaml:"FightPrompt"`             // The in-game fight prompt style (shown during combat)
	EnterRoomMessageWrapper ConfigString `yaml:"EnterRoomMessageWrapper"` // Special enter messages
	ExitRoomMessageWrapper  ConfigString `yaml:"ExitRoomMessageWrapper"`  // Special exit messages
	Time                    ConfigString `yaml:"Time"`                    // How to format time when displaying real time
	TimeShort               ConfigString `yaml:"TimeShort"`               // How to format time when displaying real time (shortform)
}

func (m *TextFormats) Validate() {

	if m.Prompt == `` {
		m.Prompt = `{8}[{t} {255}HP:{hpbar} {255}SP:{stbar} {255}CP:{cvbar}{8}]{239}{h}{8}:`
	}

	if m.FightPrompt == `` {
		m.FightPrompt = `{8}[{t} {255}HP:{hpbar} SP:{stbar} CP:{cvbar}{pos}{8}] {255}» {target}{8}[{targetpos}{8}|{targethealth}{8}] {255}{tank}{tankpos}{tankbar}{239}{h}{8}:`
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
	ensureConfigValidated()

	configDataLock.RLock()
	defer configDataLock.RUnlock()
	return configData.TextFormats
}
