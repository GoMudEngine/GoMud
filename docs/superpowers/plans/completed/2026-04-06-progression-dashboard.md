# Progression Dashboard Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a "Progression" admin tab with system-level health metrics for the use-based progression system.

**Architecture:** Go backend computes all aggregations (expected curves, stall detection, population distribution, discovery health) and serves JSON. HTMX frontend renders tables and Chart.js visualizations. Follows the existing Combat Stats admin tab pattern.

**Tech Stack:** Go, HTMX, Bootstrap 4, Chart.js (all already in use).

---

## File Structure

| Action | Path | Responsibility |
|--------|------|----------------|
| Create | `internal/web/admin.progression.go` | Player loading, aggregation logic, JSON API handler, HTML handler |
| Create | `_datafiles/html/admin/progression/index.html` | HTMX template with System Health panels + Player Overview table |
| Modify | `internal/web/web.go` | Register routes for `/admin/progression/` and `/admin/api/progression/` |

---

### Task 1: JSON Response Types and Player Loader

**Files:**
- Create: `internal/web/admin.progression.go`
- Test: `internal/web/admin_progression_test.go`

- [ ] **Step 1: Write test for player loading**

Create `internal/web/admin_progression_test.go`:

```go
package web

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestLoadRecentUserFiles_SkipsOldFiles(t *testing.T) {
	// Create a temp dir with two user YAML files
	dir := t.TempDir()

	// Recent file (< 14 days)
	recentFile := filepath.Join(dir, "1.yaml")
	os.WriteFile(recentFile, []byte("userid: 1\nrole: user\nusername: alice\ncharacter:\n  skills:\n    weapon-combat: 5\n"), 0644)

	// Old file (> 14 days)
	oldFile := filepath.Join(dir, "2.yaml")
	os.WriteFile(oldFile, []byte("userid: 2\nrole: user\nusername: bob\ncharacter:\n  skills:\n    weapon-combat: 3\n"), 0644)
	oldTime := time.Now().Add(-30 * 24 * time.Hour)
	os.Chtimes(oldFile, oldTime, oldTime)

	users := loadRecentUserFiles(dir, 14)
	assert.Len(t, users, 1)
	assert.Equal(t, "alice", users[0].Username)
}

func TestLoadRecentUserFiles_SkipsAdmins(t *testing.T) {
	dir := t.TempDir()
	adminFile := filepath.Join(dir, "1.yaml")
	os.WriteFile(adminFile, []byte("userid: 1\nrole: admin\nusername: goduser\ncharacter:\n  skills: {}\n"), 0644)

	users := loadRecentUserFiles(dir, 14)
	assert.Len(t, users, 0)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/ -run TestLoadRecentUserFiles -v`
Expected: FAIL — `loadRecentUserFiles` undefined

- [ ] **Step 3: Write JSON response types and player loader**

Create `internal/web/admin.progression.go`:

