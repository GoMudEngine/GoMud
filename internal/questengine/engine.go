package questengine

import (
	"fmt"
	"sort"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
)

// indexedTrigger pairs a definition trigger with the engine-assigned identity
// the evaluator needs (visit tracking, tracing). Replaces the old pattern of
// mutating unexported fields ON the definition struct at register time —
// definitions are immutable shared data owned by internal/quests.
type indexedTrigger struct {
	def     *TriggerDef
	questId int
	trigId  string
}

// Engine holds quest definitions and a trigger index for fast lookup by event type.
type Engine struct {
	quests       map[int]*QuestDef
	triggerIndex map[string][]*indexedTrigger
}

// NewEngine creates an empty quest engine.
func NewEngine() *Engine {
	return &Engine{
		quests:       make(map[int]*QuestDef),
		triggerIndex: make(map[string][]*indexedTrigger),
	}
}

// RegisterQuest stores a quest definition and indexes its triggers by event type.
func (e *Engine) RegisterQuest(q *QuestDef) {
	e.quests[q.QuestId] = q
	for i := range q.Triggers {
		t := &q.Triggers[i]
		e.triggerIndex[t.Event] = append(e.triggerIndex[t.Event], &indexedTrigger{
			def:     t,
			questId: q.QuestId,
			trigId:  fmt.Sprintf("q%d-t%d", q.QuestId, i),
		})
	}
}

// GetQuest returns a quest definition by ID, or nil if not found.
func (e *Engine) GetQuest(questId int) *QuestDef {
	return e.quests[questId]
}

// AllQuests returns every registered quest, ordered by quest id. For audits and
// gates that need to walk the whole quest set, e.g. the marker-decision smoke
// gate.
//
// The order is guaranteed because callers print and diff these results; map
// iteration order made audit output shuffle between runs for no reason.
func (e *Engine) AllQuests() []*QuestDef {
	out := make([]*QuestDef, 0, len(e.quests))
	for _, q := range e.quests {
		out = append(out, q)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].QuestId < out[j].QuestId })
	return out
}

// Notify is the main entry point. It evaluates all triggers matching the given
// event type and returns a result indicating whether anything fired and whether
// an item should be consumed.
func (e *Engine) Notify(eventType string, details EventDetails, player PlayerState, ctx ActionContext) NotifyResult {
	bal := configs.GetBalanceConfig()
	maxDepth := int(bal.QuestChainDepthLimit)
	warnMs := int64(bal.QuestPerformanceWarnMs)

	guard := NewEvalGuard(maxDepth)
	start := time.Now()

	result := e.evaluate(eventType, details, player, ctx, guard)

	elapsed := time.Since(start).Milliseconds()
	if elapsed > warnMs {
		LogWarn("Notify(%s) for user %d took %dms (threshold %dms), trace: %v",
			eventType, details.UserId, elapsed, warnMs, guard.GetTrace())
	}
	if guard.DepthExceeded() {
		LogError("depth limit %d exceeded for user %d event %s, trace: %v",
			maxDepth, details.UserId, eventType, guard.GetTrace())
	}

	return result
}

// evaluate is the internal recursive function that matches triggers and
// executes actions. It chains through quest_granted events when tokens are granted.
func (e *Engine) evaluate(eventType string, details EventDetails, player PlayerState, ctx ActionContext, guard *EvalGuard) NotifyResult {
	if !guard.PushDepth() {
		return NotifyResult{}
	}
	defer guard.PopDepth()

	result := NotifyResult{}

	triggers := e.triggerIndex[eventType]
	for _, t := range triggers {
		// Field matching
		if !matchTriggerFields(t.def, details) {
			continue
		}

		// Visit tracking — prevent re-entry of same trigger in one chain
		if !guard.MarkVisited(t.trigId) {
			continue
		}

		// Condition evaluation
		if !EvalConditions(t.def.Conditions, player) {
			continue
		}

		guard.AddToTrace(fmt.Sprintf("%s:%s", eventType, t.trigId))
		LogVerboseF(details.UserId, "trigger %s matched for event %s", t.trigId, eventType)

		// Execute actions
		grantedTokens := e.executeActions(t, ctx, guard, details.UserId)

		result.Handled = true

		// Check for item consumption on item_give events
		if eventType == "item_give" {
			for _, a := range t.def.Actions {
				if a.ConsumeItem > 0 {
					result.ConsumeItem = true
					break
				}
			}
		}

		// Chain: for each granted token, evaluate quest_granted triggers
		for _, token := range grantedTokens {
			chainDetails := EventDetails{
				UserId:     details.UserId,
				RoomId:     details.RoomId,
				QuestToken: token,
			}
			chainResult := e.evaluate("quest_granted", chainDetails, player, ctx, guard)
			if chainResult.Handled {
				result.Handled = true
			}
		}
	}

	return result
}

// matchTriggerFields checks whether a trigger's field filters match the event details.
func matchTriggerFields(t *TriggerDef, d EventDetails) bool {
	if t.Mob > 0 && t.Mob != d.MobId {
		return false
	}
	if t.Room > 0 && t.Room != d.RoomId {
		return false
	}
	if t.Item > 0 && t.Item != d.ItemId {
		return false
	}
	if t.Skill != "" && t.Skill != d.Skill {
		return false
	}
	if t.Command != "" && t.Command != d.Command {
		return false
	}
	if t.Topic != "" && t.Topic != d.Topic {
		return false
	}
	if t.QuestToken != "" && t.QuestToken != d.QuestToken {
		return false
	}
	if t.Noun != "" && t.Noun != d.Noun {
		return false
	}
	if t.Verb != "" && t.Verb != d.Verb {
		return false
	}
	return true
}

// executeActions runs a trigger's actions in order, with panic recovery per
// action, and returns the tokens granted.
//
// A trigger's actions are applied as a unit: if one fails or panics, the
// remaining actions are ABANDONED. Previously each action was isolated and the
// loop carried on, which let a quest step give the player the item but never
// set the flag — a state no content author designed and one that is invisible
// apart from a log line. There is still no rollback of what already ran, so a
// failure leaves a partial effect; aborting simply stops it compounding.
//
// A duplicate grant is not a failure. It is skipped and the loop continues.
func (e *Engine) executeActions(t *indexedTrigger, ctx ActionContext, guard *EvalGuard, userId int) []string {
	var granted []string

	for i, a := range t.def.Actions {
		// ok reports whether the loop may proceed to the next action.
		ok := func() (ok bool) {
			defer func() {
				if r := recover(); r != nil {
					LogError("panic in action %d of trigger %s: %v (abandoning %d remaining action(s))",
						i, t.trigId, r, len(t.def.Actions)-i-1)
					ok = false
				}
			}()

			// For grant actions, check the guard first. A duplicate grant is a
			// no-op, not an error — keep going.
			if a.Grant != "" {
				if !guard.MarkGranted(a.Grant) {
					LogVerboseF(userId, "skipping duplicate grant %s", a.Grant)
					return true
				}
			}

			if err := ExecuteAction(a, ctx); err != nil {
				LogError("action %d of trigger %s failed: %v (abandoning %d remaining action(s))",
					i, t.trigId, err, len(t.def.Actions)-i-1)
				return false
			}

			if a.Grant != "" {
				granted = append(granted, a.Grant)
			}
			return true
		}()

		if !ok {
			break
		}
	}

	return granted
}
