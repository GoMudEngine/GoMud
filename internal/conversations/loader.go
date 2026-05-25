package conversations

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/relationships"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// conversationWorldValidator is injected at startup to perform world-
// aware validation (mob existence + relationship edge presence) without
// creating an import cycle. Set via SetConversationWorldValidator before
// calling Load; if either function is nil, the world-aware pass is
// silently skipped.
var conversationWorldValidator struct {
	mobExists           func(id int) bool
	relationshipBetween func(a, b int) bool
}

// SetConversationWorldValidator wires in the mob-existence and
// relationship-edge checks. Caller (typically main.go) supplies closures
// that reach into the mobs and relationships packages. Mirrors the chunk
// 3.2 / 3.4 DI pattern.
func SetConversationWorldValidator(mobExists func(int) bool, relationshipBetween func(int, int) bool) {
	conversationWorldValidator.mobExists = mobExists
	conversationWorldValidator.relationshipBetween = relationshipBetween
}

// Load walks _datafiles/world/dogmud/conversations/{types,pairs}/*.yaml,
// parses each file, validates, and registers it in the package-level
// registries. Panics on duplicate ids or validation failures. If the
// conversations directory is missing, logs and returns (optional content).
func Load() {
	start := time.Now()

	dataRoot := configs.GetFilePathsConfig().DataFiles.String() + `/conversations`

	if _, err := os.Stat(dataRoot); os.IsNotExist(err) {
		mudlog.Info("conversations.Load()", "loadedPools", 0, "loadedPairs", 0,
			"note", "conversations directory does not exist — skipping",
			"Time Taken", time.Since(start))
		return
	}

	tmpPools := map[relationships.Type]*Pool{}
	tmpPairs := map[pairKey]*PairOverride{}

	// types/*.yaml
	typesDir := filepath.Join(dataRoot, "types")
	if _, err := os.Stat(typesDir); err == nil {
		if walkErr := walkYAMLs(typesDir, func(path string, data []byte) error {
			var p Pool
			if err := yaml.Unmarshal(data, &p); err != nil {
				return fmt.Errorf("conversation pool: unmarshal %s: %w", path, err)
			}
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			expected := util.ConvertForFilename(p.Id)
			if base != expected {
				return fmt.Errorf("conversation pool: filename mismatch: %q base %q != expected %q (id %q)",
					path, base, expected, p.Id)
			}
			if err := validatePoolStandalone(&p); err != nil {
				return fmt.Errorf("conversation pool %q (%s): %w", p.Id, path, err)
			}
			emitPoolWarnings(&p)
			t := relationships.Type(p.Id)
			if _, dup := tmpPools[t]; dup {
				return fmt.Errorf("conversation pool: duplicate id %q in %s", p.Id, path)
			}
			pp := p
			tmpPools[t] = &pp
			return nil
		}); walkErr != nil {
			panic(fmt.Sprintf("conversations.Load() pools failed: %v", walkErr))
		}
	}

	// pairs/*.yaml
	pairsDir := filepath.Join(dataRoot, "pairs")
	if _, err := os.Stat(pairsDir); err == nil {
		if walkErr := walkYAMLs(pairsDir, func(path string, data []byte) error {
			var po PairOverride
			if err := yaml.Unmarshal(data, &po); err != nil {
				return fmt.Errorf("conversation pair: unmarshal %s: %w", path, err)
			}
			// Filename convention: <smaller_id>_<larger_id>.yaml
			base := strings.TrimSuffix(filepath.Base(path), filepath.Ext(path))
			lo, hi := po.MobA, po.MobB
			if lo > hi {
				lo, hi = hi, lo
			}
			expected := fmt.Sprintf("%d_%d", lo, hi)
			if base != expected {
				return fmt.Errorf("conversation pair: filename mismatch: %q base %q != expected %q (mobs %d/%d)",
					path, base, expected, po.MobA, po.MobB)
			}
			if err := validatePairOverrideStandalone(&po); err != nil {
				return fmt.Errorf("conversation pair %q (%s): %w", po.Id, path, err)
			}
			emitPairOverrideWarnings(&po)
			k := makePairKey(po.MobA, po.MobB)
			if _, dup := tmpPairs[k]; dup {
				return fmt.Errorf("conversation pair: duplicate pair %d/%d in %s", po.MobA, po.MobB, path)
			}
			pp := po
			tmpPairs[k] = &pp
			return nil
		}); walkErr != nil {
			panic(fmt.Sprintf("conversations.Load() pairs failed: %v", walkErr))
		}
	}

	// World-aware validation: mob existence + relationship edge presence
	// for pair overrides. Only runs when both injectors are wired.
	if conversationWorldValidator.mobExists != nil && conversationWorldValidator.relationshipBetween != nil {
		for _, po := range tmpPairs {
			if !conversationWorldValidator.mobExists(po.MobA) {
				panic(fmt.Sprintf("conversation pair %q: mob_a=%d not found in mobs registry", po.Id, po.MobA))
			}
			if !conversationWorldValidator.mobExists(po.MobB) {
				panic(fmt.Sprintf("conversation pair %q: mob_b=%d not found in mobs registry", po.Id, po.MobB))
			}
			if !conversationWorldValidator.relationshipBetween(po.MobA, po.MobB) {
				panic(fmt.Sprintf("conversation pair %q: no relationship edge between mobs %d and %d — pair override would never fire",
					po.Id, po.MobA, po.MobB))
			}
		}
	}

	poolsMu.Lock()
	pools = tmpPools
	poolsMu.Unlock()

	pairsMu.Lock()
	pairs = tmpPairs
	pairsMu.Unlock()

	mudlog.Info("conversations.Load()",
		"loadedPools", len(tmpPools),
		"loadedPairs", len(tmpPairs),
		"Time Taken", time.Since(start))
}

