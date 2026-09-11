package squirrel

import (
	"hash/fnv"
	"time"
)

// pick is the day-seeded choice, over a range of numbers.
//
// One implementation so the stamp, the light and the faces cannot drift into
// disagreeing about what day it is.
func pick(salt string, on time.Time, from, to int) int {
	h := fnv.New32a()
	_, _ = h.Write([]byte(salt))
	_, _ = h.Write([]byte(on.Format("2006-01-02")))
	return from + int(h.Sum32()%uint32(to-from+1))
}

func Face(m Mood, on time.Time) string {
	if pick("face "+string(m), on, 1, 2) == 2 {
		return "mood-" + string(m) + "-2.png"
	}
	return "mood-" + string(m) + ".png"
}

// TiltRange is how far the stamp can lean, in degrees. Exported so the test
// that holds this to "a few degrees" reads the number rather than repeating it.
const (
	TiltFrom = -11
	TiltTo   = -3
)

// Tilt is the angle the stamp lands at today. Eight degrees around the -7 it
// shipped at: past about a dozen a stamp reads as crooked rather than slapped on.
// Negative throughout, so it always leans the same way.
func Tilt(on time.Time) int { return pick("tilt", on, TiltFrom, TiltTo) }

// LightFrom and LightTo bound where the room's light falls, across the field.
const (
	LightFrom = 8
	LightTo   = 26
)

// Light is where the field's highlight sits today, as a percentage across.
//
// Across only. The vertical and the alpha stay, because .35 is a measured
// contrast result — cream on the lit centre reads 4.8:1 and failed at .5. Sliding
// sideways cannot quietly undo that measurement.
func Light(on time.Time) int { return pick("light", on, LightFrom, LightTo) }
