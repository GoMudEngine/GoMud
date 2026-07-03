package behaviortree

import (
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/ferry"
	"github.com/GoMudEngine/GoMud/internal/mobs"
	"github.com/GoMudEngine/GoMud/internal/users"
)

// actBoardFerry handles "ask <agent> passage <destination>". Params:
//
//	routes:            # destination keyword → routeid
//	  plymouth: stillwater_np_packet
//	  confluence: stillwater_confluence_barge
//
// Resolves the destination word from the ask text, then delegates the
// entire boarding flow (schedule check, fare, move, messaging) to
// ferry.Board. Returns Failure when params are unusable so other tree
// branches can try; returns Success (with mob dialogue) on every handled
// outcome. The keyword_match condition gating this action in the tree
// establishes boarding intent, so an unrecognized destination lists the
// options rather than falling through.
func actBoardFerry(params map[string]any, ctx *EvalContext) Result {
	mob := mobs.GetInstance(ctx.InstanceId)
	if mob == nil {
		return Failure
	}
	user := users.GetByUserId(ctx.Event.UserId)
	if user == nil {
		return Failure
	}

	routesMap := map[string]string{}
	if raw, ok := params["routes"]; ok {
		switch m := raw.(type) {
		case map[string]any:
			for k, v := range m {
				if s, ok := v.(string); ok {
					routesMap[strings.ToLower(k)] = s
				}
			}
		case map[any]any:
			for k, v := range m {
				ks, kok := k.(string)
				vs, vok := v.(string)
				if kok && vok {
					routesMap[strings.ToLower(ks)] = vs
				}
			}
		case map[string]string:
			for k, v := range m {
				routesMap[strings.ToLower(k)] = v
			}
		}
	}
	if len(routesMap) == 0 {
		return Failure
	}

	// Sorted keyword order makes the pick deterministic when the ask text
	// contains more than one destination ("plymouth or confluence?").
	dests := make([]string, 0, len(routesMap))
	for k := range routesMap {
		dests = append(dests, k)
	}
	sort.Strings(dests)

	text := strings.ToLower(ctx.Event.Text)
	routeId := ``
	for _, keyword := range dests {
		if strings.Contains(text, keyword) {
			routeId = routesMap[keyword]
			break
		}
	}

	if routeId == `` {
		mob.Command(`say Passage where? I book for ` + strings.Join(dests, ` and `) + `.`)
		return Success
	}

	if ferry.Board(user, mob, ctx.RoomId, routeId) == ferry.BoardNotAtPort {
		return Failure // fall through to other tree branches
	}
	return Success
}
