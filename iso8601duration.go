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

	"github.com/shopspring/decimal"
)

// 小数点を持つ数値
const fractionalNumbers = `\d+(?:[\.,]\d+)?`

// 日付部
const datePattern = "(?:(?P<year>" + fractionalNumbers + ")Y)?(?:(?P<month>" + fractionalNumbers + `)M)?(?:(?P<week>\d+)W)?(?:(?P<day>` + fractionalNumbers + ")D)?"

// 時刻部
const timePattern = "T(?:(?P<hour>" + fractionalNumbers + ")H)?(?:(?P<minute>" + fractionalNumbers + ")M)?(?:(?P<second>" + fractionalNumbers + ")S)?"

var (
	// iso8601Pattern ISO-8601 Duration 書式 PnYnMnWnDTnHnMnS
	iso8601Pattern = regexp.MustCompile("^(?P<negative>-)?P(?:" + datePattern + "(?:" + timePattern + ")?)$")

	// ErrBadFormat フォーマット不正エラー
	ErrBadFormat = errors.New("bad format string")

	// ErrUnsupportedNegative マイナス期間未サポート
	ErrUnsupportedNegative = errors.New("unsupported negative duration")

	one                   = decimal.NewFromInt(1)
	monthsPerYear         = decimal.NewFromInt(12)
	hoursPerDay           = decimal.NewFromInt(24)
	minutesPerHour        = decimal.NewFromInt(60)
	secondsPerMinute      = decimal.NewFromInt(60)
	nanosecondsPerSeconds = decimal.NewFromUint64(uint64(time.Second))
)

// 型チェック
var (
	_ encoding.TextMarshaler   = Duration{}
	_ encoding.TextUnmarshaler = (*Duration)(nil)
	_ json.Marshaler           = Duration{}
	_ json.Unmarshaler         = (*Duration)(nil)
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
	return d.Hours > 0 || d.Minutes > 0 || d.Seconds > 0.0 || d.Nanoseconds > 0
}

// Add は期間を合算する
func (d Duration) Add(o Duration) (Duration, bool) {
	// 正規化
	t1, ok := d.Normalize()
	if !ok {
		return d, false
	}
	t2, ok := o.Normalize()
	if !ok {
		return d, false
	}

	// 年や日がオーバーフローしないか確認する
	years1 := t1.Years
	years2 := t2.Years
	if years1 > math.MaxInt32-years2 {
		return d, false
	}
	days1 := t1.Days
	days2 := t2.Days
	if days1 > math.MaxInt32-days2 {
		return d, false
	}

	t1.Years += years2
	t1.Months += o.Months
	t1.Weeks += o.Weeks
	t1.Days += days2
	t1.Hours += o.Hours
	t1.Minutes += o.Minutes
	t1.Seconds += o.Seconds
	t1.Nanoseconds += o.Nanoseconds

	return t1.Normalize()
}

// Negate は期間の符号を反転させた新しい Duration を返す
func (d Duration) Negate() Duration {
	d.Negative = !d.Negative
	return d
}

// Abs は期間の絶対値を返す
func (d Duration) Abs() Duration {
	d.Negative = false
	return d
}

// AddTo は指定日時から期間分経過した日時を返す
func (d Duration) AddTo(from time.Time) time.Time {
	timeDuration := time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second + time.Duration(d.Nanoseconds)

	if d.Negative {
		r := from.AddDate(-1*int(d.Years), -1*int(d.Months), -1*int(d.Weeks*7+d.Days))
		return r.Add(-1 * timeDuration)
	}
	r := from.AddDate(int(d.Years), int(d.Months), int(d.Weeks*7+d.Days))
	return r.Add(timeDuration)
}

// AddToJapan は指定日時から期間分経過した日時を返す (民法第139条,140条,141条,143条に準拠)
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
func (d Duration) AddToJapan(from time.Time) (*time.Time, error) {
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

	timeDuration := time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second + time.Duration(d.Nanoseconds)
	target = target.Add(timeDuration)
	return &target, nil
}

