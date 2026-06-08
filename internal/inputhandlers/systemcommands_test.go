package inputhandlers

import (
	"testing"

	"github.com/GoMudEngine/GoMud/internal/connections"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// TestMain initializes shared infrastructure (logging) required by the
// inputhandlers package before any tests in this package run.
func TestMain(m *testing.M) {
	mudlog.SetupLogger(nil, "", "", false)
	m.Run()
}

func TestTrySystemCommand_AuthorizationChecks(t *testing.T) {
	const (
		adminConnId    connections.ConnectionId = 1001
		nonAdminConnId connections.ConnectionId = 1002
		noUserConnId   connections.ConnectionId = 1003
	)

	setup := func() func() {
		adminUser := users.NewTestUser(1, "admin", "Admin", uint64(adminConnId))
		adminUser.Role = users.RoleAdmin

		nonAdminUser := users.NewTestUser(2, "player", "Player", uint64(nonAdminConnId))
		// Role is already RoleUser (default from NewTestUser); no user for noUserConnId

		return users.SeedUsersForTest(map[int]*users.UserRecord{
			1: adminUser,
			2: nonAdminUser,
		})
	}

	tests := []struct {
		name        string
		cmd         string
		connId      connections.ConnectionId
		wantHandled bool
		description string
	}{
		{
			name:        "shutdown rejected for non-admin user",
			cmd:         "/shutdown",
			connId:      nonAdminConnId,
			wantHandled: false,
			description: "/shutdown must be rejected when the requesting user does not have the admin role",
		},
		{
			name:        "reload rejected for non-admin user",
			cmd:         "/reload",
			connId:      nonAdminConnId,
			wantHandled: false,
			description: "/reload must be rejected when the requesting user does not have the admin role",
		},
		{
			name:        "shutdown rejected for unknown connection (no user)",
			cmd:         "/shutdown",
			connId:      noUserConnId,
			wantHandled: false,
			description: "/shutdown must be rejected when no user record is associated with the connection",
		},
		{
			name:        "reload rejected for unknown connection (no user)",
			cmd:         "/reload",
			connId:      noUserConnId,
			wantHandled: false,
			description: "/reload must be rejected when no user record is associated with the connection",
		},
		{
			name:        "shutdown accepted for admin user",
			cmd:         "/shutdown 0",
			connId:      adminConnId,
			wantHandled: true,
			description: "/shutdown must be accepted (return true) when the requesting user has the admin role",
		},
		{
			name:        "reload accepted for admin user",
			cmd:         "/reload",
			connId:      adminConnId,
			wantHandled: true,
			description: "/reload must be accepted (return true) when the requesting user has the admin role",
		},
		{
			name:        "quit allowed for non-admin user",
			cmd:         "/quit",
			connId:      nonAdminConnId,
			wantHandled: true,
			description: "/quit must be allowed for all users regardless of role",
		},
		{
			name:        "unknown system command returns false regardless of role",
			cmd:         "/notacommand",
			connId:      adminConnId,
			wantHandled: false,
			description: "an unrecognised system command must not be handled",
		},
		{
			name:        "non-slash prefix returns false",
			cmd:         "shutdown",
			connId:      adminConnId,
			wantHandled: false,
			description: "input without the / prefix must not be treated as a system command",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cleanup := setup()
			defer cleanup()
			got := trySystemCommand(tt.cmd, tt.connId)
			if got != tt.wantHandled {
				t.Errorf("%s: trySystemCommand(%q, connId=%d) = %v, want %v",
					tt.description, tt.cmd, tt.connId, got, tt.wantHandled)
			}
		})
	}
}
