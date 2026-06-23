package characters

import "testing"

// The sethome registry: every key must resolve in both maps, and the
// newbie-hub "coulee" entry (Pothole Coulee, The Mending Hut) must
// point at room 5209 — Sala's hut, where knocked-out learners wake.

func TestHomeLocations_KeysHaveNames(t *testing.T) {
	for key := range HomeLocations {
		if HomeLocationNames[key] == "" {
			t.Errorf("HomeLocations key %q has no display name", key)
		}
	}
	for key := range HomeLocationNames {
		if _, ok := HomeLocations[key]; !ok {
			t.Errorf("HomeLocationNames key %q has no room mapping", key)
		}
	}
}

func TestHomeLocations_CouleeResolvesToMendingHut(t *testing.T) {
	if got := HomeLocations["coulee"]; got != 5209 {
		t.Errorf("coulee home = room %d, want 5209 (The Mending Hut)", got)
	}
}

func TestResolveRespawnRoom_CouleeSetting(t *testing.T) {
	c := New()
	c.SetSetting("home", "coulee")
	if got := c.ResolveRespawnRoom(); got != 5209 {
		t.Errorf("ResolveRespawnRoom() = %d, want 5209", got)
	}
}

func TestResolveRespawnRoom_UnknownFallsBackToDefault(t *testing.T) {
	c := New()
	c.SetSetting("home", "nowhere_real")
	if got := c.ResolveRespawnRoom(); got != HomeLocations["default"] {
		t.Errorf("ResolveRespawnRoom() = %d, want default %d", got, HomeLocations["default"])
	}
}

func TestResolveRespawnRoom_NewPlymouthHome(t *testing.T) {
	c := New()
	c.SetSetting("home", "newplymouth")
	if got := c.ResolveRespawnRoom(); got != 5901 {
		t.Errorf("newplymouth home respawn = %d, want 5901 (the Grand Temple sanctuary)", got)
	}
}
