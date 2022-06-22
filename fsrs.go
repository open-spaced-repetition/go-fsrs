package fsrs

import (
	"fmt"
	"math"
	"time"

	"github.com/ImSingee/go-ex/set"
)

func (globalData *GlobalData) Learn(cardData *CardData, grade Grade) {
	if grade < GradeNewCard || grade > GradeEasy {
		panic(fmt.Sprintf("Invalid grade: %d", grade))
	}

	now := time.Now()

	if grade == GradeNewCard { // learn new card
		addDay := math.Round(globalData.DefaultStability * math.Log(globalData.RequestRetention) / math.Log(0.9))

		cardData.Due = now.Add(time.Duration(addDay * float64(24*time.Hour)))
		cardData.Interval = 0
		cardData.Difficulty = globalData.DefaultDifficulty
		cardData.Stability = globalData.DefaultStability
		cardData.Retrievability = 1
		cardData.LastGrade = GradeNewCard
		cardData.Review = now
		cardData.Reps = 1
		cardData.Lapses = 0

		return
	}

	// review card after learn
	lastDifficulty := cardData.Difficulty
	lastStability := cardData.Stability
	lastLapses := cardData.Lapses
	lastReps := cardData.Reps
	lastReview := cardData.Review

	h := cardData.CardDataItem
	cardData.History = append(cardData.History, &h)

	diffDay := (time.Since(lastReview) / time.Hour / 24) + 1
	if diffDay > 0 {
		cardData.Interval = uint64(diffDay)
	} else {
		cardData.Interval = 0
	}

	cardData.Review = now
	cardData.Retrievability = math.Exp(math.Log(0.9) * float64(cardData.Interval) / lastStability)
	cardData.Difficulty = math.Min(math.Max(lastDifficulty+cardData.Retrievability-float64(grade)+0.2, 1), 10)

	if grade == GradeForgetting {
		cardData.Stability = globalData.DefaultStability * math.Exp(-0.3*float64(lastLapses+1))

		if lastReps > 1 {
			globalData.TotalDiff = globalData.TotalDiff - cardData.Retrievability
		}

		cardData.Lapses = lastLapses + 1
		cardData.Reps = 1

	} else { //grade == 1 || grade == 2
		cardData.Stability = lastStability * (1 + globalData.IncreaseFactor*math.Pow(cardData.Difficulty, globalData.DifficultyDecay)*math.Pow(lastStability, globalData.StabilityDecay)*(math.Exp(1-cardData.Retrievability)-1))

		if lastReps > 1 {
			globalData.TotalDiff = globalData.TotalDiff + 1 - cardData.Retrievability
		}

		cardData.Lapses = lastLapses
		cardData.Reps = lastReps + 1
	}

	globalData.TotalCase++
	globalData.TotalReview++

	addDay := math.Round(cardData.Stability * math.Log(globalData.RequestRetention) / math.Log(0.9))
	cardData.Due = now.Add(time.Duration(addDay * float64(24*time.Hour)))

	// Adaptive globalData.defaultDifficulty
	if globalData.TotalCase > 100 {
		globalData.DefaultDifficulty = 1.0/math.Pow(float64(globalData.TotalReview), 0.3)*math.Pow(math.Log(globalData.RequestRetention)/math.Max(math.Log(globalData.RequestRetention+globalData.TotalDiff/float64(globalData.TotalCase)), 0), 1/globalData.DifficultyDecay)*5 + (1-1/math.Pow(float64(globalData.TotalReview), 0.3))*globalData.DefaultDifficulty

		globalData.TotalDiff = 0
		globalData.TotalCase = 0
	}

	// Adaptive globalData.defaultStability
	if lastReps == 1 && lastLapses == 0 {
		retrievability := uint64(0)
		if grade > GradeForgetting {
			retrievability = 1
		}
		globalData.StabilityDataArray = append(globalData.StabilityDataArray, &StabilityData{
			Interval:       cardData.Interval,
			Retrievability: retrievability,
		})

		if len(globalData.StabilityDataArray) > 0 && len(globalData.StabilityDataArray)%50 == 0 {
			intervalSetArray := set.New[uint64]()

			sumRI2S := float64(0)
			sumI2S := float64(0)

			for s := 0; s < len(globalData.StabilityDataArray); s++ {
				ivl := globalData.StabilityDataArray[s].Interval

				if !intervalSetArray.Has(ivl) {
					intervalSetArray.Add(ivl)

					retrievabilitySum := uint64(0)
					currentCount := 0
					for _, fi := range globalData.StabilityDataArray {
						if fi.Interval == ivl {
							retrievabilitySum += fi.Retrievability
							currentCount++
						}
					}

					if retrievabilitySum > 0 {
						sumRI2S = sumRI2S + float64(ivl)*math.Log(float64(retrievabilitySum)/float64(currentCount))*float64(currentCount)
						sumI2S = sumI2S + float64(ivl*ivl)*float64(currentCount)
					}
				}

			}

			globalData.DefaultStability = (math.Max(math.Log(0.9)/(sumRI2S/sumI2S), 0.1) + globalData.DefaultStability) / 2
		}
	}
}
