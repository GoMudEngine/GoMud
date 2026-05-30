package hooks

import "testing"

func TestJailExpired(t *testing.T) {
	if jailExpired(0, 100) {
		t.Error("untilRound 0 (not jailed) must not be expired")
	}
	if jailExpired(100, 99) {
		t.Error("before untilRound must not be expired")
	}
	if !jailExpired(100, 100) {
		t.Error("at untilRound must be expired")
	}
	if !jailExpired(100, 150) {
		t.Error("past untilRound must be expired")
	}
}
