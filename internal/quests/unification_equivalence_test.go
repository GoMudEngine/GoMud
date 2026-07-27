package quests

// The 5c-pre unification equivalence harness. Before the unification, quest
// YAML was parsed TWICE: by this package (yaml.v2, mostly tag-less fields —
// the copy that PAYS REWARDS) and by internal/questengine (QuestDef, fully
// snake_case tagged — the copy that owned map_target and the trigger DSL).
//
// This test carries verbatim LOCAL copies of both old struct shapes and
// asserts the unified quests.Quest reproduces, for every live quest file,
// everything either old parser saw. The locals are deliberately frozen: they
// pin the historical yaml.v2 binding (tag-less = lowercased field name, no
// underscore handling) forever, independent of the production structs.

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v2"
)

// ---- old internal/quests shapes (pre-unification, tag-less binding) ----

type oldQuestFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}

type oldQuestReward struct {
	QuestId       string // binds "questid"
	Gold          int
	ItemId        int // binds "itemid"
	BuffId        int
	SkillInfo     string
	StatInfo      string `yaml:"stat_info,omitempty"`
	RecipeInfo    string `yaml:"recipe_info,omitempty"`
	ItemInfo      string `yaml:"item_info,omitempty"`
	SpellId       string
	PlayerMessage string
	RoomMessage   string
	RoomId        int
	RepFaction    string `yaml:"rep_faction"`
	RepAmount     int    `yaml:"rep_amount"`
}

type oldQuestStep struct {
	Id          string
	Description string
	Hint        string
}

type oldQuest struct {
	QuestId        int
	Name           string
	Description    string
	Secret         bool
	Steps          []oldQuestStep
	Rewards        oldQuestReward
	Flags          []oldQuestFlagDef `yaml:"flags,omitempty"`
	Repeatable     bool              `yaml:"repeatable,omitempty"`
	CooldownRounds int               `yaml:"cooldown_rounds,omitempty"`
}

// ---- old internal/questengine shapes (pre-unification, snake_case tags) ----

type oldEngineFlagDef struct {
	Key         string   `yaml:"key"`
	Values      []string `yaml:"values"`
	Description string   `yaml:"description,omitempty"`
}