```go
package web

import (
	"encoding/json"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/GoMudEngine/GoMud/internal/characters"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/crafting"
	"github.com/GoMudEngine/GoMud/internal/skills"
	"github.com/GoMudEngine/GoMud/internal/spells"
	"github.com/GoMudEngine/GoMud/internal/users"
	"gopkg.in/yaml.v2"
	"time"
)

// ── JSON Response Types ─────────────────────────────────────────────────────

type progressionAPIResponse struct {
	Skills  map[string]skillHealthJSON  `json:"skills"`
	Stats   map[string]statHealthJSON   `json:"stats"`
	Spells  map[string]spellHealthJSON  `json:"spells"`
	Recipes map[string]recipeHealthJSON `json:"recipes"`
	Players []playerJSON               `json:"players"`
}

type skillHealthJSON struct {
	HealthScore     float64        `json:"health_score"`
	AvgDeviation    float64        `json:"avg_deviation"`
	WorstPlayer     string         `json:"worst_player"`
	WorstDeviation  float64        `json:"worst_deviation"`
	StallCount      int            `json:"stall_count"`
	TotalWithUses   int            `json:"total_with_uses"`
	Distribution    map[string]int `json:"distribution"`
	ClusteringScore float64        `json:"clustering_score"`
}

type statHealthJSON struct {
	Distribution map[string]int `json:"distribution"`
}

type spellHealthJSON struct {
	Name                   string  `json:"name"`
	School                 string  `json:"school"`
	KnownCount             int     `json:"known_count"`
	TotalPlayers           int     `json:"total_players"`
	AvgActivityAtDiscovery float64 `json:"avg_activity_at_discovery"`
	Flag                   string  `json:"flag"`
}

type recipeHealthJSON struct {
	Name                   string  `json:"name"`
	Skill                  string  `json:"skill"`
	KnownCount             int     `json:"known_count"`
	TotalPlayers           int     `json:"total_players"`
	AvgActivityAtDiscovery float64 `json:"avg_activity_at_discovery"`
	Flag                   string  `json:"flag"`
}

type playerJSON struct {
	Name          string                    `json:"name"`
	TotalActivity int                       `json:"total_activity"`
	Skills        map[string]playerSkillJSON `json:"skills"`
	Stats         map[string]playerStatJSON  `json:"stats"`
	SpellsKnown  int                        `json:"spells_known"`
	SpellsTotal  int                        `json:"spells_total"`
	RecipesKnown int                        `json:"recipes_known"`
	RecipesTotal int                        `json:"recipes_total"`
}

type playerSkillJSON struct {
	Rank             int     `json:"rank"`
	Tier             string  `json:"tier"`
	UseCount         int     `json:"use_count"`
	VirtualRank      float64 `json:"virtual_rank"`
	ProgressionChance float64 `json:"progression_chance"`
}

type playerStatJSON struct {
	Value    int `json:"value"`
	Training int `json:"training"`
	UseCount int `json:"use_count"`
}

// ── Player Loading ──────────────────────────────────────────────────────────

// loadRecentUserFiles scans a directory of user YAML files and returns
// UserRecords for non-admin users whose file was modified within maxDays.
func loadRecentUserFiles(dir string, maxDays int) []*users.UserRecord {
	cutoff := time.Now().AddDate(0, 0, -maxDays)
	var result []*users.UserRecord

	entries, err := os.ReadDir(dir)
	if err != nil {
		return result
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.ModTime().Before(cutoff) {
			continue
		}

		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}

		var u users.UserRecord
		if err := yaml.Unmarshal(data, &u); err != nil {
			continue
		}

		if u.Role != users.RoleUser || u.Character == nil {
			continue
		}

		result = append(result, &u)
	}

	return result
}

// collectPlayers merges online users with recently-active offline users,
// deduplicating by UserId. Returns only Role=="user" characters.
func collectPlayers() []*users.UserRecord {
	seen := map[int]bool{}
	var result []*users.UserRecord

	// Online users first
	for _, u := range users.GetAllActiveUsers() {
		if u.Role != users.RoleUser || u.Character == nil {
			continue
		}
		seen[u.UserId] = true
		result = append(result, u)
	}

	// Offline users from disk
	userDir := configs.GetConfig().FolderUserData.String()
	for _, u := range loadRecentUserFiles(userDir, 14) {
		if seen[u.UserId] {
			continue
		}
		seen[u.UserId] = true
		result = append(result, u)
	}

	return result
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./internal/web/ -run TestLoadRecentUserFiles -v`
Expected: PASS (both tests)

- [ ] **Step 5: Commit**

```bash
git add internal/web/admin.progression.go internal/web/admin_progression_test.go
git commit -m "feat: progression dashboard — JSON types and player loader"
```

---

### Task 2: Aggregation Logic (Skill Health, Stalls, Distribution)

**Files:**
- Modify: `internal/web/admin.progression.go`
- Test: `internal/web/admin_progression_test.go`

- [ ] **Step 1: Write test for expected rank calculation**

Append to `internal/web/admin_progression_test.go`:

```go
func TestCalcExpectedRank(t *testing.T) {
	bal := configs.GetBalanceConfig()

	// Zero uses = rank 0
	assert.Equal(t, 0.0, calcExpectedRank(0, 1.0, bal))

	// Some uses should produce a positive expected rank
	result := calcExpectedRank(500, 1.0, bal)
	assert.Greater(t, result, 0.0)

	// Higher multiplier = higher expected rank for same uses
	low := calcExpectedRank(500, 0.3, bal)
	high := calcExpectedRank(500, 1.0, bal)
	assert.Greater(t, high, low)
}

func TestCalcHealthScore(t *testing.T) {
	// Perfect health
	h := calcHealthScore(0.0, 0, 10, 0.0, 50)
	assert.InDelta(t, 1.0, h, 0.01)

	// All stalled, max clustering, max deviation
	h2 := calcHealthScore(50.0, 10, 10, 1.0, 50)
	assert.Less(t, h2, 0.1)
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/web/ -run "TestCalcExpectedRank|TestCalcHealthScore" -v`
Expected: FAIL — functions undefined

- [ ] **Step 3: Write aggregation functions**

Append to `internal/web/admin.progression.go`:

