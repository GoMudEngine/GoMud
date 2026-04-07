package web

import (
	"encoding/json"
	"html/template"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"github.com/GoMudEngine/GoMud/internal/util"

	"gopkg.in/yaml.v2"
)

// ─── JSON response types ──────────────────────────────────────────────────────

type progressionAPIResponse struct {
	Skills  map[string]skillHealthJSON  `json:"skills"`
	Stats   map[string]statHealthJSON   `json:"stats"`
	Spells  map[string]spellHealthJSON  `json:"spells"`
	Recipes map[string]recipeHealthJSON `json:"recipes"`
	Players []playerJSON                `json:"players"`
}

type skillHealthJSON struct {
	HealthScore      float64          `json:"health_score"`
	AvgDeviation     float64          `json:"avg_deviation"`
	WorstPlayer      string           `json:"worst_player"`
	WorstDeviation   float64          `json:"worst_deviation"`
	StallCount       int              `json:"stall_count"`
	TotalWithUses    int              `json:"total_with_uses"`
	Distribution     map[string]int   `json:"distribution"`
	ClusteringScore  float64          `json:"clustering_score"`
}

type statHealthJSON struct {
	Distribution map[string]int `json:"distribution"`
}

type spellHealthJSON struct {
	Name                    string  `json:"name"`
	School                  string  `json:"school"`
	KnownCount              int     `json:"known_count"`
	TotalPlayers            int     `json:"total_players"`
	AvgActivityAtDiscovery  float64 `json:"avg_activity_at_discovery"`
	Flag                    string  `json:"flag"`
}

type recipeHealthJSON struct {
	Name                    string  `json:"name"`
	Skill                   string  `json:"skill"`
	KnownCount              int     `json:"known_count"`
	TotalPlayers            int     `json:"total_players"`
	AvgActivityAtDiscovery  float64 `json:"avg_activity_at_discovery"`
	Flag                    string  `json:"flag"`
}

type playerJSON struct {
	Name          string                   `json:"name"`
	TotalActivity int                      `json:"total_activity"`
	Skills        map[string]playerSkillJSON `json:"skills"`
	Stats         map[string]playerStatJSON  `json:"stats"`
	SpellsKnown   int                      `json:"spells_known"`
	SpellsTotal   int                      `json:"spells_total"`
	RecipesKnown  int                      `json:"recipes_known"`
	RecipesTotal  int                      `json:"recipes_total"`
}

type playerSkillJSON struct {
	Rank               int     `json:"rank"`
	Tier               string  `json:"tier"`
	UseCount           int     `json:"use_count"`
	VirtualRank        int     `json:"virtual_rank"`
	ProgressionChance  float64 `json:"progression_chance"`
}

type playerStatJSON struct {
	Value    int `json:"value"`
	Training int `json:"training"`
	UseCount int `json:"use_count"`
}

// ─── Helpers ──────────────────────────────────────────────────────────────────

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

func getStatTraining(c *characters.Character, statName string) int {
	switch statName {
	case "strength":
		return c.Stats.Strength.Training
	case "dexterity":
		return c.Stats.Dexterity.Training
	case "perception":
		return c.Stats.Perception.Training
	case "vitality":
		return c.Stats.Vitality.Training
	case "willpower":
		return c.Stats.Willpower.Training
	case "charisma":
		return c.Stats.Charisma.Training
	default:
		return 0
	}
}

// ─── Player Loading ───────────────────────────────────────────────────────────

// loadRecentUserFiles reads YAML files from dir, filters by mod time <=
// maxDays ago and role == user.  Returns only non-admin, non-guest records.
func loadRecentUserFiles(dir string, maxDays int) []*users.UserRecord {
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	var result []*users.UserRecord

	entries, err := os.ReadDir(dir)
	if err != nil {
		mudlog.Error("progressionDashboard", "error", "cannot read user dir: "+err.Error())
		return result
	}

	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".yaml" {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		if info.ModTime().Before(cutoff) {
			continue
		}

		fullPath := filepath.Join(dir, e.Name())
		data, err := os.ReadFile(fullPath)
		if err != nil {
			continue
		}

		u := &users.UserRecord{}
		if err := yaml.Unmarshal(data, u); err != nil {
			continue
		}
		if u.Role == users.RoleAdmin || u.Role == users.RoleGuest {
			continue
		}
		if u.Character == nil {
			continue
		}
		result = append(result, u)
	}
	return result
}

