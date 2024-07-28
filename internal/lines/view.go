package lines

import (
	"math"
	"regexp"

	"github.com/alexasa79/hless/internal/avltree"
)

type View struct {
	forcedStart int64
	forcedEnd   int64

	visibleStart int64
	visibleEnd   int64

	lines *avltree.AVLTree[*Line]

	showPattern *regexp.Regexp
	hidePattern *regexp.Regexp
}

func NewView() *View {
	return &View{
		lines:        avltree.NewAVLTree[*Line](LineComp),
		forcedStart:  0,
		forcedEnd:    math.MaxInt64,
		visibleStart: 0,
		visibleEnd:   math.MaxInt64,
		showPattern:  nil,
		hidePattern:  nil,
	}
}

func (view *View) Reset() {
	view.visibleStart = 0
	view.visibleEnd = math.MaxInt64
	view.lines = avltree.NewAVLTree[*Line](LineComp)
}

func (view *View) OutOfRange(offset int64) bool {
	if offset < view.forcedStart ||
		offset < view.visibleStart ||
		offset > view.forcedEnd ||
		offset > view.visibleEnd {
		return true
	} else {
		return false
	}
}

func (view *View) UpdateVisibleStart(offset int64) {
	LL.Debug("Set visible start to %d", offset)
	view.visibleStart = offset
}

func (view *View) UpdateVisibleEnd(offset int64) {
	LL.Debug("Set visible end to %d", offset)
	view.visibleEnd = offset
}

func (view *View) SetFirst(line *Line) {
	view.lines.Insert(line)
	view.forcedStart = line.Offset
	view.visibleStart = line.Offset
}

func (view *View) SetLast(line *Line) {
	view.lines.Insert(line)
	view.forcedEnd = line.Offset + line.RawLength
	view.visibleEnd = line.Offset + line.RawLength
}
