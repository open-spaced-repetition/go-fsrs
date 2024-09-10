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

func (bs basicScheduler) newState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	next.Difficulty = bs.parameters.initDifficulty(grade)
	next.Stability = bs.parameters.initStability(grade)

	switch grade {
	case Again:
		next.ScheduledDays = 0
		next.Due = bs.now.Add(1 * time.Minute)
		next.State = Learning
	case Hard:
		next.ScheduledDays = 0
		next.Due = bs.now.Add(5 * time.Minute)
		next.State = Learning
	case Good:
		next.ScheduledDays = 0
		next.Due = bs.now.Add(10 * time.Minute)
		next.State = Learning
	case Easy:
		easyInterval := bs.parameters.nextInterval(
			next.Stability,
			float64(next.ElapsedDays),
		)
		next.ScheduledDays = uint64(easyInterval)
		next.Due = bs.now.Add(time.Duration(easyInterval) * 24 * time.Hour)
		next.State = Review
	}

	item := SchedulingInfo{
		Card:      next,
		ReviewLog: bs.buildLog(grade),
	}
	bs.next[grade] = item
	return item
}

func (bs basicScheduler) learningState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	interval := float64(bs.current.ElapsedDays)
	next.Difficulty = bs.parameters.nextDifficulty(bs.last.Difficulty, grade)
	next.Stability = bs.parameters.shortTermStability(bs.last.Stability, grade)

	switch grade {
	case Again:
		next.ScheduledDays = 0
		next.Due = bs.now.Add(5 * time.Minute)
		next.State = bs.last.State
	case Hard:
		next.ScheduledDays = 0
		next.Due = bs.now.Add(10 * time.Minute)
		next.State = bs.last.State
	case Good:
		goodInterval := bs.parameters.nextInterval(next.Stability, interval)
		next.ScheduledDays = uint64(goodInterval)
		next.Due = bs.now.Add(time.Duration(goodInterval) * 24 * time.Hour)
		next.State = Review
	case Easy:
		goodStability := bs.parameters.shortTermStability(bs.last.Stability, Good)
		goodInterval := bs.parameters.nextInterval(goodStability, interval)
		easyInterval := math.Max(
			bs.parameters.nextInterval(next.Stability, interval),
			float64(goodInterval)+1,
		)
		next.ScheduledDays = uint64(easyInterval)
		next.Due = bs.now.Add(time.Duration(easyInterval) * 24 * time.Hour)
		next.State = Review
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
	retrievability := bs.parameters.forgettingCurve(interval, stability)

	nextAgain := bs.current
	nextHard := bs.current
	nextGood := bs.current
	nextEasy := bs.current

	bs.nextDs(&nextAgain, &nextHard, &nextGood, &nextEasy, difficulty, stability, retrievability)
	bs.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy, interval)
	bs.nextState(&nextAgain, &nextHard, &nextGood, &nextEasy)
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

func (bs basicScheduler) nextInterval(nextAgain, nextHard, nextGood, nextEasy *Card, elapsedDays float64) {
	hardInterval := bs.parameters.nextInterval(nextHard.Stability, elapsedDays)
	goodInterval := bs.parameters.nextInterval(nextGood.Stability, elapsedDays)
	hardInterval = math.Min(hardInterval, goodInterval)
	goodInterval = math.Max(goodInterval, hardInterval+1)
	easyInterval := math.Max(
		bs.parameters.nextInterval(nextEasy.Stability, elapsedDays),
		goodInterval+1,
	)

	nextAgain.ScheduledDays = 0
	nextAgain.Due = bs.now.Add(5 * time.Minute)

	nextHard.ScheduledDays = uint64(hardInterval)
	nextHard.Due = bs.now.Add(time.Duration(hardInterval) * 24 * time.Hour)

	nextGood.ScheduledDays = uint64(goodInterval)
	nextGood.Due = bs.now.Add(time.Duration(goodInterval) * 24 * time.Hour)

	nextEasy.ScheduledDays = uint64(easyInterval)
	nextEasy.Due = bs.now.Add(time.Duration(easyInterval) * 24 * time.Hour)
}

func (bs basicScheduler) nextState(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.State = Relearning
	nextHard.State = Review
	nextGood.State = Review
	nextEasy.State = Review
}
