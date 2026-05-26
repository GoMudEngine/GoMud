package conversations

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/util"
)

// conversationPlan describes what the per-tick state machine wants to
// do for a single NPC. Pure-data, easy to unit-test.
type conversationPlan struct {
	ShouldFire  bool   // fire `say <text>` from this NPC this tick
	ShouldAbort bool   // abort the conversation (out-of-range line_idx, etc.)
	Text        string // the text to say (set when ShouldFire)
	NextLineIdx int    // shared line counter value AFTER this firing (set when ShouldFire)
	IsFinalLine bool   // true when this firing completes the exchange (set when ShouldFire)
}

// computeConversationPlan is the pure decision logic for a single NPC's
// tick: given the exchange, the NPC's role ("A" or "B"), and the shared
// line index, decide whether to fire, wait, or abort.
func computeConversationPlan(ex Exchange, role string, lineIdx int) conversationPlan {
	if lineIdx < 0 || lineIdx >= len(ex.Lines) {
		return conversationPlan{ShouldAbort: true}
	}
	line := ex.Lines[lineIdx]
	if line.Speaker != role {
		// Not my turn — wait silently.
		return conversationPlan{}
	}
	return conversationPlan{
		ShouldFire:  true,
		Text:        line.Text,
		NextLineIdx: lineIdx + 1,
		IsFinalLine: lineIdx == len(ex.Lines)-1,
	}
}

// MiscData key constants — single source of truth so callers in
// internal/hooks/ (eligibility gates) can use the same names.
const (
	MiscDataPartnerId          = "conversation_partner_id"
	MiscDataRole               = "conversation_role"
	MiscDataPoolId             = "conversation_pool_id"
	MiscDataPairOverrideId     = "conversation_pair_override_id"
	MiscDataExchangeId         = "conversation_exchange_id"
	MiscDataLineIdx            = "conversation_line_idx"
	MiscDataLastRound          = "conversation_last_round"
	MiscDataCooldownUntilRound = "conversation_cooldown_until_round"
)

// MobConversant is the interface conversations requires from a mob.
// *mobs.Mob satisfies this interface; callers in internal/hooks/ adapt
// the concrete type. This breaks the conversations ↔ mobs import cycle.
type MobConversant interface {
	// Identity
	ConvInstanceId() int
	ConvMobId() int
	ConvRoomId() int

	// Character MiscData
	ConvGetMiscData(key string) any
	ConvSetMiscData(key string, val any)

	// Character state checks
	ConvIsInCombat() bool
	ConvHasBuffFlag(flag buffs.Flag) bool
	ConvAggro() bool // true when character.Aggro != nil

	// Path state
	ConvPathLen() int
	ConvPathCurrentNonNil() bool

	// Action
	ConvCommand(text string)

	// Partner lookup (avoids passing mobs.GetInstance into this package)
	ConvGetPartner(instanceId int) MobConversant
}

// TryStart attempts to start a conversation initiated by `mob`. Picks
// a random in-room related partner, picks an exchange, stamps state on
// both NPCs, fires the first line immediately. No-op if no eligible
// partner / cooldowns active / no exchanges for the relationship type.
// roomMobIds is the list returned by room.GetMobs() — passed in so this
// package doesn't need to import rooms either.
func TryStart(initiator MobConversant, roomMobIds []int) {
	if initiator == nil {
		return
	}
	if isOnCooldown(initiator) {
		return
	}

	var candidates []MobConversant
	for _, otherInstId := range roomMobIds {
		if otherInstId == initiator.ConvInstanceId() {
			continue
		}
		other := initiator.ConvGetPartner(otherInstId)
		if other == nil {
			continue
		}
		if !relationships.AreRelated(initiator.ConvMobId(), other.ConvMobId()) {
			continue
		}
		if !isFullyIdle(other) || isOnCooldown(other) || isInConversation(other) {
			continue
		}
		candidates = append(candidates, other)
	}
	if len(candidates) == 0 {
		return
	}

	partner := candidates[util.Rand(len(candidates))]
	TryStartBetween(initiator, partner)
}

