package cronexplain

import (
	"fmt"
	"strconv"
	"strings"
)

var longMonth = [...]string{
	"",
	"January", "February", "March", "April", "May", "June",
	"July", "August", "September", "October", "November", "December",
}

var longDow = [...]string{
	"Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday", "Sunday",
}

// Describe returns a plain-English summary of the schedule.
func (e *Expr) Describe() string {
	if e.reboot {
		return "At startup"
	}
	return joinClock(clockPhrase(e.minute, e.hour), calendarPhrase(e.dom, e.month, e.dow))
}

func joinClock(clock, cal string) string {
	if cal == "" {
		if rest, ok := strings.CutPrefix(clock, "At "); ok {
			return "Every day at " + rest
		}
		return clock
	}
	if rest, ok := strings.CutPrefix(cal, "every day in "); ok {
		if at, ok := strings.CutPrefix(clock, "At "); ok {
			return "Every day at " + at + " in " + rest
		}
		return clock + " in " + rest
	}
	return clock + " on " + cal
}

func clockPhrase(minute, hour field) string {
	switch {
	case minute.coversAll() && hour.coversAll():
		return "Every minute"
	case hour.coversAll():
		return minutesAllDay(minute)
	case minute.coversAll():
		return everyMinuteIn(hour)
	}
	if step, ok := minute.fullStep(); ok {
		return everyNIn(step, hour)
	}
	if m, ok := minute.single(); ok {
		if h0, h1, span := hour.oneSpan(); span && h1 > h0 {
			if m == 0 {
				return fmt.Sprintf("Every hour from %02d:00 through %02d:00", h0, h1)
			}
			return fmt.Sprintf("Every hour at minute %d from %02d:00 through %02d:00", m, h0, h1)
		}
	}
	if times, ok := listTimes(minute, hour, 6); ok {
		return "At " + joinAnd(times)
	}
	return upperFirst(minutePhrase(minute) + " of " + hourPhrase(hour))
}

func minutesAllDay(minute field) string {
	if minute.coversAll() {
		return "Every minute"
	}
	if step, ok := minute.fullStep(); ok {
		if step == 1 {
			return "Every minute"
		}
		return fmt.Sprintf("Every %d minutes", step)
	}
	if m, ok := minute.single(); ok {
		if m == 0 {
			return "Every hour"
		}
		return fmt.Sprintf("Every hour at minute %d", m)
	}
	// A long uniform step reads better as "every n" than as a list.
	// Short ones ("15,45") stay as a list.
	if start, step, end, ok := minute.stride(); ok && len(minute.values()) > 4 {
		if step == 1 {
			return fmt.Sprintf("Every minute from %d through %d", start, end)
		}
		return fmt.Sprintf("Every %d minutes from minute %d through %d", step, start, end)
	}
	return "Every hour at " + minutePhrase(minute)
}

func everyMinuteIn(hour field) string {
	if h0, h1, ok := hour.oneSpan(); ok {
		return fmt.Sprintf("Every minute from %02d:00 through %02d:59", h0, h1)
	}
	return "Every minute during " + hourPhrase(hour)
}

func everyNIn(step int, hour field) string {
	word := fmt.Sprintf("Every %d minutes", step)
	if step == 1 {
		word = "Every minute"
	}
	if h0, h1, ok := hour.oneSpan(); ok {
		if h0 == h1 {
			return fmt.Sprintf("%s during hour %d", word, h0)
		}
		return fmt.Sprintf("%s from %02d:00 through %02d:59", word, h0, h1)
	}
	return word + " during " + hourPhrase(hour)
}

func listTimes(minute, hour field, limit int) ([]string, bool) {
	ms := minute.values()
	hs := hour.values()
	if len(ms) == 0 || len(hs) == 0 || len(ms)*len(hs) > limit {
		return nil, false
	}
	out := make([]string, 0, len(ms)*len(hs))
	for _, h := range hs {
		for _, m := range ms {
			out = append(out, fmt.Sprintf("%02d:%02d", h, m))
		}
	}
	return out, true
}

func minutePhrase(f field) string {
	if f.coversAll() {
		return "every minute"
	}
	if step, ok := f.fullStep(); ok {
		if step == 1 {
			return "every minute"
		}
		return fmt.Sprintf("every %d minutes", step)
	}
	if m, ok := f.single(); ok {
		return fmt.Sprintf("minute %d", m)
	}
	if start, step, end, ok := f.stride(); ok && len(f.values()) > 4 {
		if step == 1 {
			return fmt.Sprintf("every minute from %d through %d", start, end)
		}
		return fmt.Sprintf("every %d minutes from minute %d through %d", step, start, end)
	}
	return "minutes " + joinSpans(f.spans(), strconv.Itoa)
}

