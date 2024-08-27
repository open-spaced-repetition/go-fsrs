package fsrs

import (
	"math"
	"time"
)

type AbstractScheduler struct {
	last            Card
	current         Card
	now             time.Time
	next            RecordLog
	algorithm       Parameters
	newStateFn      func(Rating) SchedulingInfo
	learningStateFn func(Rating) SchedulingInfo
	reviewStateFn   func(Rating) SchedulingInfo
}

func NewAbstractScheduler(card Card, now time.Time, algorithm Parameters) *AbstractScheduler {
	as := &AbstractScheduler{
		algorithm: algorithm,
		next:      make(RecordLog),
	}
	as.last = card
	as.current = card
	as.now = now
	as.init()
	return as
}

func (as *AbstractScheduler) init() {
	state := as.current.State
	lastReview := as.current.LastReview
	var interval float64 = 0 // card.state === State.New => 0
	if state != New && !lastReview.IsZero() {
		interval = math.Floor(as.now.Sub(lastReview).Hours() / 24)
	}
	as.current.LastReview = as.now
	as.current.ElapsedDays = uint64(interval)
	as.current.Reps++
}

func (as *AbstractScheduler) Preview() RecordLog {
	return RecordLog{
		Again: as.Review(Again),
		Hard:  as.Review(Hard),
		Good:  as.Review(Good),
		Easy:  as.Review(Easy),
	}
}

func (as *AbstractScheduler) Review(grade Rating) SchedulingInfo {
	state := as.last.State
	var item SchedulingInfo
	switch state {
	case New:
		item = as.newStateFn(grade)
	case Learning, Relearning:
		item = as.learningStateFn(grade)
	case Review:
		item = as.reviewStateFn(grade)
	}
	return item
}

func (as *AbstractScheduler) buildLog(rating Rating) ReviewLog {
	return ReviewLog{
		Rating:        rating,
		State:         as.current.State,
		ElapsedDays:   as.current.ElapsedDays,
		ScheduledDays: as.current.ScheduledDays,
		Review:        as.now,
	}
}

type BasicScheduler struct {
	AbstractScheduler
}

