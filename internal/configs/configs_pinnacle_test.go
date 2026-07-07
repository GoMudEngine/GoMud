package configs

import "testing"

func TestPinnacleConfigDefaults(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.BandolierAttuneRounds != 10 {
		t.Fatalf("BandolierAttuneRounds = %d, want 10", int(b.BandolierAttuneRounds))
	}
	if b.SentientChatterCooldownRounds != 20 {
		t.Fatalf("SentientChatterCooldownRounds = %d, want 20", int(b.SentientChatterCooldownRounds))
	}
}
