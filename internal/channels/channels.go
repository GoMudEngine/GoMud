// Package channels defines DOGMud's fixed set of global chat channels and the
// per-user toggle rules. It is deliberately dependency-free so the events,
// hooks, usercommands, and gmcp layers can all share it without import cycles.
package channels

// Channel is one global chat channel.
type Channel struct {
	Name      string // command + CommType, e.g. "newbie"
	ConfigKey string // per-user config-option key, e.g. "channel.newbie"
	Prefix    string // display prefix, e.g. "(newbie)"
	Color     string // ansi fg name for the prefix
}

var registry = []Channel{
	{Name: "chat", ConfigKey: "channel.chat", Prefix: "(chat)", Color: "cyan"},
	{Name: "newbie", ConfigKey: "channel.newbie", Prefix: "(newbie)", Color: "green"},
	{Name: "trade", ConfigKey: "channel.trade", Prefix: "(trade)", Color: "yellow"},
}

// All returns the channels in display order.
func All() []Channel { return registry }

// Get resolves a channel by name.
func Get(name string) (Channel, bool) {
	for _, c := range registry {
		if c.Name == name {
			return c, true
		}
	}
	return Channel{}, false
}

// Enabled applies the default-on rule: a channel is off only when the stored
// config value is explicitly the boolean false. nil (unset), true, or any
// non-bool all mean on.
func Enabled(cfgValue any) bool {
	if b, ok := cfgValue.(bool); ok {
		return b
	}
	return true
}

// ShouldReceive decides whether a user gets a channel message. The sender always
// sees their own echo; everyone else must not be deafened and must have the
// channel enabled.
func ShouldReceive(isSender, deafened bool, cfgValue any) bool {
	if isSender {
		return true
	}
	return !deafened && Enabled(cfgValue)
}
