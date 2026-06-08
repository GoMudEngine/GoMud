package users

import "testing"

func TestUserTicks_SetAssignsIdAndUpdates(t *testing.T) {
	u := &UserRecord{}
	a := u.SetTick(UserTick{Name: "sip", Commands: "drink health", IntervalSec: 30, Enabled: true})
	if a.Id == 0 || len(u.Ticks) != 1 {
		t.Fatalf("create failed: %+v", u.Ticks)
	}
	a.Name = "sip2"
	u.SetTick(a)
	if len(u.Ticks) != 1 || u.Ticks[0].Name != "sip2" {
		t.Fatalf("update should not duplicate: %+v", u.Ticks)
	}
	b := u.SetTick(UserTick{Name: "fast", IntervalSec: 0})
	if b.IntervalSec != 1 {
		t.Fatalf("interval floor not applied: %d", b.IntervalSec)
	}
	u.RemoveTick(a.Id)
	for _, tk := range u.Ticks {
		if tk.Id == a.Id {
			t.Fatalf("remove failed")
		}
	}
}

func TestUserTriggers_SetAssignsIdAndUpdates(t *testing.T) {
	u := &UserRecord{}
	a := u.SetTrigger(UserTrigger{Name: "heal", Pattern: "*bleeding*", ThenCmds: "cast heal", Enabled: true})
	if a.Id == 0 || len(u.Triggers) != 1 {
		t.Fatalf("create failed: %+v", u.Triggers)
	}
	a.Name = "heal2"
	u.SetTrigger(a)
	if len(u.Triggers) != 1 || u.Triggers[0].Name != "heal2" {
		t.Fatalf("update dup: %+v", u.Triggers)
	}
	u.RemoveTrigger(a.Id)
	for _, tr := range u.Triggers {
		if tr.Id == a.Id {
			t.Fatalf("remove failed")
		}
	}
}
