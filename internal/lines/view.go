package lines

import (
	"math"
	"regexp"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/utils"
)

type LineRecord struct {
	Line *Line
	Pred *Line
	Succ *Line
}

type View struct {
	forcedStart int64
	forcedEnd   int64

	showPattern *regexp.Regexp
	hidePattern *regexp.Regexp

	lines *avltree.AVLTree[*LineRecord]
	first *Line
	last  *Line
}

func ViewRangeComp(a, b *LineRecord) int {
	if a.Line.Offset < b.Line.Offset {
		return -1
	} else if a.Line.Offset == b.Line.Offset {
		return 0
	} else {
		return 1
	}
}

func NewView() *View {
	lines := avltree.NewAVLTree[*LineRecord](ViewRangeComp)
	return &View{
		forcedStart: 0,
		forcedEnd:   math.MaxInt64,
		showPattern: nil,
		hidePattern: nil,
		lines:       lines,
	}
}

func (view *View) Reset() {
	LL.Trace("Resetting the view")
	lines := avltree.NewAVLTree[*LineRecord](ViewRangeComp)
	view.forcedStart = 0
	view.forcedEnd = math.MaxInt64
	view.lines = lines
	view.first = nil
	view.last = nil
}

func (view *View) OutOfRange(offset int64) bool {
	if offset < view.forcedStart ||
		offset > view.forcedEnd {
		return true
	} else {
		return false
	}
}

func (view *View) noteLine(line *Line) *LineRecord {
	lr := view.lines.Get(&LineRecord{Line: line})
	if lr != nil {
		return *lr
	}

	newLr := &LineRecord{Line: line}
	view.lines.Insert(newLr)

	return newLr
}

func (view *View) NoteSuccessor(line *Line, succ *Line) {
	LL.Trace("Recording %v successor of %v", succ, line)

	if line == nil || succ == nil {
		return
	}
	lineR := view.noteLine(line)
	succR := view.noteLine(succ)
	lineR.Succ = succR.Line
	succR.Pred = lineR.Line
}

func (view *View) NotePredecessor(line *Line, pred *Line) {
	LL.Trace("Recording %v predecessor of %v", pred, line)

	if line == nil || pred == nil {
		return
	}
	lineR := view.noteLine(line)
	predR := view.noteLine(pred)
	lineR.Pred = predR.Line
	predR.Succ = lineR.Line
}

func (view *View) NoteFirst(line *Line) {
	LL.Trace("Recording first line %v", line)

	if line == nil {
		return
	}
	view.noteLine(line)
	view.first = line
}

func (view *View) NoteLast(line *Line) {
	LL.Trace("Recording last line %v", line)

	if line == nil {
		return
	}
	utils.Assert(line != nil)
	view.noteLine(line)
	view.last = line
}

func (view *View) NextOf(line *Line) *Line {
	pl := view.lines.Get(&LineRecord{Line: line})
	if pl != nil {
		return (*pl).Succ
	} else {
		return nil
	}
}

func (view *View) PrevOf(line *Line) *Line {
	pl := view.lines.Get(&LineRecord{Line: line})
	if pl != nil {
		return (*pl).Pred
	} else {
		return nil
	}
}

func (view *View) GetFirst() *Line {
	return view.first
}

func (view *View) GetLast() *Line {
	return view.last
}

func (view *View) Add(line *Line) {
	lr := &LineRecord{Line: line}
	pred := view.lines.Predecessor(lr)
	if pred != nil {
		(*pred).Succ = line
		lr.Pred = (*pred).Line
	}

	succ := view.lines.Successor(lr)
	if succ != nil {
		(*succ).Pred = line
		lr.Succ = (*succ).Line
	}

	view.lines.Insert(lr)
}

func (view *View) Has(line *Line) bool {
	if line == nil {
		return false
	}
	return view.lines.Contains(&LineRecord{Line: line})
}
