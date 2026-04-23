package analysis

import "math"

type MomentumTag string

const (
	MomentumNone    MomentumTag = ""
	MomentumRising  MomentumTag = "rising"
	MomentumFalling MomentumTag = "falling-off"
	MomentumHidden  MomentumTag = "hidden-gem"
	MomentumDying   MomentumTag = "dying"
)

const momentumWindow = 4
const momentumNoiseFloor = 1.0

// TagMomentum classifies the last `momentumWindow` points of both series.
// Returns MomentumNone if either series is too short or change is below noise.
func TagMomentum(wr, pr []float64) MomentumTag {
	if len(wr) < momentumWindow || len(pr) < momentumWindow {
		return MomentumNone
	}
	wrProj := linearSlope(wr[len(wr)-momentumWindow:]) * float64(momentumWindow)
	prProj := linearSlope(pr[len(pr)-momentumWindow:]) * float64(momentumWindow)
	if math.Abs(wrProj) < momentumNoiseFloor && math.Abs(prProj) < momentumNoiseFloor {
		return MomentumNone
	}
	wrUp := wrProj >= momentumNoiseFloor
	wrDown := wrProj <= -momentumNoiseFloor
	prUp := prProj >= momentumNoiseFloor
	prDown := prProj <= -momentumNoiseFloor
	switch {
	case wrUp && prUp:
		return MomentumRising
	case wrDown && prUp:
		return MomentumFalling
	case wrUp && (prDown || (!prUp && !prDown)):
		return MomentumHidden
	case wrDown && prDown:
		return MomentumDying
	default:
		return MomentumNone
	}
}

// linearSlope returns the slope of y over x=[0..n-1] via least squares.
func linearSlope(y []float64) float64 {
	n := float64(len(y))
	if n < 2 {
		return 0
	}
	var sx, sy, sxy, sxx float64
	for i, v := range y {
		x := float64(i)
		sx += x
		sy += v
		sxy += x * v
		sxx += x * x
	}
	denom := n*sxx - sx*sx
	if denom == 0 {
		return 0
	}
	return (n*sxy - sx*sy) / denom
}
