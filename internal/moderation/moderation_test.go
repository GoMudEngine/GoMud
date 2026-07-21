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
	// Already-resolved guard: re-resolving #1 must error.
	assert.Error(t, Resolve(1, "AdminZoe", "again"))
}

func TestAccountBans(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	assert.NoError(t, BanAccount("Griefer", "spamming slurs", "AdminZoe"))
	reason, banned := IsAccountBanned("griefer") // case-insensitive
	assert.True(t, banned)
	assert.Equal(t, "spamming slurs", reason)

	_, banned = IsAccountBanned("SomeoneElse")
	assert.False(t, banned)

	assert.NoError(t, Unban("GRIEFER"))
	_, banned = IsAccountBanned("Griefer")
	assert.False(t, banned)
}

// Guards against the key-normalization drift between store and reload: a
// username banned with surrounding whitespace must still match after a restart.
func TestAccountBanWhitespaceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := SetDataDirForTest(dir)
	resetForTest()
	assert.NoError(t, BanAccount(" Bob ", "grief", "Zoe"))
	_, banned := IsAccountBanned("bob")
	assert.True(t, banned, "trimmed+lowered lookup should match pre-reload")
	restore()

	restore2 := SetDataDirForTest(dir)
	defer restore2()
	resetForTest()
	loadBans()
	_, banned = IsAccountBanned("BOB")
	assert.True(t, banned, "lookup must still match after reload")
}

func TestIPBans(t *testing.T) {
	restore := SetDataDirForTest(t.TempDir())
	defer restore()
	resetForTest()

	assert.NoError(t, BanIP("203.0.113.7", "evasion", "AdminZoe"))
	reason, banned := IsIPBanned("203.0.113.7")
	assert.True(t, banned)
	assert.Equal(t, "evasion", reason)

	_, banned = IsIPBanned("198.51.100.1")
	assert.False(t, banned)

	assert.NoError(t, UnbanIP("203.0.113.7"))
	_, banned = IsIPBanned("203.0.113.7")
	assert.False(t, banned)
}

func TestBanPersistenceRoundTrip(t *testing.T) {
	dir := t.TempDir()
	restore := SetDataDirForTest(dir)
	resetForTest()
	_ = BanAccount("Griefer", "grief", "Zoe")
	_ = BanIP("203.0.113.7", "evasion", "Zoe")
	restore()

	// Use the unexported loaders directly (LoadDataFiles logs via mudlog, which
	// is only initialized at real server boot).
	restore2 := SetDataDirForTest(dir)
	defer restore2()
	resetForTest()
	loadBans()
	_, a := IsAccountBanned("griefer")
	_, i := IsIPBanned("203.0.113.7")
	assert.True(t, a)
	assert.True(t, i)
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
	assert.False(t, all[0].Timestamp.IsZero(), "timestamp should survive the round-trip")

	// nextPetitionId must be restored from the max loaded Id so a post-reload
	// Add does not collide (regression guard for loadPetitions' id scan).
	p, _ := Add("Dan", 1, "Town", "next id check")
	assert.Equal(t, 2, p.Id)
}
