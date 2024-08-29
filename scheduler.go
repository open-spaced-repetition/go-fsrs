package fsrs

import (
	"math"
	"time"
)

type Scheduler struct {
	parameters *Parameters

	last    Card
	current Card
	now     time.Time
	next    RecordLog

	impl implScheduler
}

type implScheduler interface {
	newState(Rating) SchedulingInfo
	learningState(Rating) SchedulingInfo
	reviewState(Rating) SchedulingInfo
}

func (s *Scheduler) Preview() RecordLog {
	return RecordLog{
		Again: s.Review(Again),
		Hard:  s.Review(Hard),
		Good:  s.Review(Good),
		Easy:  s.Review(Easy),
	}
}

func (s *Scheduler) Review(grade Rating) SchedulingInfo {
	state := s.last.State
	var item SchedulingInfo
	switch state {
	case New:
		item = s.impl.newState(grade)
	case Learning, Relearning:
		item = s.impl.learningState(grade)
	case Review:
		item = s.impl.reviewState(grade)
	}
	return item
}

func (s *Scheduler) buildLog(rating Rating) ReviewLog {
	return ReviewLog{
		Rating:        rating,
		State:         s.current.State,
		ElapsedDays:   s.current.ElapsedDays,
		ScheduledDays: s.current.ScheduledDays,
		Review:        s.now,
	}
}

func (p *Parameters) newScheduler(card Card, now time.Time, newImpl func(s *Scheduler) implScheduler) *Scheduler {
	s := &Scheduler{
		parameters: p,
		next:       make(RecordLog),
		last:       card,
		current:    card,
		now:        now,
	}

	var interval float64 = 0 // card.state === State.New => 0
	if s.current.State != New && !s.current.LastReview.IsZero() {
		interval = math.Floor(s.now.Sub(s.current.LastReview).Hours() / 24)
	}
	s.current.LastReview = s.now
	s.current.ElapsedDays = uint64(interval)
	s.current.Reps++

	s.impl = newImpl(s)

	return s
}

func (p *Parameters) scheduler(card Card, now time.Time) *Scheduler {
	if p.EnableShortTerm {
		return p.NewBasicScheduler(card, now)
	} else {
		return p.NewLongTermScheduler(card, now)
	}
}
