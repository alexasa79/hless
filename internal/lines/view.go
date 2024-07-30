package lines

import (
	"math"
	"regexp"
)

type View struct {
	forcedStart int64
	forcedEnd   int64

	showPattern *regexp.Regexp
	hidePattern *regexp.Regexp
}

func NewView() *View {
	return &View{
		forcedStart: 0,
		forcedEnd:   math.MaxInt64,
		showPattern: nil,
		hidePattern: nil,
	}
}

func (view *View) Reset() {
}

func (view *View) OutOfRange(offset int64) bool {
	if offset < view.forcedStart ||
		offset > view.forcedEnd {
		return true
	} else {
		return false
	}
}

func (view *View) SetFirst(line *Line) {
	view.forcedStart = line.Offset
}

func (view *View) SetLast(line *Line) {
	view.forcedEnd = line.Offset + line.RawLength
}

func (view *View) NoteLine(line *Line) {

}
