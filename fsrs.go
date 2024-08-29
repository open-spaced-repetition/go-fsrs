package fsrs

import "time"

type FSRS struct {
	Parameters
}

func NewFSRS(param Parameters) *FSRS {
	return &FSRS{
		Parameters: param,
	}
}

func (f *FSRS) Repeat(card Card, now time.Time) RecordLog {
	return f.scheduler(card, now).Preview()
}

func (f *FSRS) Next(card Card, now time.Time, grade Rating) SchedulingInfo {
	return f.scheduler(card, now).Review(grade)
}
