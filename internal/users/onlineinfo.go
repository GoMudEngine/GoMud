package users

type OnlineInfo struct {
	Username      string
	CharacterName string
	Level         int
	Profession    string
	OnlineTime    int64
	OnlineTimeStr string
	IsAFK         bool
	IsAI          bool
	Role          string
}
