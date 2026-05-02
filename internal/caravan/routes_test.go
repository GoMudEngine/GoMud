package caravan

import "testing"

func TestRoute_OutboundIntegrity(t *testing.T) {
	if OutboundRoute.DepartFromRoomId == 0 {
		t.Error("OutboundRoute.DepartFromRoomId == 0")
	}
	if OutboundRoute.ArriveAtRoomId == 0 {
		t.Error("OutboundRoute.ArriveAtRoomId == 0")
	}
	if OutboundRoute.DepartFromRoomId == OutboundRoute.ArriveAtRoomId {
		t.Errorf("OutboundRoute depart == arrive (%d)", OutboundRoute.DepartFromRoomId)
	}
	if len(OutboundRoute.VendorStopIds) == 0 {
		t.Error("OutboundRoute has no vendor stops")
	}
	seen := map[int]bool{}
	for _, stop := range OutboundRoute.VendorStopIds {
		if stop == 0 {
			t.Error("OutboundRoute has zero vendor stop")
		}
		if seen[stop] {
			t.Errorf("OutboundRoute has duplicate stop %d", stop)
		}
		seen[stop] = true
	}
}

func TestRoute_InboundIntegrity(t *testing.T) {
	if InboundRoute.DepartFromRoomId == 0 {
		t.Error("InboundRoute.DepartFromRoomId == 0")
	}
	if InboundRoute.ArriveAtRoomId == 0 {
		t.Error("InboundRoute.ArriveAtRoomId == 0")
	}
	if InboundRoute.DepartFromRoomId == InboundRoute.ArriveAtRoomId {
		t.Errorf("InboundRoute depart == arrive (%d)", InboundRoute.DepartFromRoomId)
	}
	if len(InboundRoute.VendorStopIds) == 0 {
		t.Error("InboundRoute has no vendor stops")
	}
}

func TestRoute_OutboundAndInboundOpposite(t *testing.T) {
	if OutboundRoute.ArriveAtRoomId != InboundRoute.DepartFromRoomId {
		t.Errorf("Outbound arrives at %d but Inbound departs from %d (should match)",
			OutboundRoute.ArriveAtRoomId, InboundRoute.DepartFromRoomId)
	}
	if InboundRoute.ArriveAtRoomId != OutboundRoute.DepartFromRoomId {
		t.Errorf("Inbound arrives at %d but Outbound departs from %d (should match)",
			InboundRoute.ArriveAtRoomId, OutboundRoute.DepartFromRoomId)
	}
}

func TestThornwallVendorRoomsExcludesWhisper(t *testing.T) {
	for _, r := range thornwallVendorRooms {
		if r == 507 {
			t.Fatalf("thornwallVendorRooms still includes 507 (Whisper); she's off-route in the phantom's zone and should not be on the caravan rotation")
		}
	}
}
