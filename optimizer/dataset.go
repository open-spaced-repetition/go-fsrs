package optimizer

import (
	"math"
	"sort"
)

// PrepareTrainingData separates items into initialization set and training set
// Items with exactly 1 long-term review go to initialization set
// The rest go to training set after filtering outliers
func PrepareTrainingData(items []FSRSItem) (initItems, trainItems []FSRSItem) {
	initItems = make([]FSRSItem, 0)
	trainItems = make([]FSRSItem, 0)

	for _, item := range items {
		if item.LongTermReviewCount() == 1 {
			initItems = append(initItems, item)
		} else if item.LongTermReviewCount() > 1 {
			trainItems = append(trainItems, item)
		}
	}

	// Filter outliers from both sets
	initItems, trainItems = filterOutlier(initItems, trainItems)

	return initItems, trainItems
}

// filterOutlier removes statistical outliers based on R-matrix binning
func filterOutlier(initItems, trainItems []FSRSItem) ([]FSRSItem, []FSRSItem) {
	// Combine all items for R-matrix calculation
	allItems := append(initItems, trainItems...)

	// Build R-matrix: map of (deltaTBin, lengthBin, lapseBin) -> (sum, count)
	type rMatrixKey struct {
		deltaTBin, lengthBin, lapseBin uint32
	}
	type rMatrixValue struct {
		sum   float64
		count int
	}
	rMatrix := make(map[rMatrixKey]*rMatrixValue)

	for _, item := range allItems {
		current := item.Current()
		if current == nil {
			continue
		}

		key := rMatrixKey{}
		key.deltaTBin, key.lengthBin, key.lapseBin = item.RMatrixIndex()

		if _, ok := rMatrix[key]; !ok {
			rMatrix[key] = &rMatrixValue{}
		}

		// Label: 1 if recalled (rating > 1), 0 otherwise
		if current.Rating > 1 {
			rMatrix[key].sum += 1
		}
		rMatrix[key].count++
	}

	// Filter function - keep items that are not extreme outliers
	filterFunc := func(items []FSRSItem) []FSRSItem {
		result := make([]FSRSItem, 0, len(items))
		for _, item := range items {
			current := item.Current()
			if current == nil {
				continue
			}

			key := rMatrixKey{}
			key.deltaTBin, key.lengthBin, key.lapseBin = item.RMatrixIndex()

			if val, ok := rMatrix[key]; ok && val.count >= 10 {
				// Calculate expected recall rate
				recall := val.sum / float64(val.count)
				// Skip if recall is at extreme bounds (likely outlier)
				if recall <= 0.01 || recall >= 0.99 {
					continue
				}
			}
			// Keep the item
			result = append(result, item)
		}
		return result
	}

	return filterFunc(initItems), filterFunc(trainItems)
}

// CalculateAverageRecall computes the average recall rate across all items
func CalculateAverageRecall(items []FSRSItem) float64 {
	totalRecall := 0
	totalReviews := 0

	for _, item := range items {
		current := item.Current()
		if current == nil {
			continue
		}

		if current.Rating > 1 {
			totalRecall++
		}
		totalReviews++
	}

	if totalReviews == 0 {
		return 0.0
	}

	return float64(totalRecall) / float64(totalReviews)
}

// RecencyWeightedItems applies recency weighting to items
// More recent items get higher weights
func RecencyWeightedItems(items []FSRSItem) []WeightedFSRSItem {
	if len(items) == 0 {
		return nil
	}

	// Sort by review count (proxy for recency - more reviews = older card)
	type indexedItem struct {
		index int
		count int
	}
	indexed := make([]indexedItem, len(items))
	for i, item := range items {
		indexed[i] = indexedItem{index: i, count: len(item.Reviews)}
	}

	// Sort by count ascending (fewer reviews = newer cards)
	sort.Slice(indexed, func(i, j int) bool {
		return indexed[i].count < indexed[j].count
	})

	// Calculate recency weight based on sorted position
	// Items are weighted by their relative position after sorting
	result := make([]WeightedFSRSItem, len(items))
	n := float64(len(items))

	for i, idxItem := range indexed {
		// Linear recency weight: newer items (fewer reviews) get higher weight
		// Weight ranges from 0.5 to 1.5
		position := float64(i) / n
		weight := 0.5 + position

		result[idxItem.index] = WeightedFSRSItem{
			Item:   items[idxItem.index],
			Weight: weight,
		}
	}

	return result
}

