package characters

// CraftingState tracks an active crafting operation for a character.
// Not persisted — the field on Character uses yaml:"-".
// Logging out mid-craft silently discards progress (same behaviour as CastingState).
type CraftingState struct {
	RecipeId       string
	RoundsTotal    int
	RoundsComplete int
}
