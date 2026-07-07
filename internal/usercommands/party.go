package usercommands

import (
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/actions"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/parties"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func Party(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	args := util.SplitButRespectQuotes(rest)

	partyCommand := `list`
	if len(args) > 0 {
		partyCommand = strings.ToLower(args[0])
		rest, _ = strings.CutPrefix(rest, args[0])
		rest = strings.TrimSpace(rest)
	}

	currentParty := parties.Get(user.UserId)

	if partyCommand == `admin` {
		return cmdPartyAdmin(user, rest)
	}

	if partyCommand == `create` || partyCommand == `new` || partyCommand == `start` {
		return cmdPartyCreate(user, currentParty)
	}

	if partyCommand == `invite` {
		return cmdPartyInvite(user, room, currentParty, rest)
	}

	//
	// what follows doesn't mamke sense unless they are in a party
	//

	if currentParty == nil {
		user.SendText(messaging.CategorySystem, `You are not attached to a party.`)
		return true, nil
	}

	if partyCommand == `accept` || partyCommand == `join` {
		return cmdPartyAccept(user, currentParty)
	}

	if partyCommand == `decline` {
		return cmdPartyDecline(user, currentParty)
	}

	if partyCommand == `list` {
		cmdPartyList(user, currentParty)
	}

	if currentParty.Invited(user.UserId) {
		user.SendText(messaging.CategorySystem, `You haven't accepted an invitation to the party.`)
		return true, nil
	}

	//
	// Everything after this point you must be in a party
	//
	if partyCommand == `autoattack` {
		cmdPartyAutoattack(user, currentParty, rest)
	}

	if partyCommand == `leave` || partyCommand == `quit` {
		return cmdPartyLeave(user, currentParty)
	}

	if partyCommand == `disband` || partyCommand == `stop` {
		return cmdPartyDisband(user, currentParty)
	}

	if partyCommand == `kick` {
		cmdPartyKick(user, currentParty, rest)
	}

	if partyCommand == `promote` {
		cmdPartyPromote(user, currentParty, rest)
	}

	if partyCommand == `loot` {
		return cmdPartyLoot(user, currentParty, rest)
	}

	if partyCommand == `gold` {
		return cmdPartyGold(user, currentParty, rest)
	}

	if partyCommand == `chat` || partyCommand == `say` {
		cmdPartyChat(user, currentParty, rest)
	}

	return true, nil
}

func dispatchPartyEvent(party *parties.Party, action string) {
	events.AddToQueue(events.PartyUpdated{
		Action:  action,
		UserIds: append(party.GetMembers(), party.GetInvited()...),
	})
}

func findPartyMemberByName(party *parties.Party, name string) (int, string, bool) {
	allMembers := []string{}
	memberIds := map[string]int{}
	for _, uid := range party.GetMembers() {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		allMembers = append(allMembers, u.Character.Name)
		memberIds[u.Character.Name] = uid
	}

	matchUser, closeMatchUser := util.FindMatchIn(name, allMembers...)
	if matchUser == `` {
		matchUser = closeMatchUser
	}

	if matchUser == `` {
		return 0, ``, false
	}

	return memberIds[matchUser], matchUser, true
}

func cmdPartyCreate(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	// check if they are already part of a party
	if currentParty != nil {
		if currentParty.Invited(user.UserId) {
			user.SendText(messaging.CategorySystem, `You already have a pending party invite. Try <ansi fg="command">party accept/decline</ansi> first`)
		} else if currentParty.IsLeader(user.UserId) {
			user.SendText(messaging.CategorySystem, `You already own a party Type <ansi fg="command">party list</ansi> for more info.`)
		} else {
			user.SendText(messaging.CategorySystem, `You are already party of a party.`)
		}
		return true, nil
	}

	if currentParty = parties.New(user.UserId); currentParty != nil {
		user.EventLog.Add(`party`, `Started a new party`)
		user.SendText(messaging.CategorySystem, `You started a new party!`)

		dispatchPartyEvent(currentParty, `created`)

	} else {
		user.SendText(messaging.CategorySystem, `Something went wrong.`)
	}

	return true, nil
}

func cmdPartyInvite(user *users.UserRecord, room *rooms.Room, currentParty *parties.Party, rest string) (bool, error) {
	if rest == `` {
		user.SendText(messaging.CategorySystem, `Invite who?`)
		return true, nil
	}

	// Not in a party? Create one.
	if currentParty == nil {
		currentParty = parties.New(user.UserId)
	}

	if !currentParty.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `You are not the leader of your party.`)
		return true, nil
	}

	target, err := actions.ResolveTargetActor(room, rest)
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s not found.`, rest))
		return true, nil
	}
	if !target.IsPlayer() {
		user.SendText(messaging.CategorySystem, `You can only invite players to your party.`)
		return true, nil
	}

	invitedUser := target.(*actions.UserActor).User
	invitePlayerId := invitedUser.UserId

	if invitedParty := parties.Get(invitePlayerId); invitedParty != nil {
		user.SendText(messaging.CategorySystem, `That player is already in a party.`)
		return true, nil
	}

	if currentParty.InvitePlayer(invitePlayerId) {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You invited <ansi fg="username">%s</ansi> to your party.`, invitedUser.Character.Name))
		invitedUser.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> invited you to their party. Type <ansi fg="command">party accept</ansi> or <ansi fg="command">party decline</ansi> to respond.`, user.Character.Name))
	} else {
		user.SendText(messaging.CategorySystem, `Something went wrong.`)
	}

	dispatchPartyEvent(currentParty, `invited`)

	return true, nil
}

func cmdPartyAccept(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	// Settle the shared pool to the EXISTING members before the new member
	// joins, so the joiner doesn't collect a share of gold pooled pre-join.
	if currentParty.Invited(user.UserId) {
		settlePartyGold(currentParty)
	}

	if currentParty.AcceptInvite(user.UserId) {

		user.EventLog.Add(`party`, `Joined a party`)
		user.SendText(messaging.CategorySystem, `You joined the party!`)
		for _, uid := range currentParty.UserIds {
			if uid == user.UserId {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> joined the party!`, user.Character.Name))
			}
		}

		dispatchPartyEvent(currentParty, `joined`)

	} else {
		user.SendText(messaging.CategorySystem, `Something went wrong.`)
	}
	return true, nil
}

