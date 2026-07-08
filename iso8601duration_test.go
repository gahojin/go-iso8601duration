package iso8601duration

import (
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

var japanTz = time.FixedZone("Asia/Tokyo", 9*60*60)

func TestDuration_HelperMethods(t *testing.T) {
	d := Duration{Years: 1, Months: 2, Weeks: 3, Days: 4, Hours: 5, Minutes: 6, Seconds: 7, Nanoseconds: 800}

	t.Run("Equal", func(t *testing.T) {
		assert.True(t, d.Equal(d))
		assert.False(t, d.Equal(Duration{Years: 1}))
		assert.False(t, d.Equal(Duration{Negative: true, Years: 1, Months: 2, Weeks: 3, Days: 4, Hours: 5, Minutes: 6, Seconds: 7, Nanoseconds: 800}))
	})

	t.Run("IsZero", func(t *testing.T) {
		assert.True(t, Duration{}.IsZero())
		assert.True(t, Duration{Negative: true}.IsZero())
		assert.False(t, Duration{Years: 1}.IsZero())
	})

	t.Run("HasDatePart", func(t *testing.T) {
		assert.True(t, Duration{Years: 1}.HasDatePart())
		assert.True(t, Duration{Months: 1}.HasDatePart())
		assert.True(t, Duration{Weeks: 1}.HasDatePart())
		assert.True(t, Duration{Days: 1}.HasDatePart())
		assert.False(t, Duration{Hours: 1}.HasDatePart())
	})

	t.Run("HasTimePart", func(t *testing.T) {
		assert.True(t, Duration{Hours: 1}.HasTimePart())
		assert.True(t, Duration{Minutes: 1}.HasTimePart())
		assert.True(t, Duration{Seconds: 1}.HasTimePart())
		assert.True(t, Duration{Nanoseconds: 1}.HasTimePart())
		assert.False(t, Duration{Days: 1}.HasTimePart())
	})

	t.Run("OnlyYMWD", func(t *testing.T) {
		expected := Duration{Negative: false, Years: 1, Months: 2, Weeks: 3, Days: 4}
		assert.Equal(t, expected, d.OnlyYMWD())
	})

	t.Run("OnlyTime", func(t *testing.T) {
		expected := Duration{Negative: false, Hours: 5, Minutes: 6, Seconds: 7, Nanoseconds: 800}
		assert.Equal(t, expected, d.OnlyTime())
	})

	t.Run("GetYMWD", func(t *testing.T) {
		y, m, w, day := d.GetYMWD()
		assert.Equal(t, 1, y)
		assert.Equal(t, 2, m)
		assert.Equal(t, 3, w)
		assert.Equal(t, 4, day)

		negD := d
		negD.Negative = true
		y, m, w, day = negD.GetYMWD()
		assert.Equal(t, -1, y)
		assert.Equal(t, -2, m)
		assert.Equal(t, -3, w)
		assert.Equal(t, -4, day)
	})

	t.Run("Negate and Abs", func(t *testing.T) {
		d2 := Duration{Years: 1}
		assert.True(t, d2.Negate().Negative)
		assert.False(t, d2.Negate().Negate().Negative)
		assert.False(t, d2.Negate().Abs().Negative)
	})
}

func TestDuration_String_EdgeCases(t *testing.T) {
	tests := []struct {
		name     string
		duration Duration
		want     string
	}{
		{"Zero", Duration{}, "PT0S"},
		{"Negative Zero", Duration{Negative: true}, "-PT0S"},
		{"Only Years", Duration{Years: 1}, "P1Y"},
		{"Only Months", Duration{Months: 1}, "P1M"},
		{"Only Weeks", Duration{Weeks: 1}, "P1W"},
		{"Only Days", Duration{Days: 1}, "P1D"},
		{"Only Hours", Duration{Hours: 1}, "PT1H"},
		{"Only Minutes", Duration{Minutes: 1}, "PT1M"},
		{"Only Seconds", Duration{Seconds: 1}, "PT1S"},
		{"Only Nanoseconds (0.5s)", Duration{Nanoseconds: 500000000}, "PT0.5S"},
		{"Only Nanoseconds (0.000000001s)", Duration{Nanoseconds: 1}, "PT0.000000001S"},
		{"Seconds and Nanoseconds (1.000000001s)", Duration{Seconds: 1, Nanoseconds: 1}, "PT1.000000001S"},
		{"Nanoseconds overflow (1.5s)", Duration{Nanoseconds: 1500000000}, "PT1.5S"},
		{"Complex Negative", Duration{Negative: true, Years: 1, Seconds: 1}, "-P1YT1S"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.duration.String())
		})
	}
}

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
