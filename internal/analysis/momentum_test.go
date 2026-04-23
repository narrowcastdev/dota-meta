package analysis

import "testing"

func TestLinearSlope(t *testing.T) {
	got := linearSlope([]float64{1, 3, 5, 7})
	if got != 2 {
		t.Errorf("slope=%v want 2", got)
	}
}

func TestTagMomentum(t *testing.T) {
	cases := []struct {
		wrSeries, prSeries []float64
		want               MomentumTag
	}{
		{[]float64{50, 51, 52, 53}, []float64{5, 6, 7, 8}, MomentumRising},
		{[]float64{53, 52, 51, 50}, []float64{5, 6, 7, 8}, MomentumFalling},
		{[]float64{50, 51, 52, 53}, []float64{5, 5, 4, 4}, MomentumHidden},
		{[]float64{53, 52, 51, 50}, []float64{8, 7, 6, 5}, MomentumDying},
		{[]float64{50, 50, 50, 50}, []float64{5, 5, 5, 5}, MomentumNone},
		{[]float64{50, 50, 50}, []float64{5, 5, 5}, MomentumNone},
	}
	for i, c := range cases {
		got := TagMomentum(c.wrSeries, c.prSeries)
		if got != c.want {
			t.Errorf("case %d: got %q want %q", i, got, c.want)
		}
	}
}