// collectPlayers merges online users with recent offline users, deduping by
// UserId.  Online records are preferred over disk snapshots.
func collectPlayers() []*users.UserRecord {
	seen := map[int]bool{}
	var all []*users.UserRecord

	// Online users first (authoritative, live state)
	for _, u := range users.GetAllActiveUsers() {
		if u.Role == users.RoleAdmin || u.Role == users.RoleGuest {
			continue
		}
		if u.Character == nil {
			continue
		}
		seen[u.UserId] = true
		all = append(all, u)
	}

	// Recent offline users from disk (last 90 days)
	userDir := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), "/", "users")
	for _, u := range loadRecentUserFiles(userDir, 90) {
		if seen[u.UserId] {
			continue
		}
		seen[u.UserId] = true
		all = append(all, u)
	}
	return all
}

// ─── Aggregation Logic ────────────────────────────────────────────────────────

// calcExpectedRank simulates the cumulative progression curve for a skill.
// It returns the expected virtual rank after useCount uses, given the
// skill's progression multiplier.
func calcExpectedRank(useCount int, mult float64, bal configs.Balance) float64 {
	usesPerRank := int(bal.UsesPerRank)
	if usesPerRank <= 0 {
		usesPerRank = 25
	}
	if mult <= 0 {
		mult = 1.0
	}
	adjusted := int(float64(useCount) * mult)
	return float64(adjusted) / float64(usesPerRank)
}

// calcHealthScore combines deviation, stall, and clustering metrics into a
// 0–100 health score for a skill.
func calcHealthScore(avgDeviation float64, stallCount, totalWithUses int, clusteringScore float64, softCap int) float64 {
	if totalWithUses == 0 {
		return 0
	}
	// Deviation penalty: ideal deviation ratio is ≈0 (everyone progressing evenly)
	devRatio := clamp(avgDeviation/float64(softCap), 0, 1)
	devScore := 1.0 - devRatio

	// Stall penalty: fraction of active players who appear stalled
	stallRatio := clamp(float64(stallCount)/float64(totalWithUses), 0, 1)
	stallScore := 1.0 - stallRatio

	// Clustering bonus/penalty (0=perfectly even, 1=all at same rank)
	clusterPenalty := clamp(clusteringScore, 0, 1) * 0.2

	raw := (devScore*0.5 + stallScore*0.5) - clusterPenalty
	return clamp(raw, 0, 1) * 100
}

