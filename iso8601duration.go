package iso8601duration

import (
	"bytes"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/shopspring/decimal"
)

var (
	one                   = decimal.NewFromInt(1)
	monthsPerYear         = decimal.NewFromInt(12)
	hoursPerDay           = decimal.NewFromInt(24)
	minutesPerHour        = decimal.NewFromInt(60)
	secondsPerMinute      = decimal.NewFromInt(60)
	nanosecondsPerSeconds = decimal.NewFromUint64(uint64(time.Second))
)

type Duration struct {
	Negative    bool
	Years       uint32
	Months      uint32
	Weeks       uint32
	Days        uint32
	Hours       uint32
	Minutes     uint32
	Seconds     uint32
	Nanoseconds uint32
}

// Equal は値が一致するかを返す
func (d Duration) Equal(other Duration) bool {
	return d.Negative == d.Negative && d.Years == other.Years && d.Months == other.Months && d.Weeks == other.Weeks && d.Days == other.Days && d.Hours == other.Hours && d.Minutes == other.Minutes && d.Seconds == other.Seconds && d.Nanoseconds == other.Nanoseconds
}

// IsZero はゼロ値かを返す
func (d Duration) IsZero() bool {
	return d.Years == 0 && d.Months == 0 && d.Weeks == 0 && d.Days == 0 && d.Hours == 0 && d.Minutes == 0 && d.Seconds == 0 && d.Nanoseconds == 0
}

// IsValid は許容範囲を超えていないかを返す
func (d Duration) IsValid() bool {
	return d.Years <= math.MaxInt32 && d.Months <= math.MaxInt32 && d.Weeks <= math.MaxInt32 && d.Days <= math.MaxInt32 && d.Hours <= math.MaxInt32 && d.Minutes <= math.MaxInt32 && d.Seconds <= math.MaxInt32 && d.Nanoseconds <= math.MaxInt32
}

// HasDatePart は日付部を持っているかを返す
func (d Duration) HasDatePart() bool {
	return d.Years > 0 || d.Months > 0 || d.Weeks > 0 || d.Days > 0
}

// HasTimePart は時刻部を持っているかを返す
func (d Duration) HasTimePart() bool {
	return d.Hours > 0 || d.Minutes > 0 || d.Seconds > 0 || d.Nanoseconds > 0
}

func (d Duration) OnlyYMWD() Duration {
	return Duration{
		Negative: d.Negative,
		Years:    d.Years,
		Months:   d.Months,
		Weeks:    d.Weeks,
		Days:     d.Days,
	}
}

func (d Duration) OnlyTime() Duration {
	return Duration{
		Negative:    d.Negative,
		Hours:       d.Hours,
		Minutes:     d.Minutes,
		Seconds:     d.Seconds,
		Nanoseconds: d.Nanoseconds,
	}
}

func (d Duration) GetYMWD() (int, int, int, int) {
	if d.Negative {
		return -1 * int(d.Years), -1 * int(d.Months), -1 * int(d.Weeks), -1 * int(d.Days)
	}
	return int(d.Years), int(d.Months), int(d.Weeks), int(d.Days)
}

func (d Duration) String() string {
	if d.IsZero() {
		if d.Negative {
			return "-PT0S"
		}
		return "PT0S"
	}

	var builder strings.Builder
	if d.Negative {
		builder.WriteByte('-')
	}
	builder.WriteByte('P')
	if d.Years != 0 {
		builder.WriteString(strconv.FormatUint(uint64(d.Years), 10))
		builder.WriteByte('Y')
	}
	if d.Months != 0 {
		builder.WriteString(strconv.FormatUint(uint64(d.Months), 10))
		builder.WriteByte('M')
	}
	if d.Weeks != 0 {
		builder.WriteString(strconv.FormatUint(uint64(d.Weeks), 10))
		builder.WriteByte('W')
	}
	if d.Days != 0 {
		builder.WriteString(strconv.FormatUint(uint64(d.Days), 10))
		builder.WriteByte('D')
	}
	if d.HasTimePart() {
		builder.WriteByte('T')
		if d.Hours != 0 {
			builder.WriteString(strconv.FormatUint(uint64(d.Hours), 10))
			builder.WriteByte('H')
		}
		if d.Minutes != 0 {
			builder.WriteString(strconv.FormatUint(uint64(d.Minutes), 10))
			builder.WriteByte('M')
		}
		if d.Nanoseconds != 0 {
			// 小数以下
			sec, nano := decimal.NewFromUint64(uint64(d.Nanoseconds)).QuoRem(nanosecondsPerSeconds, 0)
			nanoStr := nano.String()
			builder.WriteString(sec.Add(decimal.NewFromUint64(uint64(d.Seconds))).String())
			builder.WriteByte('.')
			builder.Write(bytes.Repeat([]byte{'0'}, 9-len(nanoStr)))
			builder.WriteString(strings.TrimRight(nanoStr, "0"))
			builder.WriteByte('S')
		} else if d.Seconds != 0 {
			builder.WriteString(strconv.FormatUint(uint64(d.Seconds), 10))
			builder.WriteByte('S')
		}
	}

	return builder.String()
}
