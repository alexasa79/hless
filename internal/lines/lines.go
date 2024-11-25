package lines

import (
	"fmt"
	"math"
	"regexp"
	"sync"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
)

var LL *hhlog.Logger = hhlog.NewLogger("LINES")
var mx sync.Mutex

func SectionComp(a, b *Section) int {
	if a.read.start < b.read.start {
		return -1
	} else if a.read.start == b.read.start {
		return 0
	} else {
		return 1
	}
}

// Class representing a range of lines, relative to Buffer.
type Range struct {
	start int64
	end   int64
}

type Section struct {
	name string

	read  Range // Range in the Buffer we scanned looking for lines
	lines int64 // how many lines we found

	nextIndexForward  int64
	nextIndexBackward int64
}

type Line struct {
	Offset  int64
	index   int64 // index in section
	section *Section
}

type Lines struct {
	data *Buffer

	lines          *avltree.AVLTree[*Line]
	sections       *avltree.AVLTree[*Section]
	currentSection *Section

	batchSize int

	view *View
}

func LineComp(a, b *Line) int {
	if a.Offset < b.Offset {
		return -1
	} else if a.Offset == b.Offset {
		return 0
	} else {
		return 1
	}
}

func newSection(offset int64, name string) *Section {
	return &Section{
		name:              name,
		nextIndexForward:  1,
		nextIndexBackward: 0,
		read:              Range{start: offset, end: offset},
	}
}

func NewLines(data *Buffer) *Lines {
	lines := &Lines{
		data:           data,
		sections:       avltree.NewAVLTree[*Section](SectionComp),
		lines:          avltree.NewAVLTree[*Line](LineComp),
		currentSection: newSection(0, ""),
		view:           NewView(),
		batchSize:      128,
	}

	lines.sections.Insert(lines.currentSection)

	LL.Debug("Started new section %v", lines.currentSection)

	return lines
}

func (line *Line) String() string {
	return fmt.Sprintf("[Line @%p: @%d, index=%d, sec=%p]",
		line, line.Offset, line.index, line.section)
}

func (s *Section) String() string {
	return fmt.Sprintf("[Section @%p: name=%s, lines=%d, read=%d,%d index=%d,%d]",
		s, s.name, s.lines, s.read.start, s.read.end, s.nextIndexForward, s.nextIndexBackward)
}

func (lines *Lines) First() *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.forcedStart != 0 {
		return lines.at(lines.view.forcedStart)
	}

	res := lines.first()
	for res != nil && !lines.visible(res) {
		res = lines.next(res)
	}

	return res
}

func (lines *Lines) SetFirst(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	lines.view.Reset()
	lines.view.SetFirst(line)

	LL.Debug("Set first line to %v", line)
}

func (lines *Lines) SetLast(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	lines.view.Reset()
	lines.view.SetLast(line, lines.RawLength(line))

	LL.Debug("Set last line to %v", line)
}

func (lines *Lines) Last() *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.forcedEnd != math.MaxInt64 {
		return lines.at(lines.view.forcedEnd)
	}

	res := lines.last()
	for res != nil && !lines.visible(res) {
		res = lines.prev(res)
	}

	return res
}

func (lines *Lines) Next(line *Line) (next *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return nil
	}

	if line.Offset+lines.RawLength(line) >= lines.view.forcedEnd {
		return nil
	}

	next = lines.next(line)
	for next != nil {
		if lines.visible(next) {
			break
		} else {
			next = lines.next(next)
		}
	}

	return next
}

func (lines *Lines) Prev(line *Line) (prev *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return nil
	}

	if line.Offset <= lines.view.forcedStart {
		return nil
	}

	prev = lines.prev(line)
	for prev != nil {
		if lines.visible(prev) {
			break
		} else {
			prev = lines.prev(prev)
		}
	}

	return prev
}

func (lines *Lines) JumpTo(offset int64, name string) {
	mx.Lock()
	defer mx.Unlock()

	lines.jumpTo(offset, name)
}

func (lines *Lines) At(offset int64) *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.OutOfRange(offset) {
		return nil
	}

	line := lines.at(offset)
	prev := line
	next := line
	for line != nil {
		if lines.visible(prev) {
			return prev
		} else {
			prev = lines.prev(prev)
		}

		if lines.visible(next) {
			return next
		} else {
			next = lines.next(next)
		}
	}

	return line
}

func (lines *Lines) visible(line *Line) bool {
	show := true
	if lines.view.showPattern != nil && !lines.view.showPattern.MatchString(lines.Text(line)) {
		show = false
	}

	if lines.view.hidePattern != nil && lines.view.hidePattern.MatchString(lines.Text(line)) {
		show = false
	}

	return show
}

func (lines *Lines) Visible(line *Line) bool {
	mx.Lock()
	defer mx.Unlock()

	if lines.view.OutOfRange(line.Offset) {
		return false
	}

	return lines.visible(line)
}

func (lines *Lines) Text(line *Line) string {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return ""
	}

	res := ""
	size := lines.data.Size()
	for pos := line.Offset; pos < size; pos++ {
		c := lines.data.At(pos)
		if c == '\n' {
			break
		}
		res += string(c)
	}
	return res
}

func (lines *Lines) Length(line *Line) int64 {
	res := int64(0)
	size := lines.data.Size()
	for line.Offset+res < size {
		if lines.data.At(line.Offset+res) == '\n' {
			break
		}
		res++
	}
	return res
}

func (lines *Lines) RawLength(line *Line) int64 {
	res := int64(0)
	size := lines.data.Size()
	for line.Offset+res < size {
		if lines.data.At(line.Offset+res) == '\n' {
			res += 1
			break
		}
		res++
	}
	return res
}

func (lines *Lines) SetShowPattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.view.showPattern = regexp.MustCompile(pat)
	} else {
		lines.view.showPattern = nil
	}

	lines.view.Reset()
	LL.Debug("Set show pattern to %s", pat)
}

func (lines *Lines) SetHidePattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.view.hidePattern = regexp.MustCompile(pat)
	} else {
		lines.view.hidePattern = nil
	}

	lines.view.Reset()
	LL.Debug("Set hide pattern to %s", pat)
}

func (lines *Lines) Refresh() {
	mx.Lock()
	defer mx.Unlock()

	lines.data.Refresh()
	lines.readForward()
}

func (line *Line) Name() string {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return ""
	}

	s := line.section
	if s.name == "" {
		return fmt.Sprintf("%d", line.index)
	}
	var t string
	if line.index == 0 {
		t = s.name
	} else {
		t = fmt.Sprintf("%s%+d", s.name, line.index)
	}
	return t
}

func Distance(l1, l2 *Line) string {
	if l1.section == l2.section {
		dist := l2.index - l1.index
		if dist < 0 {
			dist *= -1
		}
		return fmt.Sprintf("%d", dist)
	} else {
		return "⋯" + l1.Name()
	}
}
