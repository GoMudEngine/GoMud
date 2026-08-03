package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
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
	case "chat", "say", "gc":
		guildChatSend(user, remainder)
	case "deposit":
		guildDeposit(user, remainder)
	case "withdraw":
		guildWithdraw(user, remainder)
	case "donate":
		guildDonate(user, remainder)
	case "take":
		guildTake(user, remainder)
	case "treasury", "bank", "vault":
		guildTreasury(user, remainder)
	case "title":
		guildSetTitle(user, remainder)
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
		// Escaped again on render (idempotent) so a MOTD persisted before
		// guilds.SetMotd started escaping is neutralised too.
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">"%s"</ansi>`, util.EscapeAnsiTags(g.Motd)))
	}
	// Roster: leader, then officers, then members.
	for _, rank := range []guilds.GuildRank{guilds.RankLeader, guilds.RankOfficer, guilds.RankMember} {
		for _, m := range g.Members {
			if m.Rank == rank {
				user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="white">%-20s</ansi> <ansi fg="black-bold">%s</ansi>`, m.CharacterName, g.RankTitle(m.Rank)))
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
	if _, ok := guilds.GuildWithInvite(user.UserId); !ok {
		user.SendText(messaging.CategorySystem, `You have no pending guild invitation.`)
		return
	}
	guilds.ClearInvites(user.UserId) // clear every pending invite deterministically
	user.SendText(messaging.CategorySystem, `You decline your pending guild invitation(s).`)
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
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You promote <ansi fg="username">%s</ansi> to %s.`, name, g.RankTitle(guilds.RankOfficer)))
	} else {
		if cur != guilds.RankOfficer {
			user.SendText(messaging.CategorySystem, `That member is not an officer.`)
			return
		}
		guilds.SetRank(g.Tag, targetId, guilds.RankMember)
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You demote <ansi fg="username">%s</ansi> to %s.`, name, g.RankTitle(guilds.RankMember)))
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

// guildChatRecipients returns the member userIds to deliver a guild-chat line to
// (all members except the sender). The caller filters offline members.
func guildChatRecipients(g *guilds.Guild, senderId int) []int {
	out := []int{}
	for _, m := range g.Members {
		if m.UserId != senderId {
			out = append(out, m.UserId)
		}
	}
	return out
}

// guildChatSend broadcasts a guild-chat line to online members, echoes to the
// sender, and emits a Communication event for the web/GMCP comm tab.
func guildChatSend(user *users.UserRecord, msg string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if user.Muted {
		user.SendText(messaging.CategoryWarning, `You are <ansi fg="alert-5">MUTED</ansi>.`)
		return
	}
	msg = strings.TrimSpace(msg)
	if msg == "" {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild chat &lt;message&gt;</ansi>  (or <ansi fg="command">gc &lt;message&gt;</ansi>)`)
		return
	}
	// Neutralise <ansi> markup before interpolation.
	msg = util.EscapeAnsiTags(msg)

	for _, uid := range guildChatRecipients(g, user.UserId) {
		if u := users.GetByUserId(uid); u != nil {
			line := fmt.Sprintf(`<ansi fg="cyan">(guild)</ansi> <ansi fg="username">%s</ansi>: <ansi fg="white">%s</ansi>`, user.Character.Name, msg)
			u.SendText(messaging.CategorySystem, util.SplitStringNL(line, 80))
		}
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="cyan">(guild)</ansi> You: <ansi fg="white">%s</ansi>`, msg))
	events.AddToQueue(events.Communication{
		SourceUserId: user.UserId,
		CommType:     `guild`,
		Name:         user.Character.Name,
		Message:      msg,
	})
}

// Gc is the top-level guild-chat alias for `guild chat`.
func Gc(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	guildChatSend(user, rest)
	return true, nil
}

// findVaultItem fuzzy-matches a vault item by name and returns its index.
func findVaultItem(vault []items.Item, name string) (int, bool) {
	if strings.TrimSpace(name) == "" || len(vault) == 0 {
		return 0, false
	}
	closeMatch, fullMatch := items.FindMatchIn(name, vault...)
	target := fullMatch
	if target.ItemId == 0 {
		target = closeMatch
	}
	if target.ItemId == 0 {
		return 0, false
	}
	for i, it := range vault {
		if it.Equals(target) {
			return i, true
		}
	}
	return 0, false
}

// parseGoldAmount resolves an amount string ("all" or a positive integer). ok is
// false for empty/non-positive/non-numeric input other than "all".
func parseGoldAmount(s string, allValue int) (amount int, isAll, ok bool) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, false, false
	}
	if s == "all" {
		return allValue, true, true
	}
	n, err := strconv.Atoi(s)
	if err != nil || n < 1 {
		return 0, false, false
	}
	return n, false, true
}

func guildDeposit(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	amount, _, ok := parseGoldAmount(remainder, user.Character.Bank)
	if !ok {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild deposit <amount|all></ansi>  (gold comes from your bank).`)
		return
	}
	if amount > user.Character.Bank {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You only have <ansi fg="gold">%d gold</ansi> in the bank.`, user.Character.Bank))
		return
	}
	if amount <= 0 {
		user.SendText(messaging.CategorySystem, `Deposit a positive amount.`)
		return
	}
	if err := guilds.DepositGold(g.Tag, amount); err != nil {
		user.SendText(messaging.CategorySystem, `Cannot deposit: `+err.Error())
		return
	}
	user.Character.Bank -= amount
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, BankChange: -amount})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You deposit <ansi fg="gold">%d gold</ansi> into the guild treasury.`, amount))
	announceGuild(g, fmt.Sprintf(`<ansi fg="username">%s</ansi> deposited <ansi fg="gold">%d gold</ansi> into the guild treasury.`, user.Character.Name, amount), user.UserId)
}