// buildSkillHealth aggregates per-skill health metrics across all players.
func buildSkillHealth(playerList []*users.UserRecord) map[string]skillHealthJSON {
	bal := configs.GetBalanceConfig()
	softCap := int(bal.SkillSoftCap)

	type perPlayerEntry struct {
		name       string
		rank       int
		virtualRank float64
	}

	allSkillTags := skills.GetAllSkillNames()
	result := make(map[string]skillHealthJSON, len(allSkillTags))

	for _, sk := range allSkillTags {
		skillName := string(sk)
		mult := skills.GetProgressionMultiplier(skillName)

		distribution := map[string]int{
			"0":      0,
			"1-10":   0,
			"11-25":  0,
			"26-50":  0,
			"50+":    0,
		}

		var entries []perPlayerEntry
		totalWithUses := 0
		stallCount := 0
		worstPlayer := ""
		worstDeviation := 0.0
		deviationSum := 0.0

		for _, u := range playerList {
			if u.Character == nil {
				continue
			}
			rank := 0
			if u.Character.Skills != nil {
				rank = u.Character.Skills[skillName]
			}
			useCount := u.Character.GetSkillUseCount(skillName)

			// Distribution buckets
			switch {
			case rank == 0:
				distribution["0"]++
			case rank <= 10:
				distribution["1-10"]++
			case rank <= 25:
				distribution["11-25"]++
			case rank <= 50:
				distribution["26-50"]++
			default:
				distribution["50+"]++
			}

			if useCount == 0 {
				continue
			}
			totalWithUses++

			expected := calcExpectedRank(useCount, mult, bal)
			deviation := math.Abs(float64(rank) - expected)
			deviationSum += deviation

			if deviation > worstDeviation {
				worstDeviation = deviation
				worstPlayer = u.Character.Name
			}

			// Stall detection: rank hasn't moved but has significant use count
			if rank < 2 && useCount > int(bal.UsesPerRank)*3 {
				stallCount++
			}

			entries = append(entries, perPlayerEntry{
				name:        u.Character.Name,
				rank:        rank,
				virtualRank: expected,
			})
		}

		avgDeviation := 0.0
		if totalWithUses > 0 {
			avgDeviation = deviationSum / float64(totalWithUses)
		}

		// Clustering score: stddev of ranks as fraction of softCap
		clusteringScore := 0.0
		if len(entries) > 1 {
			sum := 0.0
			for _, e := range entries {
				sum += float64(e.rank)
			}
			mean := sum / float64(len(entries))
			variance := 0.0
			for _, e := range entries {
				d := float64(e.rank) - mean
				variance += d * d
			}
			variance /= float64(len(entries))
			stddev := math.Sqrt(variance)
			// Normalize: low stddev = high clustering
			if float64(softCap) > 0 {
				clusteringScore = 1.0 - clamp(stddev/float64(softCap), 0, 1)
			}
		}

		health := calcHealthScore(avgDeviation, stallCount, totalWithUses, clusteringScore, softCap)

		result[skillName] = skillHealthJSON{
			HealthScore:     health,
			AvgDeviation:    avgDeviation,
			WorstPlayer:     worstPlayer,
			WorstDeviation:  worstDeviation,
			StallCount:      stallCount,
			TotalWithUses:   totalWithUses,
			Distribution:    distribution,
			ClusteringScore: clusteringScore,
		}
	}
	return result
}

// buildStatHealth aggregates per-stat distribution across all players.
func buildStatHealth(playerList []*users.UserRecord) map[string]statHealthJSON {
	statNames := []string{
		"strength", "dexterity", "perception", "vitality", "willpower", "charisma",
	}
	result := make(map[string]statHealthJSON, len(statNames))

	for _, statName := range statNames {
		distribution := map[string]int{
			"<80":     0,
			"80-99":   0,
			"100":     0,
			"101-120": 0,
			"121-150": 0,
			"150+":    0,
		}
		for _, u := range playerList {
			if u.Character == nil {
				continue
			}
			val := u.Character.GetStatValue(statName)
			switch {
			case val < 80:
				distribution["<80"]++
			case val < 100:
				distribution["80-99"]++
			case val == 100:
				distribution["100"]++
			case val <= 120:
				distribution["101-120"]++
			case val <= 150:
				distribution["121-150"]++
			default:
				distribution["150+"]++
			}
		}
		result[statName] = statHealthJSON{Distribution: distribution}
	}
	return result
}

// ─── Discovery Health ─────────────────────────────────────────────────────────

// playerTotalActivity returns total skill uses + stat uses for a player.
func playerTotalActivity(u *users.UserRecord) int {
	if u.Character == nil {
		return 0
	}
	total := 0
	if u.Character.SkillUseCount != nil {
		for _, v := range u.Character.SkillUseCount {
			total += v
		}
	}
	if u.Character.StatUseCount != nil {
		for _, v := range u.Character.StatUseCount {
			total += v
		}
	}
	return total
}

