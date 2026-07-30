package quests

// Registry-backed save-time validation (admin web-building 5c). Structural
// rules are (*Quest).Validate()'s job; this file checks every id, token, and
// name a quest REFERENCES against the live registries — injected (the 5b
// dialogue-editor pattern) so handler tests run with no world loaded. It also
// subsumes ValidateAllFlags' boot PANIC: a bad flag reference is refused at
// save time instead of taking the next boot down.

import (
	"fmt"
	"strings"
)

// QuestValidators are the registry checks the pure validator cannot own.
type QuestValidators struct {
	StepExists     func(token string) bool // foreign "<qid>-<step>" resolves to a real quest step
	MobExists      func(id int) bool
	ItemExists     func(id int) bool
	RoomExists     func(id int) bool
	BuffExists     func(id int) bool
	SpellExists    func(id string) bool
	SkillExists    func(name string) bool
	StatExists     func(name string) bool
	RecipeExists   func(id string) bool
	FactionExists  func(id string) bool
	FlagDeclared   func(key, value string) bool // foreign quests' flag declarations
	DialogueGrants func(token string) bool      // some dialogue file grants this token
	MobHasDialogue func(mobId int) bool
}

// refsCtx accumulates findings and carries the incoming definition, since
// own-quest tokens and flags must validate against the definition BEING
// SAVED (its steps/flags may differ from the cached copy).
type refsCtx struct {
	q     *Quest
	v     *QuestValidators
	steps map[string]bool
	flags map[string][]string // full "<qid>-<key>" -> allowed values
	errs  []string
	warns []string
}

func (c *refsCtx) errf(format string, args ...any) {
	c.errs = append(c.errs, fmt.Sprintf(format, args...))
}

func (c *refsCtx) warnf(format string, args ...any) {
	c.warns = append(c.warns, fmt.Sprintf(format, args...))
}

// checkToken validates a quest token: own-quest tokens against the incoming
// step list, foreign tokens against the injected registry.
func (c *refsCtx) checkToken(where, tok string) {
	if tok == "" {
		return
	}
	ownPrefix := fmt.Sprintf("%d-", c.q.QuestId)
	if rest, ok := strings.CutPrefix(tok, ownPrefix); ok {
		if !c.steps[rest] {
			c.errf("%s: token %q names no step of this quest", where, tok)
		}
		return
	}
	if !c.v.StepExists(tok) {
		c.errf("%s: token %q resolves to no quest step", where, tok)
	}
}

func (c *refsCtx) checkFlag(where, key, value string) {
	if key == "" {
		return
	}
	if allowed, ok := c.flags[key]; ok {
		if value == "" {
			return
		}
		for _, v := range allowed {
			if v == value {
				return
			}
		}
		c.errf("%s: flag %q has invalid value %q (allowed: %v)", where, key, value, allowed)
		return
	}
	if !c.v.FlagDeclared(key, value) {
		c.errf("%s: flag %q value %q is not declared by any quest", where, key, value)
	}
}

func (c *refsCtx) checkMob(where string, id int) {
	if id > 0 && !c.v.MobExists(id) {
		c.errf("%s: mob %d does not exist", where, id)
	}
}

func (c *refsCtx) checkItem(where string, id int) {
	if id > 0 && !c.v.ItemExists(id) {
		c.errf("%s: item %d does not exist", where, id)
	}
}

func (c *refsCtx) checkRoom(where string, id int) {
	if id > 0 && !c.v.RoomExists(id) {
		c.errf("%s: room %d does not exist", where, id)
	}
}

// checkPairList validates composite "name:number[,name:number]" reward
// strings (skillinfo, stat_info) against a name registry.
func (c *refsCtx) checkPairList(where, list, kind string, exists func(string) bool) {
	if list == "" {
		return
	}
	for _, part := range strings.Split(list, ",") {
		name := strings.TrimSpace(strings.SplitN(part, ":", 2)[0])
		if name != "" && !exists(name) {
			c.errf("%s: %s %q does not exist", where, kind, name)
		}
	}
}