func cmdPartyDecline(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	dispatchPartyEvent(currentParty, `declined`)

	if currentParty.DeclineInvite(user.UserId) {

		if u := users.GetByUserId(currentParty.LeaderUserId); u != nil {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> declined the invitation.`, user.Character.Name))
		}
		user.SendText(messaging.CategorySystem, `You decline the invitation.`)

	} else {
		user.SendText(messaging.CategorySystem, `Something went wrong.`)
	}
	return true, nil
}

func cmdPartyList(user *users.UserRecord, currentParty *parties.Party) {
	headers := []string{"Name", "Status", "Health", "Location", "Position"}
	formatting := [][]string{}

	rows := [][]string{}

	if currentParty != nil {
		isInvited := currentParty.Invited(user.UserId)
		leaderId := currentParty.LeaderUserId

		charmedMobInstanceIds := []int{}

		for _, uid := range currentParty.UserIds {
			uStatus := "In Party"
			if leaderId == uid {
				uStatus = "Leader"
			}

			u := users.GetByUserId(uid)
			uRoom := rooms.LoadRoom(u.Character.RoomId)
			uHealthPct := int(math.Floor((float64(u.Character.Health) / float64(u.Character.HealthMax.Value)) * 100))
			uHealthPctStr := fmt.Sprintf(`%d%%`, uHealthPct)
			uLoc := uRoom.Title
			rank := currentParty.GetRank(u.UserId)
			healthClass := util.HealthClass(u.Character.Health, u.Character.HealthMax.Value)

			if isInvited {
				uLoc = `-`
				uHealthPctStr = `-`
				rank = `-`
				healthClass = `black-bold`
			}

			rows = append(rows, []string{
				u.Character.Name,
				uStatus,
				uHealthPctStr,
				uLoc,
				rank,
			})

			rowFormat := []string{`<ansi fg="username">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="` + healthClass + `">%s</ansi>`,
				`<ansi fg="magenta-bold">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`}

			formatting = append(formatting, rowFormat)

			charmedMobInstanceIds = append(charmedMobInstanceIds, u.Character.GetCharmIds()...)
		}

		for _, mobInstanceId := range charmedMobInstanceIds {
			m := mobs.GetInstance(mobInstanceId)
			mRoom := rooms.LoadRoom(m.Character.RoomId)
			mHealthPct := int(math.Floor((float64(m.Character.Health) / float64(m.Character.HealthMax.Value)) * 100))
			rows = append(rows, []string{
				m.Character.Name,
				`♥friend`,
				fmt.Sprintf(`%d%%`, mHealthPct),
				mRoom.Title,
				`-`,
			})

			rowFormat := []string{`<ansi fg="username">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="` + util.HealthClass(m.Character.Health, m.Character.HealthMax.Value) + `">%s</ansi>`,
				`<ansi fg="magenta-bold">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`}

			formatting = append(formatting, rowFormat)
		}

		// Companions: show each party member's active companions.
		for _, uid := range currentParty.UserIds {
			u := users.GetByUserId(uid)
			if u == nil {
				continue
			}
			for _, comp := range u.Character.Companions {
				if comp.InstanceId == 0 {
					continue
				}
				m := mobs.GetInstance(comp.InstanceId)
				if m == nil {
					continue
				}
				mRoom := rooms.LoadRoom(m.Character.RoomId)
				if mRoom == nil {
					continue
				}
				mHealthPct := int(math.Floor((float64(m.Character.Health) / float64(m.Character.HealthMax.Value)) * 100))
				rows = append(rows, []string{
					m.Character.Name,
					`♦companion`,
					fmt.Sprintf(`%d%%`, mHealthPct),
					mRoom.Title,
					`-`,
				})
				rowFormat := []string{`<ansi fg="cyan">%s</ansi>`,
					`<ansi fg="cyan">%s</ansi>`,
					`<ansi fg="` + util.HealthClass(m.Character.Health, m.Character.HealthMax.Value) + `">%s</ansi>`,
					`<ansi fg="magenta-bold">%s</ansi>`,
					`<ansi fg="white-bold">%s</ansi>`}
				formatting = append(formatting, rowFormat)
			}
		}

		for _, uid := range currentParty.InviteUserIds {
			u := users.GetByUserId(uid)
			rows = append(rows, []string{
				u.Character.Name,
				`Invited`,
				`-`,
				`-`,
				`-`,
			})

			rowFormat := []string{`<ansi fg="username">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`,
				`<ansi fg="black-bold">%s</ansi>`,
				`<ansi fg="magenta-bold">%s</ansi>`,
				`<ansi fg="white-bold">%s</ansi>`}

			formatting = append(formatting, rowFormat)

		}

		partyTableData := templates.GetTable(`Party Members`, headers, rows, formatting...)
		partyTxt, _ := templates.Process("tables/generic", partyTableData, user.UserId)
		user.SendText(messaging.CategorySystem, partyTxt)

		if isInvited {
			user.SendText(messaging.CategorySystem, `Type <ansi fg="command">party accept/decline</ansi> to finalize your party membership.`)
		}
	}
}

func cmdPartyAutoattack(user *users.UserRecord, currentParty *parties.Party, rest string) {
	// Default is on — only "off" disables it
	wasOn := user.Character.GetSetting("autoattack") != "off"

	if rest == `on` {
		user.Character.SetSetting("autoattack", "")
		if wasOn {
			user.SendText(messaging.CategorySystem, `You already have auto-attack enabled.`)
		} else {
			user.SendText(messaging.CategorySystem, `You are now auto-attacking with your party.`)
		}
	} else if rest == `off` {
		user.Character.SetSetting("autoattack", "off")
		if wasOn {
			user.SendText(messaging.CategorySystem, `You are no longer auto-attacking with your party.`)
		} else {
			user.SendText(messaging.CategorySystem, `You already have auto-attacking disabled.`)
		}
	} else {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">party autoattack [on/off]</ansi>`)
		return
	}

	dispatchPartyEvent(currentParty, `behavior`)
}

