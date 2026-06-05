package bounties

import "testing"

func TestGetMemoryUsage_NilRegistry_NoPanic(t *testing.T) {
	registryMu.Lock()
	registry = nil
	registryMu.Unlock()

	got := GetMemoryUsage()

	row, ok := got["registry"]
	if !ok {
		t.Fatalf("expected a 'registry' row even when registry is nil")
	}
	if row.Count != 0 {
		t.Fatalf("expected Count 0 for nil registry, got %d", row.Count)
	}
}
