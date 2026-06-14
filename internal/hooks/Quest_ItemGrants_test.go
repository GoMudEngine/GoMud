package hooks

import "testing"

// parseItemGrants backs multi-item "stockpile / starter kit" quest rewards
// (e.g. an alchemy cert grants several brewed potions). These tests pin the
// parse; the application logic (StoreItem N times) lives in the handler.

func TestParseItemGrants_Empty(t *testing.T) {
	if g := parseItemGrants(""); g != nil {
		t.Errorf("empty item_info: want nil, got %v", g)
	}
}

func TestParseItemGrants_SingleBareId(t *testing.T) {
	// a bare itemid with no colon defaults to quantity one
	g := parseItemGrants("30058")
	if len(g) != 1 || g[0].itemId != 30058 || g[0].qty != 1 {
		t.Errorf("single bare id: got %+v", g)
	}
}

func TestParseItemGrants_SingleWithQty(t *testing.T) {
	g := parseItemGrants("30036:3")
	if len(g) != 1 || g[0].itemId != 30036 || g[0].qty != 3 {
		t.Errorf("single with qty: got %+v", g)
	}
}

func TestParseItemGrants_Multi(t *testing.T) {
	g := parseItemGrants("30036:3,30028:2,30058")
	if len(g) != 3 {
		t.Fatalf("multi grant: want 3, got %d (%+v)", len(g), g)
	}
	if g[0].itemId != 30036 || g[0].qty != 3 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].itemId != 30028 || g[1].qty != 2 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
	if g[2].itemId != 30058 || g[2].qty != 1 {
		t.Errorf("grant[2]: got %+v", g[2])
	}
}

func TestParseItemGrants_TrimsWhitespace(t *testing.T) {
	g := parseItemGrants(" 30036 : 3 , 30058 ")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%+v)", len(g), g)
	}
	if g[0].itemId != 30036 || g[0].qty != 3 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].itemId != 30058 || g[1].qty != 1 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}

func TestParseItemGrants_SkipsMalformed(t *testing.T) {
	// empty entries, non-numeric id, non-positive qty are all skipped
	g := parseItemGrants("30036:2,,abc,30058:0,30059:-1,30028:1")
	if len(g) != 2 {
		t.Fatalf("want 2, got %d (%+v)", len(g), g)
	}
	if g[0].itemId != 30036 || g[0].qty != 2 {
		t.Errorf("grant[0]: got %+v", g[0])
	}
	if g[1].itemId != 30028 || g[1].qty != 1 {
		t.Errorf("grant[1]: got %+v", g[1])
	}
}
