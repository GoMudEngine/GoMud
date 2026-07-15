package usercommands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Guild is the player-facing guild command. It dispatches to subcommands; all
// mutations flow through the internal/guilds registry.
func Guild(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {

	rest = strings.TrimSpace(rest)
	args := strings.Fields(rest)
	sub := ""
	if len(args) > 0 {
		sub = strings.ToLower(args[0])
	}
	remainder := strings.TrimSpace(strings.Join(args[1:], " "))

	switch sub {
	case "", "info":
		guildInfo(user)
	case "create":
		guildCreate(user, remainder, rest)
	case "list":
		guildList(user)
	case "leave":
		guildLeave(user, rest)
	case "disband":
		guildDisband(user, rest)
	case "invite":
		guildInvite(user, remainder)
	case "accept":
		guildAcceptInvite(user)
	case "decline":
		guildDeclineInvite(user)
	case "kick":
		guildKick(user, remainder)
	case "promote":
		guildSetRank(user, remainder, true)
	case "demote":
		guildSetRank(user, remainder, false)
	case "transfer":
		guildTransfer(user, remainder, rest)
	case "motd":
		guildSetMotd(user, remainder)
	default:
		user.SendText(messaging.CategorySystem, `Unknown guild command. Try <ansi fg="command">help guild</ansi>.`)
	}
	return true, nil
}

// announceGuild sends msg to every online member of g except exceptUserId.
func announceGuild(g *guilds.Guild, msg string, exceptUserId int) {
	for _, m := range g.Members {
		if m.UserId == exceptUserId {
			continue
		}
		if u := users.GetByUserId(m.UserId); u != nil {
			u.SendText(messaging.CategorySystem, msg)
		}
	}
}

func guildInfo(user *users.UserRecord) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild. Found one with <ansi fg="command">guild create <TAG> <name></ansi>, or get invited to one.`)
		return
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi> <ansi fg="cyan">[%s]</ansi> — %d member(s)`, g.Name, g.Tag, len(g.Members)))
	if g.Motd != "" {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">"%s"</ansi>`, g.Motd))
	}
	// Roster: leader, then officers, then members.
	for _, rank := range []guilds.GuildRank{guilds.RankLeader, guilds.RankOfficer, guilds.RankMember} {
		for _, m := range g.Members {
			if m.Rank == rank {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="white">%-20s</ansi> <ansi fg="black-bold">%s</ansi>`, m.CharacterName, m.Rank))
			}
		}
	}
}

func guildCreate(user *users.UserRecord, remainder, rest string) {
	if _, in := guilds.GetByUser(user.UserId); in {
		user.SendText(messaging.CategorySystem, `You are already in a guild.`)
		return
	}
	fields := strings.Fields(remainder)
	if len(fields) < 2 {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild create <TAG> <name></ansi>  (TAG is 2-4 letters/digits).`)
		return
	}
	tag := fields[0]
	name := strings.Join(fields[1:], " ")
	cost := int(configs.GetBalanceConfig().GuildFoundingCost)

	cmdPrompt, _ := user.StartPrompt(`guild`, rest)
	q := cmdPrompt.Ask(fmt.Sprintf(`Found "%s" [%s] for %d gold from your bank?`, name, strings.ToUpper(tag), cost), []string{`Yes`, `No`}, `No`)
	if !q.Done {
		return
	}
	user.ClearPrompt()
	if len(q.Response) == 0 || q.Response[0:1] != `Y` {
		user.SendText(messaging.CategorySystem, `Guild founding cancelled.`)
		return
	}

	if user.Character.Bank < cost {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You need %d gold in the bank to found a guild.`, cost))
		return
	}

	g, err := guilds.Create(tag, name, user.UserId, user.Character.Name)
	if err != nil {
		user.SendText(messaging.CategorySystem, `Cannot found the guild: `+err.Error())
		return
	}
	user.Character.Bank -= cost
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, BankChange: -cost})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You found <ansi fg="yellow-bold">%s</ansi> <ansi fg="cyan">[%s]</ansi>! You are its leader. Invite others with <ansi fg="command">guild invite <player></ansi>.`, g.Name, g.Tag))
}

