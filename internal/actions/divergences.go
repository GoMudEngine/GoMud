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
	"ai-flag":      "admin",
	"ai-list":      "admin",
	"badcommands":  "admin",
	"bounty":       "admin",
	"buff":         "admin",
	"build":        "admin",
	"caravan":      "admin",
	"command":      "admin",
	"combatstats":  "admin",
	"copyover":     "admin", // live server restart (hot-reboot)
	"crime":        "admin",
	"deafen":       "admin",
	"devtool":      "admin",
	"fact":         "admin",
	"faction":      "admin",
	"goal":         "admin", // admin.goal.go: inspect/seed mob goals (Phase 4)
	"item":         "admin",
	"knowledge":    "admin",
	"locate":       "admin",
	"mob":          "admin",
	"modify":       "admin",
	"mudmail":      "admin",
	"mute":         "admin",
	"opinion":      "admin",
	"paz":          "admin",
	"prepare":      "admin",
	"questdebug":   "admin",
	"questtoken":   "admin",
	"redescribe":   "admin",
	"relationship": "admin",
	"reload":       "admin",
	"renameitem":   "admin",
	"room":         "admin",
	"server":       "admin",
	"setmotd":      "admin",
	"skillset":     "admin",
	"spawn":        "admin",
	"spell":        "admin",
	"syslogs":      "admin",
	"teleport":     "admin",
	"undeafen":     "admin",
	"unmute":       "admin",
	"zap":          "admin",
	"zone":         "admin",

	// --- UI / display / config commands (no game action) ---
	// Includes commands registered by modules (auction, checkclient, discord,
	// leaderboard, mudletmap, mudletui, time) that are injected after the
	// static userCommands map is built.
	"afk":             "ui",
	"alias":           "ui",
	"appraise":        "ui",
	"auction":         "ui", // module: auctions
	"bank":            "ui",
	"achievements":    "ui",
	"biome":           "ui",
	"bug":             "ui",
	"cancel":          "ui",
	"character":       "ui",
	"checkclient":     "ui", // module: gmcp/mudlet
	"companion":       "ui",
	"companions":      "ui",
	"conditions":      "ui",
	"consider":        "ui",
	"cooldowns":       "ui",
	"default":         "ui",
	"deletecharacter": "ui",
	"discord":         "ui", // module: gmcp/mudlet
	"dismiss":         "ui",
	"help":            "ui",
	"hint":            "ui",
	"history":         "ui",
	"inbox":           "ui",
	"inventory":       "ui",
	"keyring":         "ui",
	"killstats":       "ui",
	"leaderboard":     "ui", // module: leaderboards
	"list":            "ui",
	"macros":          "ui",
	"map":             "ui",
	"motd":            "ui",
	"mudletmap":       "ui", // module: gmcp/mudlet
	"mudletui":        "ui", // module: gmcp/mudlet
	"mutations":       "ui",
	"online":          "ui",
	"password":        "ui",
	"pet":             "ui",
	"print":           "ui",
	"printline":       "ui",
	"pvp":             "ui",
	"quests":          "ui",
	"quit":            "ui",
	"read":            "ui",
	"rename":          "ui",
	"rep":             "ui",
	"report":          "ui",
	"save":            "ui",
	"set":             "ui",
	"setdesc":         "ui",
	"sethome":         "ui",
	"skills":          "ui",
	"spells":          "ui",
	"status":          "ui",
	"storage":         "ui",
	"suggest":         "ui",
	"time":            "ui", // module: time
	"title":           "ui",
	"who":             "ui",

	// --- Player-only mechanics ---
	"ask":        "player-mechanic",
	"assess":     "player-mechanic",
	"assist":     "player-mechanic",
	"disenchant": "player-mechanic",
	"fine":       "player-mechanic: jailed-player justice interaction (5.1)",
	"offer":      "player-mechanic",
	"payfine":    "player-mechanic: jailed-player justice interaction (5.1)",
	"party":      "player-mechanic",
	"picklock":   "player-mechanic: wontfix per chunk 2.10 deferred-gaps review — interactive minigame is intentional player-only design",
	"reply":      "player-mechanic",
	"sell":       "player-mechanic",
	"share":      "player-mechanic",
	"sort":       "player-mechanic",
	"stand":      "player-mechanic",
	"start":      "player-mechanic",
	"stash":      "player-mechanic",
	"talk":       "player-mechanic",
	"target":     "player-mechanic",
	"use":        "player-mechanic",
	"whisper":    "player-mechanic",
	"zombieact":  "player-mechanic",

	// --- Intentionally NOT allowlisted ---
	// "throw" is deliberately omitted so the CommandParity boot audit keeps
	// surfacing it as the single standing reminder that the ranged-weapon
	// system (and a mob throw equivalent) is still owed. See
	// [[project_throwable_mobs_ranged_dependency]]. Do not allowlist it until
	// that system lands.

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
//   - hamstring: shared action (ExecuteHamstring), future player species-gated ability
var mobOnlyCommands = map[string]string{
	// --- Mob AI behaviours ---
	"aid":            "mob-ai",
	"befriend":       "mob-ai",
	"callforhelp":    "mob-ai",
	"charge":         "mob-ai",
	"consume":        "mob-ai",
	"converse":       "mob-ai",
	"despawn":        "mob-ai",
	"givequest":      "mob-ai",
	"hamstring":      "mob-ai: shared ExecuteHamstring action, future player ability",
	"howl":           "mob-ai: shared ExecuteTaunt action with mob flavor text",
	"lookforaid":     "mob-ai",
	"lookfortrouble": "mob-ai",
	"pathto":         "mob-ai",
	"portal":         "mob-ai",
	"replyto":        "mob-ai",
	"sayto":          "mob-ai",
	"saytoonly":      "mob-ai",
	"wander":         "mob-ai",
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
