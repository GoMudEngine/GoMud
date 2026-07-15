package usercommands

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/guilds"
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
