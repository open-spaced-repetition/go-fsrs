# Migrating from go-fsrs v3

This guide covers v3.x (tested with v3.3.1) to v4.x. v4 changes the module path,
uses FSRS-6, returns errors from scheduling calls, and validates persisted
state. Back up cards, logs, and parameters before starting; v4 writes fields
that v3 ignores, so rolling back loses data.

## Compatibility at a glance

| Area | v3 | v4 | Action |
| --- | --- | --- | --- |
| Module path | `.../go-fsrs/v3` | `.../go-fsrs/v4` | Change the import |
| Go version | 1.21 | 1.26 | Upgrade the toolchain |
| Algorithm | FSRS-5, 19 weights | FSRS-6, 21 weights | Convert weights |
| `Repeat` / `Next` | No error | Returns `error` | Handle the error |
| Retrievability | `GetRetrievability` | `Retrievability` + error | Rename and handle error |
| Card | No `RemainingSteps` | `RemainingSteps` | Choose a step policy |
| Review log | No memory fields | Adds `Due`, `Stability`, `Difficulty`, `RemainingSteps` | Enrich or audit-only |
| Parameters | No learning steps | `LearningSteps` / `RelearningSteps` | Set defaults or custom values |
| `NewCard()` | `Due` is zero | `Due` defaults to now | Pass an explicit time if needed |

## 1. Migrate the API

```go
// v3
import "github.com/open-spaced-repetition/go-fsrs/v3"

// v4
import "github.com/open-spaced-repetition/go-fsrs/v4"
```

```bash
go mod edit -go=1.26
go get github.com/open-spaced-repetition/go-fsrs/v4@latest
```

`Repeat` and `Next` now return errors:

```go
// v3 returned values directly
log, err := f.Repeat(card, now)
if err != nil { return err }
info, err := f.Next(card, now, fsrs.Good)
if err != nil { return err }
```

`GetRetrievability` became `Retrievability` and also returns an error:

```go
r, err := f.Retrievability(card, now)
if err != nil { return err }
```

`Rating` adds `Manual` (0); Again..Easy remain 1..4. Add a `Manual` case to
exhaustive switches.

`Weights` changed from `[19]float64` to `[21]float64`:

```go
w, err := fsrs.MigrateWeights(oldWeights) // 17, 19, or 21 values
if err != nil { return err }
p := fsrs.DefaultParam()
p.W = w
```

`ConvertV5Weights` (19 values) and `ConvertV45Weights` (17 values) are explicit
alternatives. `ConvertV5Weights` sets `W[19] = 0` and `W[20] = 0.5`, preserving
the default v3 decay.

v4 adds `LearningSteps` and `RelearningSteps`. A zero-value `Parameters` has
nil step slices, which makes every rating graduate immediately. Start from
`DefaultParam()`:

```go
p := fsrs.DefaultParam()
p.LearningSteps = []float64{1, 10} // minutes
p.RelearningSteps = []float64{10}  // minutes
```

v3 had fixed delays (Again 1m, Hard 5m, Good 10m on a new card); v4 defaults
produce 1m/6m/10m. Set the slices explicitly for specific timing. v4 derives
`Decay`/`Factor` from `W[20]`; do not copy the v3 values.

`NewFSRS` clips weights and, if validation still fails, silently resets **all**
parameters to `DefaultParam()`. Validate first:

```go
if err := p.Validate(); err != nil { return err }
f := fsrs.NewFSRS(p)
```

| Field | v4 rule |
| --- | --- |
| `W` | All values finite; `W[20] > 0` |
| `RequestRetention` | `(0, 1]` |
| `MaximumInterval` | `(0, 36500]` |
| `LearningSteps` / `RelearningSteps` | Finite and `>= 0` |

v3 did not validate these. A v3 configuration with `MaximumInterval > 36500` or
retention outside `(0, 1]` now resets every parameter, including custom weights.

A v3 `Parameters` JSON has 19 weights and no step fields. Unmarshalling it
directly into v4 leaves `W[19]`/`W[20]` at zero; `NewFSRS` clips `W[20]` to its
minimum, producing the wrong decay, and the steps stay nil. Use a v3-shaped
struct:

```go
type v3Parameters struct {
	RequestRetention float64   `json:"RequestRetention"`
	MaximumInterval  float64   `json:"MaximumInterval"`
	Weights          []float64 `json:"Weights"`
	EnableShortTerm  bool      `json:"EnableShortTerm"`
	EnableFuzz       bool      `json:"EnableFuzz"`
}

var old v3Parameters
if err := json.Unmarshal(data, &old); err != nil { return err }
w, err := fsrs.MigrateWeights(old.Weights)
if err != nil { return err }
p := fsrs.DefaultParam()
p.RequestRetention = old.RequestRetention
p.MaximumInterval = old.MaximumInterval
p.W = w
p.EnableShortTerm = old.EnableShortTerm
p.EnableFuzz = old.EnableFuzz
// Keep v4 defaults, or set p.LearningSteps / p.RelearningSteps.
if err := p.Validate(); err != nil { return err }
```

If your JSON may omit fields, use pointers or a map so missing values keep the
`DefaultParam` defaults.

## 2. Migrate persisted data

| v3 `Card` | v4 `Card` | Action |
| --- | --- | --- |
| `Due` | `Due` | Keep |
| `Stability` | `Stability` | Repair if out of range |
| `Difficulty` | `Difficulty` | Repair if out of range |
| `ElapsedDays` | Removed | Ignore; v4 derives it from `LastReview` |
| `ScheduledDays` | `ScheduledDays` | Reset if above 36500 |
| `Reps`, `Lapses` | `Reps`, `Lapses` | Keep |
| `State` | `State` | Keep |
| `LastReview` | `LastReview` | Keep; set it for non-New cards |
| — | `RemainingSteps` | Choose a policy for Learning/Relearning |

