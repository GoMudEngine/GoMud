// Package seeders contains chunk-4.5 reactive goal-generation rules.
// Each rule subscribes to one or more event types (or is invoked
// directly from another package for architectural exceptions) and
// produces effects via the normal substrate APIs: goals.Add (goal
// seeders), mob.Character.SetMiscData (counter writers), or
// opinions.Bump (opinion shifters).
//
// Per-rule files live alongside this one. Each rule's init() calls
// Register. main.go imports this package + wires events.AddListener
// for each event type seeders care about.
//
// See docs/superpowers/specs/2026-05-27-mob-aliveness-4.5-reactive-goal-generation-design.md
package seeders

import (
	"fmt"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/events"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// RuleFn is invoked once per event the rule is registered for. The
// rule inspects the event payload, decides whether to act, and applies
// effects via substrate APIs. Rules must be defensive against the
// event payload not matching expectations (return early on type
// assertion failures, missing fields, etc.).
type RuleFn func(event events.Event)

// registration ties a rule's name (for logging) to its function.
type registration struct {
	name string
	fn   RuleFn
}

var (
	registryMu sync.RWMutex
	registry   = map[string][]registration{} // event type name → registered rules
)

// Register subscribes a rule to one or more event type names. Called
// from each per-rule file's init() function. ruleName is used for
// panic-recovery log lines + future admin tooling.
//
// types parameter accepts the values produced by event.Type() — the
// string identifier the events package uses for each concrete type
// (e.g., "MobDeath", "Communication"). Register the rule for each
// type it cares about.
func Register(ruleName string, fn RuleFn, types ...string) {
	registryMu.Lock()
	defer registryMu.Unlock()
	for _, t := range types {
		registry[t] = append(registry[t], registration{name: ruleName, fn: fn})
	}
}

// Dispatch is the package-level event listener wired by main.go for
// every event type seeders care about. Looks up rules for the event's
// type, invokes each under panic recovery.
//
// Returns events.Continue always — seeders observe events; they don't
// suppress them.
func Dispatch(event events.Event) events.ListenerReturn {
	typeName := event.Type()

	registryMu.RLock()
	rules := registry[typeName]
	registryMu.RUnlock()

	for _, reg := range rules {
		invokeRuleSafely(reg.name, reg.fn, event)
	}
	return events.Continue
}

// invokeRuleSafely wraps a rule call in panic recovery. Mirrors the
// 4.2 invokeContextScore / 4.3 invokeDedupKey / 4.4 invokePlannerSafely
// patterns. A panic logs a warn line with rule name + event type and
// returns; other rules' invocation continues.
func invokeRuleSafely(ruleName string, fn RuleFn, event events.Event) {
	defer func() {
		if r := recover(); r != nil {
			mudlog.Warn("seeders.rule panic",
				"rule", ruleName,
				"event_type", event.Type(),
				"panic", fmt.Sprintf("%v", r))
		}
	}()
	fn(event)
}

// resetRegistryForTest wipes the registry. Test-only seam — package-
// internal so tests can isolate rule registration.
func resetRegistryForTest() {
	registryMu.Lock()
	registry = map[string][]registration{}
	registryMu.Unlock()
}
