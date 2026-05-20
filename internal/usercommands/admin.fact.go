package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/facts"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
	"github.com/GoMudEngine/GoMud/internal/worldevents"
)

/*
 * Role Permissions:
 * fact          (Admin)
 */

func Fact(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		factUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "list":
		return factList(args[1:], user)
	case "show":
		return factShow(args[1:], user)
	case "declare":
		return factDeclare(rest, user) // pass full rest for `--` description handling
	case "withdraw":
		return factWithdraw(args[1:], user)
	case "expire":
		return factExpire(args[1:], user)
	case "prune-expired":
		return factPrune(user)
	case "awareness":
		return factAwareness(args[1:], user)
	case "teach":
		return factTeach(args[1:], user)
	case "forget":
		return factForget(args[1:], user)
	case "forget-all":
		return factForgetAll(args[1:], user)
	default:
		factUsage(user)
	}
	return true, nil
}

func factUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.fact", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendTextLegacy(out)
		return
	}
	user.SendTextLegacy(
		"Usage:\r\n" +
			"  fact list [--all]\r\n" +
			"  fact show <factId>\r\n" +
			"  fact declare <factId> [opts...] -- <description>\r\n" +
			"  fact withdraw <factId>\r\n" +
			"  fact expire <factId>\r\n" +
			"  fact prune-expired\r\n" +
			"  fact awareness <mobId>\r\n" +
			"  fact teach <mobId> <factId> [--source S]\r\n" +
			"  fact forget <mobId> <factId>\r\n" +
			"  fact forget-all <mobId>\r\n",
	)
}

func factList(args []string, user *users.UserRecord) (bool, error) {
	all := false
	for _, a := range args {
		if a == "--all" {
			all = true
		}
	}
	var rows []*facts.Fact
	if all {
		rows = facts.AllRows()
	} else {
		rows = facts.AllActiveFacts()
	}
	if len(rows) == 0 {
		user.SendTextLegacy("No facts.\r\n")
		return true, nil
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].Id < rows[j].Id })

	var b strings.Builder
	fmt.Fprintf(&b, "fact list (%d):\r\n\r\n", len(rows))
	fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
		"ID", "Status", "Sig", "Zone", "Region", "Tags")
	fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
		"----------------------", "----------", "----------", "----------", "----------", "----")
	for _, f := range rows {
		zone := f.Zone
		if zone == "" {
			zone = "-"
		}
		region := f.Region
		if region == "" {
			region = "-"
		}
		fmt.Fprintf(&b, "  %-22s  %-10s  %-10s  %-10s  %-10s  %s\r\n",
			factTruncate(f.Id, 22),
			f.Status,
			significanceLabel(f.Significance),
			factTruncate(zone, 10),
			factTruncate(region, 10),
			strings.Join(f.Tags, ","))
	}
	user.SendTextLegacy(b.String())
	return true, nil
}

func factShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	f := facts.GetFact(args[0])
	if f == nil {
		user.SendTextLegacy(fmt.Sprintf("No fact with id %q\r\n", args[0]))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Fact: %s (%s)\r\n", f.Id, f.Status)
	fmt.Fprintf(&b, "  Description: %s\r\n", f.Description)
	fmt.Fprintf(&b, "  Significance: %s   Zone: %s   Region: %s\r\n",
		significanceLabel(f.Significance), defaultDash(f.Zone), defaultDash(f.Region))
	fmt.Fprintf(&b, "  Declared round: %d\r\n", f.DeclaredRound)
	if f.ExpiryRound > 0 {
		fmt.Fprintf(&b, "  Expiry round: %d\r\n", f.ExpiryRound)
	} else {
		fmt.Fprintf(&b, "  Expiry: never\r\n")
	}
	if len(f.Tags) > 0 {
		fmt.Fprintf(&b, "  Tags: %s\r\n", strings.Join(f.Tags, ", "))
	}
	if f.WithdrawOnRespawnOf > 0 {
		fmt.Fprintf(&b, "  Bound mob: %d (auto-withdraws on respawn)\r\n", f.WithdrawOnRespawnOf)
	}
	user.SendTextLegacy(b.String())
	return true, nil
}

