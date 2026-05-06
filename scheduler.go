package fsrs

import (
	"fmt"
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

func (s *Scheduler) initSeed() {
	t := s.now
	reps := s.current.Reps
	mul := s.current.Difficulty * s.current.Stability
	s.parameters.seed = fmt.Sprintf("%d_%d_%f", t.UnixMilli(), reps, mul)
}

func (s *Scheduler) buildLog(rating Rating) ReviewLog {
	due := s.last.Due
	if !s.last.LastReview.IsZero() {
		due = s.last.LastReview
	}
	return ReviewLog{
		Rating:         rating,
		Due:            due,
		ScheduledDays:  s.current.ScheduledDays,
		ElapsedDays:    s.current.ElapsedDays,
		Review:         s.now,
		State:          s.current.State,
		Stability:      s.current.Stability,
		Difficulty:     s.current.Difficulty,
		RemainingSteps: s.current.RemainingSteps,
	}
}

func (p *Parameters) newScheduler(card Card, now time.Time, newImpl func(s *Scheduler) implScheduler) *Scheduler {
	localParams := *p

	s := &Scheduler{
		parameters: &localParams,
		next:       make(RecordLog),
		last:       card,
		current:    card,
		now:        now,
	}

	var interval float64 = 0
	if s.current.State != New && !s.current.LastReview.IsZero() {
		interval = float64(dateDiffInDays(s.current.LastReview, s.now))
	}
	s.current.LastReview = s.now
	s.current.ElapsedDays = uint64(interval)
	s.current.Reps++
	s.initSeed()

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