// TryStartBetween skips partner selection (caller has already identified
// the pair). Used by the player-arrival boost in go.go.
func TryStartBetween(a, b MobConversant) {
	if a == nil || b == nil {
		return
	}
	if isOnCooldown(a) || isOnCooldown(b) {
		return
	}
	if isInConversation(a) || isInConversation(b) {
		return
	}

	rels := relationships.RelationsBetween(a.ConvMobId(), b.ConvMobId())
	if len(rels) == 0 {
		return
	}
	rel := rels[util.Rand(len(rels))]
	subtype := rel.Subtype

	ex, ok := pickExchange(string(rel.Type), subtype, a.ConvMobId(), b.ConvMobId())
	if !ok {
		return
	}

	startConversation(a, b, ex, string(rel.Type))
}

// startConversation stamps state on both NPCs and fires the first line.
func startConversation(a, b MobConversant, ex Exchange, poolId string) {
	if len(ex.Lines) == 0 {
		return
	}
	speakerA, speakerB := a, b
	if util.Rand(2) == 0 {
		speakerA, speakerB = b, a
	}

	roundNow := util.GetRoundCount()
	exchangeId := int(roundNow)

	stampInitial := func(self MobConversant, partner MobConversant, role string) {
		self.ConvSetMiscData(MiscDataPartnerId, partner.ConvInstanceId())
		self.ConvSetMiscData(MiscDataRole, role)
		self.ConvSetMiscData(MiscDataPoolId, poolId)
		self.ConvSetMiscData(MiscDataPairOverrideId, "")
		self.ConvSetMiscData(MiscDataExchangeId, exchangeId)
		self.ConvSetMiscData(MiscDataLineIdx, 0)
	}
	stampInitial(speakerA, speakerB, "A")
	stampInitial(speakerB, speakerA, "B")

	cacheActiveExchange(exchangeId, ex)

	firstSpeaker := speakerA
	if ex.Lines[0].Speaker == "B" {
		firstSpeaker = speakerB
	}
	firstSpeaker.ConvCommand(fmt.Sprintf("say %s", ex.Lines[0].Text))

	firstSpeaker.ConvSetMiscData(MiscDataLastRound, roundNow)
	speakerA.ConvSetMiscData(MiscDataLineIdx, 1)
	speakerB.ConvSetMiscData(MiscDataLineIdx, 1)

	if len(ex.Lines) == 1 {
		finalizeConversation(speakerA, speakerB)
	}
}

// TickConversation drives an in-progress conversation for the given mob.
// Called each tick from the IdleMobs hook when partner_id MiscData is set.
func TickConversation(self MobConversant, partnerId int) {
	partner := self.ConvGetPartner(partnerId)
	if partner == nil || partner.ConvRoomId() != self.ConvRoomId() {
		abortConversation(self, partner)
		return
	}
	if !isFullyIdleForConversation(partner) {
		abortConversation(self, partner)
		return
	}

	exchangeId, _ := self.ConvGetMiscData(MiscDataExchangeId).(int)
	ex, ok := getActiveExchange(exchangeId)
	if !ok {
		abortConversation(self, partner)
		return
	}

	role, _ := self.ConvGetMiscData(MiscDataRole).(string)
	lineIdx, _ := self.ConvGetMiscData(MiscDataLineIdx).(int)

	plan := computeConversationPlan(ex, role, lineIdx)
	if plan.ShouldAbort {
		abortConversation(self, partner)
		return
	}
	if !plan.ShouldFire {
		return
	}

	roundNow := util.GetRoundCount()
	lastRound, _ := self.ConvGetMiscData(MiscDataLastRound).(uint64)
	if lastRound == roundNow {
		return
	}

	self.ConvCommand(fmt.Sprintf("say %s", plan.Text))
	self.ConvSetMiscData(MiscDataLastRound, roundNow)
	self.ConvSetMiscData(MiscDataLineIdx, plan.NextLineIdx)
	partner.ConvSetMiscData(MiscDataLineIdx, plan.NextLineIdx)

	if plan.IsFinalLine {
		finalizeConversation(self, partner)
	}
}

// finalizeConversation clears state on both NPCs and stamps cooldown.
func finalizeConversation(a, b MobConversant) {
	roundNow := util.GetRoundCount()
	cooldown := uint64(int(configs.GetBalanceConfig().ConversationCooldownRounds))
	until := roundNow + cooldown

	exchangeId, _ := a.ConvGetMiscData(MiscDataExchangeId).(int)
	clearActiveExchange(exchangeId)

	for _, m := range []MobConversant{a, b} {
		if m == nil {
			continue
		}
		m.ConvSetMiscData(MiscDataPartnerId, 0)
		m.ConvSetMiscData(MiscDataRole, "")
		m.ConvSetMiscData(MiscDataPoolId, "")
		m.ConvSetMiscData(MiscDataPairOverrideId, "")
		m.ConvSetMiscData(MiscDataExchangeId, 0)
		m.ConvSetMiscData(MiscDataLineIdx, 0)
		m.ConvSetMiscData(MiscDataLastRound, uint64(0))
		m.ConvSetMiscData(MiscDataCooldownUntilRound, until)
	}
}

