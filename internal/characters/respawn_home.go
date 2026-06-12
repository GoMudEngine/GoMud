package characters

// HomeLocations maps a player's home setting key to the
// destination room ID for respawn / set-home teleport.
// Engine default (key "default") = room 0 (Sanctum Basin entrance).
var HomeLocations = map[string]int{
	"default":    0,
	"thornwall":  468,
	"stillwater": 4123,
	"coulee":     5209,
}

// HomeLocationNames is the display-string companion to HomeLocations.
var HomeLocationNames = map[string]string{
	"default":    "Sanctum Basin",
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
