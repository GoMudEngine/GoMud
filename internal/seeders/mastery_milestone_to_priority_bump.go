package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/goals"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/util"
)

const ruleNameMasteryMilestone = "mastery_milestone_to_priority_bump"
const masteryMilestoneInterval = 10
const masterySoftCap = 50
const masteryMilestonePriority = 40

func init() {
	// DEFERRED — events.SkillUsed does not carry mob rank-up information.
	//
	// Two blockers prevent registration:
	//
	//   1. Player-only UserId: events.SkillUsed has UserId (int) — there
	//      is no MobInstanceId field. The event fires only for player skill
	//      use; mob skill progression is not broadcast through this event.
	//
	//   2. No rank field: SkillUsed carries UserId, Skill (SkillTag), and
	//      Details (sub-command string). It does NOT carry the new rank
	//      after progression, so the "fires on every use" noise vs "fires
	//      only on rank-up" question is moot — even for players there is
	//      no NewRank field to gate the milestone check on.
	//
	// To unblock this rule, extend events.SkillUsed (or add a new
	// events.MobSkillProgressed event) with:
	//   - MobInstanceId int   — 0 for player events
	//   - NewRank        int  — rank after the progression tick
	// and fire the event from the mob skill-progression path in
	// characters/progression.go (or wherever OnSkillUse is dispatched
	// for mobs). Once both fields are present, uncomment the Register
	// call below and replace the TODO-ADAPT field reads in
	// masteryMilestoneSeed.
	//
	// Register(ruleNameMasteryMilestone, masteryMilestoneSeed, "SkillUsed")
}

// masteryMilestoneSeed: when a mob's skill rank crosses a
// multiple-of-10 milestone (10, 20, 30, 40) below the soft cap,
// seeds a mastery-skill goal targeting the next milestone (priority
// 40). Self-prompting NPCs autonomously aim at their next training
// milestone.
//
// DedupKey on skill_name in the mastery-skill catalog type ensures
// only one mastery-skill goal exists per skill per mob at any time.
//
// Registration is DEFERRED — see the init() comment above. The full
// rule logic is implemented here so the only change needed when the
// event surface is extended is uncommenting the Register call (plus
// adapting the TODO-ADAPT field reads below to whatever the new event
// struct exposes).
func masteryMilestoneSeed(event events.Event) {
	su, ok := event.(events.SkillUsed)
	if !ok {
		return
	}

	// TODO-ADAPT: when events.SkillUsed gains MobInstanceId + NewRank
	// fields, replace these reads with the real struct field names.
	// Example of the expected shape after extension:
	//   skillName     := string(su.Skill)
	//   newRank       := su.NewRank       // new field — does not exist yet
	//   mobInstanceId := su.MobInstanceId // new field — does not exist yet
	//
	// For now, read the existing player-only fields so the function
	// compiles and can be reviewed. These early-returns mean the rule
	// is effectively a no-op even if somehow dispatched.
	_ = su.UserId  // placeholder: would read su.MobInstanceId
	_ = su.Details // placeholder: would read su.NewRank
	skillName := string(su.Skill)
	newRank := 0       // TODO-ADAPT: replace with su.NewRank
	mobInstanceId := 0 // TODO-ADAPT: replace with su.MobInstanceId
	if skillName == "" || mobInstanceId == 0 {
		return
	}

	// Milestone check: only act at multiples of masteryMilestoneInterval
	// (10, 20, 30, 40) below the soft cap.
	if newRank <= 0 || newRank >= masterySoftCap {
		return // below first milestone or at/above soft cap
	}
	if newRank%masteryMilestoneInterval != 0 {
		return // not a milestone round number
	}

	mob := mobs.GetInstance(mobInstanceId)
	if mob == nil {
		return
	}

	nextMilestone := newRank + masteryMilestoneInterval
	if nextMilestone > masterySoftCap {
		nextMilestone = masterySoftCap
	}

	g := &goals.Goal{
		Type:     "mastery-skill",
		Priority: masteryMilestonePriority,
		Params: map[string]any{
			"skill_name":  skillName,
			"target_rank": nextMilestone,
		},
	}
	// DedupKey on skill_name (declared in the mastery-skill catalog
	// type) collapses repeat seedings: only one mastery-skill goal per
	// skill survives in the goal store at any time.
	_, _ = goals.Add(int(mob.MobId), util.ConvertForFilename(mob.Character.Name), g)
}