// abortConversation clears state on both NPCs without stamping a cooldown.
func abortConversation(self MobConversant, partner MobConversant) {
	exchangeId, _ := self.ConvGetMiscData(MiscDataExchangeId).(int)
	clearActiveExchange(exchangeId)

	for _, m := range []MobConversant{self, partner} {
		if m == nil {
			continue
		}
		m.ConvSetMiscData(MiscDataPartnerId, 0)
		m.ConvSetMiscData(MiscDataRole, "")
		m.ConvSetMiscData(MiscDataPoolId, "")
		m.ConvSetMiscData(MiscDataPairOverrideId, "")
		m.ConvSetMiscData(MiscDataExchangeId, 0)
		m.ConvSetMiscData(MiscDataLineIdx, 0)
		m.ConvSetMiscData(MiscDataLastRound, uint64(0))
	}
}

// ── Helpers ──────────────────────────────────────────────────────────

func isOnCooldown(m MobConversant) bool {
	until, _ := m.ConvGetMiscData(MiscDataCooldownUntilRound).(uint64)
	return util.GetRoundCount() < until
}

func isInConversation(m MobConversant) bool {
	partnerId, _ := m.ConvGetMiscData(MiscDataPartnerId).(int)
	return partnerId > 0
}

// isFullyIdle: not in combat, not asleep, not in conversation, not on
// patrol mid-walk.
func isFullyIdle(m MobConversant) bool {
	if m == nil {
		return false
	}
	if m.ConvAggro() || m.ConvIsInCombat() {
		return false
	}
	if m.ConvHasBuffFlag(buffs.Sleeping) {
		return false
	}
	if m.ConvPathLen() > 0 || m.ConvPathCurrentNonNil() {
		return false
	}
	if isInConversation(m) {
		return false
	}
	return true
}

// isFullyIdleForConversation: like isFullyIdle but does not check patrol
// path (partner does not need to be mid-walk free, only not in combat/
// asleep/already talking).
func isFullyIdleForConversation(m MobConversant) bool {
	if m == nil {
		return false
	}
	if m.ConvAggro() || m.ConvIsInCombat() {
		return false
	}
	if m.ConvHasBuffFlag(buffs.Sleeping) {
		return false
	}
	if isInConversation(m) {
		return false
	}
	return true
}

// ── Character-level helpers (used by T7 hook directly) ───────────────

// IsInConversationChar reports whether a raw *characters.Character is
// currently in a conversation (partner_id MiscData > 0). Exported so
// the IdleMobs hook can gate without the full MobConversant wrapper.
func IsInConversationChar(c *characters.Character) bool {
	if c == nil {
		return false
	}
	partnerId, _ := c.GetMiscData(MiscDataPartnerId).(int)
	return partnerId > 0
}

// IsOnCooldownChar reports whether a raw *characters.Character is on
// conversation cooldown. Exported for the same reason.
func IsOnCooldownChar(c *characters.Character) bool {
	if c == nil {
		return false
	}
	until, _ := c.GetMiscData(MiscDataCooldownUntilRound).(uint64)
	return util.GetRoundCount() < until
}

// ── Active exchange cache ────────────────────────────────────────────

var (
	activeExchangesMu sync.RWMutex
	activeExchanges   = map[int]Exchange{}
)

func cacheActiveExchange(id int, ex Exchange) {
	activeExchangesMu.Lock()
	defer activeExchangesMu.Unlock()
	activeExchanges[id] = ex
}

func getActiveExchange(id int) (Exchange, bool) {
	activeExchangesMu.RLock()
	defer activeExchangesMu.RUnlock()
	ex, ok := activeExchanges[id]
	return ex, ok
}

func clearActiveExchange(id int) {
	activeExchangesMu.Lock()
	defer activeExchangesMu.Unlock()
	delete(activeExchanges, id)
}
