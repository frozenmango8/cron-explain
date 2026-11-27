package main

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

func TestRun(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		code   int
		outHas string
		errHas string
	}{
		{
			name:   "times",
			args:   []string{"-n", "2", "--tz", "UTC", "0 0 * * *"},
			code:   0,
			outHas: "Every day at 00:00\n",
		},
		{
			name:   "bad minute",
			args:   []string{"60 * * * *"},
			code:   1,
			errHas: "minute: 60 out of range (0-59)",
		},
		{
			name:   "bad zone",
			args:   []string{"--tz", "Not/AZone", "0 0 * * *"},
			code:   1,
			errHas: "time zone:",
		},
		{
			name:   "version",
			args:   []string{"-version"},
			code:   0,
			outHas: "dev\n",
		},
		{
			name:   "missing",
			args:   nil,
			code:   2,
			errHas: "usage:",
		},
		{
			name:   "negative n",
			args:   []string{"-n", "-1", "* * * * *"},
			code:   2,
			errHas: "n must be >= 0",
		},
		{
			name:   "description only",
			args:   []string{"-n", "0", "@weekly"},
			code:   0,
			outHas: "At 00:00 on Sunday\n",
		},
		{
			name:   "reboot",
			args:   []string{"-n", "0", "@reboot"},
			code:   0,
			outHas: "At startup\n",
		},
		{
			name:   "reboot wants times",
			args:   []string{"@reboot"},
			code:   1,
			outHas: "At startup\n",
			errHas: "@reboot has no next run",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var stdout, stderr bytes.Buffer
			code := run(tt.args, &stdout, &stderr)
			if code != tt.code {
				t.Fatalf("code %d, want %d\nstderr: %s\nstdout: %s", code, tt.code, stderr.String(), stdout.String())
			}
			if tt.outHas != "" && !strings.Contains(stdout.String(), tt.outHas) {
				t.Fatalf("stdout %q, want it to contain %q", stdout.String(), tt.outHas)
			}
			if tt.errHas != "" && !strings.Contains(stderr.String(), tt.errHas) {
				t.Fatalf("stderr %q, want it to contain %q", stderr.String(), tt.errHas)
			}
			if tt.name == "bad minute" && stdout.Len() != 0 {
				t.Fatalf("stdout %q, want empty", stdout.String())
			}
		})
	}
}

func TestRunPrintsTimes(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-n", "3", "--tz", "UTC", "*/15 * * * *"}, &stdout, &stderr)
	if code != 0 {
		t.Fatal(stderr.String())
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 4 {
		t.Fatalf("lines %#v", lines)
	}
	if lines[0] != "Every 15 minutes" {
		t.Fatalf("description %q", lines[0])
	}
	var prev time.Time
	for _, line := range lines[1:] {
		ts, err := time.Parse("2006-01-02 15:04:05 -0700", line)
		if err != nil {
			t.Fatalf("parse %q: %v", line, err)
		}
		if !prev.IsZero() && !ts.After(prev) {
			t.Fatalf("%s not after %s", ts, prev)
		}
		if ts.Minute()%15 != 0 || ts.Second() != 0 {
			t.Fatalf("not a quarter hour: %s", ts)
		}
		prev = ts
	}
}

func TestRunFlagsAfterExpression(t *testing.T) {
	var stdout, stderr bytes.Buffer
	args := []string{"*/15 9-17 * * 1-5", "-n", "2", "--tz", "UTC"}
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d, stderr %s", code, stderr.String())
	}
	lines := strings.Split(strings.TrimSuffix(stdout.String(), "\n"), "\n")
	if len(lines) != 3 {
		t.Fatalf("lines %#v", lines)
	}
	if lines[0] != "Every 15 minutes from 09:00 through 17:59 on Monday through Friday" {
		t.Fatalf("description %q", lines[0])
	}

	stdout.Reset()
	stderr.Reset()
	args = []string{"--tz=UTC", "-n=1", "0 0 * * *"}
	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("code %d, stderr %s", code, stderr.String())
	}
	if !strings.HasPrefix(stdout.String(), "Every day at 00:00\n") {
		t.Fatalf("stdout %q", stdout.String())
	}
}

func TestHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"-h"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("code %d, stderr %s", code, stderr.String())
	}
	if !strings.Contains(stderr.String(), "usage: cron-explain") {
		t.Fatalf("stderr %q", stderr.String())
	}
}