// playerActivities returns a sorted list of total activity values across
// all players, used for percentile calculations.
func playerActivities(playerList []*users.UserRecord) []int {
	out := make([]int, 0, len(playerList))
	for _, u := range playerList {
		out = append(out, playerTotalActivity(u))
	}
	sort.Ints(out)
	return out
}

// percentile returns the value at the given percentile (0-100) of a sorted
// integer slice.
func percentile(sorted []int, pct int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := int(math.Ceil(float64(pct)/100*float64(len(sorted)))) - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// buildSpellHealth aggregates discovery metrics for every known spell.
func buildSpellHealth(playerList []*users.UserRecord) map[string]spellHealthJSON {
	allSpells := spells.GetAllSpells()
	totalPlayers := len(playerList)

	activities := playerActivities(playerList)
	p25 := percentile(activities, 25)
	p75 := percentile(activities, 75)

	result := make(map[string]spellHealthJSON, len(allSpells))

	for spellId, spellData := range allSpells {
		knownCount := 0
		activitySum := 0

		for _, u := range playerList {
			if u.Character == nil || u.Character.SpellBook == nil {
				continue
			}
			if _, known := u.Character.SpellBook[spellId]; known {
				knownCount++
				activitySum += playerTotalActivity(u)
			}
		}

		school := ""
		if len(spellData.Schools) > 0 {
			school = spellData.Schools[0]
		}

		avgActivity := 0.0
		if knownCount > 0 {
			avgActivity = float64(activitySum) / float64(knownCount)
		}

		flag := "ok"
		if knownCount == 0 {
			flag = "unlearned"
		} else {
			discoveryRate := float64(knownCount) / float64(totalPlayers)
			if discoveryRate < 0.05 && avgActivity > float64(p75) {
				flag = "bottleneck"
			} else if discoveryRate > 0.95 && avgActivity < float64(p25) {
				flag = "trivial"
			}
		}

		result[spellId] = spellHealthJSON{
			Name:                   spellData.Name,
			School:                 school,
			KnownCount:             knownCount,
			TotalPlayers:           totalPlayers,
			AvgActivityAtDiscovery: avgActivity,
			Flag:                   flag,
		}
	}
	return result
}

// buildRecipeHealth aggregates discovery metrics for every known recipe.
func buildRecipeHealth(playerList []*users.UserRecord) map[string]recipeHealthJSON {
	allRecipes := crafting.GetAll()
	totalPlayers := len(playerList)

	activities := playerActivities(playerList)
	p25 := percentile(activities, 25)
	p75 := percentile(activities, 75)

	result := make(map[string]recipeHealthJSON, len(allRecipes))

	for recipeId, recipe := range allRecipes {
		knownCount := 0
		activitySum := 0

		for _, u := range playerList {
			if u.Character == nil || u.Character.KnownRecipes == nil {
				continue
			}
			if _, known := u.Character.KnownRecipes[recipeId]; known {
				knownCount++
				activitySum += playerTotalActivity(u)
			}
		}

		avgActivity := 0.0
		if knownCount > 0 {
			avgActivity = float64(activitySum) / float64(knownCount)
		}

		flag := "ok"
		if knownCount == 0 {
			flag = "unlearned"
		} else {
			discoveryRate := float64(knownCount) / float64(totalPlayers)
			if discoveryRate < 0.05 && avgActivity > float64(p75) {
				flag = "bottleneck"
			} else if discoveryRate > 0.95 && avgActivity < float64(p25) {
				flag = "trivial"
			}
		}

		result[recipeId] = recipeHealthJSON{
			Name:                   recipe.Name,
			Skill:                  recipe.Skill,
			KnownCount:             knownCount,
			TotalPlayers:           totalPlayers,
			AvgActivityAtDiscovery: avgActivity,
			Flag:                   flag,
		}
	}
	return result
}

// ─── Player Overview ──────────────────────────────────────────────────────────

// buildPlayerOverview returns a per-player summary for the dashboard.
func buildPlayerOverview(playerList []*users.UserRecord) []playerJSON {
	bal := configs.GetBalanceConfig()
	softCap := int(bal.SkillSoftCap)
	allSpells := spells.GetAllSpells()
	allRecipes := crafting.GetAll()

	result := make([]playerJSON, 0, len(playerList))

	for _, u := range playerList {
		if u.Character == nil {
			continue
		}
		c := u.Character

		// Skills
		skillsMap := make(map[string]playerSkillJSON)
		for _, sk := range skills.GetAllSkillNames() {
			name := string(sk)
			rank := 0
			if c.Skills != nil {
				rank = c.Skills[name]
			}
			useCount := c.GetSkillUseCount(name)
			mult := skills.GetProgressionMultiplier(name)
			virtualRank := int(calcExpectedRank(useCount, mult, bal))
			progChance := characters.CalculateProgressionChance(virtualRank, softCap)

			skillsMap[name] = playerSkillJSON{
				Rank:              rank,
				Tier:              skills.GetSkillRankDescription(rank),
				UseCount:          useCount,
				VirtualRank:       virtualRank,
				ProgressionChance: progChance,
			}
		}

		// Stats
		statNames := []string{
			"strength", "dexterity", "perception", "vitality", "willpower", "charisma",
		}
		statsMap := make(map[string]playerStatJSON, len(statNames))
		for _, statName := range statNames {
			statsMap[statName] = playerStatJSON{
				Value:    c.GetStatValue(statName),
				Training: getStatTraining(c, statName),
				UseCount: c.GetStatUseCount(statName),
			}
		}

		// Spells
		spellsKnown := 0
		if c.SpellBook != nil {
			spellsKnown = len(c.SpellBook)
		}
		spellsTotal := len(allSpells)

		// Recipes
		recipesKnown := 0
		if c.KnownRecipes != nil {
			recipesKnown = len(c.KnownRecipes)
		}
		recipesTotal := len(allRecipes)

		result = append(result, playerJSON{
			Name:          c.Name,
			TotalActivity: playerTotalActivity(u),
			Skills:        skillsMap,
			Stats:         statsMap,
			SpellsKnown:   spellsKnown,
			SpellsTotal:   spellsTotal,
			RecipesKnown:  recipesKnown,
			RecipesTotal:  recipesTotal,
		})
	}

	// Sort by total activity descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalActivity > result[j].TotalActivity
	})

	return result
}