func guildWithdraw(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.CanWithdraw(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the leader can withdraw (unless the leader has delegated treasury access to officers).`)
		return
	}
	amount, _, ok := parseGoldAmount(remainder, g.Treasury)
	if !ok {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild withdraw <amount|all></ansi>  (gold goes to your bank).`)
		return
	}
	if amount <= 0 {
		user.SendText(messaging.CategorySystem, `The treasury is empty.`)
		return
	}
	// WithdrawGold rejects (without mutating) if amount exceeds the treasury.
	if err := guilds.WithdrawGold(g.Tag, amount); err != nil {
		user.SendText(messaging.CategorySystem, `Cannot withdraw: `+err.Error())
		return
	}
	user.Character.Bank += amount
	events.AddToQueue(events.EquipmentChange{UserId: user.UserId, BankChange: amount})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You withdraw <ansi fg="gold">%d gold</ansi> from the guild treasury to your bank.`, amount))
	announceGuild(g, fmt.Sprintf(`<ansi fg="username">%s</ansi> withdrew <ansi fg="gold">%d gold</ansi> from the guild treasury.`, user.Character.Name, amount), user.UserId)
}

func guildDonate(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	name := strings.TrimSpace(remainder)
	if name == "" {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild donate <item></ansi>`)
		return
	}
	item, found := user.Character.FindInBackpack(name)
	if !found {
		user.SendText(messaging.CategorySystem, `You don't have that item.`)
		return
	}
	itemName := item.DisplayName()
	// DonateItem errors (without mutating) if the vault is full. Only strip the
	// item from the donor after it has landed in the vault.
	if err := guilds.DonateItem(g.Tag, item); err != nil {
		user.SendText(messaging.CategorySystem, `Cannot donate: `+err.Error())
		return
	}
	user.Character.RemoveItem(item)
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: item, Gained: false})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You donate %s to the guild vault.`, itemName))
	announceGuild(g, fmt.Sprintf(`<ansi fg="username">%s</ansi> donated %s to the guild vault.`, user.Character.Name, itemName), user.UserId)
}

func guildTake(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.CanWithdraw(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the leader can take from the vault (unless the leader has delegated treasury access to officers).`)
		return
	}
	name := strings.TrimSpace(remainder)
	idx, found := findVaultItem(g.Vault, name)
	if !found {
		user.SendText(messaging.CategorySystem, `No such item in the guild vault. See <ansi fg="command">guild treasury</ansi>.`)
		return
	}
	// Item-loss guard: prove the item fits in the taker's pack BEFORE removing it
	// from the vault, so a full pack never destroys a vault item.
	it := g.Vault[idx]
	if !user.Character.StoreItem(it) {
		user.SendText(messaging.CategorySystem, `Your pack is too full to carry that.`)
		return
	}
	itemName := it.DisplayName()
	if _, err := guilds.TakeItem(g.Tag, idx); err != nil {
		// Vault changed under us; undo the store so nothing is duplicated.
		user.Character.RemoveItem(it)
		user.SendText(messaging.CategorySystem, `Cannot take that item right now.`)
		return
	}
	events.AddToQueue(events.ItemOwnership{UserId: user.UserId, Item: it, Gained: true})
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`You take %s from the guild vault.`, itemName))
	announceGuild(g, fmt.Sprintf(`<ansi fg="username">%s</ansi> took %s from the guild vault.`, user.Character.Name, itemName), user.UserId)
}

