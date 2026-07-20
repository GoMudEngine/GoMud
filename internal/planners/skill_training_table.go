package planners

// SkillTrainingContext describes the kind of opportunity a mob needs
// to train a particular skill. Used by mastery-skill planner to choose
// what to do.
type SkillTrainingContext string

const (
	TrainingCombat       SkillTrainingContext = "combat"   // weapon/unarmed/spellcasting
	TrainingCrafting     SkillTrainingContext = "crafting" // blacksmithing/alchemy/etc.
	TrainingForaging     SkillTrainingContext = "foraging" // search (field exploration)
	TrainingSocial       SkillTrainingContext = "social"   // rhetoric, bartering
	TrainingSkullduggery SkillTrainingContext = "skullduggery"
	TrainingUnknown      SkillTrainingContext = "unknown"
)

// skillTrainingTable maps a skill name (matching skills.SkillTag string
// values) to its training context. Used by mastery-skill planner.
//
// Verified against internal/skills/skills.go at 4.4 authoring time.
// If a new skill is added later, append a row here.
//
// Note: legacy skill names (tanning, fletching, foraging) do not exist
// in this codebase; foraging/tracking were merged into the "search"
// skill. ranged-combat was revived by the ranged-weapons feature and is
// mapped to TrainingCombat alongside the other weapon skills.
var skillTrainingTable = map[string]SkillTrainingContext{
	"weapon-combat":  TrainingCombat,
	"unarmed-combat": TrainingCombat,
	"ranged-combat":  TrainingCombat,
	"spellcasting":   TrainingCombat,
	"manifestation":  TrainingCombat,
	"rhetoric":       TrainingSocial,
	"bartering":      TrainingSocial,
	"skullduggery":   TrainingSkullduggery,
	"blacksmithing":  TrainingCrafting,
	"alchemy":        TrainingCrafting,
	"tailoring":      TrainingCrafting,
	"cooking":        TrainingCrafting,
	"jewelcrafting":  TrainingCrafting,
	"enchanting":     TrainingCrafting,
	"salvage":        TrainingCrafting,
	"search":         TrainingForaging,
}

// SkillTrainingContextOf returns the training context for a skill name,
// or TrainingUnknown if the skill isn't in the table.
func SkillTrainingContextOf(skillName string) SkillTrainingContext {
	if ctx, ok := skillTrainingTable[skillName]; ok {
		return ctx
	}
	return TrainingUnknown
}
