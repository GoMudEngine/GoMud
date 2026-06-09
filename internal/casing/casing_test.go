package casing

import "testing"

func TestTitle(t *testing.T) {
	cases := []struct{ in, want string }{
		{"", ""},
		{"olen", "Olen"},
		{"temple priest olen", "Temple Priest Olen"},
		{"city guard", "City Guard"},
		{"iron sword", "Iron Sword"},
		{"captain of the guard", "Captain of the Guard"},
		{"tome of the deep", "Tome of the Deep"},
		{"ascendant master warrior", "Ascendant Master Warrior"},
		{"the warren", "The Warren"},
		{"keeper of the", "Keeper of The"},
		{"Olen", "Olen"},
		{"McGregor the brave", "McGregor the Brave"},
		{"Captain of the Guard", "Captain of the Guard"},
		{"valley   rat", "Valley Rat"},
		{"  lich  ", "Lich"},
		{"a", "A"},
	}
	for _, c := range cases {
		if got := Title(c.in); got != c.want {
			t.Errorf("Title(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestTitleIdempotent(t *testing.T) {
	for _, s := range []string{"temple priest olen", "captain of the guard", "iron sword", "the warren"} {
		once := Title(s)
		if twice := Title(once); twice != once {
			t.Errorf("Title not idempotent: Title(%q)=%q then Title(that)=%q", s, once, twice)
		}
	}
}
