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

// getFloatParam reads a float parameter from the params map.
// Handles both float64 (canonical YAML) and int (if an integer literal
// was provided). Returns the defaultVal if the key is missing or an
// unsupported type.
func getFloatParam(params map[string]any, key string, defaultVal float64) float64 {
	switch v := params[key].(type) {
	case float64:
		return v
	case int:
		return float64(v)
	}
	return defaultVal
}

// getStringListParam reads a []string parameter from the params map.
// YAML unmarshals lists as []any, so we type-assert each element to string.
// Returns nil if missing or wrong type.
func getStringListParam(params map[string]any, key string) []string {
	v, ok := params[key].([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(v))
	for _, el := range v {
		if s, ok := el.(string); ok {
			out = append(out, s)
		}
	}
	return out
}
