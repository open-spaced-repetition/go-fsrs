package fsrs

import (
	"math"
	"time"
)

type basicScheduler struct {
	*Scheduler
}

var _ implScheduler = basicScheduler{}

func (p *Parameters) NewBasicScheduler(card Card, now time.Time) *Scheduler {
	return p.newScheduler(card, now, func(s *Scheduler) implScheduler {
		return basicScheduler{s}
	})
}

func (bs basicScheduler) applyStep(next *Card, delayMinutes float64, toState State) {
	next.Due = bs.now.Add(minutesToDuration(delayMinutes))
	if delayMinutes >= 1440 {
		next.ScheduledDays = uint64(math.Floor(delayMinutes / 1440))
		next.State = Review
	} else {
		next.ScheduledDays = 0
		next.State = toState
	}
}

func (bs basicScheduler) graduateToReview(next *Card, stability, elapsedDays float64) {
	ivl := bs.parameters.nextInterval(stability, elapsedDays)
	next.ScheduledDays = uint64(ivl)
	next.Due = bs.now.Add(daysToDuration(ivl, bs.parameters.MaximumInterval))
	next.State = Review
	next.RemainingSteps = 0
}

func (bs basicScheduler) newState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	next.Difficulty = bs.parameters.initDifficulty(grade)
	next.Stability = bs.parameters.initStability(grade)
	steps := bs.parameters.LearningSteps

	switch grade {
	case Again:
		if len(steps) == 0 {
			bs.graduateToReview(&next, next.Stability, float64(next.ElapsedDays))
		} else {
			next.RemainingSteps = len(steps)
			bs.applyStep(&next, bs.parameters.againDelayMinutes(steps), Learning)
		}
	case Hard:
		if len(steps) == 0 {
			bs.graduateToReview(&next, next.Stability, float64(next.ElapsedDays))
		} else {
			next.RemainingSteps = len(steps)
			bs.applyStep(&next, bs.parameters.hardDelayMinutes(steps), Learning)
		}
	case Good:
		if delay, ok := bs.parameters.goodDelayMinutes(steps, len(steps)); ok {
			next.RemainingSteps = len(steps) - 1
			bs.applyStep(&next, delay, Learning)
		} else {
			bs.graduateToReview(&next, next.Stability, float64(next.ElapsedDays))
		}
	case Easy:
		bs.graduateToReview(&next, next.Stability, float64(next.ElapsedDays))
	}

	item := SchedulingInfo{
		Card:      next,
		ReviewLog: bs.buildLog(grade),
	}
	bs.next[grade] = item
	return item
}

func (bs basicScheduler) computeLearningStability(grade Rating, interval float64, retrievability float64) float64 {
	if interval == 0 {
		return bs.parameters.shortTermStability(bs.last.Stability, grade)
	}
	if grade == Again {
		return bs.parameters.nextForgetStability(bs.last.Difficulty, bs.last.Stability, retrievability)
	}
	return bs.parameters.nextRecallStability(bs.last.Difficulty, bs.last.Stability, retrievability, grade)
}

func (bs basicScheduler) learningState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	interval := float64(bs.current.ElapsedDays)
	next.Difficulty = bs.parameters.nextDifficulty(bs.last.Difficulty, grade)

	var retrievability float64
	if interval > 0 {
		retrievability = bs.parameters.ForgettingCurve(interval, bs.last.Stability)
	}
	next.Stability = bs.computeLearningStability(grade, interval, retrievability)

	var steps []float64
	var toState State
	if bs.last.State == Relearning {
		steps = bs.parameters.RelearningSteps
		toState = Relearning
	} else {
		steps = bs.parameters.LearningSteps
		toState = Learning
	}
	remaining := bs.current.RemainingSteps

	switch grade {
	case Again:
		if len(steps) == 0 || remaining <= 0 {
			bs.graduateToReview(&next, next.Stability, interval)
		} else {
			next.RemainingSteps = len(steps)
			bs.applyStep(&next, bs.parameters.againDelayMinutes(steps), toState)
		}
	case Hard:
		if len(steps) == 0 || remaining <= 0 {
			bs.graduateToReview(&next, next.Stability, interval)
		} else {
			bs.applyStep(&next, bs.parameters.hardDelayMinutes(steps), toState)
		}
	case Good:
		if delay, ok := bs.parameters.goodDelayMinutes(steps, remaining); ok {
			next.RemainingSteps = remaining - 1
			bs.applyStep(&next, delay, toState)
		} else {
			bs.graduateToReview(&next, next.Stability, interval)
		}
	case Easy:
		easyInterval := bs.parameters.nextInterval(next.Stability, interval)
		next.ScheduledDays = uint64(easyInterval)
		next.Due = bs.now.Add(daysToDuration(easyInterval, bs.parameters.MaximumInterval))
		next.State = Review
		next.RemainingSteps = 0
	}

	item := SchedulingInfo{
		Card:      next,
		ReviewLog: bs.buildLog(grade),
	}
	bs.next[grade] = item
	return item
}

