# cron-explain

[![ci](https://github.com/frozenmango8/cron-explain/actions/workflows/ci.yml/badge.svg)](https://github.com/frozenmango8/cron-explain/actions/workflows/ci.yml)

Explain a cron expression in plain English and show when it runs next.

## Install

```
go install github.com/frozenmango8/cron-explain/cmd/cron-explain@latest
```

## Usage

```
$ cron-explain '*/15 9-17 * * 1-5' -n 5 --tz America/Los_Angeles
Every 15 minutes from 09:00 through 17:59 on Monday through Friday
2026-11-27 09:00:00 -0800
2026-11-27 09:15:00 -0800
2026-11-27 09:30:00 -0800
2026-11-27 09:45:00 -0800
2026-11-27 10:00:00 -0800
```

```
$ cron-explain '@daily' -n 3 --tz UTC
Every day at 00:00
2026-11-28 00:00:00 +0000
2026-11-29 00:00:00 +0000
2026-11-30 00:00:00 +0000
```

```
$ cron-explain '60 9 * * *'
minute: 60 out of range (0-59)
```

```
$ cron-explain -n 0 '30 4 1,15 * 5'
At 04:30 on days 1 and 15 of the month, or on Friday
```

Times are strictly after now, in the zone from `--tz` (default `Local`). `-n` defaults to 5. `-n 0` prints only the description.

```go
expr, err := cronexplain.Parse("*/15 9-17 * * 1-5")
fmt.Println(expr.Describe())
times, err := expr.Next(time.Now(), 5)
```

Import path: `github.com/frozenmango8/cron-explain`.

## Limitations

Five fields only (minute, hour, day-of-month, month, day-of-week). No seconds field, no year field, no Quartz tokens (`?`, `L`, `W`, `#`). Names are the three-letter forms (`JAN`, `MON`).

When day-of-month and day-of-week are both restricted, a day matches if either field matches. That is vixie cron's behavior. Ranges do not wrap. `FRI-SUN` works, because a range may end on Sunday; `SAT-MON` does not.

`@reboot` has no next run. A wall time in a DST gap is skipped. A wall time that happens twice when clocks fall back is listed twice. Search stops after five years, so an impossible schedule such as `0 0 31 2 *` errors instead of inventing a time.
