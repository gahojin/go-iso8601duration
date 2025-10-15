package iso8601duration

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestParseString(t *testing.T) {
	// フォーマットエラー
	actual, err := ParseString("12Y10M")
	assert.Error(t, err, "Expected error for invalid ISO8601 duration")
	assert.Nil(t, actual)

	// 日付部のみ
	actual, err = ParseString("P12Y10M")
	assert.Nil(t, err)
	assert.Equal(t, "P12Y10M", actual.String())
	assert.False(t, actual.HasTimePart())

	// 時刻部のみ
	actual, err = ParseString("PT12H34M56S")
	assert.Nil(t, err)
	assert.Equal(t, "PT12H34M56S", actual.String())
	assert.True(t, actual.HasTimePart())

	// 週を含む
	actual, err = ParseString("P12Y10M3W")
	assert.Nil(t, err)
	assert.Equal(t, "P12Y10M3W", actual.String())
	assert.False(t, actual.HasTimePart())

	// 時刻に小数部を含む
	actual, err = ParseString("PT12.34H56.78M9.01S")
	assert.Nil(t, err)
	assert.Equal(t, "PT12.34H56.78M9.01S", actual.String())
	assert.True(t, actual.HasTimePart())
	actual, err = ParseString("PT12,34H56,78M9,01S")
	assert.Nil(t, err)
	assert.Equal(t, "PT12.34H56.78M9.01S", actual.String())
	assert.True(t, actual.HasTimePart())

	// マイナス
	actual, err = ParseString("-P12Y10M")
	assert.Nil(t, err)
	assert.True(t, actual.Negative)
	assert.Equal(t, "-P12Y10M", actual.String())
	assert.False(t, actual.HasTimePart())

	// プロパティテスト
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
			Negative: rapid.Bool().Draw(t, "negative"),
			Years:    rapid.Uint64().Draw(t, "years"),
			Months:   rapid.Uint64().Draw(t, "months"),
			Weeks:    rapid.Uint64().Draw(t, "weeks"),
			Days:     rapid.Uint64().Draw(t, "days"),
			Hours:    rapid.Float64Min(0).Draw(t, "hours"),
			Minutes:  rapid.Float64Min(0).Draw(t, "minutes"),
			Seconds:  rapid.Float64Min(0).Draw(t, "seconds"),
		}

		actual, err = ParseString(expect.String())
		assert.Nil(t, err)
		assert.Equal(t, expect, *actual)
	})
}

func TestTextMarshal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
			Negative: rapid.Bool().Draw(t, "negative"),
			Years:    rapid.Uint64().Draw(t, "years"),
			Months:   rapid.Uint64().Draw(t, "months"),
			Weeks:    rapid.Uint64().Draw(t, "weeks"),
			Days:     rapid.Uint64().Draw(t, "days"),
			Hours:    rapid.Float64Min(0).Draw(t, "hours"),
			Minutes:  rapid.Float64Min(0).Draw(t, "minutes"),
			Seconds:  rapid.Float64Min(0).Draw(t, "seconds"),
		}

		bytes, err := expect.MarshalText()
		assert.Nil(t, err)
		assert.NotNil(t, bytes)

		var actual Duration
		err = actual.UnmarshalText(bytes)
		assert.Nil(t, err)
		assert.Equal(t, expect, actual)
	})
}

func TestJSONMarshal(t *testing.T) {
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
			Negative: rapid.Bool().Draw(t, "negative"),
			Years:    rapid.Uint64().Draw(t, "years"),
			Months:   rapid.Uint64().Draw(t, "months"),
			Weeks:    rapid.Uint64().Draw(t, "weeks"),
			Days:     rapid.Uint64().Draw(t, "days"),
			Hours:    rapid.Float64Min(0).Draw(t, "hours"),
			Minutes:  rapid.Float64Min(0).Draw(t, "minutes"),
			Seconds:  rapid.Float64Min(0).Draw(t, "seconds"),
		}

		bytes, err := json.Marshal(expect)
		assert.Nil(t, err)
		assert.NotNil(t, bytes)

		var actual Duration
		err = json.Unmarshal(bytes, &actual)
		assert.Nil(t, err)
		assert.Equal(t, expect, actual)
	})
}

