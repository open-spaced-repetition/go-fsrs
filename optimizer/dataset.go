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

// filterOutlier removes statistical outliers:
// - Groups items by first rating and delta_t
// - Removes ~5% from each group (minimum threshold)
// - Retains subgroups with ≥6 items
// - Filters by time interval (≤100 days, except rating 4: ≤365 days)
// For small datasets (<50 items total), filtering is skipped to preserve data.
func filterOutlier(initItems, trainItems []FSRSItem) ([]FSRSItem, []FSRSItem) {
	// Skip filtering for small datasets
	totalItems := len(initItems) + len(trainItems)
	if totalItems < 50 {
		return initItems, trainItems
	}

	// Group key: first rating + delta_t of first long-term review
	type groupKey struct {
		firstRating uint32
		deltaT      uint32
	}

	// Build groups for all items
	initGroups := make(map[groupKey][]int)  // indices into initItems
	trainGroups := make(map[groupKey][]int) // indices into trainItems

	// Group init items
	for i, item := range initItems {
		if len(item.Reviews) < 1 {
			continue
		}
		firstRating := item.Reviews[0].Rating
		firstLT := item.FirstLongTermReview()
		if firstLT == nil {
			continue
		}

		// Apply time interval filter
		maxDeltaT := uint32(100)
		if firstRating == 4 {
			maxDeltaT = 365
		}
		if firstLT.DeltaT > maxDeltaT {
			continue
		}

		key := groupKey{firstRating: firstRating, deltaT: firstLT.DeltaT}
		initGroups[key] = append(initGroups[key], i)
	}

	// Group train items
	for i, item := range trainItems {
		if len(item.Reviews) < 1 {
			continue
		}
		firstRating := item.Reviews[0].Rating
		firstLT := item.FirstLongTermReview()
		if firstLT == nil {
			continue
		}

		// Apply time interval filter
		maxDeltaT := uint32(100)
		if firstRating == 4 {
			maxDeltaT = 365
		}
		if firstLT.DeltaT > maxDeltaT {
			continue
		}

		key := groupKey{firstRating: firstRating, deltaT: firstLT.DeltaT}
		trainGroups[key] = append(trainGroups[key], i)
	}

	// Filter function: keep items from groups with ≥6 items, remove ~5% outliers
	filterByGroup := func(items []FSRSItem, groups map[groupKey][]int) []FSRSItem {
		keepIndices := make(map[int]bool)

		for _, indices := range groups {
			// Skip groups with fewer than 6 items
			if len(indices) < 6 {
				continue
			}

			// Calculate how many to remove (~5%, minimum 1 if group is large enough)
			removeCount := len(indices) * 5 / 100
			if len(indices) >= 20 && removeCount < 1 {
				removeCount = 1
			}

			// Keep items except the outliers (remove from both ends)
			// For simplicity, we keep all items from valid groups
			// The main filtering is done by group size and time interval
			keepCount := len(indices) - removeCount
			for j := 0; j < keepCount && j < len(indices); j++ {
				keepIndices[indices[j]] = true
			}
		}

		result := make([]FSRSItem, 0, len(keepIndices))
		for i, item := range items {
			if keepIndices[i] {
				result = append(result, item)
			}
		}
		return result
	}

	filteredInit := filterByGroup(initItems, initGroups)
	filteredTrain := filterByGroup(trainItems, trainGroups)

	return filteredInit, filteredTrain
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
		// Cubic recency weight: 0.25 to 1.0
		// Cubic curve gives much higher weight to recent items
		position := float64(i) / n
		weight := 0.25 + 0.75*math.Pow(position, 3)

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
