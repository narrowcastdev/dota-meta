package analysis

import "testing"

func TestTagClimb(t *testing.T) {
	cases := []struct {
		lowWR, highWR float64
		qualified     bool
		want          ClimbTag
	}{
		{lowWR: 50, highWR: 53, qualified: true, want: ClimbUp},
		{lowWR: 53, highWR: 50, qualified: true, want: ClimbDown},
		{lowWR: 51, highWR: 52, qualified: true, want: ClimbUniversal},
		{lowWR: 51, highWR: 52, qualified: false, want: ClimbUnknown},
	}
	for _, c := range cases {
		got := TagClimb(c.lowWR, c.highWR, c.qualified)
		if got != c.want {
			t.Errorf("TagClimb(%v,%v,%v)=%v, want %v", c.lowWR, c.highWR, c.qualified, got, c.want)
		}
	}
}
