package rooms

// instancePlaneBase keeps instance coordinate planes well clear of authored
// world planes (small ints assigned by the 0.15.0 migration). Each live
// instance gets its own plane so its ephemeral rooms never collide with the
// template or with sibling instances.
const instancePlaneBase = 1_000_000

var instancePlaneSeq int // advanced under the instance-creation path (single game loop)

// nextInstancePlane returns a fresh, monotonically increasing plane id for a new
// instance, always >= instancePlaneBase.
func nextInstancePlane() int {
	instancePlaneSeq++
	return instancePlaneBase + instancePlaneSeq
}
