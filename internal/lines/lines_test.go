package lines

import (
	"testing"

	"github.com/alexasa79/hless/internal/utils"
)

func TestTrivial(t *testing.T) {
	s := "hello\nworlds\n"
	lines := NewLines(NewBufferFromString(s))
	lines.readForward()

	search := &Line{Offset: 0}
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert(lines.Text(*l) == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name() == "1")
	first := l

	search = &Line{Offset: 6}
	l = lines.lines.Get(search)
	utils.Assert(l == nil)

	lines.readForward()
	l = lines.lines.Get(search)
	utils.Assert(l != nil)
	ttt := lines.Text(*l)
	utils.Assert(ttt != "")
	utils.Assert(lines.Text(*l) == "worlds")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 6)
	utils.Assert((*l).Name() == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestCR(t *testing.T) {
	s := "hello\r\nworlds\r\n"
	lines := NewLines(NewBufferFromString(s))
	lines.readForward()

	search := &Line{Offset: 0}
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert(lines.Text(*l) == "hello")
	utils.Assert((*l).rawLength == 7)
	utils.Assert((*l).Offset == 0)
	utils.Assert((*l).Name() == "1")
	first := l

	lines.readForward()
	search = &Line{Offset: 7}
	l = lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert(lines.Text(*l) == "worlds")
	utils.Assert((*l).rawLength == 8)
	utils.Assert((*l).Offset == 7)
	utils.Assert((*l).Name() == "2")

	utils.Assert(lines.Next(*first) == *l)
	utils.Assert(lines.Next(*l) == nil)
}

func TestMixedEOL(t *testing.T) {
	s := "hello\nworlds\r\nfoo\n"
	lines := NewLines(NewBufferFromString(s))

	lines.readForward()
	search := &Line{Offset: 0}
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	utils.Assert((*l).Offset == int64(0))
	utils.Assert(lines.Text(*l) == "hello")
	utils.Assert((*l).rawLength == 6)
	utils.Assert((*l).Name() == "1")

	lines.readForward()
	l2 := lines.Next(*l)
	utils.Assert(l2 != nil)
	utils.Assert(lines.Text(l2) == "worlds")
	utils.Assert(l2.Name() == "2")
	utils.Assert(l2.Offset == int64(6))
	utils.Assert(l2.rawLength == 8)

	lines.readForward()
	l2 = lines.Next(l2)
	utils.Assert(l2 != nil)
	utils.Assert(lines.Text(l2) == "foo")
	utils.Assert(l2.Name() == "3")
	utils.Assert(l2.Offset == int64(14))
	utils.Assert(l2.rawLength == 4)
}

func TestReadBackward(t *testing.T) {
	s := "01\n345\n"
	utils.Assert(len(s) == 7)
	lines := NewLines(NewBufferFromString(s))
	lines.JumpTo(7, "end")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 1)

	search := &Line{Offset: 3}
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	ll := *l
	utils.Assert(ll.Offset == 3)
	utils.Assert(ll.rawLength == 4)
	utils.Assert(ll.len == 3)
	utils.Assert(ll.index == -1)
	utils.Assert(lines.Text(ll) == "345")
	utils.Assert(ll.Name() == "end-1")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	l2 := lines.Next(*l)
	utils.Assert(l2 == nil)

	l2 = lines.Prev(*l)
	utils.Assert(l2 != nil)
	utils.Assert(lines.Text(l2) == "01")
	utils.Assert(l2.Name() == "end-2")
	utils.Assert(l2.Offset == 0)
	utils.Assert(l2.len == 2)
	utils.Assert(l2.rawLength == 3)
}

func TestReadBackward2(t *testing.T) {
	s := "01\r\n456\r\n"
	utils.Assert(len(s) == 9)
	lines := NewLines(NewBufferFromString(s))
	lines.JumpTo(9, "end")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 1)

	search := &Line{Offset: 4}
	l := lines.lines.Get(search)
	utils.Assert(l != nil)
	ll := *l
	utils.Assert(ll.Offset == 4)
	utils.Assert(ll.rawLength == 5)
	utils.Assert(ll.len == 3)
	utils.Assert(ll.index == -1)
	utils.Assert(lines.Text(ll) == "456")
	utils.Assert(ll.Name() == "end-1")

	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	l2 := lines.Next(*l)
	utils.Assert(l2 == nil)

	l2 = lines.Prev(*l)
	utils.Assert(l2 != nil)
	utils.Assert(lines.Text(l2) == "01")
	utils.Assert(l2.Name() == "end-2")
	utils.Assert(l2.Offset == 0)
	utils.Assert(l2.len == 2)
	utils.Assert(l2.rawLength == 4)
}

func TestBackAndForth(t *testing.T) {
	s := "012\n34\n"
	utils.Assert(len(s) == 7)
	lines := NewLines(NewBufferFromString(s))

	lines.readForward()
	utils.Assert(lines.lines.Count() == 1)

	lines.JumpTo(7, "end")
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	l2 := lines.Last()
	utils.Assert(l2.Offset == 4)
	utils.Assert(l2.Name() == "end-1")

	l1 := lines.Prev(l2)
	utils.Assert(l1 != nil)
}

func TestBackAndForth2(t *testing.T) {
	s := "01\n23\n34\n"
	utils.Assert(len(s) == 9)
	lines := NewLines(NewBufferFromString(s))

	lines.readForward()
	utils.Assert(lines.lines.Count() == 1)

	lines.JumpTo(9, "end")
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 2)
	lines.readBackward()
	utils.Assert(lines.lines.Count() == 3)

	l3 := lines.Last()
	utils.Assert(l3.Offset == 6)
	utils.Assert(l3.Name() == "end-1")
	l2 := lines.Prev(l3)
	utils.Assert(l2.Offset == 3)
	utils.Assert(l2.Name() == "end-2")
	l1 := lines.Prev(l2)
	utils.Assert(l1.Offset == 0)
	utils.Assert(l1.Name() == "1")
}

func TestJumpTo(t *testing.T) {
	s := "01\n345\n"
	utils.Assert(len(s) == 7)
	lines := NewLines(NewBufferFromString(s))
	// lines.batchSize = 2
	lines.JumpTo(0, "")

	utils.Assert(lines.sections.Count() == 1)
}

func TestJumpTo2(t *testing.T) {
	s := "01\n345\n"
	utils.Assert(len(s) == 7)
	lines := NewLines(NewBufferFromString(s))
	// lines.batchSize = 2
	lines.JumpTo(0, "")

	lines.readForward()

	lines.JumpTo(7, "end")
	lines.readBackward()

	utils.Assert(lines.sections.Count() == 2)

	lines.JumpTo(7, "end")
	lines.readBackward()

	utils.Assert(lines.sections.Count() == 2)
}

func TestAt1(t *testing.T) {
	s := "01\n345\n678\n"
	lines := NewLines(NewBufferFromString(s))

	l := lines.At(4, "at 4")
	utils.Assert(l != nil)
	utils.Assert(lines.Text(l) == "345")
	utils.Assert(l.Offset == 3)
	utils.Assert(l.Name() == "at 4")

	l2 := lines.Next(l)
	utils.Assert(lines.Text(l2) == "678")
	utils.Assert(l2 != nil)
	utils.Assert(l2.Name() == "at 4+1")

	l3 := lines.Prev(l)
	utils.Assert(lines.Text(l3) == "01")
	utils.Assert(l3 != nil)
	utils.Assert(l3.Name() == "at 4-1")
}
