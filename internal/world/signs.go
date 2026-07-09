package world

// Mood is how the collective tide feels in the wastes (src/signs.ts).
type Mood string

const (
	MoodRising  Mood = "rising"
	MoodFalling Mood = "falling"
	MoodStill   Mood = "still"
)

// MoodForTide maps the shared war tide (-100..+100) to a mood. Signs only fire
// once it has decisively tipped; the balanced middle stays the plain wastes.
func MoodForTide(tide int) Mood {
	if tide >= 40 {
		return MoodRising
	}
	if tide <= -40 {
		return MoodFalling
	}
	return MoodStill
}
