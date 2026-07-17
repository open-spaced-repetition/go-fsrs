# go-fsrs

[![Go Reference](https://pkg.go.dev/badge/github.com/open-spaced-repetition/go-fsrs/v4.svg)](https://pkg.go.dev/github.com/open-spaced-repetition/go-fsrs/v4) [![Go Report Card](https://goreportcard.com/badge/github.com/open-spaced-repetition/go-fsrs/v4)](https://goreportcard.com/report/github.com/open-spaced-repetition/go-fsrs/v4)
![Go version](https://img.shields.io/github/go-mod/go-version/open-spaced-repetition/go-fsrs) ![Tests](https://img.shields.io/github/actions/workflow/status/open-spaced-repetition/go-fsrs/test.yml?style=flat-square&label=tests) ![License](https://img.shields.io/github/license/open-spaced-repetition/go-fsrs?style=flat-square)

A Go library for building spaced-repetition systems with [FSRS](https://github.com/open-spaced-repetition/free-spaced-repetition-scheduler).

| Package | Description | FSRS Version | Package Version |
|---|---|---|---|
| `go-fsrs` | FSRS scheduler for review flows | ![FSRS](https://img.shields.io/badge/FSRS-v6-blue?style=flat-square) | ![version](https://img.shields.io/github/v/tag/open-spaced-repetition/go-fsrs?style=flat-square) |
| `go-fsrs/optimizer` | Train FSRS parameters from review logs | ![FSRS](https://img.shields.io/badge/FSRS-v6-blue?style=flat-square) | ![status](https://img.shields.io/badge/status-in_development-orange?style=flat-square) |
| `go-fsrs/simulator` | Simulate deck workload and optimal retention | ![FSRS](https://img.shields.io/badge/FSRS-v6-blue?style=flat-square) | ![status](https://img.shields.io/badge/status-planned-lightgrey?style=flat-square) |

## Install

```bash
go get github.com/open-spaced-repetition/go-fsrs/v4@latest
```

## Quick start

```go
package main

import (
	"fmt"
	"time"

	"github.com/open-spaced-repetition/go-fsrs/v4"
)

func main() {
	s := fsrs.NewFSRS(fsrs.DefaultParam())
	now := time.Now()
	card := fsrs.NewCard(now)

	// Preview all four ratings, then apply one.
	preview, err := s.Repeat(card, now)
	if err != nil {
		panic(err)
	}
	fmt.Println(preview[fsrs.Good])

	review, err := s.Next(card, now, fsrs.Good)
	if err != nil {
		panic(err)
	}
	card = review.Card
}
```

See the [GoDoc](https://pkg.go.dev/github.com/open-spaced-repetition/go-fsrs/v4) for the full API reference.

## API

### High-level (`*FSRS`)

`*FSRS` is the complete scheduling layer. It walks a `Card` through its lifecycle
(New → Learning → Review → Relearning), applies learning steps and short-term
memory updates, optionally fuzzes intervals, and validates all input with
structured errors. This is the recommended entry point for most applications:

| Method | Returns | Purpose |
|---|---|---|
| `Repeat(card, now)` | `(RecordLog, error)` | Returns the result of each potential rating. |
| `Next(card, now, grade)` | `(SchedulingInfo, error)` | Reviews the card with a given rating; returns the updated card + `ReviewLog`. |
| `Retrievability(card, now)` | `(float64, error)` | Current probability of recall. |
| `Reschedule(card, reviews, opts)` | `(RescheduleResult, error)` | Replay a review history; rebuild state and due date. |
| `Forget(card, now, resetCount)` | `(SchedulingInfo, error)` | Reset a card to `New`, preserving its `ReviewLog`. |
| `Rollback(card, log)` | `(Card, error)` | Revert a review using its `ReviewLog`. |
| `MemoryState(history, start)` | `(*MemoryState, error)` | Derive stability/difficulty from `ReviewEntries`. |
| `HistoricalMemoryStates(history, start)` | `([]MemoryState, error)` | All intermediate memory states. |

### Low-level (`Parameters` / `Scheduler`)

`Parameters` exposes the raw FSRS algorithm — given a `MemoryState` and a rating,
it computes the next stability, difficulty, and ideal review interval. There is no
scheduler, no card state machine, no learning steps, and no fuzz — just the core
math that the scheduler itself builds upon:

| Method | Returns |
|---|---|
| `ForgettingCurve(elapsedDays, stability)` | `float64` |
| `NextState(state, retention, days, grade)` | `ItemState` |
| `NextStates(state, retention, days)` | `NextStates` |
| `ApplyFuzz(interval, elapsedDays, enable)` | `float64` |

`NewBasicScheduler` / `NewLongTermScheduler` return a lightweight `*Scheduler` that
drives a single `Card` through steps and fuzz without validation. Neither `Preview()`
nor `Review(grade)` returns an error:

| Method | Returns |
|---|---|
| `Preview()` | `RecordLog` |
| `Review(grade)` | `SchedulingInfo` |

```go
p := fsrs.DefaultParam()
next := p.NextState(&fsrs.MemoryState{Stability: 5.0, Difficulty: 5.0}, 0.9, 3, fsrs.Good)
fmt.Println(next.Interval)

now := time.Now()
card := fsrs.NewCard(now)
s := p.NewBasicScheduler(card, now)
fmt.Println(s.Review(fsrs.Good).Card.Due)
```

Enable interval fuzz with `p.EnableFuzz = true`. This will cause the scheduler to spread reviews across nearby days. The randomness is seeded per-card for reproducibility.

### Weights

FSRS v6 uses a 21-element weight vector (`Weights` = `[21]float64`). [`DefaultWeights()`](https://github.com/open-spaced-repetition/awesome-fsrs/wiki/The-Algorithm#default-parameters) returns the built-in baseline; the helpers below convert from older formats:

| Helper | Input | Converts from |
|---|---|---|
| `ConvertV5Weights` | `[19]float64` | FSRS v5 |
| `ConvertV45Weights` | `[17]float64` | FSRS v4.5 |
| `MigrateWeights` | `[]float64` | Auto-detect (17/19/21) |

### Errors

Invalid input returns a typed `*fsrs.Error` with a machine-readable `Code`,
compatible with `errors.Is` / `errors.As`:

```go
var fsrsErr *fsrs.Error
if errors.As(err, &fsrsErr) {
	fmt.Println(fsrsErr.Code) // e.g. fsrs.ErrCodeInvalidInput
}
```

## Contributing

Pull requests are welcome. Suggestions, bug reports, and feature ideas belong in [Issues](https://github.com/open-spaced-repetition/go-fsrs/issues).

For algorithm design discussions, please use the [discussions page](https://github.com/orgs/open-spaced-repetition/discussions).