func cmdPartyLeave(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	// Settle the shared gold pool BEFORE the departing member is removed, so
	// they still receive their share.
	settlePartyGold(currentParty)

	if currentParty.IsLeader(user.UserId) {

		if len(currentParty.UserIds) <= 1 {

			dispatchPartyEvent(currentParty, `disbanded`)

			user.EventLog.Add(`party`, `Disbanded your party`)
			user.SendText(messaging.CategorySystem, `You disbanded the party.`)
			currentParty.Disband()

			return true, nil
		}

		currentParty.LeaderUserId = 0

		// promote someone else to leader
		for _, uid := range currentParty.UserIds {
			if uid == user.UserId {
				continue
			}

			newLeaderUser := users.GetByUserId(uid)

			if newLeaderUser == nil {
				continue
			}

			currentParty.LeaderUserId = uid

			break
		}

		if currentParty.LeaderUserId > 0 {
			newLeaderUser := users.GetByUserId(currentParty.LeaderUserId)
			for _, uid := range currentParty.UserIds {
				if u := users.GetByUserId(uid); u != nil {
					if currentParty.LeaderUserId == uid {
						u.EventLog.Add(`party`, `Promoted to party leader`)
						u.SendText(messaging.CategorySystem, `You are now the leader of the party.`)
					} else {
						u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the leader of the party.`, newLeaderUser.Character.Name))
					}
				}
			}

			dispatchPartyEvent(currentParty, `promotion`)
		}

		currentParty.Leave(user.UserId)
		user.EventLog.Add(`party`, `Left the party`)
		user.SendText(messaging.CategorySystem, `You left the party.`)

		return true, nil
	}

	dispatchPartyEvent(currentParty, `left`)

	currentParty.Leave(user.UserId)
	user.EventLog.Add(`party`, `Left the party`)
	user.SendText(messaging.CategorySystem, `You left the party.`)

	for _, uid := range currentParty.UserIds {
		if uid == user.UserId {
			continue
		}
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> left the party.`, user.Character.Name))
		}
	}

	return true, nil
}

func cmdPartyDisband(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `You are not the leader of your party.`)
		return true, nil
	}

	// Settle the shared gold pool BEFORE disbanding, so every member gets
	// their share.
	settlePartyGold(currentParty)

	for _, uid := range currentParty.UserIds {
		if uid == user.UserId {
			continue
		}
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
		}
	}
	for _, uid := range currentParty.InviteUserIds {
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
		}
	}

	dispatchPartyEvent(currentParty, `disbanded`)

	currentParty.Disband()
	user.EventLog.Add(`party`, `Disbanded the party`)
	user.SendText(messaging.CategorySystem, `You disbanded the party.`)

	return true, nil
}

