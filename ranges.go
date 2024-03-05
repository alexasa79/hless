package main

import "github.com/alexasa79/hless/internal/utils"

type Range interface {
	Start() int
	End() int
	Subrange(start int, end int) Range
}

func Intersect(a, b *Range) bool {
    return (*a).Start() <= (*b).End() && (*b).Start() <= (*a).End()
}

/// Assume sorted, two non overlapping lists of ranges, find
func RangesProduct(ranges1 []Range, ranges2 []Range) []Range {
	res := make([]Range, 0)

	cursor := 0
	i := 0
	j := 0

	if len(ranges1) > 0 {
		cursor = utils.Min(cursor, ranges1[0].Start())
	}

	if len(ranges2) > 0 {
		cursor = utils.Min(cursor, ranges2[0].Start())
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
			end := utils.Min(start2, end1)
			res = append(res, ranges1[i].Subrange(cursor, end))
			cursor = end
		} else if cursor >= start2 && cursor < end2 {
			end := utils.Min(start1, end2)
			res = append(res, ranges2[j].Subrange(cursor, end))
			cursor = end
		} else {
			cursor = utils.Min(start1, start2)
		}
	}

	utils.Assert(i == len(ranges1) || j == len(ranges2))

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