func factDeclare(rest string, user *users.UserRecord) (bool, error) {
	// Custom parser: split on ` -- ` first to extract description.
	idx := strings.Index(rest, " -- ")
	if idx < 0 {
		user.SendTextLegacy("Missing -- separator before description.\r\nUsage: fact declare <factId> [opts] -- <description>\r\n")
		return true, nil
	}
	beforeDash := strings.TrimSpace(rest[:idx])
	description := strings.TrimSpace(rest[idx+4:])

	args := strings.Fields(beforeDash)
	if len(args) < 2 || strings.ToLower(args[0]) != "declare" {
		factUsage(user)
		return true, nil
	}
	factId := args[1]
	opts := facts.DeclareOpts{Description: description, Significance: worldevents.Local}
	for i := 2; i < len(args); i++ {
		switch args[i] {
		case "--zone":
			i++
			if i < len(args) {
				opts.Zone = args[i]
			}
		case "--region":
			i++
			if i < len(args) {
				opts.Region = args[i]
			}
		case "--significance":
			i++
			if i < len(args) {
				opts.Significance = parseSignificance(args[i])
			}
		case "--expiry-rounds":
			i++
			if i < len(args) {
				if v, err := strconv.ParseUint(args[i], 10, 64); err == nil && v > 0 {
					// Convert relative rounds to absolute expiry.
					// v == 0 means "never expire" — leave ExpiryRound as 0.
					opts.ExpiryRound = util.GetRoundCount() + v
				}
			}
		case "--tags":
			i++
			if i < len(args) {
				opts.Tags = strings.Split(args[i], ",")
			}
		case "--withdraw-on-respawn-of":
			i++
			if i < len(args) {
				if v, err := strconv.Atoi(args[i]); err == nil {
					opts.WithdrawOnRespawnOf = v
				}
			}
		}
	}

	if err := facts.Declare(factId, opts); err != nil {
		user.SendTextLegacy(fmt.Sprintf("Declare failed: %v\r\n", err))
		return true, nil
	}
	user.SendTextLegacy(fmt.Sprintf("Declared fact %q.\r\n", factId))
	return true, nil
}

func factWithdraw(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	facts.Withdraw(args[0])
	user.SendTextLegacy(fmt.Sprintf("Withdrew fact %q.\r\n", args[0]))
	return true, nil
}

func factExpire(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	facts.Expire(args[0])
	user.SendTextLegacy(fmt.Sprintf("Expired fact %q.\r\n", args[0]))
	return true, nil
}

func factPrune(user *users.UserRecord) (bool, error) {
	count := facts.PruneExpired()
	user.SendTextLegacy(fmt.Sprintf("Pruned %d expired fact(s).\r\n", count))
	return true, nil
}

func factAwareness(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendTextLegacy(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	a := facts.AllForObserver(mobId)
	known := facts.KnownFactsOf(mobId)
	var b strings.Builder
	fmt.Fprintf(&b, "Awareness for mob %d (%s):\r\n", mobId, defaultDash(a.ObserverName))
	fmt.Fprintf(&b, "  Heard events (%d): %v\r\n", len(a.HeardEvents), a.HeardEvents)
	fmt.Fprintf(&b, "  Known facts (%d):\r\n", len(known))
	for _, k := range known {
		fmt.Fprintf(&b, "    %-22s source: %-10s round: %d\r\n",
			factTruncate(k.Fact.Id, 22), k.Source, k.LearnedRound)
	}
	user.SendTextLegacy(b.String())
	return true, nil
}

func factTeach(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendTextLegacy(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	factId := args[1]
	source := facts.SourceWitnessed
	for i := 2; i < len(args); i++ {
		if args[i] == "--source" && i+1 < len(args) {
			source = facts.Source(strings.ToLower(args[i+1]))
			i++
		}
	}
	facts.RecordKnowsFact(mobId, factId, source)
	user.SendTextLegacy(fmt.Sprintf("Taught mob %d the fact %q (source: %s).\r\n", mobId, factId, source))
	return true, nil
}

func factForget(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendTextLegacy(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	facts.ForgetFact(mobId, args[1])
	user.SendTextLegacy(fmt.Sprintf("Mob %d forgot fact %q.\r\n", mobId, args[1]))
	return true, nil
}

func factForgetAll(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		factUsage(user)
		return true, nil
	}
	mobId, err := strconv.Atoi(args[0])
	if err != nil {
		user.SendTextLegacy(fmt.Sprintf("Bad mob id %q\r\n", args[0]))
		return true, nil
	}
	facts.ForgetAll(mobId)
	user.SendTextLegacy(fmt.Sprintf("Mob %d forgot all facts and events.\r\n", mobId))
	return true, nil
}

// ─── Helpers ─────────────────────────────────────────────────────────────────

// factTruncate shortens s to at most n runes, appending "…" if trimmed.
// Named with a fact prefix to avoid collision with similarly-named
// helpers in other admin command files in this package.
func factTruncate(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n-1]) + "…"
}

// defaultDash returns s if non-empty, or "-" otherwise.
func defaultDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// significanceLabel converts a Significance value to a display string.
func significanceLabel(s worldevents.Significance) string {
	switch s {
	case worldevents.Global:
		return "global"
	case worldevents.Regional:
		return "regional"
	default:
		return "local"
	}
}

// parseSignificance parses "local" | "regional" | "global" → Significance.
func parseSignificance(s string) worldevents.Significance {
	switch strings.ToLower(s) {
	case "global":
		return worldevents.Global
	case "regional":
		return worldevents.Regional
	default:
		return worldevents.Local
	}
}
