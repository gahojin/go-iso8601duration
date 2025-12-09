package iso8601duration

import (
	"errors"
	"fmt"
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
	ErrUnsupportedFractionInMonth = fmt.Errorf("%w: fractions aren't supported for the month-position", ErrUnexpectedInput)
)

func addFrac(base, frac decimal.Decimal) (decimal.Decimal, decimal.Decimal) {
	return base.Add(frac).QuoRem(one, 0)
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

func generateDuration(negative bool, years, months, days, weeks, hours, minutes, seconds decimal.Decimal) (Duration, error) {
	var yearsFrac, monthsFrac, daysFrac, hoursFrac, minutesFrac, secondsFrac decimal.Decimal

	years, yearsFrac = addFrac(years, decimal.Zero)
	months, monthsFrac = addFrac(months, yearsFrac.Mul(monthsPerYear))
	if monthsFrac.GreaterThan(decimal.Zero) {
		// 日に換算出来ないため、月の部分に小数は使用出来ない
		return Duration{}, ErrUnsupportedFractionInMonth
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

func parseValue(s string, startIndex, endIndex int) (decimal.Decimal, error) {
	if startIndex >= endIndex {
		return decimal.Zero, fmt.Errorf("%w: start index %d is out of range", ErrUnexpectedInput, startIndex)
	}
	s = strings.ReplaceAll(s[startIndex+1:endIndex], ",", ".")
	return decimal.NewFromString(s)
}

// ParseString は文字列をISO-8601 Duration書式としてパースし、 Duration を返す
func ParseString(s string) (Duration, error) {
	var (
		err error

		state      parseState = &stateInitial{}
		startIndex            = 0
	)

	for index, c := range s {
		state, startIndex, err = state.NextState(c, s, startIndex, index)
		if err != nil {
			return Duration{}, err
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
