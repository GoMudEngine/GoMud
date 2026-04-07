# Progression Dashboard — Admin Tab Design Spec

## Goal

Add a "Progression" tab to the admin web dashboard that provides an
aggregated, system-level view of whether the use-based progression
system is healthy. The primary audience is the game admin — not players.

## Architecture

Server-computed JSON API + HTMX tab, following the existing Combat Stats
pattern (`admin.combatstats.go`). All aggregation, health scoring, and
expected-curve math runs in Go. The frontend uses HTMX to load sections
and Chart.js for visualizations.

**Tech stack:** Go backend, HTMX, Bootstrap 4, Chart.js (all already in
use by the admin dashboard).

---

## Data Sources

All data is already available on live characters:

| Source | Access |
|--------|--------|
| Skill ranks | `character.Skills map[string]int` |
| Skill use counts | `character.SkillUseCount map[string]int` |
| Stat values | `character.Stats.<Stat>.Value` / `.Training` |
| Stat use counts | `character.StatUseCount map[string]int` |
| Spells known | `character.SpellBook map[string]int` |
| Recipes known | `character.KnownRecipes map[string]int` |
| All spells | `spells.GetAllSpells()` |
| All recipes | `crafting.GetAll()` |
| All online users | `users.GetAllActiveUsers()` |
| Progression config | `configs.GetBalanceConfig()` — soft caps, decay rates, multipliers |
| Progression math | `characters/progression.go` — CheckStatProgression, CheckSkillProgression |

**Player activity proxy:** Total uses across all skills + stats
(`sum(SkillUseCount) + sum(StatUseCount)`). This measures how much the
player has actually done, not wall-clock time.

---

## Tab Structure

The Progression tab has two sub-views, switchable via Bootstrap nav-pills:

### Sub-view 1: System Health (default/primary)

Four panels, each collapsible:

#### Panel 1: Expected Curve Comparison

For each skill, compute the expected rank given each player's use count
using the actual progression formula parameters, then aggregate.

**Computation:** Given a player's `SkillUseCount[skill]` and the config
values (`UsesPerRank`, `BaseProgressionChance`, `SkillProgressionDecay`,
`SkillSoftCap`, and the per-skill progression multiplier), calculate the
expected rank by simulating: `virtualRank = useCount * mult / UsesPerRank`,
then integrate the exponential decay chance curve to get expected
successful progressions. Compare against actual `Skills[skill]`.

**Display:** Table with columns:
- Skill name
- Avg deviation (actual - expected, across all players with >0 uses)
- Worst deviation (most behind player name + deviation)
- Health score badge (green/yellow/red)

**Chart:** Scatter plot per skill (selectable dropdown): X = total uses,
Y = rank. Overlay the expected curve line. Each dot is a player.

#### Panel 2: Stall Detection

Flag player+skill combinations where the player has accumulated
significantly more uses than expected for their next rank-up.

**Computation:** For each player+skill with rank < soft cap:
- `usesSinceLastGain = SkillUseCount[skill] - usesAtCurrentRank`
  (approximate: `currentRank * UsesPerRank / mult`)
- `expectedUsesForNext` = inverse of progression chance at current rank
- `stalenessRatio = usesSinceLastGain / expectedUsesForNext`
- Flag if stalenessRatio > 2.0

**Display:** Table with columns:
- Player name
- Skill name
- Current rank (tier badge)
- Uses since last gain
- Expected uses for next rank
- Staleness ratio (colored: green < 1.5, yellow 1.5-2.0, red > 2.0)

Sortable by staleness ratio (worst first). Filterable by skill.

#### Panel 3: Population Distribution

For each skill, show the rank distribution across all active players.

**Display:** Per-skill histogram (Chart.js bar chart): X = rank tier
(novice, apprentice, journeyman, adept, expert, master), Y = player
count. Skill selectable via dropdown.

**Clustering score:** For each skill, compute
`max_tier_count / total_players`. If > 0.70, flag the skill as
"clustered" — most players are stuck at the same tier.

Also display stat distributions: per-stat histogram with buckets
(0-50, 51-100, 101-150, 151+).

#### Panel 4: Discovery Health

Two sub-tables: Spells and Recipes.

**Spells table columns:**
- Spell name
- School / primary stat
- Known count (how many players have learned it)
- Total players (for percentage)
- Avg activity at discovery (average total activity of players who know it)
- Flag: `too_hidden` if no player with activity > median has it;
  `too_easy` if players with activity < 10th percentile have it
  (excluding starter spells)

**Recipes table columns:**
- Recipe name
- Skill requirement
- Known count
- Total players
- Avg activity at discovery
- Flag: same logic as spells (excluding starter recipes with
  `skill_minimum == 0`)

