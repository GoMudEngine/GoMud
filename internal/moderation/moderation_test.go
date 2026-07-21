package moderation

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPetitionQueue(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	p1, err := Add("Alice", 5209, "Sanctum", "Bob is spamming slurs at me")
	assert.NoError(t, err)
	assert.Equal(t, 1, p1.Id)
	assert.Equal(t, StatusOpen, p1.Status)

	p2, _ := Add("Cara", 100, "Town", "stuck in the well")
	assert.Equal(t, 2, p2.Id)

	assert.Len(t, ListOpen(), 2)
	assert.Len(t, ListAll(), 2)

	got, ok := Get(1)
	assert.True(t, ok)
	assert.Equal(t, "Alice", got.Reporter)

	assert.NoError(t, Resolve(1, "AdminZoe", "warned Bob"))
	assert.Len(t, ListOpen(), 1)
	r, _ := Get(1)
	assert.Equal(t, StatusResolved, r.Status)
	assert.Equal(t, "AdminZoe", r.ResolvedBy)

	assert.Error(t, Resolve(999, "x", ""))
}

func TestPetitionPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := SetDataDirForTest(dir)
	resetForTest()
	_, _ = Add("Alice", 5209, "Sanctum", "grief")
	_ = Resolve(1, "Zoe", "done")
	restore()

	// Reload from disk into a fresh in-memory state. Use the unexported loader
	// directly: the LoadDataFiles wrapper logs via mudlog, whose slog logger is
	// only initialized at real server boot (nil in unit tests).
	restore2 := SetDataDirForTest(dir)
	defer restore2()
	resetForTest()
	loadPetitions()
	all := ListAll()
	assert.Len(t, all, 1)
	assert.Equal(t, StatusResolved, all[0].Status)
	assert.Equal(t, "Alice", all[0].Reporter)
}
