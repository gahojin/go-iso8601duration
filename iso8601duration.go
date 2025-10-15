package iso8601duration

import (
	"bytes"
	"encoding"
	"encoding/json"
	"errors"
	"fmt"
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

	// ErrUnsupportedNegative マイナス期間未サポート
	ErrUnsupportedNegative = errors.New("unsupported negative duration")
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

// IsZero ゼロ値か
func (d Duration) IsZero() bool {
	return d.Years == 0 && d.Months == 0 && d.Weeks == 0 && d.Days == 0 && d.Hours == 0 && d.Minutes == 0 && d.Seconds == 0
}

// IsValid 許容範囲を超えていないか
func (d Duration) IsValid() bool {
	return d.Years <= math.MaxInt64 && d.Months <= math.MaxInt64 && d.Weeks <= math.MaxInt64 && d.Days <= math.MaxInt64 && isFinite(d.Hours) && d.Hours >= 0.0 && isFinite(d.Minutes) && d.Minutes >= 0.0 && isFinite(d.Seconds) && d.Seconds >= 0.0
}

// HasDatePart 日付部を持っているか
func (d Duration) HasDatePart() bool {
	return d.Years >= 0 || d.Months > 0 || d.Weeks > 0 || d.Days > 0
}

// HasTimePart 時刻部を持っているか
func (d Duration) HasTimePart() bool {
	return d.Hours > 0.0 || d.Minutes > 0.0 || d.Seconds > 0.0
}

// Add 指定日時から期間分経過した日時を返す
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

// AddJapan 指定日時から期間分経過した日時を返す (民法第139条,140条,141条,143条に準拠)
// 計算方法が未定義であるため、マイナス期間はサポートしない
// 民法第139条
//   - 時間によって期間を定めたときは、その期間は、即時から起算する。
//
// 民法第140条
//   - 日、週、月又は年によって期間を定めたときは、期間の初日は、算入しない。
//     ただし、その期間が午前零時から始まるときは、この限りでない。
//
// 民法第141条
//   - 前条の場合には、期間は、その末日の終了をもって満了する。
//
// 民法第143条
//   - 週、月又は年によって期間を定めたときは、その期間は、暦に従って計算する。
//   - 週、月又は年の初めから期間を起算しないときは、その期間は、最後の週、月又は年においてその起算日に応当する日の前日に満了する。
//     ただし、月又は年によって期間を定めた場合において、最後の月に応当する日がないときは、その月の末日に満了する。
func (d Duration) AddJapan(from time.Time) (*time.Time, error) {
	// マイナス期間はサポートしない
	if d.Negative {
		return nil, ErrUnsupportedNegative
	}

	// 民法139条 時間により期間を定めた時は、その期間は、即時から起算する
	target := from
	if !d.HasTimePart() {
		isStartOfDay := from.Hour() == 0 && from.Minute() == 0 && from.Second() == 0 && from.Nanosecond() == 0
		// 民法第140条により、起算日を算出 (初日不算入の原則により、翌日から起算する)
		// 00:00:00の場合、初日算入する(民法第140条ただし書)
		if !isStartOfDay {
			target = time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, from.Location())
		}
	}

	// 年月を加算し、応当日があるか判断する
	fmt.Printf("year = %v\n", d.Years)
	target = target.AddDate(int(d.Years), int(d.Months), 0)
	if target.Day() != from.Day() {
		// 応当日がない場合、翌日にする
		// 2025/01/30に1ヶ月加算の場合、AddDateでは2025/03/02(その月の月末 + 差分の日数)が返ってくる
		// 満了日時を2025/02/28 24時とするため、1日(翌日)とする (民法第143条)
		target = time.Date(target.Year(), target.Month(), 1, target.Hour(), target.Minute(), target.Second(), target.Nanosecond(), target.Location())
	}

	// 週と日を加算する
	if d.Days > 0 || d.Weeks > 0 {
		target = target.AddDate(0, 0, int(d.Days+d.Weeks*7))
	}

	timeDuration := math.Round((d.Hours*60*60 + d.Minutes*60 + d.Seconds) * 1000 * 1000 * 1000)
	target = target.Add(time.Duration(timeDuration))
	return &target, nil
}

// Normalize 正規化する (24時間を1日/60分を1時間にする)
func (d Duration) Normalize() (Duration, bool) {
	// 秒
	seconds := d.Seconds
	minutes := math.Trunc(seconds / 60)
	if math.MaxInt64-d.Minutes < minutes {
		// overflow
		return d, false
	}
	seconds -= minutes * 60
	minutes += d.Minutes

	// 分
	hours := math.Trunc(minutes / 60)
	if math.MaxInt64-d.Hours < hours {
		// overflow
		return d, false
	}
	minutes -= hours * 60
	hours += d.Hours

	// 時
	days := uint64(hours / 24)
	if math.MaxInt64-d.Days < days {
		// overflow
		return d, false
	}
	hours -= float64(days * 24)
	days += d.Days

	// 月
	months := d.Months
	years := months / 12
	if math.MaxInt64-d.Years < years {
		// overflow
		return d, false
	}
	months -= years * 12
	years += d.Years

	return Duration{
		Years:   years,
		Months:  months,
		Weeks:   d.Weeks,
		Days:    days,
		Hours:   hours,
		Minutes: minutes,
		Seconds: seconds,
	}, true
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

// ParseString 文字列をISO-8601 Duration書式としてパースする
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
