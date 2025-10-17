package iso8601duration

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
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

	// 年に小数部を含む
	actual, err = ParseString("P0.5Y")
	assert.Nil(t, err)
	assert.Equal(t, "P6M", actual.String())

	// 日に小数部を含む
	actual, err = ParseString("P0.5D")
	assert.Nil(t, err)
	assert.Equal(t, "PT12H", actual.String())

	// 時刻に小数部を含む
	// 0.34h -> 20.4m -> 20m + 24s
	// 0.78m -> 46.8s
	actual, err = ParseString("PT12.34H56.78M9.01S")
	assert.Nil(t, err)
	assert.Equal(t, "PT12H77M19.81S", actual.String())
	assert.True(t, actual.HasTimePart())
	actual, err = ParseString("PT12,34H56,78M9,01S")
	assert.Nil(t, err)
	assert.Equal(t, "PT12H77M19.81S", actual.String())
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
		assert.Nil(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, *actual)
	})
}

func TestTextMarshal(t *testing.T) {
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

		bytes, err := expect.MarshalText()
		assert.Nil(t, err)
		assert.NotNil(t, bytes)

		var actual Duration
		err = actual.UnmarshalText(bytes)
		assert.Nil(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual)
	})
}

func TestJSONMarshal(t *testing.T) {
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

		bytes, err := json.Marshal(expect)
		assert.Nil(t, err)
		assert.NotNil(t, bytes)

		var actual Duration
		err = json.Unmarshal(bytes, &actual)
		assert.Nil(t, err)
		// ナノ秒のうち、秒単位の桁は、秒に加算する
		expect.Seconds += uint32(time.Duration(expect.Nanoseconds) / time.Second)
		expect.Nanoseconds = uint32(time.Duration(expect.Nanoseconds) % time.Second)
		assert.Equal(t, expect, actual)
	})
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

func TestAdd(t *testing.T) {
	sut, err := ParseString("P1Y2M3W4DT5H6M7.8S")
	assert.Nil(t, err)

	actual, ok := sut.Add(*sut)
	assert.True(t, ok)
	assert.Equal(t, Duration{
		Years:       2,
		Months:      4,
		Weeks:       6,
		Days:        8,
		Hours:       10,
		Minutes:     12,
		Seconds:     15,
		Nanoseconds: 600 * 1000 * 1000,
	}, actual)
}

func TestAddTo(t *testing.T) {
	sut, err := ParseString("P1Y2M3W4DT5H6M7.8S")
	assert.Nil(t, err)

	base := time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC)
	actual := sut.AddTo(base)
	assert.Equal(t, time.Date(2026, 12, 10+21+4, 5, 6, 7, 800*1000*1000, time.UTC), actual)
}