Filterable by skill, by flag status (all / undiscovered / flagged).

### Sub-view 2: Player Overview

Sortable table of all active players.

**Columns:**
- Player name
- Total activity (sum of all use counts)
- Each skill as a colored tier badge (novice=gray, apprentice=green,
  journeyman=blue, adept=purple, expert=orange, master=red)

**Expandable rows:** Click a player row to expand inline detail:
- Per-skill: rank, use count, virtual rank, progression chance %
- Per-stat: value, training, use count
- Spells known count / total
- Recipes known count / total

---

## API Design

### Endpoints

**`GET /admin/progression`** — serves the HTML template via HTMX.

**`GET /api/admin/progression`** — returns JSON with all aggregated data.

### JSON Response Shape

```json
{
  "skills": {
    "weapon-combat": {
      "health_score": 0.82,
      "avg_deviation": -1.2,
      "worst_player": "quester9",
      "worst_deviation": -4,
      "stall_count": 2,
      "total_with_uses": 6,
      "distribution": {
        "novice": 1,
        "apprentice": 3,
        "journeyman": 2,
        "adept": 0,
        "expert": 0,
        "master": 0
      },
      "clustering_score": 0.50
    }
  },
  "stats": {
    "strength": {
      "distribution": {
        "0-50": 0,
        "51-100": 3,
        "101-150": 5,
        "151+": 1
      }
    }
  },
  "spells": {
    "chrysalis-haste": {
      "name": "Chrysalis Haste",
      "school": "enhancement",
      "known_count": 3,
      "total_players": 8,
      "avg_activity_at_discovery": 1240,
      "flag": ""
    }
  },
  "recipes": {
    "healing-salve": {
      "name": "Healing Salve",
      "skill": "alchemy",
      "known_count": 5,
      "total_players": 8,
      "avg_activity_at_discovery": 320,
      "flag": ""
    }
  },
  "players": [
    {
      "name": "quester9",
      "total_activity": 4520,
      "skills": {
        "weapon-combat": {
          "rank": 12,
          "tier": "journeyman",
          "use_count": 1850,
          "virtual_rank": 8.2,
          "progression_chance": 0.18
        }
      },
      "stats": {
        "strength": {
          "value": 112,
          "training": 12,
          "use_count": 940
        }
      },
      "spells_known": 5,
      "spells_total": 18,
      "recipes_known": 12,
      "recipes_total": 30
    }
  ]
}
```

---

## Health Score Formula

Composite score per skill, 0.0 to 1.0:

```
curve_score    = 1.0 - clamp(abs(avg_deviation) / soft_cap, 0, 1)
stall_score    = 1.0 - (stall_count / total_with_uses)
cluster_score  = 1.0 - clustering_score

health_score   = (curve_score + stall_score + cluster_score) / 3.0
```

**Thresholds:** green >= 0.7, yellow >= 0.4, red < 0.4.

If `total_with_uses == 0` for a skill, health score is `1.0` (no data,
no problem).

---

## Files to Create / Modify

| Action | Path | Purpose |
|--------|------|---------|
| Create | `internal/web/admin.progression.go` | Handler + JSON API + aggregation logic |
| Create | `_datafiles/html/admin/admin-progression.html` | HTMX template with tables + charts |
| Modify | `internal/web/admin.go` (or equivalent) | Register new tab + routes |

---

## Player Loading Strategy

The dashboard includes both online and recently-active offline players.

**Loading:** A new helper function scans user YAML files on disk
(`_datafiles/world/dogmud/users/{userId}.yaml`). For each file:
1. Check file modification time — skip if older than 14 days (proxy for
   last active, since users are saved on logout and periodically).
2. Load and unmarshal the YAML.
3. Filter by `Role == "user"` — exclude admins and guests.

Online users (`GetAllActiveUsers()`) are merged with the loaded offline
users, deduplicating by UserId. This runs on each API call (admin-only,
infrequent — no caching needed for V1).

**Fields available for filtering:**
- `UserRecord.Role` — `"guest"` / `"user"` / `"admin"`
- File mod time — proxy for last login/activity
- No explicit `LastLogin` field exists on UserRecord

---

## Constraints

- All computation server-side in Go. No progression math in JavaScript.
- Follow existing admin dashboard patterns (HTMX load, Bootstrap 4 tables,
  Chart.js for charts).
- Player activity = sum of all use counts (not wall-clock time).
- Starter spells/recipes (those granted at character creation) are excluded
  from "too easy" flagging.
- Only `Role == "user"` characters are included (no admins/guests).
- Only characters active within the last 14 days (file mod time) are
  included, plus all currently online users.
