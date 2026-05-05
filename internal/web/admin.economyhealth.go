package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"text/template"
	"time"

	"github.com/GoMudEngine/GoMud/internal/configs"
	"github.com/GoMudEngine/GoMud/internal/economy/health"
	"github.com/GoMudEngine/GoMud/internal/mudlog"
)

func economyIndex(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.New("index.html").Funcs(funcMap).ParseFiles(
		configs.GetFilePathsConfig().AdminHtml.String()+"/_header.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/economy/index.html",
		configs.GetFilePathsConfig().AdminHtml.String()+"/_footer.html",
	)
	if err != nil {
		mudlog.Error("HTML Template", "error", err)
		http.Error(w, "template error", http.StatusInternalServerError)
		return
	}
	if err := tmpl.Execute(w, struct{}{}); err != nil {
		mudlog.Error("HTML Execute", "error", err)
	}
}

// economyAPIResponse is the JSON shape the dashboard fetches.
type economyAPIResponse struct {
	Snapshot health.Snapshot     `json:"snapshot"`
	Scores   health.Scores       `json:"scores"`
	Deltas   map[string]deltaSet `json:"deltas"` // keyed by "1h", "6h", "1d", "3d", "1w"
}

type deltaSet struct {
	UnixTs   int64                          `json:"unix_ts,omitempty"`
	Shops    map[string]health.ShopDelta    `json:"shops"`     // key "{zone}/{mobId}/{roomId}"
	Foragers map[string]health.ForagerDelta `json:"foragers"`  // key is mobId as string
	Caravans map[string]health.CaravanDelta `json:"caravans"`  // key is instId as string
}

func economyAPI(w http.ResponseWriter, r *http.Request) {
	// Live capture is taken off-tick from the HTTP goroutine; field
	// reads on cached *ShopInventory entries are eventually-consistent
	// with concurrent buy/sell mutations on the world tick. Acceptable
	// for a debug/observability dashboard at hourly+ refresh cadence.
	cur := health.CaptureSnapshot()
	metas := health.ListSnapshots()

	// metas (full list) feeds the delta picker; historyMetas is the
	// trend-scoring window (last 168 entries = 7 days at 1h cadence).
	historyMetas := metas
	if len(historyMetas) > 168 {
		historyMetas = historyMetas[:168]
	}
	history := make([]*health.Snapshot, 0, len(historyMetas))
	for i := len(historyMetas) - 1; i >= 0; i-- { // oldest first
		s, err := health.LoadSnapshot(historyMetas[i].UnixTs)
		if err != nil || s == nil {
			continue
		}
		history = append(history, s)
	}

	scores := health.Score(&cur, history)

	// Deltas at five offsets.
	now := time.Now().Unix()
	deltas := map[string]deltaSet{}
	offsets := map[string]int64{
		"1h": 3600,
		"6h": 6 * 3600,
		"1d": 24 * 3600,
		"3d": 3 * 24 * 3600,
		"1w": 7 * 24 * 3600,
	}
	for label, off := range offsets {
		target := now - off
		// Tolerance: ±50% of the offset, so the picker still works for sparse history.
		tolerance := max(off/2, 1800)
		meta := health.PickClosestSnapshot(metas, target, tolerance)
		ds := deltaSet{
			Shops:    map[string]health.ShopDelta{},
			Foragers: map[string]health.ForagerDelta{},
			Caravans: map[string]health.CaravanDelta{},
		}
		if meta != nil {
			ds.UnixTs = meta.UnixTs
			old, err := health.LoadSnapshot(meta.UnixTs)
			if err == nil && old != nil {
				for _, s := range cur.Shops {
					oldShop := health.FindShopInSnapshot(old, s.Zone, s.MobId, s.RoomId)
					key := fmt.Sprintf("%s/%d/%d", s.Zone, s.MobId, s.RoomId)
					ds.Shops[key] = health.ComputeShopDelta(s, oldShop)
				}
				for _, f := range cur.Foragers {
					oldF := health.FindForagerInSnapshot(old, f.MobId)
					key := fmt.Sprintf("%d", f.MobId)
					ds.Foragers[key] = health.ComputeForagerDelta(f, oldF)
				}
				for _, c := range cur.Caravans {
					oldC := health.FindCaravanInSnapshot(old, c.InstId)
					key := fmt.Sprintf("%d", c.InstId)
					ds.Caravans[key] = health.ComputeCaravanDelta(c, oldC)
				}
			}
		}
		deltas[label] = ds
	}

	resp := economyAPIResponse{
		Snapshot: cur,
		Scores:   scores,
		Deltas:   deltas,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		mudlog.Error("economy API encode", "error", err)
	}
}

func economySnapshotAPI(w http.ResponseWriter, r *http.Request) {
	label := r.URL.Query().Get("label")
	snap := health.CaptureSnapshot()
	snap.Manual = true
	snap.ManualLabel = label
	if err := health.WriteSnapshot(snap); err != nil {
		mudlog.Error("manual snapshot write", "error", err)
		http.Error(w, "write failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]any{"unix_ts": snap.UnixTs, "manual": true}); err != nil {
		mudlog.Error("manual snapshot encode", "error", err)
	}
}
