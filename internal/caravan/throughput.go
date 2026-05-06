package caravan

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"sync"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/util"
	"gopkg.in/yaml.v2"
)

// Throughput is the per-caravan cumulative delivery counter, persisted
// at <DataFiles>/caravans/<zone>/<mobId>.yaml. Snapshotted hourly into
// CaravanSnapshot.DeliveriesByTier; deltas surfaced on the dashboard.
type Throughput struct {
	MobId            int         `yaml:"mob_id"`
	Zone             string      `yaml:"zone"`
	DeliveriesByTier map[int]int `yaml:"deliveries_by_tier"`
	LastUpdatedRound uint64      `yaml:"last_updated_round"`
	// LbsDelivered is the cumulative pounds of cargo this caravan has
	// delivered to shops over its lifetime. Drives logistics health
	// scoring (cargo_score axis).
	LbsDelivered uint64 `yaml:"lbs_delivered,omitempty"`
}

var (
	throughputMu      sync.RWMutex
	throughputCache   = map[string]*Throughput{}
	throughputBaseDir string // overridden in tests
)

func throughputKey(zone string, mobId int) string {
	return fmt.Sprintf("%s/%d", zone, mobId)
}

func throughputPath(zone string, mobId int) string {
	base := throughputBaseDir
	if base == "" {
		base = util.FilePath(
			configs.GetFilePathsConfig().DataFiles.String(), `/`, `caravans`,
		)
	}
	zoneSan := util.ConvertForFilename(zone)
	return filepath.Join(base, zoneSan, fmt.Sprintf("%d.yaml", mobId))
}

// GetThroughput returns the Throughput for (zone, mobId), loading from
// disk if not cached. Always returns non-nil (creates empty if missing).
func GetThroughput(zone string, mobId int) *Throughput {
	key := throughputKey(zone, mobId)

	throughputMu.RLock()
	if tp, ok := throughputCache[key]; ok {
		throughputMu.RUnlock()
		return tp
	}
	throughputMu.RUnlock()

	if tp := loadThroughputFromDisk(zone, mobId); tp != nil {
		throughputMu.Lock()
		throughputCache[key] = tp
		throughputMu.Unlock()
		return tp
	}

	tp := &Throughput{
		MobId:            mobId,
		Zone:             zone,
		DeliveriesByTier: map[int]int{},
	}
	throughputMu.Lock()
	throughputCache[key] = tp
	throughputMu.Unlock()
	return tp
}

// IncrementDelivery bumps the per-tier counter and updates LastUpdatedRound.
// Caller is responsible for calling SaveThroughput when convenient.
// Tier <= 0 is silently ignored (item had no rarity_tier set).
func IncrementDelivery(zone string, mobId int, rarityTier int) {
	if rarityTier <= 0 {
		return
	}
	tp := GetThroughput(zone, mobId)
	throughputMu.Lock()
	if tp.DeliveriesByTier == nil {
		tp.DeliveriesByTier = map[int]int{}
	}
	tp.DeliveriesByTier[rarityTier]++
	tp.LastUpdatedRound = util.GetRoundCount()
	throughputMu.Unlock()
}

// AddLbsDelivered increments LbsDelivered by lbs and updates
// LastUpdatedRound. lbs == 0 is a no-op.
// Caller is responsible for calling SaveThroughput when convenient.
func AddLbsDelivered(zone string, mobId int, lbs uint64) {
	if lbs == 0 {
		return
	}
	tp := GetThroughput(zone, mobId)
	throughputMu.Lock()
	tp.LbsDelivered += lbs
	tp.LastUpdatedRound = util.GetRoundCount()
	throughputMu.Unlock()
}

