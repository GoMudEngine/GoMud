package opinions

// Opinion is one NPC's stored disposition toward one player.
// Persisted in YAML, keyed by userId inside the parent MobOpinions.
type Opinion struct {
	Score            int    `yaml:"score"`
	LastUpdatedRound uint64 `yaml:"last_updated_round"`
}

// MobOpinions is one mob template's full opinion table — every
// player this NPC has ever held a non-default opinion of, plus the
// snapshotted default disposition the file owns as source of truth.
//
// One MobOpinions persists per mob template at:
//
//	_datafiles/world/dogmud/opinions/{mobId}-{namesimple}.yaml
//
// All instances of a mob template share this table.
type MobOpinions struct {
	MobId              int               `yaml:"mob_id"`
	DefaultDisposition int               `yaml:"default_disposition"`
	Opinions           map[int]*Opinion  `yaml:"opinions"` // userId → opinion
}

// Tier is a banded view of the disposition score, for consumers
// (dialogue gates, combat aggro logic) that prefer thresholds over
// raw numbers.
type Tier int

const (
	TierHostile  Tier = iota // <= -50
	TierCold                 // -49 .. -15
	TierNeutral              // -14 .. +14
	TierWarm                 // +15 .. +49
	TierFriendly             // >= +50
)

// Score range — every Set/Bump clamps to this window.
const (
	ScoreMin = -100
	ScoreMax = +100
)
