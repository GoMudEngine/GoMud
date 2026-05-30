package characters

import "testing"

func TestParseArrestPolicy(t *testing.T) {
	cases := map[string]ArrestPolicy{
		"surrender": ArrestSurrender,
		"resist":    ArrestResist,
	}
	for in, want := range cases {
		got, ok := ParseArrestPolicy(in)
		if !ok || got != want {
			t.Errorf("ParseArrestPolicy(%q) = %v,%v; want %v,true", in, got, ok, want)
		}
	}
	if _, ok := ParseArrestPolicy("bogus"); ok {
		t.Error("bogus should not parse")
	}
}
