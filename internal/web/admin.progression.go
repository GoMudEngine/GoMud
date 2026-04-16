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
	VirtualRank        float64 `json:"virtual_rank"`
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

// getStatBaseValue returns Base + Training for a stat, bypassing the
// computed Value field which may not be populated for disk-loaded users.
func getStatBaseValue(c *characters.Character, statName string) int {
	switch statName {
	case "strength":
		return c.Stats.Strength.Base + c.Stats.Strength.Training
	case "dexterity":
		return c.Stats.Dexterity.Base + c.Stats.Dexterity.Training
	case "perception":
		return c.Stats.Perception.Base + c.Stats.Perception.Training
	case "vitality":
		return c.Stats.Vitality.Base + c.Stats.Vitality.Training
	case "willpower":
		return c.Stats.Willpower.Base + c.Stats.Willpower.Training
	case "charisma":
		return c.Stats.Charisma.Base + c.Stats.Charisma.Training
	default:
		return 0
	}
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

	// Recent offline users from disk (last 14 days)
	userDir := util.FilePath(string(configs.GetFilePathsConfig().DataFiles), "/", "users")
	for _, u := range loadRecentUserFiles(userDir, 14) {
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
// 0.0–1.0 health score for a skill. Spec formula:
//
//	curve  = 1.0 - clamp(|avg_deviation| / soft_cap, 0, 1)
//	stall  = 1.0 - (stall_count / total_with_uses)
//	cluster = 1.0 - clustering_score
//	health = (curve + stall + cluster) / 3.0
func calcHealthScore(avgDeviation float64, stallCount, totalWithUses int, clusteringScore float64, softCap int) float64 {
	if totalWithUses == 0 {
		return 1.0 // no data, no problem
	}
	curveScore := 1.0 - clamp(math.Abs(avgDeviation)/float64(softCap), 0, 1)
	stallScore := 1.0 - float64(stallCount)/float64(totalWithUses)
	clusterScore := 1.0 - clusteringScore

	return (curveScore + stallScore + clusterScore) / 3.0
}

// buildSkillHealth aggregates per-skill health metrics across all players.
func buildSkillHealth(playerList []*users.UserRecord) map[string]skillHealthJSON {
	bal := configs.GetBalanceConfig()
	softCap := int(bal.SkillSoftCap)

	allSkillTags := skills.GetAllSkillNames()
	result := make(map[string]skillHealthJSON, len(allSkillTags))

	for _, sk := range allSkillTags {
		skillName := string(sk)
		mult := skills.GetProgressionMultiplier(skillName)

		distribution := map[string]int{
			"novice": 0, "apprentice": 0, "journeyman": 0,
			"adept": 0, "expert": 0, "master": 0,
		}

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

			// Distribution by tier name
			tier := skills.GetSkillRankDescription(rank)
			distribution[tier]++

			if useCount == 0 {
				continue
			}
			totalWithUses++

			expected := calcExpectedRank(useCount, mult, bal)
			deviation := float64(rank) - expected
			deviationSum += deviation

			if deviation < worstDeviation || worstPlayer == "" {
				worstDeviation = deviation
				worstPlayer = u.Character.Name
			}

			// Stall detection: uses since last gain vs expected uses for next
			approxUsesAtRank := float64(rank) * float64(bal.UsesPerRank) / mult
			usesSinceGain := float64(useCount) - approxUsesAtRank
			if usesSinceGain < 0 {
				usesSinceGain = 0
			}
			chanceAtRank := characters.CalculateProgressionChance(rank, softCap)
			if chanceAtRank > 0 {
				expectedUsesForNext := 1.0 / chanceAtRank
				if expectedUsesForNext > 0 && usesSinceGain/expectedUsesForNext > 2.0 {
					stallCount++
				}
			}
		}

		avgDeviation := 0.0
		if totalWithUses > 0 {
			avgDeviation = deviationSum / float64(totalWithUses)
		}

		// Clustering: max_tier_count / total_with_uses
		clusteringScore := 0.0
		if totalWithUses > 0 {
			maxCount := 0
			for _, c := range distribution {
				if c > maxCount {
					maxCount = c
				}
			}
			clusteringScore = float64(maxCount) / float64(totalWithUses)
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
			"0-50": 0, "51-100": 0, "101-150": 0, "151+": 0,
		}
		for _, u := range playerList {
			if u.Character == nil {
				continue
			}
			val := u.Character.GetStatValue(statName)
			switch {
			case val <= 50:
				distribution["0-50"]++
			case val <= 100:
				distribution["51-100"]++
			case val <= 150:
				distribution["101-150"]++
			default:
				distribution["151+"]++
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
	medianActivity := percentile(activities, 50)
	p10Activity := percentile(activities, 10)

	// Starter spells — excluded from "too easy" flagging
	starterSpells := map[string]bool{
		"conviction-spike": true,
		"minor-light":      true,
	}

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

		flag := ""
		if knownCount == 0 && totalPlayers > 0 {
			// too_hidden: no player with activity > median has it
			hasHighActivity := false
			for _, a := range activities {
				if a > medianActivity {
					hasHighActivity = true
					break
				}
			}
			if hasHighActivity {
				flag = "too_hidden"
			}
		} else if knownCount > 0 && !starterSpells[spellId] {
			// too_easy: players with activity < 10th percentile have it
			for _, u := range playerList {
				if u.Character == nil || u.Character.SpellBook == nil {
					continue
				}
				if _, known := u.Character.SpellBook[spellId]; known {
					if playerTotalActivity(u) < p10Activity {
						flag = "too_easy"
						break
					}
				}
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
	medianActivity := percentile(activities, 50)
	p10Activity := percentile(activities, 10)

	starterRecipes := crafting.GetStarterRecipes()

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

		isStarter := starterRecipes[recipeId] > 0
		flag := ""
		if knownCount == 0 && totalPlayers > 0 {
			hasHighActivity := false
			for _, a := range activities {
				if a > medianActivity {
					hasHighActivity = true
					break
				}
			}
			if hasHighActivity {
				flag = "too_hidden"
			}
		} else if knownCount > 0 && !isStarter {
			for _, u := range playerList {
				if u.Character == nil || u.Character.KnownRecipes == nil {
					continue
				}
				if _, known := u.Character.KnownRecipes[recipeId]; known {
					if playerTotalActivity(u) < p10Activity {
						flag = "too_easy"
						break
					}
				}
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
			virtualRank := math.Round(calcExpectedRank(useCount, mult, bal)*100) / 100
			progChance := characters.CalculateProgressionChance(int(virtualRank), softCap)

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
				Value:    getStatBaseValue(c, statName),
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
