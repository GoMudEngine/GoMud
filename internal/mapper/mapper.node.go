package mapper

// represents a single room
type mapNode struct {
	RoomId      int
	Symbol      rune
	Legend      string // The same that shows in the legend for this symbol
	Exits       map[string]nodeExit
	SecretExits map[string]struct{} // Just a flag for whether an exit key is secret
	Pos         positionDelta       // Its x/y/z position relative to the root node
	Plane       int                 // coordinate-space id (0 = overworld); copied from Room.Plane
}

type nodeExit struct {
	RoomId         int    // where it leads to
	Secret         bool   // is it secret?
	LockDifficulty int    // If > 0, the lock difficulty.
	LockId         string // What's the lock id?
	OneWay         bool   // intentional one-way spatial exit (skips reciprocity check)
	Gate           bool   // exit is a threshold/gate (has an ExitMessage)
	Direction      positionDelta
}
