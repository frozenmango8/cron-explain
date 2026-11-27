package cronexplain

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestNext(t *testing.T) {
	utc := time.UTC
	tests := []struct {
		spec string
		from string
		want []string
	}{
		{
			"* * * * *",
			"2026-01-01T00:00:00Z",
			[]string{"2026-01-01T00:01:00Z", "2026-01-01T00:02:00Z", "2026-01-01T00:03:00Z"},
		},
		{
			"* * * * *",
			"2026-01-01T00:00:30Z",
			[]string{"2026-01-01T00:01:00Z", "2026-01-01T00:02:00Z"},
		},
		{
			"*/15 * * * *",
			"2026-01-01T00:00:00Z",
			[]string{"2026-01-01T00:15:00Z", "2026-01-01T00:30:00Z", "2026-01-01T00:45:00Z", "2026-01-01T01:00:00Z"},
		},
		{
			"0 0 * * *",
			"2026-01-01T00:00:00Z",
			[]string{"2026-01-02T00:00:00Z", "2026-01-03T00:00:00Z"},
		},
		{
			"30 9 * * 1-5",
			"2026-01-01T00:00:00Z", // Thursday
			[]string{"2026-01-01T09:30:00Z", "2026-01-02T09:30:00Z", "2026-01-05T09:30:00Z"},
		},
		{
			"0 9 * * MON",
			"2026-01-01T00:00:00Z",
			[]string{"2026-01-05T09:00:00Z", "2026-01-12T09:00:00Z"},
		},
		{
			"0 0 1 * *",
			"2026-01-15T00:00:00Z",
			[]string{"2026-02-01T00:00:00Z", "2026-03-01T00:00:00Z"},
		},
		{
			"0 0 31 * *",
			"2026-01-31T00:00:00Z",
			[]string{"2026-03-31T00:00:00Z", "2026-05-31T00:00:00Z"},
		},
		{
			"0 0 29 2 *",
			"2026-01-01T00:00:00Z",
			[]string{"2028-02-29T00:00:00Z", "2032-02-29T00:00:00Z"},
		},
		{
			"0 0 1 1 *",
			"2026-06-01T00:00:00Z",
			[]string{"2027-01-01T00:00:00Z"},
		},
		{
			"@daily",
			"2026-11-27T12:00:00Z",
			[]string{"2026-11-28T00:00:00Z"},
		},
		{
			"15 14 1 * *",
			"2026-01-01T14:15:00Z",
			[]string{"2026-02-01T14:15:00Z"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.spec+" "+tt.from, func(t *testing.T) {
			e := mustParse(t, tt.spec)
			from, err := time.Parse(time.RFC3339, tt.from)
			if err != nil {
				t.Fatal(err)
			}
			from = from.In(utc)
			got, err := e.Next(from, len(tt.want))
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len %d, want %d", len(got), len(tt.want))
			}
			for i, w := range tt.want {
				want, err := time.Parse(time.RFC3339, w)
				if err != nil {
					t.Fatal(err)
				}
				if !got[i].Equal(want) {
					t.Fatalf("result %d: got %s, want %s", i, got[i].Format(time.RFC3339), w)
				}
			}
		})
	}
}

func TestNextOrDays(t *testing.T) {
	// 04:30 on the 1st, the 15th, or Friday. 2026-01-01 is a Thursday.
	e := mustParse(t, "30 4 1,15 * 5")
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	got, err := e.Next(from, 4)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{
		"2026-01-01T04:30:00Z", // the 1st
		"2026-01-02T04:30:00Z", // Friday
		"2026-01-09T04:30:00Z", // Friday
		"2026-01-15T04:30:00Z", // the 15th
	}
	for i, w := range want {
		tw, err := time.Parse(time.RFC3339, w)
		if err != nil {
			t.Fatal(err)
		}
		if !got[i].Equal(tw) {
			t.Fatalf("got %s, want %s", got[i].Format(time.RFC3339), w)
		}
	}
}