func guildList(user *users.UserRecord) {
	all := guilds.All()
	if len(all) == 0 {
		user.SendText(messaging.CategorySystem, `There are no guilds yet. Found one with <ansi fg="command">guild create</ansi>.`)
		return
	}
	sort.Slice(all, func(a, b int) bool { return strings.ToLower(all[a].Name) < strings.ToLower(all[b].Name) })
	user.SendText(messaging.CategorySystem, `<ansi fg="yellow-bold">Guilds:</ansi>`)
	for _, g := range all {
		leaderName := "?"
		for _, m := range g.Members {
			if m.UserId == g.LeaderUserId {
				leaderName = m.CharacterName
				break
			}
		}
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="cyan">[%s]</ansi> <ansi fg="white">%s</ansi> — %d member(s), led by %s`, g.Tag, g.Name, len(g.Members), leaderName))
	}
}

func guildLeave(user *users.UserRecord, rest string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if g.IsLeader(user.UserId) {
		if len(g.Members) > 1 {
			user.SendText(messaging.CategorySystem, `As leader you must <ansi fg="command">guild transfer <player></ansi> or <ansi fg="command">guild disband</ansi> first.`)
			return
		}
		// Sole-member leader: leaving disbands.
		cmdPrompt, _ := user.StartPrompt(`guild`, rest)
		q := cmdPrompt.Ask(`You are the only member — leaving will disband the guild. Proceed?`, []string{`Yes`, `No`}, `No`)
		if !q.Done {
			return
		}
		user.ClearPrompt()
		if len(q.Response) == 0 || q.Response[0:1] != `Y` {
			user.SendText(messaging.CategorySystem, `Cancelled.`)
			return
		}
		tag := g.Tag
		guilds.Delete(tag)
		user.SendText(messaging.CategorySystem, `You disband the guild.`)
		return
	}
	tag := g.Tag
	name := g.Name
	guilds.RemoveMember(tag, user.UserId)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You leave <ansi fg="yellow-bold">%s</ansi>.`, name))
	if g2, ok := guilds.Get(tag); ok {
		announceGuild(g2, fmt.Sprintf(`<ansi fg="username">%s</ansi> has left the guild.`, user.Character.Name), user.UserId)
	}
}

func guildDisband(user *users.UserRecord, rest string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the guild leader can disband the guild.`)
		return
	}
	cmdPrompt, _ := user.StartPrompt(`guild`, rest)
	q := cmdPrompt.Ask(fmt.Sprintf(`Disband "%s"? This cannot be undone.`, g.Name), []string{`Yes`, `No`}, `No`)
	if !q.Done {
		return
	}
	user.ClearPrompt()
	if len(q.Response) == 0 || q.Response[0:1] != `Y` {
		user.SendText(messaging.CategorySystem, `Cancelled.`)
		return
	}
	announceGuild(g, fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi> has been disbanded by its leader.`, g.Name), user.UserId)
	guilds.Delete(g.Tag)
	user.SendText(messaging.CategorySystem, `You disband the guild.`)
}

// findGuildMemberByName resolves a guild member's userId by character name
// (stored name first, then a matching online player who is a member).
func findGuildMemberByName(g *guilds.Guild, name string) (int, bool) {
	name = strings.TrimSpace(name)
	if name == "" {
		return 0, false
	}
	for _, m := range g.Members {
		if strings.EqualFold(m.CharacterName, name) {
			return m.UserId, true
		}
	}
	if u := users.GetByCharacterName(name); u != nil && g.IsMember(u.UserId) {
		return u.UserId, true
	}
	return 0, false
}

func memberName(g *guilds.Guild, userId int) string {
	for _, m := range g.Members {
		if m.UserId == userId {
			return m.CharacterName
		}
	}
	return "someone"
}

func guildInvite(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.CanManage(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only officers and the leader can invite.`)
		return
	}
	name := strings.TrimSpace(remainder)
	if name == "" {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild invite <player></ansi>`)
		return
	}
	target := users.GetByCharacterName(name)
	if target == nil {
		user.SendText(messaging.CategorySystem, `No player by that name is online to invite.`)
		return
	}
	if target.UserId == user.UserId {
		user.SendText(messaging.CategorySystem, `You can't invite yourself.`)
		return
	}
	if _, in := guilds.GetByUser(target.UserId); in {
		user.SendText(messaging.CategorySystem, `That player is already in a guild.`)
		return
	}
	if g.HasInvite(target.UserId) {
		user.SendText(messaging.CategorySystem, `That player already has a pending invite.`)
		return
	}
	guilds.AddInvite(g.Tag, target.UserId)
	target.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="username">%s</ansi> invites you to join <ansi fg="yellow-bold">%s</ansi> <ansi fg="cyan">[%s]</ansi>. <ansi fg="command">guild accept</ansi> or <ansi fg="command">guild decline</ansi>.`, user.Character.Name, g.Name, g.Tag))
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You invite <ansi fg="username">%s</ansi> to the guild.`, target.Character.Name))
}