// ─── HTTP Handlers ────────────────────────────────────────────────────────────

func progressionIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/progression/index.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
		return
	}
	tplData := struct{}{}
	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}

func progressionAPI(w http.ResponseWriter, r *http.Request) {
	playerList := collectPlayers()

	// Allow optional ?days=N to narrow the offline window (ignored for online)
	if daysStr := r.URL.Query().Get("days"); daysStr != "" {
		if days, err := strconv.Atoi(daysStr); err == nil && days > 0 {
			userDir := util.FilePath(
				string(configs.GetFilePathsConfig().DataFiles), "/", "users",
			)
			offline := loadRecentUserFiles(userDir, days)
			// Rebuild with custom window — still prefer online records
			seen := map[int]bool{}
			var merged []*users.UserRecord
			for _, u := range users.GetAllActiveUsers() {
				if u.Role == users.RoleAdmin || u.Role == users.RoleGuest {
					continue
				}
				if u.Character == nil {
					continue
				}
				seen[u.UserId] = true
				merged = append(merged, u)
			}
			for _, u := range offline {
				if seen[u.UserId] {
					continue
				}
				seen[u.UserId] = true
				merged = append(merged, u)
			}
			playerList = merged
		}
	}

	resp := progressionAPIResponse{
		Skills:  buildSkillHealth(playerList),
		Stats:   buildStatHealth(playerList),
		Spells:  buildSpellHealth(playerList),
		Recipes: buildRecipeHealth(playerList),
		Players: buildPlayerOverview(playerList),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
