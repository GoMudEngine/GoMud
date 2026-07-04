package configs

import "testing"

func TestPinnacleConfigDefaults(t *testing.T) {
	b := Balance{}
	b.Validate()
	if b.BandolierAttuneRounds <= 0 {
		t.Fatalf("BandolierAttuneRounds default expected >0, got %d", b.BandolierAttuneRounds)
	}
	if b.SentientChatterCooldownRounds <= 0 {
		t.Fatalf("SentientChatterCooldownRounds default expected >0, got %d", b.SentientChatterCooldownRounds)
	}
}
