package hooks

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/conversations"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobcommands"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/scripting"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
	"gopkg.in/yaml.v2"
)

//
// Handles default mob idle behavior
//

func HandleIdleMobs(e events.Event) events.ListenerReturn {

	evt := e.(events.MobIdle)

	mob := mobs.GetInstance(evt.MobInstanceId)
	if mob == nil {
		return events.Cancel
	}

	isCharmed := mob.Character.IsCharmed()

	// if a mob shouldn't be allowed to leave their area (via wandering)
	// but has somehow been displaced, such as pulling through combat, spells, or otherwise
	// tell them to path back home
	if mob.MaxWander == 0 && mob.Character.RoomId != mob.HomeRoomId {
		if !isCharmed {
			mob.Command("pathto home")
		}
	}

	if conversations.HasConverseFile(int(mob.MobId), mob.Character.Zone) && util.Rand(100) < int(configs.GetGamePlayConfig().MobConverseChance) {
		if mobRoom := rooms.LoadRoom(mob.Character.RoomId); mobRoom != nil {
			mobcommands.Converse(``, mob, mobRoom) // Execute this directly so that target mob doesn't leave the room before this command executes
		}
	}

	// Stage 38.5.4: Crafter mob tick — background activity alongside normal idle
	if result := mobs.TickMobCraft(mob); result != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			if result.Success {
				room.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> finishes crafting and sets a new item on the shelf.`,
					mob.Character.Name))
			} else {
				room.SendText(fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> frowns at a failed attempt and discards the ruined materials.`,
					mob.Character.Name))
			}
		}
		// Emit world event for rare crafts
		if result.Success {
			b := configs.GetBalanceConfig()
			rareThreshold := int(b.CrafterRareThreshold)
			if result.SkillMinimum >= rareThreshold {
				sig := worldevents.Regional
				if result.SkillMinimum >= rareThreshold*2 {
					sig = worldevents.Global
				}
				zone := result.Zone
				region := ""
				if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
					region = zCfg.Region
				}
				worldevents.EmitWorldEvent(worldevents.WorldEvent{
					Type:         worldevents.MobCraftedRare,
					Significance: sig,
					ZoneName:     zone,
					RegionName:   region,
					MobName:      result.MobName,
					Description: fmt.Sprintf("%s has crafted a rare %s.",
						result.MobName, result.RecipeName),
				})
			}
		}
	}

	// Stage 42.5: Gossiper mob tick — broadcast world event gossip
	if mobHasGroup(mob, "gossiper") {
		gossipIntervalRounds := uint64(configs.GetBalanceConfig().GossipIntervalRounds)
		// Stagger by MobId so patrons don't all talk at the same time
		stagger := uint64(mob.MobId%3) * (gossipIntervalRounds / 3)
		roundNow := util.GetRoundCount()

		// Check if enough rounds have passed since last gossip
		lastGossip := uint64(0)
		if v := mob.GetTempData("lastGossipRound"); v != nil {
			lastGossip, _ = v.(uint64)
		}

		if roundNow >= lastGossip+gossipIntervalRounds+stagger || lastGossip == 0 {
			line := buildGossipLine(mob)
			if line != "" {
				mob.Command("say " + line)
				mob.SetTempData("lastGossipRound", roundNow)
			}
		}
	}

	// If they have idle commands, maybe do one of them?
	handled, _ := scripting.TryMobScriptEvent("onIdle", mob.InstanceId, 0, ``, nil)
	if !handled {

		if isCharmed {
			// Only some mobs can apply first aid
			// If a charmed mob can aid someone, try.
			if mob.Character.KnowsFirstAid() {
				mob.Command(`lookforaid`)
			}

			return events.Continue
		}

		if mob.MaxWander > -1 && mob.WanderCount > mob.MaxWander {
			// Not charmed and far from home, and should never leave home.
			// So go home.
			mob.Command(`pathto home`)
			return events.Continue
		}

		//
		// Look for trouble
		//
		idleCmd := `lookfortrouble`
		if util.Rand(100) < mob.ActivityLevel {
			idleCmd = mob.GetIdleCommand()
			if idleCmd == `` {
				idleCmd = `lookfortrouble`
			}
		}
		mob.Command(idleCmd)

	}

	return events.Continue
}

