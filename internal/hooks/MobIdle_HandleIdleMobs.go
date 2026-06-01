package hooks

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/behaviortree"
	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
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
	// tell them to path back home.
	//
	// EXCEPTION: a goal planner that issued a movement command on the
	// immediately preceding idle round owns this mob's journey. The path
	// walker suppresses MobIdle while a path is active, so the previous idle
	// round (GetRoundCount()-1) is the most recent the planner could have
	// acted on; if it did, this is a mid-trip mob (e.g. MaxWander==0 guard
	// walking to a shop for upgrade-gear), not a combat-displaced one — don't
	// yank it home. Genuinely combat-displaced mobs have no recent planner
	// action and still get recovered.
	if !isCharmed && shouldRecoverDisplacedHome(mob) &&
		mobGoalActedRound(mob) != util.GetRoundCount()-1 {
		mob.Command("pathto home")
	}

	// Non-crafter merchant restock (supply cart delivery for regular shops).
	// Stage 2 caravan: vendors in caravan-served zones skip the per-mob
	// restock tick — they restock only when the caravan visits.
	if !configs.GetBalanceConfig().IsCaravanServedZone(mob.Zone) {
		if mobs.TickMobShopRestock(mob) {
			if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
				msgs := []string{
					`A supply cart pulls up outside. <ansi fg="mobname">%s</ansi> sorts through a fresh delivery.`,
					`<ansi fg="mobname">%s</ansi> unpacks a crate of supplies and restocks the shelves.`,
					`A runner drops off a bundle of goods. <ansi fg="mobname">%s</ansi> checks the contents and nods.`,
				}
				msg := fmt.Sprintf(msgs[util.Rand(len(msgs))], mob.Character.Name)
				sendVisualRoomText(room, messaging.CategoryMobIdle, msg)
			}
		}
	}

	// Stage 38.5.4: Crafter mob tick — background activity alongside normal idle
	if result := mobs.TickMobCraft(mob); result != nil {
		if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
			var msg string
			if result.Restocked && !result.Success && result.RecipeName == "" {
				// Restock-only tick — supply cart delivery, no craft.
				msgs := []string{
					`A supply cart pulls up outside. <ansi fg="mobname">%s</ansi> sorts through a fresh delivery of materials.`,
					`<ansi fg="mobname">%s</ansi> unpacks a crate of supplies and stacks them neatly behind the counter.`,
					`A runner drops off a bundle of materials. <ansi fg="mobname">%s</ansi> checks the contents and nods.`,
				}
				msg = fmt.Sprintf(msgs[util.Rand(len(msgs))], mob.Character.Name)
			} else if result.Success {
				msg = fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> finishes crafting and sets a new item on the shelf.`,
					mob.Character.Name)
			} else if result.RecipeName != "" {
				msg = fmt.Sprintf(
					`<ansi fg="mobname">%s</ansi> frowns at a failed attempt and discards the ruined materials.`,
					mob.Character.Name)
			}
			// Visual text — suppress in dark rooms except for nightvision
			if room.GetVisibility() < 1 {
				for _, uid := range room.GetPlayers() {
					u := users.GetByUserId(uid)
					if u != nil && u.Character.HasFlagFromAnySource(buffs.NightVision) {
						u.SendText(messaging.CategoryMobIdle, msg)
					}
				}
			} else {
				sendVisualRoomText(room, messaging.CategoryMobIdle, msg)
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

	// Floor-loot scan: wild non-charmed combat mobs pick up
	// gear upgrades they find on the room floor (chunk 2.3).
	if room := rooms.LoadRoom(mob.Character.RoomId); room != nil {
		EquipBestFloorItem(mob, room)
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

	// Behavior tree: handle idle before default behavior
	if behaviortree.TryMobBehavior(mob.InstanceId, behaviortree.EventContext{
		EventType: "mob_idle",
		RoomId:    mob.Character.RoomId,
	}) {
		return events.Continue
	}

	// Per-mob-tree mobs (and any mob whose btree lacks a try_goal_planner node)
	// don't run their goal planner via the tree. If this mob has a current goal
	// and the btree didn't already dispatch the planner this round, run it here
	// so named NPCs pursue goals too. Out-of-combat only (matches idle semantics).
	if !isCharmed && mob.Character.Aggro == nil &&
		mobGoalPlannerRanRound(mob) != util.GetRoundCount() {
		behaviortree.RunGoalPlanner(mob, util.GetRoundCount())
	}

	// "Planner owns the tick": TryMobBehavior returns false while a goal
	// planner is actively pursuing a goal (the planner returns Running, not
	// Success). If the planner ISSUED A COMMAND this very round (e.g. a
	// pathto-to-shop), that tick belongs to the planner — the legacy idle
	// block below (WanderCount home-pull + idle emote) must NOT also fire and
	// fight it. When the planner idled (returned no command), the stamp is
	// stale and legacy idle proceeds so the mob still emits flavor emotes.
	// Charmed mobs are excluded here so their first-aid path below is intact.
	if !isCharmed && mobGoalActedRound(mob) == util.GetRoundCount() {
		return events.Continue
	}

	{
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

// shouldRecoverDisplacedHome reports whether MobIdle's legacy "pathto home"
// displacement guard should fire for this mob. The guard exists to recover
// mobs with MaxWander==0 that got pulled out of their home room by combat,
// spells, or other forced movement.
//
// Mobs with their own movement authority (schedule executor or standalone
// patrol executor) opt out: their executors are the source of truth for
// "where this mob should be" at any given tick, and a redundant home-pull
// would fight the schedule/patrol target every other tick. Chunk 3.2's
// schedule executor force-sets MaxWander=0 every tick to suppress wander,
// which used to make the legacy guard ping-pong scheduled NPCs between
// their current segment target and their original placement room
// (Kerra in/out of tavern; Dal main↔back).
func shouldRecoverDisplacedHome(mob *mobs.Mob) bool {
	if mob == nil {
		return false
	}
	if mob.ScheduleId != "" || mob.PatrolId != "" {
		return false
	}
	return mob.MaxWander == 0 && mob.Character.RoomId != mob.HomeRoomId
}

// mobGoalActedRound returns the round on which this mob's goal planner last
// ISSUED a command (stamped by actGoalPlanner via the "goalActedRound"
// TempData key). Returns 0 if the stamp is absent or the wrong type. The
// idle handler compares this against the current round to tell when a goal
// planner owns the tick. Mirrors the lastGossipRound TempData read above.
func mobGoalActedRound(mob *mobs.Mob) uint64 {
	if mob == nil {
		return 0
	}
	if v := mob.GetTempData("goalActedRound"); v != nil {
		if r, ok := v.(uint64); ok {
			return r
		}
	}
	return 0
}

// mobGoalPlannerRanRound returns the round the goal planner last ran for this
// mob (set by behaviortree.RunGoalPlanner), 0 if absent/wrong type. Mirrors
// mobGoalActedRound. The idle handler uses this to avoid double-dispatching the
// planner for archetype mobs whose btree already ran try_goal_planner.
func mobGoalPlannerRanRound(mob *mobs.Mob) uint64 {
	if mob == nil {
		return 0
	}
	if v := mob.GetTempData("goalPlannerRanRound"); v != nil {
		if r, ok := v.(uint64); ok {
			return r
		}
	}
	return 0
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
	worldevents.PlayerDiedPvE:           "PlayerDiedPvE",
	worldevents.MobKilledByPlayer:       "MobKilledByPlayer",
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

	// Build a filter: show Local+ events for this mob's zone/region
	zone := mob.Character.Zone
	region := ""
	if zCfg := rooms.GetZoneConfig(zone); zCfg != nil {
		region = zCfg.Region
	}

	filter := &worldevents.WorldEventFilter{
		MinSignificance: worldevents.Local,
		ZoneName:        zone,
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

	// Filter out events this mob recently gossiped about (dedup via persistent facts substrate).
	var candidates []worldevents.WorldEvent
	for _, e := range evts {
		if facts.HeardEvent(int(mob.MobId), e.Id) {
			continue
		}
		candidates = append(candidates, e)
	}
	if len(candidates) == 0 {
		// All recent events already gossiped — reset by re-using full set
		candidates = evts
	}

	// Gather known facts as an additional candidate pool.
	factCandidates := facts.KnownFactsOf(int(mob.MobId))

	// Candidate-pool merging: prefer events 70%, facts 30%.
	// If only one pool is non-empty, use it exclusively.
	// If both are empty, fall through to event pick (candidates == evts at this point).
	useFactPool := false
	if len(factCandidates) > 0 && len(candidates) > 0 {
		useFactPool = util.Rand(100) < 30
	} else if len(factCandidates) > 0 {
		useFactPool = true
	}

	if useFactPool {
		kf := factCandidates[util.Rand(len(factCandidates))]
		line := renderFactGossip(kf)
		if line != "" {
			return line
		}
		// renderFactGossip returned empty (no templates); fall through to event path
	}

	// Pick a random event from deduplicated candidates
	evt := candidates[util.Rand(len(candidates))]

	// Record this event as heard via persistent facts substrate
	facts.RecordHeardEvent(int(mob.MobId), evt.Id)

	// Build the template key: "EventType-Significance"
	typeStr, ok := eventTypeKey[evt.Type]
	if !ok {
		typeStr = "Unknown"
	}
	sigStr, ok := significanceKey[evt.Significance]
	if !ok {
		sigStr = "Regional"
	}
	baseKey := typeStr + "-" + sigStr

	// Try distance-aware template: "Local" if same zone, "Distant" otherwise
	distance := "Distant"
	if evt.ZoneName == zone {
		distance = "Local"
	}

	templates, found := gossipTemplates[baseKey+"-"+distance]
	if !found || len(templates) == 0 {
		// Fall back to base key without distance suffix
		templates, found = gossipTemplates[baseKey]
	}
	if !found || len(templates) == 0 {
		// Try without significance
		for _, s := range []string{"Global", "Regional", "Local"} {
			if templates, found = gossipTemplates[typeStr+"-"+s]; found && len(templates) > 0 {
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

// renderFactGossip picks a template for a known fact and returns the rendered
// gossip line. Fallback chain: fact-{factId} → fact-{tag} → fact-default.
// Substitutes {description} placeholder with the fact's Description.
// Returns "" if no matching template is found.
func renderFactGossip(kf facts.KnownFact) string {
	if tmpls, ok := gossipTemplates["fact-"+kf.Fact.Id]; ok && len(tmpls) > 0 {
		return strings.ReplaceAll(tmpls[util.Rand(len(tmpls))], "{description}", kf.Fact.Description)
	}
	for _, tag := range kf.Fact.Tags {
		if tmpls, ok := gossipTemplates["fact-"+tag]; ok && len(tmpls) > 0 {
			return strings.ReplaceAll(tmpls[util.Rand(len(tmpls))], "{description}", kf.Fact.Description)
		}
	}
	if tmpls, ok := gossipTemplates["fact-default"]; ok && len(tmpls) > 0 {
		return strings.ReplaceAll(tmpls[util.Rand(len(tmpls))], "{description}", kf.Fact.Description)
	}
	return ""
}
