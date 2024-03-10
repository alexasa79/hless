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
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text() == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name() == "1")
	first := l

	search = &Line { Offset: 6 }
	l = lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text() == "worlds")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 6)
	utils.Assert((*l).Name() == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestCR(t *testing.T) {
	s := "hello\r\nworlds\r\n"
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.readForward()

	search := &Line { Offset: 0 }
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text() == "hello")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name() == "1")
	first := l

	search = &Line { Offset: 7 }
	l = lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Text() == "worlds")
	utils.Assert((*l).rawLength == 8)
	utils.Assert((*l).Offset == 7)
	utils.Assert((*l).Name() == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestLongLines(t *testing.T) {
	s := "hello\nworlds\r\nfoo\n"
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.batchSize = 2

	lines.readForward()
	search := &Line { Offset: 0 }
	l := lines.lines.Get(search)
	utils.Assert(l == nil)
	lines.readForward()
	l = lines.lines.Get(search)
	utils.Assert(l == nil)
	lines.readForward()
	l = lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Offset == int64(0))
	utils.Assert((*l).Text() == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Name() == "1")

	lines.readForward()
	lines.readForward()
	lines.readForward()
	lines.readForward()
	l2 := lines.Next(*l)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text() == "worlds")
	utils.Assert(l2.Name() == "2")
	utils.Assert(l2.Offset == int64(6))
	utils.Assert(l2.rawLength == 8)

	lines.readForward()
	lines.readForward()
	l2 = lines.Next(l2)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text() == "foo")
	utils.Assert(l2.Name() == "3")
	utils.Assert(l2.Offset == int64(14))
	utils.Assert(l2.rawLength == 4)
}

func TestReadBackward(t *testing.T) {
	s := "01\n345\n"
	utils.Assert(len(s) == 7)
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.batchSize = 2
	lines.JumpTo(7, "end")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 0)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 0)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 1)

	search := &Line { Offset: 3 }
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	ll := *l
	utils.Assert(ll.Offset == 3)
	utils.Assert(ll.rawLength == 4)
	utils.Assert(ll.len == 3)
	utils.Assert(ll.index == -1)
	utils.Assert(ll.Text() == "345")
	utils.Assert(ll.Name() == "end-1")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	l2 := lines.Next(*l)
	utils.Assert(l2 == nil)

	l2 = lines.Prev(*l)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text() == "01")
	utils.Assert(l2.Name() == "end-2")
	utils.Assert(l2.Offset == 0)
	utils.Assert(l2.len == 2)
	utils.Assert(l2.rawLength == 3)
}

func TestReadBackward2(t *testing.T) {
	s := "01\r\n456\r\n"
	utils.Assert(len(s) == 9)
	io := newStringReadSeeker(s)
	lines := NewLines(io, int64(len(s)))
	lines.batchSize = 2
	lines.JumpTo(9, "end")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 0)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 0)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 1)

	search := &Line { Offset: 4 }
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	ll := *l
	utils.Assert(ll.Offset == 4)
	utils.Assert(ll.rawLength == 5)
	utils.Assert(ll.len == 3)
	utils.Assert(ll.index == -1)
	utils.Assert(ll.Text() == "456")
	utils.Assert(ll.Name() == "end-1")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 1)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	l2 := lines.Next(*l)
	utils.Assert(l2 == nil)

	l2 = lines.Prev(*l)
	utils.Assert(l2 != nil)
	utils.Assert(l2.Text() == "01")
	utils.Assert(l2.Name() == "end-2")
	utils.Assert(l2.Offset == 0)
	utils.Assert(l2.len == 2)
	utils.Assert(l2.rawLength == 4)
}
