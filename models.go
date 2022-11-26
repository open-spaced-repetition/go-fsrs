package fsrs

import (
	"time"
)

type Weights [13]float64

type Card struct {
	Due           time.Time    `json:"Due"`
	Stability     float64      `json:"Stability"`
	Difficulty    float64      `json:"Difficulty"`
	ElapsedDays   uint64       `json:"ElapsedDays"`
	ScheduledDays uint64       `json:"ScheduledDays"`
	Reps          uint64       `json:"Reps"`
	Lapses        uint64       `json:"Lapses"`
	Leeched       bool         `json:"Leeched"`
	State         State        `json:"State"`
	LastReview    time.Time    `json:"LastReview"`
	ReviewLogs    []*ReviewLog `json:"ReviewLogs"`
}

type ReviewLog struct {
	Rating        Rating
	ScheduledDays uint64
	ElapsedDays   uint64
	Reivew        time.Time
}

type SchedulingCards struct {
	Again Card
	Hard  Card
	Good  Card
	Easy  Card
}

func (s *SchedulingCards) init(card *Card) {
	s.Again = *card
	s.Hard = *card
	s.Good = *card
	s.Easy = *card
}

type Rating int8

const (
	Again Rating = iota
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

func DefaultWeights() Weights {
	return Weights{1, 1, 5, -0.5, -0.5, 0.2, 1.4, -0.12, 0.8, 2, -0.2, 0.2, 1}
}
