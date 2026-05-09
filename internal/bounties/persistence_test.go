package bounties

import (
	"os"
	"sync"
	"testing"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
)

func TestSaveAndLoadRoundTrip(t *testing.T) {
	resetCache()

	r := &Registry{
		NextId: 2,
		Bounties: []*Bounty{
			{
				Id:             1,
				Issuer:         FactionIssuer("thornwall_guards"),
				Target:         knowledge.PlayerSubject(17),
				GoldReward:     300,
				RepReward:      6,
				Condition:      ConditionKill,
				DeclaredRound:  100,
				ExpiryRound:    1000,
				Status:         StatusOpen,
				DeclaredReason: "Murder of Marek",
			},
		},
	}

	if err := saveRegistry(r); err != nil {
		t.Fatalf("save failed: %v", err)
	}

	loaded := loadRegistryFromDisk()
	if loaded == nil {
		t.Fatalf("expected non-nil load")
	}
	if loaded.NextId != 2 {
		t.Errorf("NextId mismatch: got %d", loaded.NextId)
	}
	if len(loaded.Bounties) != 1 {
		t.Fatalf("expected 1 bounty, got %d", len(loaded.Bounties))
	}
	b := loaded.Bounties[0]
	if b.Issuer != FactionIssuer("thornwall_guards") {
		t.Errorf("issuer mismatch: %+v", b.Issuer)
	}
	if b.Target != knowledge.PlayerSubject(17) {
		t.Errorf("target mismatch: %+v", b.Target)
	}
	if b.GoldReward != 300 || b.RepReward != 6 {
		t.Errorf("rewards mismatch: gold=%d rep=%d", b.GoldReward, b.RepReward)
	}
	if b.DeclaredReason != "Murder of Marek" {
		t.Errorf("reason mismatch: %q", b.DeclaredReason)
	}

	// Clean up the saved file for next test
	os.Remove(registryFilePath())
}

func TestLoadMissingFileReturnsNil(t *testing.T) {
	resetCache()
	if loadRegistryFromDisk() != nil {
		t.Errorf("expected nil for missing file")
	}
}

func TestLoadOrLazyInitConcurrent(t *testing.T) {
	resetCache()
	const N = 50
	var wg sync.WaitGroup
	var seen [N]*Registry
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			seen[i] = loadOrLazyInit()
		}(i)
	}
	wg.Wait()

	first := seen[0]
	if first == nil {
		t.Fatalf("nil result from loadOrLazyInit")
	}
	for i := 1; i < N; i++ {
		if seen[i] != first {
			t.Errorf("goroutine %d got different pointer than goroutine 0", i)
		}
	}
}
