package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
 * Role Permissions:
 * bounty         (Admin)
 */

func Bounty(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	isAdmin := user.HasRolePermission("bounty", true)
	args := strings.Fields(rest)
	if len(args) == 0 {
		if isAdmin {
			bountyAdminUsage(user)
		} else {
			bountyUserUsage(user)
		}
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return bountyAdminList(args[1:], user)
	case "show":
		return bountyAdminShow(args[1:], user)
	case "declare", "withdraw", "prune-expired":
		if !isAdmin {
			user.SendText(messaging.CategorySystem, "That subcommand is admin-only.\r\n")
			return true, nil
		}
		switch strings.ToLower(args[0]) {
		case "declare":
			return bountyAdminDeclare(args[1:], user)
		case "withdraw":
			return bountyAdminWithdraw(args[1:], user)
		case "prune-expired":
			return bountyAdminPrune(user)
		}
	default:
		if isAdmin {
			bountyAdminUsage(user)
		} else {
			bountyUserUsage(user)
		}
		return true, nil
	}
	return true, nil
}

func bountyAdminUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.bounty", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem, 
		"Usage:\r\n" +
			"  bounty list [--all] [filter]\r\n" +
			"  bounty show <id>\r\n" +
			"  bounty declare <issuer-spec> <target-spec> [--gold N] [--rep N] [--expiry-rounds N] [--reason \"...\"]\r\n" +
			"  bounty withdraw <id>\r\n" +
			"  bounty prune-expired\r\n" +
			"\r\n" +
			"Issuer/target spec: <type>:<id>  e.g.  faction:thornwall_guards  player:17  mob:101  quest:14\r\n",
	)
}

func bountyUserUsage(user *users.UserRecord) {
	if out, err := templates.Process("help/bounty", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem, 
		"Usage:\r\n" +
			"  bounty list [filter]\r\n" +
			"  bounty show <id>\r\n" +
			"\r\n" +
			"Filter: mob, player, or a faction name (e.g. thornwall_guards)\r\n",
	)
}

func bountyAdminList(args []string, user *users.UserRecord) (bool, error) {
	includeNonOpen := false
	filterStr := ""
	for _, a := range args {
		if a == "--all" {
			includeNonOpen = true
			continue
		}
		if filterStr == "" && !strings.HasPrefix(a, "--") {
			filterStr = strings.ToLower(a)
		}
	}
	var rows []*bounties.Bounty
	if includeNonOpen {
		rows = bounties.AllRows()
	} else {
		rows = bounties.AllOpen()
	}
	if filterStr != "" {
		filtered := make([]*bounties.Bounty, 0, len(rows))
		for _, r := range rows {
			switch filterStr {
			case "mob", "player":
				if string(r.Target.Type) == filterStr {
					filtered = append(filtered, r)
				}
			default:
				if r.Issuer.Type == bounties.IssuerFaction && r.Issuer.Id == filterStr {
					filtered = append(filtered, r)
				}
			}
		}
		rows = filtered
	}
	if len(rows) == 0 {
		user.SendText(messaging.CategorySystem, "No bounties.\r\n")
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].GoldReward > rows[j].GoldReward })

	var b strings.Builder
	fmt.Fprintf(&b, "bounty list (%d):\r\n\r\n", len(rows))
	fmt.Fprintf(&b, "  %4s  %-10s  %-22s  %-15s  %5s  %4s  %s\r\n",
		"ID", "Status", "Issuer", "Target", "Gold", "Rep", "Reason")
	fmt.Fprintf(&b, "  %4s  %-10s  %-22s  %-15s  %5s  %4s  %s\r\n",
		"----", "----------", "----------------------", "---------------", "-----", "----", "----------")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %4d  %-10s  %-22s  %-15s  %5d  %4d  %s\r\n",
			r.Id, r.Status,
			bountyTruncate(fmt.Sprintf("%s:%s", r.Issuer.Type, r.Issuer.Id), 22),
			bountyTruncate(bountyTargetLabel(r.Target), 15),
			r.GoldReward, r.RepReward, r.DeclaredReason)
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func bountyAdminShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		if user.HasRolePermission("bounty", true) {
			bountyAdminUsage(user)
		} else {
			bountyUserUsage(user)
		}
		return true, nil
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad bounty id %q\r\n", args[0]))
		return true, nil
	}
	b := bounties.Get(id)
	if b == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No bounty with id %d\r\n", id))
		return true, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "Bounty #%d (%s)\r\n", b.Id, b.Status)
	fmt.Fprintf(&sb, "  Issuer:    %s:%s\r\n", b.Issuer.Type, b.Issuer.Id)
	fmt.Fprintf(&sb, "  Target:    %s\r\n", bountyTargetLabel(b.Target))
	fmt.Fprintf(&sb, "  Reward:    %dg + %d rep\r\n", b.GoldReward, b.RepReward)
	fmt.Fprintf(&sb, "  Condition: %s\r\n", b.Condition)
	fmt.Fprintf(&sb, "  Declared:  round %d (reason: %q)\r\n", b.DeclaredRound, b.DeclaredReason)
	if b.ExpiryRound > 0 {
		fmt.Fprintf(&sb, "  Expires:   round %d\r\n", b.ExpiryRound)
	} else {
		fmt.Fprintf(&sb, "  Expires:   never\r\n")
	}
	if b.Status == bounties.StatusClaimed {
		fmt.Fprintf(&sb, "  Claimed:   round %d by %s:%d\r\n",
			b.ClaimedRound, b.ClaimedBy.Type, b.ClaimedBy.Id)
	}
	user.SendText(messaging.CategorySystem, sb.String())
	return true, nil
}

