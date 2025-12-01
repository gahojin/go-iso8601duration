package iso8601duration

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"pgregory.net/rapid"
)

func TestParseString(t *testing.T) {
	// フォーマットエラー
	actual, err := ParseString("12Y10M")
	assert.ErrorContains(t, err, "unexpected input")

	// 日付部のみ
	actual, err = ParseString("P12Y10M")
	assert.NoError(t, err)
	assert.Equal(t, "P12Y10M", actual.String())
	assert.False(t, actual.HasTimePart())

	// 時刻部のみ
	actual, err = ParseString("PT12H34M56S")
	assert.NoError(t, err)
	assert.Equal(t, "PT12H34M56S", actual.String())
	assert.True(t, actual.HasTimePart())

	// 週を含む
	actual, err = ParseString("P12Y10M3W")
	assert.NoError(t, err)
	assert.Equal(t, "P12Y10M3W", actual.String())
	assert.False(t, actual.HasTimePart())

	// 年に小数部を含む
	actual, err = ParseString("P0.5Y")
	assert.NoError(t, err)
	assert.Equal(t, "P6M", actual.String())

	// 日に小数部を含む
	actual, err = ParseString("P0.5D")
	assert.NoError(t, err)
	assert.Equal(t, "PT12H", actual.String())

	// 時刻に小数部を含む
	// 0.34h -> 20.4m -> 20m + 24s
	// 0.78m -> 46.8s
	actual, err = ParseString("PT12.34H56.78M9.01S")
	assert.NoError(t, err)
	assert.Equal(t, "PT12H77M19.81S", actual.String())
	assert.True(t, actual.HasTimePart())
	actual, err = ParseString("PT12,34H56,78M9,01S")
	assert.NoError(t, err)
	assert.Equal(t, "PT12H77M19.81S", actual.String())
	assert.True(t, actual.HasTimePart())

	// マイナス
	actual, err = ParseString("-P12Y10M")
	assert.NoError(t, err)
	assert.True(t, actual.Negative)
	assert.Equal(t, "-P12Y10M", actual.String())
	assert.False(t, actual.HasTimePart())

	// 全ての要素が入っている
	actual, err = ParseString("-P1Y2M4DT4H56M7.8S")
	assert.NoError(t, err)
	assert.Equal(t, "-P1Y2M4DT4H56M7.8S", actual.String())
	assert.True(t, actual.HasDatePart())
	assert.True(t, actual.HasTimePart())

	// プロパティテスト
	rapid.Check(t, func(t *rapid.T) {
		expect := Duration{
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

		actual, err = ParseString(expect.String())
		assert.NoError(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual)
	})
}

func BenchmarkParseString(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseString("-P1Y2M4DT4H56M7.8S")
	}
}
