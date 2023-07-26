package fsrs

import (
	"time"
)

type Card struct {
	Due           time.Time `json:"Due"`
	Stability     float64   `json:"Stability"`
	Difficulty    float64   `json:"Difficulty"`
	ElapsedDays   uint64    `json:"ElapsedDays"`
	ScheduledDays uint64    `json:"ScheduledDays"`
	Reps          uint64    `json:"Reps"`
	Lapses        uint64    `json:"Lapses"`
	State         State     `json:"State"`
	LastReview    time.Time `json:"LastReview"`
}

func NewCard() Card {
	return Card{
		Due:           time.Time{},
		Stability:     0,
		Difficulty:    0,
		ElapsedDays:   0,
		ScheduledDays: 0,
		Reps:          0,
		Lapses:        0,
		State:         New,
		LastReview:    time.Time{},
	}
}

type ReviewLog struct {
	Rating        Rating    `json:"Rating"`
	ScheduledDays uint64    `json:"ScheduledDays"`
	ElapsedDays   uint64    `json:"ElapsedDays"`
	Review        time.Time `json:"Review"`
	State         State     `json:"State"`
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
