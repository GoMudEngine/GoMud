package conversations

import (
	"io/fs"
	"os"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
)

func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	os.Exit(m.Run())
}

type fakeFS struct {
	files map[string][]byte
}

func newFakeFS(files map[string][]byte) *fakeFS {
	return &fakeFS{files: files}
}

func (f *fakeFS) ReadFile(name string) ([]byte, error) {
	if b, ok := f.files[name]; ok {
		return b, nil
	}
	return nil, fs.ErrNotExist
}

func (f *fakeFS) Open(name string) (fs.File, error) { return nil, fs.ErrNotExist }

func (f *fakeFS) KnownPaths() []string {
	paths := make([]string, 0, len(f.files))
	for p := range f.files {
		paths = append(paths, p)
	}
	return paths
}

func (f *fakeFS) AllFileSubSystems(yield func(fs.ReadFileFS) bool) { yield(f) }

func resetPluginState() {
	pluginFileSystems = nil
	converseCheckCache = map[string]bool{}
	conversations = map[int]*Conversation{}
	conversationCounter = map[string]int{}
	conversationUniqueId = 0
}

const sampleConversation = `-
  Supported:
    "*": ["*"]
  Conversation:
    - ["#1 say hello"]
    - ["#2 say hi"]
`

func TestHasConverseFile_FindsPluginFile(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	if !HasConverseFile(9001, "TestZone") {
		t.Fatalf("expected plugin conversation file to be found")
	}
	// Second call exercises the cache path.
	if !HasConverseFile(9001, "TestZone") {
		t.Fatalf("expected cached plugin conversation file to be found")
	}
}

func TestHasConverseFile_MissingReturnsFalse(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	if HasConverseFile(9002, "TestZone") {
		t.Fatalf("expected missing conversation to return false")
	}
}

func TestAttemptConversation_UsesPluginFile(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	convId := AttemptConversation(9001, 1, "goblin", 2, "rat", "TestZone")
	if convId == 0 {
		t.Fatalf("expected a non-zero conversation id from plugin file")
	}

	c := getConversation(convId)
	if c == nil {
		t.Fatalf("expected conversation to be stored")
	}
	if len(c.ActionList) != 2 {
		t.Fatalf("expected 2 conversation actions, got %d", len(c.ActionList))
	}
}

func TestReadPluginConversationFile_KeyFormat(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/frostfang/42.yaml`: []byte(sampleConversation),
	}))

	if _, ok := readPluginConversationFile("frostfang", 42); !ok {
		t.Fatalf("expected plugin conversation file at frostfang/42")
	}
	if _, ok := readPluginConversationFile("frostfang", 43); ok {
		t.Fatalf("did not expect a file for mob 43")
	}
}

// TestAttemptConversation_FreshConversationNotStale guards against a fresh
// conversation being eligible for the random cleanup in getConversation. A
// zero LastRound made new conversations look ancient (the round counter
// starts above one million), so any lookup that won the 2% maintenance roll
// deleted a conversation that was just created.
func TestAttemptConversation_FreshConversationNotStale(t *testing.T) {
	resetPluginState()
	defer resetPluginState()

	RegisterFS(newFakeFS(map[string][]byte{
		`conversations/testzone/9001.yaml`: []byte(sampleConversation),
	}))

	convId := AttemptConversation(9001, 1, "goblin", 2, "rat", "TestZone")
	if convId == 0 {
		t.Fatalf("expected a non-zero conversation id from plugin file")
	}

	c := conversations[convId]
	if c == nil {
		t.Fatalf("expected conversation to be stored")
	}
	if c.LastRound != util.GetRoundCount() {
		t.Fatalf("expected fresh conversation LastRound to equal the current round %d, got %d", util.GetRoundCount(), c.LastRound)
	}
}
