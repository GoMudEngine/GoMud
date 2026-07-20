package web

import (
	"encoding/json"
	"net/http"
	"text/template"

	"github.com/GoMudEngine/GoMud/internal/combat"
	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

// combatStatsAPIResponse is the JSON shape returned by the API endpoint.
type combatStatsAPIResponse struct {
	TotalEvents      int     `json:"totalEvents"`
	Hits             int     `json:"hits"`
	Misses           int     `json:"misses"`
	Crits            int     `json:"crits"`
	Fumbles          int     `json:"fumbles"`
	Backfires        int     `json:"backfires"`
	Fizzles          int     `json:"fizzles"`
	TotalDamage      int     `json:"totalDamage"`
	AvgDamagePerHit  float64 `json:"avgDamagePerHit"`
	AvgAttackZScore  float64 `json:"avgAttackZScore"`
	AvgDefenseZScore float64 `json:"avgDefenseZScore"`

	ByAttackType map[string]attackTypeJSON `json:"byAttackType"`

	Defense  defenseJSON `json:"defense"`
	Matchups matchupJSON `json:"matchups"`

	PositionHitRates positionHitRatesJSON `json:"positionHitRates"`
	GrappleHitRates  grappleHitRatesJSON  `json:"grappleHitRates"`

	EarliestRound uint64 `json:"earliestRound"`
	LatestRound   uint64 `json:"latestRound"`
}

type attackTypeJSON struct {
	Events      int `json:"events"`
	Hits        int `json:"hits"`
	Crits       int `json:"crits"`
	TotalDamage int `json:"totalDamage"`
}

type defenseJSON struct {
	Dodge int `json:"dodge"`
	Parry int `json:"parry"`
	Block int `json:"block"`
}

type matchupJSON struct {
	PvM int `json:"pvm"`
	MvP int `json:"mvp"`
	PvP int `json:"pvp"`
	MvM int `json:"mvm"`
}

type positionHitRatesJSON struct {
	Standing float64 `json:"standing"`
	Prone    float64 `json:"prone"`
	Clinched float64 `json:"clinched"`
	Grounded float64 `json:"grounded"`
}

type grappleHitRatesJSON struct {
	Controller    float64 `json:"controller"`
	NonController float64 `json:"nonController"`
}

func combatStatsIndex(w http.ResponseWriter, r *http.Request) {

	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/combatstats/index.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
	}

	tplData := struct{}{}
	if err := tmpl.Execute(w, tplData); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}

func combatStatsResetAPI(w http.ResponseWriter, r *http.Request) {
	cleared := combat.ResetBuffer()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"cleared": cleared})
}

func combatStatsExportAPI(w http.ResponseWriter, r *http.Request) {
	count := combat.GetBufferLen()
	combat.ExportNow()
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]int{"flushed": count})
}

func combatStatsAPI(w http.ResponseWriter, r *http.Request) {

	filters := combat.FilterParams{
		SourceType: r.URL.Query().Get("source"),
		TargetType: r.URL.Query().Get("target"),
		Channel:    r.URL.Query().Get("channel"),
	}

	summary := combat.GetFilteredSummary(filters)

	resp := combatStatsAPIResponse{
		TotalEvents:      summary.TotalEvents,
		Hits:             summary.Hits,
		Misses:           summary.Misses,
		Crits:            summary.Crits,
		Fumbles:          summary.Fumbles,
		Backfires:        summary.Backfires,
		Fizzles:          summary.Fizzles,
		TotalDamage:      summary.TotalDamage,
		AvgAttackZScore:  summary.AvgAttackZScore,
		AvgDefenseZScore: summary.AvgDefenseZScore,
		EarliestRound:    summary.EarliestRound,
		LatestRound:      summary.LatestRound,
	}

	if summary.Hits > 0 {
		resp.AvgDamagePerHit = float64(summary.TotalDamage) / float64(summary.Hits)
	}

	resp.ByAttackType = make(map[string]attackTypeJSON, len(summary.ByAttackType))
	for name, stats := range summary.ByAttackType {
		resp.ByAttackType[name] = attackTypeJSON{
			Events:      stats.Events,
			Hits:        stats.Hits,
			Crits:       stats.Crits,
			TotalDamage: stats.TotalDamage,
		}
	}

	resp.Defense = defenseJSON{
		Dodge: summary.DodgeSuccesses,
		Parry: summary.ParrySuccesses,
		Block: summary.BlockSuccesses,
	}

	resp.Matchups = matchupJSON{
		PvM: summary.PvMEvents,
		MvP: summary.MvPEvents,
		PvP: summary.PvPEvents,
		MvM: summary.MvMEvents,
	}

	resp.PositionHitRates = positionHitRatesJSON{
		Standing: summary.HitRateTargetStanding,
		Prone:    summary.HitRateTargetProne,
		Clinched: summary.HitRateTargetClinched,
		Grounded: summary.HitRateTargetGrounded,
	}

	resp.GrappleHitRates = grappleHitRatesJSON{
		Controller:    summary.HitRateGrappleController,
		NonController: summary.HitRateNonGrappleController,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}
