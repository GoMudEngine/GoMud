package bounties

import (
	"os"
	"path/filepath"

	"github.com/GoMudEngine/GoMud/internal/knowledge"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"gopkg.in/yaml.v3"
)

// standingEntry is a single authored bounty declaration read from the
// standing seed file (bounties.standing.yaml).
type standingEntry struct {
	TargetMobId   int    `yaml:"target_mob_id"`
	IssuerFaction string `yaml:"issuer_faction"`
	Reason        string `yaml:"reason"`
}

// seedFromEntries declares standing bounties for each entry that does not
// already have an open bounty. Entries with a zero TargetMobId or empty
// IssuerFaction are silently skipped. Returns the number of new bounties
// declared (so callers can log a meaningful count).
func seedFromEntries(entries []standingEntry) int {
	count := 0
	for _, e := range entries {
		if e.TargetMobId == 0 || e.IssuerFaction == "" {
			continue
		}
		target := knowledge.MobSubject(e.TargetMobId)
		if len(OpenForTarget(target)) > 0 {
			// Idempotent: a standing bounty for this target is already open.
			continue
		}
		_, err := Declare(
			FactionIssuer(e.IssuerFaction),
			target,
			ConditionKill,
			0, // never expires
			DeclareOpts{DeclaredReason: e.Reason},
		)
		if err != nil {
			mudlog.Warn("bounties.SeedStanding: declare failed",
				"target_mob_id", e.TargetMobId,
				"issuer_faction", e.IssuerFaction,
				"error", err)
			continue
		}
		count++
	}
	return count
}

// SeedStanding reads the standing bounty seed file at
// <dataFilesPath>/bounties.standing.yaml and declares any entries that
// do not already have an open bounty. A missing file is treated as an
// empty list (no-op). Parse errors are logged as warnings.
//
// This function is idempotent: calling it multiple times (e.g., across
// server restarts) will not create duplicate bounties for the same
// target because seedFromEntries skips any target that already has an
// open bounty in the registry.
func SeedStanding(dataFilesPath string) {
	path := filepath.Join(dataFilesPath, "bounties.standing.yaml")

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Missing seed file is not an error — just nothing to seed.
			return
		}
		mudlog.Warn("bounties.SeedStanding: read failed", "path", path, "error", err)
		return
	}

	var entries []standingEntry
	if err := yaml.Unmarshal(data, &entries); err != nil {
		mudlog.Warn("bounties.SeedStanding: parse failed", "path", path, "error", err)
		return
	}

	declared := seedFromEntries(entries)
	mudlog.Info("bounties.SeedStanding", "declared", declared, "path", path)
}
