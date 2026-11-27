// Package cronexplain parses cron expressions and describes when they run.
//
// A schedule is five fields, the same set vixie cron uses:
//
//	minute hour day-of-month month day-of-week
//
// or one of the @hourly, @daily, @midnight, @weekly, @monthly, @yearly,
// @annually, and @reboot macros.
package cronexplain

// ParseError is a bad expression. Field is "minute", "hour",
// "day-of-month", "month", or "day-of-week", and empty when the
// whole expression is the problem.
type ParseError struct {
	Field string
	Msg   string
}

func (e *ParseError) Error() string {
	if e.Field == "" {
		return e.Msg
	}
	return e.Field + ": " + e.Msg
}

// Expr is a parsed schedule. The zero value is not useful; use Parse.
// A value from Parse is immutable and safe to share across goroutines.
type Expr struct {
	minute, hour, dom, month, dow field
	reboot                        bool
}

// field is the set of accepted values in [lo, hi].
// star is set only for a literal "*", which is what the day-of-month /
// day-of-week rule keys off (a field of "0-6" is not the same as "*").
type field struct {
	lo, hi int
	bits   uint64
	star   bool
}

func (f field) has(v int) bool {
	if v < 0 || v > 63 {
		return false
	}
	return f.bits&(uint64(1)<<uint(v)) != 0
}

func (f *field) set(v int) {
	f.bits |= uint64(1) << uint(v)
}

func (f *field) add(v int, sunday7 bool) {
	// 7 is Sunday, same as 0. Callers that accept it pass sunday7.
	if sunday7 && v == 7 {
		v = 0
	}
	f.set(v)
}

func (f field) values() []int {
	out := make([]int, 0, 4)
	for v := f.lo; v <= f.hi; v++ {
		if f.has(v) {
			out = append(out, v)
		}
	}
	return out
}
