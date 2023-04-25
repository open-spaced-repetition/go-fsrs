package fsrs

import (
	"math"
	"time"
)

func (p *Parameters) Repeat(card Card, now time.Time) map[Rating]SchedulingInfo {
	if card.State == New {
		card.ElapsedDays = 0
	} else {
		card.ElapsedDays = uint64(math.Round(float64(now.Sub(card.LastReview) / time.Hour / 24)))
	}
	card.LastReview = now
	card.Reps += 1
	s := new(schedulingCards)
	s.init(card)
	s.updateState(card.State)

	switch card.State {
	case New:
		p.initDS(s)

		s.Again.Due = now.Add(1 * time.Minute)
		s.Hard.Due = now.Add(5 * time.Minute)
		s.Good.Due = now.Add(10 * time.Minute)
		easyInterval := p.nextInterval(s.Easy.Stability * p.EasyBonus)
		s.Easy.ScheduledDays = uint64(easyInterval)
		s.Easy.Due = now.Add(time.Duration(easyInterval) * 24 * time.Hour)
	case Learning, Relearning:
		hardInterval := 0.0
		goodInterval := p.nextInterval(s.Good.Stability)
		easyInterval := math.Max(p.nextInterval(s.Easy.Stability*p.EasyBonus), goodInterval+1)

		s.schedule(now, hardInterval, goodInterval, easyInterval)
	case Review:
		interval := float64(card.ElapsedDays)
		lastD := card.Difficulty
		lastS := card.Stability
		retrievability := math.Exp(math.Log(0.9) * interval / lastS)
		p.nextDS(s, lastD, lastS, retrievability)

		hardInterval := p.nextInterval(lastS * p.HardFactor)
		goodInterval := p.nextInterval(s.Good.Stability)
		hardInterval = math.Min(hardInterval, goodInterval)
		goodInterval = math.Max(goodInterval, hardInterval+1)
		easyInterval := math.Max(p.nextInterval(s.Easy.Stability*p.EasyBonus), goodInterval+1)
		s.schedule(now, hardInterval, goodInterval, easyInterval)
	}
	return s.recordLog(card, now)
}

func (s *schedulingCards) updateState(state State) {
	switch state {
	case New:
		s.Again.State = Learning
		s.Hard.State = Learning
		s.Good.State = Learning
		s.Easy.State = Review
		s.Again.Lapses += 1
	case Learning, Relearning:
		s.Again.State = state
		s.Hard.State = state
		s.Good.State = Review
		s.Easy.State = Review
	case Review:
		s.Again.State = Relearning
		s.Hard.State = Review
		s.Good.State = Review
		s.Easy.State = Review
		s.Again.Lapses += 1
	}
}

func (s *schedulingCards) schedule(now time.Time, hardInterval float64, goodInterval float64, easyInterval float64) {
	s.Again.ScheduledDays = 0
	s.Hard.ScheduledDays = uint64(hardInterval)
	s.Good.ScheduledDays = uint64(goodInterval)
	s.Easy.ScheduledDays = uint64(easyInterval)
	s.Again.Due = now.Add(5 * time.Minute)
	if hardInterval > 0 {
		s.Hard.Due = now.Add(time.Duration(hardInterval) * 24 * time.Hour)
	} else {
		s.Hard.Due = now.Add(10 * time.Minute)
	}
	s.Good.Due = now.Add(time.Duration(goodInterval) * 24 * time.Hour)
	s.Easy.Due = now.Add(time.Duration(easyInterval) * 24 * time.Hour)
}

func (s *schedulingCards) recordLog(card Card, now time.Time) map[Rating]SchedulingInfo {
	m := map[Rating]SchedulingInfo{
		Again: {s.Again, ReviewLog{
			Rating:        Again,
			ScheduledDays: s.Again.ScheduledDays,
			ElapsedDays:   card.ElapsedDays,
			Review:        now,
			State:         card.State,
		}},
		Hard: {s.Hard, ReviewLog{
			Rating:        Hard,
			ScheduledDays: s.Hard.ScheduledDays,
			ElapsedDays:   card.ElapsedDays,
			Review:        now,
			State:         card.State,
		}},
		Good: {s.Good, ReviewLog{
			Rating:        Good,
			ScheduledDays: s.Good.ScheduledDays,
			ElapsedDays:   card.ElapsedDays,
			Review:        now,
			State:         card.State,
		}},
		Easy: {s.Easy, ReviewLog{
			Rating:        Easy,
			ScheduledDays: s.Easy.ScheduledDays,
			ElapsedDays:   card.ElapsedDays,
			Review:        now,
			State:         card.State,
		}},
	}
	return m
}

func (p *Parameters) initDS(s *schedulingCards) {
	s.Again.Difficulty = p.initDifficulty(Again)
	s.Again.Stability = p.initStability(Again)
	s.Hard.Difficulty = p.initDifficulty(Hard)
	s.Hard.Stability = p.initStability(Hard)
	s.Good.Difficulty = p.initDifficulty(Good)
	s.Good.Stability = p.initStability(Good)
	s.Easy.Difficulty = p.initDifficulty(Easy)
	s.Easy.Stability = p.initStability(Easy)
}

func (p *Parameters) nextDS(s *schedulingCards, lastD float64, lastS float64, retrievability float64) {
	s.Again.Difficulty = p.nextDifficulty(lastD, Again)
	s.Again.Stability = p.nextForgetStability(s.Again.Difficulty, lastS, retrievability)
	s.Hard.Difficulty = p.nextDifficulty(lastD, Hard)
	s.Hard.Stability = p.nextRecallStability(s.Hard.Difficulty, lastS, retrievability)
	s.Good.Difficulty = p.nextDifficulty(lastD, Good)
	s.Good.Stability = p.nextRecallStability(s.Good.Difficulty, lastS, retrievability)
	s.Easy.Difficulty = p.nextDifficulty(lastD, Easy)
	s.Easy.Stability = p.nextRecallStability(s.Easy.Difficulty, lastS, retrievability)
}

func (p *Parameters) initStability(r Rating) float64 {
	return math.Max(p.W[0]+p.W[1]*float64(r), 0.1)
}
func (p *Parameters) initDifficulty(r Rating) float64 {
	return constrainDifficulty(p.W[2] + p.W[3]*float64(r-2))
}

func constrainDifficulty(d float64) float64 {
	return math.Min(math.Max(d, 1), 10)
}

func (p *Parameters) nextInterval(s float64) float64 {
	newInterval := s * math.Log(p.RequestRetention) / math.Log(0.9)
	return math.Max(math.Min(math.Round(newInterval), p.MaximumInterval), 1)
}

func (p *Parameters) nextDifficulty(d float64, r Rating) float64 {
	nextD := d + p.W[4]*float64(r-2)
	return constrainDifficulty(p.meanReversion(p.W[2], nextD))
}

func (p *Parameters) meanReversion(init float64, current float64) float64 {
	return p.W[5]*init + (1-p.W[5])*current
}

func (p *Parameters) nextRecallStability(d float64, s float64, r float64) float64 {
	return s * (1 + math.Exp(p.W[6])*
		(11-d)*
		math.Pow(s, p.W[7])*
		(math.Exp((1-r)*p.W[8])-1))
}

func (p *Parameters) nextForgetStability(d float64, s float64, r float64) float64 {
	return p.W[9] * math.Pow(d, p.W[10]) * math.Pow(
		s, p.W[11]) * math.Exp((1-r)*p.W[12])
}
