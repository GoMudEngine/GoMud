package behaviortree

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/exit"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// ActionFunc is the signature for all registered action implementations.
type ActionFunc func(params map[string]any, ctx *EvalContext) Result

// actionRegistry maps action names to their implementations.
var actionRegistry = map[string]ActionFunc{}

func init() {
	actionRegistry["respond"] = actRespond
	actionRegistry["say"] = actSay
	actionRegistry["emote"] = actEmote
	actionRegistry["grant_quest"] = actGrantQuest
	actionRegistry["set_quest_flag"] = actSetQuestFlag
	actionRegistry["give_item"] = actGiveItem
	actionRegistry["return_item"] = actReturnItem
	actionRegistry["take_item"] = actTakeItem
	actionRegistry["give_gold"] = actGiveGold
	actionRegistry["take_gold"] = actTakeGold
	actionRegistry["move"] = actMove
	actionRegistry["attack"] = actAttack
	actionRegistry["flee"] = actFlee
	actionRegistry["cast"] = actCast
	actionRegistry["spawn_mob"] = actSpawnMob
	actionRegistry["add_temp_exit"] = actAddTempExit
	actionRegistry["set_state"] = actSetState
	actionRegistry["command"] = actCommand

	// New actions for boss mob / quest NPC behavior
	actionRegistry["summon_companion"] = actSummonCompanion
	actionRegistry["set_room_locked"] = actSetRoomLocked
	actionRegistry["spawn_item_in_room"] = actSpawnItemInRoom
	actionRegistry["add_buff"] = actAddBuff
	actionRegistry["command_mob"] = actCommandMob
	actionRegistry["give_item_multiple"] = actGiveItemMultiple
	actionRegistry["set_misc_data"] = actSetMiscData
	actionRegistry["increment_state"] = actIncrementState
	actionRegistry["decrement_state"] = actDecrementState
	actionRegistry["grant_quest_to_user"] = actGrantQuest // alias for grant_quest

	// New actions for room behavior trees
	actionRegistry["mob_say"] = actMobSay
	actionRegistry["mob_emote"] = actMobEmote
	actionRegistry["grant_mutation"] = actGrantMutation
	actionRegistry["send_user_text"] = actSendUserText
	actionRegistry["send_room_text"] = actSendRoomText
	actionRegistry["intercept"] = actIntercept
	actionRegistry["remove_buff"] = actRemoveBuff
	actionRegistry["move_player"] = actMovePlayer
	actionRegistry["create_instance"] = actCreateInstance
	actionRegistry["open_instance_portal"] = actOpenInstancePortal
}

// LookupAction returns the action function for the given name,
// or nil if not found.
func LookupAction(name string) ActionFunc {
	return actionRegistry[name]
}

// ActionNode wraps a registered action function with its params.
type ActionNode struct {
	Name   string
	Params map[string]any
	Fn     ActionFunc
}

// delayedActions is the set of action names that are subject to
// perception-scaled reaction delays. Internal bookkeeping actions
// (state, quest, item) remain instant.
var delayedActions = map[string]bool{
	"respond":     true,
	"say":         true,
	"emote":       true,
	"attack":      true,
	"flee":        true,
	"cast":        true,
	"move":        true,
	"add_buff":    true,
	"command_mob": true,
	"mob_say":     true,
	"mob_emote":   true,
}

func (n *ActionNode) Evaluate(ctx *EvalContext) Result {
	// Static delay check first — bypasses perception-scaled timing.
	if staticDelay, ok := n.Params["delay"]; ok {
		var delaySec float64
		switch v := staticDelay.(type) {
		case float64:
			delaySec = v
		case int:
			delaySec = float64(v)
		}
		if delaySec > 0 {
			params := n.Params
			fn := n.Fn
			evalCtx := &EvalContext{
				Event:       ctx.Event,
				MobState:    ctx.MobState,
				MobId:       ctx.MobId,
				InstanceId:  ctx.InstanceId,
				RoomId:      ctx.RoomId,
				MobName:     ctx.MobName,
				Intercepted: ctx.Intercepted,
			}
			dur := time.Duration(delaySec * float64(time.Second))
			GetEngine().QueueDelayed(dur, func() {
				fn(params, evalCtx)
			})
			return Success
		}
	}
	if delayedActions[n.Name] {
		delay := calcReactionDelay(ctx.InstanceId)
		if delay > 0 {
			params := n.Params
			fn := n.Fn
			evalCtx := &EvalContext{
				Event:       ctx.Event,
				MobState:    ctx.MobState,
				MobId:       ctx.MobId,
				InstanceId:  ctx.InstanceId,
				RoomId:      ctx.RoomId,
				MobName:     ctx.MobName,
				Intercepted: ctx.Intercepted,
			}
			GetEngine().QueueDelayed(delay, func() {
				fn(params, evalCtx)
			})
			return Success
		}
	}
	return n.Fn(n.Params, ctx)
}

// --- action implementations ---

