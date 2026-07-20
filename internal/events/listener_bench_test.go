package events

import (
	"testing"
)

// BenchmarkDoListeners measures the event dispatch hot path. DoListeners runs
// for every event, every round, so the per-listener panic recovery added for
// audit finding 1.1 needs to be cheap. Go's open-coded defers cost ~1ns when no
// panic occurs; this exists to confirm that empirically rather than assume it.
func BenchmarkDoListeners(b *testing.B) {
	for _, n := range []int{1, 4, 16} {
		b.Run(listenerCountName(n), func(b *testing.B) {
			ClearListeners()
			defer ClearListeners()

			for range n {
				RegisterListener(Buff{}, func(e Event) ListenerReturn {
					return Continue
				})
			}

			evt := Buff{BuffId: 1}

			b.ReportAllocs()
			b.ResetTimer()
			for range b.N {
				DoListeners(evt)
			}
		})
	}
}

func listenerCountName(n int) string {
	switch n {
	case 1:
		return "1_listener"
	case 4:
		return "4_listeners"
	default:
		return "16_listeners"
	}
}
