package usercommands

import (
	"strings"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/rooms"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// Full-command check of the assess reservation disclosure: supported forms
// come from the raise spells' own corpse-pool gates, and each band line
// describes the Conviction reservation without leaking a raw number.

const adUserId = 9131

func seedAssessFixture(t *testing.T, convictionMax int) (*users.UserRecord, *rooms.Room, func()) {
	t.Helper()

	cleanSpells := spells.SeedSpellsForTest(map[string]*spells.SpellData{
		"raise-skeleton": {
			SpellId: "raise-skeleton", Name: "Raise Skeleton",
			SummonMobId: 300, SummonRequiresCorpse: true,
			SummonMinCorpsePool: 30, SummonConvictionReserve: 350,
		},
		"raise-golem": {
			SpellId: "raise-golem", Name: "Raise Golem",
			SummonMobId: 305, SummonRequiresCorpse: true,
			SummonMinCorpsePool: 500, SummonConvictionReserve: 440,
		},
	})

	u := users.NewTestUser(adUserId, "assessor", "Assessor", uint64(adUserId))
	u.Character.ConvictionMax.Value = convictionMax
	cleanUsers := users.SeedUsersForTest(map[int]*users.UserRecord{adUserId: u})

	corpse := rooms.Corpse{
		MobId: 1234,
	}
	corpse.Character = *characters.New()
	corpse.Character.Name = "goblin"
	corpse.Character.Stats.Strength.Training = 20
	corpse.Character.Stats.Vitality.Training = 20 // total 40 → skeleton only

	room := &rooms.Room{RoomId: 999901, Corpses: []rooms.Corpse{corpse}}

	events.DrainQueuedMessagesForTest(adUserId)
	return u, room, func() {
		events.DrainQueuedMessagesForTest(adUserId)
		cleanUsers()
		cleanSpells()
	}
}

func TestAssess_DisclosesReservationBand(t *testing.T) {
	// Big pool: 350 reserve on 4000 max ≈ 9% → "a small part", affordable.
	u, room, cleanup := seedAssessFixture(t, 4000)
	defer cleanup()

	handled, err := Assess("goblin", u, room, events.CmdSecretly)
	if err != nil || !handled {
		t.Fatalf("Assess failed: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(adUserId), "\n")
	if !strings.Contains(msgs, "It could sustain: skeleton.") {
		t.Errorf("supported list wrong (want skeleton only from the spell gates):\n%s", msgs)
	}
	if !strings.Contains(msgs, "Raising a skeleton would set aside a small part of your conviction") {
		t.Errorf("missing/incorrect reservation disclosure:\n%s", msgs)
	}
	if strings.Contains(msgs, "could not spare") {
		t.Errorf("affordable reservation flagged as unaffordable:\n%s", msgs)
	}
	if strings.Contains(msgs, "350") {
		t.Errorf("raw reserve number leaked to the player:\n%s", msgs)
	}
}

func TestAssess_FlagsUnaffordableReservation(t *testing.T) {
	// Tiny pool: 350 reserve on 300 max → more than the pool, unaffordable.
	u, room, cleanup := seedAssessFixture(t, 300)
	defer cleanup()

	handled, err := Assess("goblin", u, room, events.CmdSecretly)
	if err != nil || !handled {
		t.Fatalf("Assess failed: handled=%v err=%v", handled, err)
	}

	msgs := strings.Join(events.DrainQueuedMessagesForTest(adUserId), "\n")
	if !strings.Contains(msgs, "more than your spirit could hold") {
		t.Errorf("over-pool reservation not described as such:\n%s", msgs)
	}
	if !strings.Contains(msgs, "could not spare") {
		t.Errorf("unaffordable reservation not flagged:\n%s", msgs)
	}
}
