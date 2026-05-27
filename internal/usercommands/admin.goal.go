package usercommands

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
 * Role Permissions:
 * goal           (Admin)
 */

// Goal is the admin command for inspecting and seeding mob goals.
// Subcommands:
//
//	goal list <mob-ident>
//	goal show <mob-ident> <goal-id>
//	goal add <mob-ident> <type> <priority> [key=value ...]
//	goal remove <mob-ident> <goal-id>
//	goal clear <mob-ident>
//
// mob-ident is either a numeric template id (e.g. 371) or a namesimple
// (e.g. tova).
//
// NOTE: do NOT add `type Goal = goals.Goal` — `Goal` is the function name
// in this package, the alias would clash. Always write `goals.Goal`
// explicitly.
func Goal(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		goalShowUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return goalList(args[1:], user)
	case "show":
		return goalShow(args[1:], user)
	case "add":
		return goalAdd(args[1:], user)
	case "remove", "rm":
		return goalRemove(args[1:], user)
	case "clear":
		return goalClear(args[1:], user)
	default:
		goalShowUsage(user)
		return true, nil
	}
}

func goalShowUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.goal", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem,
		"Usage:\r\n"+
			"  goal list <mob-ident>\r\n"+
			"  goal show <mob-ident> <goal-id>\r\n"+
			"  goal add <mob-ident> <type> <priority> [key=value ...]\r\n"+
			"  goal remove <mob-ident> <goal-id>\r\n"+
			"  goal clear <mob-ident>\r\n"+
			"\r\n"+
			"mob-ident: numeric template id (e.g. 371) or namesimple (e.g. tova).\r\n")
}

// goalResolveMobIdent accepts a numeric mobId string or a namesimple
// (like "lars" or "stillwater_smith") and returns the numeric ID,
// canonical display name, and whether it was found. Mirrors
// opinionResolveMobIdent.
func goalResolveMobIdent(s string) (mobId int, name string, ok bool) {
	if id, err := strconv.Atoi(s); err == nil {
		spec := mobs.GetMobSpec(mobs.MobId(id))
		if spec == nil {
			return 0, "", false
		}
		return id, spec.Character.Name, true
	}
	wanted := strings.ToLower(s)
	for _, spec := range mobs.AllMobTemplates() {
		if strings.EqualFold(util.ConvertForFilename(spec.Character.Name), wanted) {
			return int(spec.MobId), spec.Character.Name, true
		}
	}
	return 0, "", false
}

func goalList(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 1 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	all := goals.GoalsOf(mobId, util.ConvertForFilename(name))
	if len(all) == 0 {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("%s (%d) has no goals.\r\n", name, mobId))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Goals for %s (%d):\r\n", name, mobId)
	fmt.Fprintf(&b, "  %-4s  %-20s  %-4s  %s\r\n", "ID", "Type", "Prio", "Params")
	fmt.Fprintf(&b, "  %-4s  %-20s  %-4s  %s\r\n", "----", strings.Repeat("-", 20), "----", "------")
	for _, g := range all {
		fmt.Fprintf(&b, "  %-4s  %-20s  %-4d  %s\r\n",
			g.Id, g.Type, g.Priority, formatParamsInline(g.Params))
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func goalShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 2 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	for _, g := range goals.GoalsOf(mobId, util.ConvertForFilename(name)) {
		if g.Id == args[1] {
			var b strings.Builder
			fmt.Fprintf(&b, "Goal %s on %s (%d):\r\n", g.Id, name, mobId)
			fmt.Fprintf(&b, "  type:        %s\r\n", g.Type)
			fmt.Fprintf(&b, "  priority:    %d\r\n", g.Priority)
			fmt.Fprintf(&b, "  created_at:  %s\r\n", g.CreatedAt.Format("2006-01-02 15:04:05Z"))
			if !g.ExpiresAt.IsZero() {
				fmt.Fprintf(&b, "  expires_at:  %s\r\n", g.ExpiresAt.Format("2006-01-02 15:04:05Z"))
			}
			if len(g.Params) > 0 {
				fmt.Fprintf(&b, "  params:\r\n")
				keys := make([]string, 0, len(g.Params))
				for k := range g.Params {
					keys = append(keys, k)
				}
				sort.Strings(keys)
				for _, k := range keys {
					fmt.Fprintf(&b, "    %s: %v\r\n", k, g.Params[k])
				}
			}
			user.SendText(messaging.CategorySystem, b.String())
			return true, nil
		}
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("No goal %s on %s (%d).\r\n", args[1], name, mobId))
	return true, nil
}

func goalAdd(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 3 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	goalType := args[1]
	prio, err := strconv.Atoi(args[2])
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad priority %q: %v\r\n", args[2], err))
		return true, nil
	}
	params := map[string]any{}
	for _, kv := range args[3:] {
		k, v, found := strings.Cut(kv, "=")
		if !found {
			user.SendText(messaging.CategorySystem,
				fmt.Sprintf("Bad param %q (expected key=value)\r\n", kv))
			return true, nil
		}
		params[k] = parseScalar(v)
	}
	g := &goals.Goal{
		Type:     goalType,
		Priority: prio,
		Params:   params,
	}
	res, err := goals.Add(mobId, util.ConvertForFilename(name), g)
	var ce *goals.ConflictError
	if errors.As(err, &ce) {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("Blocked by goal %s (type=%s, priority=%d).\r\n",
				ce.BlockerGoalId, ce.BlockerType, ce.BlockerPrio))
		return true, nil
	}
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Add failed: %v\r\n", err))
		return true, nil
	}
	msg := fmt.Sprintf("Added goal %s (type=%s, priority=%d)",
		res.Added.Id, res.Added.Type, res.Added.Priority)
	if len(res.Displaced) > 0 {
		msg += fmt.Sprintf(" — displaced goals: %s", strings.Join(res.Displaced, ", "))
	}
	user.SendText(messaging.CategorySystem, msg+".\r\n")
	return true, nil
}

func goalRemove(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 2 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	err := goals.Remove(mobId, util.ConvertForFilename(name), args[1])
	if errors.Is(err, goals.ErrGoalNotFound) {
		user.SendText(messaging.CategorySystem,
			fmt.Sprintf("No goal %s on %s (%d).\r\n", args[1], name, mobId))
		return true, nil
	}
	if err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Remove failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("Removed goal %s from %s (%d).\r\n", args[1], name, mobId))
	return true, nil
}

func goalClear(args []string, user *users.UserRecord) (bool, error) {
	if len(args) != 1 {
		goalShowUsage(user)
		return true, nil
	}
	mobId, name, ok := goalResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	if err := goals.Clear(mobId, util.ConvertForFilename(name)); err != nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Clear failed: %v\r\n", err))
		return true, nil
	}
	user.SendText(messaging.CategorySystem,
		fmt.Sprintf("Cleared all goals from %s (%d).\r\n", name, mobId))
	return true, nil
}

// parseScalar converts an unquoted token to int / float / bool /
// string, in that priority order.
func parseScalar(s string) any {
	if i, err := strconv.Atoi(s); err == nil {
		return i
	}
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	switch strings.ToLower(s) {
	case "true":
		return true
	case "false":
		return false
	}
	return s
}

func formatParamsInline(p map[string]any) string {
	if len(p) == 0 {
		return "-"
	}
	keys := make([]string, 0, len(p))
	for k := range p {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, fmt.Sprintf("%s=%v", k, p[k]))
	}
	return strings.Join(parts, " ")
}
