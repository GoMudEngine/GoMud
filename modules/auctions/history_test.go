package auctions

import (
	"fmt"
	"testing"
)

// newHistoryManager builds an AuctionManager with n past auctions, named
// "item-0".."item-(n-1)" in chronological order (oldest first, matching how
// EndAuction appends).
func newHistoryManager(n int) *AuctionManager {
	am := &AuctionManager{PastAuctions: []PastAuctionItem{}}
	for i := range n {
		am.PastAuctions = append(am.PastAuctions, PastAuctionItem{
			ItemName:   fmt.Sprintf("item-%d", i),
			WinningBid: i,
		})
	}
	return am
}

// TestRegression_GetAuctionHistoryPartialRequest locks the fix for the
// 2026-07-20 audit finding: GetAuctionHistory sliced
//
//	am.PastAuctions[len(am.PastAuctions)-totalItems : totalItems]
//
// using totalItems as the HIGH bound instead of the slice end. For a 10-item
// history, GetAuctionHistory(3) evaluated to PastAuctions[7:3] — low > high —
// a guaranteed "slice bounds out of range" panic. It fired whenever
// totalItems < len(PastAuctions)/2.
//
// The bug was dormant only because the sole caller passes 0, which returns
// early. Since panics in this codebase are not recovered at the dispatch loop,
// any future "show last N auctions" feature would have taken the server down.
func TestRegression_GetAuctionHistoryPartialRequest(t *testing.T) {
	t.Run("partial_request_does_not_panic", func(t *testing.T) {
		am := newHistoryManager(10)

		got := am.GetAuctionHistory(3)

		if len(got) != 3 {
			t.Fatalf("GetAuctionHistory(3) returned %d items, want 3", len(got))
		}
		// Must be the most RECENT three, i.e. the tail of the slice.
		want := []string{"item-7", "item-8", "item-9"}
		for i, w := range want {
			if got[i].ItemName != w {
				t.Errorf("item %d = %q, want %q", i, got[i].ItemName, w)
			}
		}
	})

	// The panic only triggered when totalItems < len/2, so sweep the range that
	// straddles that boundary rather than testing a single lucky value.
	t.Run("every_partial_size_is_safe", func(t *testing.T) {
		am := newHistoryManager(10)
		for n := 1; n <= 10; n++ {
			got := am.GetAuctionHistory(n)
			if len(got) != n {
				t.Errorf("GetAuctionHistory(%d) returned %d items, want %d", n, len(got), n)
			}
			if len(got) > 0 {
				last := got[len(got)-1]
				if last.ItemName != "item-9" {
					t.Errorf("GetAuctionHistory(%d) last item = %q, want the newest (item-9)", n, last.ItemName)
				}
			}
		}
	})

	t.Run("request_larger_than_history_is_clamped", func(t *testing.T) {
		am := newHistoryManager(4)

		got := am.GetAuctionHistory(99)

		if len(got) != 4 {
			t.Fatalf("GetAuctionHistory(99) on a 4-item history returned %d items, want 4", len(got))
		}
	})

	t.Run("zero_or_negative_returns_full_history", func(t *testing.T) {
		am := newHistoryManager(5)

		for _, n := range []int{0, -1} {
			got := am.GetAuctionHistory(n)
			if len(got) != 5 {
				t.Errorf("GetAuctionHistory(%d) returned %d items, want the full 5", n, len(got))
			}
		}
	})

	t.Run("empty_history_is_safe", func(t *testing.T) {
		am := newHistoryManager(0)

		if got := am.GetAuctionHistory(3); len(got) != 0 {
			t.Errorf("GetAuctionHistory(3) on empty history returned %d items, want 0", len(got))
		}
	})
}
