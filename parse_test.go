package cronexplain

import (
	"errors"
	"strings"
	"testing"
)

func TestParseErrors(t *testing.T) {
	tests := []struct {
		in    string
		field string
		want  string
	}{
		{"", "", "empty expression"},
		{"   ", "", "empty expression"},
		{"* * * *", "", "expected 5 fields, got 4"},
		{"* * * * * *", "", "expected 5 fields, got 6 (no seconds or year field)"},
		{"* * * * * * *", "", "expected 5 fields, got 7 (no seconds or year field)"},
		{"nope", "", "expected 5 fields, got 1"},
		{"@foo", "", `unknown macro "@foo"`},
		{"@daily extra", "", `macro "@daily" does not take other fields`},
		{"60 * * * *", "minute", "minute: 60 out of range (0-59)"},
		{"-1 * * * *", "minute", `minute: invalid range "-1"`},
		{"*/0 * * * *", "minute", "minute: step must be at least 1"},
		{"*/ * * * *", "minute", `minute: invalid term "*/"`},
		{"1,,2 * * * *", "minute", "minute: empty list item"},
		{"5-1 * * * *", "minute", "minute: range 5-1 is reversed (ranges do not wrap)"},
		{"1-5-9 * * * *", "minute", `minute: invalid range "1-5-9"`},
		{"abc * * * *", "minute", `minute: "abc" is not a number`},
		{"* 24 * * *", "hour", "hour: 24 out of range (0-23)"},
		{"* * 0 * *", "day-of-month", "day-of-month: 0 out of range (1-31)"},
		{"* * 32 * *", "day-of-month", "day-of-month: 32 out of range (1-31)"},
		{"* * * 0 *", "month", "month: 0 out of range (1-12)"},
		{"* * * 13 *", "month", "month: 13 out of range (1-12)"},
		{"* * * FOO *", "month", `month: "FOO" is not a number or name`},
		{"* * * * 8", "day-of-week", "day-of-week: 8 out of range (0-7)"},
		{"* * * * FUN", "day-of-week", `day-of-week: "FUN" is not a number or name`},
		{"* * * * FRI-MON", "day-of-week", "day-of-week: range FRI-MON is reversed (ranges do not wrap)"},
		{"* * * * ?", "day-of-week", `day-of-week: "?" is not a number or name`},
		{"* * L * *", "day-of-month", `day-of-month: "L" is not a number`},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			_, err := Parse(tt.in)
			if err == nil {
				t.Fatal("expected an error")
			}
			if err.Error() != tt.want {
				t.Fatalf("got %q, want %q", err.Error(), tt.want)
			}
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("error type %T", err)
			}
			if pe.Field != tt.field {
				t.Fatalf("field %q, want %q", pe.Field, tt.field)
			}
		})
	}
}

func TestParseFields(t *testing.T) {
	tests := []struct {
		in     string
		minute []int
		hour   []int
		dom    []int
		month  []int
		dow    []int
		stars  string // "mhdMy", '*' or '.' per field
	}{
		{
			in:     "* * * * *",
			minute: seq(0, 59), hour: seq(0, 23), dom: seq(1, 31), month: seq(1, 12), dow: seq(0, 6),
			stars: "*****",
		},
		{
			in:     "*/15 9-17 * * 1-5",
			minute: []int{0, 15, 30, 45}, hour: seq(9, 17), dom: seq(1, 31), month: seq(1, 12), dow: seq(1, 5),
			stars: "..**.",
		},
		{
			in:     "5/15 * * * *",
			minute: []int{5, 20, 35, 50}, hour: seq(0, 23), dom: seq(1, 31), month: seq(1, 12), dow: seq(0, 6),
			stars: ".****",
		},
		{
			in:     "1-10/2 * * JAN,MAR *",
			minute: []int{1, 3, 5, 7, 9}, hour: seq(0, 23), dom: seq(1, 31), month: []int{1, 3}, dow: seq(0, 6),
			stars: ".**.*",
		},
		{
			in:     "0 0 1,15 * MON-FRI",
			minute: []int{0}, hour: []int{0}, dom: []int{1, 15}, month: seq(1, 12), dow: seq(1, 5),
			stars: "...*.",
		},
		{
			in:     "0 9 * jan mon",
			minute: []int{0}, hour: []int{9}, dom: seq(1, 31), month: []int{1}, dow: []int{1},
			stars: "..*..",
		},
		{
			in:     "0 0 * * 7",
			minute: []int{0}, hour: []int{0}, dom: seq(1, 31), month: seq(1, 12), dow: []int{0},
			stars: "..**.",
		},
		{
			in:     "0 0 * * SUN",
			minute: []int{0}, hour: []int{0}, dom: seq(1, 31), month: seq(1, 12), dow: []int{0},
			stars: "..**.",
		},
		{
			in:     "0 0 * * 0-7",
			minute: []int{0}, hour: []int{0}, dom: seq(1, 31), month: seq(1, 12), dow: seq(0, 6),
			stars: "..**.",
		},
		{
			in:     "0 0 * * 5-7",
			minute: []int{0}, hour: []int{0}, dom: seq(1, 31), month: seq(1, 12), dow: []int{0, 5, 6},
			stars: "..**.",
		},
		{
			in:     "0 0 * * FRI-SUN",
			minute: []int{0}, hour: []int{0}, dom: seq(1, 31), month: seq(1, 12), dow: []int{0, 5, 6},
			stars: "..**.",
		},
		{
			in:     "  */15 \t 0 1 2 3 ",
			minute: []int{0, 15, 30, 45}, hour: []int{0}, dom: []int{1}, month: []int{2}, dow: []int{3},
			stars: ".....",
		},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			e, err := Parse(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			if e.reboot {
				t.Fatal("unexpected reboot")
			}
			assertVals(t, "minute", e.minute, tt.minute)
			assertVals(t, "hour", e.hour, tt.hour)
			assertVals(t, "dom", e.dom, tt.dom)
			assertVals(t, "month", e.month, tt.month)
			assertVals(t, "dow", e.dow, tt.dow)
			gotStars := starString(e)
			if gotStars != tt.stars {
				t.Fatalf("stars %s, want %s", gotStars, tt.stars)
			}
		})
	}
}