func TestNextErrors(t *testing.T) {
	e := mustParse(t, "0 0 31 2 *")
	_, err := e.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1)
	if err == nil || err.Error() != "no matching time in the next 5 years" {
		t.Fatalf("got %v", err)
	}

	e = mustParse(t, "0 0 31 4 *")
	if _, err = e.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC), 1); err == nil {
		t.Fatal("expected no match for April 31")
	}

	e = mustParse(t, "@reboot")
	if _, err = e.Next(time.Now(), 1); err == nil || err.Error() != "@reboot has no next run" {
		t.Fatalf("got %v", err)
	}

	e = mustParse(t, "* * * * *")
	got, err := e.Next(time.Now(), 0)
	if err != nil || got != nil {
		t.Fatalf("n=0: %v %v", got, err)
	}
	if _, err = e.Next(time.Now(), -1); err == nil {
		t.Fatal("expected error for n < 0")
	}
}

func TestNextDST(t *testing.T) {
	loc := mustLoc(t, "America/Los_Angeles")

	// 2026-03-08 is the spring-forward: 02:00 does not exist.
	e := mustParse(t, "30 2 * * *")
	from := time.Date(2026, 3, 7, 0, 0, 0, 0, loc)
	got, err := e.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []time.Time{
		time.Date(2026, 3, 7, 2, 30, 0, 0, loc),
		time.Date(2026, 3, 9, 2, 30, 0, 0, loc),
		time.Date(2026, 3, 10, 2, 30, 0, 0, loc),
	}
	for i := range want {
		if !got[i].Equal(want[i]) {
			t.Fatalf("spring %d: got %s, want %s", i, got[i], want[i])
		}
		y, m, d := got[i].Date()
		if y == 2026 && m == time.March && d == 8 {
			t.Fatal("scheduled a time in the spring-forward gap")
		}
	}

	// 2026-11-01 is the fall-back. 01:30 happens twice.
	e = mustParse(t, "30 1 * * *")
	from = time.Date(2026, 11, 1, 0, 30, 0, 0, loc)
	got, err = e.Next(from, 2)
	if err != nil {
		t.Fatal(err)
	}
	first := time.Date(2026, 11, 1, 1, 30, 0, 0, loc)
	second := first.Add(time.Hour)
	if second.Hour() != 1 || second.Minute() != 30 || second.Equal(first) {
		t.Fatalf("failed to build the repeated 01:30: %s / %s", first, second)
	}
	if !got[0].Equal(first) || !got[1].Equal(second) {
		t.Fatalf("fall back: got %s and %s, want %s and %s", got[0], got[1], first, second)
	}

	// Spring forward: 01:59 is followed by 03:00, not 02:00.
	e = mustParse(t, "* * * * *")
	from = time.Date(2026, 3, 8, 1, 58, 0, 0, loc)
	got, err = e.Next(from, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 3; i++ {
		step := from.Add(time.Duration(i) * time.Minute)
		if !got[i-1].Equal(step) {
			t.Fatalf("spring walk %d: got %s, want %s", i, got[i-1], step)
		}
	}
	if got[1].Hour() != 3 {
		t.Fatalf("expected to land at 03:00, got %s", got[1])
	}

	// Every minute walks through the repeated hour instead of skipping it.
	e = mustParse(t, "* * * * *")
	from = time.Date(2026, 11, 1, 1, 58, 0, 0, loc)
	got, err = e.Next(from, 4)
	if err != nil {
		t.Fatal(err)
	}
	seq := []time.Time{
		from.Add(time.Minute),     // 01:59 PDT
		from.Add(2 * time.Minute), // 01:00 PST
		from.Add(3 * time.Minute),
		from.Add(4 * time.Minute),
	}
	for i := range seq {
		if !got[i].Equal(seq[i]) {
			t.Fatalf("fold walk %d: got %s, want %s", i, got[i], seq[i])
		}
	}
}

func TestNextZones(t *testing.T) {
	for _, name := range []string{"UTC", "America/New_York", "Asia/Kolkata", "Australia/Lord_Howe"} {
		t.Run(name, func(t *testing.T) {
			loc := mustLoc(t, name)
			e := mustParse(t, "0 12 * * *")
			from := time.Date(2026, 6, 15, 0, 0, 0, 0, loc)
			got, err := e.Next(from, 2)
			if err != nil {
				t.Fatal(err)
			}
			if got[0].Hour() != 12 || got[0].Minute() != 0 || got[0].Location().String() != loc.String() {
				t.Fatalf("got %s", got[0])
			}
			if got[1].Hour() != 12 || got[1].Minute() != 0 || !got[1].After(got[0]) {
				t.Fatalf("second %s, first %s", got[1], got[0])
			}
		})
	}
}

func TestNextMatchesSchedule(t *testing.T) {
	specs := []string{
		"* * * * *",
		"*/15 9-17 * * 1-5",
		"0 0 1,15 * 5",
		"5 4 * * SUN",
		"0 0 29 2 *",
		"@hourly",
		"0 12 * JAN-MAR MON,WED",
		"15,45 0,12,23 * * *",
		"0 0 31 1,3,5,7,8,10,12 *",
	}
	zones := []string{"UTC", "America/Los_Angeles", "Asia/Kolkata"}
	froms := []time.Time{
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		time.Date(2026, 3, 8, 1, 30, 0, 0, time.UTC),
		time.Date(2026, 11, 1, 0, 30, 45, 0, time.UTC),
	}
	for _, spec := range specs {
		e := mustParse(t, spec)
		for _, zone := range zones {
			loc := mustLoc(t, zone)
			for _, from := range froms {
				from = from.In(loc)
				got, err := e.Next(from, 8)
				if err != nil {
					t.Fatalf("%s %s from %s: %v", spec, zone, from, err)
				}
				prev := from
				for i, ts := range got {
					if !ts.After(prev) {
						t.Fatalf("%s: %s not after %s", spec, ts, prev)
					}
					if !matches(e, ts) {
						t.Fatalf("%s: %s does not match", spec, ts)
					}
					// Asking again from just before this instant must return it.
					again, err := e.Next(ts.Add(-time.Second), 1)
					if err != nil || !again[0].Equal(ts) {
						t.Fatalf("%s result %d not stable: %v %v", spec, i, again, err)
					}
					prev = ts
				}
			}
		}
	}
}

func ExampleExpr_Next() {
	e, _ := Parse("0 9 * * MON")
	from := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	times, _ := e.Next(from, 2)
	for _, ts := range times {
		fmt.Println(ts.Format(time.RFC3339))
	}
	// Output:
	// 2026-01-05T09:00:00Z
	// 2026-01-12T09:00:00Z
}

func FuzzParse(f *testing.F) {
	for _, s := range []string{
		"* * * * *",
		"*/15 9-17 * * 1-5",
		"0 0 31 2 *",
		"@daily",
		"@reboot",
		"60 * * * *",
		"0 0 * * FRI-SUN",
		"1-10/2 * JAN-MAR MON,WED",
	} {
		f.Add(s)
	}
	from := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	f.Fuzz(func(t *testing.T, expr string) {
		e, err := Parse(expr)
		if err != nil {
			var pe *ParseError
			if !errors.As(err, &pe) {
				t.Fatalf("error type %T", err)
			}
			return
		}
		_ = e.Describe()
		times, nerr := e.Next(from, 2)
		if nerr != nil {
			return
		}
		if len(times) != 2 || !times[0].After(from) || !times[1].After(times[0]) {
			t.Fatalf("times %v", times)
		}
	})
}

func matches(e *Expr, ts time.Time) bool {
	if ts.Second() != 0 || ts.Nanosecond() != 0 {
		return false
	}
	if !e.minute.has(ts.Minute()) || !e.hour.has(ts.Hour()) || !e.month.has(int(ts.Month())) {
		return false
	}
	y, m, d := ts.Date()
	return e.matchDay(y, m, d)
}

func mustParse(t *testing.T, spec string) *Expr {
	t.Helper()
	e, err := Parse(spec)
	if err != nil {
		t.Fatal(err)
	}
	return e
}

func mustLoc(t *testing.T, name string) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	return loc
}
