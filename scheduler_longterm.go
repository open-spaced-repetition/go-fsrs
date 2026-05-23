package fsrs

import (
	"time"
)

type longTermScheduler struct {
	*Scheduler
}

var _ implScheduler = longTermScheduler{}

// NewLongTermScheduler creates a Scheduler using the long-term FSRS variant
// for the given card and reference time. This variant is selected when
// Parameters.EnableShortTerm is false. Call Preview or Review on the returned
// Scheduler to obtain scheduling results.
func (p *Parameters) NewLongTermScheduler(card Card, now time.Time) *Scheduler {
	return p.newScheduler(card, now, func(s *Scheduler) implScheduler {
		return longTermScheduler{s}
	})
}

func (lts longTermScheduler) newState(grade Rating) SchedulingInfo {
	exist, ok := lts.next[grade]
	if ok {
		return exist
	}

	lts.current.ScheduledDays = 0

	nextAgain := lts.current
	nextHard := lts.current
	nextGood := lts.current
	nextEasy := lts.current

	lts.initDs(&nextAgain, &nextHard, &nextGood, &nextEasy)

	lts.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy, 0)
	setReviewState(&nextAgain, &nextHard, &nextGood, &nextEasy)
	lts.updateNext(&nextAgain, &nextHard, &nextGood, &nextEasy)

	return lts.next[grade]
}

func (lts longTermScheduler) initDs(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.Difficulty = constrainDifficulty(lts.parameters.initDifficulty(Again))
	nextAgain.Stability = lts.parameters.initStability(Again)

	nextHard.Difficulty = constrainDifficulty(lts.parameters.initDifficulty(Hard))
	nextHard.Stability = lts.parameters.initStability(Hard)

	nextGood.Difficulty = constrainDifficulty(lts.parameters.initDifficulty(Good))
	nextGood.Stability = lts.parameters.initStability(Good)

	nextEasy.Difficulty = constrainDifficulty(lts.parameters.initDifficulty(Easy))
	nextEasy.Stability = lts.parameters.initStability(Easy)
}

func (lts longTermScheduler) learningState(grade Rating) SchedulingInfo {
	return lts.reviewState(grade)
}

func (lts longTermScheduler) reviewState(grade Rating) SchedulingInfo {
	exist, ok := lts.next[grade]
	if ok {
		return exist
	}

	elapsedDays := lts.elapsedDays()
	difficulty := lts.last.Difficulty
	stability := lts.last.Stability
	retrievability := lts.parameters.ForgettingCurve(elapsedDays, stability)

	nextAgain := lts.current
	nextHard := lts.current
	nextGood := lts.current
	nextEasy := lts.current

	lts.Scheduler.nextDs(&nextAgain, &nextHard, &nextGood, &nextEasy, difficulty, stability, retrievability)
	lts.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy, elapsedDays)
	setReviewState(&nextAgain, &nextHard, &nextGood, &nextEasy)
	nextAgain.Lapses++

	lts.updateNext(&nextAgain, &nextHard, &nextGood, &nextEasy)
	return lts.next[grade]
}

func (lts longTermScheduler) nextInterval(nextAgain, nextHard, nextGood, nextEasy *Card, elapsedDays float64) {
	againInterval := lts.parameters.nextInterval(nextAgain.Stability, elapsedDays)
	hardInterval := lts.parameters.nextInterval(nextHard.Stability, elapsedDays)
	goodInterval := lts.parameters.nextInterval(nextGood.Stability, elapsedDays)
	easyInterval := lts.parameters.nextInterval(nextEasy.Stability, elapsedDays)

	againInterval = min(againInterval, hardInterval)
	hardInterval = max(hardInterval, againInterval+1)
	goodInterval = max(goodInterval, hardInterval+1)
	easyInterval = max(easyInterval, goodInterval+1)

	nextAgain.ScheduledDays = uint64(againInterval)
	nextAgain.Due = lts.now.Add(daysToDuration(againInterval, lts.parameters.MaximumInterval))

	nextHard.ScheduledDays = uint64(hardInterval)
	nextHard.Due = lts.now.Add(daysToDuration(hardInterval, lts.parameters.MaximumInterval))

	nextGood.ScheduledDays = uint64(goodInterval)
	nextGood.Due = lts.now.Add(daysToDuration(goodInterval, lts.parameters.MaximumInterval))

	nextEasy.ScheduledDays = uint64(easyInterval)
	nextEasy.Due = lts.now.Add(daysToDuration(easyInterval, lts.parameters.MaximumInterval))
}

func setReviewState(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.State = Review
	nextAgain.RemainingSteps = 0
	nextHard.State = Review
	nextHard.RemainingSteps = 0
	nextGood.State = Review
	nextGood.RemainingSteps = 0
	nextEasy.State = Review
	nextEasy.RemainingSteps = 0
}

func (lts longTermScheduler) updateNext(nextAgain, nextHard, nextGood, nextEasy *Card) {
	itemAgain := SchedulingInfo{
		Card:      *nextAgain,
		ReviewLog: lts.buildLog(Again),
	}
	itemHard := SchedulingInfo{
		Card:      *nextHard,
		ReviewLog: lts.buildLog(Hard),
	}
	itemGood := SchedulingInfo{
		Card:      *nextGood,
		ReviewLog: lts.buildLog(Good),
	}
	itemEasy := SchedulingInfo{
		Card:      *nextEasy,
		ReviewLog: lts.buildLog(Easy),
	}

	lts.next[Again] = itemAgain
	lts.next[Hard] = itemHard
	lts.next[Good] = itemGood
	lts.next[Easy] = itemEasy
}
