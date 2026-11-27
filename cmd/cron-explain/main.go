package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	_ "time/tzdata" // zone database, so a released binary works without the OS one

	"github.com/frozenmango8/cron-explain"
)

// version is set by goreleaser with -X main.version.
var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("cron-explain", flag.ContinueOnError)
	fs.SetOutput(stderr)
	n := fs.Int("n", 5, "number of upcoming runs to print (0 prints only the description)")
	tzName := fs.String("tz", "Local", "time zone (IANA name, or Local)")
	showVersion := fs.Bool("version", false, "print version and exit")
	fs.Usage = func() {
		fmt.Fprintf(stderr, "usage: cron-explain [flags] <expression>\n\n")
		fmt.Fprintf(stderr, "Explain a cron expression and print its next run times.\n\n")
		fs.PrintDefaults()
	}
	// flag stops at the first non-flag, but the expression usually comes first:
	// cron-explain '*/15 9-17 * * 1-5' -n 5 --tz UTC
	if err := fs.Parse(normalizeArgs(args)); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return 0
		}
		return 2
	}
	if *showVersion {
		fmt.Fprintln(stdout, version)
		return 0
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}
	if *n < 0 {
		fmt.Fprintln(stderr, "n must be >= 0")
		return 2
	}

	expr, err := cronexplain.Parse(fs.Arg(0))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	fmt.Fprintln(stdout, expr.Describe())
	if *n == 0 {
		return 0
	}

	loc, err := loadLocation(*tzName)
	if err != nil {
		fmt.Fprintf(stderr, "time zone: %v\n", err)
		return 1
	}
	times, err := expr.Next(time.Now().In(loc), *n)
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	for _, ts := range times {
		fmt.Fprintln(stdout, ts.Format("2006-01-02 15:04:05 -0700"))
	}
	return 0
}

// normalizeArgs moves flags in front of positional args so either order works.
func normalizeArgs(args []string) []string {
	var flags, pos []string
	for i := 0; i < len(args); i++ {
		a := args[i]
		if a == "--" {
			pos = append(pos, args[i+1:]...)
			break
		}
		if a == "-" || !strings.HasPrefix(a, "-") {
			pos = append(pos, a)
			continue
		}
		name, inline := flagSpec(a)
		flags = append(flags, a)
		if !inline && flagTakesValue(name) && i+1 < len(args) {
			i++
			flags = append(flags, args[i])
		}
	}
	return append(flags, pos...)
}

func flagSpec(a string) (name string, inline bool) {
	name = strings.TrimLeft(a, "-")
	if n, _, ok := strings.Cut(name, "="); ok {
		return n, true
	}
	return name, false
}

func flagTakesValue(name string) bool {
	switch name {
	case "n", "tz":
		return true
	default:
		return false
	}
}

func loadLocation(name string) (*time.Location, error) {
	switch name {
	case "", "Local", "local":
		return time.Local, nil
	default:
		return time.LoadLocation(name)
	}
}
