package cronexplain

import (
	"fmt"
	"strconv"
	"strings"
)

var macros = map[string]string{
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
	"@monthly":  "0 0 1 * *",
	"@weekly":   "0 0 * * 0",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@hourly":   "0 * * * *",
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var dowNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}

// Parse parses a five-field cron expression or a standard macro
// (@reboot, @hourly, @daily, @midnight, @weekly, @monthly, @yearly, @annually).
// Names (MON, JAN) are case-insensitive. Sunday is 0 or 7.
func Parse(spec string) (*Expr, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, &ParseError{Msg: "empty expression"}
	}
	fields := strings.Fields(spec)
	if strings.HasPrefix(fields[0], "@") {
		if len(fields) != 1 {
			return nil, &ParseError{Msg: fmt.Sprintf("macro %q does not take other fields", fields[0])}
		}
		name := strings.ToLower(fields[0])
		if name == "@reboot" {
			return &Expr{reboot: true}, nil
		}
		repl, ok := macros[name]
		if !ok {
			return nil, &ParseError{Msg: fmt.Sprintf("unknown macro %q", fields[0])}
		}
		return parseFields(repl)
	}
	if len(fields) != 5 {
		msg := fmt.Sprintf("expected 5 fields, got %d", len(fields))
		if len(fields) == 6 || len(fields) == 7 {
			msg += " (no seconds or year field)"
		}
		return nil, &ParseError{Msg: msg}
	}
	return parseFields(spec)
}

func parseFields(spec string) (*Expr, error) {
	parts := strings.Fields(spec)
	if len(parts) != 5 {
		return nil, &ParseError{Msg: fmt.Sprintf("expected 5 fields, got %d", len(parts))}
	}
	var e Expr
	var err error
	if e.minute, err = parseField(parts[0], 0, 59, nil, false); err != nil {
		return nil, fieldErr("minute", err)
	}
	if e.hour, err = parseField(parts[1], 0, 23, nil, false); err != nil {
		return nil, fieldErr("hour", err)
	}
	if e.dom, err = parseField(parts[2], 1, 31, nil, false); err != nil {
		return nil, fieldErr("day-of-month", err)
	}
	if e.month, err = parseField(parts[3], 1, 12, monthNames, false); err != nil {
		return nil, fieldErr("month", err)
	}
	// Day-of-week accepts 7 as Sunday, so the numeric max is wider than the bit range.
	if e.dow, err = parseField(parts[4], 0, 6, dowNames, true); err != nil {
		return nil, fieldErr("day-of-week", err)
	}
	return &e, nil
}

func fieldErr(name string, err error) error {
	return &ParseError{Field: name, Msg: err.Error()}
}

// parseField parses one cron field.
// lo and hi are the inclusive bit range. When sunday7 is set, the value 7
// is also accepted and stored as Sunday (0).
func parseField(s string, lo, hi int, names map[string]int, sunday7 bool) (field, error) {
	f := field{lo: lo, hi: hi}
	acceptHi := hi
	if sunday7 {
		acceptHi = 7
	}
	if s == "*" {
		f.star = true
		fill(&f, lo, hi, 1, false)
		return f, nil
	}
	for _, part := range strings.Split(s, ",") {
		if err := parsePart(part, lo, hi, acceptHi, names, sunday7, &f); err != nil {
			return field{}, err
		}
	}
	if f.bits == 0 {
		return field{}, fmt.Errorf("empty field")
	}
	return f, nil
}

func parsePart(part string, lo, hi, acceptHi int, names map[string]int, sunday7 bool, dst *field) error {
	part = strings.TrimSpace(part)
	if part == "" {
		return fmt.Errorf("empty list item")
	}
	if part == "*" {
		dst.star = true
		fill(dst, lo, hi, 1, false)
		return nil
	}

	step := 1
	body := part
	if slash := strings.IndexByte(part, '/'); slash >= 0 {
		body = part[:slash]
		raw := part[slash+1:]
		if body == "" || raw == "" || strings.Contains(raw, "/") {
			return fmt.Errorf("invalid term %q", part)
		}
		n, err := strconv.Atoi(raw)
		if err != nil {
			return fmt.Errorf("invalid step %q", raw)
		}
		if n < 1 {
			return fmt.Errorf("step must be at least 1")
		}
		step = n
	}

	if body == "*" {
		fill(dst, lo, acceptHi, step, sunday7)
		return nil
	}

	if dash := strings.IndexByte(body, '-'); dash >= 0 {
		left, right := body[:dash], body[dash+1:]
		if left == "" || right == "" || strings.Contains(right, "-") {
			return fmt.Errorf("invalid range %q", body)
		}
		start, err := parseValue(left, lo, acceptHi, names)
		if err != nil {
			return err
		}
		end, err := parseValue(right, lo, acceptHi, names)
		if err != nil {
			return err
		}
		// Sunday is 0 and also 7. A range that ends on Sunday is read as
		// ending at 7, so FRI-SUN means Friday through Sunday.
		if sunday7 && end == 0 && start > end {
			end = 7
		}
		if start > end {
			return fmt.Errorf("range %s is reversed (ranges do not wrap)", body)
		}
		fill(dst, start, end, step, sunday7)
		return nil
	}

	n, err := parseValue(body, lo, acceptHi, names)
	if err != nil {
		return err
	}
	if strings.Contains(part, "/") {
		// "5/15" means 5 through the top of the field, stepping by 15.
		fill(dst, n, acceptHi, step, sunday7)
		return nil
	}
	dst.add(n, sunday7)
	return nil
}

func fill(dst *field, start, end, step int, sunday7 bool) {
	for v := start; v <= end; v += step {
		dst.add(v, sunday7)
	}
}

func parseValue(s string, lo, acceptHi int, names map[string]int) (int, error) {
	if names != nil {
		if n, ok := names[strings.ToUpper(s)]; ok {
			return n, nil
		}
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		if names != nil {
			return 0, fmt.Errorf("%q is not a number or name", s)
		}
		return 0, fmt.Errorf("%q is not a number", s)
	}
	if n < lo || n > acceptHi {
		return 0, fmt.Errorf("%d out of range (%d-%d)", n, lo, acceptHi)
	}
	return n, nil
}
