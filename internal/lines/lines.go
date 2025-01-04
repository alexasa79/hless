package lines

import (
	"fmt"
	"math"
	"regexp"
	"sync"

	"github.com/alexasa79/hless/internal/avltree"
	"github.com/alexasa79/hless/internal/hhlog"
	"github.com/alexasa79/hless/internal/utils"
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
	data     *Buffer
	fileSize int64

	lines          *avltree.AVLTree[*Line]
	sections       *avltree.AVLTree[*Section]
	currentSection *Section

	batchSize int

	currentView   *View
	mainView      *View
	alternateView *View
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
		mainView:       NewView(),
		alternateView:  NewView(),
		batchSize:      128,
	}

	lines.currentView = lines.mainView

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

	if lines.currentView.forcedStart != 0 {
		res := lines.at(lines.currentView.forcedStart)
		return res
	}

	res := lines.currentView.GetFirst()
	if res != nil {
		utils.Assert(lines.visible(res), res)
		return res
	}

	res = lines.first()
	for res != nil && !lines.visible(res) {
		res = lines.next(res)
	}

	if lines.visible(res) {
		lines.currentView.NoteFirst(res)
	}

	return res
}

func (lines *Lines) SetFirst(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	utils.Assert(lines.visible(line), line)
	lines.alternateView.NoteFirst(line)

	LL.Debug("Set first line to %v", line)
}

func (lines *Lines) SetLast(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return
	}

	utils.Assert(lines.visible(line), line)
	lines.alternateView.NoteLast(line)

	LL.Debug("Set last line to %v", line)
}

func (lines *Lines) Last() *Line {
	mx.Lock()
	defer mx.Unlock()

	if lines.currentView.forcedEnd != math.MaxInt64 {
		res := lines.at(lines.currentView.forcedEnd)
		LL.Debug("Asked to return last line. Returning %+v", res)
		return res
	}

	res := lines.currentView.GetLast()
	if res != nil {
		utils.Assert(lines.visible(res), res)
		LL.Debug("Asked to return last line. Returning %+v", res)
		return res
	}

	res = lines.last()
	for res != nil && !lines.visible(res) {
		res = lines.prev(res)
	}

	if lines.visible(res) {
		lines.currentView.NoteLast(res)
	}

	LL.Debug("Asked to return last line. Returning %+v", res)
	return res
}

func (lines *Lines) Next(line *Line) (next *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return nil
	}

	if line.Offset+lines.RawLength(line) >= lines.currentView.forcedEnd {
		return nil
	}

	next = lines.currentView.NextOf(line)
	if next != nil {
		utils.Assert(lines.visible(next), next)
		return next
	}

	next = lines.next(line)
	for next != nil {
		if lines.visible(next) {
			break
		} else {
			next = lines.next(next)
		}
	}

	if lines.visible(line) && lines.visible(next) {
		lines.currentView.NoteSuccessor(line, next)
	}

	return next
}

func (lines *Lines) Prev(line *Line) (prev *Line) {
	mx.Lock()
	defer mx.Unlock()

	if line == nil {
		return nil
	}

	if line.Offset <= lines.currentView.forcedStart {
		return nil
	}

	prev = lines.currentView.PrevOf(line)
	if prev != nil {
		utils.Assert(lines.visible(prev))
		return prev
	}

	prev = lines.prev(line)
	for prev != nil {
		if lines.visible(prev) {
			break
		} else {
			prev = lines.prev(prev)
		}
	}

	if lines.visible(line) && lines.visible(prev) {
		lines.currentView.NotePredecessor(line, prev)
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

	if lines.currentView.OutOfRange(offset) {
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
	if lines.currentView.showPattern != nil && !lines.currentView.showPattern.MatchString(lines.text(line)) && !lines.currentView.Has(line) {
		show = false
	}

	if lines.currentView.hidePattern != nil && lines.currentView.hidePattern.MatchString(lines.text(line)) && !lines.currentView.Has(line) {
		show = false
	}

	return show
}

func (lines *Lines) Visible(line *Line) bool {
	mx.Lock()
	defer mx.Unlock()

	if lines.currentView.OutOfRange(line.Offset) {
		return false
	}

	return lines.visible(line)
}

func (lines *Lines) ToggleView() {
	mx.Lock()
	defer mx.Unlock()

	if lines.currentView == lines.alternateView {
		lines.currentView = lines.mainView
	} else {
		lines.currentView = lines.alternateView
	}
}

func (lines *Lines) text(line *Line) string {
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

func (lines *Lines) Text(line *Line) string {
	mx.Lock()
	defer mx.Unlock()

	return lines.text(line)
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

func (lines *Lines) resetView() {
	lines.alternateView.Reset()
}

func (lines *Lines) ResetView() {
	mx.Lock()
	defer mx.Unlock()

	lines.resetView()
}

func (lines *Lines) SetAlternateView() {
	mx.Lock()
	defer mx.Unlock()

	lines.currentView = lines.alternateView
}

func (lines *Lines) SetMainView() {
	mx.Lock()
	defer mx.Unlock()

	lines.currentView = lines.mainView
}

func (lines *Lines) SetShowPattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.alternateView.showPattern = regexp.MustCompile(pat)
	} else {
		lines.alternateView.showPattern = nil
	}

	LL.Debug("Set show pattern to %s", pat)
	lines.resetView()
}

func (lines *Lines) SetHidePattern(pat string) {
	mx.Lock()
	defer mx.Unlock()

	if pat != "" {
		lines.alternateView.hidePattern = regexp.MustCompile(pat)
	} else {
		lines.alternateView.hidePattern = nil
	}

	LL.Debug("Set hide pattern to %s", pat)
	lines.resetView()
}

func (lines *Lines) Refresh() {
	mx.Lock()
	defer mx.Unlock()

	lines.data.Refresh()
	if lines.data.fileSize != lines.fileSize {
		lines.readForward()
		lines.fileSize = lines.data.fileSize
		lines.currentView.Reset()
	}
}

func (lines *Lines) AddToView(line *Line) {
	mx.Lock()
	defer mx.Unlock()

	lines.alternateView.Add(line)
}

func (lines *Lines) IsAlternateView() bool {
	mx.Lock()
	defer mx.Unlock()

	return lines.currentView == lines.alternateView
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
