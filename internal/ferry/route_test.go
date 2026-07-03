package ferry

import "testing"

func validRoute() Route {
	return Route{
		RouteId:       "test_route",
		Name:          "The Test Packet",
		DeckRoom:      6423,
		Ports:         []Port{{DockRoom: 4118}, {DockRoom: 5502}},
		CrossingHours: 2,
		LayoverHours:  1,
		Fare:          75,
	}
}

func TestRouteValidate_Valid(t *testing.T) {
	if err := validRoute().Validate(); err != nil {
		t.Fatalf("expected valid, got %v", err)
	}
}

func TestRouteValidate_Rejects(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(*Route)
	}{
		{"missing routeid", func(r *Route) { r.RouteId = "" }},
		{"missing name", func(r *Route) { r.Name = "" }},
		{"no deck room", func(r *Route) { r.DeckRoom = 0 }},
		{"one port", func(r *Route) { r.Ports = r.Ports[:1] }},
		{"three ports", func(r *Route) { r.Ports = append(r.Ports, Port{DockRoom: 9}) }},
		{"same dock twice", func(r *Route) { r.Ports[1].DockRoom = r.Ports[0].DockRoom }},
		{"zero crossing", func(r *Route) { r.CrossingHours = 0 }},
		{"zero layover", func(r *Route) { r.LayoverHours = 0 }},
		{"layover >= 24h (gangplank exit lifetime)", func(r *Route) { r.LayoverHours = 24 }},
		{"negative phase offset", func(r *Route) { r.PhaseOffsetRounds = -1 }},
		{"negative fare", func(r *Route) { r.Fare = -1 }},
	}
	for _, tc := range cases {
		r := validRoute()
		tc.mutate(&r)
		if err := r.Validate(); err == nil {
			t.Errorf("%s: expected error, got nil", tc.name)
		}
	}
}

func TestRouteFilepath(t *testing.T) {
	if got := validRoute().Filepath(); got != "test_route.yaml" {
		t.Fatalf("Filepath() = %q", got)
	}
}
