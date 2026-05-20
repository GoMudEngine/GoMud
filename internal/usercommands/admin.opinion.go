package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/templates"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"
)

/*
 * Role Permissions:
 * opinion         (Admin)
 */

// Opinion is the admin command for inspecting and adjusting per-NPC
// disposition scores. Subcommands:
//
//	opinion show <playerName>
//	opinion show <mobName|mobId> <playerName>
//	opinion set <mobName|mobId> <playerName> <score>
//	opinion bump <mobName|mobId> <playerName> <delta>
//	opinion reset <mobName|mobId> <playerName>
func Opinion(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		opinionShowUsage(user)
		return true, nil
	}

	switch strings.ToLower(args[0]) {
	case "show":
		return opinionShow(args[1:], user)
	case "set":
		return opinionMutate(args[1:], user, mutateSet)
	case "bump":
		return opinionMutate(args[1:], user, mutateBump)
	case "reset":
		return opinionMutate(args[1:], user, mutateReset)
	default:
		opinionShowUsage(user)
		return true, nil
	}
}

func opinionShowUsage(user *users.UserRecord) {
	// Try templated help first; fall back to inline if missing.
	if out, err := templates.Process("admincommands/help/command.opinion", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem, 
		"Usage:\r\n" +
			"  opinion show <playerName>\r\n" +
			"  opinion show <mobName|mobId> <playerName>\r\n" +
			"  opinion set <mobName|mobId> <playerName> <score>\r\n" +
			"  opinion bump <mobName|mobId> <playerName> <delta>\r\n" +
			"  opinion reset <mobName|mobId> <playerName>\r\n",
	)
}

func opinionShow(args []string, user *users.UserRecord) (bool, error) {
	switch len(args) {
	case 1:
		return opinionShowAll(args[0], user)
	case 2:
		return opinionShowOne(args[0], args[1], user)
	default:
		opinionShowUsage(user)
		return true, nil
	}
}

func opinionShowAll(playerName string, user *users.UserRecord) (bool, error) {
	target := users.GetByCharacterNameOrLoad(playerName)
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}

	rows := opinions.AllRowsForUser(target.UserId)
	if len(rows) == 0 {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No NPCs hold an opinion of %s.\r\n", playerName))
		return true, nil
	}

	sort.Slice(rows, func(i, j int) bool {
		return rows[i].Score < rows[j].Score
	})

	var b strings.Builder
	fmt.Fprintf(&b, "opinion show %s:\r\n\r\n", playerName)
	fmt.Fprintf(&b, "  %-25s %5s  %-9s  %s\r\n", "Mob", "Score", "Tier", "Last bump")
	fmt.Fprintf(&b, "  %-25s %5s  %-9s  %s\r\n",
		strings.Repeat("-", 25), "-----", "---------", "---------")
	now := util.GetRoundCount()
	for _, r := range rows {
		fmt.Fprintf(&b, "  %-25s %5d  %-9s  %s\r\n",
			opinionTruncate(fmt.Sprintf("%s (%d)", r.MobName, r.MobId), 25),
			r.Score, opinionTierName(opinions.TierOf(r.Score)),
			opinionRoundsAgo(now, r.LastUpdatedRound))
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func opinionShowOne(mobIdent, playerName string, user *users.UserRecord) (bool, error) {
	mobId, mobName, ok := opinionResolveMobIdent(mobIdent)
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", mobIdent))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(playerName)
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such player: %s\r\n", playerName))
		return true, nil
	}
	score := opinions.Get(mobId, target.UserId)
	user.SendText(messaging.CategorySystem, fmt.Sprintf("%s (%d) -> %s: score=%d, tier=%s\r\n",
		mobName, mobId, playerName, score, opinionTierName(opinions.TierOf(score))))
	return true, nil
}

type opinionMutateMode int

const (
	mutateSet opinionMutateMode = iota
	mutateBump
	mutateReset
)

func opinionMutate(args []string, user *users.UserRecord, mode opinionMutateMode) (bool, error) {
	expected := 3
	if mode == mutateReset {
		expected = 2
	}
	if len(args) != expected {
		opinionShowUsage(user)
		return true, nil
	}
	mobId, mobName, ok := opinionResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(args[1])
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}

	switch mode {
	case mutateSet:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad score %q: %v\r\n", args[2], err))
			return true, nil
		}
		opinions.Set(mobId, target.UserId, v)
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Set %s (%d) -> %s = %d\r\n", mobName, mobId, args[1], v))
	case mutateBump:
		v, err := strconv.Atoi(args[2])
		if err != nil {
			user.SendText(messaging.CategorySystem, fmt.Sprintf("Bad delta %q: %v\r\n", args[2], err))
			return true, nil
		}
		opinions.Bump(mobId, target.UserId, v)
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Bumped %s (%d) -> %s by %d (now %d)\r\n",
			mobName, mobId, args[1], v, opinions.Get(mobId, target.UserId)))
	case mutateReset:
		def := opinionMobDefaultDisposition(mobId)
		opinions.Set(mobId, target.UserId, def)
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Reset %s (%d) -> %s to default %d\r\n",
			mobName, mobId, args[1], def))
	}
	return true, nil
}

// opinionResolveMobIdent accepts a numeric mobId string or a namesimple
// (like "lars" or "stillwater_smith") and returns the numeric ID,
// canonical display name, and whether it was found.
func opinionResolveMobIdent(s string) (int, string, bool) {
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

func opinionMobDefaultDisposition(mobId int) int {
	spec := mobs.GetMobSpec(mobs.MobId(mobId))
	if spec == nil {
		return 0
	}
	return spec.DefaultDisposition
}

func opinionTierName(t opinions.Tier) string {
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

func opinionRoundsAgo(now, then uint64) string {
	if then == 0 {
		return "default"
	}
	if now <= then {
		return "just now"
	}
	delta := now - then
	switch {
	case delta < 60:
		return fmt.Sprintf("%d rounds ago", delta)
	case delta < 3600:
		return fmt.Sprintf("~%d game-min ago", delta/60)
	default:
		return fmt.Sprintf("~%d game-hr ago", delta/3600)
	}
}

func opinionTruncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n < 1 {
		return ""
	}
	return s[:n-1] + "~"
}
