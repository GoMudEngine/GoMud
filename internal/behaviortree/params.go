package behaviortree

// getIntParam reads an integer parameter from the params map.
// Handles both int and float64 YAML values. Returns 0 if missing.
func getIntParam(params map[string]any, key string) int {
	switch v := params[key].(type) {
	case int:
		return v
	case float64:
		return int(v)
	}
	return 0
}

// getStringParam reads a string parameter from the params map.
// Returns empty string if missing or wrong type.
func getStringParam(params map[string]any, key string) string {
	v, _ := params[key].(string)
	return v
}

// getBoolParam reads a boolean parameter from the params map. Accepts
// proper YAML bool (`hostile: true`) as the canonical form and tolerates
// the legacy string form (`hostile: "true"`) for backward compatibility.
// Returns false if missing or wrong type.
func getBoolParam(params map[string]any, key string) bool {
	switch v := params[key].(type) {
	case bool:
		return v
	case string:
		return v == "true"
	}
	return false
}
