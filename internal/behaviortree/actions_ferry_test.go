package behaviortree

import "testing"

func TestBoardFerry_RegisteredInActionRegistry(t *testing.T) {
	if _, ok := actionRegistry["board_ferry"]; !ok {
		t.Fatal(`board_ferry not in actionRegistry`)
	}
}
