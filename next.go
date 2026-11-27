package cronexplain

import (
	"errors"
	"time"
)

// How far Next looks before it gives up. Far enough for leap-day schedules,
// short enough that an impossible one (31 Feb) returns instead of spinning.
const searchYears = 5

var (
	errNoMatch = errors.New("no matching time in the next 5 years")
	errReboot  = errors.New("@reboot has no next run")
)

// Next returns the next n firing times strictly after t, in t's location.
// n of 0 returns nil.
//
// A wall time that falls in a DST gap is skipped. A wall time that happens
// twice when clocks fall back is returned twice, once for each instant.
// @reboot, and a schedule that does not fire within five years, return an error.
func (e *Expr) Next(t time.Time, n int) ([]time.Time, error) {
	if n < 0 {
		return nil, errors.New("n must be >= 0")
	}
	if n == 0 {
		return nil, nil
	}
	if e.reboot {
		return nil, errReboot
	}
	out := make([]time.Time, 0, n)
	cur := t
	for i := 0; i < n; i++ {
		nxt, err := e.next(cur)
		if err != nil {
			return nil, err
		}
		out = append(out, nxt)
		cur = nxt
	}
	return out, nil
}

func (e *Expr) next(after time.Time) (time.Time, error) {
	loc := after.Location()
	if loc == nil {
		loc = time.UTC
	}
	t := ceilMinute(after.In(loc))
	horizon := after.AddDate(searchYears, 0, 0)

	for {
		if t.After(horizon) {
			return time.Time{}, errNoMatch
		}
		y, mon, d := t.Date()
		h, m, _ := t.Clock()

		switch {
		case !e.month.has(int(mon)):
			t = later(t, nextMonth(t))
		case !e.matchDay(y, mon, d):
			t = later(t, nextDay(t))
		case !e.hour.has(h):
			t = later(t, nextHour(t))
		case !e.minute.has(m):
			t = later(t, jumpMinutes(t, stepsAfter(e.minute, m)))
		default:
			return t, nil
		}
	}
}

// matchDay applies the vixie rule: when both day-of-month and day-of-week
// are restricted (neither is "*"), a day matches if either field matches.
func (e *Expr) matchDay(y int, mon time.Month, d int) bool {
	// Noon UTC, so the weekday comes from the civil date and not from a DST gap.
	wd := int(time.Date(y, mon, d, 12, 0, 0, 0, time.UTC).Weekday())
	domOK := e.dom.has(d)
	dowOK := e.dow.has(wd)
	switch {
	case e.dom.star && e.dow.star:
		return true
	case e.dom.star:
		return dowOK
	case e.dow.star:
		return domOK
	default:
		return domOK || dowOK
	}
}

// ceilMinute is the next minute boundary strictly after t.
func ceilMinute(t time.Time) time.Time {
	if t.Second() == 0 && t.Nanosecond() == 0 {
		return t.Add(time.Minute)
	}
	base := t.Add(-time.Duration(t.Second())*time.Second - time.Duration(t.Nanosecond()))
	return base.Add(time.Minute)
}

func later(t, n time.Time) time.Time {
	if n.After(t) {
		return n
	}
	return t.Add(time.Minute)
}

func nextMonth(t time.Time) time.Time {
	y, m, _ := t.Date()
	dest := time.Date(y, m+1, 1, 0, 0, 0, 0, t.Location())
	if dest.After(t) && dest.Sub(t) <= 32*24*time.Hour {
		return dest
	}
	return t.Add(time.Minute)
}

func nextDay(t time.Time) time.Time {
	y, m, d := t.Date()
	dest := time.Date(y, m, d+1, 0, 0, 0, 0, t.Location())
	if dest.After(t) && dest.Sub(t) <= 36*time.Hour {
		return dest
	}
	return t.Add(time.Minute)
}

func nextHour(t time.Time) time.Time {
	_, m, _ := t.Clock()
	n := 60 - m
	if n < 1 {
		n = 60
	}
	return jumpMinutes(t, n)
}

// jumpMinutes adds n minutes when the zone offset does not change.
// Across a DST transition it steps one minute, so a skipped hour is not
// invented and a repeated hour is not jumped over.
func jumpMinutes(t time.Time, n int) time.Time {
	if n < 1 {
		n = 1
	}
	dest := t.Add(time.Duration(n) * time.Minute)
	_, off1 := t.Zone()
	_, off2 := dest.Zone()
	if off1 != off2 {
		return t.Add(time.Minute)
	}
	return dest
}

// stepsAfter is how many minutes forward the next matching minute is,
// possibly rolling into the following hour (the caller checks the hour again).
func stepsAfter(f field, minute int) int {
	for n := 1; n <= 60; n++ {
		v := minute + n
		if v > f.hi {
			return n
		}
		if f.has(v) {
			return n
		}
	}
	return 1
}
