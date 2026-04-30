package fsrs

import (
	"time"
)

type Card struct {
	Due             time.Time `json:"Due"`
	Stability       float64   `json:"Stability"`
	Difficulty      float64   `json:"Difficulty"`
	ElapsedDays     uint64    `json:"ElapsedDays"`
	ScheduledDays   uint64    `json:"ScheduledDays"`
	Reps            uint64    `json:"Reps"`
	Lapses          uint64    `json:"Lapses"`
	State           State     `json:"State"`
	LastReview      time.Time `json:"LastReview"`
	RemainingSteps  int       `json:"RemainingSteps"`
}

func NewCard() Card {
	return Card{}
}

type ReviewLog struct {
	Rating         Rating    `json:"Rating"`
	Due            time.Time `json:"Due"`
	ScheduledDays  uint64    `json:"ScheduledDays"`
	ElapsedDays    uint64    `json:"ElapsedDays"`
	Review         time.Time `json:"Review"`
	State          State     `json:"State"`
	Stability      float64   `json:"Stability"`
	Difficulty     float64   `json:"Difficulty"`
	RemainingSteps int       `json:"RemainingSteps"`
}

type schedulingCards struct {
	Again Card
	Hard  Card
	Good  Card
	Easy  Card
}

func (s *schedulingCards) init(card Card) {
	s.Again = card
	s.Hard = card
	s.Good = card
	s.Easy = card
}

type SchedulingInfo struct {
	Card      Card
	ReviewLog ReviewLog
}

type RecordLog map[Rating]SchedulingInfo

type Rating int8

const (
	Again Rating = iota + 1
	Hard
	Good
	Easy
)

func (s Rating) String() string {
	switch s {
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

type MemoryState struct {
	Stability  float64 `json:"Stability"`
	Difficulty float64 `json:"Difficulty"`
}

type ItemState struct {
	Memory   MemoryState `json:"Memory"`
	Interval float64     `json:"Interval"`
}

type NextStates struct {
	Again ItemState `json:"Again"`
	Hard  ItemState `json:"Hard"`
	Good  ItemState `json:"Good"`
	Easy  ItemState `json:"Easy"`
}