// walkYAMLs walks a directory and invokes fn for each .yaml file.
func walkYAMLs(dir string, fn func(path string, data []byte) error) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".yaml") {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return fmt.Errorf("read %s: %w", path, readErr)
		}
		return fn(path, data)
	})
}

// validatePoolStandalone runs internal-consistency checks on a pool
// without touching the mobs registry or filesystem. Suitable for unit
// tests.
func validatePoolStandalone(p *Pool) error {
	// id must be one of the six known relationship types.
	switch relationships.Type(p.Id) {
	case relationships.TypeFamily, relationships.TypeFriend, relationships.TypeRival,
		relationships.TypeLover, relationships.TypeEmployer, relationships.TypeEmployee:
		// ok
	default:
		return fmt.Errorf("pool id %q is not a known relationship type", p.Id)
	}
	for i, ex := range p.Exchanges {
		if err := validateExchange(ex); err != nil {
			return fmt.Errorf("exchange %d: %w", i, err)
		}
	}
	for subKey, sub := range p.Subtypes {
		for i, ex := range sub.Exchanges {
			if err := validateExchange(ex); err != nil {
				return fmt.Errorf("subtype %q exchange %d: %w", subKey, i, err)
			}
		}
	}
	return nil
}

// validatePairOverrideStandalone is the same for PairOverride.
func validatePairOverrideStandalone(po *PairOverride) error {
	if po.MobA == 0 || po.MobB == 0 {
		return errors.New("mob_a and mob_b must both be non-zero")
	}
	if po.MobA == po.MobB {
		return fmt.Errorf("self-pair (mob_a == mob_b == %d) not allowed", po.MobA)
	}
	for i, ex := range po.Exchanges {
		if err := validateExchange(ex); err != nil {
			return fmt.Errorf("exchange %d: %w", i, err)
		}
	}
	return nil
}

// validateExchange checks the line list of a single exchange.
func validateExchange(ex Exchange) error {
	if len(ex.Lines) == 0 {
		return errors.New("exchange has no lines")
	}
	for i, line := range ex.Lines {
		if line.Speaker != "A" && line.Speaker != "B" {
			return fmt.Errorf("line %d: speaker %q must be \"A\" or \"B\"", i, line.Speaker)
		}
		if strings.TrimSpace(line.Text) == "" {
			return fmt.Errorf("line %d: empty text", i)
		}
	}
	return nil
}

// emitPoolWarnings logs non-fatal author hints.
func emitPoolWarnings(p *Pool) {
	if len(p.Exchanges) == 0 && len(p.Subtypes) == 0 {
		mudlog.Warn("conversations.Load()",
			"poolId", p.Id, "warning", "pool has zero exchanges — relationship type will be silent")
	}
	known := map[string]bool{"fond": true, "estranged": true, "professional": true, "bitter": true}
	for subKey := range p.Subtypes {
		if !known[subKey] {
			mudlog.Warn("conversations.Load()",
				"poolId", p.Id, "subtype", subKey,
				"warning", "subtype key outside documented convention (fond / estranged / professional / bitter)")
		}
	}
	for i, ex := range p.Exchanges {
		if len(ex.Lines) == 1 {
			mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i,
				"warning", "single-line exchange — that's a monologue, not a conversation")
		}
		if len(ex.Lines) > 6 {
			mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i,
				"warning", "exchange has more than 6 lines — player tab-out risk")
		}
		for li, line := range ex.Lines {
			if len(line.Text) > 78 {
				mudlog.Warn("conversations.Load()", "poolId", p.Id, "exchange", i, "line", li,
					"warning", "line text longer than 78 chars — MUD line width recommendation")
			}
		}
	}
}

// emitPairOverrideWarnings logs non-fatal author hints for pair overrides.
func emitPairOverrideWarnings(po *PairOverride) {
	for i, ex := range po.Exchanges {
		if len(ex.Lines) == 1 {
			mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i,
				"warning", "single-line exchange — that's a monologue, not a conversation")
		}
		if len(ex.Lines) > 6 {
			mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i,
				"warning", "exchange has more than 6 lines — player tab-out risk")
		}
		for li, line := range ex.Lines {
			if len(line.Text) > 78 {
				mudlog.Warn("conversations.Load()", "pairId", po.Id, "exchange", i, "line", li,
					"warning", "line text longer than 78 chars — MUD line width recommendation")
			}
		}
	}
}
