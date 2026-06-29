package questengine

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	yaml "gopkg.in/yaml.v3"
)

// TestQuest35_GiveDaggerCompletesFirstHeat locks the fix for the newbie-area
// "First Heat stalls at 66%" bug (Malia playtest, 2026-06-29). Before the fix,
// quest 35 had exactly ONE completion path for the craft→end step: the
// craft_turnin dialogue node (`ask rusk done`). A player who did the intuitive
// thing — physically GIVE the finished iron dagger (item 10009) to Smith Rusk
// (mob 9116) — fired only the generic archetype player_give handler (hands it
// back, does nothing for the quest), leaving the quest parked at 35-craft.
//
// This test loads the REAL quest 35 YAML and asserts that an item_give of the
// iron dagger to Rusk, by a player who has already crafted (35-start + 35-craft,
// missing 35-end), grants 35-end. It is the give-path counterpart to the
// existing dialogue turn-in.
func TestQuest35_GiveDaggerCompletesFirstHeat(t *testing.T) {
	const (
		ruskMobId     = 9116
		ironDaggerId  = 10009
		startToken    = "35-start"
		craftToken    = "35-craft"
		endToken      = "35-end"
		questYAMLPath = "../../_datafiles/world/dogmud/quests/35-first_heat.yaml"
	)

	data, err := os.ReadFile(questYAMLPath)
	require.NoError(t, err, "must be able to read the real quest 35 YAML")

	var q QuestDef
	require.NoError(t, yaml.Unmarshal(data, &q), "quest 35 YAML must unmarshal")
	require.Equal(t, 35, q.QuestId, "loaded the wrong quest file")

	e := NewEngine()
	e.RegisterQuest(&q)

	// Player has crafted the dagger but not yet turned it in.
	player := newFullMockPlayer(5245)
	player.quests[startToken] = true
	player.quests[craftToken] = true
	player.items[ironDaggerId] = true
	ctx := newFullMockActionContext(1, player)

	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  ruskMobId,
		ItemId: ironDaggerId,
	}, player, ctx)

	assert.True(t, result.Handled,
		"giving the iron dagger to Rusk must be intercepted by a quest trigger")
	assert.True(t, player.HasQuest(endToken),
		"giving the finished dagger to Rusk must complete First Heat (grant 35-end)")
}

// TestQuest69_GiveRubbingAdvancesGalleryStep locks the same fix for the Gallery
// Cipher quest (found by the F3 audit, 2026-06-29). The quest log says "Take the
// rubbing to Dross", but the only path that advanced rubbing→gallery was the
// q69_gallery dialogue node (`ask dross rubbing`). A player who physically gave
// the rubbing (item 40115) to Dross (mob 9360) silently transferred it with no
// quest advancement — identical class to the First Heat bug. This test loads the
// real quest 69 YAML and asserts the give-path now grants 69-gallery.
func TestQuest69_GiveRubbingAdvancesGalleryStep(t *testing.T) {
	const (
		drossMobId   = 9360
		rubbingItem  = 40115
		rubbingToken = "69-rubbing"
		galleryToken = "69-gallery"
		questPath    = "../../_datafiles/world/dogmud/quests/69-the_gallery_cipher.yaml"
	)

	data, err := os.ReadFile(questPath)
	require.NoError(t, err, "must be able to read the real quest 69 YAML")

	var q QuestDef
	require.NoError(t, yaml.Unmarshal(data, &q), "quest 69 YAML must unmarshal")
	require.Equal(t, 69, q.QuestId, "loaded the wrong quest file")

	e := NewEngine()
	e.RegisterQuest(&q)

	// Player has the rubbing but has not yet been sent to the gallery.
	player := newFullMockPlayer(5902)
	player.quests[rubbingToken] = true
	player.items[rubbingItem] = true
	ctx := newFullMockActionContext(1, player)

	result := e.Notify("item_give", EventDetails{
		UserId: 1,
		MobId:  drossMobId,
		ItemId: rubbingItem,
	}, player, ctx)

	assert.True(t, result.Handled,
		"giving the rubbing to Dross must be intercepted by a quest trigger")
	assert.True(t, player.HasQuest(galleryToken),
		"giving the rubbing to Dross must advance the quest (grant 69-gallery)")
	assert.True(t, player.HasItem(rubbingItem),
		"the player must keep a rubbing (needed at the gallery and at turn-in)")
}