func normalize(base, target *uint32, mod uint32) bool {
	t := *target / mod
	if *base > math.MaxInt32-t {
		// overflow
		return false
	}
	*base = *base + t
	*target = *target % mod
	return true
}

// Normalize は正規化を行う (ex. 24時間を1日/60分を1時間にするなど)
func (d Duration) Normalize() (Duration, bool) {
	r := d

	// 4回正規処理を行う (日 <- 時 <- 分 <- 秒 <- ナノ秒)
	for step := 0; step < 4; step++ {
		// 年
		if ok := normalize(&r.Years, &r.Months, 12); !ok {
			return d, false
		}

		// 日
		if ok := normalize(&r.Days, &r.Hours, 24); !ok {
			return d, false
		}

		// 時
		if ok := normalize(&r.Hours, &r.Minutes, 60); !ok {
			return d, false
		}

		// 分
		if ok := normalize(&r.Minutes, &r.Seconds, 60); !ok {
			return d, false
		}

		// 秒
		if ok := normalize(&r.Seconds, &r.Nanoseconds, 1000*1000*1000); !ok {
			return d, false
		}
	}

	return r, true
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

func (d *Duration) UnmarshalText(data []byte) error {
	t, err := ParseString(string(data))
	if err != nil {
		return err
	}
	if t == nil {
		return ErrBadFormat
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
	if t == nil {
		return ErrBadFormat
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

func addFrac(base, frac decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	return base.Add(frac).QuoRem(one, 0)
}

// ParseString は文字列をISO-8601 Duration書式としてパースし、 Duration を返す
func ParseString(s string) (*Duration, error) {
	groups := iso8601Pattern.FindStringSubmatch(s)
	if groups == nil {
		return nil, ErrBadFormat
	}

	var err error
	var negative bool
	var years, months, days, hours, minutes, seconds decimal.Decimal
	var yearsFrac, monthsFrac, daysFrac, hoursFrac, minutesFrac, secondsFrac decimal.Decimal
	var weeks uint64

	for i, name := range iso8601Pattern.SubexpNames() {
		if i == 0 || name == "" {
			continue
		}

		part := groups[i]
		if part == "" {
			continue
		}
		// パース処理を行えるよう、カンマをドットに変換する
		part = strings.ReplaceAll(part, ",", ".")

		switch name {
		case "negative":
			negative = part == "-"
		case "year":
			years, err = decimal.NewFromString(part)
		case "month":
			months, err = decimal.NewFromString(part)
		case "week":
			weeks, err = strconv.ParseUint(part, 10, 32)
		case "day":
			days, err = decimal.NewFromString(part)
		case "hour":
			hours, err = decimal.NewFromString(part)
		case "minute":
			minutes, err = decimal.NewFromString(part)
		case "second":
			seconds, err = decimal.NewFromString(part)
		}
		if err != nil {
			return nil, err
		}
	}

	years, yearsFrac = addFrac(years, decimal.Zero)
	months, monthsFrac = addFrac(months, yearsFrac.Mul(monthsPerYear))
	if monthsFrac.GreaterThan(decimal.Zero) {
		// 日に換算出来ないため、月の部分に小数は使用出来ない
		return nil, errors.Join(ErrBadFormat, errors.New("fractions aren't supported for the month-position"))
	}

	days, daysFrac = addFrac(days, decimal.Zero)
	hours, hoursFrac = addFrac(hours, daysFrac.Mul(hoursPerDay))
	minutes, minutesFrac = addFrac(minutes, hoursFrac.Mul(minutesPerHour))
	seconds, secondsFrac = addFrac(seconds, minutesFrac.Mul(secondsPerMinute))
	nanoSeconds := secondsFrac.Mul(nanosecondsPerSeconds)

	return &Duration{
		Negative:    negative,
		Years:       uint32(years.IntPart()),
		Months:      uint32(months.IntPart()),
		Weeks:       uint32(weeks),
		Days:        uint32(days.IntPart()),
		Hours:       uint32(hours.IntPart()),
		Minutes:     uint32(minutes.IntPart()),
		Seconds:     uint32(seconds.IntPart()),
		Nanoseconds: uint32(nanoSeconds.IntPart()),
	}, nil
}
