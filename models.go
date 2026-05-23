package fsrs

import (
	"time"
)

type Card struct {
	Due            time.Time `json:"Due"`
	Stability      float64   `json:"Stability"`
	Difficulty     float64   `json:"Difficulty"`
	ScheduledDays  uint64    `json:"ScheduledDays"`
	Reps           uint64    `json:"Reps"`
	Lapses         uint64    `json:"Lapses"`
	State          State     `json:"State"`
	LastReview     time.Time `json:"LastReview"`
	RemainingSteps int       `json:"RemainingSteps"`
}

// NewCard returns a new Card with default values. If now is provided, Due is
// set to now; otherwise Due defaults to time.Now(), aligning with ts-fsrs
// createEmptyCard() which uses new Date() as the default.
func NewCard(now ...time.Time) Card {
	card := Card{}
	if len(now) > 0 {
		card.Due = now[0]
	} else {
		card.Due = time.Now()
	}
	return card
}

type ReviewLog struct {
	Rating         Rating    `json:"Rating"`
	Due            time.Time `json:"Due"`
	ScheduledDays  uint64    `json:"ScheduledDays"`
	Review         time.Time `json:"Review"`
	State          State     `json:"State"`
	Stability      float64   `json:"Stability"`
	Difficulty     float64   `json:"Difficulty"`
	RemainingSteps int       `json:"RemainingSteps"`
}

type SchedulingInfo struct {
	Card      Card      `json:"Card"`
	ReviewLog ReviewLog `json:"ReviewLog"`
}

type RecordLog map[Rating]SchedulingInfo

type Rating int8

const Manual Rating = 0

const (
	Again Rating = iota + 1
	Hard
	Good
	Easy
)

func (r Rating) String() string {
	switch r {
	case Manual:
		return "Manual"
	case Again:
		return "Again"
	case Hard:
		return "Hard"
	case Good:
		return "Good"
	case Easy:
		return "Easy"
	}
	return "unknown"
}

type State int8

const (
	New State = iota
	Learning
	Review
	Relearning
)

func (s State) String() string {
	switch s {
	case New:
		return "New"
	case Learning:
		return "Learning"
	case Review:
		return "Review"
	case Relearning:
		return "Relearning"
	}
	return "unknown"
}

type MemoryState struct {
	Stability  float64 `json:"Stability"`
	Difficulty float64 `json:"Difficulty"`
}

type ItemState struct {
	Memory   MemoryState `json:"Memory"`
	Interval float64     `json:"Interval"`
}

// ReviewEntry represents a single review event with a rating and elapsed days since the last review.
type ReviewEntry struct {
	Rating Rating  `json:"Rating"`
	DeltaT float64 `json:"DeltaT"`
}

// ReviewEntries is a chronologically ordered sequence of ReviewEntry values.
type ReviewEntries []ReviewEntry

type NextStates struct {
	Again ItemState `json:"Again"`
	Hard  ItemState `json:"Hard"`
	Good  ItemState `json:"Good"`
	Easy  ItemState `json:"Easy"`
}

// ReviewHistory represents a single past review event used to replay and
// reconstruct card memory state via [FSRS.Reschedule].
type ReviewHistory struct {
	Rating        Rating    `json:"Rating"`
	Review        time.Time `json:"Review"`
	State         *State    `json:"State"`
	Due           time.Time `json:"Due"`
	Stability     float64   `json:"Stability"`
	Difficulty    float64   `json:"Difficulty"`
	ScheduledDays uint64    `json:"ScheduledDays"`
}

// RescheduleResult holds the output of [FSRS.Reschedule]: the full replay
// history and an optional final reschedule item when the card needs updating.
type RescheduleResult struct {
	Collections    []SchedulingInfo `json:"Collections"`
	RescheduleItem *SchedulingInfo  `json:"RescheduleItem"`
}

// RescheduleOptions configures the behaviour of [FSRS.Reschedule].
type RescheduleOptions struct {
	SkipManual        bool      `json:"SkipManual"`
	UpdateMemoryState bool      `json:"UpdateMemoryState"`
	Now               time.Time `json:"Now"`
	FirstDue          time.Time `json:"FirstDue"`
	SortReviews       bool      `json:"SortReviews"`
}

// StatePtr returns a pointer to the given State value. It is a convenience
// helper for constructing [ReviewHistory] entries with a non-nil State field.
func StatePtr(s State) *State { return &s }
