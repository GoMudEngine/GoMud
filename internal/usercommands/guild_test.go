package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/guilds"
	"github.com/GoMudEngine/GoMud/internal/items"
)

func TestGuildChatRecipients(t *testing.T) {
	g := &guilds.Guild{Tag: "QC", Members: []guilds.GuildMember{
		{UserId: 1}, {UserId: 2}, {UserId: 3},
	}}
	got := guildChatRecipients(g, 2)
	if len(got) != 2 || got[0] != 1 || got[1] != 3 {
		t.Errorf("recipients = %v, want [1 3] (members except sender)", got)
	}
	if len(guildChatRecipients(&guilds.Guild{}, 1)) != 0 {
		t.Error("empty guild -> no recipients")
	}
}

func TestParseGuildRank(t *testing.T) {
	cases := map[string]guilds.GuildRank{
		"member": guilds.RankMember, "Officer": guilds.RankOfficer, "LEADER": guilds.RankLeader,
	}
	for in, want := range cases {
		if got, ok := parseGuildRank(in); !ok || got != want {
			t.Errorf("parseGuildRank(%q) = %v,%v want %v", in, got, ok, want)
		}
	}
	if _, ok := parseGuildRank("captain"); ok {
		t.Error("unknown rank should not parse")
	}
}

func TestFindVaultItem(t *testing.T) {
	defer items.SeedItemsForTest(map[int]*items.ItemSpec{
		101: {ItemId: 101, Name: "iron sword"},
		102: {ItemId: 102, Name: "healing potion"},
	})()
	vault := []items.Item{items.New(101), items.New(102)}
	if idx, ok := findVaultItem(vault, "healing potion"); !ok || idx != 1 {
		t.Errorf("find = %d,%v, want 1,true", idx, ok)
	}
	if _, ok := findVaultItem(vault, "nonexistent"); ok {
		t.Error("miss should return ok=false")
	}
	if _, ok := findVaultItem(nil, "x"); ok {
		t.Error("empty vault -> not found")
	}
}