func hourPhrase(f field) string {
	if f.coversAll() {
		return "every hour"
	}
	if h0, h1, ok := f.oneSpan(); ok {
		if h0 == h1 {
			return fmt.Sprintf("hour %d", h0)
		}
		return fmt.Sprintf("hours %d through %d", h0, h1)
	}
	return "hours " + joinSpans(f.spans(), strconv.Itoa)
}

func calendarPhrase(dom, month, dow field) string {
	mon := ""
	if !month.star && !month.coversAll() {
		mon = monthPhrase(month)
	}
	withMonth := func(s string) string {
		if mon == "" {
			return s
		}
		if s == "" {
			return "every day in " + mon
		}
		return s + " in " + mon
	}

	switch {
	case dom.star && dow.star:
		return withMonth("")
	case dom.star:
		return withMonth(dowPhrase(dow))
	case dow.star:
		if d, ok := dom.single(); ok {
			if m, ok := month.single(); ok {
				return fmt.Sprintf("%s %d", longMonth[m], d)
			}
		}
		return withMonth(domPhrase(dom))
	default:
		// Both day fields are restricted. Vixie cron fires when either matches,
		// so a side that lists every day makes the schedule fire every day.
		if dom.coversAll() || dow.coversAll() {
			return withMonth("")
		}
		if d, ok := dom.single(); ok {
			if m, ok := month.single(); ok {
				return fmt.Sprintf("%s %d, or on %s", longMonth[m], d, dowPhrase(dow))
			}
		}
		return withMonth(domPhrase(dom) + ", or on " + dowPhrase(dow))
	}
}

func domPhrase(f field) string {
	spans := f.spans()
	if len(spans) == 1 && spans[0][0] == spans[0][1] {
		return fmt.Sprintf("day %d of the month", spans[0][0])
	}
	return "days " + joinSpans(spans, strconv.Itoa) + " of the month"
}

func monthPhrase(f field) string {
	return joinSpans(f.spans(), func(v int) string { return longMonth[v] })
}

func dowPhrase(f field) string {
	return joinSpans(dowSpans(f), func(v int) string { return longDow[v] })
}

// dowSpans lists weekday ranges. When Saturday and Sunday are both set,
// Sunday is placed at the end of the week so FRI-SUN reads as
// "Friday through Sunday" rather than "Sunday and Friday through Saturday".
func dowSpans(f field) [][2]int {
	if !f.has(0) || !f.has(6) {
		return f.spans()
	}
	var out [][2]int
	start := -1
	for v := 1; v <= 7; v++ {
		bit := v
		if v == 7 {
			bit = 0
		}
		if f.has(bit) {
			if start < 0 {
				start = v
			}
			continue
		}
		if start >= 0 {
			out = append(out, [2]int{start, v - 1})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, 7})
	}
	return out
}

func joinSpans(spans [][2]int, name func(int) string) string {
	parts := make([]string, len(spans))
	for i, sp := range spans {
		if sp[0] == sp[1] {
			parts[i] = name(sp[0])
			continue
		}
		parts[i] = name(sp[0]) + " through " + name(sp[1])
	}
	return joinAnd(parts)
}

func joinAnd(parts []string) string {
	switch len(parts) {
	case 0:
		return ""
	case 1:
		return parts[0]
	case 2:
		return parts[0] + " and " + parts[1]
	default:
		return strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
	}
}

func upperFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func (f field) coversAll() bool {
	for v := f.lo; v <= f.hi; v++ {
		if !f.has(v) {
			return false
		}
	}
	return true
}

func (f field) spans() [][2]int {
	var out [][2]int
	start := -1
	for v := f.lo; v <= f.hi; v++ {
		if f.has(v) {
			if start < 0 {
				start = v
			}
			continue
		}
		if start >= 0 {
			out = append(out, [2]int{start, v - 1})
			start = -1
		}
	}
	if start >= 0 {
		out = append(out, [2]int{start, f.hi})
	}
	return out
}

func (f field) single() (int, bool) {
	v := -1
	for n := f.lo; n <= f.hi; n++ {
		if !f.has(n) {
			continue
		}
		if v >= 0 {
			return 0, false
		}
		v = n
	}
	if v < 0 {
		return 0, false
	}
	return v, true
}

func (f field) oneSpan() (lo, hi int, ok bool) {
	sp := f.spans()
	if len(sp) != 1 {
		return 0, 0, false
	}
	return sp[0][0], sp[0][1], true
}

func (f field) stride() (start, step, end int, ok bool) {
	vals := f.values()
	if len(vals) < 2 {
		return 0, 0, 0, false
	}
	step = vals[1] - vals[0]
	if step < 1 {
		return 0, 0, 0, false
	}
	for i := 1; i < len(vals); i++ {
		if vals[i]-vals[i-1] != step {
			return 0, 0, 0, false
		}
	}
	return vals[0], step, vals[len(vals)-1], true
}

func (f field) fullStep() (int, bool) {
	start, step, end, ok := f.stride()
	if !ok || start != f.lo || end+step <= f.hi {
		return 0, false
	}
	return step, true
}
