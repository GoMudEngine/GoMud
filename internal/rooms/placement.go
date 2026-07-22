package rooms

import "fmt"

// ValidatePlacement reports an error if placing a room at (plane,x,y,z) would
// overlap another room on a Euclidean plane. excludeRoomId is the room being
// created/moved (so it doesn't collide with itself). This is the build-time
// gate the web builder (1b) calls before persisting.
func ValidatePlacement(plane, x, y, z, excludeRoomId int) error {
	nonEuclid := GetPlaneRegistry().IsNonEuclidean(plane)
	return validatePlacement(plane, x, y, z, excludeRoomId, nonEuclid, occupantAt)
}

// validatePlacement is the testable core (dependency-injected occupancy lookup).
func validatePlacement(plane, x, y, z, excludeRoomId int, nonEuclidean bool,
	occupant func(p, x, y, z int) (int, bool)) error {
	if nonEuclidean {
		return nil
	}
	if id, ok := occupant(plane, x, y, z); ok && id != excludeRoomId {
		return fmt.Errorf("cell (plane %d, %d,%d,%d) is already occupied by room %d", plane, x, y, z, id)
	}
	return nil
}

// occupantAt scans loaded rooms for one at the given authored coordinate.
func occupantAt(plane, x, y, z int) (int, bool) {
	for _, roomId := range GetAllRoomIds() {
		r := LoadRoom(roomId)
		if r != nil && r.Plane == plane && r.X == x && r.Y == y && r.Z == z {
			return roomId, true
		}
	}
	return 0, false
}
