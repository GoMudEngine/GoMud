package questengine

import (
	"fmt"
	"strconv"

	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// ActionContext is the interface actions use to modify game state.
// Implemented by the real game bridge (Phase 2) and mocks (tests).
type ActionContext interface {
	GrantQuest(token string)
	ConsumeItem(itemId int)
	GiveItem(itemId int)
	GiveGold(amount int)
	SendText(cat messaging.Category, text string)
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
		ctx.SendText(messaging.CategoryNPCDialogue, a.SendText)
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
	if a.DeclareBounty != nil {
		def := a.DeclareBounty

		// Resolve issuer.
		var issuer bounties.Issuer
		switch def.Issuer.Type {
		case "faction":
			issuer = bounties.FactionIssuer(def.Issuer.Id)
		case "quest":
			qid := def.Issuer.Id
			intId, err := strconv.Atoi(qid)
			if err != nil {
				return fmt.Errorf("declare_bounty: bad quest issuer id %q: %w", qid, err)
			}
			issuer = bounties.QuestIssuer(intId)
		case "npc":
			intId, err := strconv.Atoi(def.Issuer.Id)
			if err != nil {
				return fmt.Errorf("declare_bounty: bad npc issuer id %q: %w", def.Issuer.Id, err)
			}
			issuer = bounties.NPCIssuer(intId)
		default:
			return fmt.Errorf("declare_bounty: unknown issuer type %q", def.Issuer.Type)
		}

		// Resolve target.
		var target knowledge.Subject
		switch {
		case def.TargetPlayer:
			target = knowledge.PlayerSubject(ctx.GetUserId())
		case def.Target != nil:
			switch def.Target.Type {
			case "player":
				target = knowledge.PlayerSubject(def.Target.Id)
			case "mob":
				target = knowledge.MobSubject(def.Target.Id)
			default:
				return fmt.Errorf("declare_bounty: unknown target type %q", def.Target.Type)
			}
		default:
			return fmt.Errorf("declare_bounty: must set target_player or target")
		}

		// Compute expiry round.
		expiryRound := uint64(0)
		if def.ExpiryRounds > 0 {
			expiryRound = util.GetRoundCount() + def.ExpiryRounds
		}

		condition := bounties.Condition(def.Condition)
		if condition == "" {
			condition = bounties.ConditionKill
		}

		id, err := bounties.Declare(issuer, target, condition, expiryRound,
			bounties.DeclareOpts{
				GoldOverride:   def.GoldOverride,
				RepOverride:    def.RepOverride,
				DeclaredReason: def.Reason,
			})
		if err != nil {
			return fmt.Errorf("declare_bounty: %w", err)
		}
		LogVerboseF(ctx.GetUserId(), "declare_bounty id=%d issuer=%s:%s target=%s:%d",
			id, string(issuer.Type), issuer.Id, string(target.Type), target.Id)
		return nil
	}
	if a.Sequence != nil {
		LogVerboseF(ctx.GetUserId(), "starting sequence, %d lines", len(a.Sequence.Lines))
		ctx.QueueSequence(*a.Sequence)
		return nil
	}
	return fmt.Errorf("action has no recognized field set")
}
