package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * faction          (Admin)
 */

// Faction is the admin command for inspecting and adjusting per-faction
// reputation scores. Subcommands:
//
//	faction list
//	faction show <playerName>
//	faction show <factionId> <playerName>
//	faction set <factionId> <playerName> <rep>
//	faction bump <factionId> <playerName> <delta>
//	faction reset <factionId> <playerName>
func Faction(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		factionShowUsage(user)
		return true, nil
	}

	switch strings.ToLower(args[0]) {
	case "list":
		return factionListAll(user)
	case "show":
		return factionShow(args[1:], user)
	case "set":
		return factionMutate(args[1:], user, factionMutateSet)
	case "bump":
		return factionMutate(args[1:], user, factionMutateBump)
	case "reset":
		return factionMutate(args[1:], user, factionMutateReset)
	default:
		factionShowUsage(user)
		return true, nil
	}
}

func factionShowUsage(user *users.UserRecord) {
	// Try templated help first; fall back to inline if missing.
	if out, err := templates.Process("admincommands/help/command.faction", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  faction list\r\n" +
			"  faction show <playerName>\r\n" +
			"  faction show <factionId> <playerName>\r\n" +
			"  faction set <factionId> <playerName> <rep>\r\n" +
			"  faction bump <factionId> <playerName> <delta>\r\n" +
			"  faction reset <factionId> <playerName>\r\n",
	)
}

func factionListAll(user *users.UserRecord) (bool, error) {
	defs := factions.AllDefinitions()
	if len(defs) == 0 {
		user.SendText("No factions loaded.\r\n")
		return true, nil
	}
	sort.Slice(defs, func(i, j int) bool { return defs[i].FactionId < defs[j].FactionId })

	var b strings.Builder
	fmt.Fprintf(&b, "Loaded factions:\r\n\r\n")
	fmt.Fprintf(&b, "  %-25s %-25s %-7s %s\r\n", "ID", "Name", "Default", "Allies / Enemies")
	fmt.Fprintf(&b, "  %-25s %-25s %-7s %s\r\n",
		strings.Repeat("-", 25), strings.Repeat("-", 25), "-------", strings.Repeat("-", 30))
	for _, d := range defs {
		fmt.Fprintf(&b, "  %-25s %-25s %-7d allies=%v enemies=%v\r\n",
			factionTruncate(d.FactionId, 25),
			factionTruncate(d.DisplayName, 25),
			d.DefaultRep,
			d.Allies, d.Enemies)
	}
	user.SendText(b.String())
	return true, nil
}

func factionShow(args []string, user *users.UserRecord) (bool, error) {
	switch len(args) {
	case 1:
		return factionShowAllForUser(args[0], user)
	case 2:
		return factionShowOne(args[0], args[1], user)
	default:
		factionShowUsage(user)
		return true, nil
	}
}

func factionShowAllForUser(playerName string, user *users.UserRecord) (bool, error) {
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}

	defs := factions.AllDefinitions()
	type row struct {
		factionId string
		rep       int
	}
	rows := make([]row, 0)
	for _, d := range defs {
		rep := factions.GetRep(d.FactionId, target.UserId)
		if rep == d.DefaultRep {
			continue
		}
		rows = append(rows, row{d.FactionId, rep})
	}
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No factions hold a non-default rep for %s.\r\n", playerName))
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].rep < rows[j].rep })

	var b strings.Builder
	fmt.Fprintf(&b, "faction show %s:\r\n\r\n", playerName)
	fmt.Fprintf(&b, "  %-25s %5s  %s\r\n", "Faction", "Rep", "Tier")
	fmt.Fprintf(&b, "  %-25s %5s  %s\r\n", strings.Repeat("-", 25), "-----", "--------")
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-25s %5d  %s\r\n",
			factionTruncate(r.factionId, 25),
			r.rep, factionTierName(opinions.TierOf(r.rep)))
	}
	user.SendText(b.String())
	return true, nil
}

func factionShowOne(factionId, playerName string, user *users.UserRecord) (bool, error) {
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	target := users.GetByCharacterName(playerName)
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}
	rep := factions.GetRep(factionId, target.UserId)
	user.SendText(fmt.Sprintf("%s -> %s: rep=%d, tier=%s\r\n",
		factionId, playerName, rep, factionTierName(opinions.TierOf(rep))))
	return true, nil
}

type factionMutateMode int

const (
	factionMutateSet factionMutateMode = iota
	factionMutateBump
	factionMutateReset
)

func factionMutate(args []string, user *users.UserRecord, mode factionMutateMode) (bool, error) {
	expected := 3
	if mode == factionMutateReset {
		expected = 2
	}
	if len(args) != expected {
		factionShowUsage(user)
		return true, nil
	}
	factionId := args[0]
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	target := users.GetByCharacterName(args[1])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}

	switch mode {
	case factionMutateSet:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad rep %q: %v\r\n", args[2], err))
			return true, nil
		}
		factions.SetRep(factionId, target.UserId, v)
		user.SendText(fmt.Sprintf("Set %s -> %s = %d\r\n", factionId, args[1], v))
	case factionMutateBump:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(fmt.Sprintf("Bad delta %q: %v\r\n", args[2], err))
			return true, nil
		}
		factions.BumpRep(factionId, target.UserId, v)
		user.SendText(fmt.Sprintf("Bumped %s -> %s by %d (now %d)\r\n",
			factionId, args[1], v, factions.GetRep(factionId, target.UserId)))
	case factionMutateReset:
		def := factions.GetDefinition(factionId)
		factions.SetRep(factionId, target.UserId, def.DefaultRep)
		user.SendText(fmt.Sprintf("Reset %s -> %s to default %d\r\n",
			factionId, args[1], def.DefaultRep))
	}
	return true, nil
}

func factionTierName(t opinions.Tier) string {
	switch t {
	case opinions.TierHostile:
		return "Hostile"
	case opinions.TierCold:
		return "Cold"
	case opinions.TierNeutral:
		return "Neutral"
	case opinions.TierWarm:
		return "Warm"
	case opinions.TierFriendly:
		return "Friendly"
	default:
		return "?"
	}
}

func factionTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "~"
}
