package fsrs

import (
	"math"
	"time"
)

const requestRetention = 0.9
const maximumInterval = 36500.0
const easyBonus = 1.3
const hardFactor = 1.2

var intervalModifier = math.Log(requestRetention) / math.Log(0.9)

func (w *Weights) Repeat(card *Card, now time.Time) *SchedulingCards {

	schedulingCards := new(SchedulingCards)
	schedulingCards.init(card)
	schedulingCards.updateState(card.State)

	switch card.State {
	case New:
		w.initDS(schedulingCards)

		easyInterval := w.nextInterval(schedulingCards.Easy.Stability * easyBonus)
		schedulingCards.Easy.ScheduledDays = uint64(easyInterval)
		schedulingCards.Easy.Due = now.Add(time.Duration(float64(easyInterval) * float64(24*time.Hour)))
		return schedulingCards
	case Learning, Relearning:
		hardInterval := w.nextInterval(schedulingCards.Hard.Stability)
		goodInterval := math.Max(w.nextInterval(schedulingCards.Good.Stability), hardInterval+1)
		easyInterval := math.Max(w.nextInterval(schedulingCards.Easy.Stability*easyBonus), goodInterval+1)

		schedulingCards.schedule(now, hardInterval, goodInterval, easyInterval)
		return schedulingCards
	case Review:
		interval := float64(now.Sub(card.LastReview)/time.Hour/24 + 1.0)
		lastD := card.Difficulty
		lastS := card.Stability
		retrievability := math.Exp(math.Log(0.9) * interval / lastS)
		w.nextDS(schedulingCards, lastD, lastS, retrievability)

		hardInterval := w.nextInterval(lastS * hardFactor)
		goodInterval := math.Max(w.nextInterval(schedulingCards.Good.Stability), hardInterval+1)
		easyInterval := math.Max(w.nextInterval(schedulingCards.Easy.Stability*easyBonus), goodInterval+1)
		schedulingCards.schedule(now, hardInterval, goodInterval, easyInterval)

		return schedulingCards
	}
	return schedulingCards
}

func (s *SchedulingCards) updateState(state State) {
	switch state {
	case New:
		s.Again.State = Learning
		s.Hard.State = Learning
		s.Good.State = Learning
		s.Easy.State = Review
	case Learning, Relearning:
		s.Again.State = state
		s.Hard.State = Review
		s.Good.State = Review
		s.Easy.State = Review
	case Review:
		s.Again.State = Relearning
		s.Hard.State = Review
		s.Good.State = Review
		s.Easy.State = Review
	}
}

func (s *SchedulingCards) schedule(now time.Time, hardInterval float64, goodInterval float64, easyInterval float64) {
	s.Hard.ScheduledDays = uint64(hardInterval)
	s.Good.ScheduledDays = uint64(goodInterval)
	s.Easy.ScheduledDays = uint64(easyInterval)
	s.Hard.Due = now.Add(time.Duration(hardInterval * float64(24*time.Hour)))
	s.Good.Due = now.Add(time.Duration(goodInterval * float64(24*time.Hour)))
	s.Easy.Due = now.Add(time.Duration(easyInterval * float64(24*time.Hour)))
}

func (w *Weights) initDS(s *SchedulingCards) {
	s.Again.Difficulty = w.initDifficulty(Again)
	s.Again.Stability = w.initStability(Again)
	s.Hard.Difficulty = w.initDifficulty(Hard)
	s.Hard.Stability = w.initStability(Hard)
	s.Good.Difficulty = w.initDifficulty(Good)
	s.Good.Stability = w.initStability(Good)
	s.Easy.Difficulty = w.initDifficulty(Easy)
	s.Easy.Stability = w.initStability(Easy)
}

func (w *Weights) nextDS(s *SchedulingCards, lastD float64, lastS float64, retrievability float64) {
	s.Again.Difficulty = w.nextDifficulty(lastD, Again)
	s.Again.Stability = w.nextForgetStability(s.Again.Difficulty, lastS, retrievability)
	s.Hard.Difficulty = w.nextDifficulty(lastD, Hard)
	s.Hard.Stability = w.nextRecallStability(s.Hard.Difficulty, lastS, retrievability)
	s.Good.Difficulty = w.nextDifficulty(lastD, Good)
	s.Good.Stability = w.nextRecallStability(s.Good.Difficulty, lastS, retrievability)
	s.Easy.Difficulty = w.nextDifficulty(lastD, Easy)
	s.Easy.Stability = w.nextRecallStability(s.Easy.Difficulty, lastS, retrievability)
}

func (w *Weights) initStability(r Rating) float64 {
	return math.Max(w[0]+w[1]*float64(r), 0.1)
}
func (w *Weights) initDifficulty(r Rating) float64 {
	return constrainDifficulty(w[2] + w[3]*float64(r-2))
}

func constrainDifficulty(d float64) float64 {
	return math.Min(math.Max(d, 1), 10)
}

func (w *Weights) nextInterval(s float64) float64 {
	newInterval := s * intervalModifier
	return math.Max(math.Min(math.Round(newInterval), maximumInterval), 1)
}

func (w *Weights) nextDifficulty(d float64, r Rating) float64 {
	nextD := d + w[4]*float64(r-2)
	return constrainDifficulty(w.meanReversion(w[2], nextD))
}

func (w *Weights) meanReversion(init float64, current float64) float64 {
	return w[5]*init + (1-w[5])*current
}

func (w *Weights) nextRecallStability(d float64, s float64, r float64) float64 {
	return s * (1 + math.Exp(w[6])*
		(11-d)*
		math.Pow(s, w[7])*
		(math.Exp((1-r)*w[8])-1))
}

func (w *Weights) nextForgetStability(d float64, s float64, r float64) float64 {
	return w[9] * math.Pow(d, w[10]) * math.Pow(
		s, w[11]) * math.Exp((1-r)*w[12])
}
