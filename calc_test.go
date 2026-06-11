package iso8601duration

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"pgregory.net/rapid"
)

func TestAdd(t *testing.T) {
	sut, err := ParseString("P1Y2M3W4DT5H6M7.8S")
	assert.NoError(t, err)

	actual, ok := sut.Add(sut)
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
	assert.NoError(t, err)

	base := time.Date(2025, 10, 10, 0, 0, 0, 0, time.UTC)
	actual := sut.AddTo(base)
	assert.Equal(t, time.Date(2026, 12, 10+21+4, 5, 6, 7, 800*1000*1000, time.UTC), actual)

	// マイナス期間
	sut, err = ParseString("-P1Y2M3W4DT5H6M7.8S")
	assert.NoError(t, err)

	actual = sut.AddTo(base)
	assert.Equal(t, time.Date(2024, 8, 10-21-4, -5, -6, -7, -800*1000*1000, time.UTC), actual)
}

type AddToJapanTestData struct {
	from     string
	duration string
	preserve bool
	exclude  *bool
	want     string
}

func parseCsv(t *testing.T, record []string) *AddToJapanTestData {
	preserve := record[2]
	exclude := record[3]

	preserveTimeOnZero := false
	if preserve != "" {
		val, err := strconv.ParseBool(preserve)
		require.NoError(t, err)
		preserveTimeOnZero = val
	}

	var excludeStartDate *bool = nil
	if exclude != "" {
		val, err := strconv.ParseBool(exclude)
		require.NoError(t, err)
		excludeStartDate = &val
	}

	return &AddToJapanTestData{
		from:     record[0],
		duration: record[1],
		preserve: preserveTimeOnZero,
		exclude:  excludeStartDate,
		want:     record[4],
	}
}

func TestAddToJapan(t *testing.T) {
	t.Helper()

	testCsvFilePath := filepath.Join("testdata", "add_to_japan.csv")

	f, err := os.ReadFile(filepath.Clean(testCsvFilePath))
	assert.NoError(t, err)

	r := csv.NewReader(bytes.NewReader(f))
	r.Comment = '/'
	r.FieldsPerRecord = -1

	// ヘッダー部をスキップ
	_, err = r.Read()
	assert.NoError(t, err)

	for {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		assert.NoError(t, err)

		tt := parseCsv(t, record)

		var fromTime time.Time
		t.Run(fmt.Sprintf("%s %s", tt.from, tt.duration), func(t *testing.T) {
			if strings.Contains(tt.from, "T") {
				fromTime, err = time.ParseInLocation("2006-01-02T15:04:05", tt.from, japanTz)
			} else {
				fromTime, err = time.ParseInLocation("2006-01-02", tt.from, japanTz)
			}
			assert.NoError(t, err)
			sut, err := ParseString(tt.duration)
			assert.NoError(t, err)

			var actual time.Time
			if tt.preserve {
				actual = sut.AddToJapan(fromTime, WithPreserveTimeOnZero())
			} else if tt.exclude == nil {
				actual = sut.AddToJapan(fromTime)
			} else {
				actual = sut.AddToJapan(fromTime, WithExcludeStartDate(*tt.exclude))
			}
			expect, err := time.ParseInLocation("2006-01-02T15:04:05", tt.want, japanTz)
			assert.NoError(t, err)
			assert.Equal(t, expect, actual)
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
