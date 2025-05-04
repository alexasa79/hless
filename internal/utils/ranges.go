package utils

type Range interface {
	Start() int
	End() int
	Subrange(start int, end int) Range
}

func Intersect(a, b Range) bool {
	return a.Start() < b.End() && b.Start() < a.End()
}

func Intersection(a, b Range) Range {
	start := max(a.Start(), b.Start())
	end := min(a.End(), b.End())

	if start <= end {
		return b.Subrange(start, end)
	}

	return nil
}

func ApplyColor(ranges []Range, color Range) []Range {
	res := make([]Range, 0)
	for _, r := range ranges {
		if !Intersect(r, color) {
			res = append(res, r)
			continue
		}

		ir := Intersection(r, color)
		before := r.Subrange(r.Start(), ir.Start())
		after := r.Subrange(ir.End(),  r.End())

		if before.Start() < before.End() {
			res = append(res, before)
		}

		res = append(res, ir)
		
		if after.Start() < after.End() {
			res = append(res, after)
		}
	}
	return res
}