func (c *refsCtx) checkActions(where string, actions []ActionDef, depth int) {
	for i, a := range actions {
		aw := fmt.Sprintf("%s action %d", where, i)
		c.checkToken(aw+" grant", a.Grant)
		c.checkItem(aw+" consume_item", a.ConsumeItem)
		c.checkItem(aw+" give_item", a.GiveItem)
		if a.NpcSay != nil {
			c.checkMob(aw+" npc_say", a.NpcSay.Mob)
			for j, l := range a.NpcSay.Lines {
				c.checkMob(fmt.Sprintf("%s npc_say line %d speaker", aw, j), l.Speaker)
			}
			if a.NpcSay.Mob > 0 && c.v.MobExists(a.NpcSay.Mob) && !c.v.MobHasDialogue(a.NpcSay.Mob) {
				c.warnf("%s: npc_say mob %d has no dialogue file (it can still say lines, but players cannot talk back)", aw, a.NpcSay.Mob)
			}
		}
		if a.SpawnMob != nil {
			c.checkMob(aw+" spawn_mob", a.SpawnMob.Id)
			c.checkRoom(aw+" spawn_mob", a.SpawnMob.Room)
		}
		if a.SpawnItem != nil {
			c.checkItem(aw+" spawn_item", a.SpawnItem.Id)
			c.checkRoom(aw+" spawn_item", a.SpawnItem.Room)
		}
		if a.LockExits != nil {
			c.checkRoom(aw+" lock_exits", a.LockExits.Room)
		}
		if a.UnlockExits != nil {
			c.checkRoom(aw+" unlock_exits", a.UnlockExits.Room)
		}
		if a.TeachSpell != "" && !c.v.SpellExists(a.TeachSpell) {
			c.errf("%s: teach_spell %q does not exist", aw, a.TeachSpell)
		}
		if a.TrainSkill != nil && !c.v.SkillExists(a.TrainSkill.Skill) {
			c.errf("%s: train_skill %q does not exist", aw, a.TrainSkill.Skill)
		}
		if a.TrainStat != nil && !c.v.StatExists(a.TrainStat.Stat) {
			c.errf("%s: train_stat %q does not exist", aw, a.TrainStat.Stat)
		}
		if a.LearnRecipe != nil && !c.v.RecipeExists(a.LearnRecipe.Recipe) {
			c.errf("%s: learn_recipe %q does not exist", aw, a.LearnRecipe.Recipe)
		}
		if a.ApplyBuff != nil && a.ApplyBuff.Buff > 0 && !c.v.BuffExists(a.ApplyBuff.Buff) {
			c.errf("%s: buff %d does not exist", aw, a.ApplyBuff.Buff)
		}
		c.checkRoom(aw+" teleport", a.Teleport)
		if a.SetFlag != nil {
			c.checkFlag(aw+" set_flag", a.SetFlag.Key, a.SetFlag.Value)
		}
		if a.BumpRep != nil && a.BumpRep.Faction != "" && !c.v.FactionExists(a.BumpRep.Faction) {
			c.errf("%s: bump_rep faction %q does not exist", aw, a.BumpRep.Faction)
		}
		if a.DeclareBounty != nil && a.DeclareBounty.Target != nil && a.DeclareBounty.Target.Type == "mob" {
			c.checkMob(aw+" declare_bounty target", a.DeclareBounty.Target.Id)
		}
		if a.Sequence != nil {
			for j, l := range a.Sequence.Lines {
				c.checkMob(fmt.Sprintf("%s sequence line %d speaker", aw, j), l.Speaker)
			}
			// The engine nests exactly one level; guard against pathological
			// recursion anyway.
			if depth < 2 {
				c.checkActions(aw+" sequence on_complete", a.Sequence.OnComplete, depth+1)
			}
		}
	}
}

// grantedSteps collects every own-quest step id granted by the definition's
// own triggers (including nested sequences) or by its rewards chain token.
func (c *refsCtx) grantedSteps() map[string]bool {
	granted := map[string]bool{}
	ownPrefix := fmt.Sprintf("%d-", c.q.QuestId)
	var walk func(actions []ActionDef, depth int)
	walk = func(actions []ActionDef, depth int) {
		for _, a := range actions {
			if rest, ok := strings.CutPrefix(a.Grant, ownPrefix); ok {
				granted[rest] = true
			}
			if a.Sequence != nil && depth < 2 {
				walk(a.Sequence.OnComplete, depth+1)
			}
		}
	}
	for _, t := range c.q.Triggers {
		walk(t.Actions, 0)
	}
	if rest, ok := strings.CutPrefix(c.q.Rewards.QuestId, ownPrefix); ok {
		granted[rest] = true
	}
	return granted
}