func bountyAdminDeclare(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		bountyAdminUsage(user)
		return true, nil
	}
	issuer, ok := parseIssuerSpec(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad issuer spec %q (use type:id)\r\n", args[0]))
		return true, nil
	}
	target, ok := parseTargetSpec(args[1])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad target spec %q (use type:id)\r\n", args[1]))
		return true, nil
	}
	opts := bounties.DeclareOpts{}
	expiryRounds := uint64(0)
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--gold":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.GoldOverride = v
				}
			}
		case "--rep":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.RepOverride = v
				}
			}
		case "--expiry-rounds":
			i++
			if i < len(args) {
				if v, err := strconv.ParseUint(args[i], 10, 64); err == nil {
					expiryRounds = v
				}
			}
		case "--reason":
			// Consume the rest as the reason (handles spaces).
			opts.DeclaredReason = strings.Join(args[i+1:], " ")
			i = len(args)
		}
	}
	expiryRound := uint64(0)
	if expiryRounds > 0 {
		expiryRound = util.GetRoundCount() + expiryRounds
	}
	id, err := bounties.Declare(issuer, target, bounties.ConditionKill, expiryRound, opts)
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Declare failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Declared bounty #%d.\r\n", id))
	return true, nil
}

func bountyAdminWithdraw(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		bountyAdminUsage(user)
		return true, nil
	}
	id, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad bounty id %q\r\n", args[0]))
		return true, nil
	}
	bounties.Withdraw(id)
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Withdrew bounty #%d.\r\n", id))
	return true, nil
}

func bountyAdminPrune(user *users.UserRecord) (bool, error) {
	count := bounties.PruneExpired()
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Pruned %d expired bounty rows.\r\n", count))
	return true, nil
}

// parseIssuerSpec parses "<type>:<id>" → Issuer.
func parseIssuerSpec(s string) (bounties.Issuer, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return bounties.Issuer{}, false
	}
	switch strings.ToLower(parts[0]) {
	case "faction":
		return bounties.FactionIssuer(parts[1]), true
	case "quest":
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return bounties.Issuer{}, false
		}
		return bounties.QuestIssuer(id), true
	case "npc":
		id, err := strconv.Atoi(parts[1])
		if err != nil {
			return bounties.Issuer{}, false
		}
		return bounties.NPCIssuer(id), true
	}
	return bounties.Issuer{}, false
}

// bountyTargetLabel renders a bounty target for player-visible output,
// resolving player/mob names instead of raw "<type>:<id>" ids (5.1b smoke
// CONCERN-1). Falls back to the id form when the name can't be resolved.
func bountyTargetLabel(s knowledge.Subject) string {
	switch s.Type {
	case knowledge.SubjectPlayer:
		if u := users.GetByUserId(s.Id); u != nil && u.Character != nil && u.Character.Name != "" {
			return u.Character.Name
		}
		return fmt.Sprintf("player:%d", s.Id)
	case knowledge.SubjectMob:
		if spec := mobs.GetMobSpec(mobs.MobId(s.Id)); spec != nil && spec.Character.Name != "" {
			return spec.Character.Name
		}
		return fmt.Sprintf("mob:%d", s.Id)
	default:
		return fmt.Sprintf("%s:%d", s.Type, s.Id)
	}
}

// parseTargetSpec parses "<type>:<id>" → knowledge.Subject.
func parseTargetSpec(s string) (knowledge.Subject, bool) {
	parts := strings.SplitN(s, ":", 2)
	if len(parts) != 2 {
		return knowledge.Subject{}, false
	}
	id, err := strconv.Atoi(parts[1])
	if err != nil {
		return knowledge.Subject{}, false
	}
	switch strings.ToLower(parts[0]) {
	case "player":
		return knowledge.PlayerSubject(id), true
	case "mob":
		return knowledge.MobSubject(id), true
	}
	return knowledge.Subject{}, false
}

// bountyTruncate shortens s to at most maxLen runes, appending "…" if trimmed.
func bountyTruncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen-1]) + "…"
}
