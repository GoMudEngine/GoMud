package messaging

import "testing"

// RenderForRecipient is the per-recipient pipeline entry point used
// internally by the Room/UserRecord Send helpers. Returns the final
// text to deliver. An empty return string means "don't deliver to
// this recipient" (used by the sight gate).
func TestRenderForRecipientStubReturnsTextUnchanged(t *testing.T) {
	got := RenderForRecipient(RenderInput{
		Category:  CategoryDefault,
		Text:      "Hello, world.",
		Channel:   ChannelAudio,
		LineWidth: 80,
	})
	if got != "Hello, world." {
		t.Fatalf("stub pipeline mutated text: got %q", got)
	}
}

func TestChannelConstants(t *testing.T) {
	if ChannelAudio == ChannelVisual {
		t.Fatal("ChannelAudio and ChannelVisual must differ")
	}
}

func TestApplyCategoryColorWrapsTagForKnownCategory(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "strikes deeply")
	want := `<ansi fg="hit-melee">strikes deeply</ansi>`
	if got != want {
		t.Fatalf("color wrap: got %q want %q", got, want)
	}
}

func TestApplyCategoryColorDefaultPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryDefault, "plain text")
	if got != "plain text" {
		t.Fatalf("CategoryDefault must pass text through unchanged, got %q", got)
	}
}

func TestApplyCategoryColorEmptyTextPassesThrough(t *testing.T) {
	got := applyCategoryColor(CategoryHitMelee, "")
	if got != "" {
		t.Fatalf("empty text must pass through unchanged, got %q", got)
	}
}

// TestShouldWrapTable encodes the opt-in wrap policy: only
// narrative-prose categories get server-side wrap; everything else
// passes through because its senders pre-format layout or emit short
// single-line feedback.
func TestShouldWrapTable(t *testing.T) {
	mustWrap := []Category{
		CategoryHitMelee, CategoryDodge, CategoryParry, CategoryBlock,
		CategorySpellElemental, CategorySpellVital, CategorySpellFold,
		CategoryGrappleFlow, CategorySubmission, CategoryDeath,
		CategoryBroadcast,
	}
	for _, c := range mustWrap {
		if !shouldWrap(c) {
			t.Errorf("category %q should be in the wrap opt-in set", c)
		}
	}
	mustPassThrough := []Category{
		CategoryRoomDescription,  // template-laid side-by-side
		CategorySkillProgress,    // fixed-width banner
		CategorySystem,           // tables + short feedback
		CategoryError,            // short single-line
		CategoryWarning,          // short single-line
		CategoryTip,              // hand-formatted
		CategoryLoot,             // "You pick up X"
		CategoryEquipment,        // "You wear X"
		CategorySpeech,           // "X says, 'Hello'"
		CategoryWhisper,          // short DMs
		CategoryShout,            // short
		CategoryOOC,              // short
		CategoryEmote,            // short
		CategoryNPCDialogue,      // hand-authored
		CategoryDialogueHint,     // hand-authored
		CategoryMobIdle,          // short ambient
		CategoryMobEmote,         // short directed
		CategoryRoomEntry,        // "X arrives from the south"
		CategoryRoomExit,         // "X leaves to the north"
		CategoryLogin,            // "X has logged in"
		CategoryLogout,           // "X has gone offline"
		CategoryBuffApply,        // hand-formatted
		CategoryBuffExpire,       // hand-formatted
		CategoryDefault,          // unchanged behavior
	}
	for _, c := range mustPassThrough {
		if shouldWrap(c) {
			t.Errorf("category %q must NOT wrap (would break pre-formatted output or wrap short single-line text)", c)
		}
	}
}

// TestRoomDescriptionSkipsWrap is the regression test for the
// look-command minimap layout bug — descriptions/room templates
// render a side-by-side block (prose on the left, minimap column on
// the right). The pipeline's wrap stage would otherwise wrap each
// line at LineWidth and shatter the side-by-side layout.
func TestRoomDescriptionSkipsWrap(t *testing.T) {
	// Line containing what looks like a side-by-side row: text + many
	// spaces + map column. Total length 90 > LineWidth 40, but the
	// pipeline must NOT wrap this — the template owns the column.
	row := "the cobblestone road winds west          " + "║·····║"
	got := RenderForRecipient(RenderInput{
		Category:  CategoryRoomDescription,
		Text:      row,
		Channel:   ChannelAudio,
		LineWidth: 40,
	})
	// The line should still be one line (no \n introduced by wrap).
	// Color stage may add an ANSI tag wrapper, but no newlines.
	for _, ch := range got {
		if ch == '\n' {
			t.Fatalf("CategoryRoomDescription must not wrap (side-by-side layout would break); got %q", got)
		}
	}
}

// TestHitMeleeWrapStillFires confirms the skip table is targeted —
// combat narration still wraps at LineWidth for narrow terminals.
func TestHitMeleeWrapStillFires(t *testing.T) {
	long := "the rusty longsword bites through cloth and into flesh causing a serious wound to the defender's left shoulder"
	got := RenderForRecipient(RenderInput{
		Category:  CategoryHitMelee,
		Text:      long,
		Channel:   ChannelAudio,
		LineWidth: 40,
	})
	// Should contain at least one newline — wrap did fire.
	hasNewline := false
	for _, ch := range got {
		if ch == '\n' {
			hasNewline = true
			break
		}
	}
	if !hasNewline {
		t.Fatalf("CategoryHitMelee must still wrap at LineWidth=40, got passthrough %q", got)
	}
}