func TestParseMacros(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"@yearly", "0 0 1 1 *"},
		{"@annually", "0 0 1 1 *"},
		{"@monthly", "0 0 1 * *"},
		{"@weekly", "0 0 * * 0"},
		{"@daily", "0 0 * * *"},
		{"@midnight", "0 0 * * *"},
		{"@hourly", "0 * * * *"},
		{"@YEARLY", "0 0 1 1 *"},
		{"  @Daily  ", "0 0 * * *"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got, err := Parse(tt.in)
			if err != nil {
				t.Fatal(err)
			}
			want, err := Parse(tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if !sameSchedule(got, want) {
				t.Fatalf("%s does not match %s", tt.in, tt.want)
			}
		})
	}
}

func TestParseReboot(t *testing.T) {
	for _, in := range []string{"@reboot", "@REBOOT", "  @reboot"} {
		e, err := Parse(in)
		if err != nil {
			t.Fatalf("%s: %v", in, err)
		}
		if !e.reboot {
			t.Fatalf("%s: reboot flag not set", in)
		}
	}
}

func TestParseLargeStep(t *testing.T) {
	e, err := Parse("0/9223372036854775807 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	assertVals(t, "minute", e.minute, []int{0})

	e, err = Parse("*/9223372036854775807 * * * *")
	if err != nil {
		t.Fatal(err)
	}
	assertVals(t, "minute", e.minute, []int{0})
}

func TestParseListAndNames(t *testing.T) {
	e, err := Parse("0,30 0,12 1-3 JAN-MAR SUN,WED")
	if err != nil {
		t.Fatal(err)
	}
	assertVals(t, "minute", e.minute, []int{0, 30})
	assertVals(t, "hour", e.hour, []int{0, 12})
	assertVals(t, "dom", e.dom, []int{1, 2, 3})
	assertVals(t, "month", e.month, []int{1, 2, 3})
	assertVals(t, "dow", e.dow, []int{0, 3})
}

func seq(from, to int) []int {
	out := make([]int, 0, to-from+1)
	for v := from; v <= to; v++ {
		out = append(out, v)
	}
	return out
}

func assertVals(t *testing.T, name string, f field, want []int) {
	t.Helper()
	got := f.values()
	if len(got) != len(want) {
		t.Fatalf("%s: got %v, want %v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s: got %v, want %v", name, got, want)
		}
	}
}

func starString(e *Expr) string {
	flags := []bool{e.minute.star, e.hour.star, e.dom.star, e.month.star, e.dow.star}
	var b strings.Builder
	for _, s := range flags {
		if s {
			b.WriteByte('*')
		} else {
			b.WriteByte('.')
		}
	}
	return b.String()
}

func sameSchedule(a, b *Expr) bool {
	return a.reboot == b.reboot &&
		a.minute == b.minute &&
		a.hour == b.hour &&
		a.dom == b.dom &&
		a.month == b.month &&
		a.dow == b.dow
}
