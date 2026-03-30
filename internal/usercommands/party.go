package usercommands

import (
	"fmt"
	"math"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
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
		user.SendText(`You are not attached to a party.`)
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
		user.SendText(`You haven't accepted an invitation to the party.`)
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
			user.SendText(`You already have a pending party invite. Try <ansi fg="command">party accept/decline</ansi> first`)
		} else if currentParty.IsLeader(user.UserId) {
			user.SendText(`You already own a party Type <ansi fg="command">party list</ansi> for more info.`)
		} else {
			user.SendText(`You are already party of a party.`)
		}
		return true, nil
	}

	if currentParty = parties.New(user.UserId); currentParty != nil {
		user.EventLog.Add(`party`, `Started a new party`)
		user.SendText(`You started a new party!`)

		dispatchPartyEvent(currentParty, `created`)

	} else {
		user.SendText(`Something went wrong.`)
	}

	return true, nil
}

func cmdPartyInvite(user *users.UserRecord, room *rooms.Room, currentParty *parties.Party, rest string) (bool, error) {
	if rest == `` {
		user.SendText(`Invite who?`)
		return true, nil
	}

	// Not in a party? Create one.
	if currentParty == nil {
		currentParty = parties.New(user.UserId)
	}

	if !currentParty.IsLeader(user.UserId) {
		user.SendText(`You are not the leader of your party.`)
		return true, nil
	}

	invitePlayerId, mobInstId := room.FindByName(rest)

	if invitePlayerId == 0 && mobInstId == 0 {
		user.SendText(fmt.Sprintf(`%s not found.`, rest))
		return true, nil
	}

	if invitedParty := parties.Get(invitePlayerId); invitedParty != nil {
		user.SendText(`That player is already in a party.`)
		return true, nil
	}

	invitedUser := users.GetByUserId(invitePlayerId)

	if invitedUser != nil && currentParty.InvitePlayer(invitePlayerId) {
		user.SendText(fmt.Sprintf(`You invited <ansi fg="username">%s</ansi> to your party.`, invitedUser.Character.Name))
		invitedUser.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> invited you to their party. Type <ansi fg="command">party accept</ansi> or <ansi fg="command">party decline</ansi> to respond.`, user.Character.Name))
	} else {
		user.SendText(`Something went wrong.`)
	}

	dispatchPartyEvent(currentParty, `invited`)

	return true, nil
}

func cmdPartyAccept(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	if currentParty.AcceptInvite(user.UserId) {

		user.EventLog.Add(`party`, `Joined a party`)
		user.SendText(`You joined the party!`)
		for _, uid := range currentParty.UserIds {
			if uid == user.UserId {
				continue
			}
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> joined the party!`, user.Character.Name))
			}
		}

		dispatchPartyEvent(currentParty, `joined`)

	} else {
		user.SendText(`Something went wrong.`)
	}
	return true, nil
}

