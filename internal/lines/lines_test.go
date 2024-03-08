package lines

import (
	"io"
	"testing"

	"github.com/alexasa79/hless/internal/utils"
)

type stringReadSeeker struct {
	pos int64
	s string
}

func newStringReadSeeker(s string) *stringReadSeeker {
	return &stringReadSeeker{
		pos: 0,
		s: s,
	}
}

func (srs *stringReadSeeker) Read(p []byte) (n int, err error) {
	if srs.pos >= int64(len(srs.s)) {
		return 0, io.EOF
	}

	n = copy(p, srs.s[srs.pos:])
	srs.pos += int64(n)
	return n, nil
}

func (srs *stringReadSeeker) Seek(offset int64, whence int) (int64, error) {
	var newPos int64
	switch whence {
	case io.SeekStart:
		newPos = offset
	case io.SeekCurrent:
		newPos = srs.pos + offset
	case io.SeekEnd:
		newPos = int64(len(srs.s)) + offset
	default:
		panic("Unexpected whence value")
	}

	if newPos < 0 {
		panic("Unexpected new position")
	}

	srs.pos = newPos
	return newPos, nil
}

func TestTrivial(t *testing.T) {
	s := "hello\nworlds\n"
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.readForward()

	search := &Line { Offset: 0 }
	l := lines.lt.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name == "1")
	first := l

	search = &Line { Offset: 6 }
	l = lines.lt.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text == "worlds")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 6)
	utils.Assert((*l).Name == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestCR(t *testing.T) {
	s := "hello\r\nworlds\r\n"
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.readForward()

	search := &Line { Offset: 0 }
	l := lines.lt.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text == "hello")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name == "1")
	first := l

	search = &Line { Offset: 7 }
	l = lines.lt.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text == "worlds")
	utils.Assert((*l).rawLength == 8)
	utils.Assert((*l).Offset == 7)
	utils.Assert((*l).Name == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestCarryOver(t *testing.T) {
	s := "hello\nworlds\r\nfoo\n"
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.batchSize = 2

	lines.readForward()
	search := &Line { Offset: 0 }
	l := lines.lt.Get(search)
	utils.Assert(l == nil)
	utils.Assert(string(lines.carryOverBuffer) == "he")
	lines.readForward()
	l = lines.lt.Get(search)
	utils.Assert(l == nil)
	utils.Assert(string(lines.carryOverBuffer) == "hell")
	lines.readForward()
	l = lines.lt.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Offset == int64(0))
	utils.Assert((*l).Text == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Name == "1")

	lines.readForward()
	utils.Assert(string(lines.carryOverBuffer) == "wo")
	lines.readForward()
	utils.Assert(string(lines.carryOverBuffer) == "worl")
	lines.readForward()
	utils.Assert(string(lines.carryOverBuffer) == "worlds")
	lines.readForward()
	l2 := lines.Next(*l)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text == "worlds")
	utils.Assert(l2.Name == "2")
	utils.Assert(l2.Offset == int64(6))
	utils.Assert(l2.rawLength == int64(8))

	lines.readForward()
	lines.readForward()
	l2 = lines.Next(l2)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text == "foo")
	utils.Assert(l2.Name == "3")
	utils.Assert(l2.Offset == int64(14))
	utils.Assert(l2.rawLength == int64(4))
}