func actMove(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	direction := getStringParam(params, "direction")
	if direction == "" {
		return Failure
	}
	mob.Command("go " + direction)
	return Success
}

func actSpawnMob(params map[string]any, ctx *EvalContext) Result {
	mobId := getIntParam(params, "mob_id")
	if mobId == 0 {
		return Failure
	}
	roomId := getIntParam(params, "room_id")
	if roomId == 0 {
		roomId = ctx.RoomId
	}
	spawned := mobs.NewMobById(mobs.MobId(mobId), roomId)
	if spawned == nil {
		return Failure
	}
	return Success
}

func actCommand(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	cmd := getStringParam(params, "cmd")
	if cmd == "" {
		return Failure
	}
	mob.Command(cmd)
	return Success
}

// actSummonCompanion spawns a mob as either a hostile ally (attacks the
// player) or a charmed companion of the summoning mob.
//
// params: mob_id (int), count (int, default 1), base_pool (int, default 50),
//         hostile (string "true"/"false", default "false")
//
// When hostile=true: spawned mobs aggro the triggering player immediately.
// When hostile=false: spawned mobs are charmed to the summoning mob (companion).
func actSummonCompanion(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	mobId := getIntParam(params, "mob_id")
	if mobId == 0 {
		return Failure
	}
	count := getIntParam(params, "count")
	if count <= 0 {
		count = 1
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	manifestSkill := mob.Character.GetSkillLevel(skills.Manifestation)
	charisma := mob.Character.Stats.Charisma.ValueAdj
	basePool := getIntParam(params, "base_pool")
	if basePool <= 0 {
		basePool = 50
	}
	hostile := getStringParam(params, "hostile") == "true"
	pool := characters.CalcCompanionStatPool(basePool, charisma, manifestSkill)
	for i := 0; i < count; i++ {
		companion := mobs.NewMobById(mobs.MobId(mobId), room.RoomId, pool)
		if companion == nil {
			continue
		}
		room.AddMob(companion.InstanceId)
		if hostile {
			// Hostile ally: aggro the triggering player and engage
			if ctx.Event.UserId > 0 {
				companion.Character.SetAggro(ctx.Event.UserId, 0, characters.DefaultAttack)
				companion.Command(`lookfortrouble`, 4)
			}
		} else {
			// Charmed companion of the summoning mob
			companion.Character.Charm(0, 99999, "")
			companion.Character.EndAggro()
			info := characters.CompanionInfo{
				MobId:      int(companion.MobId),
				InstanceId: companion.InstanceId,
				SourceType: characters.CompanionSummoned,
				Name:       companion.Character.Name,
				BaseName:   companion.Character.Name,
				AutoAssist: true,
			}
			mob.Character.AddCompanion(info)
		}
	}
	return Success
}

// actCommandMob finds the first mob in the room with the given mob_id and
// issues it a command string. Returns Failure if no matching mob is found.
// params: mob_id (int), cmd (string)
func actCommandMob(params map[string]any, ctx *EvalContext) Result {
	targetMobId := getIntParam(params, "mob_id")
	if targetMobId == 0 {
		return Failure
	}
	cmd := getStringParam(params, "cmd")
	if cmd == "" {
		return Failure
	}
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		return Failure
	}
	for _, instId := range room.GetMobs(rooms.FindAll) {
		m := mobs.GetInstance(instId)
		if m != nil && int(m.MobId) == targetMobId {
			m.Command(cmd)
			return Success
		}
	}
	return Failure
}

// actCreateInstance clones an instanced zone and stores the resulting entry
// room ID in BehaviorState under the key "instance_entry_room_id". The
// caller should follow up with add_temp_exit using that state value.
//
// params:
//   zone_name   (string) — zone template name (e.g. "Instance Arena")
//   gold_amount (int)    — gold investment; must be >= 100
//   state_key   (string, default "instance_entry_room_id") — key to store result
//
// Precondition: take_gold should be called before this action. If instance
// creation fails, Returns Failure (caller should refund gold via give_gold).
func actCreateInstance(params map[string]any, ctx *EvalContext) Result {
	if ctx.MobState == nil {
		return Failure
	}
	zoneName := getStringParam(params, "zone_name")
	if zoneName == "" {
		return Failure
	}
	goldAmount := getIntParam(params, "gold_amount")
	if goldAmount <= 0 {
		return Failure
	}
	userId := ctx.Event.UserId
	if userId == 0 {
		return Failure
	}

	// Build authorized user list: owner + any party members.
	authorizedUsers := []int{userId}
	if party := parties.Get(userId); party != nil {
		for _, memberId := range party.GetMembers() {
			if memberId != userId {
				authorizedUsers = append(authorizedUsers, memberId)
			}
		}
	}

	inst, err := rooms.CreateZoneInstance(zoneName, goldAmount, userId, authorizedUsers, ctx.RoomId)
	if err != nil {
		mudlog.Error("actCreateInstance", "zone", zoneName, "userId", userId, "error", err)
		return Failure
	}

	stateKey := getStringParam(params, "state_key")
	if stateKey == "" {
		stateKey = "instance_entry_room_id"
	}
	ctx.MobState.Set(stateKey, inst.EntryRoomId)
	return Success
}