func cmdPartyKick(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `You are not the leader of your party.`)
		return
	}

	// Settle the shared gold pool BEFORE the kicked member is removed, so
	// they still receive their share.
	settlePartyGold(currentParty)

	kickUserId, matchUser, found := findPartyMemberByName(currentParty, rest)
	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s not found.`, rest))
		return
	}

	dispatchPartyEvent(currentParty, `left`)

	currentParty.Leave(kickUserId)

	if u := users.GetByUserId(kickUserId); u != nil {
		u.SendText(messaging.CategorySystem, `You were kicked from the party.`)
	}

	for _, uid := range currentParty.UserIds {
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> was kicked from the party.`, matchUser))
		}
	}
}

func cmdPartyPromote(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `You are not the leader of your party.`)
		return
	}

	promoteUserId, matchUser, found := findPartyMemberByName(currentParty, rest)
	if !found {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`%s not found.`, rest))
		return
	}

	dispatchPartyEvent(currentParty, `promotion`)

	currentParty.LeaderUserId = promoteUserId

	if u := users.GetByUserId(promoteUserId); u != nil {
		u.EventLog.Add(`party`, `Promoted to party leader`)
		u.SendText(messaging.CategorySystem, `You have been promoted to party leader.`)
	}

	for _, uid := range currentParty.UserIds {
		if uid != promoteUserId {
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the party leader.`, matchUser))
			}
		}
	}
}

// cmdPartyLoot sets the party's corpse-loot distribution mode. Leader-only.
// Accepts several aliases for each of the three canonical modes and announces
// the change to the whole party.
func cmdPartyLoot(user *users.UserRecord, currentParty *parties.Party, rest string) (bool, error) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the party leader can set the loot mode.`)
		return true, nil
	}

	// Canonicalize the requested mode from its aliases.
	mode := ``
	switch strings.ToLower(strings.TrimSpace(rest)) {
	case `ffa`, `free`, `freeforall`, `free-for-all`:
		mode = `ffa`
	case `roundrobin`, `rr`, `round-robin`, `round`:
		mode = `roundrobin`
	case `leaderhold`, `leader`, `hold`, `master`:
		mode = `leaderhold`
	}

	if mode == `` {
		// Unrecognized or empty: show usage + the current mode without changing.
		current := currentParty.LootMode
		if current == `` {
			current = `ffa`
		}
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">party loot [ffa/roundrobin/leaderhold]</ansi>`)
		user.SendText(messaging.CategorySystem, `  <ansi fg="command">ffa</ansi>         - Free-for-all: anyone can loot corpses.`)
		user.SendText(messaging.CategorySystem, `  <ansi fg="command">roundrobin</ansi>  - Loot rights rotate through party members.`)
		user.SendText(messaging.CategorySystem, `  <ansi fg="command">leaderhold</ansi>  - Only the leader may loot corpses.`)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`The current loot mode is <ansi fg="magenta-bold">%s</ansi>.`, current))
		return true, nil
	}

	currentParty.LootMode = mode

	// Friendly label for the announcement.
	label := mode
	switch mode {
	case `ffa`:
		label = `free-for-all`
	case `roundrobin`:
		label = `round-robin`
	case `leaderhold`:
		label = `leader-only`
	}

	for _, uid := range currentParty.UserIds {
		u := users.GetByUserId(uid)
		if u == nil {
			continue
		}
		if uid == user.UserId {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`You set the party loot mode to <ansi fg="magenta-bold">%s</ansi>.`, label))
		} else {
			u.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> set the party loot mode to <ansi fg="magenta-bold">%s</ansi>.`, user.Character.Name, label))
		}
	}

	dispatchPartyEvent(currentParty, `behavior`)

	return true, nil
}

// settlePartyGold splits the party's shared gold pool evenly across members
// (remainder to the leader), credits each member's character gold, emits a
// gold-sync event so prompts/GMCP update, and tells each member their share.
// SettleGold is a pure computation in the parties package (which does not
// import internal/users); the crediting + announcement happen here in the
// command layer. Returns the total gold paid out (0 if the pool was empty).
func settlePartyGold(currentParty *parties.Party) int {
	payouts := currentParty.SettleGold()
	if len(payouts) == 0 {
		return 0
	}

	total := 0
	reserved := 0
	for uid, amount := range payouts {
		if amount <= 0 {
			continue
		}
		u := users.GetByUserId(uid)
		if u == nil {
			// Member is offline/unloaded; keep their share in the pool rather
			// than destroying it (SettleGold already zeroed the pool). Re-pooled
			// below so gold is conserved for when they next settle.
			reserved += amount
			continue
		}
		total += amount
		u.Character.Gold += amount
		// Reuse the solo/pool gold-sync event shape (Task 11 / get.go): a
		// credit is signalled with a negative GoldChange so prompts refresh.
		events.AddToQueue(events.EquipmentChange{
			UserId:     uid,
			GoldChange: -amount,
		})
		u.SendText(messaging.CategoryLoot,
			fmt.Sprintf(`Your share of the party pool: <ansi fg="yellow-bold">%d gold</ansi>.`, amount))
	}
	currentParty.GoldPool += reserved // conserve: undistributed shares stay pooled
	return total
}

// cmdPartyGold reports or splits the shared party gold pool. Any member may
// trigger a split. With no argument (or "split"), the pool is settled and
// paid out; otherwise the current pool balance is shown.
func cmdPartyGold(user *users.UserRecord, currentParty *parties.Party, rest string) (bool, error) {
	arg := strings.ToLower(strings.TrimSpace(rest))

	if arg == `` || strings.HasPrefix(arg, `split`) {
		if currentParty.GoldPool == 0 {
			user.SendText(messaging.CategorySystem, `The party pool is empty.`)
			return true, nil
		}
		settlePartyGold(currentParty)
		return true, nil
	}

	user.SendText(messaging.CategorySystem,
		fmt.Sprintf(`The party pool holds <ansi fg="yellow-bold">%d gold</ansi>.`, currentParty.GoldPool))
	user.SendText(messaging.CategorySystem, `Type <ansi fg="command">party gold split</ansi> to divide it among members.`)
	return true, nil
}

func cmdPartyChat(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if len(rest) == 0 {
		user.SendText(messaging.CategorySystem, `What do you want to say?`)
		return
	}

	for _, uId := range currentParty.GetMembers() {
		if uId == user.UserId {
			continue
		}
		if u := users.GetByUserId(uId); u != nil {
			msg := fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> <ansi fg="username">%s</ansi> says, "<ansi fg="yellow">%s</ansi>"`, user.Character.Name, rest)
			u.SendText(messaging.CategorySystem, util.SplitStringNL(msg, 80))
		}
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> You say, "<ansi fg="yellow">%s</ansi>"`, rest))

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     `party`,
		Name:         user.Character.Name,
		Message:      rest,
	})
}

// cmdPartyAdmin dispatches party admin subcommands. All subcommands require
// the party.debug role permission (admin users always pass).
func cmdPartyAdmin(user *users.UserRecord, rest string) (bool, error) {
	if !user.HasRolePermission(`party.debug`) {
		user.SendText(messaging.CategorySystem, `You do not have <ansi fg="command">party.debug</ansi> permission.`)
		return true, nil
	}

	args := util.SplitButRespectQuotes(rest)
	sub := ``
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}

	switch sub {
	case `list-npc`:
		cmdPartyAdminListNPC(user)
	case `show`:
		idStr := ``
		if len(args) > 1 {
			idStr = args[1]
		}
		cmdPartyAdminShow(user, idStr)
	default:
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">party admin list-npc</ansi>`)
		user.SendText(messaging.CategorySystem, `       <ansi fg="command">party admin show <id></ansi>`)
	}
	return true, nil
}

