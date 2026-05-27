package goals

// ValidateParams checks g.Params against the registered type's schema.
// Returns *ErrBadParams on failure (use errors.As to inspect).
// nil schema → no validation (matches 4.1 freeform behavior).
//
// Chunk 4.3.
func ValidateParams(g *Goal, schema []ParamSchema) error {
	if len(schema) == 0 {
		return nil
	}
	for _, ps := range schema {
		raw, present := g.Params[ps.Key]
		if !present {
			if ps.Required {
				return &ErrBadParams{Key: ps.Key, ExpectedType: ps.GoType, Reason: "missing required key"}
			}
			continue
		}
		if !matchesGoType(raw, ps.GoType) {
			return &ErrBadParams{Key: ps.Key, ExpectedType: ps.GoType, GotType: goTypeName(raw)}
		}
	}
	return nil
}

// matchesGoType reports whether raw satisfies the declared GoType.
// Permissive on numeric widening (int64 → int, int → float64) since
// YAML round-trips integers as int64 and floats can absorb ints.
func matchesGoType(raw any, goType string) bool {
	switch goType {
	case "int":
		switch raw.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		}
		return false
	case "string":
		_, ok := raw.(string)
		return ok
	case "[]string":
		switch v := raw.(type) {
		case []string:
			return true
		case []any:
			for _, e := range v {
				if _, ok := e.(string); !ok {
					return false
				}
			}
			return true
		}
		return false
	case "float64":
		switch raw.(type) {
		case float32, float64, int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
			return true
		}
		return false
	case "bool":
		_, ok := raw.(bool)
		return ok
	}
	return false // unknown GoType in schema is treated as no-match
}

// goTypeName returns a printable Go type name for error messages.
func goTypeName(raw any) string {
	switch raw.(type) {
	case int, int8, int16, int32, int64:
		return "int"
	case uint, uint8, uint16, uint32, uint64:
		return "uint"
	case float32, float64:
		return "float"
	case string:
		return "string"
	case bool:
		return "bool"
	case []any:
		return "[]any"
	case []string:
		return "[]string"
	}
	return "unknown"
}
