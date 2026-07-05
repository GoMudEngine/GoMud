package actions

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/buffs"
	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/forager"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/messaging"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/rooms"
)

// ---------------------------------------------------------------------------
// forage-test-specific helpers
// ---------------------------------------------------------------------------

// forageFakeActor is a minimal Actor with a configurable room. It records
// SendText messages for assertion. It satisfies the full Actor interface.
type forageFakeActor struct {
	char      *characters.Character
	room      *rooms.Room
	name      string
	isPlayer  bool
	userId    int
	mobInstId int
	sent      []string
}

func newForageFakeActor(t *testing.T, name string, room *rooms.Room, isPlayer bool, userId int) *forageFakeActor {
	t.Helper()
	c := &characters.Character{
		Name:  name,
		Buffs: buffs.New(),
	}
	c.Stats.Perception.ValueAdj = 100
	return &forageFakeActor{
		char:     c,
		room:     room,
		name:     name,
		isPlayer: isPlayer,
		userId:   userId,
	}
}

func newForageMobActor(t *testing.T, mob *mobs.Mob, room *rooms.Room) *forageFakeActor {
	t.Helper()
	c := &characters.Character{
		Name:  mob.Character.Name,
		Buffs: buffs.New(),
	}
	return &forageFakeActor{
		char:      c,
		room:      room,
		name:      mob.Character.Name,
		isPlayer:  false,
		mobInstId: mob.InstanceId,
	}
}

func (a *forageFakeActor) GetCharacter() *characters.Character    { return a.char }
func (a *forageFakeActor) GetRoom() *rooms.Room                   { return a.room }
func (a *forageFakeActor) GetName() string                        { return a.name }
func (a *forageFakeActor) IsPlayer() bool                         { return a.isPlayer }
func (a *forageFakeActor) GetUserId() int                         { return a.userId }
func (a *forageFakeActor) GetMobInstanceId() int                  { return a.mobInstId }
func (a *forageFakeActor) AddBuff(_ int, _ string)                {}
func (a *forageFakeActor) OnSkillUse(_ string) bool               { return false }
func (a *forageFakeActor) OnStatUse(_ string) bool                { return false }
func (a *forageFakeActor) OnCriticalSuccess(_ string)             {}
func (a *forageFakeActor) OnCriticalFailure(_ string)             {}
func (a *forageFakeActor) SendRoomCommunication(_ string, _ bool) {}
func (a *forageFakeActor) SendText(_ messaging.Category, msg string) {
	a.sent = append(a.sent, msg)
}

// newForageTestRoom builds a minimal Room with the specified biome string.
// biomeId is written directly to Room.Biome; the biome registry lookup in
// GetBiome() will either find it (if seeded) or fall back to "default".
func newForageTestRoom(t *testing.T, roomId int, biomeId string) *rooms.Room {
	t.Helper()
	return &rooms.Room{RoomId: roomId, Biome: biomeId}
}

// newForageTestMob builds a minimal Mob with an InstanceId.
func newForageTestMob(t *testing.T, instId int, name string, roomId int) *mobs.Mob {
	t.Helper()
	m := &mobs.Mob{
		InstanceId: instId,
	}
	m.Character.Name = name
	m.Character.Buffs = buffs.New()
	m.Character.RoomId = roomId
	return m
}

