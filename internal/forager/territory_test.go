package forager

import "testing"

func TestProfileFor_KnownIds(t *testing.T) {
	cases := []int{371, 243, 366}
	for _, id := range cases {
		if ProfileFor(id) == nil {
			t.Errorf("ProfileFor(%d) = nil, want profile", id)
		}
	}
}

func TestProfileFor_UnknownReturnsNil(t *testing.T) {
	if ProfileFor(99999) != nil {
		t.Error("expected nil for unknown mob id")
	}
}

func TestAllProfilesHasThree(t *testing.T) {
	if got := len(AllProfiles()); got != 3 {
		t.Errorf("AllProfiles() len = %d, want 3", got)
	}
}

func TestAllProfilesStableOrder(t *testing.T) {
	got := AllProfiles()
	wantKinds := []ForagerKind{KindMarsh, KindSteppe, KindFernway}
	for i, want := range wantKinds {
		if got[i].Kind != want {
			t.Errorf("AllProfiles()[%d].Kind = %v, want %v", i, got[i].Kind, want)
		}
	}
}

func TestProfileBucketsNonEmpty(t *testing.T) {
	for _, p := range AllProfiles() {
		if len(p.Buckets) == 0 {
			t.Errorf("profile %s has empty Buckets", p.Name)
		}
	}
}

func TestFernwayHasMeetingRoom(t *testing.T) {
	if ProfileFor(366).MeetingRoom != 4038 {
		t.Error("Fernway forager missing meeting room 4038")
	}
}

func TestMarshAndSteppeNoMeetingRoom(t *testing.T) {
	if ProfileFor(371).MeetingRoom != 0 {
		t.Error("Marsh forager should not have meeting room")
	}
	if ProfileFor(243).MeetingRoom != 0 {
		t.Error("Steppe forager should not have meeting room")
	}
}

func TestVendorRoomsExclusiveToTownForagers(t *testing.T) {
	if len(ProfileFor(371).VendorRooms) == 0 {
		t.Error("Marsh forager should have vendor rooms")
	}
	if len(ProfileFor(243).VendorRooms) == 0 {
		t.Error("Steppe forager should have vendor rooms")
	}
	if len(ProfileFor(366).VendorRooms) != 0 {
		t.Error("Fernway forager should not have vendor rooms (caravan handoff)")
	}
}

func TestPreyWhitelistsNonEmpty(t *testing.T) {
	for _, p := range AllProfiles() {
		if len(p.PreyWhitelist) == 0 {
			t.Errorf("profile %s has empty PreyWhitelist", p.Name)
		}
	}
}

func TestSanctuaryRoomsAssigned(t *testing.T) {
	cases := []struct {
		mobId  int
		anchor int
		name   string
	}{
		{371, 4123, "Vella → Stillwater Temple"},
		{243, 468, "Halix → Thornwall Temple"},
		{366, 4197, "Kessa → Forager's Camp"},
	}
	for _, c := range cases {
		if got := ProfileFor(c.mobId).SanctuaryRoom; got != c.anchor {
			t.Errorf("%s: SanctuaryRoom = %d, want %d", c.name, got, c.anchor)
		}
	}
}
