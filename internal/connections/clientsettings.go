package connections

const (
	// DefaultScreenWidth is the default width of the screen
	DefaultScreenWidth = 80
	// DefaultScreenHeight is the default height of the screen
	DefaultScreenHeight = 24
)

type ClientSettings struct {
	Display           DisplaySettings
	MSPEnabled        bool // Do they accept sound in their client?
	SendTelnetGoAhead bool // Defaults false, should we send a IAC GA after prompts?
	AsciiMode         bool // Convert UTF-8 decorative chars to ASCII equivalents?
	IsWebConnection   bool // True for websocket (web client) connections; read by splash delivery.
}

func (c ClientSettings) IsMsp() bool {
	return c.MSPEnabled
}

type DisplaySettings struct {
	ScreenWidth  uint32
	ScreenHeight uint32
}

func (c DisplaySettings) GetScreenWidth() int {
	if c.ScreenWidth == 0 {
		return DefaultScreenWidth
	}

	return int(c.ScreenWidth)
}

func (c DisplaySettings) GetScreenHeight() int {
	if c.ScreenHeight == 0 {
		return DefaultScreenHeight
	}
	return int(c.ScreenHeight)
}
