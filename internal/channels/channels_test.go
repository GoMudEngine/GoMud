package channels

import "testing"

func TestEnabled_DefaultOn(t *testing.T) {
	if !Enabled(nil) {
		t.Error("nil (unset) should be on")
	}
	if !Enabled(true) {
		t.Error("true should be on")
	}
	if Enabled(false) {
		t.Error("false should be off")
	}
	if !Enabled("garbage") {
		t.Error("non-bool should default on")
	}
}

func TestGetAndAll(t *testing.T) {
	if len(All()) != 3 {
		t.Fatalf("expected 3 channels, got %d", len(All()))
	}
	for _, name := range []string{"chat", "newbie", "trade"} {
		if _, ok := Get(name); !ok {
			t.Errorf("channel %q should resolve", name)
		}
	}
	if _, ok := Get("nope"); ok {
		t.Error("unknown channel must not resolve")
	}
}

func TestShouldReceive(t *testing.T) {
	// Sender always sees their own echo, even toggled off / deafened.
	if !ShouldReceive(true, true, false) {
		t.Error("sender should always receive")
	}
	// Non-sender, enabled, not deafened -> receives.
	if !ShouldReceive(false, false, nil) {
		t.Error("enabled non-sender should receive")
	}
	// Non-sender, toggled off -> not.
	if ShouldReceive(false, false, false) {
		t.Error("toggled-off non-sender must not receive")
	}
	// Non-sender, deafened -> not.
	if ShouldReceive(false, true, nil) {
		t.Error("deafened non-sender must not receive")
	}
}