// seedForageBiomes registers a minimal forest biome so that rooms with
// Biome: "forest" resolve correctly via rooms.GetBiome in test context.
// Returns a cleanup function to restore the original registry.
func seedForageBiomes(t *testing.T) func() {
	t.Helper()
	return rooms.SeedBiomesForTest(map[string]*rooms.BiomeInfo{
		"forest": {
			BiomeId:      "forest",
			Name:         "Forest",
			Symbol:       "T",
			LitArea:      false,
			DarkArea:     false,
			MovementCost: 1.5,
		},
		"default": {
			BiomeId:      "default",
			Name:         "Default",
			Symbol:       "•",
			LitArea:      true,
			DarkArea:     false,
			MovementCost: 1.0,
		},
	})
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestForage_NonForagableBiomeReturnsEmpty confirms the action
// returns Found=false with a reason when the actor's biome has
// no entry in forager.ForageYields. An empty/unset biome resolves
// to the "default" biomeId which is not in ForageYields.
func TestForage_NonForagableBiomeReturnsEmpty(t *testing.T) {
	// Room.Biome="" → GetBiome() falls back to "default" (BiomeId="default").
	// "default" is not in ForageYields → biome gate fires.
	room := newForageTestRoom(t, 9301, "")
	user := newForageFakeActor(t, "ForageTester", room, true, 1)

	result := Forage(user, ForageOptions{})

	if result.Found {
		t.Error("expected Found=false for non-foragable biome")
	}
	if result.Reason == "" {
		t.Error("expected Reason to be set on biome failure")
	}
	if result.RollHappened {
		t.Error("expected RollHappened=false (no roll on biome failure)")
	}
}

// TestForage_CooldownGate confirms a second call within 6 rounds
// returns OnCooldown=true and skips the roll.
func TestForage_CooldownGate(t *testing.T) {
	cleanup := seedForageBiomes(t)
	defer cleanup()

	room := newForageTestRoom(t, 9302, "forest")
	user := newForageFakeActor(t, "ForageTester2", room, true, 2)

	_ = Forage(user, ForageOptions{})
	second := Forage(user, ForageOptions{})

	if !second.OnCooldown {
		t.Error("second call within 6-round window should return OnCooldown=true")
	}
	if second.RollHappened {
		t.Error("OnCooldown path should not roll")
	}
}

// TestForage_ZoneOverlayPlayerOnly proves the zone-forage overlay is gated
// on IsPlayer(). Forage() is a SHARED entry point — both the player command
// and the NPC forager path (behaviortree/mobcommands → actions.Forage) call
// it — so the zone/weather overlay must only apply for player actors. Here a
// UserActor foraging in a zone with a seeded overlay CAN receive the overlay
// item, while a MobActor foraging in the SAME zone NEVER does. That gate is
// the guard against zone/weather-gated ultra-rare reagents leaking into
// vendor stock via forager NPCs.
func TestForage_ZoneOverlayPlayerOnly(t *testing.T) {
	cleanupBiomes := seedForageBiomes(t)
	defer cleanupBiomes()

	// Seed ONLY the overlay item as a valid spec. Every base forest
	// forageable is therefore invalid, so the ONLY way Forage() reports
	// Found=true is by picking the overlay item — a clean observable signal
	// (items aren't loaded in unit-test context, so an unseeded pick yields
	// IsValid()=false → Found stays false).
	const overlayItemId = 999123
	cleanupItems := items.SeedItemsForTest(map[int]*items.ItemSpec{
		overlayItemId: {
			ItemId:  overlayItemId,
			Name:    "Overlay Reagent",
			Type:    items.Object,
			Subtype: items.Mundane,
		},
	})
	defer cleanupItems()

	const zone = "Overlay Test Zone"
	forager.ZoneForageYields[zone] = []int{overlayItemId}
	defer delete(forager.ZoneForageYields, zone)

	const trials = 200

	// forageOnce runs one fresh forage attempt (fresh actor each call so the
	// 6-round cooldown never collides) and reports whether the overlay item
	// was received. A very high Perception guarantees the search roll clears
	// the forest difficulty essentially every time, so a pool pick happens.
	forageOnce := func(isPlayer bool) bool {
		room := &rooms.Room{RoomId: 9400, Biome: "forest", Zone: zone}
		var actor *forageFakeActor
		if isPlayer {
			// userId 0 keeps the test hermetic (skips the ItemOwnership
			// event); IsPlayer() reads the bool field, not the userId.
			actor = newForageFakeActor(t, "OverlayPlayer", room, true, 0)
		} else {
			mob := newForageTestMob(t, 9400, "OverlayMob", room.RoomId)
			actor = newForageMobActor(t, mob, room)
		}
		actor.char.Stats.Perception.ValueAdj = 500
		res := Forage(actor, ForageOptions{})
		return res.Found && res.ItemId == overlayItemId
	}

	mobHits := 0
	for i := 0; i < trials; i++ {
		if forageOnce(false) {
			mobHits++
		}
	}
	if mobHits != 0 {
		t.Fatalf("MobActor received the zone-overlay item %d time(s) over %d forages; the overlay MUST be gated on IsPlayer and never reach NPC foragers", mobHits, trials)
	}

	playerHits := 0
	for i := 0; i < trials; i++ {
		if forageOnce(true) {
			playerHits++
		}
	}
	if playerHits == 0 {
		t.Fatalf("UserActor never received the zone-overlay item over %d forages; the player overlay path is not wiring Zone into ForageCore", trials)
	}
}

// TestForage_MobActorSilent confirms MobActor.SendText is not called.
func TestForage_MobActorSilent(t *testing.T) {
	cleanup := seedForageBiomes(t)
	defer cleanup()

	room := newForageTestRoom(t, 9303, "forest")
	mob := newForageTestMob(t, 9999, "ForageMob", room.RoomId)
	actor := newForageMobActor(t, mob, room)

	_ = Forage(actor, ForageOptions{})

	if len(actor.sent) != 0 {
		t.Errorf("MobActor should be silent; got %d messages", len(actor.sent))
	}
}