```go
// ── Aggregation Logic ───────────────────────────────────────────────────────

// calcExpectedRank simulates the progression curve to estimate what rank
// a player should be at given their use count and skill multiplier.
func calcExpectedRank(useCount int, mult float64, bal configs.Balance) float64 {
	if useCount == 0 {
		return 0
	}

	adjustedUses := float64(useCount) * mult
	virtualRank := adjustedUses / float64(bal.UsesPerRank)
	softCap := int(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}

	// Simulate progression: at each virtual rank level, accumulate the
	// probability of gaining a rank. Sum of all probabilities = expected gains.
	expected := 0.0
	for vr := 0; vr < int(virtualRank); vr++ {
		chance := characters.CalculateProgressionChance(vr, softCap)
		expected += chance
	}

	return expected
}

// calcHealthScore computes the composite health score for a skill.
func calcHealthScore(avgDeviation float64, stallCount, totalWithUses int, clusteringScore float64, softCap int) float64 {
	if totalWithUses == 0 {
		return 1.0
	}

	curveScore := 1.0 - clamp(math.Abs(avgDeviation)/float64(softCap), 0, 1)
	stallScore := 1.0 - float64(stallCount)/float64(totalWithUses)
	clusterScore := 1.0 - clusteringScore

	return (curveScore + stallScore + clusterScore) / 3.0
}

func clamp(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// buildSkillHealth aggregates skill health metrics across all players.
func buildSkillHealth(players []*users.UserRecord) map[string]skillHealthJSON {
	bal := configs.GetBalanceConfig()
	softCap := int(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}
	allSkills := skills.GetAllSkillNames()
	result := make(map[string]skillHealthJSON, len(allSkills))

	for _, sk := range allSkills {
		skName := string(sk)
		mult := skills.GetProgressionMultiplier(skName)

		var deviations []float64
		worstPlayer := ""
		worstDev := 0.0
		stallCount := 0
		dist := map[string]int{
			"unknown": 0, "novice": 0, "apprentice": 0,
			"journeyman": 0, "adept": 0, "expert": 0, "master": 0,
		}

		for _, u := range players {
			rank := u.Character.Skills[skName]
			useCount := u.Character.GetSkillUseCount(skName)
			if useCount == 0 {
				continue
			}

			// Expected curve deviation
			expected := calcExpectedRank(useCount, mult, bal)
			dev := float64(rank) - expected
			deviations = append(deviations, dev)
			if dev < worstDev || worstPlayer == "" {
				worstDev = dev
				worstPlayer = u.Character.Name
			}

			// Stall detection
			approxUsesAtRank := float64(rank) * float64(bal.UsesPerRank) / mult
			usesSinceGain := float64(useCount) - approxUsesAtRank
			if usesSinceGain < 0 {
				usesSinceGain = 0
			}
			chanceAtRank := characters.CalculateProgressionChance(rank, softCap)
			expectedUsesForNext := 0.0
			if chanceAtRank > 0 {
				expectedUsesForNext = 1.0 / chanceAtRank
			}
			if expectedUsesForNext > 0 && usesSinceGain/expectedUsesForNext > 2.0 {
				stallCount++
			}

			// Distribution
			tier := skills.GetSkillRankDescription(rank)
			dist[tier]++
		}

		totalWithUses := len(deviations)
		avgDev := 0.0
		if totalWithUses > 0 {
			sum := 0.0
			for _, d := range deviations {
				sum += d
			}
			avgDev = sum / float64(totalWithUses)
		}

		// Clustering: max count / total
		clusterScore := 0.0
		if totalWithUses > 0 {
			maxCount := 0
			for _, c := range dist {
				if c > maxCount {
					maxCount = c
				}
			}
			clusterScore = float64(maxCount) / float64(totalWithUses)
		}

		health := calcHealthScore(avgDev, stallCount, totalWithUses, clusterScore, softCap)

		result[skName] = skillHealthJSON{
			HealthScore:     math.Round(health*100) / 100,
			AvgDeviation:    math.Round(avgDev*100) / 100,
			WorstPlayer:     worstPlayer,
			WorstDeviation:  math.Round(worstDev*100) / 100,
			StallCount:      stallCount,
			TotalWithUses:   totalWithUses,
			Distribution:    dist,
			ClusteringScore: math.Round(clusterScore*100) / 100,
		}
	}

	return result
}

// buildStatHealth aggregates stat distributions across all players.
func buildStatHealth(players []*users.UserRecord) map[string]statHealthJSON {
	statNames := []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}
	result := make(map[string]statHealthJSON, len(statNames))

	for _, statName := range statNames {
		dist := map[string]int{"0-50": 0, "51-100": 0, "101-150": 0, "151+": 0}
		for _, u := range players {
			val := u.Character.GetStatValue(statName)
			switch {
			case val <= 50:
				dist["0-50"]++
			case val <= 100:
				dist["51-100"]++
			case val <= 150:
				dist["101-150"]++
			default:
				dist["151+"]++
			}
		}
		result[statName] = statHealthJSON{Distribution: dist}
	}

	return result
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/web/ -run "TestCalcExpectedRank|TestCalcHealthScore" -v`
Expected: PASS

