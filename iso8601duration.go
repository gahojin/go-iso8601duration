package iso8601duration

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// 小数点を持つ数値
const fractionalNumbers = `\d+(?:[\.,]\d+)?`

// 日付部
const datePattern = `(?:(?P<year>\d+)Y)?(?:(?P<month>\d+)M)?(?:(?P<week>\d+)W)?(?:(?P<day>\d+)D)?`

// 時刻部
const timePattern = "T(?:(?P<hour>" + fractionalNumbers + ")H)?(?:(?P<minute>" + fractionalNumbers + ")M)?(?:(?P<second>" + fractionalNumbers + ")S)?"

var (
	// iso8601Pattern ISO-8601 Duration 書式 PnYnMnWnDTnHnMnS
	iso8601Pattern = regexp.MustCompile("^(?P<negative>-)?P(?:" + datePattern + "(?:" + timePattern + ")?)$")

	// ErrBadFormat フォーマット不正エラー
	ErrBadFormat = errors.New("bad format string")
)

// 型チェック
var (
	_ encoding.TextMarshaler   = Duration{}
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ json.Marshaler           = Duration{}
	_ json.Unmarshaler         = (*Duration)(nil)
)

type Duration struct {
	Negative bool
	Years    uint64
	Months   uint64
	Weeks    uint64
	Days     uint64
	Hours    float64
	Minutes  float64
	Seconds  float64
}

func (d Duration) IsZero() bool {
	return d.Years == 0 && d.Months == 0 && d.Weeks == 0 && d.Days == 0 && d.Hours == 0 && d.Minutes == 0 && d.Seconds == 0
}

func (d Duration) IsValid() bool {
	return d.Years <= math.MaxInt64 && d.Months <= math.MaxInt64 && d.Weeks <= math.MaxInt64 && d.Days <= math.MaxInt64 && d.Hours >= 0.0 && d.Minutes >= 0.0 && d.Seconds >= 0.0
}

func (d Duration) HasTimePart() bool {
	return d.Hours > 0.0 || d.Minutes > 0.0 || d.Seconds > 0.0
}

func (d Duration) Add(from time.Time) time.Time {
	timeDuration := math.Round((d.Hours*60*60 + d.Minutes*60 + d.Seconds) * 1000 * 1000 * 1000)

	if d.Negative {
		r := from.AddDate(-1*int(d.Years), -1*int(d.Months), -1*int(d.Weeks*7+d.Days))
		return r.Add(-1 * time.Duration(timeDuration))
	} else {
		r := from.AddDate(int(d.Years), int(d.Months), int(d.Weeks*7+d.Days))
		return r.Add(time.Duration(timeDuration))
	}
}

// Normalize 正規化する (24時間を1日/60分を1時間にする)
func (d Duration) Normalize() Duration {
	// 秒
	seconds := d.Seconds
	minutes := math.Trunc(seconds / 60)
	if math.MaxFloat64-d.Minutes < minutes {
		// overflow
		minutes = d.Minutes
	} else {
		seconds -= minutes * 60
		minutes += d.Minutes
	}

	// 分
	hours := math.Trunc(minutes / 60)
	if math.MaxFloat64-d.Hours < hours {
		// overflow
		hours = d.Hours
	} else {
		minutes -= hours * 60
		hours += d.Hours
	}

	// 時
	days := uint64(hours / 24)
	if math.MaxUint64-d.Days < days {
		days = d.Days
	} else {
		hours -= float64(days * 24)
		days += d.Days
	}

	// 月
	months := d.Months
	years := months / 12
	if math.MaxUint64-d.Years < years {
		// overflow
		years = d.Years
	} else {
		months -= years * 12
		years += d.Years
	}

	return Duration{
		Years:   years,
		Months:  months,
		Weeks:   d.Weeks,
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}
}

func (d *Duration) String() string {
	if d.IsZero() {
		return "PT0S"
	}

	var builder strings.Builder
	if d.Negative {
		builder.WriteByte('-')
	}
	builder.WriteByte('P')
	if d.Years != 0 {
		builder.WriteString(strconv.FormatUint(d.Years, 10))
		builder.WriteByte('Y')
	}
	if d.Months != 0 {
		builder.WriteString(strconv.FormatUint(d.Months, 10))
		builder.WriteByte('M')
	}
	if d.Weeks != 0 {
		builder.WriteString(strconv.FormatUint(d.Weeks, 10))
		builder.WriteByte('W')
	}
	if d.Days != 0 {
		builder.WriteString(strconv.FormatUint(d.Days, 10))
		builder.WriteByte('D')
	}
	if d.HasTimePart() {
		builder.WriteByte('T')
		if d.Hours > 0.0 && isFinite(d.Hours) {
			builder.WriteString(strconv.FormatFloat(d.Hours, 'f', -1, 64))
			builder.WriteByte('H')
		}
		if d.Minutes > 0.0 && isFinite(d.Minutes) {
			builder.WriteString(strconv.FormatFloat(d.Minutes, 'f', -1, 64))
			builder.WriteByte('M')
		}
		if d.Seconds > 0.0 && isFinite(d.Seconds) {
			builder.WriteString(strconv.FormatFloat(d.Seconds, 'f', -1, 64))
			builder.WriteByte('S')
		}
	}

	return builder.String()
}

func (d *Duration) UnmarshalText(data []byte) error {
	t, err := ParseString(string(data))
	if err != nil {
		return err
	}
	*d = *t
	return nil
}

func (d Duration) MarshalText() ([]byte, error) {
	return []byte(d.String()), nil
}

func (d *Duration) UnmarshalJSON(data []byte) error {
	dec := json.NewDecoder(bytes.NewBuffer(data))
	var s string
	if err := dec.Decode(&s); err != nil {
		return err
	}
	t, err := ParseString(s)
	if err != nil {
		return err
	}
	*d = *t
	return nil
}

func (d Duration) MarshalJSON() ([]byte, error) {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	s := d.String()
	err := enc.Encode(s)
	if err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

func isFinite(n float64) bool {
	return !math.IsNaN(n) && !math.IsInf(n, 0)
}

func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.ReplaceAll(s, ",", "."), 64)
}

func ParseString(s string) (*Duration, error) {
	groups := iso8601Pattern.FindStringSubmatch(s)
	if groups == nil {
		return nil, ErrBadFormat
	}

	var err error
	d := &Duration{}

	for i, name := range iso8601Pattern.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}

		part := groups[i]
		if part == "" {
			continue
		}

		switch name {
		case "negative":
			d.Negative = part == "-"
		case "year":
			d.Years, err = strconv.ParseUint(part, 10, 64)
		case "month":
			d.Months, err = strconv.ParseUint(part, 10, 64)
		case "week":
			d.Weeks, err = strconv.ParseUint(part, 10, 64)
		case "day":
			d.Days, err = strconv.ParseUint(part, 10, 64)
		case "hour":
			d.Hours, err = parseFloat(part)
		case "minute":
			d.Minutes, err = parseFloat(part)
		case "second":
			d.Seconds, err = parseFloat(part)
		}

		if err != nil {
			return nil, err
		}
	}

	return d, nil
}
