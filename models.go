package fsrs

import "time"

type GlobalData struct {
	DifficultyDecay    float64          `json:"difficultyDecay"`
	StabilityDecay     float64          `json:"stabilityDecay"`
	IncreaseFactor     float64          `json:"increaseFactor"`
	RequestRetention   float64          `json:"requestRetention"`
	TotalCase          uint64           `json:"totalCase"`
	TotalDiff          float64          `json:"totalDiff"`
	TotalReview        uint64           `json:"totalReview"`
	DefaultDifficulty  float64          `json:"defaultDifficulty"`
	DefaultStability   float64          `json:"defaultStability"`
	StabilityDataArray []*StabilityData `json:"stabilityDataArray"`
}

type StabilityData struct {
	Interval       uint64
	Retrievability uint64
}

type CardData struct {
	CardDataItem

	History []*CardDataItem
}

type CardDataItem struct {
	Due            time.Time
	Interval       uint64 // 上次复习间隔（单位为天）
	Difficulty     float64
	Stability      float64
	Retrievability float64
	LastGrade      Grade // 上次得分
	Review         time.Time
	Reps           uint64
	Lapses         uint64
}

func (item *CardDataItem) Copy() CardDataItem {
	return *item
}

type Grade int8

const (
	GradeForgetting Grade = iota
	GradeRemembered
	GradeEasy

	GradeNewCard Grade = -1
)

// DefaultGlobalData returns the default values of GlobalData
func DefaultGlobalData() GlobalData {
	return GlobalData{
		DifficultyDecay:    -0.7,
		StabilityDecay:     0.2,
		IncreaseFactor:     60,
		RequestRetention:   0.9,
		TotalCase:          0,
		TotalDiff:          0,
		TotalReview:        0,
		DefaultDifficulty:  5,
		DefaultStability:   2,
		StabilityDataArray: nil,
	}
}
