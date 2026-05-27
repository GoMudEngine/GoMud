package seeders

import (
	"strings"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/quests"
)

const ruleNameQuestCompletionToOpinion = "quest_completion_to_opinion_boost"
const questCompletionDefaultBump = 10

func init() {
	// Registered for Quest — fires on every quest token grant (start,
	// progress, end). The rule filters to completion events only by
	// checking for the "-end" suffix on QuestToken.
	//
	// STUB NOTICE (chunk 4.5): the quests.Quest struct carries no
	// GiverMobId field, and there is no single-NPC "giver" concept in
	// the quest YAML schema as of 2026-05-27. The rule therefore
	// EARLY-RETURNS after verifying the token is a completion — it
	// subscribes correctly, exercises the event-routing path, and filters
	// accurately, but produces no opinion bump until the quests package
	// exposes a giver.
	//
	// TODO-ADAPT (future chunk): add a GiverMobTemplateId int field to
	// quests.Quest (the YAML struct, not the event). Then replace the
	// early-return body in resolveQuestGiverMobId with:
	//   spec := quests.GetQuest(questToken + "-end")  // or by quest id
	//   if spec != nil { return spec.GiverMobTemplateId }
	// and add a complete_opinion_bump int field to the YAML schema for
	// per-quest overrides (default questCompletionDefaultBump).
	//
	// Also wire opinions.Bump(giverMobTemplateId, userId, bump) once
	// resolveQuestGiverMobId returns a non-zero value.
	Register(ruleNameQuestCompletionToOpinion, questCompletionToOpinion, "Quest")
}

// questCompletionToOpinion: on quest completion (QuestToken ends in
// "-end"), bump the opinion of the quest giver toward the completing
// user by questCompletionDefaultBump. No cooldown — quest completion is
// itself rate-limited by quest gating and cannot be spammed.
//
// STUB: giver lookup is not yet implemented (see init() comment).
// The rule logs at debug level when it would have fired so future
// development can observe the framework path exercising correctly.
//
// Firer: internal/hooks/Quest_GrantQuestToken.go (existing listener
// that fires events.Quest on every token grant).
func questCompletionToOpinion(event events.Event) {
	q, ok := event.(events.Quest)
	if !ok {
		return
	}
	if q.UserId == 0 || q.QuestToken == "" {
		return
	}

	// Only act on quest completions — the conventional terminal step.
	if !strings.HasSuffix(q.QuestToken, "-end") {
		return
	}

	// Resolve quest giver mob template id.
	// Returns 0 while the quests package carries no GiverMobTemplateId
	// field; see init() TODO-ADAPT comment for the upgrade path.
	giverMobTemplateId := resolveQuestGiverMobId(q.QuestToken)
	if giverMobTemplateId == 0 {
		// Stub path: log that we *would* have bumped opinion so the
		// framework path is observable during development and future
		// implementers can confirm the rule fires correctly.
		mudlog.Debug(ruleNameQuestCompletionToOpinion,
			"stub", "giver lookup not implemented",
			"userId", q.UserId,
			"questToken", q.QuestToken,
			"wouldBump", questCompletionDefaultBump,
		)
		return
	}

	// Full path (active once resolveQuestGiverMobId is wired):
	// bump := resolveQuestOpinionBump(q.QuestToken, questCompletionDefaultBump)
	// opinions.Bump(giverMobTemplateId, q.UserId, bump)
}

// resolveQuestGiverMobId returns the mob template id of the quest's
// giver NPC (0 if unresolvable).
//
// STUB: returns 0 until quests.Quest gains a GiverMobTemplateId field.
// See init() TODO-ADAPT comment for the full upgrade path.
func resolveQuestGiverMobId(questToken string) int {
	// Defensive: ensure the quests package is queryable (non-nil
	// return confirms the token references a real quest).
	_ = quests.GetQuest(questToken)
	// TODO-ADAPT: return spec.GiverMobTemplateId when the field exists.
	return 0
}

// resolveQuestOpinionBump returns the per-quest opinion bump value.
// When the quest YAML gains a complete_opinion_bump field, read it here;
// otherwise fall back to the package-level default.
//
// STUB: always returns defaultBump until the YAML field is added.
func resolveQuestOpinionBump(_ string, defaultBump int) int {
	// TODO-ADAPT: load quest spec and return spec.CompleteOpinionBump if > 0.
	return defaultBump
}