// ── Gossiper helpers ─────────────────────────────────────────────────────────

var (
	gossipTemplates     map[string][]string
	gossipTemplatesOnce sync.Once
)

// eventTypeKey maps WorldEventType to the string prefix used in gossip_templates.yaml.
var eventTypeKey = map[worldevents.WorldEventType]string{
	worldevents.MobStatMilestone:        "MobStatMilestone",
	worldevents.MobMutationGained:       "MobMutationGained",
	worldevents.MobMutationAdvanced:     "MobMutationAdvanced",
	worldevents.MobCraftedRare:          "MobCraftedRare",
	worldevents.PackStrengthened:        "PackStrengthened",
	worldevents.PlayerMutationMilestone: "PlayerMutationMilestone",
	worldevents.PlayerCraftedRare:       "PlayerCraftedRare",
}

var significanceKey = map[worldevents.Significance]string{
	worldevents.Local:    "Local",
	worldevents.Regional: "Regional",
	worldevents.Global:   "Global",
}

func loadGossipTemplates() {
	path := string(configs.GetFilePathsConfig().DataFiles) + `/gossip_templates.yaml`

	data, err := os.ReadFile(path)
	if err != nil {
		mudlog.Error("loadGossipTemplates", "error", "failed to load gossip_templates.yaml", "path", path, "err", err)
		gossipTemplates = map[string][]string{}
		return
	}

	templates := map[string][]string{}
	if err := yaml.Unmarshal(data, &templates); err != nil {
		mudlog.Error("loadGossipTemplates", "error", "failed to parse gossip_templates.yaml", "err", err)
		gossipTemplates = map[string][]string{}
		return
	}

	gossipTemplates = templates
	mudlog.Info("...loadGossipTemplates()", "loadedKeys", len(gossipTemplates))
}

func mobHasGroup(mob *mobs.Mob, groupName string) bool {
	for _, g := range mob.Groups {
		if g == groupName {
			return true
		}
	}
	return false
}

func buildGossipLine(mob *mobs.Mob) string {
	gossipTemplatesOnce.Do(loadGossipTemplates)

	// Build a filter: show Regional+ events for this mob's region
	zone := mob.Character.Zone
	region := ""
	if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
		region = zCfg.Region
	}

	filter := &worldevents.WorldEventFilter{
		MinSignificance: worldevents.Regional,
		RegionName:      region,
	}

	evts := worldevents.GetRecentWorldEvents(10, filter)

	if len(evts) == 0 {
		// Use fallback templates
		if fallbacks, ok := gossipTemplates["fallback"]; ok && len(fallbacks) > 0 {
			return fallbacks[util.Rand(len(fallbacks))]
		}
		return ""
	}

	// Pick a random recent event
	evt := evts[util.Rand(len(evts))]

	// Build the template key: "EventType-Significance"
	typeStr, ok := eventTypeKey[evt.Type]
	if !ok {
		typeStr = "Unknown"
	}
	sigStr, ok := significanceKey[evt.Significance]
	if !ok {
		sigStr = "Regional"
	}
	key := typeStr + "-" + sigStr

	templates, ok := gossipTemplates[key]
	if !ok || len(templates) == 0 {
		// Try without significance
		for _, s := range []string{"Global", "Regional", "Local"} {
			if templates, ok = gossipTemplates[typeStr+"-"+s]; ok && len(templates) > 0 {
				break
			}
		}
	}

	if len(templates) == 0 {
		// Final fallback: just say the description
		return fmt.Sprintf("I heard that %s", evt.Description)
	}

	tmpl := templates[util.Rand(len(templates))]
	return strings.Replace(tmpl, "{desc}", evt.Description, 1)
}
