package utils

import (
	"testing"
)

type TestRange struct {
	start int
	end int
	color int
}

func (r TestRange) Start() int {
	return r.start
}

func (r TestRange) End() int {
	return r.end
}

func (r TestRange) Subrange(start int, end int) Range {
	return TestRange{
		start: start,
		end: end,
		color: r.color,
	}
}

func TestEmpty(t *testing.T) {
	r1 := make([]Range, 0)
	r2 := make([]Range, 0)

	res := RangesProduct(r1, r2)
	Assert(len(res) == 0)
}

func TestTrivial(t *testing.T) {
	r1 := make([]Range, 0)
	r2 := make([]Range, 0)

	r1 = append(r1, TestRange{ start: 0, end: 10, color: 1})

	res := RangesProduct(r1, r2)
	Assert(len(res) == 1)
	{
		t := res[0].(TestRange)
		Assert(t.start == 0)
		Assert(t.end == 10)
		Assert(t.color == 1)
	}

	r2 = append(r2, TestRange{ start: 11, end: 20, color: 2})
	res = RangesProduct(r1, r2)
	Assert(len(res) == 2)
	{
		t1 := res[0].(TestRange)
		t2 := res[1].(TestRange)
		Assert(t1.start == 0)
		Assert(t1.end == 10)
		Assert(t1.color == 1)
		Assert(t2.start == 11)
		Assert(t2.end == 20)
		Assert(t2.color == 2)
	}

	r2[0] = TestRange{ start: 10, end: 20, color: 2}
	res = RangesProduct(r1, r2)
	Assert(len(res) == 2)
	{
		t1 := res[0].(TestRange)
		t2 := res[1].(TestRange)
		Assert(t1.start == 0)
		Assert(t1.end == 10)
		Assert(t1.color == 1)
		Assert(t2.start == 10)
		Assert(t2.end == 20)
		Assert(t2.color == 2)
	}

	r2[0] = TestRange{ start: 0, end: 20, color: 2}
	res = RangesProduct(r1, r2)
	Assert(len(res) == 2)
	{
		t1 := res[0].(TestRange)
		t2 := res[1].(TestRange)
		Assert(t1.start == 0)
		Assert(t1.end == 10)
		Assert(t1.color == 1)
		Assert(t2.start == 10)
		Assert(t2.end == 20)
		Assert(t2.color == 2)
	}

	r2[0] = TestRange{ start: 5, end: 20, color: 2}
	res = RangesProduct(r1, r2)
	Assert(len(res) == 3)
	{
		t1 := res[0].(TestRange)
		t2 := res[1].(TestRange)
		t3 := res[2].(TestRange)
		Assert(t1.start == 0)
		Assert(t1.end == 5)
		Assert(t1.color == 1)
		Assert(t2.start == 5)
		Assert(t2.end == 10)
		Assert(t2.color == 1)
		Assert(t3.start == 10)
		Assert(t3.end == 20)
		Assert(t3.color == 2)
	}
}

func TestComplex(t *testing.T) {
	r1 := make([]Range, 0)
	r2 := make([]Range, 0)

	r1 = append(r1, TestRange{ start: 0, end: 123, color: 1})
	r1 = append(r1, TestRange{ start: 123, end: 246, color: 1})

	r2 = append(r2, TestRange{ start: 64, end: 68, color: 2})
	r2 = append(r2, TestRange{ start: 76, end: 80, color: 2})

	res := RangesProduct(r1, r2)
	{
		t1 := res[0].(TestRange)
		Assert(t1.start == 0)
	}
}

func TestColorOverlap(t *testing.T) {
	r1 := make([]Range, 0)
	r2 := make([]Range, 0)

	r1 = append(r1, TestRange{ start: 0, end: 20, color: 1 })
	r2 = append(r2, TestRange{ start: 7, end: 23, color: 2 })

	res := RangesProduct(r1, r2)
	{
		t1 := res[0].(TestRange)
		Assert(t1.start == 0)
		Assert(t1.end == 7)
		Assert(t1.color == 1)

		t2 := res[1].(TestRange)
		Assert(t2.start == 7)
		Assert(t2.end == 20)
		Assert(t2.color == 2)

		t3 := res[2].(TestRange)
		Assert(t3.start == 20)
		Assert(t3.end == 23)
		Assert(t3.color == 2)
	}
}