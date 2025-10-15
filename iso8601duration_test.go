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

func TestJSON(t *testing.T) {
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
