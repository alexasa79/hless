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