// ValidateQuestRefs checks every id/token/name the quest references against
// the live registries. Returns refusals and warnings.
func ValidateQuestRefs(q Quest, v QuestValidators) (errs []string, warns []string) {
	c := &refsCtx{q: &q, v: &v, steps: map[string]bool{}, flags: map[string][]string{}}
	for _, s := range q.Steps {
		c.steps[s.Id] = true
	}
	for _, f := range q.Flags {
		c.flags[flagKey(q.QuestId, f.Key)] = f.Values
	}

	// Steps: map_target rooms must exist; warn on likely-typo targets.
	roomFiltered := map[int]bool{}
	hasRoomFilters := false
	for _, t := range q.Triggers {
		if t.Room > 0 {
			roomFiltered[t.Room] = true
			hasRoomFilters = true
		}
	}
	for i, s := range q.Steps {
		if s.MapTarget > 0 {
			c.checkRoom(fmt.Sprintf("step %d (%s) map_target", i, s.Id), s.MapTarget)
			if hasRoomFilters && !roomFiltered[s.MapTarget] && c.v.RoomExists(s.MapTarget) {
				c.warnf("step %d (%s): map_target room %d is referenced by no trigger — double-check it is the intended destination", i, s.Id, s.MapTarget)
			}
		}
		if s.MapTargetMob > 0 {
			c.checkMob(fmt.Sprintf("step %d (%s) map_target_mob", i, s.Id), s.MapTargetMob)
			// -1 is the hard "no marker" switch, so pairing it with a mob target
			// is contradictory — the mob would never be consulted.
			if s.MapTarget == -1 {
				c.warnf("step %d (%s): map_target_mob %d is dead config — map_target -1 suppresses it", i, s.Id, s.MapTargetMob)
			}
			// The mob target is skipped whenever the NPC is dead or unspawned.
			// Without a static fallback the marker simply vanishes then.
			if s.MapTarget == 0 {
				c.warnf("step %d (%s): map_target_mob %d has no map_target fallback — the marker disappears whenever that NPC is not spawned", i, s.Id, s.MapTargetMob)
			}
		}
	}

	// Triggers: filters + conditions + actions.
	for i, t := range q.Triggers {
		tw := fmt.Sprintf("trigger %d", i)
		c.checkMob(tw+" mob filter", t.Mob)
		c.checkItem(tw+" item filter", t.Item)
		c.checkRoom(tw+" room filter", t.Room)
		if t.Skill != "" && !c.v.SkillExists(t.Skill) {
			c.errf("%s: skill filter %q does not exist", tw, t.Skill)
		}
		c.checkToken(tw+" quest_token filter", t.QuestToken)

		for _, tok := range t.Conditions.Has {
			c.checkToken(tw+" has", tok)
		}
		for _, tok := range t.Conditions.Missing {
			c.checkToken(tw+" missing", tok)
		}
		c.checkRoom(tw+" in_room", t.Conditions.InRoom)
		c.checkItem(tw+" has_item", t.Conditions.HasItem)
		c.checkItem(tw+" missing_item", t.Conditions.MissingItem)
		for k, val := range t.Conditions.HasFlag {
			c.checkFlag(tw+" has_flag", k, val)
		}
		for k, val := range t.Conditions.MissingFlag {
			c.checkFlag(tw+" missing_flag", k, val)
		}

		c.checkActions(tw, t.Actions, 0)
	}

	// Rewards.
	c.checkToken("rewards questid", q.Rewards.QuestId)
	c.checkItem("rewards itemid", q.Rewards.ItemId)
	if q.Rewards.BuffId > 0 && !c.v.BuffExists(q.Rewards.BuffId) {
		c.errf("rewards buffid: buff %d does not exist", q.Rewards.BuffId)
	}
	if q.Rewards.SpellId != "" && !c.v.SpellExists(q.Rewards.SpellId) {
		c.errf("rewards spellid %q does not exist", q.Rewards.SpellId)
	}
	c.checkPairList("rewards skillinfo", q.Rewards.SkillInfo, "skill", c.v.SkillExists)
	c.checkPairList("rewards stat_info", q.Rewards.StatInfo, "stat", c.v.StatExists)
	for _, part := range strings.Split(q.Rewards.RecipeInfo, ",") {
		if r := strings.TrimSpace(part); r != "" && !c.v.RecipeExists(r) {
			c.errf("rewards recipe_info: recipe %q does not exist", r)
		}
	}
	for _, part := range strings.Split(q.Rewards.ItemInfo, ",") {
		p := strings.TrimSpace(strings.SplitN(part, ":", 2)[0])
		if p == "" {
			continue
		}
		var id int
		if _, err := fmt.Sscanf(p, "%d", &id); err == nil {
			c.checkItem("rewards item_info", id)
		}
	}
	c.checkRoom("rewards roomid", q.Rewards.RoomId)
	if q.Rewards.RepFaction != "" && !c.v.FactionExists(q.Rewards.RepFaction) {
		c.errf("rewards rep_faction %q does not exist", q.Rewards.RepFaction)
	}

	// Warnings: a step nothing grants is unreachable content.
	granted := c.grantedSteps()
	for _, s := range q.Steps {
		if !granted[s.Id] && !c.v.DialogueGrants(fmt.Sprintf("%d-%s", q.QuestId, s.Id)) {
			c.warnf("step %q is granted by no trigger, no reward chain, and no dialogue file — players cannot reach it", s.Id)
		}
	}

	return c.errs, c.warns
}
