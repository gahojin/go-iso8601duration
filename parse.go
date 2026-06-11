package iso8601duration

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/shopspring/decimal"
)

type parseState interface {
	NextState(c rune, text string, startIndex, index int) (parseState, int, error)
	Generate() (Duration, error)
}

type stateInitial struct {
	negative bool
}

type stateParsingDate struct {
	*stateInitial

	setYear  bool
	setMonth bool
	setDay   bool
	setWeek  bool

	years  decimal.Decimal
	months decimal.Decimal
	days   decimal.Decimal
	weeks  decimal.Decimal
}

type stateParsingTime struct {
	*stateParsingDate

	setHour   bool
	setMinute bool
	setSecond bool

	hours   decimal.Decimal
	minutes decimal.Decimal
	seconds decimal.Decimal
}

var (
	// ErrUnexpectedInput 入力不正エラー
	ErrUnexpectedInput            = errors.New("unexpected input")
	ErrUnsupportedFractionInMonth = fmt.Errorf("fractions aren't supported for the month-position: %w", ErrUnexpectedInput)
	ErrOverflow                   = errors.New("overflow")
)

func toUint32(value int64, fieldName string) (uint32, error) {
	if value < 0 || value > math.MaxUint32 {
		return 0, fmt.Errorf("%s value %d overflows uint32 range (0 to %d): %w", fieldName, value, math.MaxUint32, ErrOverflow)
	}
	return uint32(value), nil
}

func addFrac(base, frac decimal.Decimal, fieldName string) (uint32, decimal.Decimal, error) {
	value, frac := base.Add(frac).QuoRem(one, 0)

	// overflow check
	v, err := toUint32(value.IntPart(), fieldName)
	if err != nil {
		return 0, frac, err
	}
	return v, frac, nil
}

func parseElement(s string, startIndex, index int, flag *bool, value *decimal.Decimal) error {
	if *flag {
		return fmt.Errorf("%w: invalid format: position: %d", ErrUnexpectedInput, index)
	}

	tmp, err := parseValue(s, startIndex, index)
	if err != nil {
		return err
	}

	*value = tmp
	*flag = true
	return nil
}

func generateDuration(negative bool, years, months, days, weeks, hours, minutes, seconds decimal.Decimal) (result Duration, err error) {
	var yearsFrac, monthsFrac, daysFrac, hoursFrac, minutesFrac, secondsFrac decimal.Decimal
	var yearsValue, monthsValue, weeksValue, daysValue, hoursValue, minutesValue, secondsValue, nanoSecondsValue uint32

	if yearsValue, yearsFrac, err = addFrac(years, decimal.Zero, "years"); err != nil {
		return
	}
	if monthsValue, monthsFrac, err = addFrac(months, yearsFrac.Mul(monthsPerYear), "months"); err != nil {
		return
	}
	if monthsFrac.GreaterThan(decimal.Zero) {
		// 日に換算出来ないため、月の部分に小数は使用出来ない
		return result, ErrUnsupportedFractionInMonth
	}
	if weeksValue, err = toUint32(weeks.IntPart(), "weeks"); err != nil {
		return
	}

	if daysValue, daysFrac, err = addFrac(days, decimal.Zero, "days"); err != nil {
		return
	}
	if hoursValue, hoursFrac, err = addFrac(hours, daysFrac.Mul(hoursPerDay), "hours"); err != nil {
		return
	}
	if minutesValue, minutesFrac, err = addFrac(minutes, hoursFrac.Mul(minutesPerHour), "minutes"); err != nil {
		return
	}
	if secondsValue, secondsFrac, err = addFrac(seconds, minutesFrac.Mul(secondsPerMinute), "seconds"); err != nil {
		return
	}
	if nanoSecondsValue, err = toUint32(secondsFrac.Mul(nanosecondsPerSeconds).IntPart(), "nanoseconds"); err != nil {
		return
	}

	return Duration{
		Negative:    negative,
		Years:       yearsValue,
		Months:      monthsValue,
		Weeks:       weeksValue,
		Days:        daysValue,
		Hours:       hoursValue,
		Minutes:     minutesValue,
		Seconds:     secondsValue,
		Nanoseconds: nanoSecondsValue,
	}, nil
}

func parseValue(s string, startIndex, endIndex int) (decimal.Decimal, error) {
	if startIndex >= endIndex {
		return decimal.Zero, fmt.Errorf("%w: start index %d is out of range", ErrUnexpectedInput, startIndex)
	}
	s = strings.ReplaceAll(s[startIndex+1:endIndex], ",", ".")
	return decimal.NewFromString(s)
}

// ParseString は文字列をISO-8601 Duration書式としてパースし、 Duration を返す
func ParseString(s string) (result Duration, err error) {
	var (
		state      parseState = &stateInitial{}
		startIndex            = 0
	)

	for index, c := range s {
		if state, startIndex, err = state.NextState(c, s, startIndex, index); err != nil {
			return
		}
	}

	return state.Generate()
}

func (s *stateInitial) NextState(c rune, _ string, _, index int) (parseState, int, error) {
	if c == 'P' {
		return &stateParsingDate{stateInitial: s}, index, nil
	}
	if s.negative {
		// 2回 -が出現した場合、エラー
		return nil, index, fmt.Errorf("%w: minus sign appears twice: position: %d", ErrUnexpectedInput, index)
	}
	s.negative = true
	return s, index, nil
}

func (s *stateInitial) Generate() (Duration, error) {
	return Duration{}, fmt.Errorf("%w: invalid format", ErrUnexpectedInput)
}

func (s *stateParsingDate) NextState(c rune, text string, startIndex, index int) (parseState, int, error) {
	var err error
	switch c {
	case 'T':
		return &stateParsingTime{stateParsingDate: s}, index, nil
	case 'Y':
		err = parseElement(text, startIndex, index, &s.setYear, &s.years)
	case 'M':
		err = parseElement(text, startIndex, index, &s.setMonth, &s.months)
	case 'D':
		err = parseElement(text, startIndex, index, &s.setDay, &s.days)
	case 'W':
		err = parseElement(text, startIndex, index, &s.setWeek, &s.weeks)
	default:
		if !unicode.IsDigit(c) && c != '.' && c != ',' {
			err = fmt.Errorf("%w: invalid format: position: %d", ErrUnexpectedInput, index)
		}
		return s, startIndex, err
	}
	return s, index, err
}

func (s *stateParsingDate) Generate() (Duration, error) {
	return generateDuration(s.negative, s.years, s.months, s.days, s.weeks, decimal.Zero, decimal.Zero, decimal.Zero)
}

func (s *stateParsingTime) NextState(c rune, text string, startIndex, index int) (parseState, int, error) {
	var err error
	switch c {
	case 'H':
		err = parseElement(text, startIndex, index, &s.setHour, &s.hours)
	case 'M':
		err = parseElement(text, startIndex, index, &s.setMinute, &s.minutes)
	case 'S':
		err = parseElement(text, startIndex, index, &s.setSecond, &s.seconds)
	default:
		if !unicode.IsDigit(c) && c != '.' && c != ',' {
			err = fmt.Errorf("%w: invalid format: position: %d", ErrUnexpectedInput, index)
		}
		return s, startIndex, err
	}
	return s, index, err
}

func (s *stateParsingTime) Generate() (Duration, error) {
	return generateDuration(s.negative, s.years, s.months, s.days, s.weeks, s.hours, s.minutes, s.seconds)
}
