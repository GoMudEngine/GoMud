package usercommands

import (
	"fmt"
	"sort"
	"strconv"
	"strings"

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
 * knowledge      (Admin)
 */

func Knowledge(rest string, user *users.UserRecord, room *rooms.Room, flags events.EventFlag) (bool, error) {
	args := strings.Fields(rest)
	if len(args) == 0 {
		knowledgeUsage(user)
		return true, nil
	}
	switch strings.ToLower(args[0]) {
	case "show":
		return knowledgeShow(args[1:], user)
	case "frequented":
		return knowledgeFrequented(args[1:], user)
	case "forget":
		return knowledgeForget(args[1:], user)
	default:
		knowledgeUsage(user)
		return true, nil
	}
}

func knowledgeUsage(user *users.UserRecord) {
	if out, err := templates.Process("admincommands/help/command.knowledge", nil, user.UserId); err == nil && strings.TrimSpace(out) != "" {
		user.SendText(messaging.CategorySystem, out)
		return
	}
	user.SendText(messaging.CategorySystem, 
		"Usage:\r\n" +
			"  knowledge show <mobId>\r\n" +
			"  knowledge show <mobId> <playerName>\r\n" +
			"  knowledge show <mobId> mob <targetId>\r\n" +
			"  knowledge frequented <mobId> <playerName> [topK]\r\n" +
			"  knowledge forget <mobId> <playerName> [fact]\r\n",
	)
}

// knowledgeResolveMobIdent accepts a numeric mobId string or a name
// (like "lars") and returns the numeric ID and whether it was found.
// Mirrors opinionResolveMobIdent from admin.opinion.go.
func knowledgeResolveMobIdent(s string) (int, bool) {
	if id, err := strconv.Atoi(s); err == nil {
		if mobs.GetMobSpec(mobs.MobId(id)) != nil {
			return id, true
		}
		return 0, false
	}
	wanted := strings.ToLower(s)
	for _, spec := range mobs.AllMobTemplates() {
		if strings.EqualFold(util.ConvertForFilename(spec.Character.Name), wanted) {
			return int(spec.MobId), true
		}
	}
	return 0, false
}

func knowledgeResolveSubject(args []string) (knowledge.Subject, bool) {
	switch len(args) {
	case 1:
		// player name shorthand
		target := users.GetByCharacterNameOrLoad(args[0])
		if target == nil {
			return knowledge.Subject{}, false
		}
		return knowledge.PlayerSubject(target.UserId), true
	case 2:
		if strings.ToLower(args[0]) == "mob" {
			id, err := strconv.Atoi(args[1])
			if err != nil {
				return knowledge.Subject{}, false
			}
			return knowledge.MobSubject(id), true
		}
	}
	return knowledge.Subject{}, false
}

func knowledgeShow(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 1 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := knowledgeResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}

	if len(args) == 1 {
		// List all records for this observer.
		records := knowledge.AllForObserver(mobId)
		if len(records) == 0 {
			user.SendText(messaging.CategorySystem, fmt.Sprintf("No knowledge records for mob %d.\r\n", mobId))
			return true, nil
		}
		var b strings.Builder
		fmt.Fprintf(&b, "knowledge show %d (%d records):\r\n\r\n", mobId, len(records))
		fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n", "Subject", "Met", "Name", "LastSeen")
		fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n",
			"------------", "----", "--------------------", "----------")
		sort.Slice(records, func(i, j int) bool {
			return records[i].LastSeenRound > records[j].LastSeenRound
		})
		for _, r := range records {
			subject := fmt.Sprintf("%s:%d", r.Subject.Type, r.Subject.Id)
			met := "no"
			if r.HasMet {
				met = "yes"
			}
			lastSeen := fmt.Sprintf("room %d @ round %d", r.LastSeenRoom, r.LastSeenRound)
			fmt.Fprintf(&b, "  %-12s  %4s  %-20s  %s\r\n", subject, met, r.NameLearned, lastSeen)
		}
		user.SendText(messaging.CategorySystem, b.String())
		return true, nil
	}

	// Drill into a specific subject.
	subj, ok := knowledgeResolveSubject(args[1:])
	if !ok {
		user.SendText(messaging.CategorySystem, "Unknown subject\r\n")
		return true, nil
	}
	r := knowledge.Get(mobId, subj)
	if r == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No record: mob %d <-> %s:%d\r\n", mobId, subj.Type, subj.Id))
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Subject: %s:%d\r\n", subj.Type, subj.Id)
	fmt.Fprintf(&b, "  HasMet: %v   Name: %q   Source: %s   Confidence: %s\r\n",
		r.HasMet, r.NameLearned, r.Source, r.Confidence)
	fmt.Fprintf(&b, "  LastSeen: room %d @ round %d\r\n", r.LastSeenRoom, r.LastSeenRound)
	fmt.Fprintf(&b, "  Observations: %d entries\r\n", len(r.Observations))
	wcs := knowledge.WitnessedCrimes(mobId, subj)
	if len(wcs) > 0 {
		fmt.Fprintf(&b, "  Witnessed crimes:\r\n")
		for _, w := range wcs {
			resolved := "no"
			if w.ResolvedRound > 0 {
				resolved = fmt.Sprintf("round %d", w.ResolvedRound)
			}
			fmt.Fprintf(&b, "    crime %d (%s) resolved: %s\r\n", w.CrimeId, w.Kind, resolved)
		}
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func knowledgeFrequented(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := knowledgeResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(args[1])
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}
	topK := 5
	if len(args) >= 3 {
		if v, err := strconv.Atoi(args[2]); err == nil && v > 0 {
			topK = v
		}
	}
	rows := knowledge.FrequentedRooms(mobId, knowledge.PlayerSubject(target.UserId), topK)
	if len(rows) == 0 {
		user.SendText(messaging.CategorySystem, "No observations recorded.\r\n")
		return true, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "Top %d rooms for mob %d watching %s:\r\n\r\n", topK, mobId, args[1])
	for _, r := range rows {
		fmt.Fprintf(&b, "  room %d  (%d sightings)\r\n", r.Room, r.Count)
	}
	user.SendText(messaging.CategorySystem, b.String())
	return true, nil
}

func knowledgeForget(args []string, user *users.UserRecord) (bool, error) {
	if len(args) < 2 {
		knowledgeUsage(user)
		return true, nil
	}
	mobId, ok := knowledgeResolveMobIdent(args[0])
	if !ok {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Unknown mob: %s\r\n", args[0]))
		return true, nil
	}
	target := users.GetByCharacterNameOrLoad(args[1])
	if target == nil {
		user.SendText(messaging.CategorySystem, fmt.Sprintf("No such player: %s\r\n", args[1]))
		return true, nil
	}
	subj := knowledge.PlayerSubject(target.UserId)
	if len(args) == 2 {
		knowledge.Forget(mobId, subj)
		user.SendText(messaging.CategorySystem, fmt.Sprintf("Forgot record: mob %d <-> %s\r\n", mobId, args[1]))
		return true, nil
	}
	knowledge.ForgetFact(mobId, subj, strings.ToLower(args[2]))
	user.SendText(messaging.CategorySystem, fmt.Sprintf("Forgot fact %q: mob %d <-> %s\r\n", args[2], mobId, args[1]))
	return true, nil
}
