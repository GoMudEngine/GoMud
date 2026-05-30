package bountyhunter

import (
	"github.com/GoMudEngine/GoMud/internal/bounties"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// hunt tracks a single active bounty-hunter dispatch keyed by target userId.
type hunt struct {
	hunterInstanceId int
	bountyId         int
	issuerFaction    string
}

// activeHunts is keyed by target userId. ONE active hunter per player.
var activeHunts = map[int]*hunt{}

// lastKilledFor records the round a hunter was killed for a given target userId,
// so the redispatch cooldown can be enforced.
var lastKilledFor = map[int]uint64{}

// shouldDispatch is a pure predicate that decides whether the sweep should
// spawn a new hunter for a given player.
//
//   - maxBountyGold must be >= threshold to qualify.
//   - hasActive: a hunter is already pursuing this target — do not double-dispatch.
//   - cooldown: rounds that must elapse after the last hunter was killed before
//     we redispatch. Guards against uint64 underflow by requiring nowRound >= lastKilledRound.
func shouldDispatch(maxBountyGold, threshold int, hasActive bool, lastKilledRound, nowRound, cooldown uint64) bool {
	if maxBountyGold < threshold {
		return false
	}
	if hasActive {
		return false
	}
	// Guard against underflow; if somehow nowRound < lastKilledRound treat as
	// still-on-cooldown.
	if nowRound < lastKilledRound {
		return false
	}
	if nowRound-lastKilledRound < cooldown {
		return false
	}
	return true
}

// despawnHunter removes a live hunter from its room and the instance registry.
// It does NOT drop loot, create a corpse, or clean spawn slots — this is a
// silent administrative removal (bounty closed/expired while hunter is alive).
// Guards against nil instance.
func despawnHunter(instanceId int) {
	m := mobs.GetInstance(instanceId)
	if m == nil {
		return
	}
	if room := rooms.LoadRoom(m.Character.RoomId); room != nil {
		room.RemoveMob(m.InstanceId)
	}
	mobs.DestroyInstance(m.InstanceId)
}

// RunDispatchSweep is called once per round from NewRound_MobRoundTick.
//
// Phase 1 — Reap: inspect every active hunt.
//   - Bounty closed/claimed/expired → silently despawn the hunter and forget the hunt.
//   - Hunter dead (instance gone, bounty still open) → reprieve: record cooldown round,
//     forget the hunt (the hunter earned their kill credit separately via the
//     PlayerDeath_BountyResolve hook; we just clean up the registry here).
//
// Phase 2 — Dispatch: for every player who has at least one open faction bounty
// above the gold threshold and no active hunter, spawn a new hunter.
func RunDispatchSweep(nowRound uint64) {
	bal := configs.GetBalanceConfig()
	threshold := int(bal.BountyHunterGoldThreshold)
	cooldown := uint64(bal.BountyHunterRedispatchCooldown)

	// Phase 1: reap stale / resolved hunts.
	for uid, h := range activeHunts {
		b := bounties.Get(h.bountyId)
		if b == nil || b.Status != bounties.StatusOpen {
			// Bounty has been claimed, withdrawn, or expired — despawn the hunter.
			despawnHunter(h.hunterInstanceId)
			delete(activeHunts, uid)
			continue
		}
		if mobs.GetInstance(h.hunterInstanceId) == nil {
			// Hunter instance is gone — the hunter was killed.
			// Record the reprieve round and start the redispatch cooldown.
			lastKilledFor[uid] = nowRound
			delete(activeHunts, uid)
		}
	}

	// Phase 2: decide which players need a hunter dispatched.
	// Build a per-player summary: (maxGold, bestBountyId, issuerFaction).
	type candidate struct {
		maxGold  int
		bountyId int
		faction  string
	}
	candidates := map[int]*candidate{}

	for _, b := range bounties.AllOpen() {
		// Only player-target, faction-issued bounties trigger a hunter.
		if b.Target.Type != knowledge.SubjectPlayer {
			continue
		}
		if b.Issuer.Type != bounties.IssuerFaction {
			continue
		}
		uid := b.Target.Id
		c, ok := candidates[uid]
		if !ok || b.GoldReward > c.maxGold {
			candidates[uid] = &candidate{
				maxGold:  b.GoldReward,
				bountyId: b.Id,
				faction:  b.Issuer.Id,
			}
		}
	}

	for uid, c := range candidates {
		hasActive := activeHunts[uid] != nil
		if !shouldDispatch(c.maxGold, threshold, hasActive, lastKilledFor[uid], nowRound, cooldown) {
			continue
		}
		id := spawnHunter(uid, c.bountyId, c.maxGold, c.faction)
		if id > 0 {
			activeHunts[uid] = &hunt{
				hunterInstanceId: id,
				bountyId:         c.bountyId,
				issuerFaction:    c.faction,
			}
		}
	}
}
