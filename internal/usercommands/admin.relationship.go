package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
)

/*
 * Role Permissions:
 * relationship  (Admin)
 */

func Relationship(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		relationshipUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "show":
		return relationshipShow(args[1:], user)
	case "between":
		return relationshipBetween(args[1:], user)
	case "add":
		return relationshipAdd(args[1:], user)
	case "remove":
		return relationshipRemove(args[1:], user)
	case "list":
		return relationshipList(user)
	default:
		relationshipUsage(user)
	}
	return true, nil
}

func relationshipUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.relationship", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(out)
		return
	}
	user.SendText(
		"Usage:\r\n" +
			"  relationship show <mobId>\r\n" +
			"  relationship between <mobIdA> <mobIdB>\r\n" +
			"  relationship add <mobIdA> <mobIdB> <type> [subtype]\r\n" +
			"  relationship remove <mobIdA> <mobIdB> <type>\r\n" +
			"  relationship list\r\n" +
			"\r\n" +
			"<type> ∈ family | friend | rival | lover | employer | employee\r\n",
	)
}

func relationshipShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		relationshipUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	rels := relationships.RelationsOf(mobId)
	if len(rels) == 0 {
		user.SendText(fmt.Sprintf("No relationships for mob %d.\r\n", mobId))
		return true, nil
	}
	var b strings.Builder
	mobName := relationshipMobNameFor(mobId)
	fmt.Fprintf(&b, "Relationships for mob %d (%s):\r\n", mobId, mobName)
	for _, r := range rels {
		typeLabel := string(r.Type)
		if r.Type == relationships.TypeEmployer {
			typeLabel = "employer-of"
		} else if r.Type == relationships.TypeEmployee {
			typeLabel = "employed-by"
		}
		other := relationshipMobNameFor(r.Other)
		subLabel := ""
		if r.Subtype != "" {
			subLabel = fmt.Sprintf(" [%s]", r.Subtype)
		}
		fmt.Fprintf(&b, "  %-12s → mob %d (%s)%s\r\n", typeLabel, r.Other, other, subLabel)
	}
	user.SendText(b.String())
	return true, nil
}

func relationshipBetween(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	aSide := relationships.RelationsBetween(a, b)
	bSide := relationships.RelationsBetween(b, a)
	if len(aSide) == 0 && len(bSide) == 0 {
		user.SendText(fmt.Sprintf("No relationship between mob %d and mob %d.\r\n", a, b))
		return true, nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d (%s) ↔ %d (%s):\r\n", a, relationshipMobNameFor(a), b, relationshipMobNameFor(b))
	for _, r := range aSide {
		sub := ""
		if r.Subtype != "" {
			sub = fmt.Sprintf(" subtype: %s", r.Subtype)
		}
		fmt.Fprintf(&sb, "  %-12s (%d→%d%s)\r\n", r.Type, a, b, sub)
	}
	for _, r := range bSide {
		sub := ""
		if r.Subtype != "" {
			sub = fmt.Sprintf(" subtype: %s", r.Subtype)
		}
		fmt.Fprintf(&sb, "  %-12s (%d→%d%s)\r\n", r.Type, b, a, sub)
	}
	user.SendText(sb.String())
	return true, nil
}

func relationshipAdd(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	t := relationships.Type(strings.ToLower(args[2]))
	subtype := ""
	if len(args) >= 4 {
		subtype = strings.Join(args[3:], " ")
	}
	relationships.Add(a, b, t, subtype)
	user.SendText(fmt.Sprintf("Added: %d → %d (%s).\r\n", a, b, t))
	return true, nil
}

func relationshipRemove(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		relationshipUsage(user)
		return true, nil
	}
	a, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	b, err := strconv.Atoi(args[1])
	if err != nil {
		user.SendText(fmt.Sprintf("Bad mob id %q\r\n", args[1]))
		return true, nil
	}
	t := relationships.Type(strings.ToLower(args[2]))
	relationships.Remove(a, b, t)
	user.SendText(fmt.Sprintf("Removed: %d → %d (%s).\r\n", a, b, t))
	return true, nil
}

func relationshipList(user *users.UserRecord) (bool, error) {
	all := relationships.AllRelations()
	if len(all) == 0 {
		user.SendText("Graph is empty.\r\n")
		return true, nil
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Owner != all[j].Owner {
			return all[i].Owner < all[j].Owner
		}
		return all[i].Other < all[j].Other
	})
	var b strings.Builder
	fmt.Fprintf(&b, "Relationship graph (%d edges):\r\n", len(all))
	for _, e := range all {
		sub := ""
		if e.Subtype != "" {
			sub = fmt.Sprintf(" [%s]", e.Subtype)
		}
		fmt.Fprintf(&b, "  %4d → %4d  %-10s%s\r\n", e.Owner, e.Other, e.Type, sub)
	}
	user.SendText(b.String())
	return true, nil
}

// relationshipMobNameFor returns the mob template's name for
// display, or "?" if not loaded. Named with a relationship prefix
// to avoid collision with similarly-named helpers in other admin
// command files in this package.
func relationshipMobNameFor(mobId int) string {
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return "?"
	}
	return spec.Character.Name
}