func (bs basicScheduler) reviewState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	interval := float64(bs.current.ElapsedDays)
	difficulty := bs.last.Difficulty
	stability := bs.last.Stability

	nextAgain := bs.current
	nextHard := bs.current
	nextGood := bs.current
	nextEasy := bs.current

	if interval == 0 {
		nextAgain.Difficulty = bs.parameters.nextDifficulty(difficulty, Again)
		nextAgain.Stability = bs.parameters.shortTermStability(stability, Again)
		nextHard.Difficulty = bs.parameters.nextDifficulty(difficulty, Hard)
		nextHard.Stability = bs.parameters.shortTermStability(stability, Hard)
		nextGood.Difficulty = bs.parameters.nextDifficulty(difficulty, Good)
		nextGood.Stability = bs.parameters.shortTermStability(stability, Good)
		nextEasy.Difficulty = bs.parameters.nextDifficulty(difficulty, Easy)
		nextEasy.Stability = bs.parameters.shortTermStability(stability, Easy)
	} else {
		retrievability := bs.parameters.ForgettingCurve(interval, stability)
		bs.nextDs(&nextAgain, &nextHard, &nextGood, &nextEasy, difficulty, stability, retrievability)
	}

	relearnSteps := bs.parameters.RelearningSteps
	if len(relearnSteps) > 0 {
		nextAgain.RemainingSteps = len(relearnSteps)
		bs.applyStep(&nextAgain, bs.parameters.againDelayMinutes(relearnSteps), Relearning)
	} else {
		bs.graduateToReview(&nextAgain, nextAgain.Stability, interval)
	}

	hardInterval := bs.parameters.nextInterval(nextHard.Stability, interval)
	goodInterval := bs.parameters.nextInterval(nextGood.Stability, interval)
	hardInterval = min(hardInterval, goodInterval)
	goodInterval = max(goodInterval, hardInterval+1)
	easyInterval := max(
		bs.parameters.nextInterval(nextEasy.Stability, interval),
		goodInterval+1,
	)

	nextHard.ScheduledDays = uint64(hardInterval)
	nextHard.Due = bs.now.Add(daysToDuration(hardInterval, bs.parameters.MaximumInterval))
	nextHard.State = Review
	nextHard.RemainingSteps = 0

	nextGood.ScheduledDays = uint64(goodInterval)
	nextGood.Due = bs.now.Add(daysToDuration(goodInterval, bs.parameters.MaximumInterval))
	nextGood.State = Review
	nextGood.RemainingSteps = 0

	nextEasy.ScheduledDays = uint64(easyInterval)
	nextEasy.Due = bs.now.Add(daysToDuration(easyInterval, bs.parameters.MaximumInterval))
	nextEasy.State = Review
	nextEasy.RemainingSteps = 0

	nextAgain.Lapses++

	itemAgain := SchedulingInfo{Card: nextAgain, ReviewLog: bs.buildLog(Again)}
	itemHard := SchedulingInfo{Card: nextHard, ReviewLog: bs.buildLog(Hard)}
	itemGood := SchedulingInfo{Card: nextGood, ReviewLog: bs.buildLog(Good)}
	itemEasy := SchedulingInfo{Card: nextEasy, ReviewLog: bs.buildLog(Easy)}

	bs.next[Again] = itemAgain
	bs.next[Hard] = itemHard
	bs.next[Good] = itemGood
	bs.next[Easy] = itemEasy

	return bs.next[grade]
}

func (bs basicScheduler) nextDs(nextAgain, nextHard, nextGood, nextEasy *Card, difficulty, stability, retrievability float64) {
	nextAgain.Difficulty = bs.parameters.nextDifficulty(difficulty, Again)
	nextAgain.Stability = bs.parameters.nextForgetStability(difficulty, stability, retrievability)

	nextHard.Difficulty = bs.parameters.nextDifficulty(difficulty, Hard)
	nextHard.Stability = bs.parameters.nextRecallStability(difficulty, stability, retrievability, Hard)

	nextGood.Difficulty = bs.parameters.nextDifficulty(difficulty, Good)
	nextGood.Stability = bs.parameters.nextRecallStability(difficulty, stability, retrievability, Good)

	nextEasy.Difficulty = bs.parameters.nextDifficulty(difficulty, Easy)
	nextEasy.Stability = bs.parameters.nextRecallStability(difficulty, stability, retrievability, Easy)
}
