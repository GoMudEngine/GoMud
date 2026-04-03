package actions

import (
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// userOnlyCommands lists user commands that intentionally have no mob equivalent,
// along with the reason category. Any user command absent from the mob registry
// but present in this map is expected and will not trigger a parity warning.
var userOnlyCommands = map[string]string{
	// --- Admin-only commands ---
	"ai-flag":     "admin",
	"ai-list":     "admin",
	"badcommands": "admin",
	"buff":        "admin",
	"build":       "admin",
	"command":     "admin",
	"combatstats": "admin",
	"deafen":      "admin",
	"devtool":     "admin",
	"item":        "admin",
	"locate":      "admin",
	"mob":         "admin",
	"modify":      "admin",
	"mudmail":     "admin",
	"mute":        "admin",
	"paz":         "admin",
	"prepare":     "admin",
	"questdebug":  "admin",
	"questtoken":  "admin",
	"redescribe":  "admin",
	"reload":      "admin",
	"rename":      "admin",
	"room":        "admin",
	"server":      "admin",
	"setmotd":     "admin",
	"skillset":    "admin",
	"spawn":       "admin",
	"spell":       "admin",
	"syslogs":     "admin",
	"teleport":    "admin",
	"undeafen":    "admin",
	"unmute":      "admin",
	"zap":         "admin",
	"zone":        "admin",

	// --- UI / display / config commands (no game action) ---
	"afk":        "ui",
	"alias":      "ui",
	"bank":       "ui",
	"biome":      "ui",
	"bug":        "ui",
	"cancel":     "ui",
	"character":  "ui",
	"conditions": "ui",
	"consider":   "ui",
	"cooldowns":  "ui",
	"default":    "ui",
	"help":       "ui",
	"hint":       "ui",
	"history":    "ui",
	"inbox":      "ui",
	"inventory":  "ui",
	"keyring":    "ui",
	"killstats":  "ui",
	"macros":     "ui",
	"map":        "ui",
	"motd":       "ui",
	"mutations":  "ui",
	"online":     "ui",
	"password":   "ui",
	"print":      "ui",
	"printline":  "ui",
	"pvp":        "ui",
	"quests":     "ui",
	"quit":       "ui",
	"read":       "ui",
	"rep":        "ui",
	"report":     "ui",
	"save":       "ui",
	"set":        "ui",
	"setdesc":    "ui",
	"skills":     "ui",
	"spells":     "ui",
	"status":     "ui",
	"suggest":    "ui",
	"title":      "ui",
	"who":        "ui",

	// --- Player-only mechanics ---
	"assist":   "player-mechanic",
	"party":    "player-mechanic",
	"reply":    "player-mechanic",
	"share":    "player-mechanic",
	"start":    "player-mechanic",
	"target":   "player-mechanic",
	"whisper":  "player-mechanic",
	"zombieact": "player-mechanic",

	// --- Aliases (remapped to another command, no separate mob equivalent needed) ---
	"knee":      "alias",
	"stomp":     "alias",
	"tailsweep": "alias",

	// --- Shared actions (implemented on both user and mob sides) ---
	// These are intentionally NOT in either allowlist because they should exist on both.
	// If they appear in any allowlist, it's a sign they were miscategorized.
	// craft: shared crafting action (substage 5)
	// sneak: shared stealth action (substage 5)

}

// mobOnlyCommands lists mob commands that intentionally have no user equivalent.
//
// Future work candidates (not mob-only forever):
//   - howl: functionally a renamed taunt — should use shared taunt action
//   - roar: candidate for player shout/intimidation system
//   - throw: planned for player ranged attacks
//   - alchemy: legacy mob brewing — consolidate with shared craft system
//   - backstab: redundant with surprise strike system — candidate for removal
var mobOnlyCommands = map[string]string{
	// --- Mob AI behaviours ---
	"aid":            "mob-ai",
	"backstab":       "mob-ai: redundant with surprise strike, remove later",
	"befriend":       "mob-ai",
	"callforhelp":    "mob-ai",
	"charge":         "mob-ai",
	"consume":        "mob-ai",
	"converse":       "mob-ai",
	"despawn":        "mob-ai",
	"givequest":      "mob-ai",
	"hamstring":      "mob-ai",
	"howl":           "mob-ai: renamed taunt, unify with taunt system",
	"lookforaid":     "mob-ai",
	"lookfortrouble": "mob-ai",
	"pathto":         "mob-ai",
	"portal":         "mob-ai",
	"replyto":        "mob-ai",
	"roar":           "mob-ai: future player shout/intimidation",
	"sayto":          "mob-ai",
	"saytoonly":      "mob-ai",
	"throw":          "mob-ai: future player ranged ability",
	"wander":         "mob-ai",

	// --- Legacy mob alchemy (consolidate with shared craft) ---
	"alchemy": "mob-alchemy: consolidate with shared craft system",
}

// AuditCommandParity compares the user and mob command registries and logs a
// warning for every unexpected gap — i.e. a command present on one side but
// absent on the other that is not listed in the intentional allowlists above.
//
// Call this once at server startup after both registries are fully populated.
func AuditCommandParity(userCmds []string, mobCmds []string) {

	userSet := make(map[string]struct{}, len(userCmds))
	for _, c := range userCmds {
		userSet[c] = struct{}{}
	}

	mobSet := make(map[string]struct{}, len(mobCmds))
	for _, c := range mobCmds {
		mobSet[c] = struct{}{}
	}

	warnings := 0

	// Check every user command against the mob registry.
	for _, cmd := range userCmds {
		if _, inMob := mobSet[cmd]; inMob {
			continue // present on both sides — fine
		}
		if _, allowed := userOnlyCommands[cmd]; allowed {
			continue // intentional user-only divergence
		}
		mudlog.Warn("CommandParity",
			"msg", fmt.Sprintf("user command %q has no mob equivalent and is not in the user-only allowlist", cmd))
		warnings++
	}

	// Check every mob command against the user registry.
	for _, cmd := range mobCmds {
		if _, inUser := userSet[cmd]; inUser {
			continue // present on both sides — fine
		}
		if _, allowed := mobOnlyCommands[cmd]; allowed {
			continue // intentional mob-only divergence
		}
		mudlog.Warn("CommandParity",
			"msg", fmt.Sprintf("mob command %q has no user equivalent and is not in the mob-only allowlist", cmd))
		warnings++
	}

	if warnings == 0 {
		mudlog.Info("CommandParity",
			"msg", fmt.Sprintf("registries balanced — %d user commands, %d mob commands, no unexpected gaps",
				len(userCmds), len(mobCmds)))
	}
}
