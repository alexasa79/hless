package utils

import "math"

type Range interface {
	Start() int
	End() int
	Subrange(start int, end int) Range
}

func Intersect(a, b *Range) bool {
	return (*a).Start() <= (*b).End() && (*b).Start() <= (*a).End()
}

func addGapRanges(result *[]Range, ranges1 []Range, gapStart, gapEnd int) {
	for _, r1 := range ranges1 {
		// Check if the range from ranges1 overlaps with the gap
		start := max(r1.Start(), gapStart)
		end := min(r1.End(), gapEnd)

		if start < end {
			*result = append(*result, r1.Subrange(start, end))
		}
	}
}

func RangesProduct(ranges1 []Range, ranges2 []Range) []Range {
	res := make([]Range, 0)

	// Add ranges from ranges1 that come before the first range in ranges2
	if len(ranges2) > 0 {
		firstStart := ranges2[0].Start()
		addGapRanges(&res, ranges1, math.MinInt, firstStart)
	}

	// Iterate through all ranges in ranges2
	prevEnd := 0
	for i, r2 := range ranges2 {
		if i == 0 {
			res = append(res, r2)
		} else {
			addGapRanges(&res, ranges1, prevEnd, r2.Start())
			res = append(res, r2)
		}
		prevEnd = r2.End()
	}

	// Add remaining ranges from ranges1 that come after the last range in ranges2
	if len(ranges2) > 0 {
		lastEnd := ranges2[len(ranges2)-1].End()
		addGapRanges(&res, ranges1, lastEnd, math.MaxInt)
	} else {
		addGapRanges(&res, ranges1, math.MinInt, math.MaxInt)
	}

	return res
}
