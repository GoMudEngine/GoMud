package relationships

import "testing"

func TestTypeConstants(t *testing.T) {
	cases := []struct {
		t    Type
		want string
	}{
		{TypeFamily, "family"},
		{TypeFriend, "friend"},
		{TypeRival, "rival"},
		{TypeLover, "lover"},
		{TypeEmployer, "employer"},
		{TypeEmployee, "employee"},
	}
	for _, c := range cases {
		if string(c.t) != c.want {
			t.Errorf("Type %s: got %s, want %s", c.want, c.t, c.want)
		}
	}
}

func TestIsSymmetric(t *testing.T) {
	cases := []struct {
		t    Type
		want bool
	}{
		{TypeFamily, true},
		{TypeFriend, true},
		{TypeRival, true},
		{TypeLover, true},
		{TypeEmployer, false},
		{TypeEmployee, false},
	}
	for _, c := range cases {
		if got := IsSymmetric(c.t); got != c.want {
			t.Errorf("IsSymmetric(%s): got %v, want %v", c.t, got, c.want)
		}
	}
}

func TestInverseType(t *testing.T) {
	if InverseType(TypeEmployer) != TypeEmployee {
		t.Errorf("InverseType(employer) should be employee")
	}
	if InverseType(TypeEmployee) != TypeEmployer {
		t.Errorf("InverseType(employee) should be employer")
	}
	// Symmetric types invert to themselves.
	if InverseType(TypeFamily) != TypeFamily {
		t.Errorf("InverseType(family) should be family")
	}
}
