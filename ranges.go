package main

type Range interface {
	Start() int
	End() int
	Subrange(start int, end int) Range
}

// type pivot struct {
// 	n int
// 	r int
// 	rr *Range
// 	start bool
// }

// func Intersects(a, b *Range) bool {
//     return (*a).Start() <= (*b).End() && (*b).Start() <= (*a).End()
// }

/// Assume sorted, two non overlapping lists of ranges, find
// func RangesProduct(ranges1 []Range, ranges2 []Range) []Range {
// 	pivots := make([]pivot, len(ranges1) * 2 + len(ranges2) * 2)
// 	n := 0
// 	for _, r := range ranges1 {
// 		pivots[n].n = r.Start()
// 		pivots[n].r = 1
// 		pivots[n].rr = &r
// 		pivots[n].start = true
// 		n++
// 		pivots[n].n = r.End()
// 		pivots[n].r = 1
// 		pivots[n].rr = &r
// 		pivots[n].start = false
// 		n++
// 	}

// 	for _, r := range ranges2 {
// 		pivots[n].n = r.Start()
// 		pivots[n].r = 2
// 		pivots[n].rr = &r
// 		pivots[n].start = true
// 		n++
// 		pivots[n].n = r.End()
// 		pivots[n].r = 2
// 		pivots[n].rr = &r
// 		pivots[n].start = false
// 		n++
// 	}

// 	sort.Slice(pivots, func(i, j int) bool {
// 		if pivots[i].n < pivots[j].n {
// 			return true
// 		}
// 		if pivots[i].n == pivots[j].n {
// 			return pivots[i].r == pivots[j].r
// 		}
// 		return false
// 	})

// 	res := make([]Range, 0)

// 	for i := 0; i < n - 1; i++ {
// 		if pivots[i].n != pivots[i + 1].n {
// 			if pivots[i].start {
// 				res = append(res, (*pivots[i].rr).Subrange(pivots[i].n, pivots[i + 1].n))
// 			} else {
// 				res = append(res, (*pivots[i + 1].rr).Subrange(pivots[i].n, pivots[i + 1].n))
// 			}
// 		}
// 	}

// 	return res
// }

/// Assume sorted, two non overlapping lists of ranges, find
func RangesProduct(ranges1 []Range, ranges2 []Range) []Range {
	res := make([]Range, 0)

	cursor := 0
	i := 0
	j := 0

	if len(ranges1) > 0 {
		cursor = min(cursor, ranges1[0].Start())
	}

	if len(ranges2) > 0 {
		cursor = min(cursor, ranges2[0].Start())
	}

	for i < len(ranges1) && j < len(ranges2) {
		r1 := ranges1[i]
		r2 := ranges2[j]

		start1 := r1.Start()
		start2 := r2.Start()
		end1 := r1.End()
		end2 := r2.End()

		if cursor >= end1 {
			i++
			continue
		}

		if cursor >= end2 {
			j++
			continue
		}

		if cursor >= start1 && cursor < end1 && cursor >= start2 && cursor < end2 {
			if end1 > end2 {
				res = append(res, ranges2[j].Subrange(cursor, end2))
				cursor = end2
			} else {
				res = append(res, ranges1[i].Subrange(cursor, end1))
				cursor = end1
			}
		} else if cursor >= start1 && cursor < end1 {
			end := min(start2, end1)
			res = append(res, ranges1[i].Subrange(cursor, end))
			cursor = end
		} else if cursor >= start2 && cursor < end2 {
			end := min(start1, end2)
			res = append(res, ranges2[j].Subrange(cursor, end))
			cursor = end
		} else {
			cursor = min(start1, start2)
		}
	}

	assert(i == len(ranges1) || j == len(ranges2))

	if i < len(ranges1) {
		if cursor >= ranges1[i].Start() && cursor < ranges1[i].End() {
			res = append(res, ranges1[i].Subrange(cursor, ranges1[i].End()))
			i++
		}
		for ; i < len(ranges1); i++ {
			res = append(res, ranges1[i])
		}
	}

	if j < len(ranges2) {
		if cursor >= ranges2[j].Start() && cursor < ranges2[j].End() {
			res = append(res, ranges2[j].Subrange(cursor, ranges2[j].End()))
			j++
		}
		for ; j < len(ranges1); j++ {
			res = append(res, ranges2[j])
		}
	}

	return res
}