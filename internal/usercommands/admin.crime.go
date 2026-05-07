package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/crimes"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/factions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * crime          (Admin)
 */

func Crime(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		crimeShowUsage(user)
		return true, nil
	}

	// Detect --all flag (anywhere in args).
	includeResolved := false
	cleaned := make([]string, 0, len(args))
	for _, a := range args {
		if a == "--all" {
			includeResolved = true
			continue
		}
		cleaned = append(cleaned, a)
	}
	if len(cleaned) == 0 {
		crimeShowUsage(user)
		return true, nil
	}

	switch strings.ToLower(cleaned[0]) {
	case "list":
		return crimeList(cleaned[1:], user, includeResolved)
	case "show":
		return crimeShow(cleaned[1:], user, includeResolved)
	case "resolve":
		return crimeResolve(cleaned[1:], user)
	case "prune-stale":
		return crimePruneStale(cleaned[1:], user)
	default:
		crimeShowUsage(user)
		return true, nil
	}
}

func crimeShowUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.crime", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  crime list <factionId> [--all]\r\n" +
			"  crime show <playerName> [--all]\r\n" +
			"  crime resolve <factionId> <crimeId> <reason>\r\n" +
			"  crime prune-stale <factionId>\r\n",
	)
}

func crimeList(args []string, user *users.UserRecord, includeResolved bool) (bool, error) {
	if len(args) != 1 {
		crimeShowUsage(user)
		return true, nil
	}
	factionId := args[0]
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	rows := crimes.AllForFaction(factionId, includeResolved)
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No crimes for %s.\r\n", factionId))
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Round < rows[j].Round })

	var b strings.Builder
	scope := "unresolved"
	if includeResolved {
		scope = "rows"
	}
	fmt.Fprintf(&b, "crime list %s (%d %s):\r\n\r\n", factionId, len(rows), scope)
	fmt.Fprintf(&b, "  %4s  %-8s  %12s  %-15s  %-22s  %-22s  %s\r\n",
		"ID", "Kind", "Round", "Where", "Victim", "Perp", "Resolved")
	fmt.Fprintf(&b, "  %4s  %-8s  %12s  %-15s  %-22s  %-22s  %s\r\n",
		"----", "--------", "------------", "---------------", "----------------------", "----------------------", "----------")
	for _, c := range rows {
		fmt.Fprintf(&b, "  %4d  %-8s  %12d  room %-9d  %-22s  %-22s  %s\r\n",
			c.Id, string(c.Kind), c.Round, c.RoomId,
			fmt.Sprintf("mob %d (inst %d)", c.VictimMobId, c.VictimInstanceId),
			perpString(c.Perpetrator),
			resolvedString(c))
	}
	user.SendText(b.String())
	return true, nil
}

// playerCrimeRow pairs a Crime with the faction whose log it lives
// in. crimeShow walks every loaded faction and accumulates rows for
// the target player so the display can show which faction owns each
// row (multi-faction kills produce one row per faction).
type playerCrimeRow struct {
	FactionId string
	Crime     *crimes.Crime
}

func crimeShow(args []string, user *users.UserRecord, includeResolved bool) (bool, error) {
	if len(args) != 1 {
		crimeShowUsage(user)
		return true, nil
	}
	target := users.GetByCharacterName(args[0])
	if target == nil {
		user.SendText(fmt.Sprintf("No such player: %s\r\n", args[0]))
		return true, nil
	}

	var rows []playerCrimeRow
	for _, def := range factions.AllDefinitions() {
		for _, c := range crimes.AllForFaction(def.FactionId, includeResolved) {
			if c.Perpetrator.Type != crimes.PerpPlayer || c.Perpetrator.Id != target.UserId {
				continue
			}
			rows = append(rows, playerCrimeRow{FactionId: def.FactionId, Crime: c})
		}
	}
	if len(rows) == 0 {
		user.SendText(fmt.Sprintf("No crimes recorded for %s.\r\n", args[0]))
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Crime.Round < rows[j].Crime.Round })

	var b strings.Builder
	scope := "unresolved"
	if includeResolved {
		scope = "rows"
	}
	fmt.Fprintf(&b, "crime show %s (%d %s):\r\n\r\n", args[0], len(rows), scope)
	fmt.Fprintf(&b, "  %-22s  %4s  %-8s  %12s  %-22s  %s\r\n",
		"Faction", "ID", "Kind", "Round", "Where", "Resolved")
	fmt.Fprintf(&b, "  %-22s  %4s  %-8s  %12s  %-22s  %s\r\n",
		"----------------------", "----", "--------", "------------", "----------------------", "----------")
	for _, r := range rows {
		c := r.Crime
		fmt.Fprintf(&b, "  %-22s  %4d  %-8s  %12d  %-22s  %s\r\n",
			factionTruncate(r.FactionId, 22),
			c.Id, string(c.Kind), c.Round,
			fmt.Sprintf("room %d in %s", c.RoomId, c.Zone),
			resolvedString(c))
	}
	user.SendText(b.String())
	return true, nil
}

// resolvedString returns "no" for unresolved crimes, otherwise the
// resolved_by reason. Used in both crime list and crime show.
func resolvedString(c *crimes.Crime) string {
	if c.ResolvedRound == 0 {
		return "no"
	}
	if c.ResolvedBy == "" {
		return "yes"
	}
	return c.ResolvedBy
}

func crimeResolve(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		crimeShowUsage(user)
		return true, nil
	}
	factionId := args[0]
	crimeId, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad crime id %q: %v\r\n", args[1], err))
		return true, nil
	}
	reason := strings.Join(args[2:], " ")
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	crimes.Resolve(factionId, crimeId, reason)
	user.SendText(fmt.Sprintf("Resolved %s crime %d: %s\r\n", factionId, crimeId, reason))
	return true, nil
}

func crimePruneStale(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 1 {
		crimeShowUsage(user)
		return true, nil
	}
	factionId := args[0]
	if factions.GetDefinition(factionId) == nil {
		user.SendText(fmt.Sprintf("Unknown faction: %s\r\n", factionId))
		return true, nil
	}
	count := crimes.PruneStale(factionId)
	user.SendText(fmt.Sprintf("Resolved %d stale crime(s) for %s.\r\n", count, factionId))
	return true, nil
}

func perpString(p crimes.Perpetrator) string {
	switch p.Type {
	case crimes.PerpPlayer:
		if u := users.GetByUserId(p.Id); u != nil {
			return fmt.Sprintf("%s (%d)", u.Character.Name, p.Id)
		}
		return fmt.Sprintf("player %d", p.Id)
	case crimes.PerpMob:
		return fmt.Sprintf("mob %d", p.Id)
	default:
		return "unknown"
	}
}