func TestAddToJapan(t *testing.T) {
	tests := []struct {
		from     string
		duration string
		want     string
	}{
		// 当日
		{from: "2020-06-01", duration: "P1D", want: "2020-06-02T00:00:00"},
		// 2日間
		{from: "2020-06-01", duration: "P2D", want: "2020-06-03T00:00:00"},
		// 月末/2日間
		{from: "2020-06-30", duration: "P2D", want: "2020-07-02T00:00:00"},
		// 1ヶ月
		{from: "2020-06-01", duration: "P1M", want: "2020-07-01T00:00:00"},
		{from: "2020-08-31", duration: "P1M", want: "2020-10-01T00:00:00"},
		{from: "2020-10-10", duration: "P1M", want: "2020-11-10T00:00:00"},
		{from: "2020-12-01", duration: "P1M", want: "2021-01-01T00:00:00"},
		{from: "2021-01-31", duration: "P1M", want: "2021-03-01T00:00:00"},
		{from: "2022-02-28", duration: "P1M", want: "2022-03-28T00:00:00"},
		{from: "2024-01-29", duration: "P1M", want: "2024-02-29T00:00:00"},
		{from: "2024-01-30", duration: "P1M", want: "2024-03-01T00:00:00"},
		{from: "2024-01-31", duration: "P1M", want: "2024-03-01T00:00:00"},
		{from: "2024-02-01", duration: "P1M", want: "2024-03-01T00:00:00"},
		{from: "2024-03-01", duration: "P1M", want: "2024-04-01T00:00:00"},
		// 1ヶ月 1日
		{from: "2020-06-01", duration: "P1M1D", want: "2020-07-02T00:00:00"},
		{from: "2020-08-31", duration: "P1M1D", want: "2020-10-02T00:00:00"},
		{from: "2020-10-10", duration: "P1M1D", want: "2020-11-11T00:00:00"},
		{from: "2020-12-01", duration: "P1M1D", want: "2021-01-02T00:00:00"},
		{from: "2021-01-31", duration: "P1M1D", want: "2021-03-02T00:00:00"},
		{from: "2021-02-28", duration: "P1M1D", want: "2021-03-29T00:00:00"},
		{from: "2024-01-29", duration: "P1M1D", want: "2024-03-01T00:00:00"},
		{from: "2024-01-30", duration: "P1M1D", want: "2024-03-02T00:00:00"},
		{from: "2024-01-31", duration: "P1M1D", want: "2024-03-02T00:00:00"},
		{from: "2024-02-01", duration: "P1M1D", want: "2024-03-02T00:00:00"},
		{from: "2024-03-01", duration: "P1M1D", want: "2024-04-02T00:00:00"},
		// 3ヶ月
		{from: "2020-06-01", duration: "P3M", want: "2020-09-01T00:00:00"},
		{from: "2020-08-31", duration: "P3M", want: "2020-12-01T00:00:00"},
		{from: "2020-10-10", duration: "P3M", want: "2021-01-10T00:00:00"},
		{from: "2020-12-01", duration: "P3M", want: "2021-03-01T00:00:00"},
		{from: "2021-01-31", duration: "P3M", want: "2021-05-01T00:00:00"},
		{from: "2021-02-28", duration: "P3M", want: "2021-05-28T00:00:00"},
		// 6ヶ月
		{from: "2020-06-01", duration: "P6M", want: "2020-12-01T00:00:00"},
		{from: "2020-08-31", duration: "P6M", want: "2021-03-01T00:00:00"},
		{from: "2020-10-10", duration: "P6M", want: "2021-04-10T00:00:00"},
		{from: "2020-12-01", duration: "P6M", want: "2021-06-01T00:00:00"},
		{from: "2021-01-31", duration: "P6M", want: "2021-07-31T00:00:00"},
		{from: "2021-02-28", duration: "P6M", want: "2021-08-28T00:00:00"},
		// 1週間
		{from: "2021-02-28", duration: "P1W", want: "2021-03-07T00:00:00"},
		// その他
		{from: "2023-01-01", duration: "P2M", want: "2023-03-01T00:00:00"}, // 143条2項 (平年)
		{from: "2024-01-01", duration: "P2M", want: "2024-03-01T00:00:00"}, // 143条2項 (閏年)
		{from: "2024-01-20", duration: "P2M", want: "2024-03-20T00:00:00"}, // 143条2項
		{from: "2024-01-31", duration: "P2M", want: "2024-03-31T00:00:00"}, // 143条2項
		{from: "2023-01-31", duration: "P1M", want: "2023-03-01T00:00:00"}, // 143条2項ただし書 (平年)
		{from: "2024-01-31", duration: "P1M", want: "2024-03-01T00:00:00"}, // 143条2項ただし書 (閏年)
		{from: "2024-03-31", duration: "P1M", want: "2024-05-01T00:00:00"}, // 143条2項ただし書
		{from: "2024-03-31", duration: "P1M", want: "2024-05-01T00:00:00"}, // 143条2項ただし書
		{from: "2024-05-30T01:00:00", duration: "P1M", want: "2024-07-01T00:00:00"},
		{from: "2024-05-30T01:00:00", duration: "P1MT1H", want: "2024-06-30T02:00:00"},
		{from: "2023-01-29", duration: "P1M", want: "2023-03-01T00:00:00"},
		{from: "2020-02-28", duration: "P1Y", want: "2021-02-28T00:00:00"},
		{from: "2020-02-28T01:00:00", duration: "P1Y", want: "2021-03-01T00:00:00"},
		{from: "2020-08-15", duration: "P1Y3M", want: "2021-11-15T00:00:00"},
		{from: "2020-08-31", duration: "P1Y1M", want: "2021-10-01T00:00:00"},
		{from: "2024-06-01T18:00:00", duration: "PT30H", want: "2024-06-03T00:00:00"},
		{from: "2024-06-01", duration: "P2Y", want: "2026-06-01T00:00:00"},
	}
	tz := time.FixedZone("Asia/Tokyo", 9*60*60)
	var fromTime time.Time
	var err error
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s %s", tt.from, tt.duration), func(t *testing.T) {
			if strings.Contains(tt.from, "T") {
				fromTime, err = time.ParseInLocation("2006-01-02T15:04:05", tt.from, tz)
			} else {
				fromTime, err = time.ParseInLocation("2006-01-02", tt.from, tz)
			}
			assert.Nil(t, err)
			sut, err := ParseString(tt.duration)
			assert.Nil(t, err)
			actual, err := sut.AddToJapan(fromTime)
			assert.Nil(t, err)
			expect, err := time.ParseInLocation("2006-01-02T15:04:05", tt.want, tz)
			assert.Nil(t, err)
			assert.Equal(t, expect, *actual)
		})
	}
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
		years := rapid.Uint32Max(math.MaxInt16).Draw(t, "years")
		months := rapid.Uint32Max(math.MaxInt16).Draw(t, "months")
		weeks := rapid.Uint32Max(math.MaxInt16).Draw(t, "weeks")
		days := rapid.Uint32Max(math.MaxInt16).Draw(t, "days")
		hours := rapid.Uint32Max(math.MaxInt16).Draw(t, "hours")
		minutes := rapid.Uint32Max(math.MaxInt16).Draw(t, "minutes")
		seconds := rapid.Uint32Max(math.MaxInt16).Draw(t, "seconds")
		nanoseconds := rapid.Uint32Max(math.MaxInt16).Draw(t, "nanoseconds")

		sut := Duration{
			Years:       years,
			Months:      months,
			Weeks:       weeks,
			Days:        days,
			Hours:       hours,
			Minutes:     minutes,
			Seconds:     seconds,
			Nanoseconds: nanoseconds,
		}
		actual, ok := sut.Normalize()

		assert.True(t, ok)
		assert.Less(t, actual.Months, uint32(12))
		if months >= 12 {
			assert.Greater(t, actual.Years, years)
		} else {
			assert.GreaterOrEqual(t, actual.Years, years)
		}

		assert.Less(t, actual.Hours, uint32(24))
		assert.Less(t, actual.Minutes, uint32(60))
		assert.Less(t, actual.Seconds, uint32(60))
		assert.Less(t, actual.Nanoseconds, uint32(1000*1000*1000))
	})

	// オーバーフロー
	_, ok = Duration{Years: math.MaxInt32, Months: 12}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Years: math.MaxInt32, Months: 11}.Normalize()
	assert.True(t, ok)
	_, ok = Duration{Days: math.MaxInt32, Hours: 24}.Normalize()
	assert.False(t, ok)
	_, ok = Duration{Days: math.MaxInt32, Hours: 23}.Normalize()
	assert.True(t, ok)
	actual, ok = Duration{Hours: math.MaxInt32, Minutes: 60}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Days: math.MaxInt32 / 24, Hours: math.MaxInt32%24 + 1, Minutes: 0}, actual)
	actual, ok = Duration{Hours: math.MaxInt32, Minutes: 59, Seconds: 60}.Normalize()
	assert.True(t, ok)
	assert.Equal(t, Duration{Days: math.MaxInt32 / 24, Hours: math.MaxInt32%24 + 1, Minutes: 0}, actual)
	_, ok = Duration{Hours: math.MaxInt32, Minutes: 59, Seconds: 59}.Normalize()
	assert.True(t, ok)
}