func cmdPartyDecline(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	dispatchPartyEvent(currentParty, `declined`)

	if currentParty.DeclineInvite(user.UserId) {

		if u := users.GetByUserId(currentParty.LeaderUserId); u != nil {
			u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> declined the invitation.`, user.Character.Name))
		}
		user.SendText(`You decline the invitation.`)

	} else {
		user.SendText(`Something went wrong.`)
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
		user.SendText(partyTxt)

		if isInvited {
			user.SendText(`Type <ansi fg="command">party accept/decline</ansi> to finalize your party membership.`)
		}
	}
}

func cmdPartyAutoattack(user *users.UserRecord, currentParty *parties.Party, rest string) {
	// Default is on — only "off" disables it
	wasOn := user.Character.GetSetting("autoattack") != "off"

	if rest == `on` {
		user.Character.SetSetting("autoattack", "")
		if wasOn {
			user.SendText(`You already have auto-attack enabled.`)
		} else {
			user.SendText(`You are now auto-attacking with your party.`)
		}
	} else if rest == `off` {
		user.Character.SetSetting("autoattack", "off")
		if wasOn {
			user.SendText(`You are no longer auto-attacking with your party.`)
		} else {
			user.SendText(`You already have auto-attacking disabled.`)
		}
	} else {
		user.SendText(`Usage: <ansi fg="command">party autoattack [on/off]</ansi>`)
		return
	}

	dispatchPartyEvent(currentParty, `behavior`)
}

func cmdPartyLeave(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	if currentParty.IsLeader(user.UserId) {

		if len(currentParty.UserIds) <= 1 {

			dispatchPartyEvent(currentParty, `disbanded`)

			user.EventLog.Add(`party`, `Disbanded your party`)
			user.SendText(`You disbanded the party.`)
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
						u.SendText(`You are now the leader of the party.`)
					} else {
						u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the leader of the party.`, newLeaderUser.Character.Name))
					}
				}
			}

			dispatchPartyEvent(currentParty, `promotion`)
		}

		currentParty.Leave(user.UserId)
		user.EventLog.Add(`party`, `Left the party`)
		user.SendText(`You left the party.`)

		return true, nil
	}

	dispatchPartyEvent(currentParty, `left`)

	currentParty.Leave(user.UserId)
	user.EventLog.Add(`party`, `Left the party`)
	user.SendText(`You left the party.`)

	for _, uid := range currentParty.UserIds {
		if uid == user.UserId {
			continue
		}
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> left the party.`, user.Character.Name))
		}
	}

	return true, nil
}

func cmdPartyDisband(user *users.UserRecord, currentParty *parties.Party) (bool, error) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(`You are not the leader of your party.`)
		return true, nil
	}

	for _, uid := range currentParty.UserIds {
		if uid == user.UserId {
			continue
		}
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
		}
	}
	for _, uid := range currentParty.InviteUserIds {
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> disbanded the party.`, user.Character.Name))
		}
	}

	dispatchPartyEvent(currentParty, `disbanded`)

	currentParty.Disband()
	user.EventLog.Add(`party`, `Disbanded the party`)
	user.SendText(`You disbanded the party.`)

	return true, nil
}

func cmdPartyKick(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(`You are not the leader of your party.`)
		return
	}

	kickUserId, matchUser, found := findPartyMemberByName(currentParty, rest)
	if !found {
		user.SendText(fmt.Sprintf(`%s not found.`, rest))
		return
	}

	dispatchPartyEvent(currentParty, `left`)

	currentParty.Leave(kickUserId)

	if u := users.GetByUserId(kickUserId); u != nil {
		u.SendText(`You were kicked from the party.`)
	}

	for _, uid := range currentParty.UserIds {
		if u := users.GetByUserId(uid); u != nil {
			u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> was kicked from the party.`, matchUser))
		}
	}
}

func cmdPartyPromote(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if !currentParty.IsLeader(user.UserId) {
		user.SendText(`You are not the leader of your party.`)
		return
	}

	promoteUserId, matchUser, found := findPartyMemberByName(currentParty, rest)
	if !found {
		user.SendText(fmt.Sprintf(`%s not found.`, rest))
		return
	}

	dispatchPartyEvent(currentParty, `promotion`)

	currentParty.LeaderUserId = promoteUserId

	if u := users.GetByUserId(promoteUserId); u != nil {
		u.EventLog.Add(`party`, `Promoted to party leader`)
		u.SendText(`You have been promoted to party leader.`)
	}

	for _, uid := range currentParty.UserIds {
		if uid != promoteUserId {
			if u := users.GetByUserId(uid); u != nil {
				u.SendText(fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the party leader.`, matchUser))
			}
		}
	}
}

func cmdPartyChat(user *users.UserRecord, currentParty *parties.Party, rest string) {
	if len(rest) == 0 {
		user.SendText(`What do you want to say?`)
		return
	}

	for _, uId := range currentParty.GetMembers() {
		if uId == user.UserId {
			continue
		}
		if u := users.GetByUserId(uId); u != nil {
			msg := fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> <ansi fg="username">%s</ansi> says, "<ansi fg="yellow">%s</ansi>"`, user.Character.Name, rest)
			u.SendText(util.SplitStringNL(msg, 80))
		}
	}

	user.SendText(fmt.Sprintf(`<ansi fg="magenta">(party)</ansi> You say, "<ansi fg="yellow">%s</ansi>"`, rest))

	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     `party`,
		Name:         user.Character.Name,
		Message:      rest,
	})
}
