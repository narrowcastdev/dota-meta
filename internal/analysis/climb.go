package analysis

type ClimbTag string

const (
	ClimbUnknown   ClimbTag = ""
	ClimbUp        ClimbTag = "scales-up"
	ClimbDown      ClimbTag = "scales-down"
	ClimbUniversal ClimbTag = "universal"
)

const climbDeltaThreshold = 2.0

// TagClimb returns the climb tag for a hero given its Herald-Guardian WR and
// Divine WR. If the hero didn't meet the min-picks floor in one of the
// brackets, returns ClimbUnknown.
func TagClimb(lowWR, highWR float64, qualified bool) ClimbTag {
	if !qualified {
		return ClimbUnknown
	}
	delta := highWR - lowWR
	switch {
	case delta >= climbDeltaThreshold:
		return ClimbUp
	case delta <= -climbDeltaThreshold:
		return ClimbDown
	default:
		return ClimbUniversal
	}
}
