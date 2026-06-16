package characters

// HomeLocations maps a player's home setting key to the
// destination room ID for respawn / set-home teleport.
// Default (key "default") = room 5209, The Mending Hut in Pothole Coulee.
// (Re-pointed from the old Sanctum Basin room 0 during the newbie-area
// rework, Spoke A / chunk 2, 2026-06-13 — pulled forward from the chunk-10
// cutover so the Spoke A death-and-wake lesson is testable. The Sanctum
// Basin newbie area is being retired by the rework.)
var HomeLocations = map[string]int{
	"default":    5209,
	"thornwall":  468,
	"stillwater": 4123,
	"coulee":     5209,
}

// HomeLocationNames is the display-string companion to HomeLocations.
var HomeLocationNames = map[string]string{
	"default":    "Pothole Coulee (The Mending Hut)",
	"thornwall":  "Thornwall City (Temple Interior)",
	"stillwater": "Stillwater (Temple of Stillwater)",
	"coulee":     "Pothole Coulee (The Mending Hut)",
}

// ResolveRespawnRoom returns the destination room ID for a
// player respawning from death. Reads the player's "home" setting
// and falls back to "default" if unset/invalid.
func (c *Character) ResolveRespawnRoom() int {
	homeSetting := c.GetSetting("home")
	if roomId, ok := HomeLocations[homeSetting]; ok {
		return roomId
	}
	return HomeLocations["default"]
}
