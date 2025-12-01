package iso8601duration

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

const (
	stateInitial int = iota
	stateParsingDate
	stateParsingTime
)

var (
	// ErrUnexpectedInput 入力不正エラー
	ErrUnexpectedInput            = errors.New("unexpected input")
	ErrUnsupportedFractionInMonth = errors.New("fractions aren't supported for the month-position")
)

func addFrac(base, frac decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	return base.Add(frac).QuoRem(one, 0)
}

func parseValue(s string, startIndex, endIndex int) (decimal.Decimal, error) {
	if startIndex >= endIndex {
		return decimal.Zero, errors.Join(ErrUnexpectedInput, fmt.Errorf("start index %d is out of range", startIndex))
	}
	s = strings.ReplaceAll(s[startIndex+1:endIndex], ",", ".")
	return decimal.NewFromString(s)
}

// ParseString は文字列をISO-8601 Duration書式としてパースし、 Duration を返す
func ParseString(s string) (Duration, error) {
	var (
		err error

		state      = stateInitial
		negative   = false
		startIndex = 0

		setYear, setMonth, setDay, setWeek, setHour, setMinute, setSecond bool

		years, months, days, weeks, hours, minutes, seconds                  decimal.Decimal
		yearsFrac, monthsFrac, daysFrac, hoursFrac, minutesFrac, secondsFrac decimal.Decimal
	)

	for index, c := range s {
		switch c {
		case '-':
			// 2回 -が出現した場合、エラー
			if negative {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("minus sign appears twice: location: %d", index))
			}
			negative = true
		case 'P':
			if state != stateInitial {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			state = stateParsingDate
			startIndex = index
		case 'T':
			if state != stateParsingDate {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			state = stateParsingTime
			startIndex = index
		case 'Y':
			if state != stateParsingDate || setYear {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			years, err = parseValue(s, startIndex, index)
			setYear = true
			startIndex = index
		case 'M':
			if state == stateParsingDate && !setMonth {
				months, err = parseValue(s, startIndex, index)
				setMonth = true
				startIndex = index
			} else if state == stateParsingTime && !setMinute {
				minutes, err = parseValue(s, startIndex, index)
				setMinute = true
				startIndex = index
			} else {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
		case 'D':
			if state != stateParsingDate || setDay {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			days, err = parseValue(s, startIndex, index)
			setDay = true
			startIndex = index
		case 'W':
			if state != stateParsingDate || setWeek {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			weeks, err = parseValue(s, startIndex, index)
			setWeek = true
			startIndex = index
		case 'H':
			if state != stateParsingTime || setHour {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			hours, err = parseValue(s, startIndex, index)
			setHour = true
			startIndex = index
		case 'S':
			if state != stateParsingTime || setSecond {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			seconds, err = parseValue(s, startIndex, index)
			setSecond = true
			startIndex = index
		default:
			if state != stateParsingDate && state != stateParsingTime {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			if !unicode.IsDigit(c) && c != '.' && c != ',' {
				return ZeroDuration, errors.Join(ErrUnexpectedInput, fmt.Errorf("invalid format: location: %d", index))
			}
			continue
		}

		if err != nil {
			return ZeroDuration, err
		}
	}

	years, yearsFrac = addFrac(years, decimal.Zero)
	months, monthsFrac = addFrac(months, yearsFrac.Mul(monthsPerYear))
	if monthsFrac.GreaterThan(decimal.Zero) {
		// 日に換算出来ないため、月の部分に小数は使用出来ない
		return ZeroDuration, errors.Join(ErrUnexpectedInput, ErrUnsupportedFractionInMonth)
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
		Weeks:       uint32(weeks.IntPart()),
		Days:        uint32(days.IntPart()),
		Hours:       uint32(hours.IntPart()),
		Minutes:     uint32(minutes.IntPart()),
		Seconds:     uint32(seconds.IntPart()),
		Nanoseconds: uint32(nanoSeconds.IntPart()),
	}, nil
}
