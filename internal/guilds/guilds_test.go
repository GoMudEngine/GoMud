package guilds

import "testing"

func TestValidGuildTag(t *testing.T) {
	good := []string{"QC", "ABCD", "a1", "X9y"}
	for _, s := range good {
		if err := validGuildTag(s); err != nil {
			t.Errorf("%q should be valid: %v", s, err)
		}
	}
	bad := []string{"", "A", "ABCDE", "Q C", "Q-C", "!!"}
	for _, s := range bad {
		if err := validGuildTag(s); err == nil {
			t.Errorf("%q should be invalid", s)
		}
	}
}

func TestValidGuildName(t *testing.T) {
	if err := validGuildName("Questing Cajuns"); err != nil {
		t.Errorf("normal name should validate: %v", err)
	}
	if err := validGuildName("ab"); err == nil {
		t.Error("2-char name should be too short")
	}
	if err := validGuildName(" Trim Me "); err == nil {
		t.Error("leading/trailing space should be rejected")
	}
}

func TestPermissions(t *testing.T) {
	g := &Guild{Tag: "QC", LeaderUserId: 1, Members: []GuildMember{
		{UserId: 1, Rank: RankLeader}, {UserId: 2, Rank: RankOfficer}, {UserId: 3, Rank: RankMember},
	}}
	if !g.IsLeader(1) || g.IsLeader(2) {
		t.Error("leader detection wrong")
	}
	if !g.CanManage(1) || !g.CanManage(2) || g.CanManage(3) {
		t.Error("CanManage should be officer+ only")
	}
	if !g.CanKick(1, 3) || !g.CanKick(2, 3) {
		t.Error("leader/officer should kick a member")
	}
	if g.CanKick(2, 2) || g.CanKick(2, 1) || g.CanKick(3, 2) {
		t.Error("cannot kick peer/superior; member cannot kick")
	}
}