- [ ] **Step 5: Verify full build**

Run: `go build ./...`
Expected: Clean

- [ ] **Step 6: Commit**

```bash
git add internal/web/admin.progression.go internal/web/admin_progression_test.go
git commit -m "feat: progression dashboard — skill/stat aggregation logic"
```

---

### Task 3: Discovery Health Aggregation

**Files:**
- Modify: `internal/web/admin.progression.go`

- [ ] **Step 1: Write discovery aggregation functions**

Append to `internal/web/admin.progression.go`:

```go
// ── Discovery Health ────────────────────────────────────────────────────────

// buildSpellHealth computes discovery metrics for all spells.
func buildSpellHealth(players []*users.UserRecord) map[string]spellHealthJSON {
	allSpells := spells.GetAllSpells()
	totalPlayers := len(players)
	result := make(map[string]spellHealthJSON, len(allSpells))

	// Compute activity percentiles for flagging
	activities := playerActivities(players)
	sort.Ints(activities)
	medianActivity := percentile(activities, 50)
	p10Activity := percentile(activities, 10)

	// Starter spells (learned at creation) — exclude from "too easy"
	starterSpells := map[string]bool{
		"conviction-spike": true,
		"minor-light":      true,
	}

	for spellId, spell := range allSpells {
		knownCount := 0
		totalActivity := 0.0

		for _, u := range players {
			if _, ok := u.Character.SpellBook[spellId]; ok {
				knownCount++
				totalActivity += float64(playerTotalActivity(u))
			}
		}

		avgActivity := 0.0
		if knownCount > 0 {
			avgActivity = totalActivity / float64(knownCount)
		}

		flag := ""
		if knownCount == 0 && totalPlayers > 0 {
			// Check if any high-activity player exists who should have found it
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
			// Check if low-activity players have it
			for _, u := range players {
				if _, ok := u.Character.SpellBook[spellId]; ok {
					if playerTotalActivity(u) < p10Activity {
						flag = "too_easy"
						break
					}
				}
			}
		}

		school := ""
		if len(spell.Schools) > 0 {
			school = spell.Schools[0]
		}

		result[spellId] = spellHealthJSON{
			Name:                   spell.Name,
			School:                 school,
			KnownCount:             knownCount,
			TotalPlayers:           totalPlayers,
			AvgActivityAtDiscovery: math.Round(avgActivity),
			Flag:                   flag,
		}
	}

	return result
}

// buildRecipeHealth computes discovery metrics for all recipes.
func buildRecipeHealth(players []*users.UserRecord) map[string]recipeHealthJSON {
	allRecipes := crafting.GetAll()
	totalPlayers := len(players)
	result := make(map[string]recipeHealthJSON, len(allRecipes))

	activities := playerActivities(players)
	sort.Ints(activities)
	medianActivity := percentile(activities, 50)
	p10Activity := percentile(activities, 10)

	// Starter recipes have SkillMinimum == 0
	starterRecipes := crafting.GetStarterRecipes()

	for recipeId, recipe := range allRecipes {
		knownCount := 0
		totalActivity := 0.0

		for _, u := range players {
			if _, ok := u.Character.KnownRecipes[recipeId]; ok {
				knownCount++
				totalActivity += float64(playerTotalActivity(u))
			}
		}

		avgActivity := 0.0
		if knownCount > 0 {
			avgActivity = totalActivity / float64(knownCount)
		}

		flag := ""
		isStarter := starterRecipes[recipeId] > 0
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
			for _, u := range players {
				if _, ok := u.Character.KnownRecipes[recipeId]; ok {
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
			AvgActivityAtDiscovery: math.Round(avgActivity),
			Flag:                   flag,
		}
	}

	return result
}

// ── Helpers ─────────────────────────────────────────────────────────────────

func playerTotalActivity(u *users.UserRecord) int {
	total := 0
	for _, v := range u.Character.SkillUseCount {
		total += v
	}
	for _, v := range u.Character.StatUseCount {
		total += v
	}
	return total
}

func playerActivities(players []*users.UserRecord) []int {
	result := make([]int, 0, len(players))
	for _, u := range players {
		result = append(result, playerTotalActivity(u))
	}
	return result
}

func percentile(sorted []int, pct int) int {
	if len(sorted) == 0 {
		return 0
	}
	idx := len(sorted) * pct / 100
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean

- [ ] **Step 3: Commit**

```bash
git add internal/web/admin.progression.go
git commit -m "feat: progression dashboard — discovery health aggregation"
```

---

### Task 4: Player Overview Builder and API Handler

**Files:**
- Modify: `internal/web/admin.progression.go`

- [ ] **Step 1: Write player overview builder and HTTP handlers**

Append to `internal/web/admin.progression.go`:

```go
// ── Player Overview ─────────────────────────────────────────────────────────

