package users

type OnlineInfo struct {
	Username      string
	CharacterName string
	Level         int
	Alignment     string
	Profession    string
	OnlineTime    int64
	OnlineTimeStr string
	IsAFK         bool
	IsAI          bool
	Role          string
}