type oldQuestDef struct {
	QuestId     int                `yaml:"questid"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Secret      bool               `yaml:"secret,omitempty"`
	Linear      bool               `yaml:"linear,omitempty"`
	Steps       []oldEngineStep    `yaml:"steps"`
	Rewards     oldEngineRewards   `yaml:"rewards,omitempty"`
	Triggers    []oldTriggerDef    `yaml:"triggers"`
	Flags       []oldEngineFlagDef `yaml:"flags,omitempty"`
}

type oldEngineStep struct {
	Id          string `yaml:"id"`
	Description string `yaml:"description,omitempty"`
	Hint        string `yaml:"hint,omitempty"`
	MapTarget   int    `yaml:"map_target,omitempty"`
}

type oldEngineRewards struct {
	Gold          int    `yaml:"gold,omitempty"`
	ItemId        int    `yaml:"item_id,omitempty"`
	BuffId        int    `yaml:"buff_id,omitempty"`
	SpellId       string `yaml:"spell_id,omitempty"`
	SkillInfo     string `yaml:"skill_info,omitempty"`
	StatInfo      string `yaml:"stat_info,omitempty"`
	PlayerMessage string `yaml:"player_message,omitempty"`
	RoomMessage   string `yaml:"room_message,omitempty"`
	RoomId        int    `yaml:"room_id,omitempty"`
	ChainQuest    string `yaml:"chain_quest,omitempty"`
}

type oldTriggerDef struct {
	Event      string         `yaml:"event"`
	Room       int            `yaml:"room,omitempty"`
	Mob        int            `yaml:"mob,omitempty"`
	Item       int            `yaml:"item,omitempty"`
	Skill      string         `yaml:"skill,omitempty"`
	Command    string         `yaml:"command,omitempty"`
	Topic      string         `yaml:"topic,omitempty"`
	QuestToken string         `yaml:"quest_token,omitempty"`
	Noun       string         `yaml:"noun,omitempty"`
	Verb       string         `yaml:"verb,omitempty"`
	Conditions oldConditions  `yaml:"conditions,omitempty"`
	Actions    []oldActionDef `yaml:"actions"`
}

type oldConditions struct {
	Has           []string          `yaml:"has,omitempty"`
	Missing       []string          `yaml:"missing,omitempty"`
	InRoom        int               `yaml:"in_room,omitempty"`
	HasItem       int               `yaml:"has_item,omitempty"`
	MissingItem   int               `yaml:"missing_item,omitempty"`
	HasFlag       map[string]string `yaml:"has_flag,omitempty"`
	MissingFlag   map[string]string `yaml:"missing_flag,omitempty"`
	HasGold       int               `yaml:"has_gold,omitempty"`
	HasMasterwork int               `yaml:"has_masterwork,omitempty"`
}

type oldActionDef struct {
	Grant         string               `yaml:"grant,omitempty"`
	ConsumeItem   int                  `yaml:"consume_item,omitempty"`
	GiveItem      int                  `yaml:"give_item,omitempty"`
	GiveGold      int                  `yaml:"give_gold,omitempty"`
	ChargeGold    int                  `yaml:"charge_gold,omitempty"`
	NpcSay        *oldNpcSayDef        `yaml:"npc_say,omitempty"`
	SendText      string               `yaml:"send_text,omitempty"`
	RoomText      string               `yaml:"room_text,omitempty"`
	SpawnMob      *oldSpawnDef         `yaml:"spawn_mob,omitempty"`
	SpawnItem     *oldSpawnDef         `yaml:"spawn_item,omitempty"`
	LockExits     *oldExitLock         `yaml:"lock_exits,omitempty"`
	UnlockExits   *oldExitLock         `yaml:"unlock_exits,omitempty"`
	TeachSpell    string               `yaml:"teach_spell,omitempty"`
	TrainSkill    *oldSkillDef         `yaml:"train_skill,omitempty"`
	TrainStat     *oldStatDef          `yaml:"train_stat,omitempty"`
	LearnRecipe   *oldRecipeDef        `yaml:"learn_recipe,omitempty"`
	ApplyBuff     *oldBuffDef          `yaml:"apply_buff,omitempty"`
	Teleport      int                  `yaml:"teleport,omitempty"`
	GiveMutation  bool                 `yaml:"give_mutation,omitempty"`
	SetFlag       *oldQuestFlagAction  `yaml:"set_flag,omitempty"`
	Sequence      *oldSequenceDef      `yaml:"sequence,omitempty"`
	BumpRep       *oldBumpRepDef       `yaml:"bump_rep,omitempty"`
	DeclareBounty *oldDeclareBountyDef `yaml:"declare_bounty,omitempty"`
}

type oldBumpRepDef struct {
	Faction string `yaml:"faction"`
	Delta   int    `yaml:"delta"`
}

type oldDeclareBountyDef struct {
	Issuer struct {
		Type string `yaml:"type"`
		Id   string `yaml:"id"`
	} `yaml:"issuer"`
	TargetPlayer bool `yaml:"target_player,omitempty"`
	Target       *struct {
		Type string `yaml:"type"`
		Id   int    `yaml:"id"`
	} `yaml:"target,omitempty"`
	Condition    string `yaml:"condition"`
	ExpiryRounds uint64 `yaml:"expiry_rounds,omitempty"`
	GoldOverride int    `yaml:"gold_override,omitempty"`
	RepOverride  int    `yaml:"rep_override,omitempty"`
	Reason       string `yaml:"reason,omitempty"`
}

type oldQuestFlagAction struct {
	Key   string `yaml:"key"`
	Value string `yaml:"value"`
}

type oldNpcSayDef struct {
	Mob   int             `yaml:"mob"`
	Lines []oldSayLineDef `yaml:"lines"`
}

type oldSayLineDef struct {
	Text    string `yaml:"text"`
	Delay   int    `yaml:"delay,omitempty"`
	Speaker int    `yaml:"speaker,omitempty"`
	Emote   bool   `yaml:"emote,omitempty"`
}

type oldSpawnDef struct {
	Id   int `yaml:"id"`
	Room int `yaml:"room"`
}

type oldExitLock struct {
	Room         int  `yaml:"room"`
	PlayerScoped bool `yaml:"player_scoped,omitempty"`
}

type oldSkillDef struct {
	Skill string `yaml:"skill"`
	Level int    `yaml:"level"`
}

type oldStatDef struct {
	Stat   string `yaml:"stat"`
	Amount int    `yaml:"amount"`
}

type oldRecipeDef struct {
	Recipe string `yaml:"recipe"`
}

type oldBuffDef struct {
	Buff   int    `yaml:"buff"`
	Source string `yaml:"source,omitempty"`
}

type oldSequenceDef struct {
	DelayBetween int             `yaml:"delay_between"`
	Lines        []oldSayLineDef `yaml:"lines"`
	OnComplete   []oldActionDef  `yaml:"on_complete,omitempty"`
	LockMessage  string          `yaml:"lock_message,omitempty"`
}

// ---- harness ----

// chdirRepoRootForTest points the test at the real repo config + data tree
// (test binaries run with CWD = their own package dir).
func chdirRepoRootForTest(t *testing.T) {
	t.Helper()
	mudlog.SetupLogger(nil, `LOW`, ``, false)
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	root := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(root, "go.mod")); err == nil {
			break
		}
		root = filepath.Dir(root)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(wd) })
	if err := configs.ReloadConfig(); err != nil {
		t.Fatalf("ReloadConfig: %v", err)
	}
}

func rewardsEqualOld(n QuestReward, o oldQuestReward) bool {
	return n.QuestId == o.QuestId && n.Gold == o.Gold && n.ItemId == o.ItemId &&
		n.BuffId == o.BuffId && n.SkillInfo == o.SkillInfo && n.StatInfo == o.StatInfo &&
		n.RecipeInfo == o.RecipeInfo && n.ItemInfo == o.ItemInfo && n.SpellId == o.SpellId &&
		n.PlayerMessage == o.PlayerMessage && n.RoomMessage == o.RoomMessage &&
		n.RoomId == o.RoomId && n.RepFaction == o.RepFaction && n.RepAmount == o.RepAmount
}

// triggersEqualOld compares by re-marshaling both sides: the new TriggerDef
// carries the same snake_case tags and field order the old one did, so equal
// content marshals to identical yaml.
func triggersEqualOld(t *testing.T, n TriggerDef, o oldTriggerDef) bool {
	t.Helper()
	nb, err := yaml.Marshal(n)
	if err != nil {
		t.Fatalf("marshal new trigger: %v", err)
	}
	ob, err := yaml.Marshal(o)
	if err != nil {
		t.Fatalf("marshal old trigger: %v", err)
	}
	return string(nb) == string(ob)
}

func TestUnification_EquivalentToBothOldParses(t *testing.T) {
	chdirRepoRootForTest(t)

	dir := configs.GetFilePathsConfig().DataFiles.String() + `/quests`
	files, err := filepath.Glob(dir + `/*.yaml`)
	if err != nil || len(files) == 0 {
		t.Fatalf("no quest files found under %s: %v", dir, err)
	}
	if len(files) < 79 {
		t.Fatalf("expected at least 79 live quest files, glob found %d — wrong directory?", len(files))
	}

	for _, f := range files {
		data, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("%s: %v", f, err)
		}
		var oq oldQuest
		var od oldQuestDef
		var nq Quest
		if err := yaml.Unmarshal(data, &oq); err != nil {
			t.Fatalf("%s: old quests parse: %v", f, err)
		}
		if err := yaml.Unmarshal(data, &od); err != nil {
			t.Fatalf("%s: old questengine parse: %v", f, err)
		}
		if err := yaml.Unmarshal(data, &nq); err != nil {
			t.Fatalf("%s: new unified parse: %v", f, err)
		}

		// Everything the old quests parse populated (the reward-paying copy).
		if nq.QuestId != oq.QuestId || nq.Name != oq.Name ||
			nq.Description != oq.Description || nq.Secret != oq.Secret ||
			nq.Repeatable != oq.Repeatable || nq.CooldownRounds != oq.CooldownRounds {
			t.Errorf("%s: identity mismatch vs old quests parse", f)
		}
		if !rewardsEqualOld(nq.Rewards, oq.Rewards) {
			t.Errorf("%s: REWARDS mismatch vs old quests parse:\nnew=%+v\nold=%+v", f, nq.Rewards, oq.Rewards)
		}

		// Everything the old questengine parse populated.
		if len(nq.Steps) != len(od.Steps) {
			t.Fatalf("%s: step count mismatch: new %d, old engine %d", f, len(nq.Steps), len(od.Steps))
		}
		for i := range od.Steps {
			if nq.Steps[i].Id != od.Steps[i].Id || nq.Steps[i].Description != od.Steps[i].Description ||
				nq.Steps[i].Hint != od.Steps[i].Hint || nq.Steps[i].MapTarget != od.Steps[i].MapTarget {
				t.Errorf("%s: step %d mismatch vs old questengine parse", f, i)
			}
		}
		if len(nq.Triggers) != len(od.Triggers) {
			t.Fatalf("%s: trigger count mismatch: new %d, old engine %d", f, len(nq.Triggers), len(od.Triggers))
		}
		for i := range od.Triggers {
			if !triggersEqualOld(t, nq.Triggers[i], od.Triggers[i]) {
				t.Errorf("%s: trigger %d mismatch vs old questengine parse", f, i)
			}
		}
		if len(nq.Flags) != len(od.Flags) {
			t.Errorf("%s: flag count mismatch: new %d, old engine %d", f, len(nq.Flags), len(od.Flags))
		}
	}
}