func buildPlayerOverview(players []*users.UserRecord) []playerJSON {
	bal := configs.GetBalanceConfig()
	softCap := int(bal.SkillSoftCap)
	if softCap <= 0 {
		softCap = 50
	}
	allSkills := skills.GetAllSkillNames()
	totalSpells := len(spells.GetAllSpells())
	totalRecipes := len(crafting.GetAll())

	result := make([]playerJSON, 0, len(players))

	for _, u := range players {
		pj := playerJSON{
			Name:          u.Character.Name,
			TotalActivity: playerTotalActivity(u),
			Skills:        make(map[string]playerSkillJSON, len(allSkills)),
			Stats:         make(map[string]playerStatJSON),
			SpellsKnown:  len(u.Character.SpellBook),
			SpellsTotal:  totalSpells,
			RecipesKnown: len(u.Character.KnownRecipes),
			RecipesTotal: totalRecipes,
		}

		for _, sk := range allSkills {
			skName := string(sk)
			rank := u.Character.Skills[skName]
			useCount := u.Character.GetSkillUseCount(skName)
			mult := skills.GetProgressionMultiplier(skName)
			adjustedUses := float64(useCount) * mult
			virtualRank := adjustedUses / float64(bal.UsesPerRank)
			chance := characters.CalculateProgressionChance(int(virtualRank), softCap)

			pj.Skills[skName] = playerSkillJSON{
				Rank:             rank,
				Tier:             skills.GetSkillRankDescription(rank),
				UseCount:         useCount,
				VirtualRank:      math.Round(virtualRank*100) / 100,
				ProgressionChance: math.Round(chance*10000) / 100, // as percentage
			}
		}

		statNames := []string{"strength", "dexterity", "perception", "vitality", "willpower", "charisma"}
		for _, statName := range statNames {
			pj.Stats[statName] = playerStatJSON{
				Value:    u.Character.GetStatValue(statName),
				Training: u.Character.GetStatTraining(statName),
				UseCount: u.Character.GetStatUseCount(statName),
			}
		}

		result = append(result, pj)
	}

	// Sort by total activity descending
	sort.Slice(result, func(i, j int) bool {
		return result[i].TotalActivity > result[j].TotalActivity
	})

	return result
}

// ── HTTP Handlers ───────────────────────────────────────────────────────────

