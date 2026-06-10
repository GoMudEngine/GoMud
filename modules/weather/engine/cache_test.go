package engine

import (
	"testing"

	"github.com/GoMudEngine/GoMud/modules/weather/sim"
)

func TestDecodeCache(t *testing.T) {
	// Absent / empty / malformed -> not ok, no panic.
	if _, ok := DecodeCache(nil); ok {
		t.Error("nil bytes should decode as not-ok")
	}
	if _, ok := DecodeCache([]byte("{not json")); ok {
		t.Error("malformed json should decode as not-ok")
	}

	// Stale version -> not ok (signals rebuild).
	stale := &sim.Graph{Version: sim.GraphVersion + 1}
	sb, err := stale.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := DecodeCache(sb); ok {
		t.Error("stale GraphVersion should decode as not-ok")
	}

	// Current version -> ok, returns the graph.
	good := &sim.Graph{Version: sim.GraphVersion, Nodes: map[string]sim.ZoneNode{"A": {Zone: "A"}}}
	gb, err := good.ToJSON()
	if err != nil {
		t.Fatal(err)
	}
	g, ok := DecodeCache(gb)
	if !ok || g == nil || len(g.Nodes) != 1 {
		t.Fatalf("current-version cache should decode ok; got ok=%v g=%v", ok, g)
	}
}