func guildTreasury(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	// `guild treasury delegate [on|off]` — leader-only.
	if fields := strings.Fields(strings.ToLower(remainder)); len(fields) > 0 && fields[0] == "delegate" {
		if !g.IsLeader(user.UserId) {
			user.SendText(messaging.CategorySystem, `Only the leader can delegate treasury access.`)
			return
		}
		on := !g.TreasuryDelegated
		if len(fields) > 1 {
			switch fields[1] {
			case "on", "yes", "true":
				on = true
			case "off", "no", "false":
				on = false
			}
		}
		guilds.SetTreasuryDelegated(g.Tag, on)
		if on {
			user.SendText(messaging.CategorySystem, `Officers may now withdraw gold and take items from the vault.`)
			announceGuild(g, `The leader has granted officers treasury access.`, user.UserId)
		} else {
			user.SendText(messaging.CategorySystem, `Only you may now withdraw gold and take items from the vault.`)
			announceGuild(g, `The leader has revoked officers' treasury access.`, user.UserId)
		}
		return
	}

	capacity := int(configs.GetBalanceConfig().GuildVaultCapacity)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`<ansi fg="yellow-bold">%s</ansi> treasury: <ansi fg="gold">%d gold</ansi>`, g.Name, g.Treasury))
	if len(g.Vault) == 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Vault: empty (0/%d).`, capacity))
	} else {
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`Vault (%d/%d):`, len(g.Vault), capacity))
		for i, it := range g.Vault {
			user.SendText(messaging.CategorySystem, fmt.Sprintf(`  <ansi fg="black-bold">%2d.</ansi> %s`, i+1, it.DisplayName()))
		}
	}
	if g.TreasuryDelegated {
		user.SendText(messaging.CategorySystem, `Treasury access: leader and officers.`)
	} else {
		user.SendText(messaging.CategorySystem, `Treasury access: leader only.`)
	}
}

// parseGuildRank resolves a case-insensitive rank name to a GuildRank.
func parseGuildRank(s string) (guilds.GuildRank, bool) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "member":
		return guilds.RankMember, true
	case "officer":
		return guilds.RankOfficer, true
	case "leader":
		return guilds.RankLeader, true
	}
	return "", false
}

func guildSetTitle(user *users.UserRecord, remainder string) {
	g, ok := guilds.GetByUser(user.UserId)
	if !ok {
		user.SendText(messaging.CategorySystem, `You are not in a guild.`)
		return
	}
	if !g.IsLeader(user.UserId) {
		user.SendText(messaging.CategorySystem, `Only the guild leader can rename ranks.`)
		return
	}
	fields := strings.Fields(remainder)
	if len(fields) == 0 {
		user.SendText(messaging.CategorySystem, `Usage: <ansi fg="command">guild title <member|officer|leader> [new name]</ansi>  (omit the name to reset).`)
		return
	}
	rank, ok := parseGuildRank(fields[0])
	if !ok {
		user.SendText(messaging.CategorySystem, `Name a rank: <ansi fg="white">member</ansi>, <ansi fg="white">officer</ansi>, or <ansi fg="white">leader</ansi>.`)
		return
	}
	title := strings.TrimSpace(strings.Join(fields[1:], " "))
	if title == "" {
		guilds.SetRankTitle(g.Tag, rank, "")
		user.SendText(messaging.CategorySystem, fmt.Sprintf(`You reset the <ansi fg="white">%s</ansi> rank to its default name.`, rank))
		return
	}
	if err := guilds.ValidRankTitle(title); err != nil {
		user.SendText(messaging.CategorySystem, `Invalid title: `+err.Error())
		return
	}
	guilds.SetRankTitle(g.Tag, rank, title)
	user.SendText(messaging.CategorySystem, fmt.Sprintf(`Members of <ansi fg="white">%s</ansi> rank are now titled <ansi fg="cyan">%s</ansi>.`, rank, title))
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