// SaveThroughput writes the cached Throughput for (zone, mobId) to disk.
func SaveThroughput(zone string, mobId int) error {
	key := throughputKey(zone, mobId)

	throughputMu.RLock()
	tp, ok := throughputCache[key]
	throughputMu.RUnlock()
	if !ok {
		return fmt.Errorf("caravan.SaveThroughput: no cached entry for %s", key)
	}

	p := throughputPath(zone, mobId)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		return fmt.Errorf("caravan.SaveThroughput: mkdir: %w", err)
	}
	data, err := yaml.Marshal(tp)
	if err != nil {
		return fmt.Errorf("caravan.SaveThroughput: marshal: %w", err)
	}
	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("caravan.SaveThroughput: write: %w", err)
	}
	return nil
}

// SaveAllThroughputs persists every cached entry. Intended for graceful
// shutdown.
func SaveAllThroughputs() {
	throughputMu.RLock()
	keys := make([]struct {
		zone  string
		mobId int
	}, 0, len(throughputCache))
	for _, tp := range throughputCache {
		keys = append(keys, struct {
			zone  string
			mobId int
		}{tp.Zone, tp.MobId})
	}
	throughputMu.RUnlock()
	for _, k := range keys {
		if err := SaveThroughput(k.zone, k.mobId); err != nil {
			mudlog.Error("caravan.SaveAllThroughputs", "error", err)
		}
	}
}

// ClearThroughputCache drops all cached entries (for tests).
func ClearThroughputCache() {
	throughputMu.Lock()
	throughputCache = map[string]*Throughput{}
	throughputMu.Unlock()
}

// SetThroughputBaseDirForTest overrides the on-disk base directory
// (for use in tests with t.TempDir()).
func SetThroughputBaseDirForTest(dir string) {
	throughputBaseDir = dir
}

// PrewarmThroughputFromPersistedFiles loads every persisted caravan
// throughput YAML found under <DataFiles>/caravans/<zone>/<mobId>.yaml
// into the in-memory cache. Returns the count loaded.
func PrewarmThroughputFromPersistedFiles() (int, error) {
	base := util.FilePath(
		configs.GetFilePathsConfig().DataFiles.String(), `/`, `caravans`,
	)
	return prewarmThroughputFrom(base)
}

func prewarmThroughputFrom(baseDir string) (int, error) {
	zoneDirs, err := os.ReadDir(baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, fmt.Errorf("caravan.PrewarmThroughputFromPersistedFiles: readdir: %w", err)
	}
	loaded := 0
	fileRe := regexp.MustCompile(`^(\d+)\.yaml$`)
	for _, zd := range zoneDirs {
		if !zd.IsDir() {
			continue
		}
		files, err := os.ReadDir(filepath.Join(baseDir, zd.Name()))
		if err != nil {
			mudlog.Warn("caravan.PrewarmThroughputFromPersistedFiles",
				"warning", "readdir zone failed",
				"zone", zd.Name(), "error", err)
			continue
		}
		for _, f := range files {
			m := fileRe.FindStringSubmatch(f.Name())
			if m == nil {
				continue
			}
			mobId, _ := strconv.Atoi(m[1])
			zoneDir := zd.Name()
			tp := loadThroughputFromDisk(zoneDir, mobId)
			if tp == nil {
				continue
			}
			// Cache key uses tp.Zone (display-name, e.g. "Thornwall City")
			// not the directory name (snake_case, e.g. "thornwall_city").
			// Runtime callers index by wagon.Zone which is the display
			// form. Keying by zoneDir would create a phantom entry that
			// SaveAllThroughputs can't round-trip via SaveThroughput.
			key := throughputKey(tp.Zone, mobId)
			throughputMu.Lock()
			throughputCache[key] = tp
			throughputMu.Unlock()
			loaded++
		}
	}
	return loaded, nil
}

func loadThroughputFromDisk(zone string, mobId int) *Throughput {
	p := throughputPath(zone, mobId)
	raw, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	var tp Throughput
	if err := yaml.Unmarshal(raw, &tp); err != nil {
		mudlog.Error("caravan.loadThroughputFromDisk", "path", p, "error", err)
		return nil
	}
	if tp.DeliveriesByTier == nil {
		tp.DeliveriesByTier = map[int]int{}
	}
	return &tp
}
