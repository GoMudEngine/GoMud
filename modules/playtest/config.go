package playtest

import "github.com/GoMudEngine/GoMud/internal/plugins"

// Config is the resolved module configuration. There are no account fields —
// the module does not provision or own an account; the AI agent logs in or
// creates a character via the normal new-player flow.
type Config struct {
	Enabled         bool
	SafeMode        bool
	SandboxZoneTag  string
	DeathProtection bool
	Beacons         bool
}

// getter abstracts plug.Config.Get for testability.
type getter func(string) any

func asString(v any) string { s, _ := v.(string); return s }
func asBool(v any) bool     { b, _ := v.(bool); return b }

// buildConfig resolves config from a getter. Defaults for unset keys come from
// the module's data-overlays/config.yaml, so a nil getter yields zero values.
func buildConfig(get getter) Config {
	return Config{
		Enabled:         asBool(get("Enabled")),
		SafeMode:        asBool(get("SafeMode")),
		SandboxZoneTag:  asString(get("SandboxZoneTag")),
		DeathProtection: asBool(get("DeathProtection")),
		Beacons:         asBool(get("Beacons")),
	}
}

// loadConfig reads the module's live config via the plugin API.
func loadConfig(p *plugins.Plugin) Config {
	return buildConfig(func(k string) any { return p.Config.Get(k) })
}
