package grapplemessaging

import (
	"math/rand"
	"strings"
)

// RenderTemplate substitutes {controllerName} and {controlledName}
// in a template string and returns the rendered result. Caller is
// responsible for ANSI-wrapping or other formatting.
func RenderTemplate(template, controllerName, controlledName string) string {
	out := strings.ReplaceAll(template, "{controllerName}", controllerName)
	out = strings.ReplaceAll(out, "{controlledName}", controlledName)
	return out
}

// PickTemplate selects a template from the list, preferring ones
// not yet used in this grapple's cooldown map. The cooldown key is
// computed per template as `<keyPrefix>:<template>` so each unique
// rendered template can be tracked independently across rounds.
//
// When all templates in the list have been used at least once in
// this grapple, the cooldown map is reset and selection starts over
// — forcing variety within reasonable bounds without blocking
// templates forever.
//
// Empty `pool` returns a benign fallback string so callers can
// always send something (a missing template should not crash a
// round).
func PickTemplate(pool []string, cooldowns map[string]bool, keyPrefix string) string {
	if len(pool) == 0 {
		return "(grapple messaging missing template)"
	}

	// Filter out templates whose cooldown key is marked.
	available := make([]string, 0, len(pool))
	for _, tmpl := range pool {
		ck := keyPrefix + ":" + tmpl
		if !cooldowns[ck] {
			available = append(available, tmpl)
		}
	}

	// If all are cooled-down, reset and start over.
	if len(available) == 0 {
		for _, tmpl := range pool {
			delete(cooldowns, keyPrefix+":"+tmpl)
		}
		available = pool
	}

	pick := available[rand.Intn(len(available))]
	cooldowns[keyPrefix+":"+pick] = true
	return pick
}