// AverageRecall holds recall statistics for a specific delta_t
type AverageRecall struct {
	DeltaT float64
	Recall float64
	Count  float64
}

// GroupByFirstRating groups items by their first review rating
// Returns a map of rating -> []AverageRecall (sorted by delta_t)
func GroupByFirstRating(items []FSRSItem) map[uint32][]AverageRecall {
	// Group by first rating -> delta_t -> recalls
	type groupKey struct {
		firstRating uint32
		deltaT      uint32
	}
	groups := make(map[groupKey][]int) // 0 = fail, 1 = pass

	for _, item := range items {
		if len(item.Reviews) < 2 {
			continue
		}

		firstRating := item.Reviews[0].Rating
		firstLongTerm := item.FirstLongTermReview()
		if firstLongTerm == nil {
			continue
		}

		key := groupKey{
			firstRating: firstRating,
			deltaT:      firstLongTerm.DeltaT,
		}

		label := 0
		if firstLongTerm.Rating > 1 {
			label = 1
		}

		groups[key] = append(groups[key], label)
	}

	// Convert to AverageRecall per rating
	result := make(map[uint32][]AverageRecall)

	for key, labels := range groups {
		sum := 0
		for _, l := range labels {
			sum += l
		}
		avg := float64(sum) / float64(len(labels))

		ar := AverageRecall{
			DeltaT: float64(key.deltaT),
			Recall: avg,
			Count:  float64(len(labels)),
		}

		result[key.firstRating] = append(result[key.firstRating], ar)
	}

	// Sort each rating's data by delta_t
	for rating := range result {
		sort.Slice(result[rating], func(i, j int) bool {
			return result[rating][i].DeltaT < result[rating][j].DeltaT
		})
	}

	return result
}

// TotalRatingCount counts total reviews per first rating
func TotalRatingCount(groupedData map[uint32][]AverageRecall) map[uint32]uint32 {
	result := make(map[uint32]uint32)

	for rating, data := range groupedData {
		var count float64
		for _, d := range data {
			count += d.Count
		}
		result[rating] = uint32(count)
	}

	return result
}

// ShuffleItems shuffles items in place using Fisher-Yates algorithm
func ShuffleItems(items []WeightedFSRSItem, seed int64) {
	// Simple LCG random number generator
	rng := uint64(seed)
	next := func() uint64 {
		rng = rng*6364136223846793005 + 1442695040888963407
		return rng
	}

	n := len(items)
	for i := n - 1; i > 0; i-- {
		j := int(next() % uint64(i+1))
		items[i], items[j] = items[j], items[i]
	}
}

// BatchItems splits items into batches of specified size
func BatchItems(items []WeightedFSRSItem, batchSize int) [][]WeightedFSRSItem {
	if batchSize <= 0 {
		batchSize = 512
	}

	numBatches := (len(items) + batchSize - 1) / batchSize
	batches := make([][]WeightedFSRSItem, 0, numBatches)

	for i := 0; i < len(items); i += batchSize {
		end := i + batchSize
		if end > len(items) {
			end = len(items)
		}
		batches = append(batches, items[i:end])
	}

	return batches
}

// FilterBySeqLen removes items with sequence length > maxLen
func FilterBySeqLen(items []WeightedFSRSItem, maxLen int) []WeightedFSRSItem {
	result := make([]WeightedFSRSItem, 0, len(items))
	for _, item := range items {
		if len(item.Item.Reviews) <= maxLen {
			result = append(result, item)
		}
	}
	return result
}

// Clamp restricts a value to a range
func Clamp(value, min, max float64) float64 {
	return math.Max(min, math.Min(max, value))
}
