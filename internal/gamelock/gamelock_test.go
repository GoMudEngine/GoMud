package gamelock

import "testing"

func TestLock_SetLockedIncrementsRotationSeed(t *testing.T) {
	l := Lock{Difficulty: 6}
	if l.RotationSeed != 0 {
		t.Fatalf("RotationSeed = %d, want 0 default", l.RotationSeed)
	}
	l.SetLocked()
	if l.RotationSeed != 1 {
		t.Fatalf("RotationSeed after first SetLocked = %d, want 1", l.RotationSeed)
	}
	l.SetUnlocked()
	l.SetLocked()
	if l.RotationSeed != 2 {
		t.Fatalf("RotationSeed after second SetLocked = %d, want 2", l.RotationSeed)
	}
}

func TestLock_RotationSeedDefaultsZero(t *testing.T) {
	var l Lock
	if l.RotationSeed != 0 {
		t.Errorf("zero-value Lock RotationSeed = %d, want 0", l.RotationSeed)
	}
}
