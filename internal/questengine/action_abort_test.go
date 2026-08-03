package questengine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/messaging"
)

// recordingCtx is a minimal ActionContext that records which effects fired.
type recordingCtx struct {
	granted   []string
	gaveItem  []int
	gaveGold  []int
	flagsSet  []string
	textsSent []string
}

func (c *recordingCtx) GrantQuest(token string) { c.granted = append(c.granted, token) }
func (c *recordingCtx) GiveItem(itemId int) error {
	c.gaveItem = append(c.gaveItem, itemId)
	return nil
}
func (c *recordingCtx) GiveGold(amount int) { c.gaveGold = append(c.gaveGold, amount) }
func (c *recordingCtx) SetQuestFlag(key, value string) {
	c.flagsSet = append(c.flagsSet, key+"="+value)
}
func (c *recordingCtx) SendText(_ messaging.Category, t string) { c.textsSent = append(c.textsSent, t) }

func (c *recordingCtx) ConsumeItem(int)           {}
func (c *recordingCtx) ChargeGold(int)            {}
func (c *recordingCtx) RoomText(string)           {}
func (c *recordingCtx) SpawnMob(SpawnDef)         {}
func (c *recordingCtx) SpawnItem(SpawnDef)        {}
func (c *recordingCtx) TeachSpell(string)         {}
func (c *recordingCtx) TrainSkill(string, int)    {}
func (c *recordingCtx) IncreaseStat(string, int)  {}
func (c *recordingCtx) LearnRecipe(string)        {}
func (c *recordingCtx) ApplyBuff(BuffDef)         {}
func (c *recordingCtx) Teleport(int)              {}
func (c *recordingCtx) LockExits(ExitLock)        {}
func (c *recordingCtx) UnlockExits(ExitLock)      {}
func (c *recordingCtx) QueueNpcSay(NpcSayDef)     {}
func (c *recordingCtx) QueueSequence(SequenceDef) {}
func (c *recordingCtx) GiveMutation()             {}
func (c *recordingCtx) BumpRep(string, int)       {}
func (c *recordingCtx) GetUserId() int            { return 1 }

// badBountyAction returns an action that ExecuteAction rejects: declare_bounty
// with an unknown issuer type.
func badBountyAction() ActionDef {
	a := ActionDef{DeclareBounty: &DeclareBountyDef{}}
	a.DeclareBounty.Issuer.Type = "not-a-real-issuer-type"
	a.DeclareBounty.Issuer.Id = "1"
	return a
}

// A trigger's actions apply as a unit. When one fails, the rest must not run —
// otherwise a quest step can give the item but never set the flag, leaving the
// player in a state the content author never designed.
func TestExecuteActions_AbortsRemainingActionsAfterFailure(t *testing.T) {
	ctx := &recordingCtx{}
	guard := NewEvalGuard(10)

	trig := &indexedTrigger{
		def: &TriggerDef{
			Actions: []ActionDef{
				{GiveItem: 4242},
				badBountyAction(),
				{SetFlag: &QuestFlagAction{Key: "10-branch", Value: "rhett"}},
				{Grant: "10-end"},
			},
		},
		questId: 10,
		trigId:  "q10-t0",
	}

	granted := NewEngine().executeActions(trig, ctx, guard, 1)

	if len(ctx.gaveItem) != 1 {
		t.Fatalf("action before the failure should have run; gaveItem=%v", ctx.gaveItem)
	}
	if len(ctx.flagsSet) != 0 {
		t.Errorf("action after the failure ran anyway; flagsSet=%v", ctx.flagsSet)
	}
	if len(ctx.granted) != 0 {
		t.Errorf("grant after the failure ran anyway; granted=%v", ctx.granted)
	}
	if len(granted) != 0 {
		t.Errorf("a grant after the failure was reported as granted: %v", granted)
	}
}

// A panicking action must abort the rest too, and must not take the server down.
func TestExecuteActions_AbortsAfterPanic(t *testing.T) {
	ctx := &panickingCtx{recordingCtx: recordingCtx{}}
	guard := NewEvalGuard(10)

	trig := &indexedTrigger{
		def: &TriggerDef{
			Actions: []ActionDef{
				{GiveItem: 1}, // panics via panickingCtx
				{Grant: "10-end"},
			},
		},
		questId: 10,
		trigId:  "q10-t1",
	}

	granted := NewEngine().executeActions(trig, ctx, guard, 1)

	if len(ctx.granted) != 0 {
		t.Errorf("action after the panic ran anyway; granted=%v", ctx.granted)
	}
	if len(granted) != 0 {
		t.Errorf("grant after the panic reported as granted: %v", granted)
	}
}

type panickingCtx struct{ recordingCtx }

func (c *panickingCtx) GiveItem(int) error { panic("boom") }

// The happy path must be unaffected: every action still runs in order.
func TestExecuteActions_AllActionsRunWhenNoneFail(t *testing.T) {
	ctx := &recordingCtx{}
	guard := NewEvalGuard(10)

	trig := &indexedTrigger{
		def: &TriggerDef{
			Actions: []ActionDef{
				{GiveItem: 4242},
				{SetFlag: &QuestFlagAction{Key: "10-branch", Value: "rhett"}},
				{Grant: "10-end"},
			},
		},
		questId: 10,
		trigId:  "q10-t2",
	}

	granted := NewEngine().executeActions(trig, ctx, guard, 1)

	if len(ctx.gaveItem) != 1 || len(ctx.flagsSet) != 1 || len(ctx.granted) != 1 {
		t.Fatalf("not every action ran: gaveItem=%v flagsSet=%v granted=%v",
			ctx.gaveItem, ctx.flagsSet, ctx.granted)
	}
	if len(granted) != 1 || granted[0] != "10-end" {
		t.Errorf("granted tokens wrong: %v", granted)
	}
}
