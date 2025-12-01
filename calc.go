package iso8601duration

import (
	"math"
	"time"
)

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
func (d Duration) AddToJapan(from time.Time, opts ...Option) time.Time {
	// パラメータ処理
	cfg := config{}
	for _, opt := range opts {
		opt(&cfg)
	}

	years, months, weeks, days := d.GetYMWD()

	// 民法139条 時間により期間を定めた時は、その期間は、即時から起算する
	if !d.HasTimePart() {
		if !d.IsZero() || !cfg.preserveTimeOnZero {
			exclude := false
			if cfg.excludeStartDate == nil {
				// 民法第140条により、起算日を算出 (初日不算入の原則により、翌日から起算する)
				// 00:00:00の場合、初日算入する(民法第140条ただし書)
				exclude = from.Hour() != 0 || from.Minute() != 0 || from.Second() != 0 || from.Nanosecond() != 0
			} else {
				exclude = *cfg.excludeStartDate
			}
			if exclude == d.Negative {
				from = time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, from.Location())
			} else {
				from = time.Date(from.Year(), from.Month(), from.Day()+1, 0, 0, 0, 0, from.Location())
			}
		}
	}

	// 年月を加算し、応当日があるか判断する
	target := from.AddDate(years, months, 0)
	if target.Day() != from.Day() {
		// 応当日がない場合、翌日にする
		// 2025/01/30に1ヶ月加算の場合、AddDateでは2025/03/02(その月の月末 + 差分の日数)が返ってくる
		// 満了日時を2025/02/28 24時とするため、1日(翌日)とする (民法第143条)
		target = time.Date(target.Year(), target.Month(), 1, target.Hour(), target.Minute(), target.Second(), target.Nanosecond(), target.Location())
	}

	// 週と日を加算する
	if days != 0 || weeks != 0 {
		target = target.AddDate(0, 0, days+weeks*7)
	}

	timeDuration := time.Duration(d.Hours)*time.Hour + time.Duration(d.Minutes)*time.Minute + time.Duration(d.Seconds)*time.Second + time.Duration(d.Nanoseconds)
	if d.Negative {
		target = target.Add(-1 * timeDuration)
	} else {
		target = target.Add(timeDuration)
	}
	return target
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
