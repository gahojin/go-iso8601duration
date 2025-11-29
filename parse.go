package iso8601duration

import (
	"errors"
	"strconv"
	"strings"

	"github.com/shopspring/decimal"
)

func addFrac(base, frac decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	return base.Add(frac).QuoRem(one, 0)
}

// ParseString は文字列をISO-8601 Duration書式としてパースし、 Duration を返す
func ParseString(s string) (Duration, error) {
	groups := iso8601Pattern.FindStringSubmatch(s)
	if groups == nil {
		return ZeroDuration, ErrBadFormat
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
			return ZeroDuration, err
		}
	}

	years, yearsFrac = addFrac(years, decimal.Zero)
	months, monthsFrac = addFrac(months, yearsFrac.Mul(monthsPerYear))
	if monthsFrac.GreaterThan(decimal.Zero) {
		// 日に換算出来ないため、月の部分に小数は使用出来ない
		return ZeroDuration, errors.Join(ErrBadFormat, errors.New("fractions aren't supported for the month-position"))
	}

	days, daysFrac = addFrac(days, decimal.Zero)
	hours, hoursFrac = addFrac(hours, daysFrac.Mul(hoursPerDay))
	minutes, minutesFrac = addFrac(minutes, hoursFrac.Mul(minutesPerHour))
	seconds, secondsFrac = addFrac(seconds, minutesFrac.Mul(secondsPerMinute))
	nanoSeconds := secondsFrac.Mul(nanosecondsPerSeconds)

	return Duration{
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
