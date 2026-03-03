package devtools

import (
	"encoding/json"
	"fmt"

	"github.com/GoMudEngine/GoMud/internal/gametime"
)

// APIRequest is the JSON envelope accepted by HandleJSON.
type APIRequest struct {
	Action string         `json:"action"`
	Params map[string]any `json:"params"`
}

// APIResponse is the JSON envelope returned by HandleJSON.
type APIResponse struct {
	Success bool           `json:"success"`
	Action  string         `json:"action"`
	Result  map[string]any `json:"result,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// HandleJSON parses a JSON APIRequest, dispatches to the appropriate devtool
// function, and always returns a valid JSON-encoded APIResponse string.
func HandleJSON(input string) string {

	errResp := func(action, msg string) string {
		resp := APIResponse{Success: false, Action: action, Error: msg}
		b, err := json.Marshal(resp)
		if err != nil {
			return fmt.Sprintf(`{"success":false,"action":%q,"error":%q}`, action, msg)
		}
		return string(b)
	}

	var req APIRequest
	if err := json.Unmarshal([]byte(input), &req); err != nil {
		return errResp("", fmt.Sprintf("invalid JSON: %s", err.Error()))
	}

	switch req.Action {

	case "check":
		zone, _ := req.Params["zone"].(string) // comma-ok not needed: empty string caught below
		if zone == "" {
			return errResp(req.Action, "params.zone is required")
		}
		report, issueCount, err := CheckZoneConsistency(zone)
		if err != nil {
			return errResp(req.Action, err.Error())
		}
		b, err := json.Marshal(APIResponse{
			Success: true,
			Action:  req.Action,
			Result: map[string]any{
				"issues": issueCount,
				"report": report,
			},
		})
		if err != nil {
			return errResp(req.Action, fmt.Sprintf("marshal error: %s", err.Error()))
		}
		return string(b)

	case "makezone":
		name, _ := req.Params["name"].(string) // comma-ok not needed: empty string caught below
		if name == "" {
			return errResp(req.Action, "params.name is required")
		}
		width, height := jsonInt(req.Params["width"]), jsonInt(req.Params["height"])
		if width < 1 || height < 1 {
			return errResp(req.Action, "params.width and params.height must be >= 1")
		}
		firstId, lastId, err := GenerateGrid(name, width, height)
		if err != nil {
			return errResp(req.Action, err.Error())
		}
		b, err := json.Marshal(APIResponse{
			Success: true,
			Action:  req.Action,
			Result: map[string]any{
				"rooms_created": width * height,
				"first_room_id": firstId,
				"last_room_id":  lastId,
			},
		})
		if err != nil {
			return errResp(req.Action, fmt.Sprintf("marshal error: %s", err.Error()))
		}
		return string(b)

	case "linkzones":
		zoneA, _ := req.Params["zone_a"].(string)   // comma-ok not needed: empty string caught below
		zoneB, _ := req.Params["zone_b"].(string)    // comma-ok not needed: empty string caught below
		direction, _ := req.Params["direction"].(string) // comma-ok not needed: empty string caught below
		roomA := jsonInt(req.Params["room_a"])
		roomB := jsonInt(req.Params["room_b"])

		if zoneA == "" || zoneB == "" || direction == "" || roomA == 0 || roomB == 0 {
			return errResp(req.Action, "params.zone_a, zone_b, direction, room_a, room_b are all required")
		}

		if err := LinkRooms(zoneA, roomA, direction, zoneB, roomB); err != nil {
			return errResp(req.Action, err.Error())
		}
		b, err := json.Marshal(APIResponse{
			Success: true,
			Action:  req.Action,
			Result:  map[string]any{"linked": true},
		})
		if err != nil {
			return errResp(req.Action, fmt.Sprintf("marshal error: %s", err.Error()))
		}
		return string(b)

	case "pressure":
		// Stage 17.2: Return current moon phases and derived stat modifiers.
		swift, wander, eye := gametime.GetAllPhases()
		b, err := json.Marshal(APIResponse{
			Success: true,
			Action:  req.Action,
			Result: map[string]any{
				"swiftmoon_phase": swift,  // DEX/STR modifier source
				"wanderer_phase":  wander, // VIT/WIL modifier source
				"eye_phase":       eye,    // PER/CHA modifier source; mutation rate = 0.5 + eye
				"mutation_rate_multiplier": 0.5 + eye,
			},
		})
		if err != nil {
			return errResp(req.Action, fmt.Sprintf("marshal error: %s", err.Error()))
		}
		return string(b)

	default:
		return errResp(req.Action, fmt.Sprintf("unknown action: %q", req.Action))
	}
}

// jsonInt converts a JSON number (float64 from json.Unmarshal) to int.
func jsonInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	}
	return 0
}
