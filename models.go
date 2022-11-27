package fsrs

import (
	"time"
)

type weights [13]float64

func defaultWeights() weights {
	return weights{1, 1, 5, -0.5, -0.5, 0.2, 1.4, -0.12, 0.8, 2, -0.2, 0.2, 1}
}

type Parameters struct {
	RequestRetention float64
	MaximumInterval  float64
	EasyBonus        float64
	HardFactor       float64
	W                weights
}

func DefaultParam() Parameters {
	return Parameters{
		RequestRetention: 0.9,
		MaximumInterval:  36500,
		EasyBonus:        1.3,
		HardFactor:       1.2,
		W:                defaultWeights(),
	}
}

type Card struct {
	Id            int64        `json:"Id"`
	Due           time.Time    `json:"Due"`
	Stability     float64      `json:"Stability"`
	Difficulty    float64      `json:"Difficulty"`
	ElapsedDays   uint64       `json:"ElapsedDays"`
	ScheduledDays uint64       `json:"ScheduledDays"`
	Reps          uint64       `json:"Reps"`
	Lapses        uint64       `json:"Lapses"`
	State         State        `json:"State"`
	LastReview    time.Time    `json:"LastReview"`
	ReviewLogs    []*ReviewLog `json:"ReviewLogs"`
}

func NewCard() Card {
	return Card{
		Id:            time.Now().UnixNano(),
		Due:           time.Time{},
		Stability:     0,
		Difficulty:    0,
		ElapsedDays:   0,
		ScheduledDays: 0,
		Reps:          0,
		Lapses:        0,
		State:         New,
		LastReview:    time.Time{},
		ReviewLogs:    []*ReviewLog{},
	}
}

type ReviewLog struct {
	Id            int64     `json:"Id"`
	CardId        int64     `json:"CardId"`
	Rating        Rating    `json:"Rating"`
	ScheduledDays uint64    `json:"ScheduledDays"`
	ElapsedDays   uint64    `json:"ElapsedDays"`
	Review        time.Time `json:"Review"`
	State         State     `json:"State"`
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