v3 did not persist a step counter. A migrated Learning/Relearning card arrives
with `RemainingSteps = 0`, which makes every rating graduate on the next review.
Choose a policy:

| `RemainingSteps` | Behavior |
| --- | --- |
| `0` | Graduate on the next review |
| `1` | Allow one more step; Again restarts the sequence |
| `len(steps)` | Replay the full learning/relearning sequence |

A conservative default is one remaining step:

```go
if card.State == fsrs.Learning {
	card.RemainingSteps = min(max(card.RemainingSteps, 1), len(p.LearningSteps))
}
if card.State == fsrs.Relearning {
	card.RemainingSteps = min(max(card.RemainingSteps, 1), len(p.RelearningSteps))
}
```

v3 allowed stability below `SMin` (including zero) and could leave `NaN` values.
v4 rejects non-finite or sub-minimum memory state on non-New cards. It does not
validate `ScheduledDays`, so reset values above 36500:

```go
func clampMemory(v, lo, hi float64) float64 {
	if math.IsNaN(v) {
		return lo
	}
	return math.Max(lo, math.Min(v, hi))
}

func migrateCard(card fsrs.Card, now time.Time) fsrs.Card {
	if card.State != fsrs.New {
		card.Stability = clampMemory(card.Stability, fsrs.SMin, fsrs.SMax)
		card.Difficulty = clampMemory(card.Difficulty, fsrs.DMin, fsrs.DMax)
	}
	if card.ScheduledDays > 36500 {
		card.ScheduledDays = 0
		card.Due = now
	}
	return card
}
```

`SMax`/`DMax` are saturation limits: v4 rejects only non-finite or sub-minimum
values, while the scheduler clamps above-maximum values. The recipe normalizes
them too, which is stricter than validation. It is idempotent; the next review
recalculates `ScheduledDays`, and the first post-migration `ReviewLog` records
the input card's value.

v4 derives elapsed days from `LastReview`; keep it accurate. `NewCard` now
defaults `Due` to `time.Now()` instead of the zero time.

| v3 `ReviewLog` | v4 `ReviewLog` | Action |
| --- | --- | --- |
| `Rating` | `Rating` | Keep; `Manual` (0) is new |
| `ScheduledDays` | `ScheduledDays` | Keep |
| `ElapsedDays` | Removed | Use `Review` timestamps |
| `Review` | `Review` | Keep |
| `State` | `State` | Keep |
| — | `Due` | Added; v3 logs lack it |
| — | `Stability` | Added; v3 logs lack it |
| — | `Difficulty` | Added; v3 logs lack it |
| — | `RemainingSteps` | Added; v3 logs lack it |

Old logs still unmarshal, but the added fields are zero. They remain useful for
audit and statistics; they are not sufficient for state reconstruction.
`Rollback` rebuilds the pre-review card from `ReviewLog.Due`, `Stability`,
`Difficulty`, `ScheduledDays`, and `RemainingSteps`; do not pass v3 logs to it
without enriching them from a card snapshot.

`Reschedule` replays `ReviewHistory`. Non-manual entries only need `Rating` and
`Review`, so a basic replay of old ratings is possible. Manual entries also need
`State`, `Due`, `Stability`, and `Difficulty`. Replay uses v4 weights and FSRS-6
formulas, so it is a rebuild, not a lossless conversion.

## 3. Verify and roll out

v4 is not a drop-in scheduler with identical output. Expect different intervals
because FSRS-6 uses 21 weights and new short-term/forget formulas, learning
steps are configurable, fuzz seeds differ, and v4 validates data that v3
accepted. `NewCard` also changes the default `Due`.

Rollout checklist:

1. Back up cards, logs, and parameters; keep a v3-readable copy.
2. Convert and validate parameters and weights.
3. Migrate a sample across New, Learning, Review, and Relearning.
4. Run `Repeat` for both short-term and long-term schedulers and inspect
   `ScheduledDays` and `Due`.
5. Shadow-run v3 and v4 on a larger sample; define acceptable drift before
   switching writes.
6. Deploy, monitor error rates and first reviews, and keep the rollback copy.

## Reference

Removed or renamed APIs:

| v3 | v4 | Notes |
| --- | --- | --- |
| `GetRetrievability` | `Retrievability` | Returns `(float64, error)` |
| `FUZZ_RANGES` / `FuzzRange` | `FuzzRanges()` | Element type is unexported |
| `NewAlea`, `Mash`, `PRNG.Int32`, `PRNG.State`, `PRNG.ImportState` | Removed | Use `Alea`, `PRNG.Double`, or your own RNG |
| `Card.ElapsedDays` | Removed | v4 derives it from `LastReview` |
| `ReviewLog.ElapsedDays` | Removed | Use `Review` timestamps |
| `NewCard()` | `NewCard(now ...time.Time)` | `Due` now defaults to now |

Structured errors expose `Error.Code`; use `errors.As` to match them. Codes:
`ErrCodeInvalidInput`, `ErrCodeManualRating`, `ErrCodeInvalidRating`,
`ErrCodeManualStateRequired`, `ErrCodeManualDueRequired`,
`ErrCodeInvalidWeightsLength`, `ErrCodeInvalidWeightsValue`,
`ErrCodeInvalidDecay`, `ErrCodeInvalidRetention`, `ErrCodeInvalidMaxInterval`,
`ErrCodeInvalidSteps`.

v4 also adds `Forget`, `Rollback`, `Reschedule`, `MemoryState`,
`HistoricalMemoryStates`, `NextState`, and `NextStates`; they are optional for
the migration. This guide covers v3.x to v4.x and FSRS-6, not v1/v2, FSRS-7, or
the `optimizer` and `simulator` packages.