func (bs *BasicScheduler) newState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	next.Difficulty = bs.algorithm.initDifficulty(grade)
	next.Stability = bs.algorithm.initStability(grade)

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
		easyInterval := bs.algorithm.nextInterval(
			next.Stability,
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

func (bs *BasicScheduler) learningState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	next := bs.current
	next.Difficulty = bs.algorithm.nextDifficulty(bs.last.Difficulty, grade)
	next.Stability = bs.algorithm.shortTermStability(bs.last.Stability, grade)

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
		goodInterval := bs.algorithm.nextInterval(next.Stability)
		next.ScheduledDays = uint64(goodInterval)
		next.Due = bs.now.Add(time.Duration(goodInterval) * 24 * time.Hour)
		next.State = Review
	case Easy:
		goodStability := bs.algorithm.shortTermStability(bs.last.Stability, Good)
		goodInterval := bs.algorithm.nextInterval(goodStability)
		easyInterval := math.Max(
			bs.algorithm.nextInterval(next.Stability),
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

func (bs *BasicScheduler) reviewState(grade Rating) SchedulingInfo {
	exist, ok := bs.next[grade]
	if ok {
		return exist
	}

	interval := float64(bs.current.ElapsedDays)
	difficulty := bs.last.Difficulty
	stability := bs.last.Stability
	retrievability := bs.algorithm.forgettingCurve(interval, stability)

	nextAgain := bs.current
	nextHard := bs.current
	nextGood := bs.current
	nextEasy := bs.current

	bs.nextDs(&nextAgain, &nextHard, &nextGood, &nextEasy, difficulty, stability, retrievability)
	bs.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy)
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

func (bs *BasicScheduler) nextDs(nextAgain, nextHard, nextGood, nextEasy *Card, difficulty, stability, retrievability float64) {
	nextAgain.Difficulty = bs.algorithm.nextDifficulty(difficulty, Again)
	nextAgain.Stability = bs.algorithm.nextForgetStability(difficulty, stability, retrievability)

	nextHard.Difficulty = bs.algorithm.nextDifficulty(difficulty, Hard)
	nextHard.Stability = bs.algorithm.nextRecallStability(difficulty, stability, retrievability, Hard)

	nextGood.Difficulty = bs.algorithm.nextDifficulty(difficulty, Good)
	nextGood.Stability = bs.algorithm.nextRecallStability(difficulty, stability, retrievability, Good)

	nextEasy.Difficulty = bs.algorithm.nextDifficulty(difficulty, Easy)
	nextEasy.Stability = bs.algorithm.nextRecallStability(difficulty, stability, retrievability, Easy)
}

func (bs *BasicScheduler) nextInterval(nextAgain, nextHard, nextGood, nextEasy *Card) {
	hardInterval := bs.algorithm.nextInterval(nextHard.Stability)
	goodInterval := bs.algorithm.nextInterval(nextGood.Stability)
	hardInterval = math.Min(hardInterval, goodInterval)
	goodInterval = math.Max(goodInterval, hardInterval+1)
	easyInterval := math.Max(
		bs.algorithm.nextInterval(nextEasy.Stability),
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

func (bs *BasicScheduler) nextState(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.State = Relearning
	nextHard.State = Review
	nextGood.State = Review
	nextEasy.State = Review
}

type LongTermScheduler struct {
	AbstractScheduler
}

func (lts *LongTermScheduler) newState(grade Rating) SchedulingInfo {
	exist, ok := lts.next[grade]
	if ok {
		return exist
	}

	lts.current.ScheduledDays = 0
	lts.current.ElapsedDays = 0

	nextAgain := lts.current // 假设有深拷贝方法
	nextHard := lts.current
	nextGood := lts.current
	nextEasy := lts.current

	lts.initDs(&nextAgain, &nextHard, &nextGood, &nextEasy)

	lts.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy)
	lts.nextState(&nextAgain, &nextHard, &nextGood, &nextEasy)
	lts.updateNext(&nextAgain, &nextHard, &nextGood, &nextEasy)

	return lts.next[grade]
}

func (lts *LongTermScheduler) initDs(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.Difficulty = lts.algorithm.initDifficulty(Again)
	nextAgain.Stability = lts.algorithm.initStability(Again)

	nextHard.Difficulty = lts.algorithm.initDifficulty(Hard)
	nextHard.Stability = lts.algorithm.initStability(Hard)

	nextGood.Difficulty = lts.algorithm.initDifficulty(Good)
	nextGood.Stability = lts.algorithm.initStability(Good)

	nextEasy.Difficulty = lts.algorithm.initDifficulty(Easy)
	nextEasy.Stability = lts.algorithm.initStability(Easy)
}

func (lts *LongTermScheduler) LearningState(grade Rating) SchedulingInfo {
	return lts.ReviewState(grade)
}

func (lts *LongTermScheduler) ReviewState(grade Rating) SchedulingInfo {
	exist, ok := lts.next[grade]
	if ok {
		return exist
	}

	interval := float64(lts.current.ElapsedDays)
	difficulty := lts.last.Difficulty
	stability := lts.last.Stability
	retrievability := lts.algorithm.forgettingCurve(interval, stability)

	nextAgain := lts.current
	nextHard := lts.current
	nextGood := lts.current
	nextEasy := lts.current

	lts.nextDs(&nextAgain, &nextHard, &nextGood, &nextEasy, difficulty, stability, retrievability)
	lts.nextInterval(&nextAgain, &nextHard, &nextGood, &nextEasy)
	lts.nextState(&nextAgain, &nextHard, &nextGood, &nextEasy)
	nextAgain.Lapses++

	lts.updateNext(&nextAgain, &nextHard, &nextGood, &nextEasy)
	return lts.next[grade]
}

func (lts *LongTermScheduler) nextDs(nextAgain, nextHard, nextGood, nextEasy *Card, difficulty, stability, retrievability float64) {
	nextAgain.Difficulty = lts.algorithm.nextDifficulty(difficulty, Again)
	nextAgain.Stability = lts.algorithm.nextForgetStability(difficulty, stability, retrievability)

	nextHard.Difficulty = lts.algorithm.nextDifficulty(difficulty, Hard)
	nextHard.Stability = lts.algorithm.nextRecallStability(difficulty, stability, retrievability, Hard)

	nextGood.Difficulty = lts.algorithm.nextDifficulty(difficulty, Good)
	nextGood.Stability = lts.algorithm.nextRecallStability(difficulty, stability, retrievability, Good)

	nextEasy.Difficulty = lts.algorithm.nextDifficulty(difficulty, Easy)
	nextEasy.Stability = lts.algorithm.nextRecallStability(difficulty, stability, retrievability, Easy)
}

func (lts *LongTermScheduler) nextInterval(nextAgain, nextHard, nextGood, nextEasy *Card) {
	againInterval := lts.algorithm.nextInterval(nextAgain.Stability)
	hardInterval := lts.algorithm.nextInterval(nextHard.Stability)
	goodInterval := lts.algorithm.nextInterval(nextGood.Stability)
	easyInterval := lts.algorithm.nextInterval(nextEasy.Stability)

	againInterval = math.Min(againInterval, hardInterval)
	hardInterval = math.Max(hardInterval, againInterval+1)
	goodInterval = math.Max(goodInterval, hardInterval+1)
	easyInterval = math.Max(easyInterval, goodInterval+1)

	nextAgain.ScheduledDays = uint64(againInterval)
	nextAgain.Due = lts.now.Add(time.Duration(againInterval) * 24 * time.Hour)

	nextHard.ScheduledDays = uint64(hardInterval)
	nextHard.Due = lts.now.Add(time.Duration(hardInterval) * 24 * time.Hour)

	nextGood.ScheduledDays = uint64(goodInterval)
	nextGood.Due = lts.now.Add(time.Duration(goodInterval) * 24 * time.Hour)

	nextEasy.ScheduledDays = uint64(easyInterval)
	nextEasy.Due = lts.now.Add(time.Duration(easyInterval) * 24 * time.Hour)
}

func (lts *LongTermScheduler) nextState(nextAgain, nextHard, nextGood, nextEasy *Card) {
	nextAgain.State = Review
	nextHard.State = Review
	nextGood.State = Review
	nextEasy.State = Review
}

func (lts *LongTermScheduler) updateNext(nextAgain, nextHard, nextGood, nextEasy *Card) {
	itemAgain := SchedulingInfo{
		Card:      *nextAgain,
		ReviewLog: lts.buildLog(Again),
	}
	itemHard := SchedulingInfo{
		Card:      *nextHard,
		ReviewLog: lts.AbstractScheduler.buildLog(Hard),
	}
	itemGood := SchedulingInfo{
		Card:      *nextGood,
		ReviewLog: lts.AbstractScheduler.buildLog(Good),
	}
	itemEasy := SchedulingInfo{
		Card:      *nextEasy,
		ReviewLog: lts.AbstractScheduler.buildLog(Easy),
	}

	lts.next[Again] = itemAgain
	lts.next[Hard] = itemHard
	lts.next[Good] = itemGood
	lts.next[Easy] = itemEasy
}

type Scheduler interface {
	Preview() RecordLog
	Review(grade Rating) SchedulingInfo
}

type FSRS struct {
	Parameters
	Scheduler func(card Card, now time.Time, algorithm Parameters) Scheduler
}

func NewFSRS(param Parameters) *FSRS {
	fsrs := &FSRS{
		Parameters: param,
	}
	fsrs.updateScheduler()
	return fsrs
}

func NewBasicScheduler(card Card, now time.Time, algorithm Parameters) Scheduler {
	abstractScheduler := NewAbstractScheduler(card, now, algorithm)
	basicScheduler := &BasicScheduler{
		AbstractScheduler: *abstractScheduler,
	}
	basicScheduler.newStateFn = basicScheduler.newState
	basicScheduler.learningStateFn = basicScheduler.learningState
	basicScheduler.reviewStateFn = basicScheduler.reviewState
	return basicScheduler
}

func NewLongTermScheduler(card Card, now time.Time, algorithm Parameters) Scheduler {
	abstractScheduler := NewAbstractScheduler(card, now, algorithm)
	longTermScheduler := &LongTermScheduler{
		AbstractScheduler: *abstractScheduler,
	}
	longTermScheduler.newStateFn = longTermScheduler.newState
	longTermScheduler.learningStateFn = longTermScheduler.LearningState
	longTermScheduler.reviewStateFn = longTermScheduler.ReviewState
	return longTermScheduler
}

func (f *FSRS) updateScheduler() {
	if f.Parameters.EnableShortTerm {
		f.Scheduler = func(card Card, now time.Time, algorithm Parameters) Scheduler {
			return NewBasicScheduler(card, now, algorithm)
		}
	} else {
		f.Scheduler = func(card Card, now time.Time, algorithm Parameters) Scheduler {
			return NewLongTermScheduler(card, now, algorithm)
		}
	}
}

func (f *FSRS) Repeat(card Card, now time.Time) RecordLog {
	scheduler := f.Scheduler(card, now, f.Parameters)
	recordLog := scheduler.Preview()
	return recordLog
}

func (f *FSRS) Next(card Card, now time.Time, grade Rating) SchedulingInfo {
	scheduler := f.Scheduler(card, now, f.Parameters)
	recordLogItem := scheduler.Review(grade)
	return recordLogItem
}