// actOpenInstancePortal is a compound action that handles the full Sable
// portal-vendor flow in a single atomic step. It parses the ask text for
// the pattern "<zone> <gold>", validates the request, charges gold, clones
// the instance, and wires up the temporary exit. Failures are self-handling:
// gold is refunded and the mob says an explanation.
//
// params:
//   zones        (map: shortname → template zone name, e.g. arena: "Instance Arena")
//   min_gold     (int, default 100)
//   exit_expires (string, default "30 real minutes")
//
// On success the mob says the portal-open message and returns Success.
// On any validation or creation failure the mob says an explanation and
// returns Success (from the player's perspective the event was handled).
func actOpenInstancePortal(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}

	text := ctx.Event.Text

	// Parse "<zoneName> <gold>" from ask text.
	parts := splitTwo(text)
	if len(parts) < 2 {
		return Failure // not our pattern — let the tree try other branches
	}
	zoneName := parts[0]
	goldAmount := parseIntStr(parts[1])
	if goldAmount <= 0 {
		mob.Command("say Name a place and an amount of gold.")
		return Success
	}

	// Resolve zone short name → template name.
	templateZone := ""
	if zonesRaw, ok := params["zones"]; ok {
		switch zm := zonesRaw.(type) {
		case map[string]any:
			if v, ok := zm[zoneName]; ok {
				templateZone, _ = v.(string)
			}
		case map[any]any:
			if v, ok := zm[zoneName]; ok {
				templateZone, _ = v.(string)
			}
		case map[string]string:
			templateZone = zm[zoneName]
		}
	}
	if templateZone == "" {
		return Failure // unrecognized zone name — fall through to other branches
	}

	minGold := getIntParam(params, "min_gold")
	if minGold <= 0 {
		minGold = 100
	}
	if goldAmount < minGold {
		mob.Command(fmt.Sprintf("say That barely covers the runes. I need at least %d gold.", minGold))
		return Success
	}
	if user.Character.Gold < goldAmount {
		mob.Command("say You do not have that much gold.")
		return Success
	}

	// Charge gold before creating the instance.
	user.Character.Gold -= goldAmount

	authorizedUsers := []int{user.UserId}
	if party := parties.Get(user.UserId); party != nil {
		for _, memberId := range party.GetMembers() {
			if memberId != user.UserId {
				authorizedUsers = append(authorizedUsers, memberId)
			}
		}
	}

	inst, err := rooms.CreateZoneInstance(templateZone, goldAmount, user.UserId, authorizedUsers, ctx.RoomId)
	if err != nil {
		mudlog.Error("actOpenInstancePortal", "zone", templateZone, "userId", user.UserId, "error", err)
		user.Character.Gold += goldAmount // refund
		mob.Command("say Something went wrong. The planes resist me. Your gold is returned.")
		return Success
	}

	expires := getStringParam(params, "exit_expires")
	if expires == "" {
		expires = "30 real minutes"
	}
	exitKey := zoneName + "-" + fmt.Sprintf("%d", user.UserId)
	room := rooms.LoadRoom(ctx.RoomId)
	if room == nil {
		user.Character.Gold += goldAmount // refund
		mob.Command("say Something went wrong. Your gold is returned.")
		return Success
	}
	added := room.AddTemporaryExit(exitKey, exit.TemporaryRoomExit{
		RoomId:  inst.EntryRoomId,
		Title:   exitKey + " rift",
		UserId:  user.UserId,
		Expires: expires,
	})
	if !added {
		user.Character.Gold += goldAmount // refund
		mob.Command("say The archway is already sustaining too many rifts. Your gold is returned.")
		return Success
	}

	mob.Command("say The rift is open. Type " + exitKey + " to enter.")
	mob.Command("say It will hold for a time. Do not tarry.")
	room.SendTextToExits("You hear someone talking.", true)
	room.SendText("The stone archway flares with energy. A shimmering portal appears.")
	return Success
}

// splitTwo splits a string on the first whitespace run, returning up to 2 parts.
func splitTwo(s string) []string {
	s = strings.TrimSpace(s)
	for i, c := range s {
		if c == ' ' || c == '\t' {
			first := s[:i]
			rest := strings.TrimSpace(s[i:])
			return []string{first, rest}
		}
	}
	return []string{s}
}

// parseIntStr parses the first token from s as an integer. Returns 0 on failure.
func parseIntStr(s string) int {
	// Take first whitespace-delimited token in case there are trailing words.
	token := strings.Fields(s)
	if len(token) == 0 {
		return 0
	}
	n, err := strconv.Atoi(token[0])
	if err != nil {
		return 0
	}
	return n
}