func progressionIndex(w http.ResponseWriter, r *http.Request) {
	tmpl := prepAdminTemplate(
		readAdminFile(`_header.html`),
		readAdminFile(`progression/index.html`),
		readAdminFile(`_footer.html`),
	)
	if tmpl == nil {
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	tmpl.Execute(w, nil)
}

func progressionAPI(w http.ResponseWriter, r *http.Request) {
	players := collectPlayers()

	resp := progressionAPIResponse{
		Skills:  buildSkillHealth(players),
		Stats:   buildStatHealth(players),
		Spells:  buildSpellHealth(players),
		Recipes: buildRecipeHealth(players),
		Players: buildPlayerOverview(players),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean (may need to check `GetStatValue`, `GetStatTraining`, `GetStatUseCount` method signatures exist — they are on `Character` in `progression.go`)

- [ ] **Step 3: Commit**

```bash
git add internal/web/admin.progression.go
git commit -m "feat: progression dashboard — player overview builder and API handlers"
```

---

### Task 5: Route Registration

**Files:**
- Modify: `internal/web/web.go`

- [ ] **Step 1: Find the combat stats route block and add progression routes**

In `internal/web/web.go`, find the block starting with `// Combat Stats Admin` (around line 325) and add after it:

```go
// Progression Admin
http.HandleFunc("GET /admin/progression/", RunWithMUDLocked(
	doBasicAuth(progressionIndex),
))
http.HandleFunc("GET /admin/api/progression/", RunWithMUDLocked(
	doBasicAuth(progressionAPI),
))
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: Clean

- [ ] **Step 3: Commit**

```bash
git add internal/web/web.go
git commit -m "feat: progression dashboard — register admin routes"
```

---

### Task 6: HTML Template — System Health Panels

**Files:**
- Create: `_datafiles/html/admin/progression/index.html`

- [ ] **Step 1: Create the HTML template**

Create `_datafiles/html/admin/progression/index.html`:

```html
{{template "header" .}}

<script src="/admin/static/js/chart.js"></script>

<div class="container-fluid mt-3">
  <h2>Progression Health</h2>

  <!-- Sub-view tabs -->
  <ul class="nav nav-pills mb-3" id="progTabs">
    <li class="nav-item">
      <a class="nav-link active" href="#" onclick="showTab('health')">System Health</a>
    </li>
    <li class="nav-item">
      <a class="nav-link" href="#" onclick="showTab('players')">Player Overview</a>
    </li>
  </ul>

  <!-- ═══ SYSTEM HEALTH ═══ -->
  <div id="tab-health">

    <!-- Panel 1: Skill Health Table -->
    <div class="card mb-3">
      <div class="card-header" data-toggle="collapse" data-target="#skillHealthBody" style="cursor:pointer">
        <strong>Skill Health Scores</strong>
      </div>
      <div class="collapse show" id="skillHealthBody">
        <div class="card-body">
          <table class="table table-sm table-striped" id="skillHealthTable">
            <thead>
              <tr>
                <th>Skill</th>
                <th>Health</th>
                <th>Avg Deviation</th>
                <th>Worst Player</th>
                <th>Stalls</th>
                <th>Players</th>
                <th>Clustering</th>
              </tr>
            </thead>
            <tbody></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Panel 2: Stall Detection -->
    <div class="card mb-3">
      <div class="card-header" data-toggle="collapse" data-target="#stallBody" style="cursor:pointer">
        <strong>Stall Detection</strong>
      </div>
      <div class="collapse" id="stallBody">
        <div class="card-body">
          <div class="alert alert-info" id="noStalls" style="display:none">No stalled progressions detected.</div>
          <table class="table table-sm table-striped" id="stallTable">
            <thead>
              <tr>
                <th>Player</th>
                <th>Skill</th>
                <th>Rank</th>
                <th>Uses Since Gain</th>
                <th>Expected Uses</th>
                <th>Staleness</th>
              </tr>
            </thead>
            <tbody></tbody>
          </table>
        </div>
      </div>
    </div>

    <!-- Panel 3: Distribution Charts -->
    <div class="card mb-3">
      <div class="card-header" data-toggle="collapse" data-target="#distBody" style="cursor:pointer">
        <strong>Population Distribution</strong>
      </div>
      <div class="collapse" id="distBody">
        <div class="card-body">
          <div class="row mb-2">
            <div class="col-md-3">
              <select class="form-control form-control-sm" id="distSkillSelect" onchange="updateDistChart()"></select>
            </div>
          </div>
          <div class="row">
            <div class="col-md-6"><canvas id="skillDistChart" height="250"></canvas></div>
            <div class="col-md-6"><canvas id="statDistChart" height="250"></canvas></div>
          </div>
        </div>
      </div>
    </div>

    <!-- Panel 4: Discovery Health -->
    <div class="card mb-3">
      <div class="card-header" data-toggle="collapse" data-target="#discoveryBody" style="cursor:pointer">
        <strong>Discovery Health</strong>
      </div>
      <div class="collapse" id="discoveryBody">
        <div class="card-body">
          <h5>Spells</h5>
          <table class="table table-sm table-striped" id="spellTable">
            <thead>
              <tr><th>Spell</th><th>School</th><th>Known</th><th>Total</th><th>Avg Activity</th><th>Flag</th></tr>
            </thead>
            <tbody></tbody>
          </table>
          <h5 class="mt-3">Recipes</h5>
          <table class="table table-sm table-striped" id="recipeTable">
            <thead>
              <tr><th>Recipe</th><th>Skill</th><th>Known</th><th>Total</th><th>Avg Activity</th><th>Flag</th></tr>
            </thead>
            <tbody></tbody>
          </table>
        </div>
      </div>
    </div>

  </div>

  <!-- ═══ PLAYER OVERVIEW ═══ -->
  <div id="tab-players" style="display:none">
    <div class="card">
      <div class="card-body">
        <table class="table table-sm table-striped" id="playerTable">
          <thead>
            <tr>
              <th>Player</th>
              <th>Activity</th>
              <th id="skillHeaders"></th>
            </tr>
          </thead>
          <tbody></tbody>
        </table>
      </div>
    </div>
  </div>

</div>

<script>
(function() {
  var data = null;
  var skillDistChart = null;
  var statDistChart = null;

  var tierColors = {
    'unknown':'#6c757d','novice':'#6c757d','apprentice':'#28a745',
    'journeyman':'#007bff','adept':'#6f42c1','expert':'#fd7e14','master':'#dc3545'
  };

  function healthBadge(score) {
    var cls = score >= 0.7 ? 'success' : score >= 0.4 ? 'warning' : 'danger';
    return '<span class="badge badge-' + cls + '">' + score.toFixed(2) + '</span>';
  }

  function tierBadge(tier) {
    var color = tierColors[tier] || '#6c757d';
    return '<span class="badge" style="background:'+color+';color:#fff">'+tier+'</span>';
  }

  function flagBadge(flag) {
    if (!flag) return '';
    var cls = flag === 'too_hidden' ? 'warning' : 'info';
    return '<span class="badge badge-' + cls + '">' + flag.replace('_',' ') + '</span>';
  }

  function showTab(name) {
    document.getElementById('tab-health').style.display = name === 'health' ? '' : 'none';
    document.getElementById('tab-players').style.display = name === 'players' ? '' : 'none';
    document.querySelectorAll('#progTabs .nav-link').forEach(function(el) {
      el.classList.remove('active');
    });
    event.target.classList.add('active');
  }
  window.showTab = showTab;

  function updateSkillHealthTable(d) {
    var tbody = document.querySelector('#skillHealthTable tbody');
    tbody.innerHTML = '';
    var entries = Object.entries(d.skills).sort(function(a,b) { return a[1].health_score - b[1].health_score; });
    entries.forEach(function(e) {
      var sk = e[0], s = e[1];
      var row = '<tr><td>'+sk+'</td><td>'+healthBadge(s.health_score)+'</td>';
      row += '<td>'+s.avg_deviation.toFixed(1)+'</td>';
      row += '<td>'+s.worst_player+' ('+s.worst_deviation.toFixed(1)+')</td>';
      row += '<td>'+s.stall_count+'</td><td>'+s.total_with_uses+'</td>';
      row += '<td>'+s.clustering_score.toFixed(2)+'</td></tr>';
      tbody.innerHTML += row;
    });
  }

  function updateStallTable(d) {
    var tbody = document.querySelector('#stallTable tbody');
    tbody.innerHTML = '';
    var stalls = [];
    // Rebuild stall list from player data
    d.players.forEach(function(p) {
      Object.entries(p.skills).forEach(function(e) {
        var sk = e[0], s = e[1];
        if (s.use_count > 0 && s.virtual_rank > 0) {
          var expected = 1.0 / (s.progression_chance / 100 || 0.001);
          var usesAtRank = s.rank * expected;
          var usesSince = s.use_count - usesAtRank;
          if (usesSince < 0) usesSince = 0;
          var staleness = expected > 0 ? usesSince / expected : 0;
          if (staleness > 1.5) {
            stalls.push({player:p.name, skill:sk, rank:s.rank, tier:s.tier,
                        usesSince:Math.round(usesSince), expected:Math.round(expected),
                        staleness:staleness});
          }
        }
      });
    });
    stalls.sort(function(a,b) { return b.staleness - a.staleness; });
    if (stalls.length === 0) {
      document.getElementById('noStalls').style.display = '';
      return;
    }
    document.getElementById('noStalls').style.display = 'none';
    stalls.forEach(function(s) {
      var cls = s.staleness > 2.0 ? 'danger' : s.staleness > 1.5 ? 'warning' : 'success';
      var row = '<tr><td>'+s.player+'</td><td>'+s.skill+'</td>';
      row += '<td>'+tierBadge(s.tier)+'</td>';
      row += '<td>'+s.usesSince+'</td><td>'+s.expected+'</td>';
      row += '<td><span class="badge badge-'+cls+'">'+s.staleness.toFixed(1)+'x</span></td></tr>';
      tbody.innerHTML += row;
    });
  }

  function updateDistChart() {
    var sel = document.getElementById('distSkillSelect');
    var sk = sel.value;
    if (!data || !data.skills[sk]) return;
    var dist = data.skills[sk].distribution;
    var labels = ['novice','apprentice','journeyman','adept','expert','master'];
    var values = labels.map(function(l) { return dist[l] || 0; });
    var colors = labels.map(function(l) { return tierColors[l]; });

    if (skillDistChart) skillDistChart.destroy();
    skillDistChart = new Chart(document.getElementById('skillDistChart'), {
      type: 'bar',
      data: { labels: labels, datasets: [{label: sk, data: values, backgroundColor: colors}] },
      options: { scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } }, plugins: { legend: { display: false } } }
    });
  }
  window.updateDistChart = updateDistChart;

  function updateStatDistChart(d) {
    var labels = ['0-50','51-100','101-150','151+'];
    var datasets = [];
    var statColors = {strength:'#dc3545',dexterity:'#28a745',perception:'#fd7e14',
                      vitality:'#6f42c1',willpower:'#007bff',charisma:'#ffc107'};
    Object.entries(d.stats).forEach(function(e) {
      datasets.push({
        label: e[0],
        data: labels.map(function(l) { return e[1].distribution[l] || 0; }),
        backgroundColor: statColors[e[0]] || '#6c757d'
      });
    });
    if (statDistChart) statDistChart.destroy();
    statDistChart = new Chart(document.getElementById('statDistChart'), {
      type: 'bar',
      data: { labels: labels, datasets: datasets },
      options: { scales: { y: { beginAtZero: true, ticks: { stepSize: 1 } } } }
    });
  }

  function updateDiscoveryTables(d) {
    var spellBody = document.querySelector('#spellTable tbody');
    spellBody.innerHTML = '';
    Object.entries(d.spells).sort(function(a,b) { return a[1].name.localeCompare(b[1].name); }).forEach(function(e) {
      var s = e[1];
      spellBody.innerHTML += '<tr><td>'+s.name+'</td><td>'+s.school+'</td><td>'+s.known_count+'</td><td>'+s.total_players+'</td><td>'+Math.round(s.avg_activity_at_discovery)+'</td><td>'+flagBadge(s.flag)+'</td></tr>';
    });

    var recipeBody = document.querySelector('#recipeTable tbody');
    recipeBody.innerHTML = '';
    Object.entries(d.recipes).sort(function(a,b) { return a[1].name.localeCompare(b[1].name); }).forEach(function(e) {
      var s = e[1];
      recipeBody.innerHTML += '<tr><td>'+s.name+'</td><td>'+s.skill+'</td><td>'+s.known_count+'</td><td>'+s.total_players+'</td><td>'+Math.round(s.avg_activity_at_discovery)+'</td><td>'+flagBadge(s.flag)+'</td></tr>';
    });
  }

  function updatePlayerTable(d) {
    var thead = document.getElementById('skillHeaders');
    var allSkills = Object.keys(d.skills).sort();
    thead.outerHTML = allSkills.map(function(sk) { return '<th>'+sk.replace('-combat','')+'</th>'; }).join('');

    var tbody = document.querySelector('#playerTable tbody');
    tbody.innerHTML = '';
    d.players.forEach(function(p) {
      var row = '<tr><td>'+p.name+'</td><td>'+p.total_activity+'</td>';
      allSkills.forEach(function(sk) {
        var s = p.skills[sk];
        row += '<td>'+ (s ? tierBadge(s.tier) : '-') +'</td>';
      });
      row += '</tr>';
      tbody.innerHTML += row;
    });
  }

  function populateSkillDropdown(d) {
    var sel = document.getElementById('distSkillSelect');
    sel.innerHTML = '';
    Object.keys(d.skills).sort().forEach(function(sk) {
      var opt = document.createElement('option');
      opt.value = sk; opt.textContent = sk;
      sel.appendChild(opt);
    });
  }

  function fetchData() {
    fetch('/admin/api/progression/')
      .then(function(r) { return r.json(); })
      .then(function(d) {
        data = d;
        updateSkillHealthTable(d);
        updateStallTable(d);
        populateSkillDropdown(d);
        updateDistChart();
        updateStatDistChart(d);
        updateDiscoveryTables(d);
        updatePlayerTable(d);
      });
  }

  fetchData();
  setInterval(fetchData, 30000);
})();
</script>

{{template "footer" .}}
```

- [ ] **Step 2: Verify server starts and page loads**

Run: `go build ./...` then start the server and navigate to `/admin/progression/`
Expected: Page loads with tables and charts populated from live data

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/admin/progression/index.html
git commit -m "feat: progression dashboard — HTML template with all panels"
```

---

### Task 7: Add Tab Link to Admin Navigation

**Files:**
- Modify: `_datafiles/html/admin/_header.html`

- [ ] **Step 1: Find the admin navigation and add a Progression tab link**

In `_datafiles/html/admin/_header.html`, find the nav bar that contains links to other admin tabs (Dashboard, Items, Mobs, Combat Stats, etc.) and add:

```html
<a class="nav-link" href="/admin/progression/">Progression</a>
```

Place it after the Combat Stats link.

- [ ] **Step 2: Verify the tab appears in the admin nav**

Start server, navigate to any admin page. The "Progression" tab should appear in the navigation.

- [ ] **Step 3: Commit**

```bash
git add _datafiles/html/admin/_header.html
git commit -m "feat: progression dashboard — add nav tab link"
```

---

### Task 8: Final Integration Test

- [ ] **Step 1: Run full test suite**

Run: `go test ./... -count=1`
Expected: All tests pass

- [ ] **Step 2: Run full build**

Run: `go build ./...`
Expected: Clean

- [ ] **Step 3: Manual verification checklist**

Start the server and check:
- [ ] Progression tab appears in admin nav
- [ ] System Health: Skill health table populates with colored health badges
- [ ] System Health: Stall detection shows flagged entries or "no stalls" message
- [ ] System Health: Skill distribution chart renders with dropdown
- [ ] System Health: Stat distribution chart renders with grouped bars
- [ ] System Health: Discovery tables show spells and recipes with flags
- [ ] Player Overview: Table shows all players with tier badges
- [ ] Auto-refresh works (data updates every 30 seconds)

- [ ] **Step 4: Final commit**

```bash
git add -A
git commit -m "feat: progression dashboard — complete admin tab"
```
