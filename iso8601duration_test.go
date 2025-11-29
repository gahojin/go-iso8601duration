package iso8601duration

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

var japanTz = time.FixedZone("Asia/Tokyo", 9*60*60)

func TestIsValid(t *testing.T) {
	// プロパティテスト
	rapid.Check(t, func(t *rapid.T) {
		sut := Duration{
			Negative:    rapid.Bool().Draw(t, "negative"),
			Years:       rapid.Uint32().Draw(t, "years"),
			Months:      rapid.Uint32().Draw(t, "months"),
			Weeks:       rapid.Uint32().Draw(t, "weeks"),
			Days:        rapid.Uint32().Draw(t, "days"),
			Hours:       rapid.Uint32().Draw(t, "hours"),
			Minutes:     rapid.Uint32().Draw(t, "minutes"),
			Seconds:     rapid.Uint32().Draw(t, "seconds"),
			Nanoseconds: rapid.Uint32().Draw(t, "nanoseconds"),
		}

		if sut.Years > math.MaxInt32 || sut.Months > math.MaxInt32 || sut.Weeks > math.MaxInt32 || sut.Days > math.MaxInt32 || sut.Hours > math.MaxInt32 || sut.Minutes > math.MaxInt32 || sut.Seconds > math.MaxInt32 || sut.Nanoseconds > math.MaxInt32 {
			assert.False(t, sut.IsValid())
		} else {
			assert.True(t, sut.IsValid())
		}
	})
}
