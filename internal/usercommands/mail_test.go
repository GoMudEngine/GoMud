package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/items"
	"github.com/GoMudEngine/GoMud/internal/users"
)

func TestMailOnCooldown(t *testing.T) {
	if !mailOnCooldown(100, 105, 10) {
		t.Error("round 105 is within 100+10; should be on cooldown")
	}
	if mailOnCooldown(100, 110, 10) {
		t.Error("round 110 == 100+10; cooldown elapsed")
	}
	if mailOnCooldown(100, 200, 10) {
		t.Error("well past cooldown should be clear")
	}
	if mailOnCooldown(0, 5, 10) {
		t.Error("never-sent (lastSent 0) should never be on cooldown")
	}
	if mailOnCooldown(100, 101, 0) {
		t.Error("cooldown 0 (disabled) should never block")
	}
}

func TestResolveMailRecipient(t *testing.T) {
	online := &users.UserRecord{UserId: 7, Character: &characters.Character{Name: "Onlinerecipient"}}

	onlineByName := func(name string) *users.UserRecord {
		if name == "Onlinerecipient" {
			return online
		}
		return nil
	}
	offlineSearch := func(name string) (int, string) {
		if name == "Offlinerecipient" {
			return 9, "offlineacct"
		}
		return 0, ""
	}

	rec, _, ok := resolveMailRecipient("Onlinerecipient", 1, onlineByName, offlineSearch)
	if !ok || !rec.online || rec.userId != 7 {
		t.Errorf("online resolve = %+v ok=%v, want {7, online}", rec, ok)
	}

	rec, uname, ok := resolveMailRecipient("Offlinerecipient", 1, onlineByName, offlineSearch)
	if !ok || rec.online || rec.userId != 9 || uname != "offlineacct" {
		t.Errorf("offline resolve = %+v uname=%q ok=%v, want {9, offline, offlineacct}", rec, uname, ok)
	}

	if _, _, ok := resolveMailRecipient("Nobody", 1, onlineByName, offlineSearch); ok {
		t.Error("unknown recipient should not resolve")
	}

	if _, _, ok := resolveMailRecipient("Onlinerecipient", 7, onlineByName, offlineSearch); ok {
		t.Error("self-mail (online) must be rejected")
	}
	if _, _, ok := resolveMailRecipient("Offlinerecipient", 9, onlineByName, offlineSearch); ok {
		t.Error("self-mail (offline) must be rejected")
	}
}

func TestApplyMailReceipt_GoldOnly(t *testing.T) {
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	msg := &users.Message{Gold: 250}
	if !applyMailReceipt(u, msg) {
		t.Fatal("gold-only receipt should commit")
	}
	if u.Character.Bank != 250 {
		t.Errorf("bank = %d, want 250", u.Character.Bank)
	}
}

func TestApplyMailReceipt_ItemFits(t *testing.T) {
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 8001, Name: "ring", Type: items.Ring, Value: 100}) // weightless -> fits
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	itm := items.New(8001)
	msg := &users.Message{Gold: 50, Item: &itm}
	if !applyMailReceipt(u, msg) {
		t.Fatal("fitting item should commit")
	}
	if u.Character.Bank != 50 {
		t.Errorf("bank = %d, want 50", u.Character.Bank)
	}
	if len(u.Character.Items) != 1 {
		t.Errorf("backpack items = %d, want 1", len(u.Character.Items))
	}
}

func TestApplyMailReceipt_OverCapacityDefers(t *testing.T) {
	// A very heavy item can't fit a zero-stat character -> StoreItem fails.
	items.RegisterTestItemSpec(&items.ItemSpec{ItemId: 8002, Name: "anvil", Type: items.Weapon, Value: 100, Weight: 100000})
	u := &users.UserRecord{UserId: 1, Character: &characters.Character{}}
	itm := items.New(8002)
	msg := &users.Message{Gold: 999, Item: &itm}

	if applyMailReceipt(u, msg) {
		t.Fatal("over-capacity item receipt must defer (return false)")
	}
	if u.Character.Bank != 0 {
		t.Errorf("bank = %d, want 0 (gold not credited when deferred)", u.Character.Bank)
	}
	if len(u.Character.Items) != 0 {
		t.Errorf("backpack items = %d, want 0", len(u.Character.Items))
	}
}
