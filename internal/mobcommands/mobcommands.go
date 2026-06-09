package mobcommands

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/keywords"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// Signature of user command
type MobCommand func(rest string, mob *mobs.Mob, room *rooms.Room) (bool, error)

type CommandAccess struct {
	Func              MobCommand
	AllowedWhenDowned bool
}

var (
	mobCommands map[string]CommandAccess = map[string]CommandAccess{
		"aid":            {Aid, false},
		"attack":         {Attack, false},
		"bash":           {Bash, false},
		"befriend":       {Befriend, false},
		"blinding-flash": {BlindingFlash, false},
		"blinding-spit":  {BlindingSpit, false},
		"break":          {Break, false},
		"broadcast":      {Broadcast, false},
		"buy":            {Buy, false},
		"cancel":         {Cancel, false},
		"cast":           {Cast, false},
		"craft":          {Craft, false},
		"charge":         {Charge, false},
		"consume":        {Consume, false},
		"consider":       {Consider, false},
		"callforhelp":    {CallForHelp, false},
		"defuse":         {Defuse, false},
		"despawn":        {Despawn, false},
		"drink":          {Drink, false},
		"drop":           {Drop, false},
		"eat":            {Eat, false},
		"emote":          {Emote, true},
		"equip":          {Equip, false},
		"flee":           {Flee, false},
		"forage":         {Forage, false},
		"get":            {Get, false},
		"give":           {Give, false},
		"givequest":      {GiveQuest, false},
		"grapple":        {Grapple, false},
		"gearup":         {Gearup, false},
		"hamstring":      {Hamstring, false},
		"healing-gel":    {HealingGel, false},
		"howl":           {Howl, false},
		"taunt":          {Taunt, false},
		"go":             {Go, false},
		"kick":           {Kick, false},
		"lock":           {Lock, false},
		"look":           {Look, false},
		"lookforaid":     {LookForAid, false},
		"lookfortrouble": {LookForTrouble, false},
		"noop":           {Noop, true},
		"pacifism-aura":  {PacifismAura, false},
		"pathto":         {Pathto, false},
		"portal":         {Portal, false},
		"plant":          {Plant, false},
		"put":            {Put, false},
		"drain":          {Drain, false},
		"gore":           {Gore, false},
		"maul":           {Maul, false},
		"pounce":         {Pounce, false},
		"rake":           {Rake, false},
		"rally":          {Rally, false},
		"remove":         {Remove, false},
		"replyto":        {ReplyTo, true},
		"salvage":        {Salvage, false},
		"scan":           {Scan, false},
		"shadow":         {Shadow, false},
		"say":            {Say, true},
		"sayto":          {SayTo, true},
		"search":         {Search, false},
		"sell":           {Sell, false},
		"saytoonly":      {SayToOnly, true},
		"shout":          {Shout, true},
		"shoot":          {Shoot, false},
		"show":           {Show, false},
		"sleep":          {Sleep, false},
		"sneak":          {Sneak, false},
		"stand":          {Stand, false},
		"sonic-shout":    {SonicShout, false},
		"steal":          {Steal, false},
		"suicide":        {Suicide, true},
		//		"stash":  {Stash, false},
		"toxic-bite":     {ToxicBite, false},
		"track":          {Track, false},
		"trip":           {Trip, false},
		"unlock":         {Unlock, false},
		"warcry":  {Warcry, false},
		"wander":  {Wander, false},
	}
)

func GetAllMobCommands() []string {
	result := []string{}

	for cmd, _ := range mobCommands {
		result = append(result, cmd)
	}

	return result
}

func TryCommand(cmd string, rest string, mobId int) (bool, error) {

	cmd = strings.ToLower(cmd)
	rest = strings.TrimSpace(rest)

	cmd = keywords.TryCommandAlias(cmd)

	mobDisabled := false

	mob := mobs.GetInstance(mobId)
	if mob == nil {
		return false, errors.New(`mob instance doesn't exist`)
	}

	room := rooms.LoadRoom(mob.Character.RoomId)
	if room == nil {
		return false, fmt.Errorf(`room %d not found`, mob.Character.RoomId)
	}

	mobDisabled = mob.Character.IsDisabled()

	// Try any room props, only return if the response indicates it was handled
	/*
		if !mobDisabled {
			if handled, err := RoomProps(cmd, rest, userId); err != nil {
				return response, err
			} else if response.Handled {
				return response, err
			}
		}
	*/

	if alias := keywords.TryCommandAlias(cmd); alias != cmd {
		// If it's a multi-word aliase, we need to extract the first word to replace the command
		// The rest will be combined with any "rest" the mob provided.
		if strings.Contains(alias, ` `) {
			parts := strings.Split(alias, ` `)
			// grab the first word as the new cmd
			cmd = parts[0]
			// Add the "rest" to the end if any
			if len(rest) > 0 {
				rest = strings.TrimPrefix(alias, cmd+` `) + ` ` + rest
			} else {
				rest = strings.TrimPrefix(alias, cmd+` `)
			}
		} else {
			cmd = alias
		}
	}

	if cmdInfo, ok := mobCommands[cmd]; ok {
		if mobDisabled && !cmdInfo.AllowedWhenDowned {

			return true, nil
		}

		start := time.Now()
		defer func() {
			util.TrackTime(`mob-cmd[`+cmd+`]`, time.Since(start).Seconds())
		}()

		handled, err := cmdInfo.Func(rest, mob, room)
		return handled, err

	}
	// Try moving if they aren't disabled
	if !mobDisabled {
		start := time.Now()
		defer func() {
			util.TrackTime(`mob-cmd[go]`, time.Since(start).Seconds())
		}()

		if handled, err := Go(cmd, mob, room); err != nil {
			return handled, err
		} else if handled {
			return true, nil
		}

	}
	if emoteText, ok := actions.EmoteAliases[cmd]; ok {
		handled, err := Emote(emoteText, mob, room)
		return handled, err
	}

	return false, nil
}

// Register mob commands from outside of the package
func RegisterCommand(command string, handlerFunc MobCommand, isBlockable bool) {
	mobCommands[command] = CommandAccess{
		handlerFunc,
		isBlockable,
	}
}
