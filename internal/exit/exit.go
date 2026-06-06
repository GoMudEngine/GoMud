package exit

import "github.com/GoMudEngine/GoMud/internal/gamelock"

// There is a magic portal of Chuckles, magic portal of Henry here!
// There is a magical hole in the east wall here!
type TemporaryRoomExit struct {
	RoomId       int    // Where does it lead to?
	Title        string // Does this exit have a special title?
	UserId       int    // Who created it?
	SpawnedRound uint64 `yaml:"-"` // When the temp exit was created
	Expires      string // When will it be auto-cleaned up?
}

type RoomExit struct {
	RoomId       int
	Secret       bool          `yaml:"secret,omitempty"`
	MapDirection string        `yaml:"mapdirection,omitempty"` // Optionaly indicate the direction of this exit for mapping purposes
	ExitMessage  string        `yaml:"exitmessage,omitempty"`  // If set, this message is sent to the user, followed by a delay, before they actually go through the exit.
	Lock         gamelock.Lock `yaml:"lock,omitempty"`         // 0 - no lock. greater than zero = difficulty to unlock.
	OneWay       bool          `yaml:"oneway,omitempty"`       // true = intentional one-way spatial exit; skips the reciprocity consistency check.
}

func (re RoomExit) HasLock() bool {
	return re.Lock.Difficulty > 0
}