func guildAcceptInvite(user *users.UserRecord) {
	if _, in := guilds.GetByUser(user.UserId); in {
		user.SendText(messaging.CategorySystem, `You are already in a guild.`)
		return
	}
	g, ok := guilds.GuildWithInvite(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You have no pending guild invitation.`)
		return
	}
	if err := guilds.AddMember(g.Tag, user.UserId, user.Character.Name); err != nil {
		user.SendText(messaging.CategorySystem, `Cannot join: `+err.Error())
		return
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You join <ansi fg="yellow-bold">%s</ansi> <ansi fg="cyan">[%s]</ansi>!`, g.Name, g.Tag))
	if g2, ok := guilds.Get(g.Tag); ok {
		announceGuild(g2, fmt.Sprintf(`<ansi fg="username">%s</ansi> has joined the guild.`, user.Character.Name), user.UserId)
	}
}

func guildDeclineInvite(user *users.UserRecord) {
	g, ok := guilds.GuildWithInvite(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You have no pending guild invitation.`)
		return
	}
	guilds.RemoveInvite(g.Tag, user.UserId)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You decline the invitation to <ansi fg="yellow-bold">%s</ansi>.`, g.Name))
}

func guildKick(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.CanManage(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only officers and the leader can kick.`)
		return
	}
	targetId, found := findGuildMemberByName(g, remainder)
	if !found {
		user.SendText(messaging.CategorySystem, `No guild member by that name.`)
		return
	}
	if targetId == user.UserId {
		user.SendText(messaging.CategorySystem, `To leave, use <ansi fg="command">guild leave</ansi>.`)
		return
	}
	if !g.CanKick(user.UserId, targetId) {
		user.SendText(messaging.CategorySystem, `You can't kick someone of equal or higher rank.`)
		return
	}
	name := memberName(g, targetId)
	guilds.RemoveMember(g.Tag, targetId)
	if tu := users.GetByUserId(targetId); tu != nil {
		tu.SendText(messaging.CategorySystem, fmt.Sprintf(`You have been removed from <ansi fg="yellow-bold">%s</ansi>.`, g.Name))
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You remove <ansi fg="username">%s</ansi> from the guild.`, name))
	if g2, ok := guilds.Get(g.Tag); ok {
		announceGuild(g2, fmt.Sprintf(`<ansi fg="username">%s</ansi> was removed from the guild.`, name), user.UserId)
	}
}

func guildSetRank(user *users.UserRecord, remainder string, promote bool) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the guild leader can change ranks.`)
		return
	}
	targetId, found := findGuildMemberByName(g, remainder)
	if !found || targetId == user.UserId {
		user.SendText(messaging.CategorySystem, `Name another guild member.`)
		return
	}
	cur, _ := g.MemberRank(targetId)
	name := memberName(g, targetId)
	if promote {
		if cur != guilds.RankMember {
			user.SendText(messaging.CategorySystem, `That member is already an officer.`)
			return
		}
		guilds.SetRank(g.Tag, targetId, guilds.RankOfficer)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You promote <ansi fg="username">%s</ansi> to officer.`, name))
	} else {
		if cur != guilds.RankOfficer {
			user.SendText(messaging.CategorySystem, `That member is not an officer.`)
			return
		}
		guilds.SetRank(g.Tag, targetId, guilds.RankMember)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You demote <ansi fg="username">%s</ansi> to member.`, name))
	}
	if tu := users.GetByUserId(targetId); tu != nil {
		tu.SendText(messaging.CategorySystem, `Your guild rank has changed.`)
	}
}

func guildTransfer(user *users.UserRecord, remainder, rest string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the guild leader can transfer leadership.`)
		return
	}
	targetId, found := findGuildMemberByName(g, remainder)
	if !found || targetId == user.UserId {
		user.SendText(messaging.CategorySystem, `Name another guild member to lead.`)
		return
	}
	name := memberName(g, targetId)
	cmdPrompt, _ := user.StartPrompt(`guild`, rest)
	q := cmdPrompt.Ask(fmt.Sprintf(`Hand leadership of %s to %s?`, g.Name, name), []string{`Yes`, `No`}, `No`)
	if !q.Done {
		return
	}
	user.ClearPrompt()
	if len(q.Response) == 0 || q.Response[0:1] != `Y` {
		user.SendText(messaging.CategorySystem, `Cancelled.`)
		return
	}
	guilds.TransferLeader(g.Tag, targetId)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You hand leadership to <ansi fg="username">%s</ansi> and become an officer.`, name))
	if g2, ok := guilds.Get(g.Tag); ok {
		announceGuild(g2, fmt.Sprintf(`<ansi fg="username">%s</ansi> is now the guild leader.`, name), user.UserId)
	}
}

func guildSetMotd(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.CanManage(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only officers and the leader can set the MOTD.`)
		return
	}
	text := strings.TrimSpace(remainder)
	guilds.SetMotd(g.Tag, text)
	if text == "" {
		user.SendText(messaging.CategorySystem, `You clear the guild's message of the day.`)
	} else {
		user.SendText(messaging.CategorySystem, `You set the guild's message of the day.`)
	}
}
