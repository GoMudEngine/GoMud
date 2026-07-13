package migration

// classify.go — player mutation-migration classifier (0.14.0). Slots each
// existing account into a cluster from its play history, per
// docs/superpowers/specs/2026-07-11-mutation-migration-design.md §3, with
// thresholds fit against the 34 real prod accounts (§4).

// PlayerSignals are the classification inputs extracted from a player save.
type PlayerSignals struct {
	Role             string // top-level UserRecord role ("admin" | "user" | ...)
	Companions       int    // len(character.companions)
	Manifestation    int    // character.skills.manifestation
	SpellbookDepth   int    // len(character.spellbook)
	WillpowerTrain   int    // character.stats.willpower.training
	TopCombat        int    // highest cluster-mapped combat skill level
	TopCombatCluster string // cluster of TopCombat (colossus|ravener|stalker|zealot)
	CraftSum         int    // summed crafting-skill levels
}

// Thresholds fit against the 34 real prod accounts (tools/analyze_players.py) so
// each lands per spec §4. Verified: Megalomania→admin, meirok→manifester,
// Deios→ethereal, Saphira→ravener, fyttyn/pruuk/Duard/Oriana→chrysifier, rest→generalist.
const (
	companionManifesterMin  = 2   // >=2 companions -> Manifester (only meirok)
	crafterSumMin           = 150 // summed crafting -> Chrysifier (crafters 370-477; Deios 36 stays caster)
	casterSpellbookMin      = 5   // spellbook entries
	casterWillpowerTrainMin = 50  // willpower training (Deios 88 clears; Calabe 30 does not)
	combatStandoutMin       = 25  // top combat skill (Saphira 27 clears; aitester 17 does not)
)

// ClassifyPlayer slots a player into a cluster. Order matters: the crafter check
// runs BEFORE Ethereal/combat because the heaviest crafters also dabbled deep
// spellbooks + high willpower (they would otherwise mis-fire as casters) — their
// identity is *maker* (spec §3.4). Returns one of: admin, manifester, chrysifier,
// ethereal, colossus, ravener, stalker, zealot, generalist.
func ClassifyPlayer(s PlayerSignals) string {
	if s.Role == "admin" {
		return "admin"
	}
	// A true broodmaster fields a real stable (meirok: 2 companions). A single
	// dabbled companion does not qualify (spec §3.1) — those accounts are crafters.
	if s.Companions >= companionManifesterMin {
		return "manifester"
	}
	// Heavy crafting dominates, even over an incidental deep spellbook.
	if s.CraftSum >= crafterSumMin {
		return "chrysifier"
	}
	// Deep spellbook + trained will -> caster.
	if s.SpellbookDepth >= casterSpellbookMin && s.WillpowerTrain >= casterWillpowerTrainMin {
		return "ethereal"
	}
	// Genuine combat standout -> its cluster.
	if s.TopCombat >= combatStandoutMin && s.TopCombatCluster != "" {
		return s.TopCombatCluster
	}
	// Kept your options open, keep them open.
	return "generalist"
}