func TestIsValid(t *testing.T) {
	// プロパティテスト
	rapid.Check(t, func(t *rapid.T) {
		sut := Duration{
			Negative: rapid.Bool().Draw(t, "negative"),
			Years:    rapid.Uint64().Draw(t, "years"),
			Months:   rapid.Uint64().Draw(t, "months"),
			Weeks:    rapid.Uint64().Draw(t, "weeks"),
			Days:     rapid.Uint64().Draw(t, "days"),
			Hours:    rapid.Float64().Draw(t, "hours"),
			Minutes:  rapid.Float64().Draw(t, "minutes"),
			Seconds:  rapid.Float64().Draw(t, "seconds"),
		}

		if sut.Years > math.MaxInt64 || sut.Months > math.MaxInt64 || sut.Weeks > math.MaxInt64 || sut.Days > math.MaxInt64 || sut.Hours < 0.0 || sut.Minutes < 0.0 || sut.Seconds < 0.0 {
			assert.False(t, sut.IsValid())
		} else {
			assert.True(t, sut.IsValid())
		}
	})
}

func TestAdd(t *testing.T) {
	dur, err := ParseString("P1Y2M3W4DT5H6M7.8S")
	assert.Nil(t, err)

	base := time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC)
	actual := dur.Add(base)
	assert.Equal(t, time.Date(2026, 12, 10+21+4, 5, 6, 7, 800*1000*1000, time.UTC), actual)
}

func TestNormalize(t *testing.T) {
	// 境界チェック
	actual, ok := Duration{Months: 12}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Years: 1}, actual)

	actual, ok = Duration{Hours: 24}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Days: 1}, actual)

	actual, ok = Duration{Minutes: 60}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Hours: 1}, actual)

	actual, ok = Duration{Seconds: 60}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Minutes: 1}, actual)

	actual, ok = Duration{Months: 12, Hours: 24, Minutes: 60, Seconds: 60}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Years: 1, Days: 1, Hours: 1, Minutes: 1}, actual)

	// プロパティテスト
	rapid.Check(t, func(t *rapid.T) {
		years := rapid.Uint64Max(10000).Draw(t, "years")
		months := rapid.Uint64Max(10000).Draw(t, "months")
		weeks := rapid.Uint64Max(10000).Draw(t, "weeks")
		days := rapid.Uint64Max(10000).Draw(t, "days")
		hours := rapid.Float64Range(0, 1000).Draw(t, "hours")
		minutes := rapid.Float64Range(0, 1000).Draw(t, "minutes")
		seconds := rapid.Float64Range(0, 1000).Draw(t, "seconds")

		sut := Duration{
			Years:   years,
			Months:  months,
			Weeks:   weeks,
			Days:    days,
			Hours:   hours,
			Minutes: minutes,
			Seconds: seconds,
		}
		actual, ok := sut.Normalize()

		assert.True(t, ok)
		assert.Less(t, actual.Months, uint64(12))
			if months >= 12 {
				assert.Greater(t, actual.Years, years)
			} else {
				assert.GreaterOrEqual(t, actual.Years, years)
			}

		assert.Less(t, actual.Hours, float64(24))
		assert.Less(t, actual.Minutes, float64(60))
		assert.Less(t, actual.Seconds, float64(60))
	})

	// オーバーフロー
	_, ok = Duration{Years: math.MaxInt64, Months: 12}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Years: math.MaxInt64, Months: 11}.Normalize()
	assert.True(t, ok)
	_, ok = Duration{Days: math.MaxInt64, Hours: 24}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Days: math.MaxInt64, Hours: 23}.Normalize()
	assert.True(t, ok)
	_, ok = Duration{Hours: math.MaxInt64, Minutes: 60}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Hours: math.MaxInt64}.Normalize()
	assert.True(t, ok)
	_, ok = Duration{Hours: math.MaxInt64, Minutes: 59, Seconds: 60}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Hours: math.MaxInt64, Minutes: 59, Seconds: 59}.Normalize()
	assert.True(t, ok)
}
