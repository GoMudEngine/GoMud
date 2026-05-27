package seeders

import (
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/opinions"
)

const ruleNameGiftToOpinionBoost = "gift_to_opinion_boost"
const giftCooldownRounds uint64 = 100

func init() {
	Register(ruleNameGiftToOpinionBoost, giftToOpinionBoost, "GiftAccepted")
}

// giftToOpinionBoost: on GiftAccepted (mob KEPT the item — not just
// received it), bump the mob's opinion of the giving player by N
// scaled to item value. Per-(giver, receiver) cooldown prevents spam.
//
// Value tiers per spec §3.7:
//
//	value 1-49     → +1
//	value 50-199   → +3
//	value 200-999  → +5
//	value 1000+    → +8
//
// Subscribes to GiftAccepted, NOT GiftOffered — GiftOffered fires on
// every give regardless of whether the mob valued the item. Worthless-
// rock spam doesn't bump opinion because the mob's consider-keep btree
// (chunk 2.3 equip-if-better) won't keep worthless items.
//
// Firer: internal/usercommands/give.go (Task 3).
func giftToOpinionBoost(event events.Event) {
	ga, ok := event.(events.GiftAccepted)
	if !ok {
		return
	}
	if ga.UserId == 0 || ga.MobInstanceId == 0 || ga.ItemId == 0 {
		return
	}

	receiver := mobs.GetInstance(ga.MobInstanceId)
	if receiver == nil {
		return
	}

	// Resolve item value via the items registry. GetItemSpec returns a
	// pointer — nil means the item id is unknown (e.g., the item was
	// deleted between the give and this handler firing).
	spec := items.GetItemSpec(ga.ItemId)
	if spec == nil || spec.Value <= 0 {
		return // unknown item OR worthless item — defensive
	}

	bump := giftValueToOpinionBump(spec.Value)
	if bump == 0 {
		return
	}

	// Cooldown: once per 100 rounds per (giver, receiver) pair.
	// Prevents legitimate-gift spam from inflating opinion without
	// real relationship investment.
	if !applyCooldown(receiver, ruleNameGiftToOpinionBoost, userIdAsKey(ga.UserId), giftCooldownRounds) {
		return // cooldown active
	}

	opinions.Bump(int(receiver.MobId), ga.UserId, bump)
}

// giftValueToOpinionBump maps item value to an opinion delta per
// spec §3.7 value-tier table.
func giftValueToOpinionBump(value int) int {
	switch {
	case value >= 1000:
		return 8
	case value >= 200:
		return 5
	case value >= 50:
		return 3
	case value >= 1:
		return 1
	}
	return 0
}
