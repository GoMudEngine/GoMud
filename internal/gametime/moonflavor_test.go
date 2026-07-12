package gametime

import "testing"

func TestMoonFlavorBucketMapping(t *testing.T) {
	cases := []struct {
		eye, wander float64
		want        int
	}{
		{0.2, 0.2, 1}, // new / new
		{0.2, 0.9, 2}, // new / full
		{0.9, 0.2, 3}, // full / new
		{0.9, 0.9, 4}, // full / full
		{0.5, 0.5, 4}, // boundary: >=0.5 counts as full
	}
	for _, c := range cases {
		if got := moonFlavorBucket(c.eye, c.wander); got != c.want {
			t.Fatalf("moonFlavorBucket(%v,%v) = %d, want %d", c.eye, c.wander, got, c.want)
		}
	}
}