// cmdPartyAdminListNPC lists all parties whose leader is an NPC.
func cmdPartyAdminListNPC(user *users.UserRecord) {
	all := parties.ListAllParties()
	lines := []string{}
	for _, p := range all {
		if p.Leader == nil || p.Leader.IsPlayer() {
			continue
		}
		homeStr := `-`
		if p.HomeRoomId != 0 {
			homeStr = strconv.Itoa(p.HomeRoomId)
		}
		helpStr := `-`
		if p.HelpRoomId != 0 {
			helpStr = strconv.Itoa(p.HelpRoomId)
		}
		lines = append(lines, fmt.Sprintf(
			`  #%-3d  Leader: %-30s Members: %-3d HomeRoom: %-6s HelpRoom: %s`,
			p.PartyIdInternal(),
			fmt.Sprintf(`%s (mob %d)`, p.Leader.GetName(), p.Leader.GetMobInstanceId()),
			len(p.Members),
			homeStr,
			helpStr,
		))
	}
	if len(lines) == 0 {
		user.SendText(messaging.CategorySystem, `No active NPC parties.`)
		return
	}
	user.SendText(messaging.CategorySystem, `NPC Parties:`)
	for _, l := range lines {
		user.SendText(messaging.CategorySystem, l)
	}
}

// cmdPartyAdminShow dumps full state of one party by PartyIdInternal().
func cmdPartyAdminShow(user *users.UserRecord, idStr string) {
	if idStr == `` {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">party admin show <id></ansi>`)
		return
	}
	targetId, err := strconv.Atoi(idStr)
	if err != nil || targetId <= 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Invalid party id: %s`, idStr))
		return
	}

	var found *parties.Party
	for _, p := range parties.ListAllParties() {
		if p.PartyIdInternal() == targetId {
			found = p
			break
		}
	}
	if found == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`No party with id %d.`, targetId))
		return
	}

	homeStr := `(none)`
	if found.HomeRoomId != 0 {
		homeStr = strconv.Itoa(found.HomeRoomId)
	}
	helpStr := `(none)`
	if found.HelpRoomId != 0 {
		helpStr = strconv.Itoa(found.HelpRoomId)
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Party #%d`, found.PartyIdInternal()))

	if found.Leader != nil {
		leaderRoom := 0
		if r := found.Leader.GetRoom(); r != nil {
			leaderRoom = r.RoomId
		}
		if found.Leader.IsPlayer() {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`  Leader: %s (player %d, room %d)`,
				found.Leader.GetName(),
				found.Leader.GetUserId(),
				leaderRoom,
			))
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`  Leader: %s (mob %d, room %d)`,
				found.Leader.GetName(),
				found.Leader.GetMobInstanceId(),
				leaderRoom,
			))
		}
	} else {
		user.SendText(messaging.CategorySystem, `  Leader: (none)`)
	}

	user.SendText(messaging.CategorySystem, `  Members:`)
	for _, m := range found.Members {
		memberRoom := 0
		if r := m.GetRoom(); r != nil {
			memberRoom = r.RoomId
		}
		leaderTag := ``
		if found.Leader != nil && m == found.Leader {
			leaderTag = ` [LEADER]`
		}
		if m.IsPlayer() {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`    %s (player %d, room %d)%s`,
				m.GetName(), m.GetUserId(), memberRoom, leaderTag,
			))
		} else {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(
				`    %s (mob %d, room %d)%s`,
				m.GetName(), m.GetMobInstanceId(), memberRoom, leaderTag,
			))
		}
	}

	user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Home room: %s`, homeStr))
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`  Help room: %s`, helpStr))
}
