package questengine

import "fmt"

// ActionContext is the interface actions use to modify game state.
// Implemented by the real game bridge (Phase 2) and mocks (tests).
type ActionContext interface {
	GrantQuest(token string)
	ConsumeItem(itemId int)
	GiveItem(itemId int)
	GiveGold(amount int)
	SendText(text string)
	RoomText(text string)
	SpawnMob(s SpawnDef)
	SpawnItem(s SpawnDef)
	TeachSpell(spellId string)
	TrainSkill(skill string, level int)
	ApplyBuff(b BuffDef)
	Teleport(roomId int)
	LockExits(e ExitLock)
	UnlockExits(e ExitLock)
	QueueNpcSay(n NpcSayDef)
	QueueSequence(s SequenceDef)
	GiveMutation()
	SetQuestFlag(key, value string)
	BumpRep(factionId string, delta int)
	GetUserId() int
}

// ExecuteAction runs a single action definition against the given context.
func ExecuteAction(a ActionDef, ctx ActionContext) error {
	if a.Grant != "" {
		LogMinimalF(ctx.GetUserId(), "granted %s", a.Grant)
		ctx.GrantQuest(a.Grant)
		return nil
	}
	if a.ConsumeItem > 0 {
		LogVerboseF(ctx.GetUserId(), "consuming item %d", a.ConsumeItem)
		ctx.ConsumeItem(a.ConsumeItem)
		return nil
	}
	if a.GiveItem > 0 {
		LogVerboseF(ctx.GetUserId(), "giving item %d", a.GiveItem)
		ctx.GiveItem(a.GiveItem)
		return nil
	}
	if a.GiveGold > 0 {
		LogVerboseF(ctx.GetUserId(), "giving %d gold", a.GiveGold)
		ctx.GiveGold(a.GiveGold)
		return nil
	}
	if a.SendText != "" {
		ctx.SendText(a.SendText)
		return nil
	}
	if a.RoomText != "" {
		ctx.RoomText(a.RoomText)
		return nil
	}
	if a.NpcSay != nil {
		LogVerboseF(ctx.GetUserId(), "npc_say mob %d, %d lines", a.NpcSay.Mob, len(a.NpcSay.Lines))
		ctx.QueueNpcSay(*a.NpcSay)
		return nil
	}
	if a.SpawnMob != nil {
		LogVerboseF(ctx.GetUserId(), "spawn mob %d in room %d", a.SpawnMob.Id, a.SpawnMob.Room)
		ctx.SpawnMob(*a.SpawnMob)
		return nil
	}
	if a.SpawnItem != nil {
		LogVerboseF(ctx.GetUserId(), "spawn item %d in room %d", a.SpawnItem.Id, a.SpawnItem.Room)
		ctx.SpawnItem(*a.SpawnItem)
		return nil
	}
	if a.LockExits != nil {
		LogVerboseF(ctx.GetUserId(), "lock exits room %d (player_scoped=%v)", a.LockExits.Room, a.LockExits.PlayerScoped)
		ctx.LockExits(*a.LockExits)
		return nil
	}
	if a.UnlockExits != nil {
		LogVerboseF(ctx.GetUserId(), "unlock exits room %d", a.UnlockExits.Room)
		ctx.UnlockExits(*a.UnlockExits)
		return nil
	}
	if a.TeachSpell != "" {
		LogVerboseF(ctx.GetUserId(), "teach spell %s", a.TeachSpell)
		ctx.TeachSpell(a.TeachSpell)
		return nil
	}
	if a.TrainSkill != nil {
		LogVerboseF(ctx.GetUserId(), "train %s to %d", a.TrainSkill.Skill, a.TrainSkill.Level)
		ctx.TrainSkill(a.TrainSkill.Skill, a.TrainSkill.Level)
		return nil
	}
	if a.ApplyBuff != nil {
		LogVerboseF(ctx.GetUserId(), "apply buff %d", a.ApplyBuff.Buff)
		ctx.ApplyBuff(*a.ApplyBuff)
		return nil
	}
	if a.Teleport > 0 {
		LogVerboseF(ctx.GetUserId(), "teleport to room %d", a.Teleport)
		ctx.Teleport(a.Teleport)
		return nil
	}
	if a.GiveMutation {
		LogVerboseF(ctx.GetUserId(), "give_mutation")
		ctx.GiveMutation()
		return nil
	}
	if a.SetFlag != nil {
		LogVerboseF(ctx.GetUserId(), "set quest flag %s=%s", a.SetFlag.Key, a.SetFlag.Value)
		ctx.SetQuestFlag(a.SetFlag.Key, a.SetFlag.Value)
		return nil
	}
	if a.BumpRep != nil {
		LogVerboseF(ctx.GetUserId(), "bump_rep %s %+d", a.BumpRep.Faction, a.BumpRep.Delta)
		ctx.BumpRep(a.BumpRep.Faction, a.BumpRep.Delta)
		return nil
	}
	if a.Sequence != nil {
		LogVerboseF(ctx.GetUserId(), "starting sequence, %d lines", len(a.Sequence.Lines))
		ctx.QueueSequence(*a.Sequence)
		return nil
	}
	return fmt.Errorf("action has no recognized field set")
}
